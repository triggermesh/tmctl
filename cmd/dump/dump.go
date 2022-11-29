/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dump

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type dumpOptions struct {
	Format   string
	Context  string
	Version  string
	CRD      string
	Manifest *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &dumpOptions{}
	knativeEventing := false
	dumpCmd := &cobra.Command{
		Use:     "dump [broker]",
		Short:   "Generate Kubernetes manifest",
		Example: "tmctl dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Version = viper.GetString("triggermesh.version")
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			o.Context = broker
			o.Manifest = manifest.New(filepath.Join(filepath.Dir(viper.ConfigFileUsed()), broker, triggermesh.ManifestFile))
			cobra.CheckErr(o.Manifest.Read())
			crds, err := crd.Fetch(filepath.Dir(viper.ConfigFileUsed()), o.Version)
			cobra.CheckErr(err)
			o.CRD = crds
			return o.dump(knativeEventing)
		},
	}
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")
	dumpCmd.Flags().BoolVar(&knativeEventing, "knative", false, "Use Knative Eventing components")
	return dumpCmd
}

func (o *dumpOptions) dump(useKnativeEventing bool) error {
	var externalReconcilable []string
	for _, object := range o.Manifest.Objects {
		if component, err := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest); err == nil {
			if reconcilable, ok := component.(triggermesh.Reconcilable); ok {
				if container, ok := component.(triggermesh.Runnable); ok {
					if _, err := container.Info(context.Background()); err == nil {
						var resources []string
						for _, r := range reconcilable.GetExternalResources() {
							resources = append(resources, r.(string))
						}
						if len(resources) != 0 {
							externalReconcilable = append(externalReconcilable, fmt.Sprintf("%s(%s)", component.GetName(), strings.Join(resources, ", ")))
						}
					}
				}
			}
		}
		if useKnativeEventing {
			object = o.knativeEventingTransformation(object)
		}
		switch o.Format {
		case "json":
			jsn, err := json.MarshalIndent(object, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsn))
		case "yaml":
			yml, err := yaml.Marshal(object)
			if err != nil {
				return err
			}
			fmt.Println("---")
			fmt.Println(string(yml))
		default:
			return fmt.Errorf("format %q is not supported", o.Format)
		}
	}
	if len(externalReconcilable) != 0 {
		fmt.Fprintf(os.Stderr, "\nWARNING: manifest contains running components that use external shared resources to produce events.\n"+
			"It is strongly recommended to stop the broker before deploying integration in the cluster to avoid events read race conditions.\n"+
			"External resources: %s\n", strings.Join(externalReconcilable, ", "))
	}
	return nil
}

func (o *dumpOptions) knativeEventingTransformation(object kubernetes.Object) kubernetes.Object {
	switch object.APIVersion {
	case tmbroker.APIVersion:
		switch object.Kind {
		case tmbroker.BrokerKind:
			object.APIVersion = "eventing.knative.dev/v1"
			object.Kind = "Broker"
		case tmbroker.TriggerKind:
			newSpec := map[string]interface{}{
				"broker":     o.Context,
				"subscriber": object.Spec["target"],
			}
			if filter, set := object.Spec["filters"]; set {
				newSpec["filters"] = filter
			}
			object.APIVersion = "eventing.knative.dev/v1"
			object.Spec = newSpec
		}
	case "sources.triggermesh.io/v1alpha1":
		object.Spec["sink"] = map[string]interface{}{
			"ref": map[string]interface{}{
				"name":       o.Context,
				"kind":       "Broker",
				"apiVersion": "eventing.knative.dev/v1",
			},
		}
	}
	return object
}

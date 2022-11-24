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
	"encoding/json"
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

type DumpOptions struct {
	Format   string
	Context  string
	Manifest *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &DumpOptions{}
	knativeEventing := false
	dumpCmd := &cobra.Command{
		Use:   "dump [broker]",
		Short: "Generate Kubernetes manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			o.Context = broker
			o.Manifest = manifest.New(path.Join(path.Dir(viper.ConfigFileUsed()), broker, triggermesh.ManifestFile))
			cobra.CheckErr(o.Manifest.Read())
			return o.dump(knativeEventing)
		},
	}
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")
	dumpCmd.Flags().BoolVar(&knativeEventing, "knative", false, "Use Knative Eventing components")
	return dumpCmd
}

func (o *DumpOptions) dump(useKnativeEventing bool) error {
	if useKnativeEventing {
		for i, object := range o.Manifest.Objects {
			o.Manifest.Objects[i] = o.knativeEventingTranformation(object)
		}
	}
	switch o.Format {
	case "json":
		for _, object := range o.Manifest.Objects {
			jsn, err := json.MarshalIndent(object, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsn))
		}
	case "yaml":
		for _, object := range o.Manifest.Objects {
			yml, err := yaml.Marshal(object)
			if err != nil {
				return err
			}
			fmt.Println("---")
			fmt.Println(string(yml))
		}
	default:
		return fmt.Errorf("format %q is not supported", o.Format)
	}
	return nil
}

func (o *DumpOptions) knativeEventingTranformation(object kubernetes.Object) kubernetes.Object {
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

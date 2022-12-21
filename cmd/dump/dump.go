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

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
	kyaml "sigs.k8s.io/yaml"
)

const (
	platformKubernetes    = "kubernetes"
	platformKnative       = "knative"
	platformDockerCompose = "docker-compose"
	platformDigitalOcean  = "digitalocean"
)

type dumpOptions struct {
	Format   string
	Context  string
	Version  string
	CRD      string
	Manifest *manifest.Manifest
	Platform string
}

func NewCmd() *cobra.Command {
	o := &dumpOptions{}
	dumpCmd := &cobra.Command{
		Use:     "dump [broker] -p [platform]",
		Short:   "Generate manifest",
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
			return o.dump()
		},
	}
	dumpCmd.Flags().StringVarP(&o.Platform, "platform", "p", "kubernetes", "kubernetes, knative, docker-compose, digitalocean")
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")
	return dumpCmd
}

func (o *dumpOptions) dump() error {
	var externalReconcilable []string
	var doServices []*godo.AppServiceSpec
	composeObjects := make(map[string]triggermesh.DockerComposeService)
	brokerConfig := eventingbroker.Config{
		Triggers: make(map[string]eventingbroker.Trigger),
	}

	for _, object := range o.Manifest.Objects {
		var secretsEnv map[string]string
		if component, err := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest); err == nil {
			if container, ok := component.(triggermesh.Runnable); ok {
				if _, err := container.Info(context.Background()); err == nil {
					if platform, ok := component.(triggermesh.Platform); ok {
						if _, ok := component.(triggermesh.Parent); ok {
							_, secretsEnv, err = components.ProcessSecrets(component.(triggermesh.Parent), o.Manifest)
							if err != nil {
								return fmt.Errorf("processing secrets: %v", err)
							}
						}
						switch o.Platform {
						case platformDigitalOcean:
							doService, err := platform.AsDigitalOcean(secretsEnv)
							if err != nil {
								return fmt.Errorf("processing DigitalOcean: %v", err)
							}
							doServices = append(doServices, doService)

						case platformDockerCompose:
							composeService, err := platform.AsDockerComposeObject(secretsEnv)
							if err != nil {
								return fmt.Errorf("processing Docker-Compose: %v", err)
							}
							composeObjects[object.Metadata.Name] = *composeService
						}
					}
					if reconcilable, ok := component.(triggermesh.Reconcilable); ok {
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
			switch o.Platform {
			case platformDigitalOcean:
				switch object.Kind {
				case tmbroker.TriggerKind:
					trigger := eventingbroker.Trigger{}

					target := component.(*tmbroker.Trigger).Target
					targetURL := fmt.Sprintf("${%s.PRIVATE_URL}", target.Ref.Name)
					filter := component.(*tmbroker.Trigger).Filters[0]
					trigger.Filters = append(trigger.Filters, filter)
					trigger.Target = eventingbroker.Target{
						URL: &targetURL,
					}
					brokerConfig.Triggers[object.Metadata.Name] = trigger
				}
			}
		}
		switch o.Platform {
		case platformKubernetes:
			fmt.Println("---")
			o.format(object)
		case platformKnative:
			fmt.Println("---")
			o.format(o.knativeEventingTransformation(object))
		}
	}

	switch o.Platform {
	case platformDockerCompose:
		composeObject := triggermesh.DockerCompose{
			Services: composeObjects,
		}
		o.format(composeObject)
	case platformDigitalOcean:
		for _, service := range doServices {
			if service.Name == o.Context {
				jsonBrokerConfig, err := json.Marshal(brokerConfig)
				if err != nil {
					return fmt.Errorf("processing broker config: %v", err)
				}
				service.Envs = append(service.Envs, &godo.AppVariableDefinition{
					Key: "BROKER_CONFIG", Value: string(jsonBrokerConfig),
				})
			}
		}
		doObject := godo.AppSpec{
			Name:     "triggermesh",
			Services: doServices,
		}

		o.format(doObject)
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

func (o *dumpOptions) format(object interface{}) error {
	switch o.Format {
	case "json":
		jsn, err := json.MarshalIndent(object, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsn))
	case "yaml":
		yml, err := kyaml.Marshal(object)
		if err != nil {
			return err
		}
		fmt.Println(string(yml))
	default:
		return fmt.Errorf("format %q is not supported", o.Format)
	}
	return nil
}

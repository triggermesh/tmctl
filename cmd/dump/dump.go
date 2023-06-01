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

	kyaml "sigs.k8s.io/yaml"

	"github.com/digitalocean/godo"
	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	platformKubernetes        = "kubernetes"
	platformKubernetesGeneric = "kubernetes-generic"
	platformKnative           = "knative"
	platformDockerCompose     = "docker-compose"
	platformDigitalOcean      = "digitalocean"
)

type doOptions struct {
	Region       string
	InstanceSize string
}

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
	CRD      map[string]crd.CRD

	Format   string
	Platform string

	NoSecrets bool
}

func NewCmd(config *config.Config, m *manifest.Manifest, crd map[string]crd.CRD) *cobra.Command {
	o := &CliOptions{
		CRD:      crd,
		Config:   config,
		Manifest: m,
	}
	do := &doOptions{}
	dumpCmd := &cobra.Command{
		Use:       "dump [broker] -p <kubernetes|knative|docker-compose|digitalocean> [-o json]",
		Short:     "Generate TriggerMesh manifests",
		Example:   "tmctl dump",
		ValidArgs: []string{"--platform", "--output"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				o.Config.Context = args[0]
				o.Manifest = manifest.New(filepath.Join(
					o.Config.ConfigHome,
					o.Config.Context,
					triggermesh.ManifestFile))
			}
			cobra.CheckErr(o.Manifest.Read())
			return o.dump(do)
		},
	}

	dumpCmd.Flags().StringVarP(&o.Platform, "platform", "p", "kubernetes", "Target platform. One of kubernetes, knative, docker-compose, digitalocean")
	dumpCmd.Flags().BoolVar(&o.NoSecrets, "no-secrets", false, "Remove secret values from the manifest")
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")

	dumpCmd.Flags().StringVarP(&do.Region, "do-region", "r", "fra", "DigitalOcean region")
	dumpCmd.Flags().StringVarP(&do.InstanceSize, "do-instance", "i", "professional-xs", "DigitalOcean instance size")

	cobra.CheckErr(dumpCmd.RegisterFlagCompletionFunc("platform", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
			platformKubernetes,
			platformKubernetesGeneric,
			platformKnative,
			platformDockerCompose,
			platformDigitalOcean,
		}, cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(dumpCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	}))

	return dumpCmd
}

func (o *CliOptions) dump(do *doOptions) error {
	var externalReconcilable []string
	var output interface{}
	for _, object := range o.Manifest.Objects {
		additionalEnv := make(map[string]string)
		component, err := components.GetObject(object.Metadata.Name, o.Config, o.Manifest, o.CRD)
		if err != nil {
			continue
		}
		if o.NoSecrets && component.GetAPIVersion() == "v1" && component.GetKind() == "Secret" {
			redactedData := make(map[string]string, len(component.GetSpec()))
			for key := range component.GetSpec() {
				redactedData[key] = triggermesh.UserInputTag
			}
			component = secret.New(component.GetName(), o.Config.Context, redactedData)
			object, _ = component.AsK8sObject()
		}
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
		if parent, ok := component.(triggermesh.Parent); ok {
			if _, additionalEnv, err = components.ProcessSecrets(parent, o.Manifest); err != nil {
				return fmt.Errorf("processing secrets: %v", err)
			}
		}

		switch o.Platform {
		case platformDigitalOcean:
			exportable, ok := component.(triggermesh.Exportable)
			if !ok {
				continue
			}
			if component.GetKind() == tmbroker.BrokerKind {
				config, err := o.getStaticBrokerConfig()
				if err != nil {
					return fmt.Errorf("broker static config: %w", err)
				}
				additionalEnv["BROKER_CONFIG"] = string(config)
			}
			if output == nil {
				output = map[string]interface{}{
					"name":     o.Config.Context,
					"region":   do.Region,
					"services": []interface{}{},
					"workers":  []interface{}{},
				}
			}
			platformObject, err := exportable.AsDigitalOceanObject(additionalEnv)
			if err != nil {
				return fmt.Errorf("unable to export component %q to %q: %v", component.GetName(), o.Platform, err)
			}
			platformObject = injectDOInstanceSize(platformObject, do.InstanceSize)
			if component.GetAPIVersion() == "sources.triggermesh.io/v1alpha1" {
				output.(map[string]interface{})["workers"] = append(output.(map[string]interface{})["workers"].([]interface{}), platformObject)
			} else {
				output.(map[string]interface{})["services"] = append(output.(map[string]interface{})["services"].([]interface{}), platformObject)
			}
		case platformDockerCompose:
			exportable, ok := component.(triggermesh.Exportable)
			if !ok {
				continue
			}
			if component.GetKind() == tmbroker.BrokerKind {
				config, err := o.getStaticBrokerConfig()
				if err != nil {
					return fmt.Errorf("broker static config: %w", err)
				}
				additionalEnv["BROKER_CONFIG"] = string(config)
			}
			if output == nil {
				output = map[string]interface{}{
					"services": map[string]interface{}{},
				}
			}
			platformObject, err := exportable.AsDockerComposeObject(additionalEnv)
			if err != nil {
				return fmt.Errorf("unable to export component %q to %q: %v", component.GetName(), o.Platform, err)
			}
			output.(map[string]interface{})["services"].(map[string]interface{})[component.GetName()] = platformObject
		case platformKubernetesGeneric:
			if component.GetKind() == tmbroker.BrokerKind {
				config, err := o.getStaticBrokerConfig()
				if err != nil {
					return fmt.Errorf("broker static config: %w", err)
				}
				additionalEnv["BROKER_CONFIG"] = string(config)
			}
			exportable, ok := component.(triggermesh.Exportable)
			if !ok {
				continue
			}
			deployment, err := exportable.AsKubernetesDeployment(additionalEnv)
			if err != nil {
				return fmt.Errorf("unable to export component %q to %q: %v", component.GetName(), o.Platform, err)
			}

			svc := kubernetes.CreateService(object.Metadata.Name)

			if output == nil {
				output = []interface{}{deployment, svc}
				continue
			}
			output = append(output.([]interface{}), deployment, svc)
		case platformKnative:
			object.Metadata.Namespace = ""
			if output == nil {
				output = []interface{}{o.knativeEventingTransformation(object)}
				continue
			}
			output = append(output.([]interface{}), o.knativeEventingTransformation(object))
		case platformKubernetes:
			object.Metadata.Namespace = ""
			if output == nil {
				output = []interface{}{object}
				continue
			}
			output = append(output.([]interface{}), object)
		default:
			return fmt.Errorf("platform %q is not supported", o.Platform)
		}
	}
	res, err := o.format(output)
	if err != nil {
		return fmt.Errorf("output format error: %w", err)
	}
	fmt.Println(string(res))

	if len(externalReconcilable) != 0 {
		fmt.Fprintf(os.Stderr, "\nWARNING: manifest contains running components that use external shared resources to produce events.\n"+
			"It is strongly recommended to stop the broker before deploying integration in the cluster to avoid events read race conditions.\n"+
			"External resources: %s\n", strings.Join(externalReconcilable, ", "))
	}
	return nil
}

func (o *CliOptions) getStaticBrokerConfig() ([]byte, error) {
	var staticBrokerConfig tmbroker.Configuration
	for _, object := range o.Manifest.Objects {
		component, err := components.GetObject(object.Metadata.Name, o.Config, o.Manifest, o.CRD)
		if err != nil {
			continue
		}
		if component == nil || component.GetKind() != tmbroker.TriggerKind {
			continue
		}

		trigger := component.(*tmbroker.Trigger)
		if staticBrokerConfig.Triggers == nil {
			staticBrokerConfig.Triggers = make(map[string]tmbroker.LocalTriggerSpec, 1)
		}
		staticBrokerConfig.Triggers[trigger.Name] = tmbroker.LocalTriggerSpec{
			Filters: trigger.Filters,
			Target: tmbroker.LocalTarget{
				URL: func() string {
					switch o.Platform {
					case platformDigitalOcean:
						return fmt.Sprintf("${%s.PRIVATE_URL}", trigger.Target.Ref.Name)
					case platformDockerCompose, platformKubernetesGeneric:
						return fmt.Sprintf("http://%s:8080", trigger.Target.Ref.Name)
					}
					return ""
				}(),
			},
		}
	}
	return json.Marshal(staticBrokerConfig)
}

func (o *CliOptions) knativeEventingTransformation(object kubernetes.Object) kubernetes.Object {
	switch object.APIVersion {
	case tmbroker.APIVersion:
		switch object.Kind {
		case tmbroker.BrokerKind:
			object.APIVersion = "eventing.knative.dev/v1"
			object.Kind = "Broker"
		case tmbroker.TriggerKind:
			newSpec := map[string]interface{}{
				"broker":     o.Config.Context,
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
				"name":       o.Config.Context,
				"kind":       "Broker",
				"apiVersion": "eventing.knative.dev/v1",
			},
		}
	}
	return object
}

func (o *CliOptions) format(object interface{}) ([]byte, error) {
	var result []byte
	switch o.Format {
	case "json":
		if array, ok := object.([]interface{}); ok {
			for _, item := range array {
				jsonItem, err := json.MarshalIndent(item, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("object encoding error: %w", err)
				}
				result = append(result, append(jsonItem, []byte("\n")...)...)
			}
			return result, nil
		}
		return json.MarshalIndent(object, "", "  ")
	case "yaml":
		if array, ok := object.([]interface{}); ok {
			for _, item := range array {
				yamlItem, err := kyaml.Marshal(item)
				if err != nil {
					return nil, fmt.Errorf("object encoding error: %w", err)
				}
				result = append(result, append([]byte("---\n"), yamlItem...)...)
			}
			return result, nil
		}
		return kyaml.Marshal(object)
	}
	return nil, fmt.Errorf("format %q is not supported", o.Format)
}

func injectDOInstanceSize(doObject interface{}, size string) interface{} {
	if service, ok := doObject.(godo.AppServiceSpec); ok {
		service.InstanceSizeSlug = size
		doObject = service
	}
	if worker, ok := doObject.(godo.AppWorkerSpec); ok {
		worker.InstanceSizeSlug = size
		doObject = worker
	}
	return doObject
}

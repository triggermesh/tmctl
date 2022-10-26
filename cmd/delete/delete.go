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

package delete

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

type DeleteOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
}

func NewCmd() *cobra.Command {
	o := &DeleteOptions{}
	var deleteBroker string
	deleteCmd := &cobra.Command{
		Use:               "delete <component_name_1, component_name_2...> [--broker <name>]",
		Short:             "Delete components by names",
		ValidArgsFunction: o.deleteCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deleteBroker != "" {
				return o.deleteBroker(deleteBroker)
			}
			return o.deleteComponents(args, false)
		},
	}
	cobra.OnInitialize(o.initialize)
	deleteCmd.Flags().StringVar(&deleteBroker, "broker", "", "Delete the broker")
	deleteCmd.RegisterFlagCompletionFunc("broker", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		list, err := brokers.List(path.Dir(viper.ConfigFileUsed()), "")
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		return list, cobra.ShellCompDirectiveNoFileComp
	})
	return deleteCmd
}

func (o *DeleteOptions) initialize() {
	o.ConfigBase = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds
}

func (o *DeleteOptions) deleteBroker(broker string) error {
	oo := *o
	oo.Context = broker
	if err := oo.deleteComponents([]string{}, true); err != nil {
		return fmt.Errorf("deleting component: %w", err)
	}
	if err := os.RemoveAll(path.Join(o.ConfigBase, broker)); err != nil {
		return fmt.Errorf("delete broker %q: %v", broker, err)
	}
	if broker == o.Context {
		return o.switchContext()
	}
	return nil
}

func (o *DeleteOptions) deleteComponents(names []string, force bool) error {
	ctx := context.Background()
	manifestPath := path.Join(o.ConfigBase, o.Context, manifestFile)
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	manifest := manifest.New(manifestPath)
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("manifest read: %w", err)
	}
	for _, object := range manifest.Objects {
		skip := false
		if len(names) > 0 {
			skip = true
			for _, v := range names {
				if v == object.Metadata.Name && object.Kind != "Secret" {
					skip = false
					break
				}
			}
		}
		if skip {
			continue
		}
		if object.Kind == "Broker" {
			if !force {
				continue
			}
			object.Metadata.Name += "-broker"
		}
		log.Printf("Deleting %q %s", object.Metadata.Name, strings.ToLower(object.Kind))
		if err := o.removeExternalServices(ctx, object); err != nil {
			log.Printf("WARN: external services are not deleted: %v", err)
		}
		o.stopContainer(ctx, object.Metadata.Name, client)
		o.removeObject(object.Metadata.Name, manifest)
		o.cleanupTriggers(object.Metadata.Name, manifest)
		o.cleanupSecrets(object.Metadata.Name, manifest)
	}
	return manifest.Write()
}

func (o *DeleteOptions) removeObject(component string, manifest *manifest.Manifest) {
	for _, object := range manifest.Objects {
		if component != object.Metadata.Name {
			continue
		}
		if object.Kind == "Trigger" {
			trigger := tmbroker.NewTrigger(object.Metadata.Name, o.Context, path.Join(o.ConfigBase, o.Context), nil)
			if err := trigger.RemoveTriggerFromConfig(); err != nil {
				log.Printf("Deleting %q: %v", object.Metadata.Name, err)
				continue
			}
		}
		manifest.Remove(object.Metadata.Name, object.Kind)
	}
}

func (o *DeleteOptions) stopContainer(ctx context.Context, name string, client *client.Client) error {
	return docker.ForceStop(ctx, name, client)
}

func (o *DeleteOptions) cleanupTriggers(component string, manifest *manifest.Manifest) {
	triggers, err := tmbroker.GetTargetTriggers(path.Join(o.ConfigBase, o.Context), component)
	if err != nil {
		return
	}
	for name, trigger := range triggers {
		if err := trigger.RemoveTriggerFromConfig(); err != nil {
			log.Printf("Deleting trigger %q: %v", trigger.Name, err)
			continue
		}
		manifest.Remove(name, trigger.GetKind())
	}
}

func (o *DeleteOptions) cleanupSecrets(component string, manifest *manifest.Manifest) {
	for _, object := range manifest.Objects {
		if object.Metadata.Name == component && object.Kind == "Secret" {
			manifest.Remove(component, object.Kind)
		}
	}
}

func (o *DeleteOptions) removeExternalServices(ctx context.Context, object kubernetes.Object) error {
	manifestPath := path.Join(o.ConfigBase, o.Context, manifestFile)
	component, err := components.GetObject(object.Metadata.Name, manifestPath, o.CRD, o.Version)
	if err != nil {
		return err
	}
	p, ok := component.(triggermesh.Parent)
	if !ok {
		return nil
	}
	secrets, _, err := triggermesh.ProcessSecrets(ctx, p, manifestPath)
	if err != nil {
		return err
	}
	return triggermesh.FinalizeExternalServices(ctx, component, secrets)
}

func (o *DeleteOptions) switchContext() error {
	list, err := brokers.List(o.ConfigBase, o.Context)
	if err != nil {
		return fmt.Errorf("list brokers: %w", err)
	}
	var context string
	if len(list) > 0 {
		context = list[0]
		log.Printf("Active broker is %q", context)
	}
	viper.Set("context", context)
	return viper.WriteConfig()
}

func (o *DeleteOptions) deleteCompletion(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return append(completion.ListAll(path.Join(o.ConfigBase, o.Context, manifestFile)), "--broker"),
			cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

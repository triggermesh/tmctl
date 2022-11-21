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

type DeleteOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &DeleteOptions{}
	var broker string
	deleteCmd := &cobra.Command{
		Use:               "delete <component_name_1, component_name_2...> [--broker <name>]",
		Short:             "Delete components by names",
		ValidArgsFunction: o.deleteCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if broker != "" {
				return o.deleteBroker(broker)
			}
			if len(args) == 0 {
				return fmt.Errorf("expected at least 1 component name, got 0")
			}
			return o.deleteComponents(args, false)
		},
	}
	cobra.OnInitialize(o.initialize)
	deleteCmd.Flags().StringVar(&broker, "broker", "", "Delete the broker")
	cobra.CheckErr(deleteCmd.RegisterFlagCompletionFunc("broker", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		list, err := brokers.List(path.Dir(viper.ConfigFileUsed()), "")
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		return list, cobra.ShellCompDirectiveNoFileComp
	}))
	return deleteCmd
}

func (o *DeleteOptions) initialize() {
	o.ConfigBase = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = manifest.New(path.Join(o.ConfigBase, o.Context, triggermesh.ManifestFile))
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds

	// try to read manifest even if it does not exists.
	// required for autocompletion.
	_ = o.Manifest.Read()
}

func (o *DeleteOptions) deleteBroker(broker string) error {
	oo := *o
	oo.Context = broker
	oo.Manifest = manifest.New(path.Join(oo.ConfigBase, broker, triggermesh.ManifestFile))
	cobra.CheckErr(oo.Manifest.Read())

	if err := oo.deleteComponents([]string{}, true); err != nil {
		return fmt.Errorf("deleting component: %w", err)
	}
	if err := os.RemoveAll(path.Join(oo.ConfigBase, broker)); err != nil {
		return fmt.Errorf("delete broker %q: %v", broker, err)
	}
	if broker == o.Context {
		return o.switchContext()
	}
	return nil
}

func (o *DeleteOptions) deleteComponents(names []string, deleteBroker bool) error {
	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	for _, object := range o.Manifest.Objects {
		if object.Kind == "Secret" {
			// do not remove secrets ,
			// we may need them to finalize external services
			continue
		}
		if deleteBroker {
			o.deleteEverything(ctx, object, client)
			continue
		}
		skip := true
		for _, v := range names {
			if v == object.Metadata.Name {
				skip = false
				break
			}
		}
		if skip {
			continue
		}
		if object.Kind == tmbroker.BrokerKind {
			log.Printf("use \"tmctl delete --broker %s\" to delete the broker. Skipping", object.Metadata.Name)
			continue
		}
		o.deleteEverything(ctx, object, client)
	}
	return nil
}

func (o *DeleteOptions) deleteEverything(ctx context.Context, object kubernetes.Object, client *client.Client) {
	log.Printf("Deleting %q %s", object.Metadata.Name, strings.ToLower(object.Kind))
	if object.Kind == tmbroker.BrokerKind {
		object.Metadata.Name = object.Metadata.Name + "-broker"
	}
	if err := o.removeExternalServices(ctx, object); err != nil {
		log.Printf("WARN: external services are not deleted: %v", err)
	}
	// not all components are runnable, but removeContainer should try to stop it anyway
	_ = o.removeContainer(ctx, object.Metadata.Name, client)
	o.removeObject(object.Metadata.Name)
	o.cleanupTriggers(object.Metadata.Name)
	o.cleanupSecrets(object.Metadata.Name)
}

func (o *DeleteOptions) removeObject(component string) {
	for _, object := range o.Manifest.Objects {
		if component != object.Metadata.Name {
			continue
		}
		if object.Kind == tmbroker.TriggerKind {
			trigger, err := tmbroker.NewTrigger(object.Metadata.Name, o.Context, o.ConfigBase, nil, nil)
			if err != nil {
				log.Printf("Creating trigger object %q: %v", object.Metadata.Name, err)
				continue
			}
			if err := trigger.(*tmbroker.Trigger).RemoveFromLocalConfig(); err != nil {
				log.Printf("Updating broker config %q: %v", object.Metadata.Name, err)
			}
		}
		if err := o.Manifest.Remove(object.Metadata.Name, object.Kind); err != nil {
			log.Printf("Deleting %q: %v", object.Metadata.Name, err)
		}
	}
}

func (o *DeleteOptions) removeContainer(ctx context.Context, name string, client *client.Client) error {
	return docker.ForceStop(ctx, name, client)
}

func (o *DeleteOptions) cleanupTriggers(target string) {
	triggers, err := tmbroker.GetTargetTriggers(target, o.Context, o.ConfigBase)
	if err != nil {
		return
	}
	for _, trigger := range triggers {
		if err := trigger.(*tmbroker.Trigger).RemoveFromLocalConfig(); err != nil {
			log.Printf("Updating broker config %q: %v", trigger.GetName(), err)
			continue
		}
		if err := o.Manifest.Remove(trigger.GetName(), trigger.GetKind()); err != nil {
			log.Printf("Deleting trigger %q: %v", trigger.GetName(), err)
		}
	}
}

func (o *DeleteOptions) cleanupSecrets(component string) {
	for _, object := range o.Manifest.Objects {
		if object.Metadata.Name == component+"-secret" && object.Kind == "Secret" {
			if err := o.Manifest.Remove(object.Metadata.Name, object.Kind); err != nil {
				log.Printf("Deleting secret %q: %v", object.Metadata.Name, err)
			}
		}
	}
}

func (o *DeleteOptions) removeExternalServices(ctx context.Context, object kubernetes.Object) error {
	component, err := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest)
	if err != nil {
		return err
	}
	r, ok := component.(triggermesh.Reconcilable)
	if !ok {
		return nil
	}
	p, ok := component.(triggermesh.Parent)
	if !ok {
		return nil
	}
	_, secretsEnv, err := components.ProcessSecrets(p, o.Manifest)
	if err != nil {
		return fmt.Errorf("secrets extraction: %w", err)
	}
	return r.Finalize(ctx, secretsEnv)
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
		return append(completion.ListAll(o.Manifest), "--broker"),
			cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

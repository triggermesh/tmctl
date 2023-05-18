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
	"strings"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
	CRD      map[string]crd.CRD
}

func NewCmd(config *config.Config, manifest *manifest.Manifest, crds map[string]crd.CRD) *cobra.Command {
	o := &CliOptions{
		CRD:      crds,
		Config:   config,
		Manifest: manifest,
	}
	deleteCmd := &cobra.Command{
		Use:   "delete <kind> <name>",
		Short: "Delete TriggerMesh component",
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
		Args: cobra.MinimumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cobra.CheckErr(docker.CheckDaemon())
			if cmd.Name() != "broker" {
				cobra.CheckErr(o.Manifest.Read())
			}
		},
	}
	deleteCmd.AddCommand(o.deleteBrokerCmd())
	deleteCmd.AddCommand(o.deleteSourceCmd())
	deleteCmd.AddCommand(o.deleteTargetCmd())
	deleteCmd.AddCommand(o.deleteTransformationCmd())
	deleteCmd.AddCommand(o.deleteTriggerCmd())
	return deleteCmd
}

func (o *CliOptions) deleteBrokerComponents(names []string, deleteBroker bool) error {
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

func (o *CliOptions) deleteEverything(ctx context.Context, object kubernetes.Object, client *client.Client) {
	log.Printf("Deleting %q %s", object.Metadata.Name, strings.ToLower(object.Kind))
	if object.Kind == tmbroker.BrokerKind {
		object.Metadata.Name = object.Metadata.Name + "-broker"
	}
	if err := o.removeExternalServices(ctx, object); err != nil && !strings.HasPrefix(err.Error(), "Unsubscribed from topic") {
		log.Printf("WARNING: external services are not deleted: %v", err)
	}
	// not all components are runnable, but removeContainer should try to stop it anyway
	_ = o.removeContainer(ctx, object.Metadata.Name, client)
	o.removeObject(object.Metadata.Name)
	o.cleanupTriggers(object.Metadata.Name)
	o.cleanupSecrets(object.Metadata.Name)
}

func (o *CliOptions) removeObject(component string) {
	for _, object := range o.Manifest.Objects {
		if component != object.Metadata.Name {
			continue
		}
		if object.Kind == tmbroker.TriggerKind {
			trigger, err := tmbroker.NewTrigger(object.Metadata.Name, o.Config.Context, o.Config.ConfigHome, nil, nil)
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

func (o *CliOptions) removeContainer(ctx context.Context, name string, client *client.Client) error {
	return docker.ForceStop(ctx, name, client)
}

func (o *CliOptions) cleanupTriggers(target string) {
	triggers, err := tmbroker.GetTargetTriggers(target, o.Config.Context, o.Config.ConfigHome)
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

func (o *CliOptions) cleanupSecrets(component string) {
	for _, object := range o.Manifest.Objects {
		if object.Metadata.Name == component+"-secret" && object.Kind == "Secret" {
			if err := o.Manifest.Remove(object.Metadata.Name, object.Kind); err != nil {
				log.Printf("Deleting secret %q: %v", object.Metadata.Name, err)
			}
		}
	}
}

func (o *CliOptions) removeExternalServices(ctx context.Context, object kubernetes.Object) error {
	component, err := components.GetObject(object.Metadata.Name, o.Config, o.Manifest, o.CRD)
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

func (o *CliOptions) switchContext() error {
	list, err := brokers.List(o.Config.ConfigHome, o.Config.Context)
	if err != nil {
		return fmt.Errorf("list brokers: %w", err)
	}
	var context string
	if len(list) > 0 {
		context = list[0]
		log.Printf("Active broker is %q", context)
	}
	o.Config.Context = context
	return o.Config.Save()
}

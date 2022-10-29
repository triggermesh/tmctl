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

package start

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

type StartOptions struct {
	ConfigBase string
	Context    string
	Version    string
	Manifest   string
	Restart    bool
	CRD        string
}

func NewCmd() *cobra.Command {
	o := &StartOptions{}
	createCmd := &cobra.Command{
		Use:   "start [broker]",
		Short: "Starts TriggerMesh components",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"--broker", "--restart", "--version"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return o.start(args[0])
			}
			return o.start(viper.GetString("context"))
		},
	}
	cobra.OnInitialize(o.initialize)
	createCmd.Flags().BoolVar(&o.Restart, "restart", false, "Restart components")

	return createCmd
}

func (o *StartOptions) initialize() {
	o.ConfigBase = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = path.Join(o.ConfigBase, o.Context, manifestFile)
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds
}

func (o *StartOptions) start(broker string) error {
	ctx := context.Background()
	manifest := manifest.New(o.Manifest)
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	componentTriggers := make(map[string][]triggermesh.Component)
	var brokerPort string
	// start eventing first
	for _, object := range manifest.Objects {
		switch object.Kind {
		case "Broker":
			broker, err := tmbroker.New(object.Metadata.Name, o.Manifest)
			if err != nil {
				return fmt.Errorf("creating broker object: %w", err)
			}
			log.Println("Starting broker")
			container, err := broker.(triggermesh.Runnable).Start(ctx, nil, o.Restart)
			if err != nil {
				return fmt.Errorf("starting broker container: %w", err)
			}
			brokerPort = container.HostPort()
		case "Trigger":
			t := tmbroker.NewTrigger(object.Metadata.Name, broker, o.ConfigBase, nil)
			trigger := t.(*tmbroker.Trigger)
			if err := trigger.LookupTrigger(); err != nil {
				return fmt.Errorf("trigger configuration: %w", err)
			}
			if triggers, set := componentTriggers[trigger.GetTarget().Component]; set {
				componentTriggers[trigger.GetTarget().Component] = append(triggers, trigger)
			} else {
				componentTriggers[trigger.GetTarget().Component] = []triggermesh.Component{trigger}
			}
			if _, err := t.Add(o.Manifest); err != nil {
				return fmt.Errorf("creating trigger: %w", err)
			}
		}
	}

	for i, object := range manifest.Objects {
		var c triggermesh.Component
		switch object.APIVersion {
		case "sources.triggermesh.io/v1alpha1":
			manifest.Objects[i].Spec["sink"] = map[string]interface{}{"uri": fmt.Sprintf("http://host.docker.internal:%s", brokerPort)}
			c = source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
		case "targets.triggermesh.io/v1alpha1":
			c = target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
		case "flow.triggermesh.io/v1alpha1":
			c = transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
		}
		if c == nil {
			continue
		}

		secrets := make(map[string]string, 0)
		if parent, ok := c.(triggermesh.Parent); ok {
			s, _, err := triggermesh.ProcessSecrets(ctx, parent, o.Manifest)
			if err != nil {
				return fmt.Errorf("processing secrets: %w", err)
			}
			secrets = s
		}
		if reconcilable, ok := c.(triggermesh.Reconcilable); ok {
			status, err := reconcilable.Initialize(ctx, secrets)
			if err != nil {
				return fmt.Errorf("external services initialization: %w", err)
			}
			reconcilable.UpdateStatus(status)
		}
		log.Printf("Starting %s\n", object.Metadata.Name)
		if _, err := c.(triggermesh.Runnable).Start(ctx, secrets, o.Restart); err != nil {
			return fmt.Errorf("starting container: %w", err)
		}
		if consumer, ok := c.(triggermesh.Consumer); ok {
			port, err := consumer.GetPort(ctx)
			if err != nil {
				return fmt.Errorf("container port: %w", err)
			}
			if triggers, exists := componentTriggers[object.Metadata.Name]; exists {
				for _, t := range triggers {
					trigger := t.(*tmbroker.Trigger)
					trigger.SetTarget(object.Metadata.Name, fmt.Sprintf("http://host.docker.internal:%s", port))
					if err := trigger.UpdateBrokerConfig(); err != nil {
						return fmt.Errorf("broker config: %w", err)
					}
					if _, err := t.Add(o.Manifest); err != nil {
						return fmt.Errorf("broker manifest: %w", err)
					}
				}
			}
		}
		if _, err := c.Add(o.Manifest); err != nil {
			return fmt.Errorf("updating manifest: %w", err)
		}
	}
	return nil
}

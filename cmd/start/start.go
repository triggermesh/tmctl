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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type startOptions struct {
	ConfigBase string
	Context    string
	Version    string
	Restart    bool
	CRD        string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &startOptions{}
	createCmd := &cobra.Command{
		Use:     "start [broker]",
		Short:   "Starts TriggerMesh components",
		Example: "tmctl start",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"--broker", "--restart", "--version"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Manifest.Read(); err != nil {
				return fmt.Errorf("cannot read manifest. Does the broker exist?")
			}
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

func (o *startOptions) initialize() {
	o.ConfigBase = filepath.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = manifest.New(filepath.Join(o.ConfigBase, o.Context, triggermesh.ManifestFile))
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds
}

func (o *startOptions) start(broker string) error {
	ctx := context.Background()
	var brokerPort string
	// start eventing first
	for _, object := range o.Manifest.Objects {
		if object.Kind == tmbroker.BrokerKind {
			b, err := tmbroker.New(object.Metadata.Name, o.Manifest.Path)
			if err != nil {
				return fmt.Errorf("creating broker object: %w", err)
			}
			log.Println("Starting broker")
			container, err := b.(triggermesh.Runnable).Start(ctx, nil, o.Restart)
			if err != nil {
				return fmt.Errorf("starting broker container: %w", err)
			}
			brokerPort = container.HostPort()
		}
	}

	for _, object := range o.Manifest.Objects {
		if object.APIVersion == tmbroker.APIVersion {
			continue
		}
		c, _ := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest)
		if c == nil {
			continue
		}
		if _, ok := c.(triggermesh.Producer); ok {
			sink := "http://host.docker.internal:" + brokerPort
			spec := c.GetSpec()
			if spec == nil {
				spec = make(map[string]interface{})
			}
			if service, ok := c.(*service.Service); ok && service.IsSource() {
				spec["K_SINK"] = sink
			} else {
				spec["sink"] = map[string]interface{}{"uri": sink}
			}
		}
		secrets := make(map[string]string, 0)
		if parent, ok := c.(triggermesh.Parent); ok {
			_, secretsEnv, err := components.ProcessSecrets(parent, o.Manifest)
			if err != nil {
				return fmt.Errorf("processing secrets: %w", err)
			}
			secrets = secretsEnv
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
		if _, ok := c.(triggermesh.Consumer); ok {
			triggers, err := tmbroker.GetTargetTriggers(c.GetName(), o.Context, o.ConfigBase)
			if err != nil {
				return fmt.Errorf("%q target triggers: %w", c.GetName(), err)
			}
			for _, t := range triggers {
				t.(*tmbroker.Trigger).SetTarget(c)
				if err := t.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
					return fmt.Errorf("updating broker config: %w", err)
				}
			}
		}
	}
	return nil
}

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

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
	CRD      map[string]crd.CRD

	Restart bool
}

func NewCmd(config *config.Config, m *manifest.Manifest, crd map[string]crd.CRD) *cobra.Command {
	var version string
	o := &CliOptions{
		CRD:      crd,
		Config:   config,
		Manifest: m,
	}
	startCmd := &cobra.Command{
		Use:     "start [broker]",
		Short:   "Starts TriggerMesh components",
		Example: "tmctl start",
		Args:    cobra.RangeArgs(0, 1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"--restart", "--version"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				o.Config.Context = args[0]
				o.Manifest = manifest.New(filepath.Join(
					o.Config.ConfigHome,
					o.Config.Context,
					triggermesh.ManifestFile))
			}
			cobra.CheckErr(o.Manifest.Read())
			return o.start(version)
		},
	}
	startCmd.Flags().BoolVar(&o.Restart, "restart", false, "Restart components")
	startCmd.Flags().StringVar(&version, "version", o.Config.Triggermesh.Broker.Version, "TriggerMesh broker version.")

	return startCmd
}

func (o *CliOptions) start(version string) error {
	ctx := context.Background()
	var brokerPort string
	// start eventing first
	for _, object := range o.Manifest.Objects {
		if object.Kind == tmbroker.BrokerKind {
			o.Config.Triggermesh.Broker.Version = version
			b, err := tmbroker.New(object.Metadata.Name, o.Config.Triggermesh.Broker)
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
		c, _ := components.GetObject(object.Metadata.Name, o.Config, o.Manifest, o.CRD)
		if c == nil {
			continue
		}
		if _, ok := c.(triggermesh.Runnable); !ok {
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
			return fmt.Errorf("starting component %q: %w", c.GetName(), err)
		}
		if _, ok := c.(triggermesh.Consumer); ok {
			triggers, err := tmbroker.GetTargetTriggers(c.GetName(), o.Config.Context, o.Config.ConfigHome)
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

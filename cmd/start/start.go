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
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/target"
)

const manifestFile = "manifest.yaml"

type StartOptions struct {
	ConfigDir string
	Version   string
	Restart   bool
	CRD       string
}

func NewCmd() *cobra.Command {
	o := &StartOptions{}
	createCmd := &cobra.Command{
		Use:   "start <broker>",
		Short: "starts TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Version = viper.GetString("triggermesh.version")
			o.CRD = viper.GetString("triggermesh.servedCRD")

			if len(args) != 1 {
				return fmt.Errorf("expected only 1 argument")
			}

			return o.start(args[0])
		},
	}
	createCmd.Flags().BoolVar(&o.Restart, "restart", false, "Restart components")

	return createCmd
}

func (o *StartOptions) start(broker string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigDir, broker)
	manifestFile := path.Join(configDir, manifestFile)
	manifestOrig := manifest.New(manifestFile)
	if err := manifestOrig.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	manifestWoEventing := manifestOrig
	componentTriggers := make(map[string]*tmbroker.Trigger)
	var brokerPort string
	// start eventing first
	for i, object := range manifestOrig.Objects {
		switch object.Kind {
		case "Broker":
			manifestWoEventing.Objects = append(manifestOrig.Objects[:i], manifestOrig.Objects[i+1:]...)
			broker, err := tmbroker.NewBroker(object.Metadata.Name, configDir)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}
			container, err := triggermesh.Start(ctx, broker, true)
			if err != nil {
				return fmt.Errorf("starting broker container: %v", err)
			}
			brokerPort = container.HostPort()
		case "Trigger":
			manifestWoEventing.Objects = append(manifestOrig.Objects[:i], manifestOrig.Objects[i+1:]...)
			trigger := tmbroker.NewTrigger(object.Metadata.Name, broker, "", configDir)
			if err := trigger.LookupTrigger(); err != nil {
				return fmt.Errorf("trigger configuration: %v", err)
			}
			for _, target := range trigger.GetSpec().Targets {
				componentTriggers[target.Component] = trigger
			}
			if _, err := triggermesh.Create(ctx, trigger, manifestFile); err != nil {
				return fmt.Errorf("creating trigger: %v", err)
			}
		}
	}

	if brokerPort == "" {
		return fmt.Errorf("broker is not available")
	}

	for i, object := range manifestWoEventing.Objects {
		switch {
		case strings.HasSuffix(object.Kind, "Source"):
			manifestOrig.Objects[i].Spec["sink"] = map[string]interface{}{"uri": fmt.Sprintf("http://host.docker.internal:%s", brokerPort)}
			c := source.NewSource(o.CRD, object.Kind, broker, o.Version, manifestOrig.Objects[i].Spec)
			if _, err := triggermesh.Create(ctx, c, manifestFile); err != nil {
				return fmt.Errorf("creating object: %w", err)
			}
			if _, err := triggermesh.Start(ctx, c, true); err != nil {
				return fmt.Errorf("starting container: %w", err)
			}
		case strings.HasSuffix(object.Kind, "Target"):
			c := target.NewTarget(o.CRD, object.Kind, broker, o.Version, object.Spec)
			if _, err := triggermesh.Create(ctx, c, manifestFile); err != nil {
				return fmt.Errorf("creating object: %w", err)
			}
			container, err := triggermesh.Start(ctx, c, true)
			if err != nil {
				return fmt.Errorf("starting container: %w", err)
			}
			if trigger, exists := componentTriggers[object.Metadata.Name]; exists {
				trigger.SetTarget(object.Metadata.Name, fmt.Sprintf("http://host.docker.internal:%s", container.HostPort()))
				if err := trigger.UpdateBrokerConfig(); err != nil {
					return fmt.Errorf("broker config: %w", err)
				}
				if err := trigger.UpdateManifest(); err != nil {
					return fmt.Errorf("broker manifest: %w", err)
				}
			}
		}
	}
	return nil
}

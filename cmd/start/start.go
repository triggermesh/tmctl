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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/transformation"
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
		Use:   "start [broker]",
		Short: "starts TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Version = viper.GetString("triggermesh.version")
			o.CRD = viper.GetString("triggermesh.servedCRD")

			return o.start(broker)
		},
	}
	createCmd.Flags().BoolVar(&o.Restart, "restart", true, "Restart components")

	return createCmd
}

func (o *StartOptions) start(broker string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigDir, broker)
	manifestFile := path.Join(configDir, manifestFile)
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	componentTriggers := make(map[string]*tmbroker.Trigger)
	var brokerPort string
	// start eventing first
	for _, object := range manifest.Objects {
		switch object.Kind {
		case "Broker":
			broker, err := tmbroker.New(object.Metadata.Name, configDir)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}
			log.Println("Starting broker")
			container, err := triggermesh.Start(ctx, broker, o.Restart)
			if err != nil {
				return fmt.Errorf("starting broker container: %v", err)
			}
			brokerPort = container.HostPort()
		case "Trigger":
			trigger := tmbroker.NewTrigger(object.Metadata.Name, broker, configDir, []string{})
			if err := trigger.LookupTrigger(); err != nil {
				return fmt.Errorf("trigger configuration: %v", err)
			}
			for _, target := range trigger.GetTargets() {
				componentTriggers[target.Component] = trigger
			}
			if _, err := triggermesh.WriteObject(ctx, trigger, manifestFile); err != nil {
				return fmt.Errorf("creating trigger: %v", err)
			}
		}
	}

	if brokerPort == "" {
		return fmt.Errorf("broker is not available")
	}

	for i, object := range manifest.Objects {
		switch {
		case strings.HasSuffix(object.Kind, "Source"):
			manifest.Objects[i].Spec["sink"] = map[string]interface{}{"uri": fmt.Sprintf("http://host.docker.internal:%s", brokerPort)}
			c := source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			if _, err := triggermesh.WriteObject(ctx, c, manifestFile); err != nil {
				return fmt.Errorf("creating object: %w", err)
			}
			log.Printf("Starting %s\n", object.Metadata.Name)
			if _, err := triggermesh.Start(ctx, c, o.Restart); err != nil {
				return fmt.Errorf("starting container: %w", err)
			}
		case strings.HasSuffix(object.Kind, "Target") ||
			strings.HasSuffix(object.Kind, "Transformation"):

			var c triggermesh.Component
			if object.Kind == "Transformation" {
				c = transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			} else {
				c = target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			}

			if _, err := triggermesh.WriteObject(ctx, c, manifestFile); err != nil {
				return fmt.Errorf("creating object: %w", err)
			}
			log.Printf("Starting %s\n", object.Metadata.Name)
			container, err := triggermesh.Start(ctx, c.(triggermesh.Runnable), o.Restart)
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

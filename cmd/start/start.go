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
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
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
		Short: "Starts TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			o.Version = viper.GetString("triggermesh.version")
			crds, err := crd.Fetch(o.ConfigDir, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
			if len(args) == 1 {
				return o.start(args[0])
			}
			return o.start(viper.GetString("context"))
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

	componentTriggers := make(map[string][]*tmbroker.Trigger)
	secrets := make(map[string]map[string]string)
	var brokerPort string
	// start eventing first
	for _, object := range manifest.Objects {
		switch object.Kind {
		case "Broker":
			broker, err := tmbroker.New(object.Metadata.Name, configDir)
			if err != nil {
				return fmt.Errorf("creating broker object: %w", err)
			}
			log.Println("Starting broker")
			container, err := triggermesh.Start(ctx, broker, o.Restart, nil)
			if err != nil {
				return fmt.Errorf("starting broker container: %w", err)
			}
			brokerPort = container.HostPort()
		case "Trigger":
			trigger := tmbroker.NewTrigger(object.Metadata.Name, broker, configDir, nil)
			if err := trigger.LookupTrigger(); err != nil {
				return fmt.Errorf("trigger configuration: %w", err)
			}
			if triggers, set := componentTriggers[trigger.GetTarget().Component]; set {
				componentTriggers[trigger.GetTarget().Component] = append(triggers, trigger)
			} else {
				componentTriggers[trigger.GetTarget().Component] = []*tmbroker.Trigger{trigger}
			}
			if _, err := triggermesh.WriteObject(trigger, manifestFile); err != nil {
				return fmt.Errorf("creating trigger: %w", err)
			}
		case "Secret":
			secrets[strings.ToLower(object.Metadata.Name)] = object.Data
		}
	}

	if brokerPort == "" {
		return fmt.Errorf("broker is not available")
	}

	for i, object := range manifest.Objects {
		var objectSecrets map[string]string
		var c triggermesh.Runnable
		switch {
		case strings.HasSuffix(object.Kind, "Source"):
			manifest.Objects[i].Spec["sink"] = map[string]interface{}{"uri": fmt.Sprintf("http://host.docker.internal:%s", brokerPort)}
			c = source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			if s, ok := secrets[strings.ToLower(object.Metadata.Name)]; ok {
				objectSecrets = s
			}
			log.Printf("Starting %s\n", object.Metadata.Name)
			if _, err := triggermesh.Start(ctx, c, o.Restart, objectSecrets); err != nil {
				return fmt.Errorf("starting container: %w", err)
			}
		case strings.HasSuffix(object.Kind, "Target") ||
			strings.HasSuffix(object.Kind, "Transformation"):
			if object.Kind == "Transformation" {
				c = transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			} else {
				c = target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			}
			if s, ok := secrets[strings.ToLower(object.Metadata.Name)]; ok {
				objectSecrets = s
			}
			log.Printf("Starting %s\n", object.Metadata.Name)
			container, err := triggermesh.Start(ctx, c, o.Restart, objectSecrets)
			if err != nil {
				return fmt.Errorf("starting container: %w", err)
			}
			if triggers, exists := componentTriggers[object.Metadata.Name]; exists {
				for _, trigger := range triggers {
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
		if c == nil {
			continue
		}
		if _, err := triggermesh.WriteObject(c.(triggermesh.Component), manifestFile); err != nil {
			return fmt.Errorf("updating manifest: %w", err)
		}
	}
	return nil
}

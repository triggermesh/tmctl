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

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/manifest"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
)

const manifestFile = "manifest.yaml"

type DeleteOptions struct {
	ConfigDir string
	Context   string
}

func NewCmd() *cobra.Command {
	o := &DeleteOptions{}
	var deleteBroker string
	deleteCmd := &cobra.Command{
		Use:   "delete <component1, component2...>",
		Short: "Delete components",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = configDir
			o.Context = viper.GetString("context")
			if deleteBroker != "" {
				return o.deleteBroker(deleteBroker)
			}
			return o.deleteComponents(args)
		},
	}
	deleteCmd.Flags().StringVar(&deleteBroker, "broker", "", "Delete the broker")
	return deleteCmd
}

func (o *DeleteOptions) deleteBroker(broker string) error {
	oo := *o
	oo.Context = broker
	if err := oo.deleteComponents([]string{}); err != nil {
		return fmt.Errorf("deleting component: %w", err)
	}
	if err := os.RemoveAll(path.Join(o.ConfigDir, broker)); err != nil {
		return fmt.Errorf("delete broker %q: %v", broker, err)
	}
	return nil
}

func (o *DeleteOptions) deleteComponents(components []string) error {
	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	currentManifest := manifest.New(path.Join(o.ConfigDir, o.Context, manifestFile))
	if err := currentManifest.Read(); err != nil {
		return fmt.Errorf("manifest read: %w", err)
	}
	for _, object := range currentManifest.Objects {
		skip := false
		if len(components) > 0 {
			skip = true
			for _, v := range components {
				if v == object.Metadata.Name {
					skip = false
					break
				}
			}
		}
		if skip {
			continue
		}
		log.Printf("Deleting %q", object.Metadata.Name)
		if object.Kind == "Broker" {
			object.Metadata.Name += "-broker"
		}
		o.stopContainer(ctx, object.Metadata.Name, client)
		o.removeObject(object.Metadata.Name, currentManifest)
	}
	return currentManifest.Write()
}

func (o *DeleteOptions) removeObject(component string, manifest *manifest.Manifest) {
	for _, object := range manifest.Objects {
		if component != object.Metadata.Name {
			continue
		}
		if object.Kind == "Trigger" {
			trigger := tmbroker.NewTrigger(object.Metadata.Name, o.Context, path.Join(o.ConfigDir, o.Context), []string{})
			if err := trigger.RemoveTriggerFromConfig(); err != nil {
				log.Printf("Deleting %q: %v", object.Metadata.Name, err)
				continue
			}
		}
		manifest.Remove(object.Metadata.Name)
	}
}

func (o *DeleteOptions) stopContainer(ctx context.Context, name string, client *client.Client) error {
	return docker.ForceStop(ctx, name, client)
}

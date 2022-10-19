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

package stop

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/manifest"
)

const manifestFile = "manifest.yaml"

type StopOptions struct {
	ConfigDir string
}

func NewCmd() *cobra.Command {
	o := &StopOptions{}
	stopCmd := &cobra.Command{
		Use:   "stop [broker]",
		Short: "Stops TriggerMesh components",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			if len(args) == 1 {
				return o.stop(args[0])
			}
			return o.stop(viper.GetString("context"))
		},
	}

	return stopCmd
}

func (o *StopOptions) stop(broker string) error {
	ctx := context.Background()
	manifest := manifest.New(path.Join(o.ConfigDir, broker, manifestFile))
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	for _, object := range manifest.Objects {
		if object.Kind == "Trigger" || object.Kind == "Secret" {
			continue
		}
		if object.Kind == "Broker" {
			object.Metadata.Name += "-broker"
		}
		log.Printf("Stopping %s\n", object.Metadata.Name)
		if err := docker.ForceStop(ctx, object.Metadata.Name, client); err != nil {
			log.Printf("Stopping %q: %v", object.Metadata.Name, err)
		}
	}
	return nil
}

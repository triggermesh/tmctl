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
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/manifest"
)

const manifestFile = "manifest.yaml"

type StopOptions struct {
	ConfigDir string
}

func NewCmd() *cobra.Command {
	o := &StopOptions{}
	stopCmd := &cobra.Command{
		Use:   "stop <broker>",
		Short: "stops TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			if len(args) != 1 {
				return fmt.Errorf("expected only 1 argument")
			}

			return o.stop(args[0])
		},
	}
	// createCmd.Flags().StringVarP(&o.Context, "broker", "b", "", "Connect components to this broker")

	return stopCmd
}

func (o *StopOptions) stop(broker string) error {
	ctx := context.Background()
	manifest := manifest.New(path.Join(o.ConfigDir, broker+"-broker", manifestFile))
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(manifest.Objects))

	for i, object := range manifest.Objects {
		go func(i int, object kubernetes.Object) {
			name := fmt.Sprintf("%s-%s", broker, strings.ToLower(object.Kind))
			if err := docker.ForceStop(ctx, name, client); err != nil {
				log.Printf("Stopping %q: %v", name, err)
			}
			wg.Done()
		}(i, object)
	}
	wg.Wait()
	return nil
}

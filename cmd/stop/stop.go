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
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
}

func NewCmd(config *config.Config, m *manifest.Manifest) *cobra.Command {
	o := &CliOptions{
		Config:   config,
		Manifest: m,
	}
	return &cobra.Command{
		Use:     "stop [broker]",
		Short:   "Stops TriggerMesh components, removes docker containers",
		Example: "tmctl stop",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				o.Config.Context = args[0]
				o.Manifest = manifest.New(filepath.Join(
					o.Config.ConfigHome,
					o.Config.Context,
					triggermesh.ManifestFile))
			}
			cobra.CheckErr(o.Manifest.Read())
			return o.stop()
		},
	}
}

func (o *CliOptions) stop() error {
	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

	for _, object := range o.Manifest.Objects {
		if object.Kind == tmbroker.TriggerKind || object.Kind == "Secret" {
			continue
		}
		if object.Kind == tmbroker.BrokerKind {
			object.Metadata.Name += "-broker"
		}
		log.Printf("Stopping %s\n", object.Metadata.Name)
		if err := docker.ForceStop(ctx, object.Metadata.Name, client); err != nil {
			log.Printf("Stopping %q: %v", object.Metadata.Name, err)
		}
	}
	return nil
}

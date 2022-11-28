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
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

type stopOptions struct {
	ConfigBase string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &stopOptions{}
	stopCmd := &cobra.Command{
		Use:       "stop [broker]",
		Short:     "Stops TriggerMesh components, removes docker containers",
		Example:   "tmctl stop",
		ValidArgs: []string{},
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			configBase, err := filepath.Abs(path.Dir(viper.ConfigFileUsed()))
			if err != nil {
				return err
			}
			o.ConfigBase = configBase
			o.Manifest = manifest.New(path.Join(o.ConfigBase, broker, triggermesh.ManifestFile))
			cobra.CheckErr(o.Manifest.Read())
			return o.stop(broker)
		},
	}
	return stopCmd
}

func (o *stopOptions) stop(broker string) error {
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

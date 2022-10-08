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

package create

import (
	"context"
	"fmt"
	"log"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/output"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
)

func (o *CreateOptions) NewBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "broker <name>",
		Short: "TriggerMesh broker",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			if len(args) != 1 {
				return fmt.Errorf("broker name is required")
			}
			return o.broker(args[0])
		},
	}
}

func (o *CreateOptions) broker(name string) error {
	ctx := context.Background()

	configDir := path.Join(o.ConfigBase, name)
	broker, err := tmbroker.New(name, configDir)
	if err != nil {
		return fmt.Errorf("broker: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(ctx, broker, path.Join(configDir, manifestFile))
	if err != nil {
		return err
	}

	log.Println("Starting container")
	if _, err := triggermesh.Start(ctx, broker, restart, nil); err != nil {
		return err
	}

	viper.Set("context", name)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	output.PrintStatus("broker", broker, "", []string{})
	return nil
}

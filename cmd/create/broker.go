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
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/triggermesh/tmcli/pkg/runtime"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
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
			return o.Broker(args[0])
		},
	}
}

func (o *CreateOptions) Broker(name string) error {
	ctx := context.Background()
	manifest := path.Join(o.ConfigBase, name, manifestFile)

	object, dirty, err := tmbroker.CreateBrokerObject(name, manifest)
	if err != nil {
		return fmt.Errorf("broker creation: %w", err)
	}
	viper.Set("context", object.Metadata.Name)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if _, err := runtime.Initialize(ctx, object, o.Version, dirty); err != nil {
		return fmt.Errorf("container initialization: %w", err)
	}
	return nil
}

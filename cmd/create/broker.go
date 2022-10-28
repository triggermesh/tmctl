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

	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

func (o *CreateOptions) NewBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use: "broker <name>",
		// Short: "TriggerMesh broker",
		Args: cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Manifest = path.Join(o.ConfigBase, args[0], manifestFile)
			return o.broker(args[0])
		},
	}
}

func (o *CreateOptions) broker(name string) error {
	ctx := context.Background()
	broker, err := tmbroker.New(name, o.Manifest)
	if err != nil {
		return fmt.Errorf("broker: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := broker.Add(o.Manifest)
	if err != nil {
		return err
	}

	log.Println("Starting container")
	if _, err := broker.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}

	viper.Set("context", name)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	output.PrintStatus("broker", broker, []string{}, []string{})
	return nil
}

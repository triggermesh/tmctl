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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

func (o *CliOptions) newBrokerCmd() *cobra.Command {
	var version string
	brokerCmd := &cobra.Command{
		Use:               "broker <name>",
		Short:             "Create TriggerMesh Broker. More information at https://docs.triggermesh.io/brokers/",
		Example:           "tmctl create broker foo",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.broker(args[0], version)
		},
	}
	brokerCmd.Flags().StringVar(&version, "version", o.Config.Triggermesh.Broker.Version, "TriggerMesh broker version.")
	return brokerCmd
}

func (o *CliOptions) broker(name, version string) error {
	ctx := context.Background()
	o.Manifest.Path = filepath.Join(o.Config.ConfigHome, name, triggermesh.ManifestFile)
	if _, err := os.Stat(o.Manifest.Path); !os.IsNotExist(err) {
		return fmt.Errorf("broker %q already exists", name)
	}

	if _, err := tmbroker.CreateBrokerConfig(o.Config.ConfigHome, name); err != nil {
		return fmt.Errorf("creating broker config: %w", err)
	}

	brokerConfig := o.Config.Triggermesh.Broker
	brokerConfig.Version = version

	broker, err := tmbroker.New(name, brokerConfig)
	if err != nil {
		return fmt.Errorf("broker: %w", err)
	}

	if err := o.Manifest.Read(); err != nil {
		return fmt.Errorf("broker manifest: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(broker)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}

	o.Config.Context = name
	if err := o.Config.Save(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	log.Println("Starting container")
	if _, err := broker.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}

	output.PrintStatus("broker", broker, []string{}, []string{})
	return nil
}

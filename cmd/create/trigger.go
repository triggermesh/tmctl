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
	"fmt"
	"path"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmcli/pkg/triggermesh/broker"
)

func (o *CreateOptions) NewTriggerCmd() *cobra.Command {
	var eventType string
	triggerCmd := &cobra.Command{
		Use:   "trigger <name> <event type>",
		Short: "TriggerMesh trigger",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			if len(args) < 1 {
				return fmt.Errorf("trigger name is required")
			}
			return o.trigger(args[0], eventType)
		},
	}
	triggerCmd.Flags().StringVarP(&eventType, "eventType", "e", "", "Filter data based on the event type")
	return triggerCmd
}

func (o *CreateOptions) trigger(name, eventType string) error {
	configDir := path.Join(o.ConfigBase, o.Context)
	trigger := broker.NewTrigger(name, o.Context, eventType, configDir)

	if err := trigger.UpdateBrokerConfig(); err != nil {
		return fmt.Errorf("broker config update: %w", err)
	}

	if err := trigger.UpdateManifest(); err != nil {
		return fmt.Errorf("manifest update: %w", err)
	}
	return nil
}

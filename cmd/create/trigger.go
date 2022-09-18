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
			return o.Trigger(args[0], eventType)
		},
	}
	triggerCmd.Flags().StringVarP(&eventType, "eventType", "e", "", "Filter data based on the event type")
	return triggerCmd
}

func (o *CreateOptions) Trigger(name, eventType string) error {
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)
	_, err := broker.CreateTrigger(name, manifest, o.Context, eventType)
	if err != nil {
		return fmt.Errorf("trigger creation: %w", err)
	}
	return nil
}

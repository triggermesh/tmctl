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
	"github.com/spf13/viper"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

func (o *CreateOptions) NewTriggerCmd() *cobra.Command {
	triggerCmd := &cobra.Command{
		Use:   "trigger <event type> <target>",
		Short: "TriggerMesh trigger",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigBase = c
			o.Context = viper.GetString("context")
			return o.Trigger(args)
		},
	}

	return triggerCmd
}

func (o *CreateOptions) Trigger(args []string) error {
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)

	_, _, err := triggermesh.CreateTrigger(manifest, o.Context)
	if err != nil {
		return fmt.Errorf("trigger creation: %w", err)
	}

	return nil
}

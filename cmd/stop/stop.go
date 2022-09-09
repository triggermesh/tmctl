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
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/triggermesh/tmcli/pkg/runtime"
)

const manifestFile = "manifest.yaml"

type StartOptions struct {
	ConfigDir string
}

func NewCmd() *cobra.Command {
	o := &StartOptions{}
	createCmd := &cobra.Command{
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

	return createCmd
}

func (o *StartOptions) stop(broker string) error {
	manifest := path.Join(o.ConfigDir, broker, manifestFile)
	return runtime.NewLocalSetup(manifest, "", []string{}).StopAll()
}

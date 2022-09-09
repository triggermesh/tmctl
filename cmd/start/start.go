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

package start

import (
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/triggermesh/tmcli/pkg/runtime"
)

const manifestFile = "manifest.yaml"

type StartOptions struct {
	ConfigDir string
	Version   string
	Restart   bool
}

func NewCmd() *cobra.Command {
	o := &StartOptions{}
	createCmd := &cobra.Command{
		Use:   "start <broker>",
		Short: "starts TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Version = viper.GetString("triggermesh.version")

			if len(args) != 1 {
				return fmt.Errorf("expected only 1 argument")
			}

			return o.start(args[0])
		},
	}
	createCmd.Flags().BoolVar(&o.Restart, "restart", false, "Restart components")

	return createCmd
}

func (o *StartOptions) start(broker string) error {
	manifestFile := path.Join(o.ConfigDir, broker, manifestFile)
	return runtime.NewLocalSetup(manifestFile, o.Version, []string{}).RunAll(o.Restart)
}

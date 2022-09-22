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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const manifestFile = "manifest.yaml"

type CreateOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
}

func NewCmd() *cobra.Command {
	o := &CreateOptions{}
	createCmd := &cobra.Command{
		Use:   "create <resource>",
		Short: "create TriggerMesh objects",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	createCmd.AddCommand(o.NewBrokerCmd())
	createCmd.AddCommand(o.NewSourceCmd())
	createCmd.AddCommand(o.NewTargetCmd())
	createCmd.AddCommand(o.NewTransformationCmd())

	// createCmd.Flags().StringVarP(&o.Context, "broker", "b", "", "Connect components to this broker")

	return createCmd
}

func (o *CreateOptions) initializeOptions(cmd *cobra.Command) {
	configBase, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	o.ConfigBase = configBase
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.CRD = viper.GetString("triggermesh.servedCRD")
}

func parse(args []string) (string, []string, error) {
	if l := len(args); l < 1 {
		return "", []string{}, fmt.Errorf("expected at least 1 arguments, got %d", l)
	}
	return args[0], args[1:], nil
}

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

package config

import (
	"fmt"

	"github.com/spf13/cobra"

	cliconfig "github.com/triggermesh/tmctl/pkg/config"
)

func NewCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config [set|get]",
		Short: "Read and write config values",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	configCmd.AddCommand(getCmd())
	configCmd.AddCommand(setCmd())
	return configCmd
}

func getCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Read config value",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := ""
			if len(args) == 1 {
				key = args[0]
			}
			value, err := cliconfig.Get(key)
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		},
	}
}

func setCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Write config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cliconfig.Set(args[0], args[1])
		},
	}
}

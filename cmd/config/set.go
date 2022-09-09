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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Write config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value, err := parseSetCmd(args)
			if err != nil {
				return err
			}
			return set(key, value)
		},
	}
}

func parseSetCmd(args []string) (string, string, error) {
	var key, value string
	switch len(args) {
	case 1:
		kv := strings.Split(args[0], "=")
		if len(kv) != 2 {
			return "", "", fmt.Errorf("expected key-value pair, found %q", args)
		}
		key = kv[0]
		value = kv[1]
	case 2:
		key = args[0]
		value = args[1]
	default:
		return "", "", fmt.Errorf("expected key-value pair, found %q", args)
	}
	return key, value, nil
}

func set(key, value string) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

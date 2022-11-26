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
	"github.com/spf13/viper"
)

func NewGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Show config value",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := ParseGetCmd(args)
			if err != nil {
				return err
			}
			get(key)
			return nil
		},
	}
}

func ParseGetCmd(args []string) (string, error) {
	switch len(args) {
	case 0:
		return "", nil
	case 1:
		return args[0], nil
	}
	return "", fmt.Errorf("unsupported number of args: %s", args)
}

func get(key string) {
	if key == "" {
		for _, k := range viper.AllKeys() {
			fmt.Printf("%s: %s\n", k, viper.Get(k))
		}
		return
	}
	fmt.Printf("%v\n", viper.Get(key))
}

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

package list

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const manifestFile = "manifest.yaml"

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show brokers list",
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			return list(configDir, viper.GetString("context"))
		},
	}
}

func list(configDir, currentContext string) error {
	dirs, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("listing dirs: %w", err)
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		files, err := os.ReadDir(path.Join(configDir, dir.Name()))
		if err != nil {
			return fmt.Errorf("listing files: %w", err)
		}
		for _, file := range files {
			if file.Name() == manifestFile {
				if dir.Name() == currentContext {
					fmt.Printf("*")
				}
				fmt.Println(dir.Name())
			}
		}
	}
	return nil
}

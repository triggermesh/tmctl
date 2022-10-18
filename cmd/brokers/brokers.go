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

package brokers

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const manifestFile = "manifest.yaml"

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "brokers",
		Aliases: []string{"list"},
		Short:   "Show the list of brokers",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := List(path.Dir(viper.ConfigFileUsed()), viper.GetString("context"))
			if err != nil {
				return err
			}
			if len(list) == 0 {
				return nil
			}
			fmt.Println(strings.Join(list, "\n"))
			return nil
		},
	}
}

func List(configDir, currentContext string) ([]string, error) {
	dirs, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("listing dirs: %w", err)
	}
	var output []string
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		files, err := os.ReadDir(path.Join(configDir, dir.Name()))
		if err != nil {
			return nil, fmt.Errorf("listing files: %w", err)
		}
		for _, file := range files {
			if file.Name() == manifestFile {
				if dir.Name() == currentContext {
					output = append(output, fmt.Sprintf("*%s", dir.Name()))
					continue
				}
				output = append(output, dir.Name())
			}
		}
	}
	return output, nil
}

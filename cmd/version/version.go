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

package version

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/docker"
)

func NewCmd(ver, commit string) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "CLI version information",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("CLI:")
			fmt.Println(" Version: ", ver)
			fmt.Println(" Commit: ", commit)
			fmt.Printf(" OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Println("\nTriggerMesh:")
			fmt.Println(" Components version: ", viper.GetString("triggermesh.version"))
			fmt.Println("\nDocker:")
			fmt.Println(" ", dockerVersion())
		},
	}
	return versionCmd
}

func dockerVersion() string {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Sprintf("Not available (%v)", err)
	}
	ver, err := client.ServerVersion(context.Background())
	if err != nil {
		return fmt.Sprintf("Not available (%v)", err)
	}
	return ver.Platform.Name
}

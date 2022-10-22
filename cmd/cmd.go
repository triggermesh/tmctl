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

package cmd

import (
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/cmd/config"
	"github.com/triggermesh/tmctl/cmd/create"
	"github.com/triggermesh/tmctl/cmd/delete"
	"github.com/triggermesh/tmctl/cmd/describe"
	"github.com/triggermesh/tmctl/cmd/dump"
	"github.com/triggermesh/tmctl/cmd/sendevent"
	"github.com/triggermesh/tmctl/cmd/start"
	"github.com/triggermesh/tmctl/cmd/stop"
	"github.com/triggermesh/tmctl/cmd/version"
	"github.com/triggermesh/tmctl/cmd/watch"
)

const (
	defaultTriggermeshVersion = "v1.21.1"
	defaultBroker             = ""

	configDir = ".triggermesh/cli"
)

func NewRootCommand(ver, commit string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tmctl",
		Short: "A command line interface to build event-driven applications",
		Long: `tmctl is a CLI to help you create event brokers, sources, targets and transformations.

Find more information at: https://docs.triggermesh.io`,
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
	}

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("version", defaultTriggermeshVersion, "TriggerMesh components version.")
	rootCmd.PersistentFlags().String("broker", defaultBroker, "Optional broker name.")
	// rootCmd.PersistentFlags().MarkHidden("broker")

	viper.BindPFlag("context", rootCmd.PersistentFlags().Lookup("broker"))
	viper.BindPFlag("triggermesh.version", rootCmd.PersistentFlags().Lookup("version"))

	rootCmd.AddCommand(brokers.NewCmd())
	rootCmd.AddCommand(create.NewCmd())
	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(delete.NewCmd())
	rootCmd.AddCommand(describe.NewCmd())
	rootCmd.AddCommand(dump.NewCmd())
	rootCmd.AddCommand(sendevent.NewCmd())
	rootCmd.AddCommand(start.NewCmd())
	rootCmd.AddCommand(stop.NewCmd())
	rootCmd.AddCommand(watch.NewCmd())
	rootCmd.AddCommand(version.NewCmd(ver, commit))

	rootCmd.RegisterFlagCompletionFunc("broker", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		list, err := brokers.List(path.Dir(viper.ConfigFileUsed()), "")
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		return list, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.RegisterFlagCompletionFunc("version", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	})

	return rootCmd
}

func initConfig() {
	home, err := os.UserHomeDir()
	cobra.CheckErr(err)
	configHome := path.Join(home, configDir)

	// Search config in home directory with name ".cobra" (without extension).
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(configHome)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			cobra.CheckErr(os.MkdirAll(configHome, os.ModePerm))
			viper.SetDefault("context", defaultBroker)
			viper.SetDefault("triggermesh.version", defaultTriggermeshVersion)
			cobra.CheckErr(viper.SafeWriteConfig())
			cobra.CheckErr(viper.ReadInConfig())
		} else {
			cobra.CheckErr(err)
		}
	}
}

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

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/cmd/config"
	"github.com/triggermesh/tmctl/cmd/create"
	"github.com/triggermesh/tmctl/cmd/delete"
	"github.com/triggermesh/tmctl/cmd/describe"
	"github.com/triggermesh/tmctl/cmd/dump"
	"github.com/triggermesh/tmctl/cmd/logs"
	"github.com/triggermesh/tmctl/cmd/sendevent"
	"github.com/triggermesh/tmctl/cmd/start"
	"github.com/triggermesh/tmctl/cmd/stop"
	"github.com/triggermesh/tmctl/cmd/version"
	"github.com/triggermesh/tmctl/cmd/watch"

	cliconfig "github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func NewRootCommand(ver, commit string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tmctl",
		Short: "A command line interface to build event-driven applications",
		Long: `tmctl is a CLI to help you create event brokers, sources, targets and transformations.

Find more information at: https://docs.triggermesh.io`,
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
	}

	conf, err := cliconfig.New()
	cobra.CheckErr(err)
	crds, err := crd.Fetch(conf.ConfigHome, conf.Triggermesh.ComponentsVersion)
	cobra.CheckErr(err)

	rootCmd.AddCommand(brokers.NewCmd(conf))
	rootCmd.AddCommand(create.NewCmd(conf, crds))
	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(delete.NewCmd(conf, crds))
	rootCmd.AddCommand(describe.NewCmd(conf, crds))
	rootCmd.AddCommand(dump.NewCmd(conf, crds))
	rootCmd.AddCommand(logs.NewCmd(conf, crds))
	rootCmd.AddCommand(sendevent.NewCmd(conf, crds))
	rootCmd.AddCommand(start.NewCmd(conf, crds))
	rootCmd.AddCommand(stop.NewCmd(conf))
	rootCmd.AddCommand(watch.NewCmd(conf))
	rootCmd.AddCommand(version.NewCmd(ver, commit, conf))

	rootCmd.PersistentFlags().StringVar(&conf.Triggermesh.ComponentsVersion, "version", conf.Triggermesh.ComponentsVersion, "TriggerMesh components version.")
	cobra.CheckErr(rootCmd.RegisterFlagCompletionFunc("version", cobra.NoFileCompletions))

	if os.Getenv("TMCTL_GENERATE_DOCS") == "true" {
		rootCmd.DisableAutoGenTag = true
		if err := doc.GenMarkdownTree(rootCmd, "./docs"); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	return rootCmd
}

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
	"path/filepath"

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
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
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

	c, err := cliconfig.New()
	cobra.CheckErr(err)
	crds, err := crd.Fetch(c.ConfigHome, c.Triggermesh.ComponentsVersion)
	cobra.CheckErr(err)

	manifest := manifest.New(filepath.Join(
		c.ConfigHome,
		c.Context,
		triggermesh.ManifestFile))
	_ = manifest.Read()

	rootCmd.AddCommand(brokers.NewCmd(c))
	rootCmd.AddCommand(create.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(delete.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(describe.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(dump.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(logs.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(sendevent.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(start.NewCmd(c, manifest, crds))
	rootCmd.AddCommand(stop.NewCmd(c, manifest))
	rootCmd.AddCommand(watch.NewCmd(c))
	rootCmd.AddCommand(version.NewCmd(ver, commit, c))

	rootCmd.PersistentFlags().StringVar(&c.Triggermesh.ComponentsVersion, "version", c.Triggermesh.ComponentsVersion, "TriggerMesh components version.")
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

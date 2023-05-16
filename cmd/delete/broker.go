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

package delete

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

func (o *CliOptions) deleteBrokerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "broker <name>",
		Short:   "Delete TriggerMesh Broker",
		Example: "tmctl delete broker foo",
		Args:    cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return completion.ListObjectsByKind("RedisBroker", o.Manifest), cobra.ShellCompDirectiveNoFileComp
		}, RunE: func(cmd *cobra.Command, args []string) error {
			return o.deleteBroker(args[0])
		},
	}
}

func (o *CliOptions) deleteBroker(broker string) error {
	oo := *o
	oo.Config.Context = broker
	oo.Manifest = manifest.New(filepath.Join(oo.Config.ConfigHome, broker, triggermesh.ManifestFile))
	cobra.CheckErr(oo.Manifest.Read())

	if err := oo.deleteBrokerComponents([]string{}, true); err != nil {
		return fmt.Errorf("deleting component: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(oo.Config.ConfigHome, broker)); err != nil {
		return fmt.Errorf("delete broker %q: %v", broker, err)
	}
	if broker == o.Config.Context {
		return o.switchContext()
	}
	return nil
}

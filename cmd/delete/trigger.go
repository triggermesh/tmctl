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
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/docker"
)

func (o *CliOptions) deleteTriggerCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "trigger <name>",
		Short:   "Delete TriggerMesh Trigger",
		Example: "tmctl delete trigger foo",
		Args:    cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return completion.ListObjectsByKind("Trigger", o.Manifest), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.deleteTrigger(args)
		},
	}
}

func (o *CliOptions) deleteTrigger(names []string) error {
	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	for _, object := range o.Manifest.Objects {
		if object.Kind != "Trigger" {
			continue
		}
		for _, name := range names {
			if name == object.Metadata.Name {
				o.deleteEverything(ctx, object, client)
				break
			}
		}
	}
	return nil
}

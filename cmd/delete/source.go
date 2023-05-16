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

func (o *CliOptions) deleteSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "source <name>",
		Short:   "Delete TriggerMesh Source",
		Example: "tmctl delete source foo",
		Args:    cobra.MinimumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return append(completion.ListObjectsByAPI("sources.triggermesh.io/v1alpha1", o.Manifest),
					completion.ListObjectsByAPI("serving.knative.dev/v1", o.Manifest)...),
				cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.deleteSources(args)
		},
	}
}

func (o *CliOptions) deleteSources(names []string) error {
	ctx := context.Background()
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	for _, object := range o.Manifest.Objects {
		if object.APIVersion != "sources.triggermesh.io/v1alpha1" &&
			object.APIVersion != "serving.knative.dev/v1" {
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

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

package create

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmcli/pkg/output"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
)

func (o *CreateOptions) NewTargetCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "target <kind> <args>",
		Short:              "TriggerMesh target",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			if len(args) == 0 {
				sources, err := crd.ListTargets(o.CRD)
				if err != nil {
					return fmt.Errorf("list sources: %w", err)
				}
				fmt.Printf("Available targets:\n---\n%s\n", strings.Join(sources, "\n"))
				return nil
			}
			kind, args, err := parse(args)
			if err != nil {
				return err
			}
			name, args := parameterFromArgs("name", args)
			eventSourceFilter, args := parameterFromArgs("source", args)
			eventTypesFilter, args := parameterFromArgs("eventTypes", args)
			var eventFilter []string
			if eventTypesFilter != "" {
				eventFilter = strings.Split(eventTypesFilter, ",")
			}
			return o.target(name, kind, args, eventSourceFilter, eventFilter)
		},
	}
}

func (o *CreateOptions) target(name, kind string, args []string, eventSourceFilter string, eventTypesFilter []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)
	manifest := path.Join(configDir, manifestFile)

	if eventSourceFilter != "" {
		et, err := o.producersEventTypes(eventSourceFilter)
		if err != nil {
			return fmt.Errorf("event types filter: %w", err)
		}
		eventTypesFilter = append(eventTypesFilter, et...)
	}

	t := target.New(name, o.CRD, kind, o.Context, o.Version, args)

	secretEnv, secretsChanged, err := triggermesh.ProcessSecrets(ctx, t, manifest)
	if err != nil {
		return fmt.Errorf("component secrets: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(ctx, t, manifest)
	if err != nil {
		return err
	}
	log.Println("Starting container")
	container, err := triggermesh.Start(ctx, t, (restart || secretsChanged), secretEnv)
	if err != nil {
		return err
	}

	if len(eventTypesFilter) != 0 {
		log.Println("Creating trigger")
		if err := o.createTrigger(fmt.Sprintf("%s-trigger", t.GetName()), eventTypesFilter, container.Name, container.HostPort()); err != nil {
			return err
		}
	}

	output.PrintStatus("consumer", t, eventSourceFilter, eventTypesFilter)
	return nil
}

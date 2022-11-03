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
	"strings"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func (o *CreateOptions) NewTargetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "target <kind> [--name <name>][--source <name>,<name>...][--eventTypes <type>,<type>...]",
		// Short:              "TriggerMesh target",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		ValidArgsFunction:  o.targetsCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "--help" {
				targets, err := crd.ListTargets(o.CRD)
				if err != nil {
					return fmt.Errorf("list sources: %w", err)
				}
				cmd.Help()
				fmt.Printf("\nAvailable target kinds:\n---\n%s\n", strings.Join(targets, "\n"))
				return nil
			}
			params := argsToMap(args[1:])
			var name string
			if n, exists := params["name"]; exists {
				name = n
				delete(params, "name")
			}
			if v, exists := params["version"]; exists {
				o.Version = v
				delete(params, "version")
			}
			var eventSourcesFilter, eventTypesFilter []string
			if sf, exists := params["source"]; exists {
				eventSourcesFilter = strings.Split(sf, ",")
				if len(eventSourcesFilter) == 1 {
					eventSourcesFilter = strings.Split(sf, " ")
				}
				delete(params, "source")
			}
			if tf, exists := params["eventTypes"]; exists {
				eventTypesFilter = strings.Split(tf, ",")
				if len(eventTypesFilter) == 1 {
					eventTypesFilter = strings.Split(tf, " ")
				}
				delete(params, "eventTypes")
			}
			return o.target(name, args[0], params, eventSourcesFilter, eventTypesFilter)
		},
	}
}

func (o *CreateOptions) target(name, kind string, args map[string]string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()

	eventSourcesFilter, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}

	t := target.New(name, o.CRD, kind, o.Context, o.Version, args)

	secretEnv, secretsChanged, err := triggermesh.ProcessSecrets(ctx, t.(triggermesh.Parent), o.Manifest)
	if err != nil {
		return fmt.Errorf("spec processing: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := t.Add(o.Manifest)
	if err != nil {
		return err
	}
	log.Println("Starting container")
	container, err := t.(triggermesh.Runnable).Start(ctx, secretEnv, (restart || secretsChanged))
	if err != nil {
		return err
	}

	for _, es := range eventSourcesFilter {
		if err := tmbroker.CreateTrigger("", container.Name, container.HostPort(), o.Context, o.ConfigBase, tmbroker.FilterExactAttribute("source", es)); err != nil {
			return err
		}
	}
	for _, et := range eventTypesFilter {
		if err := tmbroker.CreateTrigger("", container.Name, container.HostPort(), o.Context, o.ConfigBase, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return err
		}
	}

	output.PrintStatus("consumer", t, eventSourcesFilter, eventTypesFilter)
	return nil
}

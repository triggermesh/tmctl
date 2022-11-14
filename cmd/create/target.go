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

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
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
			cobra.CheckErr(o.Manifest.Read())
			params := argsToMap(args[0:])
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
			if image, exists := params["fromImage"]; exists {
				delete(params, "fromImage")
				return o.targetFromImage(name, image, params, eventSourcesFilter, eventTypesFilter)
			}
			return o.target(name, args[0], params, eventSourcesFilter, eventTypesFilter)
		},
	}
}

func (o *CreateOptions) target(name, kind string, args map[string]string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()

	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	t := target.New(name, o.CRD, kind, o.Context, o.Version, args)

	secrets, secretsChanged, err := components.ProcessSecrets(t.(triggermesh.Parent), o.Manifest)
	if err != nil {
		return fmt.Errorf("processing secrets: %v", err)
	}

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(t)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}

	log.Println("Starting container")
	if _, err := t.(triggermesh.Runnable).Start(ctx, secrets, (restart || secretsChanged)); err != nil {
		return err
	}

	for _, et := range eventTypesFilter {
		if _, err := o.createTrigger("", t, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return fmt.Errorf("creating trigger: %w", err)
		}
	}

	output.PrintStatus("consumer", t, eventSourcesFilter, eventTypesFilter)
	return nil
}

func (o *CreateOptions) createTrigger(name string, target triggermesh.Component, filter *eventingbroker.Filter) (triggermesh.Component, error) {
	trigger, err := tmbroker.NewTrigger(name, o.Context, o.ConfigBase, target, filter)
	if err != nil {
		return nil, err
	}
	if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
		return nil, err
	}
	if _, err := o.Manifest.Add(trigger); err != nil {
		return nil, err
	}
	return trigger, nil
}

func (o *CreateOptions) targetFromImage(name, image string, params map[string]string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()

	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	s := service.New(name, image, o.Context, service.Consumer, params)

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(s)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}
	log.Println("Starting container")
	if _, err := s.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}
	for _, et := range eventTypesFilter {
		if _, err := o.createTrigger("", s, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return fmt.Errorf("creating trigger: %w", err)
		}
	}
	output.PrintStatus("consumer", s, eventSourcesFilter, eventTypesFilter)

	return nil
}

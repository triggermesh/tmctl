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

	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

func (o *CreateOptions) NewTriggerCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "trigger --source <source> [--eventType <event type>] --target <target>",
		Short:              "TriggerMesh trigger",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			name, args := parameterFromArgs("name", args)
			eventSourcesFilter, args := parameterFromArgs("sources", args)
			eventTypesFilter, args := parameterFromArgs("eventTypes", args)
			target, _ := parameterFromArgs("target", args)
			if target == "" {
				return fmt.Errorf("\"--target <name>\" argument is required")
			}
			if eventSourcesFilter == "" && eventTypesFilter == "" {
				return fmt.Errorf("\"--sources <name>\" or \"--eventTypes <type>\" argument is required")
			}
			var typeFilter, sourceFilter []string
			if eventTypesFilter != "" {
				typeFilter = strings.Split(eventTypesFilter, ",")
			}
			if eventSourcesFilter != "" {
				sourceFilter = strings.Split(eventSourcesFilter, ",")
			}
			return o.trigger(name, sourceFilter, typeFilter, target)
		},
	}
}

func (o *CreateOptions) trigger(name string, eventSourcesFilter, eventTypesFilter []string, target string) error {
	configDir := path.Join(o.ConfigBase, o.Context)
	manifest := path.Join(configDir, manifestFile)

	for _, source := range eventSourcesFilter {
		et, err := o.producersEventTypes(source)
		if err != nil {
			return fmt.Errorf("event types filter: %w", err)
		}
		eventTypesFilter = append(eventTypesFilter, et...)
	}

	component, err := o.getObject(target, manifest)
	if err != nil {
		return fmt.Errorf("%q not found: %w", target, err)
	}

	consumer, ok := component.(triggermesh.Consumer)
	if !ok {
		return fmt.Errorf("%q is not an event target", target)
	}

	port, err := consumer.GetPort(context.Background())
	if err != nil {
		return fmt.Errorf("target port: %w", err)
	}

	log.Println("Creating trigger")
	if err := o.createTrigger(name, eventTypesFilter, component.GetName(), port); err != nil {
		return err
	}
	return nil
}

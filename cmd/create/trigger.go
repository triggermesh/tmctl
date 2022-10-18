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

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
)

func (o *CreateOptions) NewTriggerCmd() *cobra.Command {
	var name, target string
	var eventSourcesFilter, eventTypesFilter []string
	triggerCmd := &cobra.Command{
		Use:   "trigger --target <name> [--source <name>][--eventType <event type>]",
		Short: "TriggerMesh trigger",
		RunE: func(cmd *cobra.Command, args []string) error {
			crds, err := crd.Fetch(o.ConfigBase, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
			return o.trigger(name, eventSourcesFilter, eventTypesFilter, target)
		},
	}
	triggerCmd.Flags().StringVar(&name, "name", "", "Trigger name")
	triggerCmd.Flags().StringVar(&target, "target", "", "Target name")
	triggerCmd.Flags().StringSliceVar(&eventSourcesFilter, "source", []string{}, "Event sources filter")
	triggerCmd.Flags().StringSliceVar(&eventTypesFilter, "eventTypes", []string{}, "Event types filter")
	triggerCmd.MarkFlagRequired("target")
	return triggerCmd
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
	if len(eventTypesFilter) == 0 {
		return o.createTrigger(name, component.GetName(), port, nil)
	}
	for _, et := range eventTypesFilter {
		if err := o.createTrigger(name, component.GetName(), port, tmbroker.FilterExactType(et)); err != nil {
			return err
		}
	}
	return nil
}

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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

func (o *createOptions) newTriggerCmd() *cobra.Command {
	var name, target string
	var eventSourcesFilter, eventTypesFilter []string
	triggerCmd := &cobra.Command{
		Use:       "trigger --target <name> [--source <name>...][--eventTypes <type>...]",
		Short:     "Create TriggerMesh trigger. More information at https://docs.triggermesh.io/brokers/triggers/",
		Example:   "tmctl create trigger --target sockeye --source foo-httppollersource",
		ValidArgs: []string{"--target", "--name", "--source", "--eventTypes"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cobra.CheckErr(o.Manifest.Read())
			return o.trigger(name, eventSourcesFilter, eventTypesFilter, target)
		},
	}
	triggerCmd.Flags().StringVar(&name, "name", "", "Trigger name")
	triggerCmd.Flags().StringVar(&target, "target", "", "Target name")
	triggerCmd.Flags().StringSliceVar(&eventSourcesFilter, "source", []string{}, "Event sources filter")
	triggerCmd.Flags().StringSliceVar(&eventTypesFilter, "eventTypes", []string{}, "Event types filter")
	cobra.CheckErr(triggerCmd.MarkFlagRequired("target"))

	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListSources(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("eventTypes", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListEventTypes(o.Manifest, o.CRD, o.Version), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListTargets(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	return triggerCmd
}

func (o *createOptions) trigger(name string, eventSourcesFilter, eventTypesFilter []string, target string) error {
	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	component, err := components.GetObject(target, o.CRD, o.Version, o.Manifest)
	if err != nil {
		return fmt.Errorf("%q not found: %w", target, err)
	}

	if _, ok := component.(triggermesh.Consumer); !ok {
		return fmt.Errorf("%q is not an event target", target)
	}

	log.Println("Creating trigger")
	if len(eventTypesFilter) == 0 && len(eventSourcesFilter) == 0 {
		if _, err = o.createTrigger(name, component, nil); err != nil {
			return err
		}
	}
	for _, et := range eventTypesFilter {
		if _, err = o.createTrigger(name, component, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return err
		}
	}
	return nil
}

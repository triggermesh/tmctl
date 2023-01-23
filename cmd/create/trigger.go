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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

func (o *CliOptions) newTriggerCmd() *cobra.Command {
	var name, target, rawFilter string
	var eventSourcesFilter, eventTypesFilter []string
	triggerCmd := &cobra.Command{
		Use:       "trigger --target <name> [--source <name>...][--eventTypes <type>...]",
		Short:     "Create TriggerMesh trigger. More information at https://docs.triggermesh.io/brokers/triggers/",
		Example:   "tmctl create trigger --target sockeye --source foo-httppollersource",
		ValidArgs: []string{"--target", "--name", "--source", "--eventTypes"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.trigger(name, rawFilter, eventSourcesFilter, eventTypesFilter, target)
		},
	}
	triggerCmd.Flags().StringVar(&name, "name", "", "Trigger name")
	triggerCmd.Flags().StringVar(&target, "target", "", "Target name")
	triggerCmd.Flags().StringVar(&rawFilter, "filter", "", "Raw filter JSON")
	triggerCmd.Flags().StringSliceVar(&eventSourcesFilter, "source", []string{}, "Event sources filter")
	triggerCmd.Flags().StringSliceVar(&eventTypesFilter, "eventTypes", []string{}, "Event types filter")
	cobra.CheckErr(triggerCmd.MarkFlagRequired("target"))

	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListSources(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("eventTypes", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListEventTypes(o.Manifest, o.Config), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(triggerCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListTargets(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	return triggerCmd
}

func (o *CliOptions) trigger(name string, rawFilter string, eventSourcesFilter, eventTypesFilter []string, target string) error {
	var filters []*eventingbroker.Filter
	if rawFilter != "" {
		var filter eventingbroker.Filter
		if err := json.Unmarshal([]byte(rawFilter), &filter); err != nil {
			return fmt.Errorf("cannot decode filter JSON %q: %w", rawFilter, err)
		}
		filters = []*eventingbroker.Filter{&filter}
	} else {
		et, err := o.translateEventSource(eventSourcesFilter)
		if err != nil {
			return err
		}
		for _, eventTypes := range append(eventTypesFilter, et...) {
			filters = append(filters, tmbroker.FilterAttribute("type", eventTypes))
		}
	}

	component, err := components.GetObject(target, o.Config, o.Manifest)
	if err != nil {
		return fmt.Errorf("%q not found: %w", target, err)
	}
	if _, ok := component.(triggermesh.Consumer); !ok {
		return fmt.Errorf("%q is not an event target", target)
	}

	log.Println("Creating trigger")
	if len(filters) == 0 {
		if _, err = o.createTrigger(name, component, nil); err != nil {
			return err
		}
	}
	for _, filter := range filters {
		if _, err = o.createTrigger(name, component, filter); err != nil {
			return err
		}
	}
	return nil
}

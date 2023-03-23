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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/completion"
	transformationgui "github.com/triggermesh/tmctl/pkg/gui/transformation"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	helpColorCode    = "\033[90m"
	defaultColorCode = "\033[39m"
	helpText         = `Transformation example:

context:
- operation: add
  paths:
  - key: source
    value: triggermesh-local-source
data:
- operation: add
  paths:
  - key: foo
    value: bar
- operation: delete
  paths:
  - key: delete-me
- operation: shift
  paths:
  - key: old-path:new-path

For more samples please visit:
https://github.com/triggermesh/triggermesh/tree/main/config/samples/bumblebee`
)

func (o *CliOptions) newTransformationCmd() *cobra.Command {
	var name, target, file string
	var eventSourcesFilter, eventTypesFilter []string
	transformationCmd := &cobra.Command{
		Use:   "transformation [--target <name>][--source <name>...][--eventTypes <type>...][--from <path>]",
		Short: "Create TriggerMesh transformation. More information at https://docs.triggermesh.io/transformation/jsontransformation/",
		Example: `tmctl create transformation <<EOF
  data:
  - operation: add
    paths:
    - key: new-field
      value: hello from Transformation!
EOF`,
		ValidArgs: []string{"--name", "--target", "--source", "--eventTypes", "--from"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case o.Config.SchemaRegistry != "":
				name, sourceEventType, target, spec, err := transformationgui.Create(o.CRD, o.Manifest, o.Config)
				if err == gocui.ErrQuit {
					return nil
				}
				if err != nil {
					return fmt.Errorf("transformation wizard error: %w", err)
				}
				return o.transformation(name, target, spec, []string{}, []string{sourceEventType})
			case file != "":
				data, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("file %q read: %w", file, err)
				}
				return o.transformation(name, target, bytes.NewBuffer(data), eventSourcesFilter, eventTypesFilter)
			}
			return o.transformation(name, target, nil, eventSourcesFilter, eventTypesFilter)
		},
	}

	crd, err := crd.Fetch(o.Config.ConfigHome, o.Config.Triggermesh.ComponentsVersion)
	cobra.CheckErr(err)
	o.CRD = crd

	transformationCmd.Flags().StringVar(&name, "name", "", "Transformation name")
	transformationCmd.Flags().StringVarP(&file, "from", "f", "", "Transformation specification file")
	transformationCmd.Flags().StringVar(&target, "target", "", "Target name")
	transformationCmd.Flags().StringSliceVar(&eventSourcesFilter, "source", []string{}, "Sources component names")
	transformationCmd.Flags().StringSliceVar(&eventTypesFilter, "eventTypes", []string{}, "Event types filter")

	cobra.CheckErr(transformationCmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions))
	cobra.CheckErr(transformationCmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListSources(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(transformationCmd.RegisterFlagCompletionFunc("eventTypes", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListEventTypes(o.Manifest, o.Config, o.CRD), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(transformationCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListTargets(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	return transformationCmd
}

func (o *CliOptions) transformation(name, target string, specReader io.Reader, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()
	var targetComponent triggermesh.Component
	if target != "" {
		t, err := o.lookupTarget(ctx, target)
		if err != nil {
			return err
		}
		targetComponent = t
	}

	var expectedEventTypes []string
	if consumer, ok := targetComponent.(triggermesh.Consumer); ok {
		expectedEventTypes, _ = consumer.ConsumedEventTypes()
	}

	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	var data []byte
	if specReader == nil {
		input, err := fromStdIn()
		if err != nil {
			return fmt.Errorf("stdin read: %w", err)
		}
		data = []byte(input)
	} else {
		specFile, err := io.ReadAll(specReader)
		if err != nil {
			return fmt.Errorf("spec file read: %w", err)
		}
		data = specFile
	}
	if len(data) == 0 {
		return fmt.Errorf("empty spec")
	}
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("decode spec: %w", err)
	}

	crd, exists := o.CRD["transformation"]
	if !exists {
		return fmt.Errorf("CRD for kind \"transformation\" not found")
	}

	t := transformation.New(name, "transformation", o.Config.Context, o.Config.Triggermesh.ComponentsVersion, crd, spec)

	transformationEventType := fmt.Sprintf("%s.output", t.GetName())
	if len(expectedEventTypes) > 0 {
		transformationEventType = expectedEventTypes[0]
	}

	producedEventTypes, _ := t.(triggermesh.Producer).GetEventTypes()
	if len(producedEventTypes) == 0 {
		if err := t.(triggermesh.Producer).SetEventAttributes(map[string]string{
			"type": transformationEventType,
		}); err != nil {
			return fmt.Errorf("setting event type: %w", err)
		}
	} else {
		transformationEventType = producedEventTypes[0]
	}

	eventTypesMatch := false
	if len(expectedEventTypes) == 0 {
		eventTypesMatch = true
	}
	for _, eet := range expectedEventTypes {
		if eet == transformationEventType {
			eventTypesMatch = true
			break
		}
	}

	if targetComponent != nil && !eventTypesMatch {
		log.Printf(`WARNING! The transformation produces events of %q type, while target %q expectes %s. The target adapter may not work in this configuration.`,
			transformationEventType, targetComponent.GetName(), strings.Join(expectedEventTypes, ","))
	}

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(t)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}

	log.Println("Starting container")
	if _, err := t.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}

	// update our triggers in case of target container restart
	if restart {
		if err := o.updateTriggers(t); err != nil {
			return err
		}
	}

	var targetTriggers []triggermesh.Component
	// creating new trigger from transformation to target
	if targetComponent != nil {
		if targetTriggers, err = tmbroker.GetTargetTriggers(targetComponent.GetName(), o.Config.Context, o.Config.ConfigHome); err != nil {
			return fmt.Errorf("target triggers: %w", err)
		}
		if _, err := o.createTrigger("", targetComponent, tmbroker.FilterAttribute("type", transformationEventType)); err != nil {
			return fmt.Errorf("create trigger: %w", err)
		}
	}

	// updating existing triggers from sources to target
	for _, et := range eventTypesFilter {
		filter := tmbroker.FilterAttribute("type", et)
		if _, err := o.createTrigger("", t, filter); err != nil {
			return err
		}
		for _, component := range targetTriggers {
			trigger := component.(*tmbroker.Trigger)
			if trigger.Filters[0].Exact == nil ||
				trigger.Filters[0].Exact["type"] != et {
				continue
			}
			if err := trigger.RemoveFromLocalConfig(); err != nil {
				return err
			}
			if err := o.Manifest.Remove(trigger.GetName(), trigger.GetKind()); err != nil {
				return err
			}
		}
	}

	if len(eventTypesFilter) == 0 {
		for _, trigger := range targetTriggers {
			if len(trigger.(*tmbroker.Trigger).Filters) == 1 &&
				trigger.(*tmbroker.Trigger).Filters[0].Exact["type"] == transformationEventType {
				continue
			}
			trigger.(*tmbroker.Trigger).SetTarget(t)
			if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
				return err
			}
			if _, err := o.Manifest.Add(trigger); err != nil {
				return err
			}
		}
	}
	output.PrintStatus("consumer", t, eventSourcesFilter, eventTypesFilter)
	return nil
}

func fromStdIn() (string, error) {
	fmt.Printf("%s%s%s\n\n", helpColorCode, helpText, defaultColorCode)
	fmt.Printf("Insert Bumblebee transformation below\nPress Enter key twice to finish:\n")
	input, err := readInput()
	if err != nil {
		return "", fmt.Errorf("input read: %w", err)
	}
	input = strings.TrimRight(input, "\n")
	input = strings.TrimLeft(input, "\n")
	return input, nil
}

func readInput() (string, error) {
	var lines string
	scn := bufio.NewScanner(os.Stdin)
	for scn.Scan() {
		line := scn.Text()
		if len(line) == 0 {
			break
		}
		lines = fmt.Sprintf("%s\n%s", lines, line)
	}
	return lines, scn.Err()
}

func (o *CliOptions) lookupTarget(ctx context.Context, target string) (triggermesh.Component, error) {
	targetObject, err := components.GetObject(target, o.Config, o.Manifest, o.CRD)
	if err != nil {
		return nil, fmt.Errorf("transformation target: %w", err)
	}
	if _, ok := targetObject.(triggermesh.Consumer); !ok {
		return nil, fmt.Errorf("%q is not an event consumer", target)
	}
	return targetObject, nil
}

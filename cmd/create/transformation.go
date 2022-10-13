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
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmcli/pkg/output"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/transformation"
)

const (
	helpColorCode    = "\033[90m"
	defaultColorCode = "\033[39m"
	helpText         = `Transformation example:

context:
- operation: add
  paths:
  - key: source
    value: some-test-source
data:
- operation: store
  paths:
  - key: $foo
    value: Body
- operation: delete
  paths:
  - key:
- operation: add
  paths:
  - key: foo
    value: $foo

For more samples please visit:
https://github.com/triggermesh/triggermesh/tree/main/config/samples/bumblebee`
)

func (o *CreateOptions) NewTransformationCmd() *cobra.Command {
	var name, target, file string
	var eventSourcesFilter, eventTypesFilter []string
	transformationCmd := &cobra.Command{
		Use:   "transformation [--source <name>] [--target <name>] [--from <path>]",
		Short: "TriggerMesh transformation",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			return o.transformation(name, target, file, eventSourcesFilter, eventTypesFilter)
		},
	}
	transformationCmd.Flags().StringVar(&name, "name", "", "Transformation name")
	transformationCmd.Flags().StringVarP(&file, "from", "f", "", "Transformation specification file")
	transformationCmd.Flags().StringVar(&target, "target", "", "Target name")
	transformationCmd.Flags().StringSliceVar(&eventSourcesFilter, "source", []string{}, "Event sources filter")
	transformationCmd.Flags().StringSliceVar(&eventTypesFilter, "eventTypes", []string{}, "Event types filter")
	return transformationCmd
}

func (o *CreateOptions) transformation(name, target, file string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)
	manifest := path.Join(configDir, manifestFile)

	for _, source := range eventSourcesFilter {
		et, err := o.producersEventTypes(source)
		if err != nil {
			return fmt.Errorf("%q event types: %w", source, err)
		}
		eventTypesFilter = append(eventTypesFilter, et...)
	}
	var data []byte
	if file == "" {
		input, err := fromStdIn()
		if err != nil {
			return fmt.Errorf("spec read: %w", err)
		}
		data = []byte(input)
	} else {
		specFile, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("spec file read: %w", err)
		}
		data = specFile
	}
	var spec map[string]interface{}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("decode spec: %w", err)
	}
	t := transformation.New(name, o.CRD, "transformation", o.Context, o.Version, spec)
	if et, _ := t.GetEventTypes(); len(et) == 0 {
		if err := t.SetEventType(fmt.Sprintf("%s.output", t.Name)); err != nil {
			return fmt.Errorf("setting event type: %w", err)
		}
	}
	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(ctx, t, manifest)
	if err != nil {
		return err
	}
	log.Println("Starting container")
	container, err := triggermesh.Start(ctx, t, restart, nil)
	if err != nil {
		return err
	}

	var targetFilters []tmbroker.Filter
	if target != "" {
		transformationEventTypes, err := t.GetEventTypes()
		if err != nil {
			return fmt.Errorf("transformation event type: %w", err)
		}
		if targetFilters, err = o.updateTarget(ctx, target, transformationEventTypes); err != nil {
			return fmt.Errorf("updating target: %w", err)
		}
	}

	if len(eventTypesFilter) == 0 && len(targetFilters) != 0 {
		for _, filter := range targetFilters {
			if eventType, ok := filter.Exact["type"]; ok {
				eventTypesFilter = append(eventTypesFilter, eventType)
			}
		}
	}
	if len(eventTypesFilter) != 0 {
		log.Println("Creating trigger")
		if err := o.createTrigger("", container.Name, container.HostPort(), eventTypesFilter...); err != nil {
			return err
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

func (o *CreateOptions) updateTarget(ctx context.Context, target string, transformationEventTypes []string) ([]tmbroker.Filter, error) {
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)
	targetObject, err := o.getObject(target, manifest)
	if err != nil {
		return []tmbroker.Filter{}, fmt.Errorf("transformation target: %w", err)
	}
	consumer, ok := targetObject.(triggermesh.Consumer)
	if !ok {
		return []tmbroker.Filter{}, fmt.Errorf("%q is not an event consumer", target)
	}
	targetPort, err := consumer.GetPort(ctx)
	if err != nil {
		return []tmbroker.Filter{}, fmt.Errorf("target port: %w", err)
	}
	broker, err := tmbroker.New(o.Context, path.Join(o.ConfigBase, o.Context))
	if err != nil {
		return []tmbroker.Filter{}, fmt.Errorf("broker: %w", err)
	}
	triggers, err := broker.GetTriggers()
	if err != nil {
		return []tmbroker.Filter{}, fmt.Errorf("triggers: %w", err)
	}
	var triggerExisted bool
	var targetFilters []tmbroker.Filter
	for triggerName, triggerToTarget := range triggers {
		if target == triggerToTarget.GetTarget().Component {
			triggerExisted = true
			targetFilters = triggerToTarget.GetFilters()
			if err := o.createTrigger(triggerName, target, targetPort, transformationEventTypes...); err != nil {
				return []tmbroker.Filter{}, fmt.Errorf("update trigger: %w", err)
			}
		}
	}
	if !triggerExisted {
		if err := o.createTrigger("", target, targetPort, transformationEventTypes...); err != nil {
			return []tmbroker.Filter{}, fmt.Errorf("create trigger: %w", err)
		}
	}
	return targetFilters, nil
}

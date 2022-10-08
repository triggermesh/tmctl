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
	return &cobra.Command{
		Use:                "transformation <args>",
		Short:              "TriggerMesh transformation",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			name, args := parameterFromArgs("name", args)
			eventSourceFilter, args := parameterFromArgs("source", args)
			eventTypesFilter, _ := parameterFromArgs("eventTypes", args)
			var eventFilter []string
			if eventTypesFilter != "" {
				eventFilter = strings.Split(eventTypesFilter, ",")
			}
			return o.transformation(name, eventSourceFilter, eventFilter)
		},
	}
}

func (o *CreateOptions) transformation(name, eventSourceFilter string, eventTypesFilter []string) error {
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

	fmt.Printf("%s%s%s\n\n", helpColorCode, helpText, defaultColorCode)
	fmt.Printf("Insert Bumblebee transformation below\nPress Enter key twice to finish:\n")
	input, err := readInput()
	if err != nil {
		return fmt.Errorf("input read: %w", err)
	}
	input = strings.TrimRight(input, "\n")
	input = strings.TrimLeft(input, "\n")

	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(input), &spec); err != nil {
		return fmt.Errorf("spec unmarshal: %w", err)
	}

	t := transformation.New(name, o.CRD, "transformation", o.Context, o.Version, spec)
	if et, _ := t.GetEventTypes(); len(et) == 0 {
		if err := t.SetEventType(eventSourceFilter + "-transformed-event"); err != nil {
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

	if len(eventTypesFilter) != 0 {
		log.Println("Creating trigger")
		if err := o.createTrigger(fmt.Sprintf("%s-trigger", t.GetName()), eventTypesFilter, container.Name, container.HostPort()); err != nil {
			return err
		}
	}
	output.PrintStatus("consumer", t, eventSourceFilter, eventTypesFilter)
	return nil
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

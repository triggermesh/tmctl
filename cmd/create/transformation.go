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
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/transformation"
)

/*
context:
  - operation: store
    paths:
  - key: $time
    value: time
  - key: $id
    value: id
  - operation: add
    paths:
  - key: id
    value: $person-$id
  - key: type
    value: io.triggermesh.transformation.pingsource

data:
  - operation: store
    paths:
  - key: $person
    value: First Name
  - operation: add
    paths:
  - key: event.ID
    value: $id
  - key: event.time
    value: $time
  - operation: shift
    paths:
  - key: Date of birth:birthday
  - key: First Name:firstname
  - key: Last Name:lastname
  - operation: delete
    paths:
  - key: Mobile phone
  - key: Children[1].Year of birth
  - value: Martin
*/
func (o *CreateOptions) NewTransformationCmd() *cobra.Command {
	transformationCmd := &cobra.Command{
		Use:                "transformation <args>",
		Short:              "TriggerMesh transformation",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			sourceFilter, args := parameterFromArgs("source", args)
			eventTypesFilter, _ := parameterFromArgs("eventTypes", args)
			if sourceFilter == "" && eventTypesFilter == "" {
				return fmt.Errorf("\"--source=<kind>\" or \"--eventTypes=<a,b,c>\" is required")
			}
			var eventFilter []string
			if eventTypesFilter != "" {
				eventFilter = strings.Split(eventTypesFilter, ",")
			}
			return o.transformation(sourceFilter, eventFilter)
		},
	}

	return transformationCmd

}

func (o *CreateOptions) transformation(sourceKind string, eventTypes []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)

	if sourceKind != "" {
		s := source.New(o.CRD, sourceKind, o.Context, o.Version, nil)
		et, err := s.GetEventTypes()
		if err != nil {
			return fmt.Errorf("source event types: %w", err)
		}
		eventTypes = append(eventTypes, et...)
	}

	fmt.Printf("Insert Bumblebee transformation below\nPress Enter key twice to finish:\n")
	spec, err := readInput()
	if err != nil {
		return fmt.Errorf("input read: %w", err)
	}
	spec = strings.TrimRight(spec, "\n")
	spec = strings.TrimLeft(spec, "\n")

	var s map[string]interface{}
	if err := yaml.Unmarshal([]byte(spec), &s); err != nil {
		return fmt.Errorf("spec unmarshal: %w", err)
	}

	t := transformation.New(o.CRD, "transformation", o.Context, o.Version, s)

	restart, err := triggermesh.Create(ctx, t, path.Join(configDir, manifestFile))
	if err != nil {
		return err
	}

	container, err := triggermesh.Start(ctx, t, restart)
	if err != nil {
		return err
	}

	tr := tmbroker.NewTrigger(fmt.Sprintf("%s-trigger", t.GetName()), o.Context, configDir, eventTypes)
	tr.SetTarget(container.Name, fmt.Sprintf("http://host.docker.internal:%s", container.HostPort()))
	if err := tr.UpdateBrokerConfig(); err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	if err := tr.UpdateManifest(); err != nil {
		return fmt.Errorf("broker manifest: %w", err)
	}

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

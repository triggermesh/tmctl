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

	"github.com/triggermesh/tmctl/cmd/brokers"
	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
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
			kind, args, err := parse(args)
			if err != nil {
				return err
			}
			name, args := parameterFromArgs("name", args)
			version, args := parameterFromArgs("version", args)
			if version != "" {
				o.Version = version
			}
			eventSourcesFilter, args := parameterFromArgs("source", args)
			eventTypesFilter, args := parameterFromArgs("eventTypes", args)
			var typeFilter, sourceFilter []string
			if eventTypesFilter != "" {
				typeFilter = strings.Split(eventTypesFilter, ",")
			}
			if eventSourcesFilter != "" {
				sourceFilter = strings.Split(eventSourcesFilter, ",")
			}
			return o.target(name, kind, args, sourceFilter, typeFilter)
		},
	}
}

func (o *CreateOptions) target(name, kind string, args []string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()
	configBase := path.Join(o.ConfigBase, o.Context)
	manifestPath := path.Join(configBase, manifestFile)

	for _, source := range eventSourcesFilter {
		et, err := components.ProducersEventTypes(source, manifestPath, o.CRD, o.Version)
		if err != nil {
			return fmt.Errorf("%q event types: %w", source, err)
		}
		eventTypesFilter = append(eventTypesFilter, et...)
	}

	t := target.New(name, o.CRD, kind, o.Context, o.Version, args)

	secretEnv, secretsChanged, err := triggermesh.ProcessSecrets(ctx, t, manifestPath)
	if err != nil {
		return fmt.Errorf("spec processing: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(t, manifestPath)
	if err != nil {
		return err
	}
	log.Println("Starting container")
	container, err := triggermesh.Start(ctx, t, (restart || secretsChanged), secretEnv)
	if err != nil {
		return err
	}

	for _, et := range eventTypesFilter {
		if err := tmbroker.CreateTrigger("", container.Name, container.HostPort(), o.Context, configBase, tmbroker.FilterExactType(et)); err != nil {
			return err
		}
	}

	output.PrintStatus("consumer", t, eventSourcesFilter, eventTypesFilter)
	return nil
}

func (o *CreateOptions) targetsCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		sources, err := crd.ListTargets(o.CRD)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return sources, cobra.ShellCompDirectiveNoFileComp
	}

	if toComplete == "--source" ||
		toComplete == "--eventTypes" ||
		toComplete == "--name" {
		return []string{toComplete}, cobra.ShellCompDirectiveNoFileComp
	}
	manifestPath := path.Join(o.ConfigBase, o.Context, manifestFile)
	switch args[len(args)-1] {
	case "--source":
		return completion.ListSources(manifestPath), cobra.ShellCompDirectiveNoFileComp
	case "--eventTypes":
		return completion.ListEventTypes(manifestPath, o.CRD), cobra.ShellCompDirectiveNoFileComp
	case "--broker":
		list, err := brokers.List(o.ConfigBase, "")
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		return list, cobra.ShellCompDirectiveNoFileComp
	}
	if strings.HasPrefix(args[len(args)-1], "--") {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	prefix := ""
	toComplete = strings.TrimLeft(toComplete, "-")
	var properties map[string]crd.Property

	if !strings.Contains(toComplete, ".") {
		_, properties = completion.SpecFromCRD(args[0]+"target", o.CRD)
		if property, exists := properties[toComplete]; exists {
			if property.Typ == "object" {
				return []string{"--" + toComplete + "."}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
			return []string{"--" + toComplete}, cobra.ShellCompDirectiveNoFileComp
		}
	} else {
		path := strings.Split(toComplete, ".")
		exists, nestedProperties := completion.SpecFromCRD(args[0]+"target", o.CRD, path...)
		if len(nestedProperties) != 0 {
			prefix = toComplete
			if !strings.HasSuffix(prefix, ".") && prefix != "--" {
				prefix += "."
			}
			properties = nestedProperties
		} else if exists {
			return []string{"--" + toComplete}, cobra.ShellCompDirectiveNoFileComp
		} else {
			_, properties = completion.SpecFromCRD(args[0]+"target", o.CRD, path[:len(path)-1]...)
			prefix = strings.Join(path[:len(path)-1], ".") + "."
		}
	}

	var spec []string
	for name, property := range properties {
		attr := property.Typ
		if property.Required {
			attr = fmt.Sprintf("required,%s", attr)
		}
		name = prefix + name
		spec = append(spec, fmt.Sprintf("--%s\t(%s) %s", name, attr, property.Description))
	}
	return append(spec,
		"--source\tEvent source name.",
		"--eventTypes\tEvent types filter.",
		"--name\tOptional component name.",
	), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

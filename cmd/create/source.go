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
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func (o *CreateOptions) NewSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use: "source <kind> [--name <name>]",
		// Short:              "TriggerMesh source",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		ValidArgsFunction:  o.sourcesCompletion,
		// CompletionOptions:  cobra.CompletionOptions{DisableDescriptions: false},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "--help" {
				sources, err := crd.ListSources(o.CRD)
				if err != nil {
					return fmt.Errorf("list sources: %w", err)
				}
				cmd.Help()
				fmt.Printf("\nAvailable source kinds:\n---\n%s\n", strings.Join(sources, "\n"))
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
			return o.source(name, kind, args)
		},
	}
}

func (o *CreateOptions) source(name, kind string, args []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)
	manifest := path.Join(configDir, manifestFile)

	broker, err := tmbroker.New(o.Context, configDir)
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	port, err := broker.GetPort(ctx)
	if err != nil {
		return fmt.Errorf("broker port: %v", err)
	}

	spec := append(args, fmt.Sprintf("--sink.uri=http://host.docker.internal:%s", port))

	s := source.New(name, o.CRD, kind, o.Context, o.Version, spec)

	secretEnv, secretsChanged, err := triggermesh.ProcessSecrets(ctx, s, manifest)
	if err != nil {
		return fmt.Errorf("spec processing: %w", err)
	}

	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(s, manifest)
	if err != nil {
		return err
	}
	log.Println("Starting container")
	if _, err := triggermesh.Start(ctx, s, (restart || secretsChanged), secretEnv); err != nil {
		return err
	}
	output.PrintStatus("producer", s, []string{}, []string{})
	return nil
}

func (o *CreateOptions) sourcesCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		sources, err := crd.ListSources(o.CRD)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return sources, cobra.ShellCompDirectiveNoFileComp
	}
	if args[len(args)-1] == "--broker" {
		list, err := brokers.List(o.ConfigBase, "")
		if err != nil {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		}
		return list, cobra.ShellCompDirectiveNoFileComp
	} else if strings.HasPrefix(args[len(args)-1], "--") {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	prefix := ""
	toComplete = strings.TrimLeft(toComplete, "-")
	var properties map[string]crd.Property

	if !strings.Contains(toComplete, ".") {
		properties = completion.SpecFromCRD(args[0]+"source", o.CRD)
		if property, exists := properties[toComplete]; exists {
			if property.Typ == "object" {
				return []string{"--" + toComplete + "."}, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
		}
	} else {
		path := strings.Split(toComplete, ".")
		if nestedProperties := completion.SpecFromCRD(args[0]+"source", o.CRD, path...); len(nestedProperties) != 0 {
			prefix = toComplete
			if !strings.HasSuffix(prefix, ".") && prefix != "--" {
				prefix += "."
			}
			properties = nestedProperties
		} else {
			prefix = strings.Join(path[:len(path)-1], ".") + "."
			properties = completion.SpecFromCRD(args[0]+"source", o.CRD, path[:len(path)-1]...)
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
	return append(spec, "--name\tOptional component name."), cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

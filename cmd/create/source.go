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

	"github.com/triggermesh/tmcli/pkg/output"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
)

func (o *CreateOptions) NewSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "source <kind> [--name <name>]",
		Short:              "TriggerMesh source",
		DisableFlagParsing: true,
		SilenceErrors:      true,
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
			crds, err := crd.Fetch(o.ConfigBase, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
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

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
			params := argsToMap(args[1:])
			var name string
			if n, exists := params["name"]; exists {
				name = n
				delete(params, "name")
			}
			if v, exists := params["version"]; exists {
				o.Version = v
				delete(params, "version")
			}
			return o.source(name, args[0], params)
		},
	}
}

func (o *CreateOptions) source(name, kind string, params map[string]string) error {
	ctx := context.Background()

	broker, err := tmbroker.New(o.Context, path.Join(o.Manifest))
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	port, err := broker.(triggermesh.Consumer).GetPort(ctx)
	if err != nil {
		return fmt.Errorf("broker offline: %v", err)
	}
	params["sink.uri"] = "http://host.docker.internal:" + port

	s := source.New(name, o.CRD, kind, o.Context, o.Version, params)

	secretEnv, secretsChanged, err := triggermesh.ProcessSecrets(ctx, s.(triggermesh.Parent), o.Manifest)
	if err != nil {
		return fmt.Errorf("spec processing: %w", err)
	}
	log.Println("Updating manifest")
	restart, err := s.Add(o.Manifest)
	if err != nil {
		return err
	}
	status, err := s.(triggermesh.Reconcilable).Initialize(ctx, secretEnv)
	if err != nil {
		return fmt.Errorf("source initialization: %w", err)
	}
	s.(triggermesh.Reconcilable).UpdateStatus(status)
	log.Println("Starting container")
	if _, err := s.(triggermesh.Runnable).Start(ctx, secretEnv, (restart || secretsChanged)); err != nil {
		return err
	}
	output.PrintStatus("producer", s, []string{}, []string{})
	return nil
}

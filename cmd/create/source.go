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
		Use:                "source <kind> <args>",
		Short:              "TriggerMesh source",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			if len(args) == 0 {
				sources, err := crd.ListSources(o.CRD)
				if err != nil {
					return fmt.Errorf("list sources: %w", err)
				}
				fmt.Printf("Available sources:\n---\n%s\n", strings.Join(sources, "\n"))
				return nil
			}
			kind, args, err := parse(args)
			if err != nil {
				return err
			}
			name, args := parameterFromArgs("name", args)
			return o.source(name, kind, args)
		},
	}
}

func (o *CreateOptions) source(name, kind string, args []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)

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

	// extract secrets {
	// extract passed values from spec
	// wrap them into secret objects
	// set refs in parent object
	// return secret objects that need to be created
	// }
	// write secrets to manifest
	// set container env vars

	// secrets := triggermesh.ExtractSecrets(s)

	log.Println("Updating manifest")
	restart, err := triggermesh.WriteObject(ctx, s, path.Join(configDir, manifestFile))
	if err != nil {
		return err
	}
	log.Println("Starting container")
	if _, err := triggermesh.Start(ctx, s, restart); err != nil {
		return err
	}
	output.PrintStatus("producer", s, "", []string{})
	return nil
}

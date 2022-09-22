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

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
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
			return o.source(kind, args)
		},
	}
}

func (o *CreateOptions) source(kind string, args []string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigBase, o.Context)

	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	broker, err := tmbroker.New(o.Context, configDir)
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	brontainer, err := broker.AsContainer()
	if err != nil {
		return fmt.Errorf("broker container: %v", err)
	}
	brontainer, err = brontainer.LookupHostConfig(ctx, client)
	if err != nil {
		return fmt.Errorf("broker config: %v", err)
	}

	s := source.New(o.CRD, kind, o.Context, o.Version,
		append(args, fmt.Sprintf("--sink.uri=http://host.docker.internal:%s", brontainer.HostPort())))

	log.Println("Updating manifest")
	restart, err := triggermesh.Create(ctx, s, path.Join(configDir, manifestFile))
	if err != nil {
		return err
	}
	log.Println("Starting container")
	if _, err := triggermesh.Start(ctx, s, restart); err != nil {
		return err
	}

	eventTypesMessage := "This event source does not announce its event types"
	eventTypes, err := s.GetEventTypes()
	if err != nil {
		return err
	}
	if len(eventTypes) != 0 {
		eventTypesMessage = fmt.Sprintf("Event types produced by this source:\n\t%s", strings.Join(eventTypes, ", "))
	}
	fmt.Println("---")
	fmt.Println(eventTypesMessage)
	fmt.Println("\nNext steps:")
	fmt.Printf("\ttmcli create target <kind> --source %s [--eventTypes <types>]\t - create target that will consume events from this source\n", kind)
	fmt.Println("\ttmcli watch\t\t\t\t\t\t\t\t - show events flowing through the broker in the real time")
	fmt.Println("---")

	return nil
}

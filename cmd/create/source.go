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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func (o *CliOptions) newSourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "source [kind]/[--from-image <image>][--name <name>]",
		Short: "Create TriggerMesh source. More information at https://docs.triggermesh.io",
		Example: `tmctl create source httppoller \
	--endpoint https://www.example.com \
	--eventType sample-event \
	--interval 30s  \
	--method GET`,
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
				// help can never return an error
				_ = cmd.Help()
				fmt.Printf("\nAvailable source kinds:\n---\n%s\n", strings.Join(sources, "\n"))
				return nil
			}
			params := argsToMap(args)
			var name string
			if n, exists := params["name"]; exists {
				name = n
				delete(params, "name")
			}
			if v, exists := params["version"]; exists {
				o.Config.Triggermesh.ComponentsVersion = v
				delete(params, "version")
			}
			crd, err := crd.Fetch(o.Config.ConfigHome, o.Config.Triggermesh.ComponentsVersion)
			if err != nil {
				return err
			}
			// defer crd.Close()
			o.CRD = crd

			if _, readDisabled := params["disable-file-args"]; !readDisabled {
				for key, value := range params {
					data, err := os.ReadFile(value)
					if err != nil {
						continue
					}
					params[key] = string(data)
				}
			} else {
				delete(params, "disable-file-args")
			}
			if image, exists := params["from-image"]; exists {
				delete(params, "from-image")
				return o.sourceFromImage(name, image, params)
			}
			return o.source(name, args[0], params)
		},
	}
}

func (o *CliOptions) source(name, kind string, params map[string]string) error {
	ctx := context.Background()
	broker, err := tmbroker.New(o.Config.Context, o.Config.Triggermesh.Broker)
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	port, err := broker.(triggermesh.Consumer).GetPort(ctx)
	if err != nil {
		return fmt.Errorf("broker offline: %v", err)
	}
	params["sink.uri"] = "http://host.docker.internal:" + port

	crd, exists := o.CRD[kind+"source"]
	if !exists {
		return fmt.Errorf("CRD for kind %q not found", kind)
	}
	s := source.New(name, kind, o.Config.Context, o.Config.Triggermesh.ComponentsVersion, crd, params, nil)

	secrets, secretsEnv, err := components.ProcessSecrets(s.(triggermesh.Parent), o.Manifest)
	if err != nil {
		return fmt.Errorf("processing secrets: %v", err)
	}
	secretsChanged := false

	log.Println("Updating manifest")
	for _, secret := range secrets {
		dirty, err := o.Manifest.Add(secret)
		if err != nil {
			return fmt.Errorf("unable to write secret: %w", err)
		}
		if dirty {
			secretsChanged = true
		}
	}

	status, err := s.(triggermesh.Reconcilable).Initialize(ctx, secretsEnv)
	if err != nil {
		return fmt.Errorf("source initialization: %w", err)
	}
	s.(triggermesh.Reconcilable).UpdateStatus(status)

	restart, err := o.Manifest.Add(s)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}
	log.Println("Starting container")
	if _, err := s.(triggermesh.Runnable).Start(ctx, secretsEnv, (restart || secretsChanged)); err != nil {
		return err
	}
	output.PrintStatus("producer", s, []string{}, []string{})
	return nil
}

func (o *CliOptions) sourceFromImage(name, image string, params map[string]string) error {
	ctx := context.Background()
	broker, err := tmbroker.New(o.Config.Context, o.Config.Triggermesh.Broker)
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	port, err := broker.(triggermesh.Consumer).GetPort(ctx)
	if err != nil {
		return fmt.Errorf("broker offline: %v", err)
	}
	params["K_SINK"] = "http://host.docker.internal:" + port

	s := service.New(name, image, o.Config.Context, service.Producer, params)

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(s)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}
	log.Println("Starting container")
	if _, err := s.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}
	output.PrintStatus("producer", s, []string{}, []string{})
	return nil
}

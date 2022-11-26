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

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func (o *createOptions) NewTargetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "target [kind]/[--from-image <image>][--name <name>][--source <name>...][--event-types <type>...]",
		Short: "Create TriggerMesh target",
		Example: `tmctl create target http \
	--endpoint https://image-charts.com \
	--method GET \
	--response.eventType qr-data.response`,
		DisableFlagParsing: true,
		SilenceErrors:      true,
		ValidArgsFunction:  o.targetsCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "--help" {
				targets, err := crd.ListTargets(o.CRD)
				if err != nil {
					return fmt.Errorf("list sources: %w", err)
				}
				// help can never return an error
				_ = cmd.Help()
				fmt.Printf("\nAvailable target kinds:\n---\n%s\n", strings.Join(targets, "\n"))
				return nil
			}
			cobra.CheckErr(o.Manifest.Read())
			params := argsToMap(args[0:])
			var name string
			if n, exists := params["name"]; exists {
				name = n
				delete(params, "name")
			}
			if v, exists := params["version"]; exists {
				o.Version = v
				delete(params, "version")
			}
			var eventSourcesFilter, eventTypesFilter []string
			if sf, exists := params["source"]; exists {
				eventSourcesFilter = strings.Split(sf, ",")
				if len(eventSourcesFilter) == 1 {
					eventSourcesFilter = strings.Split(sf, " ")
				}
				delete(params, "source")
			}
			if tf, exists := params["event-types"]; exists {
				eventTypesFilter = strings.Split(tf, ",")
				if len(eventTypesFilter) == 1 {
					eventTypesFilter = strings.Split(tf, " ")
				}
				delete(params, "event-types")
			}
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
				return o.targetFromImage(name, image, params, eventSourcesFilter, eventTypesFilter)
			}
			return o.target(name, args[0], params, eventSourcesFilter, eventTypesFilter)
		},
	}
}

func (o *createOptions) target(name, kind string, args map[string]string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()

	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	t := target.New(name, o.CRD, kind, o.Context, o.Version, args)

	secrets, secretsEnv, err := components.ProcessSecrets(t.(triggermesh.Parent), o.Manifest)
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
	restart, err := o.Manifest.Add(t)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}

	log.Println("Starting container")
	if _, err := t.(triggermesh.Runnable).Start(ctx, secretsEnv, (restart || secretsChanged)); err != nil {
		return err
	}

	// update our triggers in case of target container restart
	if restart || secretsChanged {
		if err := o.updateTriggers(t); err != nil {
			return err
		}
	}

	for _, et := range eventTypesFilter {
		if _, err := o.createTrigger("", t, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return fmt.Errorf("creating trigger: %w", err)
		}
	}

	output.PrintStatus("consumer", t, eventSourcesFilter, eventTypesFilter)
	return nil
}

func (o *createOptions) createTrigger(name string, target triggermesh.Component, filter *eventingbroker.Filter) (triggermesh.Component, error) {
	trigger, err := tmbroker.NewTrigger(name, o.Context, o.ConfigBase, target, filter)
	if err != nil {
		return nil, err
	}
	if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
		return nil, err
	}
	if _, err := o.Manifest.Add(trigger); err != nil {
		return nil, err
	}
	return trigger, nil
}

func (o *createOptions) updateTriggers(target triggermesh.Component) error {
	triggers, err := tmbroker.GetTargetTriggers(target.GetName(), o.Context, o.ConfigBase)
	if err != nil {
		return fmt.Errorf("target triggers: %w", err)
	}
	for _, trigger := range triggers {
		trigger.(*tmbroker.Trigger).SetTarget(target)
		if err := trigger.(*tmbroker.Trigger).WriteLocalConfig(); err != nil {
			return fmt.Errorf("broker config update: %w", err)
		}
	}
	return nil
}

func (o *createOptions) targetFromImage(name, image string, params map[string]string, eventSourcesFilter, eventTypesFilter []string) error {
	ctx := context.Background()

	et, err := o.translateEventSource(eventSourcesFilter)
	if err != nil {
		return err
	}
	eventTypesFilter = append(eventTypesFilter, et...)

	s := service.New(name, image, o.Context, service.Consumer, params)

	log.Println("Updating manifest")
	restart, err := o.Manifest.Add(s)
	if err != nil {
		return fmt.Errorf("unable to update manifest: %w", err)
	}
	log.Println("Starting container")
	if _, err := s.(triggermesh.Runnable).Start(ctx, nil, restart); err != nil {
		return err
	}
	// update our triggers in case of target container restart
	if restart {
		if err := o.updateTriggers(s); err != nil {
			return err
		}
	}
	for _, et := range eventTypesFilter {
		if _, err := o.createTrigger("", s, tmbroker.FilterExactAttribute("type", et)); err != nil {
			return fmt.Errorf("creating trigger: %w", err)
		}
	}
	output.PrintStatus("consumer", s, eventSourcesFilter, eventTypesFilter)

	return nil
}

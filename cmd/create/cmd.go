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
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/monitoring"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type CliOptions struct {
	Config     *config.Config
	Manifest   *manifest.Manifest
	CRD        map[string]crd.CRD
	Monitoring *monitoring.Configuration
}

func NewCmd(config *config.Config, manifest *manifest.Manifest, crds map[string]crd.CRD, prom *monitoring.Configuration) *cobra.Command {
	o := &CliOptions{
		CRD:        crds,
		Config:     config,
		Manifest:   manifest,
		Monitoring: prom,
	}
	createCmd := &cobra.Command{
		Use:   "create <kind>",
		Short: "Create TriggerMesh component",
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
		Args: cobra.MinimumNArgs(1),
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cobra.CheckErr(docker.CheckDaemon())
			if cmd.Name() != "broker" {
				cobra.CheckErr(o.Manifest.Read())
			}
		},
	}
	createCmd.AddCommand(o.newBrokerCmd())
	createCmd.AddCommand(o.newSourceCmd())
	createCmd.AddCommand(o.newTargetCmd())
	createCmd.AddCommand(o.newTransformationCmd())
	createCmd.AddCommand(o.newTriggerCmd())
	return createCmd
}

func argsToMap(args []string) map[string]string {
	result := make(map[string]string)
	for k := 0; k < len(args); k++ {
		if isFlag(args[k]) {
			key := args[k]
			var value string
			if kv := strings.Split(args[k], "="); len(kv) == 2 {
				key = kv[0]
				value = kv[1]
			}
			key = strings.TrimLeft(key, "-")
			for j := k + 1; j < len(args) && !isFlag(args[j]); j++ {
				value = fmt.Sprintf("%s %s", value, args[j])
				k = j
			}
			result[key] = strings.TrimSpace(value)
			continue
		}
	}
	return result
}

func isFlag(s string) bool {
	return len(strings.TrimLeft(s, "-")) == len(s)-2
}

func (o *CliOptions) translateEventSource(eventSourcesFilter []string) ([]string, error) {
	var result []string
	for _, source := range eventSourcesFilter {
		s, err := components.GetObject(source, o.Config, o.Manifest, o.CRD)
		if err != nil {
			return nil, fmt.Errorf("%q event producer object: %w", source, err)
		}
		if _, ok := s.(triggermesh.Producer); !ok {
			return nil, fmt.Errorf("%q is not an event producer", source)
		}
		et, err := s.(triggermesh.Producer).GetEventTypes()
		if err != nil {
			return nil, fmt.Errorf("%q event source: %w", source, err)
		}
		result = append(result, et...)
	}
	return result, nil
}

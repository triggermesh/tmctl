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
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

type CreateOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
	Manifest   string
}

func NewCmd() *cobra.Command {
	o := &CreateOptions{}
	createCmd := &cobra.Command{
		Use:   "create <resource>",
		Short: "Create TriggerMesh objects",
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
		Args: cobra.MinimumNArgs(1),
	}

	cobra.OnInitialize(o.initialize)

	createCmd.AddCommand(o.NewBrokerCmd())
	createCmd.AddCommand(o.NewSourceCmd())
	createCmd.AddCommand(o.NewTargetCmd())
	createCmd.AddCommand(o.NewTransformationCmd())
	createCmd.AddCommand(o.NewTriggerCmd())

	return createCmd
}

func (o *CreateOptions) initialize() {
	o.ConfigBase = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = path.Join(o.ConfigBase, o.Context, manifestFile)
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds
}

func argsToMap(args []string) map[string]string {
	result := make(map[string]string)
	for k := 0; k < len(args); k++ {
		if strings.HasPrefix(args[k], "--") {
			key := strings.TrimLeft(args[k], "-")
			var value string
			if kv := strings.Split(args[k], "="); len(kv) == 2 {
				value = kv[1]
			}
			for j := k + 1; j < len(args) && !strings.HasPrefix(args[j], "--"); j++ {
				value = fmt.Sprintf("%s %s", value, args[j])
				k = j
			}
			result[key] = strings.TrimSpace(value)
			continue
		}
	}
	return result
}

func (o *CreateOptions) translateEventSource(eventSourcesFilter []string) ([]string, error) {
	for i, source := range eventSourcesFilter {
		s, err := components.GetObject(source, o.CRD, o.Version, o.Manifest)
		if err != nil {
			return nil, fmt.Errorf("%q event producer object: %w", source, err)
		}
		if _, ok := s.(triggermesh.Producer); !ok {
			return nil, fmt.Errorf("%q is not an event producer", source)
		}
		if eventSourcesFilter[i], err = s.(triggermesh.Producer).GetEventSource(); err != nil {
			return nil, fmt.Errorf("%q event source: %w", source, err)
		}
	}
	return eventSourcesFilter, nil
}

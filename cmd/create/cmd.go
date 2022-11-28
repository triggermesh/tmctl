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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type createOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &createOptions{}
	createCmd := &cobra.Command{
		Use:   "create <kind>",
		Short: "Create TriggerMesh component",
		// CompletionOptions: cobra.CompletionOptions{DisableDescriptions: true},
		Args: cobra.MinimumNArgs(1),
	}

	cobra.OnInitialize(o.initialize)

	createCmd.AddCommand(o.newBrokerCmd())
	createCmd.AddCommand(o.newSourceCmd())
	createCmd.AddCommand(o.newTargetCmd())
	createCmd.AddCommand(o.newTransformationCmd())
	createCmd.AddCommand(o.newTriggerCmd())

	return createCmd
}

func (o *createOptions) initialize() {
	o.ConfigBase = filepath.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = manifest.New(filepath.Join(o.ConfigBase, o.Context, triggermesh.ManifestFile))
	crds, err := crd.Fetch(o.ConfigBase, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds

	// try to read manifest even if it does not exists.
	// required for autocompletion.
	_ = o.Manifest.Read()
}

func argsToMap(args []string) map[string]string {
	result := make(map[string]string)
	for k := 0; k < len(args); k++ {
		if strings.HasPrefix(args[k], "--") {
			key := args[k]
			var value string
			if kv := strings.Split(args[k], "="); len(kv) == 2 {
				key = kv[0]
				value = kv[1]
			}
			key = strings.TrimLeft(key, "-")
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

func (o *createOptions) translateEventSource(eventSourcesFilter []string) ([]string, error) {
	var result []string
	for _, source := range eventSourcesFilter {
		s, err := components.GetObject(source, o.CRD, o.Version, o.Manifest)
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

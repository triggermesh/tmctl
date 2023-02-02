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

package import_

import (
	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/load"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func NewCmd(config *config.Config, crd map[string]crd.CRD) *cobra.Command {
	var from, name string
	importCmd := &cobra.Command{
		Use:     "import -f <path/to/manifest.yaml>/<manifest URL>",
		Short:   "Import TriggerMesh manifest",
		Example: "tmctl import -f manifest.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return load.Import(name, from, config, crd)
		},
	}
	importCmd.Flags().StringVar(&name, "name", "", "Set imported broker name")
	importCmd.Flags().StringVarP(&from, "from", "f", "", "Import manifest from")
	importCmd.MarkFlagRequired("f")
	return importCmd
}

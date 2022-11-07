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

package dump

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmctl/pkg/manifest"
)

const manifestFile = "manifest.yaml"

type DumpOptions struct {
	Format   string
	Manifest *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &DumpOptions{}
	dumpCmd := &cobra.Command{
		Use:   "dump [broker]",
		Short: "Generate Kubernetes manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			o.Manifest = manifest.New(path.Join(path.Dir(viper.ConfigFileUsed()), broker, manifestFile))
			cobra.CheckErr(o.Manifest.Read())
			return o.dump(broker)
		},
	}
	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")
	return dumpCmd
}

func (o *DumpOptions) dump(broker string) error {
	switch o.Format {
	case "json":
		for _, v := range o.Manifest.Objects {
			jsn, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsn))
		}
	case "yaml":
		for _, v := range o.Manifest.Objects {
			yml, err := yaml.Marshal(v)
			if err != nil {
				return err
			}
			fmt.Println("---")
			fmt.Println(string(yml))
		}
	default:
		return fmt.Errorf("format %q is not supported", o.Format)
	}

	return nil
}

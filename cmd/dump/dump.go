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
	"github.com/triggermesh/tmcli/pkg/manifest"
	"gopkg.in/yaml.v3"
)

const manifestFile = "manifest.yaml"

type DumpOptions struct {
	ConfigDir string
	Context   string
	Format    string
}

func NewCmd() *cobra.Command {
	o := &DumpOptions{}
	dumpCmd := &cobra.Command{
		Use:   "dump <broker>",
		Short: "Generate Kubernetes manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			broker, err := parseArgs(args)
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Context = broker
			return o.Dump()
		},
	}

	dumpCmd.Flags().StringVarP(&o.Format, "output", "o", "yaml", "Output format")

	return dumpCmd
}

func parseArgs(args []string) (string, error) {
	if l := len(args); l != 1 {
		return "", fmt.Errorf("expected 1 arguments, got %d", l)
	}
	return args[0], nil
}

func (o *DumpOptions) Dump() error {
	manifest := manifest.New(path.Join(o.ConfigDir, o.Context, manifestFile))
	if err := manifest.Read(); err != nil {
		return err
	}
	switch o.Format {
	case "json":
		for _, v := range manifest.Objects {
			jsn, err := json.MarshalIndent(v, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsn))
		}
	case "yaml":
		for _, v := range manifest.Objects {
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

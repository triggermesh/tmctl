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
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/triggermesh/tmcli/pkg/runtime"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

func (o *CreateOptions) NewSourceCmd() *cobra.Command {
	sourceCmd := &cobra.Command{
		Use:                "source <kind> <args>",
		Short:              "TriggerMesh source",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigBase = c
			o.Context = viper.GetString("context")
			o.Version = viper.GetString("triggermesh.version")
			o.CRD = viper.GetString("triggermesh.servedCRD")
			kind, args, err := parse(args)
			if err != nil {
				return err
			}
			return o.Source(kind, args)
		},
	}

	return sourceCmd
}

func (o *CreateOptions) Source(kind string, args []string) error {
	ctx := context.Background()
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)

	object, dirty, err := triggermesh.CreateSource(kind, o.Context, args, manifest, o.CRD)
	if err != nil {
		return fmt.Errorf("source creation: %w", err)
	}

	return runtime.Initialize(ctx, object, o.Version, dirty)
}

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
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/target"
	"github.com/triggermesh/tmcli/pkg/triggermesh/transformation"
)

const manifestFile = "manifest.yaml"

type CreateOptions struct {
	ConfigBase string
	Context    string
	Version    string
	CRD        string
}

func NewCmd() *cobra.Command {
	o := &CreateOptions{}
	createCmd := &cobra.Command{
		Use:   "create <resource>",
		Short: "create TriggerMesh objects",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	createCmd.AddCommand(o.NewBrokerCmd())
	createCmd.AddCommand(o.NewSourceCmd())
	createCmd.AddCommand(o.NewTargetCmd())
	createCmd.AddCommand(o.NewTransformationCmd())

	// createCmd.Flags().StringVarP(&o.Context, "broker", "b", "", "Connect components to this broker")

	return createCmd
}

func (o *CreateOptions) initializeOptions(cmd *cobra.Command) {
	configBase, err := cmd.Flags().GetString("config")
	if err != nil {
		panic(err)
	}
	o.ConfigBase = configBase
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.CRD = viper.GetString("triggermesh.servedCRD")
}

func parse(args []string) (string, []string, error) {
	if l := len(args); l < 1 {
		return "", []string{}, fmt.Errorf("expected at least 1 arguments, got %d", l)
	}
	return args[0], args[1:], nil
}

func (o *CreateOptions) getObject(name, manifestPath string) (triggermesh.Component, error) {
	manifest := manifest.New(manifestPath)
	if err := manifest.Read(); err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	for _, object := range manifest.Objects {
		if object.Metadata.Name == name {
			switch object.APIVersion {
			case "sources.triggermesh.io/v1alpha1":
				return source.New(object.Metadata.Name, o.CRD, object.Kind, "", o.Version, object.Spec), nil
			case "targets.triggermesh.io/v1alpha1":
				return target.New(object.Metadata.Name, o.CRD, object.Kind, "", o.Version, object.Spec), nil
			case "flow.triggermesh.io/v1alpha1":
				return transformation.New(object.Metadata.Name, o.CRD, object.Kind, "", o.Version, object.Spec), nil
			}
		}
	}
	return nil, nil
}

func parameterFromArgs(parameter string, args []string) (string, []string) {
	var value string
	for k := 0; k < len(args); k++ {
		if strings.HasPrefix(args[k], "--"+parameter) {
			if kv := strings.Split(args[k], "="); len(kv) == 2 {
				value = kv[1]
			} else if len(args) > k+1 && !strings.HasPrefix(args[k+1], "--") {
				value = args[k+1]
				k++
			}
			args = append(args[:k-1], args[k+1:]...)
			break
		}
	}
	return value, args
}

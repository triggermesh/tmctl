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

package describe

import (
	"context"
	"fmt"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/output"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/target"
	"github.com/triggermesh/tmcli/pkg/triggermesh/transformation"
)

const manifestFile = "manifest.yaml"

type integration struct {
	Broker         component
	Source         component
	Transformation component
	Target         component
}

type component struct {
	object    []triggermesh.Component
	container []*docker.Container
}

type DescribeOptions struct {
	ConfigDir string
	CRD       string
	Version   string
}

func NewCmd() *cobra.Command {
	o := &DescribeOptions{}
	return &cobra.Command{
		Use:   "describe <broker>",
		Short: "Show broker status",
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			configDir, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = configDir
			o.Version = viper.GetString("triggermesh.version")
			o.CRD = viper.GetString("triggermesh.servedCRD")
			return o.describe(broker)
		},
	}
}

func (o DescribeOptions) describe(broker string) error {
	ctx := context.Background()
	configDir := path.Join(o.ConfigDir, broker)
	manifestFile := path.Join(configDir, manifestFile)
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	var intg integration
	for _, object := range manifest.Objects {
		switch {
		case object.Kind == "Broker":
			co, err := tmbroker.New(object.Metadata.Name, configDir)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Broker = component{
				object:    []triggermesh.Component{co},
				container: []*docker.Container{cc},
			}
		case object.Kind == "Transformation":
			co := transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Transformation.object = append(intg.Transformation.object, co)
			intg.Transformation.container = append(intg.Transformation.container, cc)
		case object.APIVersion == "sources.triggermesh.io/v1alpha1":
			co := source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Source.object = append(intg.Source.object, co)
			intg.Source.container = append(intg.Source.container, cc)
		case object.APIVersion == "targets.triggermesh.io/v1alpha1":
			co := target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Target.object = append(intg.Target.object, co)
			intg.Target.container = append(intg.Target.container, cc)
		default:
			continue
		}
	}

	output.DescribeBroker(intg.Broker.object, intg.Broker.container)
	output.DescribeSource(intg.Source.object, intg.Source.container)
	output.DescribeTransformation(intg.Transformation.object, intg.Transformation.container)
	output.DescribeTarget(intg.Target.object, intg.Target.container)

	return nil
}

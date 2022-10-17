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
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmcli/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

type integration struct {
	Broker          components
	Sources         components
	Transformations components
	Targets         components

	Triggers []*tmbroker.Trigger
}

type components struct {
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
		Use:   "describe [broker]",
		Short: "Show broker status",
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			o.Version = viper.GetString("triggermesh.version")
			crds, err := crd.Fetch(o.ConfigDir, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
			if len(args) == 1 {
				return o.describe(args[0])
			}
			return o.describe(viper.GetString("context"))
		},
	}
}

func (o DescribeOptions) describe(broker string) error {
	ctx := context.Background()
	brokerConfigDir := path.Join(o.ConfigDir, broker)
	manifestFile := path.Join(brokerConfigDir, manifestFile)
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil
	}

	var intg integration
	for _, object := range manifest.Objects {
		switch {
		case object.Kind == "Broker":
			co, err := tmbroker.New(object.Metadata.Name, brokerConfigDir)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Broker = components{
				object:    []triggermesh.Component{co},
				container: []*docker.Container{cc},
			}
		case object.Kind == "Trigger":
			trigger := tmbroker.NewTrigger(object.Metadata.Name, broker, brokerConfigDir, tmbroker.Filter{})
			if err := trigger.LookupTrigger(); err != nil {
				return fmt.Errorf("trigger config: %w", err)
			}
			intg.Triggers = append(intg.Triggers, trigger)
		case object.Kind == "Transformation":
			co := transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Transformations.object = append(intg.Transformations.object, co)
			intg.Transformations.container = append(intg.Transformations.container, cc)
		case object.APIVersion == "sources.triggermesh.io/v1alpha1":
			co := source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Sources.object = append(intg.Sources.object, co)
			intg.Sources.container = append(intg.Sources.container, cc)
		case object.APIVersion == "targets.triggermesh.io/v1alpha1":
			co := target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Targets.object = append(intg.Targets.object, co)
			intg.Targets.container = append(intg.Targets.container, cc)
		default:
			continue
		}
	}

	output.DescribeBroker(intg.Broker.object, intg.Broker.container)
	output.DescribeTrigger(intg.Triggers)
	output.DescribeSource(intg.Sources.object, intg.Sources.container)
	output.DescribeTransformation(intg.Transformations.object, intg.Transformations.container)
	output.DescribeTarget(intg.Targets.object, intg.Targets.container)

	return nil
}

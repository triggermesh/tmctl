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

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/output"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

type integration struct {
	Broker          components
	Sources         components
	Transformations components
	Targets         components
	Triggers        components
}

type components struct {
	object    []triggermesh.Component
	container []*docker.Container
}

type DescribeOptions struct {
	ConfigBase string
	CRD        string
	Version    string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &DescribeOptions{}
	return &cobra.Command{
		Use:   "describe [broker]",
		Short: "Show broker status",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			o.ConfigBase = path.Dir(viper.ConfigFileUsed())
			o.Version = viper.GetString("triggermesh.version")
			o.Manifest = manifest.New(path.Join(o.ConfigBase, broker, manifestFile))
			cobra.CheckErr(o.Manifest.Read())
			crds, err := crd.Fetch(o.ConfigBase, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
			return o.describe(broker)
		},
	}
}

func (o DescribeOptions) describe(broker string) error {
	ctx := context.Background()
	var intg integration
	for _, object := range o.Manifest.Objects {
		switch {
		case object.Kind == "Broker":
			broker, err := tmbroker.New(object.Metadata.Name, o.Manifest.Path)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}
			container, err := broker.(triggermesh.Runnable).Info(ctx)
			if err != nil {
				container = nil
			}
			intg.Broker = components{
				object:    []triggermesh.Component{broker},
				container: []*docker.Container{container},
			}
		case object.Kind == "Trigger":
			trigger, err := tmbroker.NewTrigger(object.Metadata.Name, broker, o.ConfigBase, nil, nil)
			if err != nil {
				return fmt.Errorf("trigger object: %w", err)
			}
			if err := trigger.(*tmbroker.Trigger).LookupTrigger(); err != nil {
				return fmt.Errorf("trigger config: %w", err)
			}
			intg.Triggers.object = append(intg.Triggers.object, trigger)
		case object.Kind == "Transformation":
			trn := transformation.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			container, err := trn.(triggermesh.Runnable).Info(ctx)
			if err != nil {
				container = nil
			}
			intg.Transformations.object = append(intg.Transformations.object, trn)
			intg.Transformations.container = append(intg.Transformations.container, container)
		case object.APIVersion == "sources.triggermesh.io/v1alpha1":
			source := source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			container, err := source.(triggermesh.Runnable).Info(ctx)
			if err != nil {
				container = nil
			}
			intg.Sources.object = append(intg.Sources.object, source)
			intg.Sources.container = append(intg.Sources.container, container)
		case object.APIVersion == "targets.triggermesh.io/v1alpha1":
			target := target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			container, err := target.(triggermesh.Runnable).Info(ctx)
			if err != nil {
				container = nil
			}
			intg.Targets.object = append(intg.Targets.object, target)
			intg.Targets.container = append(intg.Targets.container, container)
		default:
			continue
		}
	}

	output.DescribeBroker(intg.Broker.object, intg.Broker.container)
	output.DescribeTrigger(intg.Triggers.object)
	output.DescribeSource(intg.Sources.object, intg.Sources.container)
	output.DescribeTransformation(intg.Transformations.object, intg.Transformations.container)
	output.DescribeTarget(intg.Targets.object, intg.Targets.container)

	return nil
}

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

package compose

import (
	"context"
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const manifestFile = "manifest.yaml"

var w *tabwriter.Writer

func init() {
	w = tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
}

// type ComposeOptions struct {
// 	Services []components `json:"services"`
// }

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

type ComposeOptions struct {
	ConfigDir string
	CRD       string
	Version   string
}

type DockerCompose struct {
	Version  string
	Services map[string]Service
}

type Service struct {
	Name        string                 `json:"name"`
	Command     string                 `json:"command"`
	Image       string                 `json:"image"`
	Ports       []string               `json:"ports"`
	Environment map[string]interface{} `json:"environment"`
	Volumes     []string               `json:"volumes"`
}

func NewCmd() *cobra.Command {
	o := &ComposeOptions{}
	return &cobra.Command{
		Use:   "compose [broker]",
		Short: "Show broker status",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			o.Version = viper.GetString("triggermesh.version")
			crds, err := crd.Fetch(o.ConfigDir, o.Version)
			if err != nil {
				return err
			}
			o.CRD = crds
			if len(args) == 1 {
				return o.compose(args[0])
			}
			return o.compose(viper.GetString("context"))
		},
	}
}

func (o ComposeOptions) compose(broker string) error {
	ctx := context.Background()
	brokerConfigDir := path.Join(o.ConfigDir, broker)
	manifestFile := path.Join(brokerConfigDir, manifestFile)
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil
	}
	var intg integration
	var svc []Service
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
			// create an array of containers here
			dc := &Service{
				Name:        object.Metadata.Name,
				Image:       cc.Image(),
				Ports:       []string{"8080:8080"},
				Environment: intg.Broker.object[0].GetSpec(),
			}
			svc = append(svc, *dc)

		// dont know how to handle trigger yet
		case object.Kind == "Trigger":
			trigger := tmbroker.NewTrigger(object.Metadata.Name, broker, brokerConfigDir, nil)
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
			dc := &Service{
				Name:        object.Metadata.Name,
				Image:       cc.Image(),
				Ports:       []string{"8080:8080"},
				Environment: intg.Transformations.object[0].GetSpec(),
			}
			svc = append(svc, *dc)
		case object.APIVersion == "sources.triggermesh.io/v1alpha1":
			co := source.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Sources.object = append(intg.Sources.object, co)
			intg.Sources.container = append(intg.Sources.container, cc)
			dc := &Service{
				Name:        object.Metadata.Name,
				Image:       cc.Image(),
				Ports:       []string{"8080:8080"},
				Environment: intg.Sources.object[0].GetSpec(),
			}
			svc = append(svc, *dc)
		case object.APIVersion == "targets.triggermesh.io/v1alpha1":
			co := target.New(object.Metadata.Name, o.CRD, object.Kind, broker, o.Version, object.Spec)
			cc, err := triggermesh.Info(ctx, co)
			if err != nil {
				// ignore the error
				cc = nil
			}
			intg.Targets.object = append(intg.Targets.object, co)
			intg.Targets.container = append(intg.Targets.container, cc)

			// de structure the list of environment variables
			gs := intg.Targets.object[0].GetSpec()
			var env []string
			for k, v := range gs {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}

			dc := &Service{
				Name:        object.Metadata.Name,
				Image:       cc.Image(),
				Ports:       []string{"8080:8080"},
				Environment: gs,
			}
			svc = append(svc, *dc)
		default:
			continue
		}
	}

	fmt.Println("services:")
	// print the arrays of containers
	for _, s := range svc {
		fmt.Printf("   name: %s\n", s.Name)
		fmt.Printf("     image: %s\n", s.Image)
		fmt.Printf("     ports: \n")
		for _, p := range s.Ports {
			fmt.Printf("       - %s\n", p)
		}
		fmt.Printf("     environment: \n")
		for k, v := range s.Environment {
			// look for "auth" in the key
			fmt.Printf("       - %s: %s\n", k, v)
		}
	}
	return nil
}

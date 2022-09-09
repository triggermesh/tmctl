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
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/runtime"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
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
		Use:                "create <resource>",
		Short:              "create TriggerMesh objects",
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
			resource, kind, args, err := o.Parse(args)
			if err != nil {
				return err
			}
			if err := o.Create(resource, kind, args); err != nil {
				return err
			}
			fmt.Println("Object created")
			return nil
		},
	}
	// createCmd.Flags().StringVarP(&o.Context, "broker", "b", "", "Connect components to this broker")

	return createCmd
}

func (c *CreateOptions) Parse(args []string) (string, string, []string, error) {
	if l := len(args); l < 2 {
		return "", "", []string{}, fmt.Errorf("expected at least 2 arguments, got %d", l)
	}
	return args[0], args[1], args[2:], nil
}

func (o *CreateOptions) Create(resource, kind string, args []string) error {
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)
	var object *kubernetes.Object
	var dirty bool
	var err error

	switch resource {
	case "broker":
		name := kind
		manifest = path.Join(o.ConfigBase, name, manifestFile)
		object, dirty, err = triggermesh.CreateBroker(name, manifest)
		if err != nil {
			return fmt.Errorf("broker creation error: %w", err)
		}
		viper.Set("context", object.Metadata.Name)
		if err := viper.WriteConfig(); err != nil {
			return err
		}
	case "source":
		object, dirty, err = triggermesh.CreateSource(kind, o.Context, args, manifest, o.CRD)
		if err != nil {
			return fmt.Errorf("source creation error: %w", err)
		}
	case "target":
		object, dirty, err = triggermesh.CreateTarget(kind, o.Context, args, manifest, o.CRD)
		if err != nil {
			return fmt.Errorf("target creation error: %w", err)
		}
	default:
		return fmt.Errorf("unsupported resource type %q", resource)
	}

	status, err := runtime.GetStatus(object)
	if err != nil {
		return fmt.Errorf("cannot read container status: %w", err)
	}

	switch {
	case status == "not found":
		// create
		if _, err := runtime.RunObject(object, []string{}, o.Version); err != nil {
			return fmt.Errorf("cannot start container: %w", err)
		}
	case dirty || status == "exited" || status == "dead":
		// recreate
		if err := runtime.StopObject(object); err != nil {
			return fmt.Errorf("cannot stop container: %w", err)
		}
		if _, err := runtime.RunObject(object, []string{}, o.Version); err != nil {
			return fmt.Errorf("cannot start container: %w", err)
		}
	default:
		fmt.Printf("Doing nothing because status is %q and dirty flag is %t\n", status, dirty)
	}

	return nil
}

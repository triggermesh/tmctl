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

package start

import (
	"context"
	"fmt"
	"log"
	"path"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/source"
	"github.com/triggermesh/tmcli/pkg/triggermesh/target"
)

const manifestFile = "manifest.yaml"

type StartOptions struct {
	ConfigDir string
	Version   string
	Restart   bool
	CRD       string
}

func NewCmd() *cobra.Command {
	o := &StartOptions{}
	createCmd := &cobra.Command{
		Use:   "start <broker>",
		Short: "starts TriggerMesh components",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}
			o.ConfigDir = c
			o.Version = viper.GetString("triggermesh.version")
			o.CRD = viper.GetString("triggermesh.servedCRD")

			if len(args) != 1 {
				return fmt.Errorf("expected only 1 argument")
			}

			return o.start(args[0])
		},
	}
	createCmd.Flags().BoolVar(&o.Restart, "restart", false, "Restart components")

	return createCmd
}

func (o *StartOptions) start(broker string) error {
	ctx := context.Background()
	manifest := manifest.New(path.Join(o.ConfigDir, broker, manifestFile))
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(manifest.Objects))

	for i, object := range manifest.Objects {
		go func(i int, object kubernetes.Object) {
			var c triggermesh.Component
			var err error
			switch object.Kind {
			case "Source":
				c, err = source.NewSource(manifestFile, o.CRD, object.Kind, broker, o.Version, []string{})
				if err != nil {
					log.Printf("Creating source: %v", err)
				}
			case "Target":
				c = target.NewTarget(manifestFile, o.CRD, object.Kind, broker, o.Version, []string{})
			case "Broker":
				c, err = tmbroker.NewBroker(manifestFile, object.Metadata.Name, o.Version)
				if err != nil {
					log.Printf("Creating broker: %v", err)
				}
			}
			container, err := triggermesh.Run(ctx, c)
			if err != nil {
				log.Printf("Starting container: %v", err)
			}
			fmt.Println(container)

			wg.Done()
		}(i, object)
	}
	wg.Wait()
	return nil
}

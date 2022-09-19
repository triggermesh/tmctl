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
	"strings"
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
	manifestFile := path.Join(o.ConfigDir, broker, "manifest.yaml")
	manifestOrig := manifest.New(manifestFile)
	if err := manifestOrig.Read(); err != nil {
		return fmt.Errorf("cannot parse manifest: %w", err)
	}

	var socket string
	manifestWoBroker := manifestOrig

	// start broker first
	for i, object := range manifestOrig.Objects {
		if object.Kind == "Broker" {
			manifestWoBroker.Objects = append(manifestOrig.Objects[:i], manifestOrig.Objects[i+1:]...)
			broker, err := tmbroker.NewBroker(manifestFile, object.Metadata.Name)
			if err != nil {
				return fmt.Errorf("creating broker object: %v", err)
			}

			container, err := triggermesh.Start(ctx, broker, true)
			if err != nil {
				return fmt.Errorf("starting broker container: %v", err)
			}
			socket = container.Socket()
		}
	}

	if socket == "" {
		return fmt.Errorf("broker is not available")
	}

	var wg sync.WaitGroup
	wg.Add(len(manifestWoBroker.Objects))

	for i, object := range manifestWoBroker.Objects {
		go func(i int, object kubernetes.Object) {
			var c triggermesh.Component
			switch {
			case strings.HasSuffix(object.Kind, "Source"):
				manifestOrig.Objects[i].Spec["sink"] = map[string]interface{}{"uri": "http://" + socket}
				c = source.NewSource(manifestFile, o.CRD, object.Kind, broker, o.Version, manifestOrig.Objects[i].Spec)

				// update sink in local manifest. Not required
				// manifestOrig.Write()
			case strings.HasSuffix(object.Kind, "Target"):
				c = target.NewTarget(manifestFile, o.CRD, object.Kind, broker, o.Version, object.Spec)
			case object.Kind == "Trigger":
				wg.Done()
				return
			}
			_, err := triggermesh.Start(ctx, c, false)
			if err != nil {
				log.Printf("Starting container: %v", err)
			}
			wg.Done()
		}(i, object)
	}
	wg.Wait()
	return nil
}

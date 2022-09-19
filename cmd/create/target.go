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
	"strings"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmcli/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/target"
)

func (o *CreateOptions) NewTargetCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "target <kind> <args>",
		Short:              "TriggerMesh target",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.initializeOptions(cmd)
			kind, args, err := parse(args)
			if err != nil {
				return err
			}
			return o.Target(kind, args)
		},
	}
}

func triggerFromArgs(args []string) (string, []string) {
	var target string
	for k := 0; k < len(args); k++ {
		if strings.HasPrefix(args[k], "--trigger") {
			if kv := strings.Split(args[k], "="); len(kv) == 2 {
				target = kv[1]
			} else if len(args) > k+1 && !strings.HasPrefix(args[k+1], "--") {
				target = args[k+1]
				k++
			}
			args = append(args[:k-1], args[k+1:]...)
			break
		}
		k++
	}
	return target, args
}

func (o *CreateOptions) Target(kind string, args []string) error {
	ctx := context.Background()
	manifest := path.Join(o.ConfigBase, o.Context, manifestFile)

	trigger, aargs := triggerFromArgs(args)
	tr := tmbroker.NewTrigger(trigger, manifest, o.Context, "")
	ts, err := tr.LookupTrigger()
	if err != nil {
		return fmt.Errorf("trigger lookup: %w", err)
	}

	t := target.NewTarget(manifest, o.CRD, kind, o.Context, o.Version, aargs)

	restart, err := triggermesh.Create(ctx, t, manifest)
	if err != nil {
		return err
	}

	container, err := triggermesh.Start(ctx, t, restart)
	if err != nil {
		return err
	}

	tr.SetFilter(ts.Filters[0].Exact.Type)
	tr.SetTarget(fmt.Sprintf("http://host.docker.internal:%s", strings.Split(container.Socket(), ":")[1]))
	if err := tr.UpdateBrokerConfig(); err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	if err := tr.UpdateManifest(); err != nil {
		return fmt.Errorf("broker manifest: %w", err)
	}

	return nil
}

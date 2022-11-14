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
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	manifestFile = "manifest.yaml"

	successColorCode = "\033[92m"
	defaultColorCode = "\033[39m"
	offlineColorCode = "\u001b[31m"
)

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

func (o DescribeOptions) describe(b string) error {
	broker := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	triggers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	producers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	consumers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)

	fmt.Fprintln(broker, "Broker\tStatus")
	fmt.Fprintln(triggers, "Trigger\tTarget\tFilter")
	fmt.Fprintln(producers, "Producer\tKind\tEventTypes\tStatus")
	fmt.Fprintln(consumers, "Consumer\tKind\tExpected Events\tStatus")

	for _, object := range o.Manifest.Objects {
		c, err := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest)
		if err != nil {
			return fmt.Errorf("creating component interface: %w", err)
		}
		if c == nil {
			continue
		}
		switch c.GetAPIVersion() {
		case tmbroker.APIVersion:
			switch c.GetKind() {
			case tmbroker.BrokerKind:
				fmt.Fprintf(broker, "%s\t%s\n", c.GetName(), status(c))
			case tmbroker.TriggerKind:
				filterString := "*"
				if len(c.(*tmbroker.Trigger).Filters) != 0 {
					filterString = triggerFilterToString(c.(*tmbroker.Trigger).Filters[0])
				}
				fmt.Fprintf(triggers, "%s\t%s\t%s\n", c.GetName(), c.(*tmbroker.Trigger).Target.Ref.Name, filterString)
			}
		case "sources.triggermesh.io/v1alpha1", "flow.triggermesh.io/v1alpha1":
			et, _ := c.(triggermesh.Producer).GetEventTypes()
			if len(et) == 0 {
				et = []string{"*"}
			}
			fmt.Fprintf(producers, "%s\t%s\t%s\t%s\n", c.GetName(), c.GetKind(), strings.Join(et, ", "), status(c))
		case "targets.triggermesh.io/v1alpha1":
			et, _ := c.(triggermesh.Consumer).ConsumedEventTypes()
			if len(et) == 0 {
				et = []string{"*"}
			}
			fmt.Fprintf(consumers, "%s\t%s\t%s\t%s\n", c.GetName(), c.GetKind(), strings.Join(et, ", "), status(c))
		case service.APIVersion:
			if c.(*service.Service).IsSource() {
				et, _ := c.(triggermesh.Producer).GetEventTypes()
				if len(et) == 0 {
					et = []string{"*"}
				}
				fmt.Fprintf(producers, "%s\t%s\t%s\t%s\n", c.GetName(), "kn-service", strings.Join(et, ", "), status(c))
			}
			if c.(*service.Service).IsTarget() {
				et, _ := c.(triggermesh.Consumer).ConsumedEventTypes()
				if len(et) == 0 {
					et = []string{"*"}
				}
				fmt.Fprintf(consumers, "%s\t%s\t%s\t%s\n", c.GetName(), "kn-service", strings.Join(et, ", "), status(c))
			}
		case "v1":
		}
	}
	fmt.Fprintln(broker)
	fmt.Fprintln(triggers)
	fmt.Fprintln(producers)
	fmt.Fprintln(consumers)
	return nil
}

func status(component triggermesh.Component) string {
	offlineStatus := fmt.Sprintf("%soffline%s", offlineColorCode, defaultColorCode)
	if container, ok := component.(triggermesh.Runnable); ok {
		c, err := container.Info(context.Background())
		if err != nil {
			return offlineStatus
		}
		return fmt.Sprintf("%sonline(http://localhost:%s)%s", successColorCode, c.HostPort(), defaultColorCode)
	}
	return offlineStatus
}

func triggerFilterToString(filter eventingbroker.Filter) string {
	var result []string
	for k, v := range filter.Exact {
		result = append(result, fmt.Sprintf("%s is %s", k, v))
	}
	return strings.Join(result, ", ")
}

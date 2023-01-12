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
	"path/filepath"
	"strings"
	"text/tabwriter"

	kyaml "sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	successColorCode = "\033[92m"
	defaultColorCode = "\033[39m"
	offlineColorCode = "\033[31m"
)

type describeOptions struct {
	ConfigBase string
	CRD        string
	Version    string
	Manifest   *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &describeOptions{}
	return &cobra.Command{
		Use:     "describe [broker]",
		Short:   "List broker components and their statuses",
		Example: "tmctl describe",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			broker := viper.GetString("context")
			if len(args) == 1 {
				broker = args[0]
			}
			o.ConfigBase = filepath.Dir(viper.ConfigFileUsed())
			o.Version = viper.GetString("triggermesh.version")
			o.Manifest = manifest.New(filepath.Join(o.ConfigBase, broker, triggermesh.ManifestFile))
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

func (o *describeOptions) describe(b string) error {
	broker := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	triggers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	producers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	consumers := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	transformations := tabwriter.NewWriter(os.Stdout, 10, 5, 5, ' ', 0)
	fmt.Fprintln(broker, "Broker\tStatus")
	fmt.Fprintln(triggers, "Trigger\tTarget\tFilter")
	fmt.Fprintln(transformations, "Transformation\tEventTypes\tStatus")
	fmt.Fprintln(producers, "Source\tKind\tEventTypes\tStatus")
	fmt.Fprintln(consumers, "Target\tKind\tExpected Events\tStatus")
	brokersPrint := false
	triggersPrint := false
	transformationsPrint := false
	producersPrint := false
	consumersPrint := false

	for _, object := range o.Manifest.Objects {
		c, err := components.GetObject(object.Metadata.Name, o.CRD, o.Version, o.Manifest)
		if err != nil {
			return fmt.Errorf("creating component interface: %w", err)
		}
		if c == nil {
			continue
		}
		if c.GetAPIVersion() == tmbroker.APIVersion {
			switch c.GetKind() {
			case tmbroker.BrokerKind:
				brokersPrint = true
				fmt.Fprintf(broker, "%s\t%s\n", c.GetName(), status(c))
			case tmbroker.TriggerKind:
				filterString := "*"
				if len(c.(*tmbroker.Trigger).Filters) != 0 {
					filterString = triggerFilterToString(c.(*tmbroker.Trigger).Filters)
				}
				triggersPrint = true
				fmt.Fprintf(triggers, "%s\t%s\t%s\n", c.GetName(), c.(*tmbroker.Trigger).Target.Ref.Name, filterString)
			}
			continue
		}

		producer, pOk := c.(triggermesh.Producer)
		consumer, cOk := c.(triggermesh.Consumer)
		switch {
		case pOk && cOk:
			// service
			if service, ok := c.(*service.Service); ok {
				if service.IsSource() {
					et, _ := c.(triggermesh.Producer).GetEventTypes()
					if len(et) == 0 {
						et = []string{"*"}
					}
					producersPrint = true
					fmt.Fprintf(producers, "%s\tservice (%s)\t%s\t%s\n", c.GetName(), service.Image, strings.Join(et, ", "), status(c))
				}
				if service.IsTarget() {
					et, _ := c.(triggermesh.Consumer).ConsumedEventTypes()
					if len(et) == 0 {
						et = []string{"*"}
					}
					consumersPrint = true
					fmt.Fprintf(consumers, "%s\tservice (%s)\t%s\t%s\n", c.GetName(), service.Image, strings.Join(et, ", "), status(c))
				}
			}
			// transformation
			if _, ok := c.(*transformation.Transformation); ok {
				et, _ := producer.GetEventTypes()
				if len(et) == 0 {
					et = []string{"*"}
				}
				transformationsPrint = true
				fmt.Fprintf(transformations, "%s\t%s\t%s\n", c.GetName(), strings.Join(et, ", "), status(c))
			}
		case pOk:
			// source
			et, _ := producer.GetEventTypes()
			if len(et) == 0 {
				et = []string{"*"}
			}
			producersPrint = true
			fmt.Fprintf(producers, "%s\t%s\t%s\t%s\n", c.GetName(), c.GetKind(), strings.Join(et, ", "), status(c))
		case cOk:
			// target
			et, _ := consumer.ConsumedEventTypes()
			if len(et) == 0 {
				et = []string{"*"}
			}
			consumersPrint = true
			fmt.Fprintf(consumers, "%s\t%s\t%s\t%s\n", c.GetName(), c.GetKind(), strings.Join(et, ", "), status(c))
		}
	}
	if brokersPrint {
		fmt.Fprintln(broker)
	}
	if triggersPrint {
		fmt.Fprintln(triggers)
	}
	if transformationsPrint {
		fmt.Fprintln(transformations)
	}
	if producersPrint {
		fmt.Fprintln(producers)
	}
	if consumersPrint {
		fmt.Fprintln(consumers)
	}
	return nil
}

func status(component triggermesh.Component) string {
	offlineStatus := fmt.Sprintf("%soffline%s", offlineColorCode, defaultColorCode)
	if container, ok := component.(triggermesh.Runnable); ok {
		c, err := container.Info(context.Background())
		if err != nil || !c.Online {
			return offlineStatus
		}
		return fmt.Sprintf("%sonline(http://localhost:%s)%s", successColorCode, c.HostPort(), defaultColorCode)
	}
	return offlineStatus
}

func triggerFilterToString(filters []eventingbroker.Filter) string {
	var result []string
	for _, filter := range filters {
		output, err := kyaml.Marshal(filter)
		if err != nil {
			continue
		}
		components := strings.Split(string(output), ":")
		prefixCondition := ""
		if len(components) > 3 {
			prefixCondition = strings.TrimRight(strings.TrimSpace(components[0]), ":")
			components = components[1:]
		}
		if len(components) != 3 {
			continue
		}
		condition := strings.TrimPrefix(components[0], ":\n")
		attribute := strings.TrimRight(strings.TrimSpace(components[1]), ":")
		value := strings.TrimRight(strings.TrimSpace(components[2]), ":")
		switch condition {
		case "exact":
			result = append(result, fmt.Sprintf("%s is %s", attribute, value))
		case "prefix":
			result = append(result, fmt.Sprintf("%s is %s*", attribute, value))
		case "suffix":
			result = append(result, fmt.Sprintf("%s is *%s", attribute, value))
		default:
			result = append(result, fmt.Sprintf("%s is %s %s", attribute, prefixCondition, value))
		}
	}
	return strings.Join(result, ", ")
}

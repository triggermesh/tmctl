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

package sendevent

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	defaultEventType   = "triggermesh-local-event"
	defaultEventSource = "triggermesh-cli"
)

type sendOptions struct {
	Context   string
	ConfigDir string
	Version   string
	CRD       string
	Manifest  *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &sendOptions{}
	var eventType, target string
	sendCmd := &cobra.Command{
		Use:     "send-event [--eventType <type>][--target <name>] <data>",
		Short:   "Send CloudEvent to the target",
		Example: "tmctl send-event '{\"hello\":\"world\"}'",
		RunE: func(cmd *cobra.Command, args []string) error {
			if target == "" {
				target = o.Context
			}
			return o.send(eventType, target, strings.Join(args, " "))
		},
	}
	cobra.OnInitialize(o.initialize)

	sendCmd.Flags().StringVar(&target, "target", "", "Component to send the event to. Default is the broker")
	sendCmd.Flags().StringVar(&eventType, "eventType", defaultEventType, "CloudEvent Type attribute")

	cobra.CheckErr(sendCmd.RegisterFlagCompletionFunc("eventType", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListFilteredEventTypes(o.Context, o.ConfigDir, o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(sendCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListTargets(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	return sendCmd
}

func (o *sendOptions) initialize() {
	o.ConfigDir = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = manifest.New(path.Join(o.ConfigDir, o.Context, triggermesh.ManifestFile))
	crds, err := crd.Fetch(o.ConfigDir, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds

	// try to read manifest even if it does not exists.
	// required for autocompletion.
	_ = o.Manifest.Read()
}

func (o *sendOptions) send(eventType, target, data string) error {
	ctx := context.Background()
	component, err := components.GetObject(target, o.CRD, o.Version, o.Manifest)
	if err != nil {
		return fmt.Errorf("destination target: %w", err)
	}
	consumer, ok := component.(triggermesh.Consumer)
	if !ok {
		return fmt.Errorf("%q is not an event consumer", target)
	}
	port, err := consumer.GetPort(ctx)
	if err != nil {
		return fmt.Errorf("target port: %w", err)
	}

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		return fmt.Errorf("cloudevents client, %w", err)
	}
	event := cloudevents.NewEvent()
	event.SetSource(defaultEventSource)
	event.SetType(eventType)
	contentType := cloudevents.TextPlain
	if json.Valid([]byte(data)) {
		contentType = cloudevents.ApplicationJSON
	}
	if err := event.SetData(contentType, []byte(data)); err != nil {
		return fmt.Errorf("event data: %w", err)
	}

	brokerEndpoint := fmt.Sprintf("http://localhost:%s", port)
	fmt.Printf("Destination: %s(%s)\n", target, brokerEndpoint)
	fmt.Printf("Request:\n------\n%s------", event.String())
	result := c.Send(cloudevents.ContextWithTarget(ctx, brokerEndpoint), event)
	response := "\033[92mOK\033[39m"
	if !cloudevents.IsACK(result) {
		response = fmt.Sprintf("\u001b[31mError\033[39m(%s)", result.Error())
	}
	fmt.Printf("\nResponse: %s\n", response)
	return nil
}

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
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	defaultEventType   = "triggermesh-local-event"
	defaultEventSource = "triggermesh-cli"
)

type CliOptions struct {
	Config   *config.Config
	Manifest *manifest.Manifest
	CRD      map[string]crd.CRD
}

func NewCmd(config *config.Config, manifest *manifest.Manifest, crd map[string]crd.CRD) *cobra.Command {
	o := &CliOptions{
		CRD:      crd,
		Config:   config,
		Manifest: manifest,
	}
	var eventType, target string
	sendCmd := &cobra.Command{
		Use:     "send-event [--eventType <type>][--target <name>] <data>",
		Short:   "Send CloudEvent to the target",
		Example: "tmctl send-event '{\"hello\":\"world\"}'",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"--target", "--eventType"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cobra.CheckErr(o.Manifest.Read())
			if target == "" {
				target = o.Config.Context
			}
			return o.send(eventType, target, strings.Join(args, " "))
		},
	}
	sendCmd.Flags().StringVar(&target, "target", "", "Component to send the event to. Default is the broker")
	sendCmd.Flags().StringVar(&eventType, "eventType", defaultEventType, "CloudEvent Type attribute")

	cobra.CheckErr(sendCmd.RegisterFlagCompletionFunc("eventType", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListFilteredEventTypes(o.Config.Context, o.Config.ConfigHome, o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	cobra.CheckErr(sendCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListTargets(o.Manifest), cobra.ShellCompDirectiveNoFileComp
	}))
	return sendCmd
}

func (o *CliOptions) send(eventType, target, data string) error {
	ctx := context.Background()
	component, err := components.GetObject(target, o.Config, o.Manifest, o.CRD)
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

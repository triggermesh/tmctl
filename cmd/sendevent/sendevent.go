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
	"fmt"
	"path"
	"strings"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
)

const (
	defaultEventType   = "triggermesh-local-event"
	defaultEventSource = "triggermesh-cli"
)

type SendOptions struct {
	Context   string
	ConfigDir string
	EventType string
}

func NewCmd() *cobra.Command {
	o := &SendOptions{}
	sendCmd := &cobra.Command{
		Use:   "send-event <data> [--eventType <type>]",
		Short: "Send CloudEvent to the broker",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"--eventType"}, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Context = viper.GetString("context")
			o.ConfigDir = path.Dir(viper.ConfigFileUsed())
			return o.send(strings.Join(args, " "))
		},
	}
	sendCmd.Flags().StringVar(&o.EventType, "eventType", defaultEventType, "CloudEvent Type attribute")
	return sendCmd
}

func (o *SendOptions) send(data string) error {
	ctx := context.Background()
	broker, err := tmbroker.New(o.Context, o.ConfigDir)
	if err != nil {
		return fmt.Errorf("broker object: %v", err)
	}
	port, err := broker.(triggermesh.Consumer).GetPort(ctx)
	if err != nil {
		return fmt.Errorf("broker socket: %v", err)
	}

	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		return fmt.Errorf("cloudevents client, %w", err)
	}

	event := cloudevents.NewEvent()
	event.SetSource(defaultEventSource)
	event.SetType(o.EventType)
	event.SetData(cloudevents.ApplicationJSON, data)

	brokerEndpoint := fmt.Sprintf("http://localhost:%s", port)
	fmt.Printf("%s -> %s\n", data, brokerEndpoint)
	result := c.Send(cloudevents.ContextWithTarget(ctx, brokerEndpoint), event)
	if cloudevents.IsUndelivered(result) {
		return fmt.Errorf("send event: %w", result)
	}
	fmt.Println("OK")
	return nil
}

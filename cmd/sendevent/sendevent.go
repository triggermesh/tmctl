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

	"github.com/triggermesh/tmctl/pkg/completion"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const (
	manifestFile = "manifest.yaml"

	defaultEventType   = "triggermesh-local-event"
	defaultEventSource = "triggermesh-cli"
)

type SendOptions struct {
	Context   string
	ConfigDir string
	Version   string
	CRD       string
	Manifest  *manifest.Manifest
}

func NewCmd() *cobra.Command {
	o := &SendOptions{}
	var eventType string
	sendCmd := &cobra.Command{
		Use:   "send-event <data> --eventType <type>",
		Short: "Send CloudEvent to the broker",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.send(eventType, strings.Join(args, " "))
		},
	}
	cobra.OnInitialize(o.initialize)
	sendCmd.Flags().StringVar(&eventType, "eventType", defaultEventType, "CloudEvent Type attribute")
	sendCmd.RegisterFlagCompletionFunc("eventType", func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		return completion.ListFilteredEventTypes(o.Context, o.ConfigDir, o.Manifest), cobra.ShellCompDirectiveNoFileComp
	})
	return sendCmd
}

func (o *SendOptions) initialize() {
	o.ConfigDir = path.Dir(viper.ConfigFileUsed())
	o.Context = viper.GetString("context")
	o.Version = viper.GetString("triggermesh.version")
	o.Manifest = manifest.New(path.Join(o.ConfigDir, o.Context, manifestFile))
	crds, err := crd.Fetch(o.ConfigDir, o.Version)
	cobra.CheckErr(err)
	o.CRD = crds

	// try to read manifest even if it does not exists.
	// required for autocompletion.
	o.Manifest.Read()
}

func (o *SendOptions) send(eventType, data string) error {
	ctx := context.Background()
	broker, err := tmbroker.New(o.Context, path.Join(o.ConfigDir, o.Context, manifestFile))
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
	event.SetType(eventType)
	event.SetData(cloudevents.ApplicationJSON, []byte(data))

	brokerEndpoint := fmt.Sprintf("http://localhost:%s", port)
	fmt.Printf("Request:\n%s\nDestination: %s-broker(%s)\n", event.String(), o.Context, brokerEndpoint)
	result := c.Send(cloudevents.ContextWithTarget(ctx, brokerEndpoint), event)
	if cloudevents.IsUndelivered(result) {
		return fmt.Errorf("send event: %w", result)
	}
	fmt.Printf("Response: %s \033[92mOK\033[39m\n", result)
	return nil
}

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

package wiretap

import (
	"context"
	"fmt"
	"io"
	"path"

	"github.com/triggermesh/tmcli/pkg/docker"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
)

type Wiretap struct {
	Broker    string
	ConfigDir string
	// EventType string
	Destination string
}

const (
	image = "gcr.io/knative-releases/knative.dev/eventing/cmd/event_display"
	port  = "8080/tcp"
)

func New(broker, configDir string) *Wiretap {
	return &Wiretap{
		Broker:    broker,
		ConfigDir: path.Join(configDir, broker),
	}
}

func (w *Wiretap) CreateAdapter(ctx context.Context) (io.ReadCloser, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	co := []docker.ContainerOption{
		docker.WithImage(image),
		docker.WithPort(port),
		docker.WithEnv([]string{"K_CONFIG_TRACING={}"}),
	}
	ho := []docker.HostOption{
		docker.WithHostPortBinding(port),
		docker.WithExtraHost(),
	}
	container := &docker.Container{
		Name:                   fmt.Sprintf("%s-wiretap", w.Broker),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}

	if err := container.PullImage(ctx, client, image); err != nil {
		return nil, fmt.Errorf("pull image: %w", err)
	}
	c, err := container.Start(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}
	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("container connect: %w", err)
	}
	w.Destination = fmt.Sprintf("http://host.docker.internal:%s", c.HostPort())
	return c.Logs(ctx, client)
}

func (w *Wiretap) CreateTrigger(eventTypes []string) error {
	trigger := tmbroker.NewTrigger("wiretap", w.Broker, w.ConfigDir, eventTypes)
	trigger.SetTarget("wiretap", w.Destination)
	return trigger.UpdateBrokerConfig()
}

func (w *Wiretap) Cleanup(ctx context.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	trigger := tmbroker.NewTrigger("wiretap", w.Broker, w.ConfigDir, []string{})
	if err := trigger.RemoveTriggerFromConfig(); err != nil {
		return fmt.Errorf("removing trigger: %v", err)
	}
	return docker.ForceStop(ctx, fmt.Sprintf("%s-wiretap", w.Broker), client)
}

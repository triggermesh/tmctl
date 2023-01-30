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
	"path/filepath"
	"time"

	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"

	"github.com/docker/docker/client"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/triggermesh-core/pkg/apis/eventing/v1alpha1"
)

type Wiretap struct {
	Broker      string
	ConfigBase  string
	Destination string

	client *client.Client
}

const (
	image = "gcr.io/knative-releases/knative.dev/eventing/cmd/event_display"
	port  = "8080/tcp"
)

func New(broker, configBase string) (*Wiretap, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return &Wiretap{
		Broker:     broker,
		ConfigBase: configBase,
		client:     dockerClient,
	}, nil
}

func (w *Wiretap) CreateAdapter(ctx context.Context) (io.ReadCloser, error) {
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
		Image:                  image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}
	c, err := container.Start(ctx, w.client, true)
	if err != nil {
		return nil, fmt.Errorf("starting container: %w", err)
	}
	w.Destination = fmt.Sprintf("http://host.docker.internal:%s", c.HostPort())
	return c.Logs(ctx, w.client, time.Now().Add(2*time.Second), true)
}

func (w *Wiretap) CreateTrigger(eventTypes []string) error {
	url, err := apis.ParseURL(w.Destination)
	if err != nil {
		return fmt.Errorf("wiretap URL: %w", err)
	}
	trigger := &tmbroker.Trigger{
		Name:       "wiretap",
		ConfigBase: w.ConfigBase,
		LocalURL:   url,
		TriggerSpec: v1alpha1.TriggerSpec{
			Target: v1.Destination{
				Ref: &v1.KReference{
					Name: "wiretap",
				},
			},
			Broker: v1.KReference{
				Name: w.Broker,
			},
		},
	}
	if err := trigger.WriteLocalConfig(); err != nil {
		return err
	}
	return nil
}

func (w *Wiretap) BrokerLogs(ctx context.Context, c config.BrokerConfig) (io.ReadCloser, error) {
	bro, err := tmbroker.New(w.Broker, filepath.Join(w.ConfigBase, w.Broker, triggermesh.ManifestFile), c)
	if err != nil {
		return nil, err
	}
	broc, err := bro.(triggermesh.Runnable).Info(ctx)
	if err != nil {
		return nil, err
	}
	return broc.Logs(ctx, w.client, time.Now().Add(2*time.Second), true)
}

func (w *Wiretap) Cleanup(ctx context.Context) error {
	trigger := &tmbroker.Trigger{
		Name:       "wiretap",
		ConfigBase: w.ConfigBase,
		TriggerSpec: v1alpha1.TriggerSpec{
			Broker: v1.KReference{
				Name: w.Broker,
			},
		},
	}
	if err := trigger.RemoveFromLocalConfig(); err != nil {
		return fmt.Errorf("removing trigger: %v", err)
	}
	return docker.ForceStop(ctx, fmt.Sprintf("%s-wiretap", w.Broker), w.client)
}

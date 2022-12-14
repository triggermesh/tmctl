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

package broker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/spf13/viper"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
)

var (
	_ triggermesh.Component = (*Broker)(nil)
	_ triggermesh.Runnable  = (*Broker)(nil)
	_ triggermesh.Consumer  = (*Broker)(nil)
)

const (
	BrokerKind  = "RedisBroker"
	TriggerKind = "Trigger"
	APIVersion  = "eventing.triggermesh.io/v1alpha1"

	brokerConfigFile = "broker.conf"
)

type Broker struct {
	ConfigFile string
	Name       string

	spec map[string]interface{}
}

func (b *Broker) asUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion(APIVersion)
	u.SetKind(BrokerKind)
	u.SetName(b.Name)
	u.SetNamespace(triggermesh.Namespace)
	return u, unstructured.SetNestedField(u.Object, nil, "spec")
}

func (b *Broker) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.Object{
		APIVersion: APIVersion,
		Kind:       BrokerKind,
		Metadata: kubernetes.Metadata{
			Name:      b.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": b.Name,
			},
		},
	}, nil
}

func (b *Broker) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := b.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	co, ho, err := adapter.RuntimeParams(o, viper.GetString("triggermesh.broker.image"), additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}

	entrypoint := []string{
		"/memory-broker",
		"start",
		"--memory.buffer-size",
		viper.GetString("triggermesh.broker.memory.buffer-size"),
		"--memory.produce-timeout",
		viper.GetString("triggermesh.broker.memory.produce-timeout"),
		"--broker-config-path",
		"/etc/triggermesh/broker.conf",
	}
	pollingPeriod := viper.GetString("triggermesh.broker.memory.config-polling-period")
	if pollingPeriod != "" {
		entrypoint = append(entrypoint, []string{"--config-polling-period", pollingPeriod}...)
	}
	co = append(co, docker.WithEntrypoint(entrypoint))

	bind := fmt.Sprintf("%s:/etc/triggermesh/broker.conf", b.ConfigFile)
	ho = append(ho, docker.WithVolumeBind(bind))

	name := o.GetName()
	if !strings.HasSuffix(name, "-broker") {
		name = name + "-broker"
	}
	return &docker.Container{
		Name:                   name,
		Image:                  viper.GetString("triggermesh.broker.image"),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (b *Broker) GetKind() string {
	return BrokerKind
}

func (b *Broker) GetName() string {
	return b.Name
}

func (b *Broker) GetAPIVersion() string {
	return APIVersion
}

func (b *Broker) GetSpec() map[string]interface{} {
	return b.spec
}

func (b *Broker) GetPort(ctx context.Context) (string, error) {
	container, err := b.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort(), nil
}

func (b *Broker) ConsumedEventTypes() ([]string, error) {
	return []string{}, nil
}

func (b *Broker) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.Start(ctx, client, restart)
}

func (b *Broker) Stop(ctx context.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return fmt.Errorf("container object: %w", err)
	}
	return container.Remove(ctx, client)
}

func (b *Broker) Info(ctx context.Context) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

func (b *Broker) Logs(ctx context.Context, since time.Time, follow bool) (io.ReadCloser, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	if _, err := container.LookupHostConfig(ctx, client); err != nil {
		return nil, fmt.Errorf("container config: %w", err)
	}
	return container.Logs(ctx, client, since, follow)
}

func New(name, manifestPath string) (triggermesh.Component, error) {
	// create config folder
	if err := os.MkdirAll(filepath.Dir(manifestPath), os.ModePerm); err != nil {
		return nil, fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if _, err := os.Create(manifestPath); err != nil {
			return nil, fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("manifest file access: %w", err)
	}

	config := filepath.Join(filepath.Dir(manifestPath), brokerConfigFile)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if _, err := os.Create(config); err != nil {
			return nil, fmt.Errorf("creating broker config: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("broker config read: %w", err)
	}

	return &Broker{
		ConfigFile: config,
		Name:       name,
	}, nil
}

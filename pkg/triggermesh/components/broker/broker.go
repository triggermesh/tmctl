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
	"os"
	"path"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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
	brokerConfigFile = "broker.conf"
	image            = "tzununbekov/memory-broker"
)

type Broker struct {
	ConfigFile string
	Name       string

	spec map[string]interface{}
}

type Configuration struct {
	Triggers map[string]Trigger `yaml:"triggers"`
}

func (b *Broker) asUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("eventing.triggermesh.io/v1alpha1")
	u.SetKind("Broker")
	u.SetName(b.Name)
	u.SetNamespace(triggermesh.Namespace)
	u.SetLabels(map[string]string{"context": b.Name})
	return u, unstructured.SetNestedField(u.Object, nil, "spec")
}

func (b *Broker) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Broker",
		Metadata: kubernetes.Metadata{
			Name:      b.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": b.Name,
			},
		},
		Spec: map[string]interface{}{"storage": "inmemory"},
	}, nil
}

func (b *Broker) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := b.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image := image
	co, ho, err := adapter.RuntimeParams(o, image, additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}

	bind := fmt.Sprintf("%s:/etc/triggermesh/broker.conf", b.ConfigFile)
	ho = append(ho, docker.WithVolumeBind(bind))

	name := o.GetName()
	if !strings.HasSuffix(name, "-broker") {
		name = name + "-broker"
	}
	return &docker.Container{
		Name:                   name,
		Image:                  image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (b *Broker) GetKind() string {
	return "Broker"
}

func (b *Broker) GetName() string {
	return b.Name
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

func GetTargetTriggers(broker, configBase, target string) ([]triggermesh.Component, error) {
	config, err := readBrokerConfig(path.Join(configBase, broker, brokerConfigFile))
	if err != nil {
		return nil, fmt.Errorf("read broker config: %w", err)
	}
	var triggers []triggermesh.Component
	for name, trigger := range config.Triggers {
		if trigger.GetTarget().Component != target {
			continue
		}
		f := Filter{}
		if len(trigger.Filters) != 0 {
			f = trigger.Filters[0]
		}
		t, err := NewTrigger(name, broker, configBase, trigger.Target.URL, trigger.Target.Component, f)
		if err != nil {
			return nil, fmt.Errorf("creating trigger object: %w", err)
		}
		triggers = append(triggers, t)
	}
	return triggers, nil
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

func New(name, manifestPath string) (triggermesh.Component, error) {
	// create config folder
	if err := os.MkdirAll(path.Dir(manifestPath), os.ModePerm); err != nil {
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

	config := path.Join(path.Dir(manifestPath), brokerConfigFile)
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

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

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/adapter"
)

var (
	_ triggermesh.Component = (*Broker)(nil)
	_ triggermesh.Runnable  = (*Broker)(nil)
	_ triggermesh.Consumer  = (*Broker)(nil)
)

const (
	manifestFile     = "manifest.yaml"
	brokerConfigFile = "broker.conf"
	image            = "tzununbekov/memory-broker"
)

type Broker struct {
	ConfigFile string
	Name       string

	image string
	spec  map[string]interface{}
	// Configuration Configuration
}

type Configuration struct {
	Triggers []TriggerSpec `yaml:"triggers"`
}

func (b *Broker) AsUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("eventing.triggermesh.io/v1alpha1")
	u.SetKind("Broker")
	u.SetName(b.Name)
	u.SetLabels(map[string]string{"context": b.Name})
	return u, unstructured.SetNestedField(u.Object, nil, "spec")
}

func (b *Broker) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Broker",
		Metadata: kubernetes.Metadata{
			Name: b.Name,
			Labels: map[string]string{
				"triggermesh.io/context": b.Name,
			},
		},
		Spec: map[string]interface{}{"storage": "inmemory"},
	}, nil
}

func (b *Broker) AsContainer(opts ...docker.ContainerOption) (*docker.Container, error) {
	o, err := b.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	b.image = image
	co, ho, err := adapter.RuntimeParams(o, b.image)
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
		CreateHostOptions:      ho,
		CreateContainerOptions: append(co, opts...),
	}, nil
}

func (b *Broker) GetKind() string {
	return "Broker"
}

func (b *Broker) GetName() string {
	return b.Name
}

func (b *Broker) GetImage() string {
	return b.image
}

func (b *Broker) GetSpec() map[string]interface{} {
	return b.spec
}

func (b *Broker) GetPort(ctx context.Context) (string, error) {
	client, err := docker.NewClient()
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}
	container, err := b.AsContainer()
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	if container, err = container.LookupHostConfig(ctx, client); err != nil {
		return "", fmt.Errorf("container runtime config: %w", err)
	}
	return container.HostPort(), nil
}

func (b *Broker) ConsumedEventTypes() ([]string, error) {
	return []string{}, nil
}

func New(name, brokerConfigDir string) (*Broker, error) {
	// create config folder
	if err := os.MkdirAll(brokerConfigDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	manifest := path.Join(brokerConfigDir, manifestFile)
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		if _, err := os.Create(manifest); err != nil {
			return nil, fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("manifest file access: %w", err)
	}

	config := path.Join(brokerConfigDir, brokerConfigFile)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		if _, err := os.Create(config); err != nil {
			return nil, fmt.Errorf("config file: %w", err)
		}
	}

	return &Broker{
		ConfigFile: config,
		Name:       name,
	}, nil
}

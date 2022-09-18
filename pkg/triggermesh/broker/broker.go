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
	"fmt"
	"os"
	"path"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/adapter"
)

var _ triggermesh.Component = (*Broker)(nil)

const (
	image = "tzununbekov/memory-broker"
)

type Broker struct {
	ManifestFile string
	ConfigFile   string
	Name         string

	image string
	// spec  map[string]interface{}
	// Configuration Configuration
}

type Configuration struct {
	Triggers []TriggerSpec `yaml:"triggers"`
}

func (b *Broker) AsUnstructured() (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("eventing.triggermesh.io/v1alpha1")
	u.SetKind("Broker")
	u.SetName(b.Name)
	u.SetLabels(map[string]string{"context": b.Name})
	return u, unstructured.SetNestedField(u.Object, nil, "spec")
}

func (b *Broker) AsK8sObject() (*kubernetes.Object, error) {
	return &kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Broker",
		Metadata: kubernetes.Metadata{
			Name: b.Name,
			Labels: map[string]string{
				"triggermesh.io/context": b.Name,
			},
		},
		Spec: map[string]interface{}{"storage": viper.GetString("storage")},
	}, nil
}

func (b *Broker) AsContainer() (*docker.Container, error) {
	o, err := b.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	b.image = image
	co, ho, err := adapter.RuntimeParams(o, b.image)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   o.GetName(),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (b *Broker) GetKind() string {
	return "Broker"
}

func (b *Broker) GetName() string {
	return fmt.Sprintf("%s-broker", b.Name)
}

func (b *Broker) GetImage() string {
	return b.image
}

func NewBroker(manifest, name string) (*Broker, error) {
	// create config folder
	if err := os.MkdirAll(path.Dir(manifest), os.ModePerm); err != nil {
		return nil, fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		if _, err := os.Create(manifest); err != nil {
			return nil, fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("manifest file access: %w", err)
	}

	configFile := path.Join(path.Dir(manifest), "broker.conf")
	if _, err := os.Create(configFile); err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	return &Broker{
		ManifestFile: manifest,
		ConfigFile:   configFile,
		Name:         name,
	}, nil
}

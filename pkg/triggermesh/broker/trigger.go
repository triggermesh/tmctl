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
	"path"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/manifest"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ triggermesh.Component = (*Trigger)(nil)

type Trigger struct {
	ManifestFile string
	BrokerConfig string
	Broker       string
	Name         string

	spec TriggerSpec
}

type TriggerSpec struct {
	Name    string
	Filters []Filter `yaml:"filters,omitempty"`
	Targets []Target `yaml:"targets"`
}

type Filter struct {
	Exact Exact `yaml:"exact"`
}

type Exact struct {
	Type string `yaml:"type"`
}

type Target struct {
	URL             string `yaml:"url"`
	DeliveryOptions struct {
		Retries       int    `yaml:"retries,omitempty"`
		BackoffDelay  string `yaml:"backoffDelay,omitempty"`
		BackoffPolicy string `yaml:"backoffPolicy,omitempty"`
	} `yaml:"deliveryOptions,omitempty"`
}

func (t *Trigger) AsUnstructured() (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("eventing.triggermesh.io/v1alpha1")
	u.SetKind("Broker")
	u.SetName(t.Name)
	u.SetLabels(map[string]string{"context": t.Broker})
	return u, unstructured.SetNestedField(u.Object, t.spec, "spec")
}

func (t *Trigger) AsK8sObject() (*kubernetes.Object, error) {
	return &kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Trigger",
		Metadata: kubernetes.Metadata{
			Name: t.Name,
			Labels: map[string]string{
				"triggermesh.io/context": t.Broker,
			},
		},
		Spec: map[string]interface{}{
			"filter":  t.spec.Filters,
			"targets": t.spec.Targets,
		},
	}, nil
}

func (t *Trigger) AsContainer() (*docker.Container, error) {
	return nil, nil
}

func (t *Trigger) GetKind() string {
	return "Trigger"
}

func (t *Trigger) GetName() string {
	return t.Name
}

func (t *Trigger) GetImage() string {
	return ""
}

func NewTrigger(name, manifest, broker, eventType string) *Trigger {
	return &Trigger{
		ManifestFile: manifest,
		BrokerConfig: path.Join(path.Dir(manifest), "broker.conf"),
		Broker:       broker,
		Name:         name,
		spec: TriggerSpec{
			Filters: []Filter{
				{Exact{Type: eventType}},
			}},
	}
}

func AppendTriggerToBroker(config Configuration, name, eventType string) Configuration {
	for _, trigger := range config.Triggers {
		if trigger.Name == name {
			trigger.Filters[0].Exact.Type = eventType
			return config
		}
	}
	config.Triggers = append(config.Triggers, TriggerSpec{Name: name, Filters: []Filter{{Exact: Exact{Type: eventType}}}})
	return config
}

func TriggerObjectsFromBrokerConfig(config Configuration, broker string) []kubernetes.Object {
	var triggers []kubernetes.Object
	for _, trigger := range config.Triggers {
		triggers = append(triggers, kubernetes.Object{
			APIVersion: "eventing.triggermesh.io/v1alpha1",
			Kind:       "Trigger",
			Metadata: kubernetes.Metadata{
				Name: trigger.Name,
				Labels: map[string]string{
					"triggermesh.io/context": broker,
				},
			},
			Spec: map[string]interface{}{
				"filters": trigger.Filters,
				"targets": trigger.Targets,
			},
		})
	}
	return triggers
}

func CreateTrigger(name, manifestFile, broker, eventType string) (*kubernetes.Object, error) {
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil, fmt.Errorf("unable to read the manifest: %w", err)
	}
	brokerConfFile := path.Join("/Users/tzununbekov/.triggermesh/cli", broker, "broker.conf")
	config, err := ReadConfigFile(brokerConfFile)
	if err != nil {
		return nil, fmt.Errorf("broker config read: %w", err)
	}
	config = AppendTriggerToBroker(config, name, eventType)
	triggers := TriggerObjectsFromBrokerConfig(config, broker)
	var dirty bool
	for _, trigger := range triggers {
		newObject, err := manifest.Add(trigger)
		if err != nil {
			return nil, fmt.Errorf("adding trigger: %w", err)
		}
		if newObject {
			dirty = true
		}
	}
	if !dirty {
		return nil, nil
	}
	if err := manifest.Write(); err != nil {
		return nil, fmt.Errorf("manifest write: %w", err)
	}
	if err := WriteConfigFile(brokerConfFile, &config); err != nil {
		return nil, fmt.Errorf("broker config write: %w", err)
	}
	return nil, nil
}

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

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

var _ triggermesh.Component = (*Trigger)(nil)

type Trigger struct {
	Broker          string `yaml:"-"`
	BrokerConfigDir string `yaml:"-"`
	Name            string `yaml:"-"`

	Filters []Filter `yaml:"filters,omitempty"`
	Target  Target   `yaml:"target"`
}

type Filter struct {
	All    []Filter          `yaml:"all,omitempty"`
	Any    []Filter          `yaml:"any,omitempty"`
	Not    *Filter           `yaml:"not,omitempty"`
	Exact  map[string]string `yaml:"exact,omitempty"`
	Prefix map[string]string `yaml:"prefix,omitempty"`
	Suffix map[string]string `yaml:"suffix,omitempty"`
	CESQL  string            `yaml:"cesql,omitempty"`
}

type Target struct {
	URL             string `yaml:"url"`
	Component       string `yaml:"component,omitempty"` // for local version only
	DeliveryOptions struct {
		Retry         int32  `yaml:"retry,omitempty"`
		BackoffDelay  string `yaml:"backoffDelay,omitempty"`
		BackoffPolicy string `yaml:"backoffPolicy,omitempty"`
		DeadLetterURL string `yaml:"deadLetterURL,omitempty"`
	} `yaml:"deliveryOptions,omitempty"`
}

func (t *Trigger) AsUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("eventing.triggermesh.io/v1alpha1")
	u.SetKind("Trigger")
	u.SetName(t.Name)
	u.SetLabels(map[string]string{"context": t.Broker})
	return u, unstructured.SetNestedField(u.Object, t, "spec")
}

func (t *Trigger) AsK8sObject() (kubernetes.Object, error) {
	spec := map[string]interface{}{
		"target": t.Target,
	}
	if len(t.Filters) != 0 {
		spec["filters"] = t.Filters
	}
	return kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Trigger",
		Metadata: kubernetes.Metadata{
			Name: t.Name,
			Labels: map[string]string{
				"triggermesh.io/context": t.Broker,
			},
		},
		Spec: spec,
	}, nil
}

func (t *Trigger) GetKind() string {
	return "Trigger"
}

func (t *Trigger) GetName() string {
	return t.Name
}

func (t *Trigger) GetTarget() Target {
	return t.Target
}

func (t *Trigger) GetFilters() []Filter {
	return t.Filters
}

func (t *Trigger) GetSpec() map[string]interface{} {
	return map[string]interface{}{
		"filters": t.Filters,
		"target":  t.Target,
	}
}

func NewTrigger(name, broker, configDir string, filter *Filter) *Trigger {
	trigger := Trigger{
		Name:            name,
		Broker:          broker,
		BrokerConfigDir: configDir,
	}
	if filter != nil {
		trigger.Filters = []Filter{*filter}
	}
	return &trigger
}

func (t *Trigger) SetTarget(component, destination string) {
	t.Target = Target{
		Component: component,
		URL:       destination,
	}
}

func (t *Trigger) LookupTrigger() error {
	configFile := path.Join(t.BrokerConfigDir, "broker.conf")
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	trigger, exists := configuration.Triggers[t.Name]
	if !exists {
		return fmt.Errorf("trigger %q not found", t.Name)
	}
	t.Filters = trigger.Filters
	t.Target = trigger.Target
	return nil
}

func (t *Trigger) RemoveTriggerFromConfig() error {
	configFile := path.Join(t.BrokerConfigDir, "broker.conf")
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	delete(configuration.Triggers, t.Name)
	return writeBrokerConfig(configFile, &configuration)
}

func (t *Trigger) UpdateBrokerConfig() error {
	configFile := path.Join(t.BrokerConfigDir, "broker.conf")
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}

	trigger, exists := configuration.Triggers[t.Name]
	if exists {
		trigger.Filters = t.Filters
		trigger.Target = t.Target
		configuration.Triggers[t.Name] = trigger
	} else {
		if configuration.Triggers == nil {
			configuration.Triggers = make(map[string]Trigger, 1)
		}
		configuration.Triggers[t.Name] = *t
	}
	return writeBrokerConfig(configFile, &configuration)
}

func (t *Trigger) UpdateManifest() error {
	m := manifest.New(path.Join(t.BrokerConfigDir, manifestFile))
	if err := m.Read(); err != nil {
		return fmt.Errorf("manifest read: %w", err)
	}
	o, err := t.AsK8sObject()
	if err != nil {
		return fmt.Errorf("trigger object: %w", err)
	}
	if dirty := m.Add(o); dirty {
		return m.Write()
	}
	return nil
}

func (f *Filter) String() (string, error) {
	output, err := yaml.Marshal(f)
	return string(output), err
}

func FilterExactType(eventType string) *Filter {
	return &Filter{
		Exact: map[string]string{"type": eventType},
	}
}

func readBrokerConfig(path string) (Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read file: %w", err)
	}
	var config Configuration
	return config, yaml.Unmarshal(data, &config)
}

func writeBrokerConfig(path string, configuration *Configuration) error {
	out, err := yaml.Marshal(configuration)
	if err != nil {
		return fmt.Errorf("marshal broker configuration: %w", err)
	}
	return os.WriteFile(path, out, os.ModePerm)
}

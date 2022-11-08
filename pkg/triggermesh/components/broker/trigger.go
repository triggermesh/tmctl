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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"

	duckv1 "knative.dev/pkg/apis/duck/v1"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"
	eventingv1alpha1 "github.com/triggermesh/triggermesh-core/pkg/apis/eventing/v1alpha1"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

var _ triggermesh.Component = (*Trigger)(nil)

type Trigger struct {
	Name       string `yaml:"-"`
	ConfigBase string `yaml:"-"`

	eventingv1alpha1.TriggerSpec `yaml:"spec,omitempty"`
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
			Name:      t.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": t.Broker.Name,
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

func (t *Trigger) GetAPIVersion() string {
	return "v1alpha1"
}

func (t *Trigger) GetTarget() duckv1.Destination {
	return t.Target
}

func (t *Trigger) GetFilters() []eventingbroker.Filter {
	return t.Filters
}

func (t *Trigger) GetSpec() map[string]interface{} {
	return map[string]interface{}{
		"filters": t.Filters,
		"target":  t.Target,
	}
}

func NewTrigger(name, broker, configBase string, target triggermesh.Component, filter *eventingbroker.Filter) (triggermesh.Component, error) {
	if name == "" {
		filterStruct, _ := yaml.Marshal(filter)
		// in case of event types hash collision, replace with sha256
		hash := md5.Sum([]byte(fmt.Sprintf("%s-%s", target.GetName(), string(filterStruct))))
		name = fmt.Sprintf("%s-trigger-%s", broker, hex.EncodeToString(hash[:4]))
	}

	trigger := &Trigger{
		Name:       name,
		ConfigBase: configBase,
		TriggerSpec: eventingv1alpha1.TriggerSpec{
			Broker: duckv1.KReference{
				Name: broker,
			},
			Target: duckv1.Destination{
				Ref: &duckv1.KReference{
					Kind:       target.GetKind(),
					Name:       target.GetName(),
					APIVersion: target.GetAPIVersion(),
				},
			},
		},
	}
	if filter != nil {
		trigger.Filters = []eventingbroker.Filter{*filter}
	}
	return trigger, nil
}

func (t *Trigger) SetTarget(target triggermesh.Component) {
	t.Target = duckv1.Destination{
		Ref: &duckv1.KReference{
			Kind:       target.GetKind(),
			Name:       target.GetName(),
			APIVersion: target.GetAPIVersion(),
		},
	}
}

func (t *Trigger) LookupTrigger() error {
	configFile := path.Join(t.ConfigBase, t.Broker.Name, brokerConfigFile)
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
	configFile := path.Join(t.ConfigBase, t.Broker.Name, brokerConfigFile)
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	delete(configuration.Triggers, t.Name)
	return writeBrokerConfig(configFile, &configuration)
}

func (t *Trigger) UpdateBrokerConfig() error {
	configFile := path.Join(t.ConfigBase, t.Broker.Name, brokerConfigFile)
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

func FilterExactAttribute(attribute, value string) *eventingbroker.Filter {
	return &eventingbroker.Filter{
		Exact: map[string]string{attribute: value},
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

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

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"

	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

type Configuration struct {
	Triggers map[string]LocalTriggerSpec `yaml:"triggers"`
}

type LocalTriggerSpec struct {
	Filters []eventingbroker.Filter `yaml:"filters,omitempty"`
	Target  LocalTarget             `yaml:"target"`
}

type LocalTarget struct {
	URL             string                          `yaml:"url,omitempty"`
	Component       string                          `yaml:"component,omitempty"`
	DeliveryOptions *eventingbroker.DeliveryOptions `yaml:"deliveryOptions,omitempty"`
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

func (t *Trigger) WriteLocalConfig() error {
	configFile := path.Join(t.configBase, t.Broker.Name, brokerConfigFile)
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}

	triggerSpec := LocalTriggerSpec{
		Filters: t.Filters,
		Target: LocalTarget{
			URL:       t.localURL.String(),
			Component: t.componentName,
		},
	}

	trigger, exists := configuration.Triggers[t.name]
	if exists {
		trigger.Filters = triggerSpec.Filters
		trigger.Target = triggerSpec.Target
		configuration.Triggers[t.name] = trigger
	} else {
		if configuration.Triggers == nil {
			configuration.Triggers = make(map[string]LocalTriggerSpec, 1)
		}
		configuration.Triggers[t.name] = triggerSpec
	}
	return writeBrokerConfig(configFile, &configuration)
}

func (t *Trigger) RemoveFromLocalConfig() error {
	configFile := path.Join(t.configBase, t.Broker.Name, brokerConfigFile)
	configuration, err := readBrokerConfig(configFile)
	if err != nil {
		return fmt.Errorf("broker config: %w", err)
	}
	delete(configuration.Triggers, t.name)
	return writeBrokerConfig(configFile, &configuration)
}

func GetTargetTriggers(broker, configBase string, target triggermesh.Component) ([]triggermesh.Component, error) {
	config, err := readBrokerConfig(path.Join(configBase, broker, brokerConfigFile))
	if err != nil {
		return nil, fmt.Errorf("read broker config: %w", err)
	}
	var triggers []triggermesh.Component
	for name, trigger := range config.Triggers {
		if trigger.Target.Component != target.GetName() {
			continue
		}
		var filter *eventingbroker.Filter
		if len(trigger.Filters) == 1 {
			filter = &trigger.Filters[0]
		}
		trigger, err := NewTrigger(name, broker, configBase, target, filter)
		if err != nil {
			return nil, fmt.Errorf("creating trigger: %w", err)
		}
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

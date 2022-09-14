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
	"github.com/triggermesh/tmcli/pkg/kubernetes"
)

type Trigger struct {
	Name    string   `yaml:"name"`
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

func AppendTriggerToBroker(config Configuration, name, eventType string) Configuration {
	for _, trigger := range config.Triggers {
		if trigger.Name == name {
			trigger.Filters[0].Exact.Type = eventType
			return config
		}
	}
	config.Triggers = append(config.Triggers, Trigger{Name: name, Filters: []Filter{{Exact: Exact{Type: eventType}}}})
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

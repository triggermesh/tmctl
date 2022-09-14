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
	"reflect"

	"gopkg.in/yaml.v3"

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

func CreateTriggerObject(name, eventType, targetURL, broker string) kubernetes.Object {
	return kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Trigger",
		Metadata: kubernetes.Metadata{
			Name: name,
			Labels: map[string]string{
				"triggermesh.io/context": broker,
			},
		},
		Spec: map[string]interface{}{
			"filters": []Filter{
				{Exact: Exact{Type: eventType}},
			},
			"targets": []Target{
				{URL: targetURL},
			},
		},
	}
}

func AppendTriggerToConfig(object kubernetes.Object, config Configuration) (Configuration, bool) {
	filters, set := object.Spec["filters"]
	if !set {
		return Configuration{}, false
	}
	targets, set := object.Spec["targets"]
	if !set {
		return Configuration{}, false
	}
	t := Trigger{
		Name:    object.Metadata.Name,
		Filters: filters.([]Filter),
		Targets: targets.([]Target),
	}

	for k, v := range config.Triggers {
		if v.Name == t.Name {
			if reflect.DeepEqual(config.Triggers[k], t) {
				return config, false
			}
			config.Triggers[k] = t
			return config, true
		}
	}
	return Configuration{
		Triggers: append(config.Triggers, t),
	}, true
}

func ReadConfigFile(path string) (Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read file: %w", err)
	}
	var config Configuration
	return config, yaml.Unmarshal(data, &config)
}

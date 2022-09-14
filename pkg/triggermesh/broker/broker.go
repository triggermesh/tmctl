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
	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmcli/pkg/kubernetes"
)

const brokerConfig = `triggers:
- name: trigger1
  filters:
  - exact:
      type: example.type
  targets:
  - url: http://localhost:8888
    deliveryOptions:
      retries: 2
      backoffDelay: 2s
      backoffPolicy: linear
- name: trigger2
  targets:
  - url: http://localhost:9999
    deliveryOptions:
      retries: 5
      backoffDelay: 5s
      backoffPolicy: constant
      deadLetterURL: http://localhost:9090`

type Configuration struct {
	Triggers []Trigger `yaml:"triggers"`
}

func CreateBrokerObject(name, manifestFile string) (*kubernetes.Object, bool, error) {
	// create config folder
	if err := os.MkdirAll(path.Dir(manifestFile), os.ModePerm); err != nil {
		return nil, false, fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		if _, err := os.Create(manifestFile); err != nil {
			return nil, false, fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return nil, false, fmt.Errorf("manifest file access: %w", err)
	}

	broker := kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Broker",
		Metadata: kubernetes.Metadata{
			Name: name,
			Labels: map[string]string{
				"triggermesh.io/context": name,
			},
		},
		Spec: map[string]interface{}{"storage": viper.GetString("storage")},
	}

	manifest := kubernetes.NewManifest(manifestFile)
	dirty, err := manifest.Add(broker)
	if err != nil {
		return nil, false, fmt.Errorf("manifest update: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return nil, false, fmt.Errorf("manifest write operation: %w", err)
		}
	}
	return &broker, dirty, nil
}

func AddTarget(c *Configuration, triggerName, targetURL string) *Configuration {
	for k, trigger := range c.Triggers {
		if trigger.Name != triggerName {
			continue
		}
		for _, target := range trigger.Targets {
			if target.URL == targetURL {
				return c
			}
		}
		c.Triggers[k].Targets = append(c.Triggers[k].Targets, Target{URL: targetURL})
	}
	return c
}

func ReadConfigFile(path string) (Configuration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("read file: %w", err)
	}
	var config Configuration
	return config, yaml.Unmarshal(data, &config)
}

func WriteConfigFile(filePath string, config *Configuration) error {
	out, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal broker configuration: %w", err)
	}
	return os.WriteFile(filePath, out, os.ModePerm)
}

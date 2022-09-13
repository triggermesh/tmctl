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

type Filter struct {
	Exact Exact `yaml:"exact"`
}

type Exact struct {
	Type string `yaml:"type"`
}

type Target struct {
	URL             string `yaml:"url"`
	DeliveryOptions struct {
		Retries       int    `yaml:"retries"`
		BackoffDelay  string `yaml:"backoffDelay"`
		BackoffPolicy string `yaml:"backoffPolicy"`
	} `yaml:"deliveryOptions"`
}

type triggerK8s struct {
	Name string
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func (c *Configuration) AddTrigger(name, targetURL, eventType string) {
	trigger := Trigger{
		Name: name,
		Filters: []Filter{
			{
				Exact: Exact{
					Type: eventType,
				},
			},
		},
		Targets: []Target{
			{
				URL: targetURL,
			},
		},
	}
	c.Triggers = append(c.Triggers, trigger)
}

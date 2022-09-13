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

func CreateTriggerObject(name, eventType, targetURL, broker string) kubernetes.Object {
	filters := []Filter{
		{
			Exact: Exact{
				Type: eventType,
			},
		},
	}
	targets := []Target{
		{
			URL: targetURL,
		},
	}

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
			"filters": filters,
			"targets": targets,
		},
	}
}

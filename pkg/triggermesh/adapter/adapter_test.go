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

package adapter

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// var testObjects = map[string]struct {
// 	object triggermesh.Component
// }{
// 	"from-image": {
// 		object: service.New("test-service", "registry/image", "foo", service.Consumer, map[string]string{"foo-env": "foo-env-value"}),
// 	},
// 	"source": {
// 		object: source.New("test-source", test.CRD(), "httppoller", "foo", "latest", map[string]string{
// 			"endpoint":  "https://www.example.com",
// 			"eventType": "test-event",
// 			"interval":  "30s",
// 			"method":    "GET",
// 		}, nil),
// 	},
// 	"target": {
// 		object: target.New("test-target", test.CRD(), "http", "foo", "latest", map[string]string{
// 			"endpoint":           "https://www.example.com",
// 			"method":             "GET",
// 			"response.eventType": "test-event.response",
// 		}),
// 	},
// 	"transformation": {
// 		object: transformation.New("test-transformation", test.CRD(), "transformation", "foo", "latest", map[string]interface{}{}),
// 	},
// }

func newUnstructured(t *testing.T, name, kind, api string, spec map[string]interface{}) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetAPIVersion(api)
	u.SetKind(kind)
	u.SetName(name)
	assert.NoError(t, unstructured.SetNestedField(u.Object, spec, "spec"))
	return u
}

func TestRuntimeParams(t *testing.T) {
	testObjects := map[string]struct {
		object unstructured.Unstructured
	}{
		"source": {
			object: newUnstructured(t, "test-source", "HTTPPollerSource", "sources.triggermesh.io/v1alpha1", map[string]interface{}{}),
		},
		"target": {
			object: newUnstructured(t, "test-target", "CloudEventsTarget", "targets.triggermesh.io/v1alpha1", map[string]interface{}{}),
		},
		"transformation": {
			object: newUnstructured(t, "test-transformation", "Transformation", "flow.triggermesh.io/v1alpha1", map[string]interface{}{}),
		},
		"service": {
			object: newUnstructured(t, "test-service", "Service", "flow.triggermesh.io/v1alpha1", map[string]interface{}{}),
		},
	}

	for name, test := range testObjects {
		t.Run(name, func(t *testing.T) {
			co, ho, err := RuntimeParams(test.object, "registry/image", map[string]string{"additional-env": "value"})
			assert.NoError(t, err)

			cc := &container.Config{}
			hc := &container.HostConfig{}
			for _, opt := range co {
				opt(cc)
			}
			for _, opt := range ho {
				opt(hc)
			}
			// validate cc and hc
		})
	}
}

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

package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/test"
)

func TestRead(t *testing.T) {
	m := New(test.Manifest())
	assert.NoError(t, m.Read())
	assert.Lenf(t, m.Objects, 7, "Test manifest %q len is incorrect", test.Manifest())

	existingSockeye := service.New("sockeye", "docker.io/n3wscott/sockeye:v0.7.0", "foo", service.Consumer, nil)

	cases := map[string]struct {
		wantUpdate bool
		cleanup    bool
		component  triggermesh.Component
	}{
		"existing component": {
			wantUpdate: false,
			component:  existingSockeye,
		},
		"updated component": {
			wantUpdate: true,
			component: service.New("sockeye", "docker.io/n3wscott/sockeye:v0.7.0", "foo", service.Consumer, map[string]string{
				"env-var": "env-value",
			}),
		},
		"new component": {
			wantUpdate: true,
			cleanup:    true,
			component:  service.New("test-service", "triggermesh/image", "foo", service.Consumer, nil),
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			changed, err := m.Add(test.component)
			assert.NoError(t, err)
			assert.Equal(t, test.wantUpdate, changed, "Manifest update diff")
			if test.cleanup {
				assert.NoError(t, m.Remove(test.component.GetName(), test.component.GetKind()))
			}
		})
	}

	_, err := m.Add(existingSockeye)
	assert.NoError(t, err)
	assert.Lenf(t, m.Objects, 7, "Test manifest %q objects len differs after test", test.Manifest())
}

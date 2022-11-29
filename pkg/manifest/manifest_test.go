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

	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/test"
)

func TestRead(t *testing.T) {
	m := New("wrong/path")
	assert.Error(t, m.Read())

	m = New(test.Manifest())
	assert.NoError(t, m.Read())
	assert.Lenf(t, m.Objects, 7, "Test manifest %q len is incorrect", test.Manifest())

	existingComponent := service.New("sockeye", "docker.io/n3wscott/sockeye:v0.7.0", "foo", service.Consumer, nil)
	updatedComponent := service.New("sockeye", "docker.io/n3wscott/sockeye:v0.7.0", "foo", service.Consumer, map[string]string{
		"new-env-var": "new-env-value",
	})
	newComponent := service.New("test-service", "triggermesh/image", "foo", service.Consumer, nil)

	changed, err := m.Add(existingComponent)
	assert.NoError(t, err)
	assert.Equal(t, false, changed, "Existing component triggered manifest update")

	changed, err = m.Add(updatedComponent)
	assert.NoError(t, err)
	assert.Equal(t, true, changed, "Updated component didn't trigger manifest update")

	changed, err = m.Add(newComponent)
	assert.NoError(t, err)
	assert.Equal(t, true, changed, "New component didn't trigger manifest update")

	assert.NoError(t, m.Remove(newComponent.GetName(), newComponent.GetKind()))

	changed, err = m.Add(existingComponent)
	assert.NoError(t, err)
	assert.Equal(t, true, changed, "Restoring original component didn't update the manifest")

	assert.Lenf(t, m.Objects, 7, "Test manifest %q objects len differs after test", test.Manifest())
}

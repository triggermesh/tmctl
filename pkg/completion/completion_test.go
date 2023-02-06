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

package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
	"github.com/triggermesh/tmctl/test"
)

const version = "v1.21.1"

func TestListSources(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	expectedSources := []string{"foo-awss3source", "foo-transformation"}
	assert.Equal(t, expectedSources, ListSources(m))
}

func TestListTargets(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	expectedTargets := []string{"sockeye", "foo-transformation"}
	assert.Equal(t, expectedTargets, ListTargets(m))
}

func TestListAll(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	assert.Len(t, ListAll(m), 7)
}

func TestListEventTypes(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	expectedEventTypes := []string{
		"com.amazon.s3.objectcreated",
		"com.amazon.s3.objectremoved",
		"com.amazon.s3.replication",
		"com.amazon.s3.testevent",
		"foo-transformation.output",
	}
	c := &config.Config{
		Triggermesh: config.TmConfig{ComponentsVersion: version},
	}
	assert.Equal(t, expectedEventTypes, ListEventTypes(m, c, test.CRD()))
}

func TestFilteredEventTypes(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	expectedFilteresEventTypes := []string{
		"foo-transformation.output",
		"com.amazon.s3.objectcreated",
	}
	assert.Equal(t, expectedFilteresEventTypes, ListFilteredEventTypes("", test.ConfigBase(), m))
}

func TestSpecFromCRD(t *testing.T) {
	s3crd := test.CRD()["awss3source"]
	exists, arnOptions := SpecFromCRD(s3crd, "auth", "credentials")
	credsStruct := map[string]crd.Property{
		"accessKeyID": {
			Required:    false,
			Typ:         "string/secret",
			Description: "Access key ID.",
		},
		"secretAccessKey": {
			Required:    false,
			Typ:         "string/secret",
			Description: "Secret access key.",
		},
	}
	assert.True(t, exists)
	assert.Equal(t, credsStruct, arnOptions)

	exists, r := SpecFromCRD(s3crd, "nonexisting")
	assert.False(t, exists)
	assert.Equal(t, map[string]crd.Property{}, r)
}

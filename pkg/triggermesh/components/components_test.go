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

package components

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/test"
)

const version = "v1.21.1"

func TestGetObject(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())
	c := &config.Config{
		CRDPath:     test.CRD(),
		Triggermesh: config.TmConfig{ComponentsVersion: version},
	}
	for _, object := range m.Objects {
		component, err := GetObject(object.Metadata.Name, c, m)
		assert.NoError(t, err)
		if component == nil {
			continue
		}
		k8sObject, err := component.AsK8sObject()
		assert.NoError(t, err)
		object.Metadata.Namespace = triggermesh.Namespace
		assert.Equal(t, object.Metadata, k8sObject.Metadata)
		// object's spec created from the CRD and decoded from the manifest
		// may have different types, e.g. component's v1.KReference will be
		// translated as map[string]interface{} from the manifest.
	}
}

func TestProcessSecrets(t *testing.T) {
	m := manifest.New(test.Manifest())
	assert.NoError(t, m.Read())

	specs := map[string]map[string]string{
		"input args": {
			"auth.credentials.accessKeyID":     "AWSACCESSKEYID",
			"auth.credentials.secretAccessKey": "AWSSECRETACCESSKEY",
		},
		"spec refs": {
			"auth.credentials.accessKeyID.valueFromSecret.key":      "accessKeyID",
			"auth.credentials.accessKeyID.valueFromSecret.name":     "foo-awss3source-secret",
			"auth.credentials.secretAccessKey.valueFromSecret.key":  "secretAccessKey",
			"auth.credentials.secretAccessKey.valueFromSecret.name": "foo-awss3source-secret",
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			s := source.New("foo-awss3source", test.CRD(), "awss3source", "foo", version, spec, nil)
			secrets, plainValues, err := ProcessSecrets(s.(triggermesh.Parent), m)
			assert.NoError(t, err)

			assert.Equal(t, "AWSACCESSKEYID", plainValues["accessKeyID"])
			assert.Equal(t, "AWSSECRETACCESSKEY", plainValues["secretAccessKey"])
			assert.Equal(t, "QVdTQUNDRVNTS0VZSUQ=", secrets[0].GetSpec()["accessKeyID"])
			assert.Equal(t, "QVdTU0VDUkVUQUNDRVNTS0VZ", secrets[0].GetSpec()["secretAccessKey"])
		})
	}
}

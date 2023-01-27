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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/triggermesh/tmctl/test"
)

var specs = map[string]struct {
	kind      string
	spec      map[string]interface{}
	wantError bool
}{
	"nil spec": {
		kind:      "awss3source",
		wantError: true,
		spec:      nil,
	},
	"wrong kind": {
		kind:      "nonexistingsource",
		wantError: true,
	},
	"malformed spec 1": {
		kind:      "httptarget",
		wantError: true,
		spec: map[string]interface{}{
			"endpoint": "examplecom",
			"method":   "GET",
		},
	},
	"malformed spec 2": {
		kind:      "httptarget",
		wantError: true,
		spec: map[string]interface{}{
			"endpoint": "http://www.example.com",
			"method":   "GE",
		},
	},
	"malformed spec 3": {
		kind:      "httptarget",
		wantError: true,
		spec: map[string]interface{}{
			"endpoint": "http://www.example.com",
			"method":   "GET",
			"foo":      "bar",
		},
	},
	"malformed spec 4": {
		kind:      "httptarget",
		wantError: true,
		spec: map[string]interface{}{
			"method": "GET",
		},
	},
	"malformed spec 5": {
		kind:      "httptarget",
		wantError: true,
		spec: map[string]interface{}{
			"endpoint": "http://www.example.com",
			"method":   "GET",
			"adapterOverrides": map[string]interface{}{
				"public": true,
				"foo":    "bar",
			},
		},
	},
	"ok spec 1": {
		kind:      "httptarget",
		wantError: false,
		spec: map[string]interface{}{
			"endpoint":   "http://www.example.com",
			"method":     "GET",
			"skipVerify": "true",
		},
	},
	"ok spec 2": {
		kind:      "httptarget",
		wantError: false,
		spec: map[string]interface{}{
			"endpoint":   "http://www.example.com",
			"method":     "GET",
			"skipVerify": true,
			"adapterOverrides": map[string]interface{}{
				"public": true,
			},
		},
	},
}

func TestCreateObject(t *testing.T) {
	for name, object := range specs {
		t.Run(name, func(t *testing.T) {
			_, err := CreateObject(test.CRD()[object.kind], Metadata{}, object.spec)
			if object.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateUnstructured(t *testing.T) {
	meta := Metadata{
		Name:      "foo",
		Namespace: "foo-namespace",
		Labels: map[string]string{
			"label-a": "label-a-value",
			"label-b": "label-b-value",
		},
		Annotations: map[string]string{
			"annotation-a": "annotation-a-value",
			"annotation-b": "annotation-b-value",
		},
	}
	status := map[string]interface{}{
		"key-a-status": "key-a-value",
		"key-b-status": "key-b-value",
	}

	for name, object := range specs {
		t.Run(name, func(t *testing.T) {
			u, err := CreateUnstructured(test.CRD()[object.kind], meta, object.spec, status)
			if object.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			actualMeta, set := u.Object["metadata"]
			assert.True(t, set, "unstructured object metadata not set")
			assert.Equal(t, "label-b-value", actualMeta.(map[string]interface{})["labels"].(map[string]interface{})["label-b"])
			actualStatus, set := u.Object["status"]
			assert.True(t, set, "unstructured object status not set")
			assert.Equal(t, "key-b-value", actualStatus.(map[string]interface{})["key-b-status"])
		})
	}
}

func TestExtractSecrets(t *testing.T) {
	objects := map[string]struct {
		kind           string
		spec           map[string]interface{}
		expectedSecret map[string]string
	}{
		"valid secrets": {
			kind: "awss3source",
			spec: map[string]interface{}{
				"auth": map[string]interface{}{
					"credentials": map[string]interface{}{
						"secretAccessKey": "foo",
						"accessKeyID":     "bar",
					},
				},
			},
			expectedSecret: map[string]string{
				"secretAccessKey": "Zm9v",
				"accessKeyID":     "YmFy",
			},
		},
		"extra secret key": {
			kind: "awss3source",
			spec: map[string]interface{}{
				"auth": map[string]interface{}{
					"credentials": map[string]interface{}{
						"secretAccessKey": "foo",
						"accessKeyID":     "bar",
						"bruh":            "bleh",
					},
				},
			},
			expectedSecret: map[string]string{
				"secretAccessKey": "Zm9v",
				"accessKeyID":     "YmFy",
			},
		},
		"wrong secret key": {
			kind: "awss3source",
			spec: map[string]interface{}{
				"auth": map[string]interface{}{
					"credentials": map[string]interface{}{
						"notSoSecretAccessKey": "foo",
					},
				},
			},
			expectedSecret: map[string]string{},
		},
	}

	for name, object := range objects {
		t.Run(name, func(t *testing.T) {
			secrets, err := ExtractSecrets("foo", test.CRD()[object.kind], object.spec)
			assert.NoError(t, err)
			assert.Equal(t, object.expectedSecret, secrets)
		})
	}
}

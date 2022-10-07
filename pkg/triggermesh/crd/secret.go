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

package crd

import (
	"fmt"

	"k8s.io/kube-openapi/pkg/validation/spec"
)

// secret objects in TriggerMesh are named
// secretKeyRef or valueFromSecret

func isSecretRef(s spec.Schema) (string, bool) {
	for k := range s.Properties {
		if k == "valueFromSecret" || k == "secretKeyRef" {
			return k, true
		}
	}
	return "", false
}

func ExtractSecrets(componentName string, schema Schema, spec map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range spec {
		if nestedSchema, ok := schema.schema.Properties[k]; ok {
			if key, ok := isSecretRef(nestedSchema); ok {
				secretName := fmt.Sprintf("%s-%s", componentName, k)
				if secretValue, ok := v.(string); ok {
					result[secretName] = map[string]interface{}{k: secretValue}
				} else {
					// error, we want a secret value here
				}
				spec[k] = map[string]interface{}{
					key: map[string]interface{}{
						"name": secretName,
						"key":  k,
					},
				}
			}
			if nestedSpec, ok := v.(map[string]interface{}); ok {
				result[k] = ExtractSecrets(componentName, Schema{nestedSchema}, nestedSpec)
			}
		}
	}
	return result
}

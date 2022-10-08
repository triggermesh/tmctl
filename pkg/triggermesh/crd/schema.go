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
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/kube-openapi/pkg/validation/spec"
	"k8s.io/kube-openapi/pkg/validation/strfmt"
	"k8s.io/kube-openapi/pkg/validation/validate"
)

type Schema struct {
	schema spec.Schema
}

func GetSchema(schema map[string]interface{}) (*Schema, error) {
	jsn, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}

	var sch spec.Schema
	if err := json.Unmarshal(jsn, &sch); err != nil {
		return nil, err
	}

	return &Schema{
		schema: sch,
	}, nil
}

func (s *Schema) Process(spec map[string]interface{}) (map[string]interface{}, error) {
	for k, v := range spec {
		schemaKey, exists := s.schema.Properties[k]
		// Consider support for unknown variables
		if !exists {
			return nil, fmt.Errorf("property %q does not exist, available values are: %s",
				k, propertyKeysAsString(s.schema.Properties))
		}

		// plain secret value only supported right now
		// if isSecretRef(schemaKey.Properties) {
		// if _, ok := v.(map[string]interface{}); !ok {
		// v = map[string]interface{}{
		// "value": v,
		// }
		// spec[k] = v
		// }
		// }

		switch value := v.(type) {
		case string:
			if schemaKey.Type[0] == "array" {
				spec[k] = strings.Split(value, ",")
			}
			if schemaKey.Type[0] == "object" {
				return nil, fmt.Errorf("%q is expected to be an object with properties: %s",
					k, propertyKeysAsString(schemaKey.Properties))
			}
		case map[string]interface{}:
			nestedSchema := Schema{
				schema: schemaKey,
			}
			nestedValue, err := nestedSchema.Process(value)
			if err != nil {
				return nil, err
			}
			spec[k] = nestedValue
		}
	}
	return spec, nil
}

func (s *Schema) Validate(spec map[string]interface{}) error {
	return validate.AgainstSchema(&s.schema, spec, strfmt.Default)
}

func propertyKeysAsString(s map[string]spec.Schema) string {
	var keys []string
	for k := range s {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

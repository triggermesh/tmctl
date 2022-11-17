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
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

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
		if !exists {
			return nil, fmt.Errorf("property %q does not exist, available values are: %s",
				k, propertyKeysAsString(s.schema.Properties))
		}

		if schemaKey.AdditionalProperties != nil {
			if schemaKey.AdditionalProperties.Schema == nil {
				return nil, fmt.Errorf("additional properties schema is missing in %q", k)
			}
			additionalParams, err := additionalPropertiesSpec(v, schemaKey.AdditionalProperties.Schema)
			if err != nil {
				return nil, fmt.Errorf("additional properties %q: %w", k, err)
			}
			spec[k] = additionalParams
			continue
		}

		switch value := v.(type) {
		case string:
			if len(schemaKey.Type) == 0 {
				break
			}
			switch schemaKey.Type[0] {
			case "array":
				if schemaKey.Items != nil && schemaKey.Items.Schema != nil {
					if len(schemaKey.Items.Schema.Type) == 1 && schemaKey.Items.Schema.Type[0] == "object" {
						// try to unmarshal input into array of objects
						var items []map[string]interface{}
						if err := yaml.Unmarshal([]byte(value), &items); err == nil {
							nestedArraySchema := Schema{
								schema: *schemaKey.Items.Schema,
							}
							var array []interface{}
							for _, nestedObject := range items {
								nestedArrayItem, err := nestedArraySchema.Process(nestedObject)
								if err == nil {
									array = append(array, nestedArrayItem)
								}
							}
							spec[k] = array
							continue
						}
					}
				}
				values := strings.Split(value, " ")
				if len(values) == 1 {
					values = strings.Split(value, ",")
				}
				spec[k] = values
			case "integer":
				integer, _ := strconv.Atoi(value) // let the CRD validation complain about the type
				spec[k] = int64(integer)
			case "boolean":
				spec[k] = (value == "true")
			case "object":
				var object map[string]interface{}
				if err := yaml.Unmarshal([]byte(value), &object); err == nil {
					spec[k] = object
					continue
				}
				return nil, fmt.Errorf("%q is expected to be an object with properties: %s",
					k, propertyKeysAsString(schemaKey.Properties))
			}
		case int:
			spec[k] = int64(value)
		case map[string]interface{}:
			nestedSchema := Schema{
				schema: schemaKey,
			}
			nestedValue, err := nestedSchema.Process(value)
			if err != nil {
				return nil, err
			}
			spec[k] = nestedValue
		case []interface{}:
			nestedSchema := Schema{
				schema: *schemaKey.Items.Schema,
			}
			var array []interface{}
			for _, v := range value {
				if nestedObject, ok := v.(map[string]interface{}); ok {
					nestedArrayItem, err := nestedSchema.Process(nestedObject)
					if err == nil {
						array = append(array, nestedArrayItem)
					}
				}
			}
			spec[k] = array
		default:
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

func additionalPropertiesSpec(value interface{}, spec *spec.Schema) (map[string]interface{}, error) {
	if result, ok := value.(map[string]interface{}); ok {
		return result, nil
	}
	result := make(map[string]interface{}, 0)

	if len(spec.AnyOf) != 0 {
		for _, nestedAPSchema := range spec.AnyOf {
			// try to convert to anyOf provided types
			if nestedSpec, err := additionalPropertiesSpec(value, &nestedAPSchema); err == nil {
				result = nestedSpec
			}
		}
	} else if typ := spec.Type[0]; typ != "" {
		input, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("input value type \"%T\" is not supported", value)
		}
		for _, pair := range strings.Split(input, ",") {
			var key, value string
			if kv := strings.Split(pair, ":"); len(kv) == 2 {
				key = kv[0]
				value = kv[1]
			} else if kv := strings.Split(pair, "="); len(kv) == 2 {
				key = kv[0]
				value = kv[1]
			} else {
				return nil, fmt.Errorf("cannot split %q into key-value pair", pair)
			}
			switch typ {
			case "string", "object":
				result[key] = value
			case "boolean":
				result[key] = (value == "true")
			case "integer":
				intValue, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("input value type conversion: %w", err)
				}
				result[key] = int64(intValue)
			default:
				return nil, fmt.Errorf("property type %q is not supported", typ)
			}
		}
	}
	return result, nil
}

type Property struct {
	Required    bool
	Typ         string
	Description string
}

func (s *Schema) GetAttributesCompletion(path ...string) (bool, map[string]Property) {
	result := make(map[string]Property, len(s.schema.Properties))
	schema := s.schema.Properties
	for _, key := range path {
		if key == "" {
			continue
		}
		nestedSchema, exists := schema[key]
		if !exists {
			return false, result
		}
		schema = nestedSchema.Properties
	}
	for name, prop := range schema {
		if _, secret := isSecretRef(prop); secret {
			prop.Type = []string{"string/secret"}
		}
		if name == "sink" ||
			name == "valueFromSecret" ||
			name == "secretKeyRef" {
			continue
		}
		property := Property{
			Typ:         strings.Join(prop.Type, ","),
			Description: strings.ReplaceAll(prop.Description, "\n", " "),
		}
		for _, req := range s.schema.Required {
			if req == name {
				property.Required = true
				break
			}
		}
		result[name] = property
	}
	return true, result
}

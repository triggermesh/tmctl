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

package transformation

import (
	"encoding/json"
)

const (
	jsonSchemaDraft04   = "http://json-schema.org/draft-04/schema#"
	jsonSchemaDraft07   = "http://json-schema.org/draft-07/schema#"
	jsonSchemaDraft2020 = "https://json-schema.org/draft/2020-12/schema"
)

type Schema struct {
	Draft       string `json:"$schema"`
	Ref         string `json:"$ref"`
	Definitions Index  `json:"definitions"`
	Defs        Index  `json:"$defs"`

	Title                string        `json:"title"`
	Description          string        `json:"description"`
	Name                 string        `json:"name"`
	Type                 interface{}   `json:"type"`
	Properties           Index         `json:"properties"`
	PatternProperties    Index         `json:"patternProperties"`
	AdditionalProperties interface{}   `json:"additionalProperties"`
	Items                *Schema       `json:"items"`
	Media                Media         `json:"media"`
	Required             []string      `json:"required"`
	Examples             []interface{} `json:"examples"`
}

type Index map[string]*Schema

type Media struct {
	BinaryEncoding string `json:"binaryEncoding"`
}

func schemaToData(data []byte) map[string]interface{} {
	// the list of sources for sample event:
	// - examples
	// - $ref to $definitions
	// - required property

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return map[string]interface{}{
			"error unmarshaling schema": err,
		}
	}

	if len(s.Examples) != 0 {
		return s.Examples[0].(map[string]interface{})
	}

	definitions := make(map[string]*Schema)

	switch s.Draft {
	case jsonSchemaDraft04, jsonSchemaDraft07:
		for name, definition := range s.Definitions {
			definitions["#/definitions/"+name] = definition
		}
	case jsonSchemaDraft2020:
		for name, definition := range s.Defs {
			definitions["#/$defs/"+name] = definition
		}
	default:
		return map[string]interface{}{
			"unsupported schema version": s.Draft,
		}
	}

	if s.Ref != "" {
		d, exists := definitions[s.Ref]
		if !exists {
			return map[string]interface{}{
				"definition is missing": s.Ref,
			}
		}
		return generateSample(*d, definitions)
	}

	if len(s.Required) != 0 && len(s.Properties) != 0 {
		result := make(map[string]interface{}, len(s.Required))
		for _, required := range s.Required {
			p, exists := s.Properties[required]
			if !exists {
				return map[string]interface{}{
					"property is missing": required,
				}
			}
			result[required] = generateSample(*p, definitions)
		}
		return result
	}
	return generateSample(s, definitions)
}

func generateSample(s Schema, definitions map[string]*Schema) map[string]interface{} {
	result := make(map[string]interface{})
	for name, property := range s.Properties {
		if property.Type == nil && property.Ref != "" {
			result[name] = generateSample(*definitions[property.Ref], definitions)
			continue
		}
		switch getType(property.Type) {
		case "object":
			if pattern, exists := property.PatternProperties[".*"]; exists {
				pp := Schema{
					Properties: map[string]*Schema{
						"sampleAttribute": pattern,
					},
				}
				result[name] = generateSample(pp, definitions)
				continue
			}

			result[name] = generateSample(*property, definitions)

			if b, ok := property.AdditionalProperties.(bool); ok && b {
				result[name].(map[string]interface{})["key"] = "value"
			} else if s, ok := property.AdditionalProperties.(Schema); ok {
				result[name].(map[string]interface{})["key"] = generateSample(s, definitions)
			}
		case "array":
			array := Schema{
				Properties: map[string]*Schema{
					"item": property.Items,
				},
			}
			ar := generateSample(array, definitions)
			result[name] = []interface{}{ar["item"]}
		default:
			if len(property.Examples) != 0 {
				result[name] = property.Examples[0]
			} else if property.Media.BinaryEncoding == "base64" {
				result[name] = "c2FtcGxlIHN0cmluZw=="
			} else {
				result[name] = value(getType(property.Type))
			}
			// if title := property.Title; title != "" {
			// result[name] = fmt.Sprintf("%s //%s", result[name], title)
			// } else if description := property.Description; description != "" {
			// result[name] = fmt.Sprintf("%s //%s", result[name], description)
			// }
		}
	}
	return result
}

func getType(t interface{}) string {
	switch s := t.(type) {
	case string:
		return s
	case []interface{}:
		if len(s) != 2 {
			return ""
		}
		if s[0] != "null" {
			return s[0].(string)
		} else {
			return s[1].(string)
		}
	}
	return ""
}

// "array", "boolean", "integer", "null", "number", "object", "string"

func value(t string) interface{} {
	switch t {
	// case "object":
	// case "array":
	// case "null":
	case "boolean":
		return true
	case "integer", "number":
		return 123
	case "string":
		return "sample string"
	}
	return nil
}

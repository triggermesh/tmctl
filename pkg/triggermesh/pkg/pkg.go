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

package pkg

import (
	"strings"
)

func ParseArgs(args map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(args))
	for key, value := range args {
		keys := strings.Split(key, ".")
		if len(keys) == 1 {
			result[key] = value
			continue
		}
		result = mergeMaps(result, nestedMap(keys, value))
	}
	return result
}

func nestedMap(key []string, value string) map[string]interface{} {
	if len(key) == 1 {
		return map[string]interface{}{key[0]: value}
	}
	return map[string]interface{}{key[0]: nestedMap(key[1:], value)}
}

func mergeMaps(src, dst map[string]interface{}) map[string]interface{} {
	for srcK, srcV := range src {
		if _, exists := dst[srcK]; exists {
			if _, ok := srcV.(map[string]interface{}); ok {
				if _, ok := dst[srcK].(map[string]interface{}); ok {
					dst[srcK] = mergeMaps(srcV.(map[string]interface{}), dst[srcK].(map[string]interface{}))
				} else {
					continue
				}
			} else {
				dst[srcK] = srcV
			}
		} else {
			dst[srcK] = srcV
		}
	}
	return dst
}

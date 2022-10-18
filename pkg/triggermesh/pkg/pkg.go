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
	"fmt"
	"strconv"
	"strings"
)

func ParseArgs(args []string) map[string]interface{} {
	var key string
	var value interface{}
	s := make(map[string]interface{}, len(args))
	for k := 0; k < len(args); k++ {
		v := strings.TrimLeft(args[k], "-")
		if kv := strings.Split(v, "="); len(kv) == 2 {
			key = kv[0]
			value = kv[1]
		} else {
			if len(args) > k+1 && !strings.HasPrefix(args[k+1], "--") {
				key = v
				value = args[k+1]
				k++
			} else {
				key = v
				value = true
			}
		}
		if str, ok := value.(string); ok {
			if valInt, err := strconv.Atoi(str); err == nil {
				value = int64(valInt)
			}
		}
		keys := strings.Split(key, ".")
		if len(keys) == 1 {
			s[key] = value
		} else {
			nestedMap := ParseArgs([]string{fmt.Sprintf("--%s=%v", strings.Join(keys[1:], "."), value)})
			if val, exists := s[keys[0]]; exists {
				if _, ok := val.(map[string]interface{}); ok {
					s[keys[0]] = mergeMaps(nestedMap, val.(map[string]interface{}))
				} else {
					s[keys[0]] = nestedMap
				}
			} else {
				s[keys[0]] = nestedMap
			}
		}
	}
	return s
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

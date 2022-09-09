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

package triggermesh

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	"github.com/triggermesh/tmcli/pkg/kubernetes"
)

func CreateBroker(name string, manifestFile string) (*kubernetes.Object, bool, error) {
	// create config folder
	if err := os.MkdirAll(path.Dir(manifestFile), os.ModePerm); err != nil {
		return nil, false, fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		if _, err := os.Create(manifestFile); err != nil {
			return nil, false, fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return nil, false, fmt.Errorf("manifest file access: %w", err)
	}

	broker := kubernetes.Object{
		APIVersion: "eventing.triggermesh.io/v1alpha1",
		Kind:       "Broker",
		Metadata: kubernetes.Metadata{
			Name: name,
			Labels: map[string]string{
				"triggermesh.io/context": name,
			},
		},
		Spec: map[string]interface{}{"storage": viper.GetString("storage")},
	}

	manifest := kubernetes.NewManifest(manifestFile)
	dirty, err := manifest.Add(broker)
	if err != nil {
		return nil, false, fmt.Errorf("manifest update: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return nil, false, fmt.Errorf("manifest write operation: %w", err)
		}
	}
	return &broker, dirty, nil
}

func CreateSource(kind string, broker string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
	manifest := kubernetes.NewManifest(manifestFile)
	err := manifest.Read()
	if err != nil {
		return nil, false, fmt.Errorf("unable to read the manifest: %w", err)
	}

	spec := argsToMap(args)
	spec["sink"] = map[string]interface{}{
		"uri": fmt.Sprintf("http://%s.user-namespace.svc.cluster.local", broker),
	}

	source, err := kubernetes.CreateObject(strings.ToLower(kind)+"source", broker+"-source", broker, crdFile, spec)
	if err != nil {
		return nil, false, fmt.Errorf("creating object: %w", err)
	}

	dirty, err := manifest.Add(source)
	if err != nil {
		return nil, false, fmt.Errorf("manifest update: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return nil, false, fmt.Errorf("manifest write operation: %w", err)
		}
	}
	return &source, dirty, nil
}

func CreateTarget(kind string, broker string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
	manifest := kubernetes.NewManifest(manifestFile)
	err := manifest.Read()
	if err != nil {
		return nil, false, fmt.Errorf("unable to read the manifest: %w", err)
	}

	spec := argsToMap(args)

	target, err := kubernetes.CreateObject(strings.ToLower(kind)+"target", broker+"-target", broker, crdFile, spec)
	if err != nil {
		return nil, false, fmt.Errorf("spec processing: %w", err)
	}

	dirty, err := manifest.Add(target)
	if err != nil {
		return nil, false, fmt.Errorf("manifest update: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return nil, false, fmt.Errorf("manifest write operation: %w", err)
		}
	}
	return &target, dirty, nil
}

func argsToMap(args []string) map[string]interface{} {
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
				value = valInt
			}
		}
		keys := strings.Split(key, ".")
		if len(keys) == 1 {
			s[key] = value
		} else {
			nestedMap := argsToMap([]string{fmt.Sprintf("--%s=%v", strings.Join(keys[1:], "."), value)})
			// Convert this logic to recursive function
			if val, exists := s[keys[0]]; exists {
				if vval, ok := val.(map[string]interface{}); ok {
					for k, v := range nestedMap {
						vval[k] = v
					}
					s[keys[0]] = vval
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

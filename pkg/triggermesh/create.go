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
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
)

func CreateTrigger(manifestFile, broker, eventType, target string) (*kubernetes.Object, error) {
	manifest := kubernetes.NewManifest(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil, fmt.Errorf("unable to read the manifest: %w", err)
	}
	trigger := tmbroker.CreateTriggerObject(broker+"-"+eventType, eventType, target, broker)
	dirty, err := manifest.Add(trigger)
	if err != nil {
		return nil, fmt.Errorf("manifest update: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return nil, fmt.Errorf("manifest write operation: %w", err)
		}
	}
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	jsn, err := client.Inspect(context.Background(), broker)
	if err != nil {
		return nil, fmt.Errorf("broker inspect: %w", err)
	}
	if len(jsn.Mounts) != 1 {
		return nil, fmt.Errorf("broker config volume not found")
	}
	config, err := tmbroker.ReadConfigFile(jsn.Mounts[0].Source)
	if err != nil {
		return nil, fmt.Errorf("broker config read: %w", err)
	}
	newConfig, dirty := tmbroker.AppendTriggerToConfig(trigger, config)
	if !dirty {
		return &trigger, nil
	}
	out, err := yaml.Marshal(newConfig)
	if err != nil {
		return nil, fmt.Errorf("broker config marshal: %w", err)
	}
	if err := os.WriteFile(jsn.Mounts[0].Source, out, os.ModePerm); err != nil {
		return nil, fmt.Errorf("broker config write: %w", err)
	}
	return &trigger, nil
}

func CreateSource(kind, broker string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
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

func CreateTarget(kind, broker string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
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

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
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	eventingbroker "github.com/triggermesh/brokers/pkg/config/broker"
	"gopkg.in/yaml.v3"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func GetObject(name string, config *config.Config, manifest *manifest.Manifest, crds map[string]crd.CRD) (triggermesh.Component, error) {
	for _, object := range manifest.Objects {
		if object.Metadata.Name != name {
			continue
		}
		broker, set := object.Metadata.Labels["triggermesh.io/context"]
		if !set {
			return nil, fmt.Errorf("context label not set")
		}
		crd := crds[strings.ToLower(object.Kind)]
		switch object.APIVersion {
		case "sources.triggermesh.io/v1alpha1":
			status := make(map[string]interface{}, 0)
			externalResources, set := object.Metadata.Annotations[triggermesh.ExternalResourcesAnnotation]
			if set {
				for _, resource := range strings.Split(externalResources, ",") {
					entry := strings.Split(resource, "=")
					if len(entry) == 2 {
						status[entry[0]] = entry[1]
					}
				}
			}
			return source.New(object.Metadata.Name, object.Kind, broker, config.Triggermesh.ComponentsVersion, crd, object.Spec, status), nil
		case "targets.triggermesh.io/v1alpha1":
			return target.New(object.Metadata.Name, object.Kind, broker, config.Triggermesh.ComponentsVersion, crd, object.Spec), nil
		case "flow.triggermesh.io/v1alpha1":
			return transformation.New(object.Metadata.Name, object.Kind, broker, config.Triggermesh.ComponentsVersion, crd, object.Spec), nil
		case "eventing.triggermesh.io/v1alpha1":
			switch object.Kind {
			case "RedisBroker":
				return tmbroker.New(object.Metadata.Name, config.Triggermesh.Broker)
			case "Trigger":
				brokerConfigPath := filepath.Dir(manifest.Path)
				baseConfigPath := filepath.Dir(brokerConfigPath)
				targetName, filter, err := parseTriggerSpec(object.Spec)
				if err != nil {
					return nil, fmt.Errorf("trigger spec: %w", err)
				}
				target, err := GetObject(targetName, config, manifest, crds)
				if err != nil {
					return nil, fmt.Errorf("trigger's target ref: %w", err)
				}
				trigger, err := tmbroker.NewTrigger(object.Metadata.Name, broker, baseConfigPath, nil, filter)
				if err != nil {
					return nil, fmt.Errorf("trigger object: %w", err)
				}
				trigger.(*tmbroker.Trigger).SetTarget(target)
				return trigger, nil
			}
		case "serving.knative.dev/v1":
			role, set := object.Metadata.Labels["triggermesh.io/role"]
			if !set {
				break
			}
			// TODO: Fix this
			params := make(map[string]string)
			container := object.Spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})[0]
			image := container.(map[string]interface{})["image"].(string)
			env := container.(map[string]interface{})["env"]
			if env != nil {
				for _, v := range env.([]interface{}) {
					val, ok := v.(map[string]interface{})
					if !ok {
						continue
					}
					name, ok := val["name"]
					if !ok {
						continue
					}
					value, ok := val["value"]
					if !ok {
						continue
					}
					params[name.(string)] = value.(string)
				}
			}
			return service.New(name, image, broker, service.Role(role), params), nil
		case "v1":
			if object.Kind == "Secret" {
				return secret.New(object.Metadata.Name, broker, object.Data), nil
			}
		}
	}
	return nil, nil
}

func ProcessSecrets(p triggermesh.Parent, manifest *manifest.Manifest) ([]triggermesh.Component, map[string]string, error) {
	secrets := readSecrets(p, manifest)
	plainSecretsEnv, err := decodeSecrets(secrets)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding secret: %w", err)
	}
	return secrets, plainSecretsEnv, nil
}

func readSecrets(p triggermesh.Parent, manifest *manifest.Manifest) []triggermesh.Component {
	secrets, err := p.GetChildren()
	if err != nil {
		// Secrets are already extracted, read manifest
		for _, object := range manifest.Objects {
			if object.Kind == "Secret" && object.Metadata.Name == p.(triggermesh.Component).GetName()+"-secret" {
				data := make(map[string]string)
				for key, value := range object.Data {
					data[key] = value
				}
				secrets = append(secrets, secret.New(object.Metadata.Name, "", data))
			}
		}
	}
	return secrets
}

func decodeSecrets(secrets []triggermesh.Component) (map[string]string, error) {
	result := make(map[string]string)
	for _, secret := range secrets {
		for k, v := range secret.GetSpec() {
			plainValue, err := base64.StdEncoding.DecodeString(v.(string))
			if err != nil {
				return nil, fmt.Errorf("decoding secret value: %w", err)
			}
			result[k] = string(plainValue)
		}
	}
	return result, nil
}

func parseTriggerSpec(spec map[string]interface{}) (string, *eventingbroker.Filter, error) {
	filter, err := yaml.Marshal(spec["filters"])
	if err != nil {
		return "", nil, err
	}
	var f []eventingbroker.Filter
	if err := yaml.Unmarshal(filter, &f); err != nil {
		return "", nil, err
	}
	if l := len(f); l != 1 {
		return "", nil, fmt.Errorf("expected 1 filter condition, got %d", l)
	}
	target, err := yaml.Marshal(spec["target"])
	if err != nil {
		return "", nil, err
	}
	var t duckv1.Destination
	if err := yaml.Unmarshal(target, &t); err != nil {
		return "", nil, err
	}
	return t.Ref.Name, &f[0], nil
}

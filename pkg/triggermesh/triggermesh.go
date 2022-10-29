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
	"encoding/base64"
	"fmt"

	"github.com/triggermesh/tmctl/pkg/manifest"
	corev1 "k8s.io/api/core/v1"
)

func secretToEnv(secret Component) ([]corev1.EnvVar, error) {
	if kind := secret.GetKind(); kind != "Secret" {
		return []corev1.EnvVar{}, fmt.Errorf("%q is not convertable to env variables", kind)
	}
	var result []corev1.EnvVar
	for k, v := range secret.GetSpec() {
		plainValue, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return []corev1.EnvVar{}, fmt.Errorf("decoding secret value: %w", err)
		}
		result = append(result, corev1.EnvVar{Name: k, Value: string(plainValue)})
	}
	return result, nil
}

func ProcessSecrets(ctx context.Context, p Parent, manifestPath string) (map[string]string, bool, error) {
	secrets, err := p.GetChildren()
	if err != nil {
		// return nil, false, fmt.Errorf("component nested objects: %w", err)
		// Secrets are already extracted, read manifest
		secrets, err := readSecrets(p.(Component).GetName(), manifestPath)
		return secrets, false, err
	}
	secretsChanged := false
	secretEnv := make(map[string]string)
	for _, s := range secrets {
		dirty, err := s.Add(manifestPath)
		if err != nil {
			return nil, false, fmt.Errorf("write nested object: %w", err)
		}
		if dirty {
			secretsChanged = true
		}
		env, err := secretToEnv(s)
		if err != nil {
			return nil, false, fmt.Errorf("secret env: %w", err)
		}
		for _, v := range env {
			secretEnv[v.Name] = v.Value
		}
	}
	return secretEnv, secretsChanged, nil
}

func readSecrets(parent string, manifestPath string) (map[string]string, error) {
	manifest := manifest.New(manifestPath)
	if err := manifest.Read(); err != nil {
		return nil, err
	}
	secrets := make(map[string]string)
	for _, object := range manifest.Objects {
		if object.Kind == "Secret" && object.Metadata.Name == parent {
			for key, value := range object.Data {
				plainValue, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					return nil, fmt.Errorf("decoding secret: %w", err)
				}
				secrets[key] = string(plainValue)
			}
		}
	}
	return secrets, nil
}

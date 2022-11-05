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

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
)

func GetObject(name, crdFile, version string, manifest *manifest.Manifest) (triggermesh.Component, error) {
	for _, object := range manifest.Objects {
		if object.Metadata.Name == name {
			switch object.APIVersion {
			case "sources.triggermesh.io/v1alpha1":
				return source.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			case "targets.triggermesh.io/v1alpha1":
				return target.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			case "flow.triggermesh.io/v1alpha1":
				return transformation.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			}
		}
	}
	return nil, nil
}

func ProcessSecrets(p triggermesh.Parent, manifest *manifest.Manifest) (map[string]string, bool, error) {
	secrets := readSecrets(p, manifest)
	secretsChanged := false
	for _, secret := range secrets {
		dirty, err := manifest.Add(secret)
		if err != nil {
			return nil, false, fmt.Errorf("unable to update manifest: %w", err)
		}
		if dirty {
			secretsChanged = true
		}
	}
	secretEnv, err := decodeSecrets(secrets)
	if err != nil {
		return nil, false, fmt.Errorf("decoding secret: %w", err)
	}
	return secretEnv, secretsChanged, nil
}

func readSecrets(p triggermesh.Parent, manifest *manifest.Manifest) []triggermesh.Component {
	secrets, err := p.GetChildren()
	if err != nil {
		// Secrets are already extracted, read manifest
		for _, object := range manifest.Objects {
			if object.Kind == "Secret" && object.Metadata.Name == p.(triggermesh.Component).GetName() {
				for key, value := range object.Data {
					secret := secret.New(object.Metadata.Name, "", map[string]string{
						key: string(value),
					})
					secrets = append(secrets, secret)
				}
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

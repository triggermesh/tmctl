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

	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler"
)

func WriteObject(object Component, manifestFile string) (bool, error) {
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return false, fmt.Errorf("reading manifest: %w", err)
	}
	k8sObject, err := object.AsK8sObject()
	if err != nil {
		return false, fmt.Errorf("creating object: %w", err)
	}
	if dirty := manifest.Add(k8sObject); !dirty {
		return false, nil
	}
	return true, manifest.Write()
}

func RemoveObject(name, kind string, manifestFile string) error {
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}
	manifest.Remove(name, kind)
	return manifest.Write()
}

func Start(ctx context.Context, object Runnable, restart bool, additionalEnv map[string]string) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	container, err := object.AsContainer(additionalEnv)
	if err != nil {
		return nil, fmt.Errorf("creating container object: %w", err)
	}
	if err := container.PullImage(ctx, client, object.GetImage()); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}
	var containerIsRunning bool
	existingContainer, _ := container.LookupHostConfig(ctx, client)
	if existingContainer != nil {
		if object.GetImage() != existingContainer.Image() {
			restart = true
		}
		if err := existingContainer.Connect(ctx); err == nil {
			containerIsRunning = true
		}
	}
	if restart {
		container.Remove(ctx, client)
	} else if containerIsRunning {
		return existingContainer, nil
	}
	container, err = container.Start(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("adapter initialization: %w", err)
	}
	return container, nil
}

func Info(ctx context.Context, object Runnable) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := object.AsContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

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

func ProcessSecrets(ctx context.Context, p Parent, manifestFile string) (map[string]string, bool, error) {
	secrets, err := p.GetChildren()
	if err != nil {
		// return nil, false, fmt.Errorf("component nested objects: %w", err)
		// Secrets are already extracted, read manifest
		secrets, err := readSecrets(p.(Component).GetName(), manifestFile)
		return secrets, false, err
	}
	secretsChanged := false
	secretEnv := make(map[string]string)
	for _, s := range secrets {
		dirty, err := WriteObject(s, manifestFile)
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

func readSecrets(parent, manifestFile string) (map[string]string, error) {
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
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

func InitializeServicesAndStatus(ctx context.Context, object Component, secrets map[string]string) error {
	u, err := object.AsUnstructured()
	if err != nil {
		return err
	}
	status, err := reconciler.InitializeAndGetStatus(ctx, u, secrets)
	if err != nil {
		return err
	}
	object.SetStatus(status)
	return nil
}

func FinalizeExternalServices(ctx context.Context, object Component, secrets map[string]string) error {
	u, err := object.AsUnstructured()
	if err != nil {
		return err
	}
	return reconciler.Finalize(ctx, u, secrets)
}

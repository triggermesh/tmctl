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

	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/manifest"
)

func WriteObject(ctx context.Context, object Component, manifestFile string) (bool, error) {
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

func Start(ctx context.Context, object Runnable, restart bool, opts ...docker.ContainerOption) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	container, err := object.AsContainer(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating container object: %w", err)
	}
	if err := container.PullImage(ctx, client, object.GetImage()); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}
	if restart {
		// skip errors
		container.Remove(ctx, client)
	}
	if existingContainer, err := container.LookupHostConfig(ctx, client); err == nil {
		if err := existingContainer.Connect(ctx); err == nil {
			// container is up
			return existingContainer, nil
		}
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
	container, err := object.AsContainer()
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

func ToEnv(secret Component) ([]corev1.EnvVar, error) {
	if kind := secret.GetKind(); kind != "Secret" {
		return []corev1.EnvVar{}, fmt.Errorf("%q is not convertable to env variables", kind)
	}
	var result []corev1.EnvVar
	for k, v := range secret.GetSpec() {
		result = append(result, corev1.EnvVar{Name: k, Value: v.(string)})
	}
	return result, nil
}

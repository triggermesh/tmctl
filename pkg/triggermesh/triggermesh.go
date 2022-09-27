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
	dirty, err := manifest.Add(*k8sObject)
	if err != nil {
		return false, fmt.Errorf("adding to manifest: %w", err)
	}
	if dirty {
		if err := manifest.Write(); err != nil {
			return false, fmt.Errorf("writing manifest: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func Start(ctx context.Context, object Runnable, restart bool) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	container, err := object.AsContainer()
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

func Stop(ctx context.Context, object Runnable) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("creating docker client: %w", err)
	}
	container, err := object.AsContainer()
	if err != nil {
		return fmt.Errorf("creating container object: %w", err)
	}
	return container.Remove(ctx, client)
}

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

	"github.com/triggermesh/tmctl/pkg/docker"
)

// Component is the common interface for all TriggerMesh components.
type Component interface {
	Add(manifest string) (bool, error)
	Delete(manifest string) error

	GetName() string
	GetKind() string
	GetSpec() map[string]interface{}
}

// Runnable is the interface for components that can run as Docker containers.
type Runnable interface {
	Start(context.Context, map[string]string, bool) (*docker.Container, error)
	Stop(context.Context) error
	Info(context.Context) (*docker.Container, error)
}

// Producer is implemeted by all components that produce events.
type Producer interface {
	SetEventAttributes(map[string]string) error

	GetEventTypes() ([]string, error)
	GetEventSource() (string, error)
}

// Consumer is implemented by all components that consume events.
type Consumer interface {
	ConsumedEventTypes() ([]string, error)
	GetPort(context.Context) (string, error)
}

// Parent is the interface of the components that produce additional components.
type Parent interface {
	GetChildren() ([]Component, error)
}

// Reconcilable is implemented by the components that depend on external services
// and require additional initialization and finalization logic.
type Reconcilable interface {
	Initialize(context.Context, map[string]string) (map[string]interface{}, error)
	Finalize(context.Context, map[string]string) error
	UpdateStatus(map[string]interface{})
}

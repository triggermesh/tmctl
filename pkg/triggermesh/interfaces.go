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
	"github.com/triggermesh/tmctl/pkg/kubernetes"
)

// Component is the common interface for all TriggerMesh components.
type Component interface {
	AsK8sObject() (kubernetes.Object, error)
	AsDockerComposeObject() (*DockerComposeService, error)

	GetName() string
	GetKind() string
	GetAPIVersion() string
	GetSpec() map[string]interface{}
}

// Runnable is the interface for components that can run as Docker containers.
type Runnable interface {
	Start(ctx context.Context, additionalEnv map[string]string, restart bool) (*docker.Container, error)
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
	GetExternalResources() map[string]interface{}
}

// TODO (move to another place?)
type DockerCompose struct {
	Services Services `json:"services"`
}

type Services map[string]DockerComposeService

type DockerComposeService struct {
	Command     string                `json:"command"`
	Image       string                `json:"image"`
	Ports       []string              `json:"ports"`
	Environment []string              `json:"environment"`
	Volumes     []DockerComposeVolume `json:"volumes"`
}

type DockerComposeVolume struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

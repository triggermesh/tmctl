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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
)

type Component interface {
	AsUnstructured() (unstructured.Unstructured, error)
	AsK8sObject() (kubernetes.Object, error)

	GetName() string
	GetKind() string
	GetSpec() map[string]interface{}
}

type Runnable interface {
	AsContainer(additionalEnvs map[string]string) (*docker.Container, error)

	GetImage() string
}

type Producer interface {
	SetEventType(string) error
	GetEventTypes() ([]string, error)
}

type Consumer interface {
	ConsumedEventTypes() ([]string, error)
	GetPort(context.Context) (string, error)
}

type Parent interface {
	GetChildren() ([]Component, error)
}

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

package service

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
)

const (
	APIVersion = "serving.knative.dev/v1"
	Kind       = "Service"
)

var (
	Producer Role = "producer"
	Consumer Role = "consumer"

	_ triggermesh.Component = (*Service)(nil)
	// _ triggermesh.Consumer  = (*Service)(nil)
	// _ triggermesh.Producer  = (*Service)(nil)
	_ triggermesh.Runnable = (*Service)(nil)
)

type Role string

type Service struct {
	Name   string
	Broker string
	Image  string

	role   Role
	params map[string]string
}

func (s *Service) asUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion(APIVersion)
	u.SetKind(Kind)
	u.SetName(s.Name)
	u.SetNamespace(triggermesh.Namespace)
	u.SetLabels(map[string]string{"context": s.Broker})
	return u, unstructured.SetNestedField(u.Object, kserviceSpec(s.Image, s.params), "spec")
}

func (s *Service) AsK8sObject() (kubernetes.Object, error) {
	manifestParams := make(map[string]string, len(s.params))
	for k, v := range s.params {
		manifestParams[k] = v
	}
	manifestParams["K_SINK"] = fmt.Sprintf("http://%s-rb-broker:8080", s.Broker)
	return kubernetes.Object{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata: kubernetes.Metadata{
			Name:      s.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": s.Broker,
			},
		},
		Spec: kserviceSpec(s.Image, manifestParams),
	}, nil
}

func (s *Service) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	u, err := s.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	for k, v := range additionalEnvs {
		s.params[k] = v
	}
	co, ho, err := adapter.RuntimeParams(u, s.Image, s.params)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   s.Name,
		Image:                  s.Image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (s *Service) GetKind() string {
	return Kind
}

func (s *Service) GetName() string {
	return s.Name
}

func (s *Service) GetAPIVersion() string {
	return APIVersion
}

func (s *Service) GetSpec() map[string]interface{} {
	return nil
}

func (s *Service) GetPort(ctx context.Context) (string, error) {
	container, err := s.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort(), nil
}

func (s *Service) GetEventTypes() ([]string, error) {
	return []string{}, nil
}

func (s *Service) GetEventSource() (string, error) {
	return "", nil
}

func (s *Service) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := s.asContainer(additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.Start(ctx, client, restart)
}

func (s *Service) Stop(ctx context.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	container, err := s.asContainer(nil)
	if err != nil {
		return fmt.Errorf("container object: %w", err)
	}
	return container.Remove(ctx, client)
}

func (s *Service) Info(ctx context.Context) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := s.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

func New(name, image, broker string, role Role, params map[string]string) triggermesh.Component {
	if name == "" {
		name = fmt.Sprintf("%s-service", broker)
	}
	return &Service{
		Name:   name,
		Broker: broker,
		Image:  image,
		params: params,
		role:   role,
	}
}

func paramsToEnv(params map[string]string) []interface{} {
	var env []interface{}
	for k, v := range params {
		env = append(env, map[string]interface{}{
			"name":  strings.ToUpper(k),
			"value": v,
		})
	}
	return env
}

func kserviceSpec(image string, params map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"template": map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"image": image,
						"name":  "user-container",
						"env":   paramsToEnv(params),
					},
				},
			},
		},
	}
}

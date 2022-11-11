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

package source

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

var (
	_ triggermesh.Reconcilable = (*Source)(nil)
	_ triggermesh.Component    = (*Source)(nil)
	_ triggermesh.Producer     = (*Source)(nil)
	_ triggermesh.Runnable     = (*Source)(nil)
	_ triggermesh.Parent       = (*Source)(nil)
)

type Source struct {
	Name    string
	CRDFile string

	Broker  string
	Kind    string
	Version string

	spec   map[string]interface{}
	status map[string]interface{}
}

func (s *Source) asUnstructured() (unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(s.GetKind(), s.GetName(), triggermesh.Namespace, s.Broker, s.CRDFile, s.spec, s.status)
}

func (s *Source) AsK8sObject() (kubernetes.Object, error) {
	spec := make(map[string]interface{}, len(s.spec))
	for k, v := range s.spec {
		spec[k] = v
	}
	spec["sink"] = map[string]interface{}{
		"ref": map[string]interface{}{
			"name":       s.Broker,
			"kind":       tmbroker.BrokerKind,
			"apiVersion": tmbroker.APIVersion,
		},
	}
	return kubernetes.CreateObject(s.GetKind(), s.GetName(), triggermesh.Namespace, s.Broker, s.CRDFile, spec)
}

func (s *Source) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := s.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image := adapter.Image(o, s.Version)
	co, ho, err := adapter.RuntimeParams(o, image, additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   s.GetName(),
		Image:                  image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (s *Source) GetName() string {
	return s.Name
}

func (s *Source) GetKind() string {
	return s.Kind
}

func (s *Source) GetAPIVersion() string {
	o, err := s.AsK8sObject()
	if err != nil {
		return ""
	}
	return o.APIVersion
}

func (s *Source) GetSpec() map[string]interface{} {
	return s.spec
}

func (s *Source) GetEventTypes() ([]string, error) {
	// try GetEventTypes method first
	o, err := s.asUnstructured()
	if err != nil {
		return []string{}, fmt.Errorf("unstructured object: %w", err)
	}
	eventAttributes, err := adapter.EventAttributes(o)
	if err != nil {
		return []string{}, fmt.Errorf("source event attributes: %w", err)
	}
	if len(eventAttributes.ProducedEventTypes) == 0 {
		return []string{}, fmt.Errorf("%q does not expose event type attributes", s.Kind)
	}
	return eventAttributes.ProducedEventTypes, nil
}

func (s *Source) GetEventSource() (string, error) {
	// Second, get event attributes from the core object methods
	o, err := s.asUnstructured()
	if err != nil {
		return "", fmt.Errorf("unstructured object: %w", err)
	}
	eventAttributes, err := adapter.EventAttributes(o)
	if err != nil {
		return "", fmt.Errorf("source event attributes: %w", err)
	}
	if eventAttributes.ProducedEventSource == "" {
		return "", fmt.Errorf("%q does not expose event source attribute", s.Kind)
	}
	return eventAttributes.ProducedEventSource, nil
}

func (s *Source) GetChildren() ([]triggermesh.Component, error) {
	secrets, err := kubernetes.ExtractSecrets(s.Name, s.Kind, s.CRDFile, s.spec)
	if err != nil {
		return nil, fmt.Errorf("extracting secrets: %w", err)
	}
	if len(secrets) == 0 {
		return nil, nil
	}
	return []triggermesh.Component{secret.New(strings.ToLower(s.Name), s.Broker, secrets)}, nil
}

func (s *Source) SetEventAttributes(map[string]string) error {
	return fmt.Errorf("event source does not support context attributes override")
}

func (s *Source) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
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

func (s *Source) Stop(ctx context.Context) error {
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

func (s *Source) Info(ctx context.Context) (*docker.Container, error) {
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

func (s *Source) Initialize(ctx context.Context, secrets map[string]string) (map[string]interface{}, error) {
	u, err := s.asUnstructured()
	if err != nil {
		return nil, err
	}
	return adapter.InitializeAndGetStatus(ctx, u, secrets)
}

func (s *Source) Finalize(ctx context.Context, secrets map[string]string) error {
	u, err := s.asUnstructured()
	if err != nil {
		return err
	}
	return adapter.Finalize(ctx, u, secrets)
}

func (s *Source) UpdateStatus(status map[string]interface{}) {
	s.status = status
}

func New(name, crdFile, kind, broker, version string, params interface{}) triggermesh.Component {
	var spec map[string]interface{}
	switch p := params.(type) {
	case map[string]string:
		// cli args
		spec = pkg.ParseArgs(p)
	case map[string]interface{}:
		// spec map
		spec = p
	}

	// kind initially can be awss3, webhook, etc.
	k := strings.ToLower(kind)
	if !strings.Contains(k, "source") {
		k = fmt.Sprintf("%ssource", kind)
	}

	if name == "" {
		name = fmt.Sprintf("%s-%s", broker, k)
	}

	return &Source{
		Name:    name,
		CRDFile: crdFile,
		Broker:  broker,
		Kind:    k,
		Version: version,
		spec:    spec,
	}
}

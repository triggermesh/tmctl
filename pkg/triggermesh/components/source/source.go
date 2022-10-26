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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	_ triggermesh.Component = (*Source)(nil)
	_ triggermesh.Producer  = (*Source)(nil)
	_ triggermesh.Runnable  = (*Source)(nil)
	_ triggermesh.Parent    = (*Source)(nil)
)

type Source struct {
	Name    string
	CRDFile string

	Broker  string
	Kind    string
	Version string

	image string
	spec  map[string]interface{}
}

func (s *Source) AsUnstructured() (unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.spec)
}

func (s *Source) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.CreateObject(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.spec)
}

func (s *Source) AsContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := s.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	s.image = adapter.Image(o, s.Version)
	co, ho, err := adapter.RuntimeParams(o, s.image, additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   s.GetName(),
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

func (s *Source) GetImage() string {
	return s.image
}

func (s *Source) GetSpec() map[string]interface{} {
	return s.spec
}

func (s *Source) GetEventTypes() ([]string, error) {
	if ceOverrides, set := s.spec["ceOverrides"]; set {
		// interface type is verified by CRD validation
		if extensions, set := ceOverrides.(map[string]interface{})["extensions"]; set {
			if typeOverride, exists := extensions.(map[string]string)["type"]; exists {
				return []string{typeOverride}, nil
			}
		}
	}
	if et, exists := s.spec["eventType"]; exists {
		return []string{et.(string)}, nil
	}
	sourceCRD, err := crd.GetResourceCRD(s.Kind, s.CRDFile)
	if err != nil {
		return []string{}, fmt.Errorf("source CRD: %w", err)
	}
	var et crd.EventTypes
	if err := json.Unmarshal([]byte(sourceCRD.Metadata.Annotations.EventTypes), &et); err != nil {
		return []string{}, fmt.Errorf("event types: %w", err)
	}
	var result []string
	for _, v := range et {
		result = append(result, v.Type)
	}
	return result, nil
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

func (s *Source) SetEventType(string) error {
	return fmt.Errorf("event source does not support context attributes override")
}

func New(name, crdFile, kind, broker, version string, params interface{}) *Source {
	var spec map[string]interface{}
	switch p := params.(type) {
	case []string:
		// args slice
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

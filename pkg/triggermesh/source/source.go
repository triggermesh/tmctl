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
	"fmt"
	"strings"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/adapter"
	"github.com/triggermesh/tmcli/pkg/triggermesh/pkg"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ triggermesh.Component = (*Source)(nil)

type Source struct {
	Name string

	ManifestFile string
	CRDFile      string

	Broker  string
	Kind    string
	Version string

	image string
	spec  map[string]interface{}
}

func (s *Source) AsUnstructured() (*unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.spec)
}

func (s *Source) AsK8sObject() (*kubernetes.Object, error) {
	return kubernetes.CreateObject(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.spec)
}

func (s *Source) AsContainer() (*docker.Container, error) {
	o, err := s.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image, err := adapter.Image(o, s.Version)
	if err != nil {
		return nil, fmt.Errorf("adapter image: %w", err)
	}
	s.image = image
	co, ho, err := adapter.RuntimeParams(o, image)
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

func NewSource(manifest, crd string, kind, broker, version string, params interface{}) *Source {
	var spec map[string]interface{}
	switch p := params.(type) {
	case []string:
		// args slice
		spec = pkg.ParseArgs(p)
	case map[string]interface{}:
		// spec map
		spec = p
	default:
	}

	// kind initially can be awss3, webhook, etc.
	k := strings.ToLower(kind)
	if !strings.Contains(k, "source") {
		k = fmt.Sprintf("%ssource", kind)
	}
	return &Source{
		Name:         fmt.Sprintf("%s-%s", broker, k),
		ManifestFile: manifest,
		CRDFile:      crd,
		Broker:       broker,
		Kind:         k,
		Version:      version,
		spec:         spec,
	}
}

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

package transformation

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/adapter"
)

var _ triggermesh.Component = (*Transformation)(nil)

type Transformation struct {
	Name    string
	CRDFile string
	Broker  string
	Version string

	image string
	spec  map[string]interface{}
}

func (t *Transformation) AsUnstructured() (*unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(t.GetKind(), t.GetName(), t.Broker, t.CRDFile, t.spec)
}

func (t *Transformation) AsK8sObject() (*kubernetes.Object, error) {
	return kubernetes.CreateObject(t.GetKind(), t.GetName(), t.Broker, t.CRDFile, t.spec)
}

func (t *Transformation) AsContainer() (*docker.Container, error) {
	o, err := t.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	t.image = adapter.Image(o, t.Version)
	co, ho, err := adapter.RuntimeParams(o, t.image, "")
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   t.GetName(),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (t *Transformation) GetName() string {
	return t.Name
}

func (t *Transformation) GetKind() string {
	return "transformation"
}

func (t *Transformation) GetImage() string {
	return t.image
}

func New(crdFile, kind, broker, version string, spec map[string]interface{}) *Transformation {
	return &Transformation{
		Name:    strings.ToLower(broker + "-transformation"),
		CRDFile: crdFile,
		Broker:  broker,
		Version: version,

		spec: spec,
	}
}

// tmcli create broker foo
// tmcli create source awss3
// ?[ tmcli create transformation --source awss3 ]
// tmcli create target --source awss3 [--eventType]
// tmcli create transformation --source awss3

// tmcli create trigger bar --eventType bar.message.sample

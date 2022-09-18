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

package target

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

var _ triggermesh.Component = (*Target)(nil)

type Target struct {
	Name string

	ManifestFile string
	CRDFile      string

	Broker  string
	Version string
	Kind    string

	image string
	spec  map[string]interface{}
}

func (t *Target) AsUnstructured() (*unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(t.GetKind(), t.GetName(), t.Broker, t.CRDFile, t.spec)
}

func (t *Target) AsK8sObject() (*kubernetes.Object, error) {
	return kubernetes.CreateObject(t.GetKind(), t.GetName(), t.Broker, t.CRDFile, t.spec)
}

func (t *Target) AsContainer() (*docker.Container, error) {
	o, err := t.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image, err := adapter.Image(o, t.Version)
	if err != nil {
		return nil, fmt.Errorf("adapter image: %w", err)
	}
	t.image = image
	co, ho, err := adapter.RuntimeParams(o, image)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   t.GetName(),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (t *Target) GetName() string {
	return t.Name
}

func (t *Target) GetKind() string {
	return t.Kind
}

func (t *Target) GetImage() string {
	return t.image
}

// func (t *Target) GetSpec() (map[string]interface{}, error) {
// 	o, err := t.AsK8sObject()
// 	if err != nil {
// 		return nil, fmt.Errorf("creating object: %w", err)
// 	}
// 	return o.Spec, nil
// }

func NewTarget(manifest, crd string, kind, broker, version string, params interface{}) *Target {
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
	if !strings.Contains(k, "target") {
		k = fmt.Sprintf("%starget", kind)
	}
	return &Target{
		Name:         fmt.Sprintf("%s-%s", broker, k),
		ManifestFile: manifest,
		CRDFile:      crd,
		Broker:       broker,
		Kind:         k,
		Version:      version,
		spec:         spec,
	}
}

// func Create(kind, broker string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
// 	manifest := manifest.New(manifestFile)
// 	err := manifest.Read()
// 	if err != nil {
// 		return nil, false, fmt.Errorf("unable to read the manifest: %w", err)
// 	}
// 	spec := pkg.ParseArgs(args)
// 	t, err := kubernetes.CreateObject(strings.ToLower(kind)+"target", broker+"-target", broker, crdFile, spec)
// 	if err != nil {
// 		return nil, false, fmt.Errorf("spec processing: %w", err)
// 	}

// 	dirty, err := manifest.Add(*t)
// 	if err != nil {
// 		return nil, false, fmt.Errorf("manifest update: %w", err)
// 	}
// 	if dirty {
// 		if err := manifest.Write(); err != nil {
// 			return nil, false, fmt.Errorf("manifest write operation: %w", err)
// 		}
// 	}
// 	return t, dirty, nil
// }

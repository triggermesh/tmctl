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

package kubernetes

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/triggermesh/tmcli/pkg/oas"
	"github.com/triggermesh/tmcli/pkg/triggermesh/crd"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	labelKey = "triggermesh.io/context"
)

type Manifest struct {
	Path    string
	Objects []Object
}

type Object struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   Metadata               `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec"`
}

type Metadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}

func NewManifest(path string) *Manifest {
	return &Manifest{
		Path: path,
	}
}

func (m *Manifest) Read() error {
	o, err := parseYAML(m.Path)
	if err != nil {
		return err
	}
	m.Objects = o
	return nil
}

func (m *Manifest) Write() error {
	f, err := os.OpenFile(m.Path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, object := range m.Objects {
		body, err := yaml.Marshal(object)
		if err != nil {
			return err
		}
		if _, err := f.WriteString("---\n"); err != nil {
			return err
		}
		if _, err := f.Write(body); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manifest) Add(object Object) (bool, error) {
	for i, o := range m.Objects {
		if matchObjects(object, o) {
			if !reflect.DeepEqual(o, object) {
				m.Objects[i] = object
				return true, nil
			}
			return false, nil
		}
	}
	m.Objects = append(m.Objects, object)
	return true, nil
}

func CreateObject(resource, name, broker, crdFile string, spec map[string]interface{}) (Object, error) {
	crdObject, err := crd.GetResource(resource, crdFile)
	if err != nil {
		return Object{}, fmt.Errorf("CRD schema not found: %w", err)
	}
	var version string
	for _, v := range crdObject.Spec.Versions {
		if v.Served {
			version = v.Name
			schema, err := oas.GetSchema(v.Schema.OpenAPIV3Schema.Properties.Spec)
			if err != nil {
				return Object{}, fmt.Errorf("CRD schema: %w", err)
			}
			if spec, err = schema.Process(spec); err != nil {
				return Object{}, fmt.Errorf("spec processing: %w", err)
			}
			if err := schema.Validate(spec); err != nil {
				return Object{}, fmt.Errorf("CR validation: %w", err)
			}
			break
		}
	}
	return Object{
		APIVersion: fmt.Sprintf("%s/%s", crdObject.Spec.Group, version),
		Kind:       crdObject.Spec.Names.Kind,
		Metadata: Metadata{
			Name: name,
			Labels: map[string]string{
				labelKey: broker,
			},
		},
		Spec: spec,
	}, nil
}

func parseYAML(path string) ([]Object, error) {
	f, err := os.Open(path)
	if err != nil {
		return []Object{}, err
	}

	fstat, err := f.Stat()
	if err != nil {
		return []Object{}, err
	}

	if !fstat.IsDir() {
		return readFile(f)
	}

	entries, err := f.ReadDir(-1)
	if err != nil {
		return []Object{}, err
	}

	var result []Object
	for _, d := range entries {
		if d.IsDir() {
			// Skip directories
			continue
		}
		objects, err := parseYAML(path + "/" + d.Name())
		if err != nil {
			return []Object{}, err
		}
		result = append(result, objects...)
	}
	return result, nil
}

func readFile(file io.Reader) ([]Object, error) {
	var result []Object
	decoder := yaml.NewDecoder(file)
	for {
		o := new(Object)
		err := decoder.Decode(&o)
		if o == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return []Object{}, err
		}
		result = append(result, *o)
	}
	return result, nil
}

func matchObjects(a, b Object) bool {
	return (a.APIVersion == b.APIVersion) &&
		(a.Kind == b.Kind) &&
		(a.Metadata.Name == b.Metadata.Name)
}

func (o *Object) ToUnstructured() (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}

	u.SetAPIVersion(o.APIVersion)
	u.SetKind(o.Kind)
	u.SetName(o.Metadata.Name)

	return u, unstructured.SetNestedField(u.Object, o.Spec, "spec")
}

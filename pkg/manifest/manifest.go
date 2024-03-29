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

package manifest

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"

	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"

	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
)

// Manifest is the representation of the YAML file with the TriggerMesh components.
type Manifest struct {
	mut     sync.Mutex
	Path    string
	Objects []kubernetes.Object
}

func New(path string) *Manifest {
	return &Manifest{
		Path: path,
	}
}

func (m *Manifest) Read() error {
	m.mut.Lock()
	defer m.mut.Unlock()
	o, err := parseYAML(m.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("manifest does not exist, please create the broker")
		}
		return err
	}
	m.Objects = o
	return nil
}

func (m *Manifest) Write() error {
	var output []byte
	for _, object := range m.Objects {
		body, err := kyaml.Marshal(object)
		if err != nil {
			return err
		}
		body = append([]byte("---\n"), body...)
		output = append(output, body...)
	}
	if err := os.WriteFile(m.Path, output, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func (m *Manifest) Add(object triggermesh.Component) (bool, error) {
	m.mut.Lock()
	defer m.mut.Unlock()
	k8sObject, err := object.AsK8sObject()
	if err != nil {
		return false, fmt.Errorf("creating k8s object: %w", err)
	}
	k8sObject.Metadata.Namespace = "" // local manifest should not set namespace
	for i, o := range m.Objects {
		if matchObjects(k8sObject, o) {
			if reflect.DeepEqual(k8sObject, o) {
				return false, nil
			}
			m.Objects[i] = k8sObject
			return true, m.Write()
		}
	}
	m.Objects = append(m.Objects, k8sObject)
	return true, m.Write()
}

func (m *Manifest) Remove(name, kind string) error {
	m.mut.Lock()
	defer m.mut.Unlock()
	objects := make([]kubernetes.Object, 0, len(m.Objects))
	for _, o := range m.Objects {
		if o.Metadata.Name == name && o.Kind == kind {
			continue
		}
		objects = append(objects, o)
	}
	m.Objects = objects
	return m.Write()
}

func parseYAML(path string) ([]kubernetes.Object, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []kubernetes.Object
	decoder := yaml.NewDecoder(file)
	for {
		o := new(kubernetes.Object)
		err := decoder.Decode(&o)
		if o == nil {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return []kubernetes.Object{}, err
		}
		result = append(result, *o)
	}
	return result, nil
}

func matchObjects(a, b kubernetes.Object) bool {
	return (a.APIVersion == b.APIVersion) &&
		(a.Kind == b.Kind) &&
		(a.Metadata.Name == b.Metadata.Name)
}

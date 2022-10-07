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

package secret

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
)

var _ triggermesh.Component = (*Secret)(nil)

type Secret struct {
	Name    string
	Context string

	data map[string]interface{}
}

func (s *Secret) AsUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion("v1")
	u.SetKind(s.GetKind())
	u.SetName(s.Name)
	u.SetLabels(map[string]string{"context": s.Context})
	if err := unstructured.SetNestedField(u.Object, "Opaque", "type"); err != nil {
		return unstructured.Unstructured{}, err
	}
	return u, unstructured.SetNestedMap(u.Object, s.data, "data")
}

func (s *Secret) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.Object{
		APIVersion: "v1",
		Kind:       s.GetKind(),
		Metadata: kubernetes.Metadata{
			Name: s.Name,
			Labels: map[string]string{
				"triggermesh.io/context": s.Context,
			},
		},
		Type: "Opaque",
		Data: s.data,
	}, nil
}

func (s *Secret) GetName() string {
	return s.Name
}

func (s *Secret) GetKind() string {
	return "Secret"
}

func (s *Secret) GetSpec() map[string]interface{} {
	return s.data
}

func New(name, context string, data map[string]interface{}) *Secret {
	return &Secret{
		Name:    name,
		Context: context,

		data: data,
	}
}

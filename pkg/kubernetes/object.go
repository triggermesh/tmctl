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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

const deploymentLabel = "app.kubernetes.io/name"

type Object struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   Metadata               `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty" yaml:"spec,omitempty"`

	// for Secrets
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
	Type string            `json:"type,omitempty" yaml:"type,omitempty"`
}

type Metadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

func CreateObject(crd crd.CRD, metadata Metadata, spec map[string]interface{}) (Object, error) {
	schema, version, err := getObjectCRD(crd)
	if err != nil {
		return Object{}, fmt.Errorf("object schema: %w", err)
	}
	if spec, err = schema.Process(spec); err != nil {
		return Object{}, fmt.Errorf("spec processing: %w", err)
	}
	if err := schema.Validate(spec); err != nil {
		return Object{}, fmt.Errorf("CR validation: %w", err)
	}
	return Object{
		APIVersion: fmt.Sprintf("%s/%s", crd.Spec.Group, version),
		Kind:       crd.Spec.Names.Kind,
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}

func CreateUnstructured(crd crd.CRD, metadata Metadata, spec, status map[string]interface{}) (unstructured.Unstructured, error) {
	schema, version, err := getObjectCRD(crd)
	if err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("object schema: %w", err)
	}
	if spec, err = schema.Process(spec); err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("spec processing: %w", err)
	}
	if err := schema.Validate(spec); err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("CR validation: %w", err)
	}
	u := unstructured.Unstructured{}
	u.SetAPIVersion(fmt.Sprintf("%s/%s", crd.Spec.Group, version))
	u.SetKind(crd.Spec.Names.Kind)
	u.SetName(metadata.Name)
	u.SetNamespace(metadata.Namespace)
	u.SetLabels(metadata.Labels)
	u.SetAnnotations(metadata.Annotations)
	for k, v := range spec {
		switch val := v.(type) {
		case []string:
			if err := unstructured.SetNestedStringSlice(u.Object, val, "spec", k); err != nil {
				return unstructured.Unstructured{}, fmt.Errorf("object key %q: %w", k, err)
			}
		case map[string]interface{}:
			if err := unstructured.SetNestedMap(u.Object, val, "spec", k); err != nil {
				return unstructured.Unstructured{}, fmt.Errorf("object key %q: %w", k, err)
			}
		default:
			if err := unstructured.SetNestedField(u.Object, val, "spec", k); err != nil {
				return unstructured.Unstructured{}, fmt.Errorf("object key %q: %w", k, err)
			}
		}
	}
	if err := unstructured.SetNestedMap(u.Object, status, "status"); err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("object status: %w", err)
	}
	return u, nil
}

func CreateDeployment(name, image string, envs []corev1.EnvVar) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					deploymentLabel: name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						deploymentLabel: name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "adapter",
							Image: image,
							Env:   envs,
							Ports: []corev1.ContainerPort{
								{
									Name:          "adapter",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
}

func CreateService(name string) interface{} {
	return corev1.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     8080,
					TargetPort: intstr.IntOrString{
						StrVal: "adapter",
					},
				},
			},
			Selector: map[string]string{
				deploymentLabel: name,
			},
		},
	}
}

func getObjectCRD(crdObject crd.CRD) (*crd.Schema, string, error) {
	for _, v := range crdObject.Spec.Versions {
		if v.Served {
			schema, err := crd.GetSchema(v.Schema.OpenAPIV3Schema.Properties.Spec)
			if err != nil {
				return nil, "", fmt.Errorf("CRD schema: %w", err)
			}
			return schema, v.Name, nil
		}
	}
	return nil, "", fmt.Errorf("CRD schema not found")
}

// ExtractSecrets looks up resource schema, extracts secret objects
// if passed spec contains secret data and returns a map with base64 encoded values.
// It does not validate the spec against the CRD.
func ExtractSecrets(componentName string, c crd.CRD, spec map[string]interface{}) (map[string]string, error) {
	schema, _, err := getObjectCRD(c)
	if err != nil {
		return nil, err
	}
	return crd.ExtractSecrets(componentName, *schema, spec)
}

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

package env

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	extensionsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/extensions/reconciler/function"
)

var functionRuntimes = map[string]string{
	"python": "gcr.io/triggermesh/knative-lambda-python37",
	"node":   "gcr.io/triggermesh/knative-lambda-node10",
	"ruby":   "gcr.io/triggermesh/knative-lambda-ruby25",
}

func extensions(object *unstructured.Unstructured) ([]corev1.EnvVar, error) {
	switch object.GetKind() {
	// Extensions API group
	case "Function":
		var o *extensionsv1alpha1.Function
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
			return nil, err
		}
		return function.MakeAppEnv(o), nil
	}
	return nil, fmt.Errorf("kind %q is not supported", object.GetKind())
}

func FunctionRuntimeImage(object *unstructured.Unstructured) (string, error) {
	var o *extensionsv1alpha1.Function
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
		panic(err)
	}
	image, exists := functionRuntimes[o.Spec.Runtime]
	if !exists {
		return "", fmt.Errorf("container image for %q runtime not found", o.Spec.Runtime)
	}
	return image, nil
}

func FunctionCode(object *unstructured.Unstructured) string {
	var o *extensionsv1alpha1.Function
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
		panic(err)
	}
	return o.Spec.Code
}

func FunctionEntrypoint(object *unstructured.Unstructured) string {
	var o *extensionsv1alpha1.Function
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
		panic(err)
	}
	return o.Spec.Entrypoint
}

func FunctionFileExtension(object *unstructured.Unstructured) string {
	var o *extensionsv1alpha1.Function
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &o); err != nil {
		panic(err)
	}
	switch strings.ToLower(o.Spec.Runtime) {
	case "python":
		return "py"
	case "node", "js":
		return "js"
	case "ruby":
		return "rb"
	case "sh":
		return "sh"
	}
	return "txt"
}

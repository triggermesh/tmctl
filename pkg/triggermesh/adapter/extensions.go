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

package adapter

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	extensionsv1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/extensions/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/extensions/reconciler/function"
)

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

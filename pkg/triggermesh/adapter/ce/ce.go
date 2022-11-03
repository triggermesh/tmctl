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

package ce

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type EventAttributes struct {
	ProducedEventSource string
	ProducedEventTypes  []string
	AcceptedEventTypes  []string
}

func Attributes(o unstructured.Unstructured) (EventAttributes, error) {
	switch o.GetAPIVersion() {
	case "sources.triggermesh.io/v1alpha1":
		return sources(o)
	case "targets.triggermesh.io/v1alpha1":
		return targets(o)
		// case "flow.triggermesh.io/v1alpha1":
		// return flow(o)
		// case "extensions.triggermesh.io/v1alpha1":
		// return extensions(o)
		// case "routing.triggermesh.io/v1alpha1":
		// return routing(o)
	}
	return EventAttributes{}, fmt.Errorf("API group %q is not supported", o.GetKind())
}

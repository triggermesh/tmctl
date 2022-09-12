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

package function

import (
	"fmt"
	"strings"

	"github.com/triggermesh/tmcli/pkg/kubernetes"
)

var functionRuntimes = map[string]string{
	"python": "gcr.io/triggermesh/knative-lambda-python37",
	"node":   "gcr.io/triggermesh/knative-lambda-node10",
	"ruby":   "gcr.io/triggermesh/knative-lambda-ruby25",
}

func ImageName(object *kubernetes.Object) (string, error) {
	runtime := object.Spec["runtime"].(string)
	image, exists := functionRuntimes[runtime]
	if !exists {
		return "", fmt.Errorf("container image for %q runtime not found", runtime)
	}
	return image, nil
}

func Code(object *kubernetes.Object) string {
	return object.Spec["code"].(string)
}

func Entrypoint(object *kubernetes.Object) string {
	return object.Spec["entrypoint"].(string)
}

func FileExtension(object *kubernetes.Object) string {
	switch strings.ToLower(object.Spec["runtime"].(string)) {
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

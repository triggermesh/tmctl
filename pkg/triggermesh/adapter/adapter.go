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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/docker"
)

const (
	registry    = "gcr.io/triggermesh"
	adapterPort = "8080/tcp"
)

func Image(object unstructured.Unstructured, version string) string {
	return fmt.Sprintf("%s/%s-adapter:%s", registry, strings.ToLower(object.GetKind()), version)
}

func RuntimeParams(object unstructured.Unstructured, image string) ([]docker.ContainerOption, []docker.HostOption, error) {
	co := []docker.ContainerOption{
		docker.WithImage(image),
		docker.WithPort(adapterPort),
		docker.WithErrorLoggingLevel(),
	}
	ho := []docker.HostOption{
		docker.WithHostPortBinding(adapterPort),
		docker.WithExtraHost(),
	}

	kenv, err := buildEnv(object)
	if err != nil {
		return nil, nil, fmt.Errorf("adapter environment: %w", err)
	}
	sinkURI, set, err := unstructured.NestedString(object.Object, "spec", "sink", "uri")
	if err != nil {
		return nil, nil, fmt.Errorf("sink URI type: %w", err)
	}
	if set {
		kenv = append(kenv, corev1.EnvVar{Name: "K_SINK", Value: sinkURI})
	}
	co = append(co, docker.WithEnv(envsToString(kenv)))

	return co, ho, nil
}

func envsToString(envs []corev1.EnvVar) []string {
	var result []string
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}

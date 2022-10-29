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

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/env"
)

const (
	registry    = "gcr.io/triggermesh"
	adapterPort = "8080/tcp"
)

func Image(object unstructured.Unstructured, version string) string {
	// components with custom images
	switch object.GetKind() {
	case "AWSS3Source":
		return fmt.Sprintf("%s/awssqssource-adapter:%s", registry, version)
	}
	return fmt.Sprintf("%s/%s-adapter:%s", registry, strings.ToLower(object.GetKind()), version)
}

func RuntimeParams(object unstructured.Unstructured, image string, additionalEnvs map[string]string) ([]docker.ContainerOption, []docker.HostOption, error) {
	co := []docker.ContainerOption{
		docker.WithImage(image),
		docker.WithPort(adapterPort),
		docker.WithErrorLoggingLevel(),
	}
	ho := []docker.HostOption{
		docker.WithHostPortBinding(adapterPort),
		docker.WithExtraHost(),
	}

	if object.GetKind() == "Broker" {
		return co, ho, nil
	}

	kenv, err := env.Build(object)
	if err != nil {
		return nil, nil, fmt.Errorf("adapter environment: %w", err)
	}

	for i, v := range kenv {
		if v.ValueFrom != nil && additionalEnvs != nil {
			if secret, ok := additionalEnvs[v.ValueFrom.SecretKeyRef.Key]; ok {
				kenv[i] = corev1.EnvVar{Name: v.Name, Value: string(secret)}
				delete(additionalEnvs, v.ValueFrom.SecretKeyRef.Key)
			}
		}
	}
	for k, v := range additionalEnvs {
		kenv = append(kenv, corev1.EnvVar{Name: k, Value: v})
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

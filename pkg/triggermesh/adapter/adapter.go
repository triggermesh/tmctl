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
	"os"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/docker"
)

const (
	registry    = "gcr.io/triggermesh"
	brokerImage = "docker.io/tzununbekov/memory-broker"
	adapterPort = "8080/tcp"
)

func Image(object *unstructured.Unstructured, version string) (string, error) {
	var image string
	switch object.GetKind() {
	case "Broker":
		image = brokerImage
	// case "Function":
	// 	fimage, err := function.ImageName(runtime)
	// 	if err != nil {
	// 		return "", fmt.Errorf("cannot parse function image: %w", err)
	// 	}
	// 	image = fmt.Sprintf("%s:%s", fimage, version)
	default:
		image = fmt.Sprintf("%s/%s-adapter:%s", registry, strings.ToLower(object.GetKind()), version)
	}
	return image, nil
}

func RuntimeParams(object *unstructured.Unstructured, image string) ([]docker.ContainerOption, []docker.HostOption, error) {
	co := []docker.ContainerOption{
		docker.WithImage(image),
		docker.WithPort(adapterPort),
	}
	ho := []docker.HostOption{
		docker.WithHostPortBinding(adapterPort),
		docker.WithExtraHost(),
	}

	switch object.GetKind() {
	case "Broker":
		// get rid of viper here
		configFile := path.Join("/Users/tzununbekov/.triggermesh/cli", object.GetName(), "broker.conf")
		if err := writeFile(configFile, []byte{}); err != nil {
			return nil, nil, fmt.Errorf("writing configuration: %w", err)
		}
		bind := fmt.Sprintf("%s:/etc/triggermesh/broker.conf", configFile)
		ho = append(ho, docker.WithVolumeBind(bind))
	// case "Function":
	// 	configFile := path.Join("/Users/tzununbekov/.triggermesh/cli", viper.GetString("context"), object.Metadata.Name)
	// 	if err := writeFile(configFile, []byte(function.Code(object))); err != nil {
	// 		return nil, nil, fmt.Errorf("writing function: %w", err)
	// 	}
	// 	bind := fmt.Sprintf("%s:/opt/source.%s", configFile, function.FileExtension(object))
	// 	ho = append(ho, docker.WithVolumeBind(bind))
	// 	co = append(co, docker.WithEntrypoint("/opt/aws-custom-runtime"))
	// 	// yikes
	// 	fallthrough
	default:
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
	}

	return co, ho, nil
}

func writeFile(filePath string, data []byte) error {
	if err := os.MkdirAll(path.Dir(filePath), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(filePath, data, os.ModePerm)
}

func envsToString(envs []corev1.EnvVar) []string {
	var result []string
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}

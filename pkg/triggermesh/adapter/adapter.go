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
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/ce"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/env"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/reconciler"
)

const (
	registry    = "gcr.io/triggermesh"
	adapterPort = "8080/tcp"
)

func Image(object unstructured.Unstructured, version string) string {
	// components with custom images
	switch object.GetKind() {
	case "AWSS3Source",
		"AWSEventBridgeSource":
		return fmt.Sprintf("%s/awssqssource-adapter:%s", registry, version)
	case "AzureServiceBusTopicSource",
		"AzureServiceBusQueueSource":
		return fmt.Sprintf("%s/azureservicebussource-adapter:%s", registry, version)
	case "AzureBlobStorageSource":
		return fmt.Sprintf("%s/azureeventhubsource-adapter:%s", registry, version)
	case "GoogleCloudAuditLogsSource",
		"GoogleCloudStorageSource",
		"GoogleCloudSourceRepositoriesSource":
		return fmt.Sprintf("%s/googlecloudpubsubsource-adapter:%s", registry, version)
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

	finalEnv := []corev1.EnvVar{}

	if object.GetKind() != "RedisBroker" &&
		object.GetKind() != "Service" {
		adapterEnv, err := env.Build(object)
		if err != nil {
			return nil, nil, fmt.Errorf("adapter environment: %w", err)
		}
		for _, v := range adapterEnv {
			if v.ValueFrom != nil && additionalEnvs != nil {
				if secret, ok := additionalEnvs[v.ValueFrom.SecretKeyRef.Key]; ok {
					finalEnv = append(finalEnv, corev1.EnvVar{Name: v.Name, Value: string(secret)})
					delete(additionalEnvs, v.ValueFrom.SecretKeyRef.Key)
				}
			} else {
				finalEnv = append(finalEnv, v)
			}
		}
	}
	for k, v := range additionalEnvs {
		finalEnv = append(finalEnv, corev1.EnvVar{Name: k, Value: v})
	}

	sinkURI, set, err := unstructured.NestedString(object.Object, "spec", "sink", "uri")
	if err != nil {
		return nil, nil, fmt.Errorf("sink URI type: %w", err)
	}
	if set {
		finalEnv = append(finalEnv, corev1.EnvVar{Name: "K_SINK", Value: sinkURI})
	}
	co = append(co, docker.WithEnv(envsToString(finalEnv)))
	return co, ho, nil
}

func envsToString(envs []corev1.EnvVar) []string {
	var result []string
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}

func InitializeAndGetStatus(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) (map[string]interface{}, error) {
	return reconciler.InitializeAndGetStatus(ctx, object, secrets)
}

func Finalize(ctx context.Context, object unstructured.Unstructured, secrets map[string]string) error {
	return reconciler.Finalize(ctx, object, secrets)
}

func EventAttributes(object unstructured.Unstructured) (ce.EventAttributes, error) {
	attributes, err := ce.Attributes(object)
	if err != nil {
		return ce.EventAttributes{}, err
	}
	if attributes.ProducedEventSource == "*" || attributes.ProducedEventSource == "-" {
		attributes.ProducedEventSource = ""
	}
	if len(attributes.AcceptedEventTypes) == 1 &&
		(attributes.AcceptedEventTypes[0] == "*" || attributes.AcceptedEventTypes[0] == "-") {
		attributes.AcceptedEventTypes = []string{}
	}
	if len(attributes.ProducedEventTypes) == 1 &&
		(attributes.ProducedEventTypes[0] == "*" || attributes.ProducedEventTypes[0] == "-") {
		attributes.ProducedEventTypes = []string{}
	}
	return attributes, nil
}

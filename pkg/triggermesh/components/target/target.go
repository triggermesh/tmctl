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

package target

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/digitalocean/godo"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter/env"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/secret"
	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

var (
	_ triggermesh.Component  = (*Target)(nil)
	_ triggermesh.Consumer   = (*Target)(nil)
	_ triggermesh.Runnable   = (*Target)(nil)
	_ triggermesh.Parent     = (*Target)(nil)
	_ triggermesh.Exportable = (*Target)(nil)
)

type Target struct {
	Name    string
	CRDFile string

	Broker  string
	Version string
	Kind    string

	spec map[string]interface{}
}

func (t *Target) asUnstructured() (unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(t.GetKind(), t.CRDFile, t.getMeta(), t.spec, nil)
}

func (t *Target) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.CreateObject(t.GetKind(), t.CRDFile, t.getMeta(), t.spec)
}

func (t *Target) getMeta() kubernetes.Metadata {
	return kubernetes.Metadata{
		Name:      t.GetName(),
		Namespace: triggermesh.Namespace,
		Labels: map[string]string{
			triggermesh.ContextLabel: t.Broker,
		},
	}
}

func (t *Target) AsDockerComposeObject(additionalEnvs map[string]string) (interface{}, error) {
	o, err := t.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image := adapter.Image(o, t.Version)

	adapterEnv, err := env.Build(o)
	if err != nil {
		return nil, fmt.Errorf("adapter environment: %w", err)
	}

	envs := []corev1.EnvVar{}
	for _, v := range adapterEnv {
		if v.ValueFrom != nil && additionalEnvs != nil {
			if secret, ok := additionalEnvs[v.ValueFrom.SecretKeyRef.Key]; ok {
				envs = append(envs, corev1.EnvVar{Name: v.Name, Value: string(secret)})
				delete(additionalEnvs, v.ValueFrom.SecretKeyRef.Key)
			}
		} else {
			envs = append(envs, v)
		}
	}

	for k, v := range additionalEnvs {
		envs = append(envs, corev1.EnvVar{Name: k, Value: v})
	}

	return &docker.ComposeService{
		ContainerName: t.Name,
		Image:         image,
		Environment:   pkg.EnvsToString(envs),
		Ports:         []string{strconv.Itoa(pkg.OpenPort()) + ":8080"},
	}, nil
}

func (t *Target) AsDigitalOceanObject(additionalEnvs map[string]string) (interface{}, error) {
	o, err := t.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}

	adapterEnv, err := env.Build(o)
	if err != nil {
		return nil, fmt.Errorf("adapter environment: %w", err)
	}

	envs := []*godo.AppVariableDefinition{}
	for _, v := range adapterEnv {
		if v.ValueFrom != nil && additionalEnvs != nil {
			if secret, ok := additionalEnvs[v.ValueFrom.SecretKeyRef.Key]; ok {
				envs = append(envs, &godo.AppVariableDefinition{Key: v.Name, Value: string(secret)})
				delete(additionalEnvs, v.ValueFrom.SecretKeyRef.Key)
			}
		} else {
			envs = append(envs, &godo.AppVariableDefinition{Key: v.Name, Value: v.Value})
		}
	}

	for k, v := range additionalEnvs {
		envs = append(envs, &godo.AppVariableDefinition{Key: k, Value: v})
	}

	// Get the image and tag
	imageURI := strings.Split(adapter.Image(o, ""), "/")
	adapterName := strings.TrimRight(imageURI[len(imageURI)-1], ":")

	return godo.AppServiceSpec{
		Name: t.Name,
		Image: &godo.ImageSourceSpec{
			DeployOnPush: &godo.ImageSourceSpecDeployOnPush{
				Enabled: true,
			},
			RegistryType: godo.ImageSourceSpecRegistryType_DOCR,
			Repository:   adapterName,
			Tag:          t.Version,
		},
		InternalPorts:    []int64{8080},
		Envs:             envs,
		InstanceCount:    1,
		InstanceSizeSlug: "professional-xs",
	}, nil
}

func (t *Target) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := t.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image := adapter.Image(o, t.Version)
	co, ho, err := adapter.RuntimeParams(o, image, additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   t.GetName(),
		Image:                  image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (t *Target) GetName() string {
	return t.Name
}

func (t *Target) GetKind() string {
	return t.Kind
}

func (t *Target) GetAPIVersion() string {
	o, err := t.AsK8sObject()
	if err != nil {
		return ""
	}
	return o.APIVersion
}

func (t *Target) GetSpec() map[string]interface{} {
	return t.spec
}

func (t *Target) GetPort(ctx context.Context) (string, error) {
	container, err := t.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort(), nil
}

func (t *Target) GetChildren() ([]triggermesh.Component, error) {
	secrets, err := kubernetes.ExtractSecrets(t.Name, t.Kind, t.CRDFile, t.spec)
	if err != nil {
		return nil, fmt.Errorf("extracting secrets: %w", err)
	}
	if len(secrets) == 0 {
		return nil, nil
	}
	return []triggermesh.Component{secret.New(strings.ToLower(t.Name)+"-secret", t.Broker, secrets)}, nil
}

func (t *Target) ConsumedEventTypes() ([]string, error) {
	o, err := t.asUnstructured()
	if err != nil {
		return nil, err
	}
	eventAttributes, err := adapter.EventAttributes(o)
	if err != nil {
		return []string{}, err
	}
	return eventAttributes.AcceptedEventTypes, nil
}

func (t *Target) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := t.asContainer(additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.Start(ctx, client, restart)
}

func (t *Target) Stop(ctx context.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	container, err := t.asContainer(nil)
	if err != nil {
		return fmt.Errorf("container object: %w", err)
	}
	return container.Remove(ctx, client)
}

func (t *Target) Info(ctx context.Context) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := t.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

func (t *Target) Logs(ctx context.Context, since time.Time, follow bool) (io.ReadCloser, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := t.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	if _, err := container.LookupHostConfig(ctx, client); err != nil {
		return nil, fmt.Errorf("container config: %w", err)
	}
	return container.Logs(ctx, client, since, follow)
}

func New(name, crdFile, kind, broker, version string, params interface{}) triggermesh.Component {
	var spec map[string]interface{}
	switch p := params.(type) {
	case map[string]string:
		// cli args
		spec = pkg.ParseArgs(p)
	case map[string]interface{}:
		// spec map
		spec = p
	default:
	}

	// kind initially can be awss3, webhook, etc.
	k := strings.ToLower(kind)
	if !strings.HasSuffix(k, "target") {
		k = fmt.Sprintf("%starget", kind)
	}

	if name == "" {
		name = fmt.Sprintf("%s-%s", broker, k)
	}

	return &Target{
		Name:    name,
		CRDFile: crdFile,
		Broker:  broker,
		Kind:    k,
		Version: version,
		spec:    spec,
	}
}

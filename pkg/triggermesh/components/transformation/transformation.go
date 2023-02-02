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

package transformation

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
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

var (
	_ triggermesh.Component  = (*Transformation)(nil)
	_ triggermesh.Consumer   = (*Transformation)(nil)
	_ triggermesh.Producer   = (*Transformation)(nil)
	_ triggermesh.Runnable   = (*Transformation)(nil)
	_ triggermesh.Exportable = (*Transformation)(nil)
)

type Transformation struct {
	Name    string
	CRD     crd.CRD
	Broker  string
	Version string

	spec map[string]interface{}
}

func (t *Transformation) asUnstructured() (unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(t.CRD, t.getMeta(), t.spec, nil)
}

func (t *Transformation) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.CreateObject(t.CRD, t.getMeta(), t.spec)
}

func (t *Transformation) getMeta() kubernetes.Metadata {
	return kubernetes.Metadata{
		Name:      t.GetName(),
		Namespace: triggermesh.Namespace,
		Labels: map[string]string{
			triggermesh.ContextLabel: t.Broker,
		},
	}
}

func (t *Transformation) AsDockerComposeObject(additionalEnvs map[string]string) (interface{}, error) {
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

func (t *Transformation) AsDigitalOceanObject(additionalEnvs map[string]string) (interface{}, error) {
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
	imageSplit := strings.Split(adapter.Image(o, t.Version), "/")[2]
	image := strings.Split(imageSplit, ":")

	return godo.AppServiceSpec{
		Name: t.Name,
		Image: &godo.ImageSourceSpec{
			DeployOnPush: &godo.ImageSourceSpecDeployOnPush{
				Enabled: true,
			},
			RegistryType: godo.ImageSourceSpecRegistryType_DOCR,
			Repository:   image[0],
			Tag:          image[1],
		},
		InternalPorts:    []int64{8080},
		Envs:             envs,
		InstanceCount:    1,
		InstanceSizeSlug: "professional-xs",
	}, nil
}

func (t *Transformation) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
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

func (t *Transformation) GetName() string {
	return t.Name
}

func (t *Transformation) GetKind() string {
	return "transformation"
}

func (t *Transformation) GetAPIVersion() string {
	o, err := t.AsK8sObject()
	if err != nil {
		return ""
	}
	return o.APIVersion
}

func (t *Transformation) GetSpec() map[string]interface{} {
	return t.spec
}

func (t *Transformation) SetSpec(spec map[string]interface{}) {
	t.spec = spec
}

func (t *Transformation) GetEventTypes() ([]string, error) {
	if et := t.getContextTransformationValue("type"); len(et) != 0 {
		return et, nil
	}
	return []string{}, fmt.Errorf("%q does not expose event type attributes", t.Name)
}

func (t *Transformation) GetEventSource() (string, error) {
	if src := t.getContextTransformationValue("source"); len(src) != 0 {
		return src[0], nil
	}
	return "", fmt.Errorf("%q does not expose event source attribute", t.Name)
}

func (t *Transformation) ConsumedEventTypes() ([]string, error) {
	return []string{}, nil
}

// SetEventType sets events context attributes.
func (t *Transformation) SetEventAttributes(attributes map[string]string) error {
	var paths []interface{}
	for key, value := range attributes {
		paths = append(paths, map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}
	operation := map[string]interface{}{
		"operation": "add",
		"paths":     paths,
	}

	if contextTransformations, exists := t.spec["context"]; exists {
		contextTransformations = append(contextTransformations.([]interface{}), operation)
		t.spec["context"] = contextTransformations
		return nil
	}
	t.spec["context"] = []interface{}{operation}
	return nil
}

func (t *Transformation) GetPort(ctx context.Context) (string, error) {
	container, err := t.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort(), nil
}

func (t *Transformation) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
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

func (t *Transformation) Stop(ctx context.Context) error {
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

func (t *Transformation) Info(ctx context.Context) (*docker.Container, error) {
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

func (t *Transformation) Logs(ctx context.Context, since time.Time, follow bool) (io.ReadCloser, error) {
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

func New(name, kind, broker, version string, crd crd.CRD, spec map[string]interface{}) triggermesh.Component {
	if name == "" {
		name = fmt.Sprintf("%s-transformation", broker)
	}
	return &Transformation{
		Name:    name,
		CRD:     crd,
		Broker:  broker,
		Version: version,

		spec: spec,
	}
}

// getContextTransformationValue return the value of "Add" transformation
// applied on context attributes. Does not support complex tramsformations.
func (t *Transformation) getContextTransformationValue(key string) []string {
	contextTransformation, exists := t.spec["context"]
	if !exists {
		return []string{}
	}
	for _, op := range contextTransformation.([]interface{}) {
		if opp, ok := op.(map[string]interface{}); ok {
			if opp["operation"] == "add" {
				if p, ok := opp["paths"].([]interface{}); ok {
					for _, pp := range p {
						if pm, ok := pp.(map[string]interface{}); ok {
							if pm["key"] == key {
								return []string{pm["value"].(string)}
							}
						}
					}
				}
			}
		}
	}
	return []string{}
}

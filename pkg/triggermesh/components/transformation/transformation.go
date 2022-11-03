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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
)

var (
	_ triggermesh.Component = (*Transformation)(nil)
	_ triggermesh.Consumer  = (*Transformation)(nil)
	_ triggermesh.Producer  = (*Transformation)(nil)
	_ triggermesh.Runnable  = (*Transformation)(nil)
)

type Transformation struct {
	Name    string
	CRDFile string
	Broker  string
	Version string

	spec map[string]interface{}
}

func (t *Transformation) asUnstructured() (unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(t.GetKind(), t.GetName(), triggermesh.Namespace, t.Broker, t.CRDFile, t.spec, nil)
}

func (t *Transformation) asK8sObject() (kubernetes.Object, error) {
	return kubernetes.CreateObject(t.GetKind(), t.GetName(), triggermesh.Namespace, t.Broker, t.CRDFile, t.spec)
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

func (t *Transformation) Add(manifestPath string) (bool, error) {
	manifest := manifest.New(manifestPath)
	if err := manifest.Read(); err != nil {
		return false, err
	}
	o, err := t.asK8sObject()
	if err != nil {
		return false, err
	}
	if dirty := manifest.Add(o); !dirty {
		return false, nil
	}
	return true, manifest.Write()
}

func (t *Transformation) Delete(manifestPath string) error {
	manifest := manifest.New(manifestPath)
	if err := manifest.Read(); err != nil {
		return err
	}
	manifest.Remove(t.Name, t.GetKind())
	return manifest.Write()
}

func (t *Transformation) GetName() string {
	return t.Name
}

func (t *Transformation) GetKind() string {
	return "transformation"
}

func (t *Transformation) GetSpec() map[string]interface{} {
	return t.spec
}

func (t *Transformation) GetEventTypes() ([]string, error) {
	return t.getContextTransformationValue("type"), nil
}

func (t *Transformation) GetEventSource() (string, error) {
	if src := t.getContextTransformationValue("source"); len(src) != 0 {
		return src[0], nil
	}
	return "", nil
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

func New(name, crdFile, kind, broker, version string, spec map[string]interface{}) triggermesh.Component {
	if name == "" {
		name = fmt.Sprintf("%s-transformation", broker)
	}
	return &Transformation{
		Name:    name,
		CRDFile: crdFile,
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

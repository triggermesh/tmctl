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

package broker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/digitalocean/godo"
	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/kubernetes"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/adapter"
	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

var (
	_ triggermesh.Component   = (*Broker)(nil)
	_ triggermesh.Runnable    = (*Broker)(nil)
	_ triggermesh.Consumer    = (*Broker)(nil)
	_ triggermesh.Exportable  = (*Broker)(nil)
	_ triggermesh.Monitorable = (*Broker)(nil)
)

const (
	BrokerKind  = "RedisBroker"
	TriggerKind = "Trigger"
	APIVersion  = "eventing.triggermesh.io/v1alpha1"
)

type Broker struct {
	Name string

	image      string
	entrypoint []string
	spec       map[string]interface{}
}

func (b *Broker) asUnstructured() (unstructured.Unstructured, error) {
	u := unstructured.Unstructured{}
	u.SetAPIVersion(APIVersion)
	u.SetKind(BrokerKind)
	u.SetName(b.Name)
	u.SetNamespace(triggermesh.Namespace)
	return u, unstructured.SetNestedField(u.Object, nil, "spec")
}

func (b *Broker) AsK8sObject() (kubernetes.Object, error) {
	return kubernetes.Object{
		APIVersion: APIVersion,
		Kind:       BrokerKind,
		Metadata: kubernetes.Metadata{
			Name:      b.Name,
			Namespace: triggermesh.Namespace,
			Labels: map[string]string{
				"triggermesh.io/context": b.Name,
			},
		},
	}, nil
}

func (b *Broker) AsDockerComposeObject(additionalEnvs map[string]string) (interface{}, error) {
	var env []string
	for k, v := range additionalEnvs {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return &docker.ComposeService{
		ContainerName: b.Name,
		Image:         b.image,
		Entrypoint:    b.entrypoint,
		Ports:         []string{strconv.Itoa(pkg.OpenPort()) + ":8080"},
		Environment:   env,
	}, nil
}

func (b *Broker) AsDigitalOceanObject(additionalEnvs map[string]string) (interface{}, error) {
	// Get the image and tag
	imageSplit := strings.Split(b.image, "/")[2]
	image := strings.Split(imageSplit, ":")

	var env []*godo.AppVariableDefinition
	for k, v := range additionalEnvs {
		env = append(env, &godo.AppVariableDefinition{
			Key:   k,
			Value: v,
		})
	}
	return godo.AppServiceSpec{
		Name: b.Name,
		Image: &godo.ImageSourceSpec{
			RegistryType: godo.ImageSourceSpecRegistryType_DockerHub,
			Registry:     config.DockerRegistry,
			Repository:   image[0],
			Tag:          image[1],
		},
		RunCommand:       strings.Join(b.entrypoint, " "),
		InternalPorts:    []int64{8080},
		Envs:             env,
		InstanceCount:    1,
		InstanceSizeSlug: "professional-xs",
	}, nil
}

func (b *Broker) asContainer(additionalEnvs map[string]string) (*docker.Container, error) {
	o, err := b.asUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	co, ho, err := adapter.RuntimeParams(o, b.image, additionalEnvs, triggermesh.AdapterPort, triggermesh.MetricsPort)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}

	co = append(co, docker.WithEntrypoint(b.entrypoint))

	bind := fmt.Sprintf("%s:/etc/triggermesh/broker.conf",
		filepath.Join(config.HomeAbsPath(), b.Name, triggermesh.BrokerConfigFile))
	ho = append(ho, docker.WithVolumeBind(bind))

	name := o.GetName()
	if !strings.HasSuffix(name, "-broker") {
		name = name + "-broker"
	}
	return &docker.Container{
		Name:                   name,
		Image:                  b.image,
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (b *Broker) GetKind() string {
	return BrokerKind
}

func (b *Broker) GetName() string {
	return b.Name
}

func (b *Broker) GetAPIVersion() string {
	return APIVersion
}

func (b *Broker) GetSpec() map[string]interface{} {
	return b.spec
}

func (b *Broker) SetSpec(spec map[string]interface{}) {
	b.spec = spec
}

func (b *Broker) GetPort(ctx context.Context) (string, error) {
	container, err := b.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort("8080"), nil
}

func (b *Broker) ConsumedEventTypes() ([]string, error) {
	return []string{}, nil
}

func (b *Broker) Start(ctx context.Context, additionalEnvs map[string]string, restart bool) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(additionalEnvs)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.Start(ctx, client, restart)
}

func (b *Broker) Stop(ctx context.Context) error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return fmt.Errorf("container object: %w", err)
	}
	return container.Remove(ctx, client)
}

func (b *Broker) Info(ctx context.Context) (*docker.Container, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	return container.LookupHostConfig(ctx, client)
}

func (b *Broker) Logs(ctx context.Context, since time.Time, follow bool) (io.ReadCloser, error) {
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err := b.asContainer(nil)
	if err != nil {
		return nil, fmt.Errorf("container object: %w", err)
	}
	if _, err := container.LookupHostConfig(ctx, client); err != nil {
		return nil, fmt.Errorf("container config: %w", err)
	}
	return container.Logs(ctx, client, since, follow)
}

func CreateBrokerConfig(configHome, broker string) (string, error) {
	brokerHome := filepath.Join(configHome, broker)
	manifestFile := filepath.Join(brokerHome, triggermesh.ManifestFile)
	// create config folder
	if err := os.MkdirAll(brokerHome, os.ModePerm); err != nil {
		return "", fmt.Errorf("broker dir creation: %w", err)
	}
	// create empty manifest
	if _, err := os.Stat(manifestFile); os.IsNotExist(err) {
		if _, err := os.Create(manifestFile); err != nil {
			return "", fmt.Errorf("manifest file creation: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("manifest file access: %w", err)
	}
	brokerConfigPath := filepath.Join(brokerHome, triggermesh.BrokerConfigFile)
	if _, err := os.Stat(brokerConfigPath); os.IsNotExist(err) {
		if _, err := os.Create(brokerConfigPath); err != nil {
			return "", fmt.Errorf("creating broker config: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("broker config read: %w", err)
	}
	return brokerConfigPath, nil
}

func (b *Broker) MetricsPort(ctx context.Context) (string, error) {
	container, err := b.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("container object: %w", err)
	}
	return container.HostPort("9092"), nil
}

func image(c config.BrokerConfig) string {
	switch {
	case c.Memory != nil:
		return config.MemoryBrokerImage + ":" + c.Version
	case c.Redis != nil:
		return config.RedisBrokerImage + ":" + c.Version
	}
	return ""
}

func brokerEntrypoint(c config.BrokerConfig) []string {
	var entrypoint []string
	switch {
	case c.Memory != nil:
		entrypoint = []string{
			"/memory-broker",
			"start",
			"--memory.buffer-size",
			c.Memory.BufferSize,
			"--memory.produce-timeout",
			c.Memory.ProduceTimeout,
		}
	case c.Redis != nil:
		entrypoint = []string{
			"/redis-broker",
			"start",
			"--redis.username",
			c.Redis.Username,
			"--redis.password",
			c.Redis.Password,
			"--redis.address",
			c.Redis.Address,
		}
		if c.Redis.TLSEnabled {
			entrypoint = append(entrypoint, "--redis.tls-enabled")
		}
		if c.Redis.SkipVerify {
			entrypoint = append(entrypoint, "--redis.tls-skip-verify")
		}
		if c.ConfigPollingPeriod != "" {
			entrypoint = append(entrypoint, []string{"--config-polling-period", c.ConfigPollingPeriod}...)
		}
	}
	return append(entrypoint, []string{"--broker-config-path", "/etc/triggermesh/broker.conf"}...)
}

func New(name string, brokerConfig config.BrokerConfig) (triggermesh.Component, error) {
	return &Broker{
		Name: name,

		image:      image(brokerConfig),
		entrypoint: brokerEntrypoint(brokerConfig),
	}, nil
}

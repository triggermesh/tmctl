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

package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh"
	"github.com/triggermesh/tmcli/pkg/triggermesh/adapter"
	tmbroker "github.com/triggermesh/tmcli/pkg/triggermesh/broker"
	"github.com/triggermesh/tmcli/pkg/triggermesh/pkg"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ triggermesh.Component = (*Source)(nil)

type Source struct {
	ManifestFile string
	CRDFile      string

	Broker  string
	Kind    string
	Version string

	image string
	args  map[string]interface{}
}

func (s *Source) AsUnstructured() (*unstructured.Unstructured, error) {
	return kubernetes.CreateUnstructured(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.args)
}

func (s *Source) AsK8sObject() (*kubernetes.Object, error) {
	return kubernetes.CreateObject(s.GetKind(), s.GetName(), s.Broker, s.CRDFile, s.args)
}

func (s *Source) AsContainer() (*docker.Container, error) {
	o, err := s.AsUnstructured()
	if err != nil {
		return nil, fmt.Errorf("creating object: %w", err)
	}
	image, err := adapter.Image(o, s.Version)
	if err != nil {
		return nil, fmt.Errorf("adapter image: %w", err)
	}
	s.image = image
	co, ho, err := adapter.RuntimeParams(o, image)
	if err != nil {
		return nil, fmt.Errorf("creating adapter params: %w", err)
	}
	return &docker.Container{
		Name:                   o.GetName(),
		CreateHostOptions:      ho,
		CreateContainerOptions: co,
	}, nil
}

func (s *Source) GetName() string {
	return fmt.Sprintf("%s-%ssource", s.Broker, s.Kind)
}

func (s *Source) GetKind() string {
	return fmt.Sprintf("%ssource", strings.ToLower(s.Kind))
}

func (s *Source) GetImage() string {
	return s.image
}

func NewSource(manifest, crd string, kind, broker, version string, args []string) (*Source, error) {
	b, err := tmbroker.NewBroker(manifest, broker, version)
	if err != nil {
		return nil, fmt.Errorf("broker error: %w", err)
	}
	container, err := b.AsContainer()
	if err != nil {
		return nil, fmt.Errorf("broker container: %w", err)
	}
	// TODO remove this
	client, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	container, err = container.LookupHostConfig(context.Background(), client)
	if err != nil {
		return nil, fmt.Errorf("broker lookup: %w", err)
	}
	socket := container.Socket()
	if socket == "" {
		return nil, fmt.Errorf("broker socket is empty")
	}
	args = append(args, fmt.Sprintf("--sink.uri=http://%s", socket))
	return &Source{
		ManifestFile: manifest,
		CRDFile:      crd,
		Broker:       broker,
		Kind:         kind,
		Version:      version,
		args:         pkg.ParseArgs(args),
	}, nil
}

// func Create(kind, broker, socket string, args []string, manifestFile, crdFile string) (*kubernetes.Object, bool, error) {
// 	manifest := manifest.New(manifestFile)
// 	err := manifest.Read()
// 	if err != nil {
// 		return nil, false, fmt.Errorf("unable to read the manifest: %w", err)
// 	}

// 	spec := pkg.ParseArgs(args)
// 	spec["sink"] = map[string]interface{}{
// 		"uri": socket,
// 	}

// 	source, err := kubernetes.CreateObject(strings.ToLower(kind)+"source", broker+"-source", broker, crdFile, spec)
// 	if err != nil {
// 		return nil, false, fmt.Errorf("creating object: %w", err)
// 	}

// 	dirty, err := manifest.Add(*source)
// 	if err != nil {
// 		return nil, false, fmt.Errorf("manifest update: %w", err)
// 	}
// 	if dirty {
// 		if err := manifest.Write(); err != nil {
// 			return nil, false, fmt.Errorf("manifest write operation: %w", err)
// 		}
// 	}
// 	return source, dirty, nil
// }

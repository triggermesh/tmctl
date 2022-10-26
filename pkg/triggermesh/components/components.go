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

package components

import (
	"fmt"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/target"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/transformation"
)

func GetObject(name, manifestFile, crdFile, version string) (triggermesh.Component, error) {
	manifest := manifest.New(manifestFile)
	if err := manifest.Read(); err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	for _, object := range manifest.Objects {
		if object.Metadata.Name == name {
			switch object.APIVersion {
			case "sources.triggermesh.io/v1alpha1":
				return source.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			case "targets.triggermesh.io/v1alpha1":
				return target.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			case "flow.triggermesh.io/v1alpha1":
				return transformation.New(object.Metadata.Name, crdFile, object.Kind, "", version, object.Spec), nil
			}
		}
	}
	return nil, nil
}

func ProducersEventTypes(name, manifestFile, crdFile, version string) ([]string, error) {
	c, err := GetObject(name, manifestFile, crdFile, version)
	if err != nil {
		return []string{}, fmt.Errorf("%q does not exist", name)
	}
	producer, ok := c.(triggermesh.Producer)
	if !ok {
		return []string{}, fmt.Errorf("event producer %q is not available", name)
	}
	et, err := producer.GetEventTypes()
	if err != nil {
		return []string{}, fmt.Errorf("%q event types: %w", name, err)
	}
	if len(et) == 0 {
		return []string{}, fmt.Errorf("%q does not expose its event types", name)
	}
	return et, nil
}

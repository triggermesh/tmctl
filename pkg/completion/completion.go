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

package completion

import (
	"strings"

	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/source"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func ListSources(manifestFile string) []string {
	var list []string
	m := manifest.New(manifestFile)
	if err := m.Read(); err != nil {
		return []string{}
	}
	for _, object := range m.Objects {
		if strings.HasPrefix(object.APIVersion, "sources.triggermesh.io") {
			list = append(list, object.Metadata.Name)
		}
	}
	return list
}

func ListTargets(manifestFile string) []string {
	var list []string
	m := manifest.New(manifestFile)
	if err := m.Read(); err != nil {
		return []string{}
	}
	for _, object := range m.Objects {
		if strings.HasPrefix(object.APIVersion, "targets.triggermesh.io") {
			list = append(list, object.Metadata.Name)
		}
	}
	return list
}

func ListAll(manifestFile string) []string {
	m := manifest.New(manifestFile)
	if err := m.Read(); err != nil {
		return []string{}
	}
	var list []string
	for _, object := range m.Objects {
		list = append(list, object.Metadata.Name)
	}
	return list
}

func ListEventTypes(manifestFile, crdFile string) []string {
	var eventTypes []string
	m := manifest.New(manifestFile)
	if err := m.Read(); err != nil {
		return []string{}
	}
	for _, object := range m.Objects {
		if strings.HasPrefix(object.APIVersion, "sources.triggermesh.io") {
			s := source.New(object.Metadata.Name, crdFile, object.Kind, "", "", object.Spec)
			et, err := s.GetEventTypes()
			if err != nil {
				continue
			}
			eventTypes = append(eventTypes, et...)
		}
	}
	return eventTypes
}

func SpecFromCRD(name, crdFile string, path ...string) (bool, map[string]crd.Property) {
	result := make(map[string]crd.Property, 0)
	c, err := crd.GetResourceCRD(name, crdFile)
	if err != nil {
		return false, result
	}
	var schema *crd.Schema
	for _, version := range c.Spec.Versions {
		if version.Served {
			if schema, err = crd.GetSchema(version.Schema.OpenAPIV3Schema.Properties.Spec); err != nil {
				return false, result
			}
			break
		}
	}
	return schema.GetAttributesCompletion(path...)
}

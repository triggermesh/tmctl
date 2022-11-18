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
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	tmbroker "github.com/triggermesh/tmctl/pkg/triggermesh/components/broker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func ListSources(m *manifest.Manifest) []string {
	var list []string
	for _, object := range m.Objects {
		if object.APIVersion == "sources.triggermesh.io/v1alpha1" ||
			object.APIVersion == "flow.triggermesh.io/v1alpha1" {
			list = append(list, object.Metadata.Name)
		}
		if object.APIVersion == service.APIVersion {
			if role, set := object.Metadata.Labels[service.RoleLabel]; set && role == string(service.Producer) {
				list = append(list, object.Metadata.Name)
			}
		}
	}
	return list
}

func ListTargets(m *manifest.Manifest) []string {
	var list []string
	for _, object := range m.Objects {
		if object.APIVersion == "targets.triggermesh.io/v1alpha1" ||
			object.APIVersion == "flow.triggermesh.io/v1alpha1" {
			list = append(list, object.Metadata.Name)
		}
		if object.APIVersion == service.APIVersion {
			if role, set := object.Metadata.Labels[service.RoleLabel]; set && role == string(service.Consumer) {
				list = append(list, object.Metadata.Name)
			}
		}
	}
	return list
}

func ListAll(m *manifest.Manifest) []string {
	var list []string
	for _, object := range m.Objects {
		list = append(list, object.Metadata.Name)
	}
	return list
}

func ListEventTypes(m *manifest.Manifest, crdFile, version string) []string {
	var eventTypes []string
	for _, object := range m.Objects {
		c, err := components.GetObject(object.Metadata.Name, crdFile, version, m)
		if err == nil {
			if producer, ok := c.(triggermesh.Producer); ok {
				et, _ := producer.GetEventTypes()
				eventTypes = append(eventTypes, et...)
			}
		}
	}
	return eventTypes
}

func ListFilteredEventTypes(broker, configBase string, m *manifest.Manifest) []string {
	var eventTypes []string
	for _, object := range m.Objects {
		if object.Kind != tmbroker.TriggerKind {
			continue
		}
		trigger, err := tmbroker.NewTrigger(object.Metadata.Name, broker, configBase, nil, nil)
		if err != nil {
			continue
		}
		trigger.(*tmbroker.Trigger).LookupTarget()
		for _, filter := range trigger.(*tmbroker.Trigger).Filters {
			if eventType, set := filter.Exact["type"]; set {
				eventTypes = append(eventTypes, eventType)
			}
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

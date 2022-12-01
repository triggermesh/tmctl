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

package triggermesh

import (
	"encoding/json"
	"net/http"
)

var DefaultConfig = map[string]interface{}{
	"context":                                   defaultContext,
	"triggermesh.version":                       latestOrDefault(defaultVersion),
	"triggermesh.broker.image":                  MemoryBrokerImage,
	"triggermesh.broker.memory.buffer-size":     MemoryBrokerBufferSize,
	"triggermesh.broker.memory.produce-timeout": MemoryBrokerProduceTimeout,
}

// TriggerMesh constant values used as default paths, configs, etc.
const (
	ConfigFile = "config.yaml"
	ConfigDir  = ".triggermesh/cli"

	defaultContext = ""
	Namespace      = "local"
	ManifestFile   = "manifest.yaml"

	// objects meta
	ContextLabel                = "triggermesh.io/context"
	ExternalResourcesAnnotation = "triggermesh.io/external-resources"

	// version default parameters
	ghLatestRelease = "https://api.github.com/repos/triggermesh/triggermesh/releases/latest"
	defaultVersion  = "v1.22.0"

	// broker default parameters
	MemoryBrokerImage          = "gcr.io/triggermesh/memory-broker:latest"
	MemoryBrokerBufferSize     = "100"
	MemoryBrokerProduceTimeout = "1s"
)

type release struct {
	TagName string `json:"tag_name"`
}

func latestOrDefault(defaultVersion string) string {
	r, err := http.Get(ghLatestRelease)
	if err != nil {
		return defaultVersion
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return defaultVersion
	}
	var j release
	if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
		return defaultVersion
	}
	return j.TagName
}

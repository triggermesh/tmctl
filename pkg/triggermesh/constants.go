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

// TriggerMesh constant values used as default paths, configs, etc.
const (
	Namespace        = "local"
	ManifestFile     = "manifest.yaml"
	BrokerConfigFile = "broker.conf"

	UserInputTag = "<user_input>"

	// objects meta
	ContextLabel                = "triggermesh.io/context"
	ExternalResourcesAnnotation = "triggermesh.io/external-resources"

	// adapter params
	AdapterPort = "8080/tcp"
	MetricsPort = "9092/tcp"
)

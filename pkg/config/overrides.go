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

package config

import "strings"

type configOverride func(*Config) bool

var overrides = []configOverride{
	brokerImageReplacement(),
	dockerTimeoutAppend(),
	schemaRegistryAppend(),
}

func (c *Config) applyOverrides() error {
	overwritten := false
	for _, override := range overrides {
		if override(c) {
			overwritten = true
		}
	}
	if overwritten {
		return c.Save()
	}
	return nil
}

func brokerImageReplacement() configOverride {
	return func(c *Config) bool {
		if c.Triggermesh.Broker.Image == "" {
			return false
		}
		imageRef := strings.Split(c.Triggermesh.Broker.Image, ":")
		if len(imageRef) == 2 {
			if c.Triggermesh.Broker.Version == "" {
				if imageRef[1] == "latest" {
					imageRef[1] = latestOrDefaultTag("brokers", defaultBrokerVersion)
				}
				c.Triggermesh.Broker.Version = imageRef[1]
			}
		}
		if c.Triggermesh.Broker.Version == "" {
			c.Triggermesh.Broker.Version = latestOrDefaultTag("brokers", defaultBrokerVersion)
		}
		c.Triggermesh.Broker.Image = ""
		return true
	}
}

func dockerTimeoutAppend() configOverride {
	return func(c *Config) bool {
		if c.Docker.StartTimeout != "" {
			return false
		}
		c.Docker.StartTimeout = defaultDockerTimeout
		return true
	}
}

func schemaRegistryAppend() configOverride {
	return func(c *Config) bool {
		if c.SchemaRegistry != "" {
			return false
		}
		c.SchemaRegistry = defaultSchemaRegistryURL
		return true
	}
}

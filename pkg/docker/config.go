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

package docker

import (
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

const errorLoggingLevel = `K_LOGGING_CONFIG={"zap-logger-config":"{\"level\": \"error\"}"}`

type ContainerOption func(*container.Config)
type HostOption func(*container.HostConfig)

func WithImage(image string) ContainerOption {
	return func(cc *container.Config) {
		cc.Image = image
	}
}

func WithEnv(env []string) ContainerOption {
	return func(cc *container.Config) {
		cc.Env = append(cc.Env, env...)
	}
}

func WithPort(port nat.Port) ContainerOption {
	return func(cc *container.Config) {
		cc.ExposedPorts = nat.PortSet{
			port: struct{}{},
		}
	}
}

func WithEntrypoint(entrypoint []string) ContainerOption {
	return func(cc *container.Config) {
		cc.Entrypoint = entrypoint
	}
}

func WithVolumeBind(bind string) HostOption {
	return func(hc *container.HostConfig) {
		hc.Binds = []string{bind}
	}
}

func WithHostPortBinding(containerPort nat.Port) HostOption {
	return func(hc *container.HostConfig) {
		hc.PortBindings = nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(pkg.OpenPort()),
				},
			},
		}
	}
}

func WithExtraHost() HostOption {
	return func(hc *container.HostConfig) {
		hc.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}
}

func WithErrorLoggingLevel() ContainerOption {
	return func(cc *container.Config) {
		cc.Env = append(cc.Env, errorLoggingLevel)
	}
}

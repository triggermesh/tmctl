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
)

type ContainerOption func(*container.Config)
type HostOption func(*container.HostConfig)

func (c *Client) WithImage(image string) ContainerOption {
	return func(cc *container.Config) {
		cc.Image = image
	}
}

func (c *Client) WithEnv(env []string) ContainerOption {
	return func(cc *container.Config) {
		cc.Env = env
	}
}

func (c *Client) WithPort(port nat.Port) ContainerOption {
	return func(cc *container.Config) {
		cc.ExposedPorts = nat.PortSet{
			port: struct{}{},
		}
	}
}

func (c *Client) WithEntrypoint(entrypoint string) ContainerOption {
	return func(cc *container.Config) {
		cc.Entrypoint = []string{entrypoint}
	}
}

func (c *Client) WithVolumeBind(bind string) HostOption {
	return func(hc *container.HostConfig) {
		hc.Binds = []string{bind}
	}
}

func (c *Client) WithHostPortBinding(containerPort nat.Port) HostOption {
	return func(hc *container.HostConfig) {
		hc.PortBindings = nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(openPort()),
				},
			},
		}
	}
}

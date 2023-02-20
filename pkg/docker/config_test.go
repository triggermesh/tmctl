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
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"

	"github.com/stretchr/testify/assert"
)

func TestWithImage(t *testing.T) {
	image := "foo/bar"
	cc := &container.Config{}
	WithImage(image)(cc)
	assert.Equal(t, image, cc.Image)
}

func TestWithEnv(t *testing.T) {
	env := []string{"foo=bar", "blah=bleh"}
	cc := &container.Config{}
	WithEnv(env)(cc)
	assert.Equal(t, env, cc.Env)
}

func TestWithPort(t *testing.T) {
	port, err := nat.NewPort("TCP", "8080")
	assert.NoError(t, err)
	portSet := nat.PortSet{
		port: struct{}{},
	}
	cc := &container.Config{}
	WithPort(string(port))(cc)
	assert.Equal(t, portSet, cc.ExposedPorts)
}

func TestWithEntrypoint(t *testing.T) {
	entrypoint := []string{"/bin/triggermesh", "start"}
	cc := &container.Config{}
	WithEntrypoint(entrypoint)(cc)
	assert.Equal(t, strslice.StrSlice(entrypoint), cc.Entrypoint)
}

func TestWithVolumeBind(t *testing.T) {
	bind := "foo:bar"
	hc := &container.HostConfig{}
	WithVolumeBind(bind)(hc)
	assert.Equal(t, []string{bind}, hc.Binds)
}

func TestWithHostPortBinding(t *testing.T) {
	port, err := nat.NewPort("TCP", "8080")
	assert.NoError(t, err)
	hc := &container.HostConfig{}
	WithHostPortBinding(string(port))(hc)
	assert.Len(t, hc.PortBindings[port], 1)
	assert.Equal(t, "0.0.0.0", hc.PortBindings[port][0].HostIP)
}

func TestWithExtraHost(t *testing.T) {
	hc := &container.HostConfig{}
	WithExtraHost()(hc)
	assert.Len(t, hc.ExtraHosts, 1)
	assert.Equal(t, "host.docker.internal:host-gateway", hc.ExtraHosts[0])
}

func TestWithErrorLoggingLevel(t *testing.T) {
	cc := &container.Config{}
	WithErrorLoggingLevel()(cc)
	assert.Contains(t, cc.Env, envErrorLoggingLevel)
}

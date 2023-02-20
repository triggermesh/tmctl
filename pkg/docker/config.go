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
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/triggermesh/tmctl/pkg/triggermesh/pkg"
)

const (
	envErrorLoggingLevel = `K_LOGGING_CONFIG={"zap-logger-config":"{\"level\": \"error\"}"}`
	envMetricsPort       = "METRICS_PROMETHEUS_PORT=9092"
)

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

func WithPort(ports ...string) ContainerOption {
	portSet := make(nat.PortSet)
	for _, port := range ports {
		portSet[nat.Port(port)] = struct{}{}
	}
	return func(cc *container.Config) { cc.ExposedPorts = portSet }
}

func WithEntrypoint(entrypoint []string) ContainerOption {
	return func(cc *container.Config) {
		cc.Entrypoint = entrypoint
	}
}

func WithCmd(cmd []string) ContainerOption {
	return func(cc *container.Config) {
		cc.Cmd = cmd
	}
}

func WithVolumeBind(bind string) HostOption {
	return func(hc *container.HostConfig) {
		hc.Binds = append(hc.Binds, bind)
	}
}

func WithHostPortBinding(containerPorts ...string) HostOption {
	ports := make(nat.PortMap)
	for _, containerPort := range containerPorts {
		ports[nat.Port(containerPort)] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(pkg.OpenPort()),
			},
		}
	}
	return func(hc *container.HostConfig) { hc.PortBindings = ports }
}

func WithExtraHost() HostOption {
	return func(hc *container.HostConfig) {
		hc.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}
}

func WithErrorLoggingLevel() ContainerOption {
	return func(cc *container.Config) {
		cc.Env = append(cc.Env, envErrorLoggingLevel)
	}
}

func WithMetricsConfig(component string) ContainerOption {
	metricsEnv := fmt.Sprintf("K_METRICS_CONFIG={\"Domain\":\"triggermesh.io\",\"Component\":\"%s\",\"PrometheusPort\":0,\"PrometheusHost\":\"\",\"ConfigMap\":{\"metrics.backend-destination\":\"prometheus\"}}",
		strings.ToLower(component))
	return func(cc *container.Config) {
		cc.Env = append(cc.Env, []string{metricsEnv, envMetricsPort}...)
	}
}

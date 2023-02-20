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

package monitoring

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/triggermesh/tmctl/pkg/docker"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"gopkg.in/yaml.v3"
)

func CreatePrometheusContainer(ctx context.Context, config string) (*docker.Container, error) {
	s := service.New("triggermesh-prometheus", "prom/prometheus", "dummy", service.Role("custom"), nil)
	container, err := s.(*service.Service).AsContainer(nil)
	if err != nil {
		return nil, err
	}
	bind := fmt.Sprintf("%s:/etc/prometheus/prometheus.yml", config)
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithVolumeBind(bind))
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithHostPortBinding("9090"))
	container.CreateContainerOptions = append(container.CreateContainerOptions, docker.WithPort("9090"))
	container.CreateContainerOptions = append(container.CreateContainerOptions, docker.WithCmd([]string{
		"--web.enable-lifecycle",
		"--config.file=/etc/prometheus/prometheus.yml",
	}))

	client, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return container.Start(ctx, client, false)
}

func CreateGrafanaContainer(ctx context.Context, promURL, configHome string) (*docker.Container, error) {
	s := service.New("triggermesh-grafana", "grafana/grafana-oss", "dummy", service.Role("custom"), nil)
	container, err := s.(*service.Service).AsContainer(nil)
	if err != nil {
		return nil, err
	}
	dsProvision := createDataSourceProvision(promURL)
	ds, err := yaml.Marshal(dsProvision)
	if err != nil {
		return nil, err
	}
	dsProvisionConf := filepath.Join(configHome, "/grafana.yaml")
	if err := os.WriteFile(dsProvisionConf, ds, os.ModePerm); err != nil {
		return nil, err
	}

	dashboardProvisionConf := filepath.Join(configHome, "/dashboard.yaml")
	if err := os.WriteFile(dashboardProvisionConf, createDashboardProvision(), os.ModePerm); err != nil {
		return nil, err
	}
	dashboardJSON := filepath.Join(configHome, "/dashboard.json")
	if err := os.WriteFile(dashboardJSON, []byte(dashboard), os.ModePerm); err != nil {
		return nil, err
	}
	dataSourceBind := fmt.Sprintf("%s:/etc/grafana/provisioning/datasources/prometheus.yaml", dsProvisionConf)
	dashboardProvisionBind := fmt.Sprintf("%s:/etc/grafana/provisioning/dashboards/dashboard.yaml", dashboardProvisionConf)
	dashboardBind := fmt.Sprintf("%s:/etc/grafana/provisioning/dashboards/default.json", dashboardJSON)
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithVolumeBind(dataSourceBind))
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithVolumeBind(dashboardProvisionBind))
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithVolumeBind(dashboardBind))
	container.CreateHostOptions = append(container.CreateHostOptions, docker.WithHostPortBinding("3000"))
	container.CreateContainerOptions = append(container.CreateContainerOptions, docker.WithPort("3000"))

	client, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	return container.Start(ctx, client, false)
}

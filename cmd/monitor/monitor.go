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

package monitor

import (
	"context"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/log"
	"github.com/triggermesh/tmctl/pkg/monitoring"
)

type CliOptions struct {
	Config     *config.Config
	Monitoring *monitoring.Configuration
}

func NewCmd(config *config.Config, prom *monitoring.Configuration) *cobra.Command {
	o := &CliOptions{
		Config:     config,
		Monitoring: prom,
	}
	brokerCmd := &cobra.Command{
		Use:               "monitor",
		Short:             "Monitor TriggerMesh components",
		Example:           "tmctl monitor",
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.monitor()
		},
	}
	return brokerCmd
}

func (o *CliOptions) monitor() error {
	log.Println("Initializing Prometheus")
	prom, err := monitoring.CreatePrometheusContainer(context.Background(), o.Monitoring.Path)
	if err != nil {
		return err
	}
	promURL := "http://host.docker.internal:" + prom.HostPort("9090")

	log.Println("Initializing Grafana")
	grafana, err := monitoring.CreateGrafanaContainer(context.Background(), promURL, o.Config.ConfigHome)
	if err != nil {
		return err
	}

	grafanaURL := "http://localhost:" + grafana.HostPort("3000")
	cmd := exec.Command("open", grafanaURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Println("Opening dashboard")
	return cmd.Run()
}

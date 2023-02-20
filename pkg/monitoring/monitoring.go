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
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	scrapeInterval     = "5s"
	evaluationInterval = "5s"
)

type Configuration struct {
	Path string `yaml:"-"`

	Global        Global   `yaml:"global"`
	ScrapeConfigs []Target `yaml:"scrape_configs"`
}

type Global struct {
	ScrapeInterval     string `yaml:"scrape_interval"`
	EvaluationInterval string `yaml:"evaluation_interval"`
}

type Target struct {
	JobName       string         `yaml:"job_name"`
	MetricsPath   string         `yaml:"metrics_path,omitempty"`
	StaticConfigs []TargetConfig `yaml:"static_configs"`
}

type TargetConfig struct {
	Targets []string          `yaml:"targets"`
	Labels  map[string]string `yaml:"labels,omitempty"`
}

func NewConfiguration(path string) (*Configuration, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err := os.Create(path); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return &Configuration{
		Path: path,
		Global: Global{
			ScrapeInterval:     scrapeInterval,
			EvaluationInterval: evaluationInterval,
		},
	}, nil
}

func (c *Configuration) AddTarget(name, port, context string) error {
	newTarget := Target{
		JobName:     strings.Replace(name, "-", "_", -1),
		MetricsPath: "/metrics",
		StaticConfigs: []TargetConfig{
			{
				Targets: []string{
					"host.docker.internal:" + port,
				},
				Labels: map[string]string{
					"context": context,
				},
			},
		},
	}
	for i, target := range c.ScrapeConfigs {
		if target.JobName == strings.Replace(name, "-", "_", -1) {
			if len(target.StaticConfigs) == 1 &&
				len(target.StaticConfigs[0].Targets) == 1 &&
				target.StaticConfigs[0].Targets[0] == "host.docker.internal:"+port {
				return nil
			}
			c.ScrapeConfigs[i] = newTarget
			return c.Write()
		}
	}
	c.ScrapeConfigs = append(c.ScrapeConfigs, newTarget)
	return c.Write()
}

func (c *Configuration) DeleteTarget(name string) error {
	newTargets := make([]Target, 0)
	for _, target := range c.ScrapeConfigs {
		if target.JobName == strings.Replace(name, "-", "_", -1) {
			continue
		}
		newTargets = append(newTargets, target)
	}
	c.ScrapeConfigs = newTargets
	return c.Write()
}

func (c *Configuration) Read() error {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func (c *Configuration) Write() error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, data, os.ModePerm)
}

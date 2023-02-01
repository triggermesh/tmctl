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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/triggermesh/tmctl/pkg/config"
)

// time to wait for adapter init logs to show up.
var initLogsWaitPeriod time.Duration = 2 * time.Second

type imagePullEvent struct {
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"progressDetail"`
}

type Container struct {
	ID     string
	Name   string
	Image  string
	Online bool

	CreateContainerOptions []ContainerOption
	CreateHostOptions      []HostOption

	runtimeHostConfig      container.HostConfig
	runtimeContainerConfig container.Config
}

func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func CheckDaemon() error {
	c, err := NewClient()
	if err != nil {
		return err
	}
	_, err = c.ServerVersion(context.Background())
	return err
}

func (c *Container) Logs(ctx context.Context, client *client.Client, since time.Time, follow bool) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Since:      since.Format("2006-01-02T15:04:05.999999999Z07:00")}
	return client.ContainerLogs(ctx, c.ID, options)
}

func (c *Container) Remove(ctx context.Context, client *client.Client) error {
	id, err := nameToID(ctx, c.Name, client)
	if err != nil {
		return err
	}
	return client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (c *Container) pullImage(ctx context.Context, client *client.Client) error {
	reader, err := client.ImagePull(ctx, c.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	d := json.NewDecoder(reader)
	var e *imagePullEvent
	var downloading bool
	for {
		if err := d.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if e.Status == "Downloading" {
			downloading = true
			fmt.Printf("\r%s", e.Progress)
		}
	}
	if downloading {
		fmt.Printf("\n")
	}
	return nil
}

func (c *Container) Start(ctx context.Context, client *client.Client, restart bool) (*Container, error) {
	cc := container.Config{}
	for _, opt := range c.CreateContainerOptions {
		opt(&cc)
	}

	hc := container.HostConfig{}
	for _, opt := range c.CreateHostOptions {
		opt(&hc)
	}

	if err := c.pullImage(ctx, client); err != nil {
		return nil, fmt.Errorf("pulling image: %w", err)
	}

	var containerIsRunning bool
	existingContainer, _ := c.LookupHostConfig(ctx, client)
	if existingContainer != nil {
		if c.Image != existingContainer.Image {
			restart = true
		}
		if existingContainer.Online {
			containerIsRunning = true
		}
	}
	if restart {
		// remove errors usually means that container doesn't exist
		// ignore it and try to create a new one.
		_ = c.Remove(ctx, client)
	} else if containerIsRunning {
		return existingContainer, nil
	}

	resp, err := client.ContainerCreate(ctx, &cc, &hc, nil, nil, c.Name)
	if err != nil {
		return nil, fmt.Errorf("docker create: %w", err)
	}

	c.ID = resp.ID
	c.runtimeHostConfig = hc
	c.runtimeContainerConfig = cc

	sinceStart := time.Now()
	if err := client.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("docker start: %w", err)
	}
	configTimeout, err := config.Get("docker.timeout")
	if err != nil {
		return nil, fmt.Errorf("config read: %w", err)
	}
	timeout, err := time.ParseDuration(configTimeout)
	if err != nil {
		return nil, fmt.Errorf("config timeout value: %w", err)
	}
	if err := c.isRunning(ctx, client, timeout); err != nil {
		return nil, fmt.Errorf("docker connect: %w", err)
	}
	time.Sleep(initLogsWaitPeriod)
	logsReader, err := c.Logs(ctx, client, sinceStart, false)
	if err != nil {
		return nil, fmt.Errorf("docker read logs: %w", err)
	}
	defer logsReader.Close()

	for _, log := range readLogs(logsReader) {
		var l map[string]interface{}
		if err := json.Unmarshal([]byte(log), &l); err != nil {
			// unstructured log output, e.g. go's panic dump
			if strings.Contains(log, "panic: ") {
				return nil, fmt.Errorf("container log: %s", log)
			}
			continue
		}
		if isError(l) {
			return nil, fmt.Errorf("container log: %s", log)
		}
	}
	return c, nil
}

func nameToID(ctx context.Context, name string, client *client.Client) (string, error) {
	containers, err := client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}
	for _, container := range containers {
		for _, cName := range container.Names {
			if cName == "/"+name {
				return container.ID, nil
			}
		}
	}
	return "", nil
}

func (c *Container) LookupHostConfig(ctx context.Context, client *client.Client) (*Container, error) {
	id, err := nameToID(ctx, c.Name, client)
	if err != nil {
		return nil, err
	}
	jsn, err := client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	c.ID = id
	if jsn.State.Running {
		c.Online = true
	}
	c.runtimeHostConfig = *jsn.HostConfig
	c.runtimeContainerConfig = *jsn.Config
	return c, nil
}

func (c *Container) HostPort() string {
	for _, bindings := range c.runtimeHostConfig.PortBindings {
		for _, binding := range bindings {
			return binding.HostPort
		}
	}
	return ""
}

func (c *Container) isRunning(ctx context.Context, client *client.Client, timeout time.Duration) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	cancel := time.After(timeout)
	for {
		container, err := client.ContainerInspect(ctx, c.ID)
		if err != nil {
			return err
		}
		select {
		case <-cancel:
			return fmt.Errorf("container init timeout, state: %s", container.State.Status)
		case <-ticker.C:
			if container.State.Running {
				return nil
			}
		}
	}
}

func ForceStop(ctx context.Context, name string, client *client.Client) error {
	id, err := nameToID(ctx, name, client)
	if err != nil {
		return err
	}
	return client.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func readLogs(logs io.ReadCloser) []string {
	var output []string
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		if l := len(scanner.Bytes()); l > 8 {
			output = append(output, string(scanner.Bytes()[8:]))
		}
	}
	return output
}

func isError(logEntry map[string]interface{}) bool {
	for k, v := range logEntry {
		if k == "level" || k == "severity" {
			switch strings.ToLower(v.(string)) {
			case "error", "fatal", "alert", "panic":
				return true
			}
		}
	}
	return false
}

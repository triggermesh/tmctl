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
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	connRetries     = 5
	logsReadTimeout = 2 // timeout to wait for logs, in seconds
)

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
	ID    string
	Name  string
	Image string

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
		// Since: (time.Now().Add(2 * time.Second).Format("2006-01-02T15:04:05.999999999Z07:00"))}
		Since: since.Format("2006-01-02T15:04:05.999999999Z07:00")}
	return client.ContainerLogs(ctx, c.ID, options)
}

func (c *Container) Remove(ctx context.Context, client *client.Client) error {
	// c.ID = ""
	// c.runtimeContainerConfig = container.Config{}
	// c.runtimeHostConfig = container.HostConfig{}
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
		if err := existingContainer.Connect(ctx); err == nil {
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

	since := time.Now()
	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("docker start: %w", err)
	}

	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("docker connect: %w", err)
	}

	time.Sleep(logsReadTimeout * time.Second)
	logsReader, err := c.Logs(ctx, client, since, false)
	if err != nil {
		return nil, fmt.Errorf("docker read logs: %w", err)
	}
	defer logsReader.Close()

	for _, log := range readLogs(logsReader) {
		var l map[string]interface{}
		if err := json.Unmarshal([]byte(log), &l); err != nil {
			continue
		}
		if isError(l) {
			return nil, fmt.Errorf("container error: %s", string(log))
		}
	}
	return c, nil
}

func openPort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
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
	if !jsn.State.Running {
		return nil, fmt.Errorf("container is offline")
	}
	c.ID = id
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

func (c *Container) Connect(ctx context.Context) error {
	timer := time.NewTicker(time.Second)
	till := time.Now().Add(connRetries * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-timer.C:
			if now.After(till) {
				return fmt.Errorf("service wait timeout")
			}
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("0.0.0.0:%s", c.HostPort()), time.Second)
			if err != nil {
				continue
			}
			if conn != nil {
				conn.Close()
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

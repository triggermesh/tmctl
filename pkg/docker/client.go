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
	ID   string
	Name string

	CreateContainerOptions []ContainerOption
	CreateHostOptions      []HostOption

	runtimeHostConfig      container.HostConfig
	runtimeContainerConfig container.Config
}

func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func (c *Container) Logs(ctx context.Context, client *client.Client) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
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

func (c *Container) PullImage(ctx context.Context, client *client.Client, image string) error {
	reader, err := client.ImagePull(ctx, image, types.ImagePullOptions{})
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

func (c *Container) Start(ctx context.Context, client *client.Client) (*Container, error) {
	cc := container.Config{}
	for _, opt := range c.CreateContainerOptions {
		opt(&cc)
	}

	hc := container.HostConfig{}
	for _, opt := range c.CreateHostOptions {
		opt(&hc)
	}

	resp, err := client.ContainerCreate(ctx, &cc, &hc, nil, nil, c.Name)
	if err != nil {
		return nil, fmt.Errorf("docker create: %w", err)
	}

	c.ID = resp.ID
	c.runtimeHostConfig = hc
	c.runtimeContainerConfig = cc

	if err := client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("docker start: %w", err)
	}

	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("docker connect: %w", err)
	}

	logsReader, err := c.Logs(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("docker read logs: %w", err)
	}
	defer logsReader.Close()

	stopLogs := make(chan struct{}, 1)
	logs := readLogs(logsReader, stopLogs)
	defer close(stopLogs)
	timer := time.NewTimer(time.Second * logsReadTimeout)

	for {
		select {
		case <-timer.C:
			stopLogs <- struct{}{}
			return c, nil
		case log := <-logs:
			var l map[string]interface{}
			if err := json.Unmarshal(log, &l); err != nil {
				continue
				// return nil, fmt.Errorf("docker unmarshal logs: %w", err)
			}
			if isError(l) {
				return nil, fmt.Errorf(string(log))
			}
		}
	}
}

func (c *Container) Inspect(ctx context.Context, client client.Client) (types.ContainerJSON, error) {
	return client.ContainerInspect(ctx, c.ID)
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
	c.runtimeHostConfig = *jsn.HostConfig
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

func readLogs(output io.ReadCloser, done chan struct{}) chan []byte {
	logs := make(chan []byte)
	scanner := bufio.NewScanner(output)
	go func() {
		for scanner.Scan() {
			select {
			case <-done:
				close(logs)
				return
			default:
				if l := len(scanner.Bytes()); l > 8 {
					logs <- scanner.Bytes()[8:]
				}
			}
		}
	}()
	return logs
}

func isError(logEntry map[string]interface{}) bool {
	for k, v := range logEntry {
		if k == "level" || k == "severity" {
			if strings.ToLower(v.(string)) == "error" {
				return true
			}
		}
	}
	return false
}

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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
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
	HC container.HostConfig
	CC container.Config
}

type Client struct {
	docker *client.Client
}

func NewClient() (Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	return Client{cli}, err
}

func (c Client) Logs(ctx context.Context, id string) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true}
	return c.docker.ContainerLogs(ctx, id, options)
}

func (c Client) RemoveContainer(ctx context.Context, name string) error {
	id, err := c.nameToID(ctx, name)
	if err != nil {
		return err
	}
	if id == "" {
		return nil
	}
	return c.docker.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

func (c Client) PullImage(ctx context.Context, image string) error {
	reader, err := c.docker.ImagePull(ctx, image, types.ImagePullOptions{})
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

func (c Client) StartContainer(ctx context.Context, copts []ContainerOption, hopts []HostOption, name string) (Container, error) {
	cc := container.Config{}
	for _, opt := range copts {
		opt(&cc)
	}

	hc := container.HostConfig{}
	for _, opt := range hopts {
		opt(&hc)
	}

	resp, err := c.docker.ContainerCreate(ctx, &cc, &hc, nil, nil, name)
	if err != nil {
		return Container{}, err
	}

	if err := c.docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return Container{}, err
	}

	// var socket string
	// for _, bindings := range hc.PortBindings {
	// 	for _, binding := range bindings {
	// 		socket = fmt.Sprintf("%s:%s", binding.HostIP, binding.HostPort)
	// 	}
	// }

	return Container{
		HC: hc,
		CC: cc,
	}, nil
}

func (c Client) Inspect(ctx context.Context, name string) (types.ContainerJSON, error) {
	id, err := c.nameToID(ctx, name)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	return c.docker.ContainerInspect(ctx, id)
}

func openPort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func (c Client) nameToID(ctx context.Context, name string) (string, error) {
	containers, err := c.docker.ContainerList(ctx, types.ContainerListOptions{
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

func Socket(c Container) string {
	for _, bindings := range c.HC.PortBindings {
		for _, binding := range bindings {
			return fmt.Sprintf("%s:%s", binding.HostIP, binding.HostPort)
		}
	}
	return ""
}

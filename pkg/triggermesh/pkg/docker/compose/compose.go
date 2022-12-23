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

package compose

import (
	"math/rand"
	"strconv"
	"time"
)

type DockerCompose struct {
	Services Services `json:"services"`
}

type Services map[string]DockerComposeService

type DockerComposeService struct {
	ContainerName string                `json:"container_name"`
	Command       string                `json:"command,omitempty"`
	Image         string                `json:"image"`
	Ports         []string              `json:"ports"`
	Environment   []string              `json:"environment"`
	Volumes       []DockerComposeVolume `json:"volumes"`
}

type DockerComposeVolume struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func RandomPort() string {
	rand.Seed(time.Now().UnixNano())

	min := 49152
	max := 65535

	number := rand.Intn(max-min) + min

	return strconv.Itoa(number)
}

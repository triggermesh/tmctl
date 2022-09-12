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

package runtime

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
	"github.com/triggermesh/tmcli/pkg/triggermesh/env"
	"github.com/triggermesh/tmcli/pkg/triggermesh/function"
)

func runAdapter(ctx context.Context, d docker.Client, k8sObject *kubernetes.Object, version string) (string, error) {
	containerOptions := []docker.ContainerOption{
		d.WithPort(adapterPort),
	}
	hostOptions := []docker.HostOption{
		d.WithHostPortBinding(adapterPort),
	}

	image := imageURI(k8sObject.Kind, version)
	var err error
	var kenv []corev1.EnvVar

	var brkrfl *os.File

	buildEnv := true
	switch k8sObject.Kind {
	case "Function":
		if image, err = function.ImageName(k8sObject); err != nil {
			return "", fmt.Errorf("cannot parse function image: %w", err)
		}
		image = fmt.Sprintf("%s:%s", image, version)
		file, err := createSharedFile(function.Code(k8sObject))
		if err != nil {
			return "", fmt.Errorf("writing function: %w", err)
		}
		bind := fmt.Sprintf("%s:/opt/source.%s", file.Name(), function.FileExtension(k8sObject))
		hostOptions = append(hostOptions, d.WithVolumeBind(bind))
		containerOptions = append(containerOptions, d.WithEntrypoint("/opt/aws-custom-runtime"))
	case "Broker":
		image = "docker.io/tzununbekov/memory-broker"
		buildEnv = false
		brkrfl, err = createSharedFile("")
		if err != nil {
			return "", fmt.Errorf("writing function: %w", err)
		}
		bind := fmt.Sprintf("%s:/etc/triggermesh/broker.conf", brkrfl.Name())
		hostOptions = append(hostOptions, d.WithVolumeBind(bind))
	}
	containerOptions = append(containerOptions, d.WithImage(image))

	if buildEnv {
		kenv, err = env.BuildEnv(k8sObject)
		if err != nil {
			return "", fmt.Errorf("adapter environment: %w", err)
		}
		containerOptions = append(containerOptions, d.WithEnv(envsToString(kenv)))
	}

	log.Println("Checking adapter image", image)
	if err := d.PullImage(ctx, image); err != nil {
		return "", fmt.Errorf("cannot pull Docker image: %w", err)
	}

	log.Printf("Starting %q", k8sObject.Metadata.Name)
	socket, err := d.StartContainer(ctx, containerOptions, hostOptions, k8sObject.Metadata.Name)
	if err != nil {
		return "", fmt.Errorf("cannot run Docker container: %w", err)
	}

	waitForService(ctx, socket)

	// if _, err := brkrfl.WriteString(brokerConfig); err != nil {
	// panic(fmt.Errorf("cannot write file payload: %w", err))
	// }

	return socket, nil
}

func waitForService(ctx context.Context, socket string) {
	timer := time.NewTicker(time.Second)
	till := time.Now().Add(connRetries * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-timer.C:
			if now.After(till) {
				panic(fmt.Errorf("service wait timeout"))
			}
			conn, err := net.DialTimeout("tcp", socket, time.Second)
			if err != nil {
				continue
			}
			if conn != nil {
				conn.Close()
				return
			}
		}
	}
}

func imageURI(kind, version string) string {
	adapter := fmt.Sprintf("%s-adapter:%s", strings.ToLower(kind), version)
	return path.Join(tmContainerRegistry, adapter)
}

func envsToString(envs []corev1.EnvVar) []string {
	var result []string
	for _, env := range envs {
		result = append(result, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return result
}

func createSharedFile(data string) (*os.File, error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp file: %w", err)
	}
	if _, err := file.WriteString(data); err != nil {
		return nil, fmt.Errorf("cannot write file payload: %w", err)
	}
	return file, nil
}

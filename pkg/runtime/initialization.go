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
	"time"

	"github.com/triggermesh/tmcli/pkg/docker"
	"github.com/triggermesh/tmcli/pkg/kubernetes"
)

func Initialize(ctx context.Context, object *kubernetes.Object, version string, dirty bool) error {
	status, err := getStatus(ctx, object)
	if err != nil {
		return fmt.Errorf("cannot read container status: %w", err)
	}

	switch {
	case status == "not found":
		// create
		if _, err := runObject(ctx, object, version); err != nil {
			return fmt.Errorf("cannot start container: %w", err)
		}
	case dirty || status == "exited" || status == "dead":
		// recreate
		if err := stopObject(ctx, object); err != nil {
			return fmt.Errorf("cannot stop container: %w", err)
		}
		if _, err := runObject(ctx, object, version); err != nil {
			return fmt.Errorf("cannot start container: %w", err)
		}
	default:
		fmt.Printf("Doing nothing because status is %q and dirty flag is %t\n", status, dirty)
	}
	return nil
}

func runAdapter(ctx context.Context, d docker.Client, name, image string,
	containerOptions []docker.ContainerOption, hostOptions []docker.HostOption) (string, error) {

	log.Println("Checking adapter image")
	if err := d.PullImage(ctx, image); err != nil {
		return "", fmt.Errorf("cannot pull Docker image: %w", err)
	}

	log.Printf("Starting adapter")
	socket, err := d.StartContainer(ctx, containerOptions, hostOptions, name)
	if err != nil {
		return "", fmt.Errorf("cannot run Docker container: %w", err)
	}

	if err := waitForService(ctx, socket); err != nil {
		return "", fmt.Errorf("adapter initialization: %w", err)
	}

	// if _, err := brkrfl.WriteString(brokerConfig); err != nil {
	// panic(fmt.Errorf("cannot write file payload: %w", err))
	// }

	return socket, nil
}

func waitForService(ctx context.Context, socket string) error {
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
			conn, err := net.DialTimeout("tcp", socket, time.Second)
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

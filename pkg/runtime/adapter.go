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
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/triggermesh/tmcli/pkg/docker"
	tm "github.com/triggermesh/tmcli/pkg/triggermesh/env"
)

func initializeAdapter(ctx context.Context, d docker.Client, k8sObject *unstructured.Unstructured, version string, secrets []string) (container, error) {
	containerOptions := []docker.ContainerOption{
		d.WithPort(adapterPort),
	}
	hostOptions := []docker.HostOption{
		d.WithHostPortBinding(adapterPort),
	}

	image := imageURI(k8sObject.GetKind(), version)
	var err error
	var file *os.File

	buildEnv := true
	var kenv []corev1.EnvVar

	switch k8sObject.GetKind() {
	case "Function":
		if image, err = tm.FunctionRuntimeImage(k8sObject); err != nil {
			return container{}, fmt.Errorf("cannot parse function image: %w", err)
		}
		image = fmt.Sprintf("%s:%s", image, version)

		if file, err = ioutil.TempFile("", ""); err != nil {
			return container{}, fmt.Errorf("cannot create temp file: %w", err)
		}

		if _, err := file.WriteString(tm.FunctionCode(k8sObject)); err != nil {
			return container{}, fmt.Errorf("cannot write function code: %w", err)
		}
		bind := fmt.Sprintf("%s:/opt/source.%s", file.Name(), tm.FunctionFileExtension(k8sObject))
		hostOptions = append(hostOptions, d.WithVolumeBind(bind))
		containerOptions = append(containerOptions, d.WithEntrypoint("/opt/aws-custom-runtime"))
	case "Broker":
		image = "docker.io/tzununbekov/memory-broker"
		buildEnv = false
	}
	containerOptions = append(containerOptions, d.WithImage(image))

	if buildEnv {
		kenv, err = adapterEnv(k8sObject, secrets)
		if err != nil {
			return container{}, fmt.Errorf("adapter environment: %w", err)
		}
		containerOptions = append(containerOptions, d.WithEnv(envsToString(kenv)))
	}

	log.Println("Checking adapter image", image)
	if err := d.PullImage(ctx, image); err != nil {
		return container{}, fmt.Errorf("cannot pull Docker image: %w", err)
	}

	log.Printf("Starting %q", k8sObject.GetName())
	socket, err := d.RunAdapter(ctx, containerOptions, hostOptions, k8sObject.GetName())
	if err != nil {
		return container{}, fmt.Errorf("cannot run Docker container: %w", err)
	}

	// go func() {
	// 	<-ctx.Done()
	// 	if file != nil {
	// 		os.Remove(file.Name())
	// 	}
	// 	if err := d.RemoveAdapter(context.Background(), id); err != nil {
	// 		panic(fmt.Errorf("cannot remove Docker container: %w\nPlease remove container %q manually", err, id))
	// 	}
	// }()

	waitForService(ctx, socket)

	return container{
		object: k8sObject,
		socket: socket,
	}, nil
}

func adapterEnv(object *unstructured.Unstructured, secrets []string) ([]corev1.EnvVar, error) {
	kenv, err := tm.BuildEnv(object)
	if err != nil {
		return []corev1.EnvVar{}, fmt.Errorf("cannot build object environment: %w", err)
	}

	for k, v := range kenv {
		if v.ValueFrom != nil {
			provided := false
			ref := fmt.Sprintf("%s/%s=", v.ValueFrom.SecretKeyRef.LocalObjectReference.Name, v.ValueFrom.SecretKeyRef.Key)
			for _, secret := range secrets {
				if strings.HasPrefix(secret, ref) {
					provided = true
					kenv[k].Value = secret[len(ref):]
					kenv[k].ValueFrom = nil
				}
			}
			if !provided {
				fmt.Printf("Please provide the secret as a flag attribute: \"-s %s<secret value>\"\n", ref)
				return []corev1.EnvVar{}, fmt.Errorf("secret is required")
			}
		}
	}
	return kenv, nil
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

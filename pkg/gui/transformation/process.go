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

package transformation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"sigs.k8s.io/yaml"

	"github.com/jroimartin/gocui"
)

const (
	registryUrl = "http://localhost:8080/apis/registry/v2/groups/schema/artifacts/"
)

func ProcessKeystrokes(g *gocui.Gui, signals chan signal) error {
	nesting := make([]string, 10)

	for s := range signals {
		switch s.origin {
		case "sources":
			outputView, _ := g.View("sourceEvent")
			outputView.Clear()
			outputView.Wrap = true
			fmt.Fprintln(outputView, loadSample(strings.TrimLeft(strings.TrimSpace(s.line), "-")))
		case "targets":
			outputView, _ := g.View("targetEvent")
			outputView.Clear()
			outputView.Wrap = true
			fmt.Fprintln(outputView, loadSample(strings.TrimLeft(strings.TrimSpace(s.line), "-")))
		case "sourceEvent":
			parts := strings.Split(s.line, "\":")
			if len(parts) == 1 {
				continue
			}
			parts = strings.Split(parts[0], "\"")
			spaces := len(parts[0])
			key := strings.TrimSpace(parts[1])
			nesting[spaces/2-1] = key // indentation is 2 spaces per object
			for i := spaces / 2; i < len(nesting); i++ {
				nesting[i] = ""
			}
		case "transformationOperation":
			transformations := []v1alpha1.Transform{}
			transformationView, _ := g.View("transformationContext")
			if err := yaml.Unmarshal([]byte(transformationView.Buffer()), &transformations); err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}

			value := ""
			operation := strings.TrimLeft(s.line, "-")
			if line := strings.Split(operation, ":"); len(line) == 2 {
				operation = line[0]
				value = line[1]
			}
			transformations = updateTransformations(transformations, operation, nesting, value)

			output, err := yaml.Marshal(transformations)
			if err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}

			transformationView.Clear()
			transformationView.Write(output)
		default:
			// debug
			debugOutput, _ := g.View("transformationContext")
			debugOutput.Autoscroll = true
			fmt.Fprintln(debugOutput, s)
		}
		g.Update(func(g *gocui.Gui) error { return nil })
	}
	return nil
}

func loadSample(eventType string) string {
	url := registryUrl + eventType
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Sprintf("registry request error: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return "Not found"
	}
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("registry response error: %v", err)
	}
	data, err := json.MarshalIndent(schemaToData(responseData), "", "  ")
	if err != nil {
		return fmt.Sprintf("sample error: %v", err)
	}
	return string(data)
}

func removeEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func updateTransformations(transformations []v1alpha1.Transform, operation string, path []string, value string) []v1alpha1.Transform {
	if len(transformations) == 0 {
		return append(transformations, v1alpha1.Transform{
			Operation: operation,
			Paths: []v1alpha1.Path{
				{
					Key:   strings.Join(removeEmptyStrings(path), "."),
					Value: value,
				},
			},
		})
	}

	for k, v := range transformations {
		if v.Operation == operation {
			v.Paths = append(v.Paths, v1alpha1.Path{
				Key:   strings.Join(removeEmptyStrings(path), "."),
				Value: value,
			})
			transformations[k] = v
			return transformations
		}
	}

	return append(transformations, v1alpha1.Transform{
		Operation: operation,
		Paths: []v1alpha1.Path{
			{
				Key:   strings.Join(removeEmptyStrings(path), "."),
				Value: value,
			},
		},
	})
}

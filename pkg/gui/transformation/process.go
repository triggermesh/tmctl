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
	"fmt"
	"strings"

	"github.com/jroimartin/gocui"
	"sigs.k8s.io/yaml"

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
)

type input struct {
	key   string
	value string
}

func ProcessKeystrokes(g *gocui.Gui, signals chan signal, cache registryCache, transformations map[string]transformationObject) {
	nesting := make([]string, 10) // maximum level of objects netsing in the event
	selectedSource := ""
	selectedTarget := ""

	for s := range signals {
		switch s.origin {
		case "sources":
			line := strings.TrimLeft(s.line, " -")
			if selectedSource == line {
				continue
			}
			outputView, _ := g.View("sourceEvent")
			outputView.Clear()

			fmt.Fprintln(outputView, string(cache[line]))

			transformationView, _ := g.View("transformation")
			transformationView.Clear()
			selectedSource = line
			if transformation := existingTransformation(selectedSource, selectedTarget, transformations); transformation.spec != "" {
				fmt.Fprintln(transformationView, transformation.spec)
			}
		case "targets":
			line := strings.TrimLeft(s.line, " -")
			if selectedTarget == line {
				continue
			}
			outputView, _ := g.View("targetEvent")
			outputView.Clear()
			if sample, exists := cache[line]; exists {
				fmt.Fprintln(outputView, string(sample))
			} else {
				fmt.Fprintln(outputView, string("Component accepts arbitrary event format"))
			}

			transformationView, _ := g.View("transformation")
			transformationView.Clear()
			selectedTarget = line
			if transformation := existingTransformation(selectedSource, selectedTarget, transformations); transformation.spec != "" {
				fmt.Fprintln(transformationView, transformation.spec)
			}
		case "sourceEvent":
			switch s.line {
			case "{", "}":
				nesting[0] = "."
			default:
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
			}
		case "transformationOperation":
			transformations := []v1alpha1.Transform{}
			transformationView, err := g.View("transformation")
			if err != nil {
				continue
			}

			operation := strings.TrimLeft(s.line, "-")
			path := strings.Join(removeEmptyStrings(nesting), ".")
			if s.isHotKey {
				path = ""
			}

			key, value, err := readOperation(operation, path, g)
			if err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}
			if err := yaml.Unmarshal([]byte(transformationView.Buffer()), &transformations); err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}

			transformations = updateTransformations(transformations, operation, key, value)
			transformations = rearrange(transformations)
			if len(transformations) == 0 {
				continue
			}

			output, err := yaml.Marshal(transformations)
			if err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}
			transformationView.Clear()
			fmt.Fprintln(transformationView, string(output))
		}
		g.Update(func(g *gocui.Gui) error { return nil })
	}
}

func readOperation(operation, path string, g *gocui.Gui) (string, string, error) {
	value := ""
	switch operation {
	case "delete", "parse":
		g.DeleteKeybindings("transformationOperation")
		_ = g.DeleteView("transformationOperation")
		_, _ = g.SetCurrentView("sourceEvent")
	case "add", "store", "shift":
		inputValue := make(chan *input)
		if err := popInputValueView(path, inputValue, g); err != nil {
			return "", "", err
		}
		g.Update(func(g *gocui.Gui) error { return nil })

		in := <-inputValue
		if in == nil {
			return "", "", nil
		}

		path = "."
		value = in.value
		if in.key != "" {
			path = in.key
		}
	}
	return path, value, nil
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

func updateTransformations(transformations []v1alpha1.Transform, operation, path, value string) []v1alpha1.Transform {
	switch operation {
	case "store":
		p := path
		path = value
		value = p
	case "shift":
		path = fmt.Sprintf("%s:%s", path, value)
		value = ""
	}

	if len(transformations) == 0 {
		return []v1alpha1.Transform{
			{
				Operation: operation,
				Paths: []v1alpha1.Path{
					{
						Key:   path,
						Value: value,
					},
				},
			},
		}
	}

	for k, v := range transformations {
		if v.Operation == operation {
			v.Paths = append(v.Paths, v1alpha1.Path{
				Key:   path,
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
				Key:   path,
				Value: value,
			},
		},
	})
}

func rearrange(transformations []v1alpha1.Transform) []v1alpha1.Transform {
	store := []v1alpha1.Transform{}
	delete := []v1alpha1.Transform{}
	add := []v1alpha1.Transform{}
	shift := []v1alpha1.Transform{}
	parse := []v1alpha1.Transform{}

	wipeData := false

	for _, transformation := range transformations {
		transformation = sanitize(transformation)
		if transformation.Paths == nil {
			continue
		}
		switch transformation.Operation {
		case "parse":
			parse = append(parse, transformation)
		case "store":
			store = append(store, transformation)
		case "delete":
			for _, path := range transformation.Paths {
				if path.Key == "" {
					wipeData = true
					delete = []v1alpha1.Transform{{
						Operation: "delete",
						Paths:     []v1alpha1.Path{{Key: ""}},
					}}
					break
				}
			}
			if !wipeData {
				delete = append(delete, transformation)
			}
		case "add":
			add = append(add, transformation)
		case "shift":
			shift = append(shift, transformation)
		}
	}

	// first operations are parse and store
	transformations = append(parse, store...)
	// then we delete, including full event wipe
	transformations = append(transformations, delete...)
	// modification operations go last
	transformations = append(transformations, append(add, shift...)...)

	return transformations
}

func existingTransformation(source, target string, transformations map[string]transformationObject) transformationObject {
	transformationLabel := source
	if target != "" {
		transformationLabel = source + "-" + target
	}
	if t, exists := transformations[transformationLabel]; exists {
		return t
	}
	return transformationObject{}
}

func sanitize(transformation v1alpha1.Transform) v1alpha1.Transform {
	result := v1alpha1.Transform{}
	result.Operation = transformation.Operation

	for _, path := range transformation.Paths {
		switch transformation.Operation {
		case "delete", "parse":
		case "add", "store":
			if path.Value == "" || path.Key == "" {
				continue
			}
		case "shift":
			pair := strings.Split(path.Key, ":")
			if len(pair) != 2 || pair[0] == "" || pair[1] == "" {
				continue
			}
		}
		if i := index(path.Key, result.Paths); i > -1 {
			result.Paths[i] = path
			continue
		}
		result.Paths = append(result.Paths, path)
	}
	return result
}

func index(key string, paths []v1alpha1.Path) int {
	for i, j := range paths {
		if j.Key == key {
			return i
		}
	}
	return -1
}

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

	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"sigs.k8s.io/yaml"

	"github.com/jroimartin/gocui"
)

func ProcessKeystrokes(g *gocui.Gui, signals chan signal, registryUrl string) error {
	nesting := make([]string, 10) // maximum level of objects netsing in the event
	cache := make(registryCache, 20)
	currentEventType := ""

	for s := range signals {
		switch s.origin {
		case "sources":
			outputView, _ := g.View("sourceEvent")
			outputView.Clear()
			outputView.Wrap = true

			eventType := strings.TrimLeft(strings.TrimSpace(s.line), "-")
			fmt.Fprintln(outputView, loadSample(registryUrl, eventType, cache))

			if currentEventType == "" {
				currentEventType = eventType
			} else if currentEventType != eventType {
				currentEventType = eventType
				transformationView, _ := g.View("transformation")
				transformationView.Clear()
			}
		case "targets":
			outputView, _ := g.View("targetEvent")
			outputView.Clear()
			outputView.Wrap = true
			fmt.Fprintln(outputView, loadSample(registryUrl, strings.TrimLeft(strings.TrimSpace(s.line), "-"), cache))
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
			transformationView, _ := g.View("transformation")

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

			output, err := yaml.Marshal(transformations)
			if err != nil {
				fmt.Fprintln(transformationView, err.Error())
				continue
			}
			transformationView.Clear()
			transformationView.Write(output)
		default:
		}
		g.Update(func(g *gocui.Gui) error { return nil })
	}
	return nil
}

func readOperation(operation, path string, g *gocui.Gui) (string, string, error) {
	value := ""
	switch operation {
	case "delete", "parse":
	case "add", "store", "shift":
		inputValue := make(chan string)
		inputView, err := popInputValueView(path, g)
		if err != nil {
			return "", "", err
		}
		g.Update(func(g *gocui.Gui) error { return nil })
		if err := g.SetKeybinding(inputView.Name(), gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
			value := v.Buffer()
			if err := g.DeleteView(inputView.Name()); err != nil {
				return err
			}
			g.DeleteKeybindings(inputView.Name())
			inputValue <- value
			return nil
		}); err != nil {
			return "", "", err
		}
		input := <-inputValue

		inputs := strings.Split(input, ":")
		path = strings.TrimSpace(inputs[0])
		value = strings.TrimSpace(inputs[1])
	}
	g.DeleteView("transformationOperation")
	if _, err := g.SetCurrentView("sourceEvent"); err != nil {
		return "", "", err
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

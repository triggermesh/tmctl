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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jroimartin/gocui"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

func Create(crds map[string]crd.CRD, manifest *manifest.Manifest, config *config.Config) (string, string, string, io.Reader, error) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return "", "", "", nil, err
	}
	defer g.Close()

	layout := NewLayout()

	g.Cursor = true
	g.SetManagerFunc(layout.draw)

	errC := make(chan error)
	go func() {
		if err := g.MainLoop(); err != nil {
			errC <- err
		}
		close(errC)
	}()

	counter := 0
	for {
		if layout.sources != nil || counter > 3 {
			break
		}
		time.Sleep(300 * time.Millisecond)
		counter++
	}

	if layout.sources == nil {
		return "", "", "", nil, fmt.Errorf("view init timeout")
	}

	sources, targets, err := sourcesAndTargets(manifest, config, crds)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("component event types: %w", err)
	}
	for source, eventTypes := range sources {
		fmt.Fprintf(layout.sources, "%s:\n", source)
		for _, et := range eventTypes {
			fmt.Fprintf(layout.sources, " -%s\n", et)
		}
		// fmt.Fprintln(layout.sources)
	}
	for target, eventTypes := range targets {
		fmt.Fprintf(layout.targets, "%s:\n", target)
		for _, et := range eventTypes {
			fmt.Fprintf(layout.targets, " -%s\n", et)
		}
		// fmt.Fprintln(layout.targets)
	}
	g.Update(func(g *gocui.Gui) error { return nil })

	keybindingHandler := NewKeybindingHandler()
	if err := keybindingHandler.Create(g); err != nil {
		return "", "", "", nil, err
	}

	go ProcessKeystrokes(g, keybindingHandler.signals)

	select {
	case err := <-errC:
		return "", "", "", nil, err
	case name := <-keybindingHandler.createAndExit:
		sourceET, target, spec, err := readLayout(layout)
		if err != nil {
			return "", "", "", nil, err
		}
		g.Close()
		s := map[string]interface{}{
			"data": spec,
		}
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(s); err != nil {
			return "", "", "", nil, err
		}
		return name, target, sourceET, b, err
	}
}

func sourcesAndTargets(m *manifest.Manifest, config *config.Config, crds map[string]crd.CRD) (map[string][]string, map[string][]string, error) {
	sources := make(map[string][]string, 0)
	targets := make(map[string][]string, 0)
	for _, object := range m.Objects {
		switch object.APIVersion {
		case "sources.triggermesh.io/v1alpha1", "flow.triggermesh.io/v1alpha1":
			et, err := sourceEventTypes(object.Metadata.Name, config, m, crds)
			if err != nil {
				return nil, nil, err
			}
			sources[object.Metadata.Name] = et
		case "targets.triggermesh.io/v1alpha1":
			et, err := targetEventTypes(object.Metadata.Name, config, m, crds)
			if err != nil {
				return nil, nil, err
			}
			targets[object.Metadata.Name] = et
			// case service.APIVersion:
			// 	role, set := object.Metadata.Labels[service.RoleLabel]
			// 	if !set {
			// 		continue
			// 	}
			// 	switch role {
			// 	case string(service.Producer):
			// 		sources[object.Metadata.Name] = et
			// 	case string(service.Consumer):
			// 		targets[object.Metadata.Name] = et
			// 	}
		}
	}
	return sources, targets, nil
}

func sourceEventTypes(name string, config *config.Config, manifest *manifest.Manifest, crds map[string]crd.CRD) ([]string, error) {
	object, err := components.GetObject(name, config, manifest, crds)
	if err != nil {
		return []string{}, err
	}
	return object.(triggermesh.Producer).GetEventTypes()
}

func targetEventTypes(name string, config *config.Config, manifest *manifest.Manifest, crds map[string]crd.CRD) ([]string, error) {
	object, err := components.GetObject(name, config, manifest, crds)
	if err != nil {
		return []string{}, err
	}
	return object.(triggermesh.Consumer).ConsumedEventTypes()
}

func readLayout(l *layout) (string, string, string, error) {
	_, cy := l.sources.Cursor()
	source, err := l.sources.Line(cy)
	if err != nil {
		return "", "", "", err
	}
	source = strings.TrimLeft(strings.TrimSpace(source), "-")
	_, cy = l.targets.Cursor()
	target, err := l.targets.Line(cy)
	if err != nil {
		return "", "", "", err
	}
	if target == "*" {
		target = ""
	}
	spec := l.transformation.Buffer()
	return source, target, spec, nil
}

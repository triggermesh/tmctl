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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jroimartin/gocui"

	"github.com/triggermesh/tmctl/pkg/config"
	"github.com/triggermesh/tmctl/pkg/manifest"
	"github.com/triggermesh/tmctl/pkg/triggermesh"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components"
	"github.com/triggermesh/tmctl/pkg/triggermesh/components/service"
	"github.com/triggermesh/tmctl/pkg/triggermesh/crd"
)

type list struct {
	targets map[string]meta
	sources map[string]meta
}

type meta struct {
	eventTypes []string
	kind       string
}

type registryCache map[string][]byte

func Create(crds map[string]crd.CRD, manifest *manifest.Manifest, config *config.Config) (string, string, string, io.Reader, error) {
	componentsList, err := parseComponents(manifest, config, crds)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("manifest read error: %w", err)
	}

	cache := make(registryCache)

	if err := preloadRegistryData(componentsList.sources, config.SchemaRegistry, cache); err != nil {
		return "", "", "", nil, fmt.Errorf("registry error: %w", err)
	}
	if err := preloadRegistryData(componentsList.targets, config.SchemaRegistry, cache); err != nil {
		return "", "", "", nil, fmt.Errorf("registry error: %w", err)
	}

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

	for name, source := range componentsList.sources {
		if len(source.eventTypes) != 0 {
			name = name + ":"
		}
		fmt.Fprintln(layout.sources, name)
		for _, et := range source.eventTypes {
			fmt.Fprintf(layout.sources, " -%s\n", et)
		}
	}
	for name, target := range componentsList.targets {
		if len(target.eventTypes) != 0 {
			name = name + ":"
		}
		fmt.Fprintln(layout.targets, name)
		for _, et := range target.eventTypes {
			fmt.Fprintf(layout.targets, " -%s\n", et)
		}
	}
	g.Update(func(g *gocui.Gui) error { return nil })

	keybindingHandler := NewKeybindingHandler()
	if err := keybindingHandler.Create(g); err != nil {
		return "", "", "", nil, err
	}

	go ProcessKeystrokes(g, keybindingHandler.signals, cache)

	select {
	case err := <-errC:
		return "", "", "", nil, err
	case name := <-keybindingHandler.createAndExit:
		sourceEventType, targetComponent, targetEventType, spec, err := readLayout(layout)
		if err != nil {
			return "", "", "", nil, err
		}
		g.Close()
		s := map[string]interface{}{
			"data": spec,
		}
		if targetEventType != "" {
			s["context"] = json.RawMessage("[{\"operation\":\"add\",\"paths\":[{\"key\":\"type\",\"value\":\"" + targetEventType + "\"}]}]")
		}
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(s); err != nil {
			return "", "", "", nil, err
		}
		return name, sourceEventType, targetComponent, b, err
	}
}

func parseComponents(m *manifest.Manifest, config *config.Config, crds map[string]crd.CRD) (list, error) {
	sources := make(map[string]meta, 0)
	targets := make(map[string]meta, 0)
	for _, object := range m.Objects {
		switch object.APIVersion {
		case "sources.triggermesh.io/v1alpha1", "flow.triggermesh.io/v1alpha1":
			et, err := sourceEventTypes(object.Metadata.Name, config, m, crds)
			if err != nil {
				return list{}, err
			}
			sources[object.Metadata.Name] = meta{
				eventTypes: et,
				kind:       object.Kind,
			}
		case "targets.triggermesh.io/v1alpha1":
			et, err := targetEventTypes(object.Metadata.Name, config, m, crds)
			if err != nil {
				return list{}, err
			}
			targets[object.Metadata.Name] = meta{
				eventTypes: et,
				kind:       object.Kind,
			}
		case service.APIVersion:
			role, set := object.Metadata.Labels[service.RoleLabel]
			if !set {
				continue
			}
			switch role {
			case string(service.Producer):
				sources[object.Metadata.Name] = meta{}
			case string(service.Consumer):
				targets[object.Metadata.Name] = meta{}
			}
		}
	}
	return list{
		targets: targets,
		sources: sources,
	}, nil
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

func readLayout(l *layout) (string, string, string, string, error) {
	_, cy := l.sources.Cursor()
	sourceEventType, err := l.sources.Line(cy)
	if err != nil {
		return "", "", "", "", err
	}
	sourceEventType = strings.TrimLeft(strings.TrimSpace(sourceEventType), "-")

	_, cy = l.targets.Cursor()
	targetSelectedLine, err := l.targets.Line(cy)
	if err != nil {
		return "", "", "", "", err
	}
	targetEventType := ""
	targetComponent := ""

	switch {
	case targetSelectedLine == "*":
		targetComponent = ""
		targetEventType = ""
	case strings.HasPrefix(targetSelectedLine, " -"):
		for i := cy - 1; i > 0; i-- {
			selectedLine, err := l.targets.Line(i)
			if err != nil {
				break
			}
			if strings.HasPrefix(selectedLine, " -") {
				continue
			}
			targetComponent = selectedLine
			break
		}
		targetEventType = strings.TrimLeft(targetSelectedLine, " -")
	default:
		targetComponent = strings.TrimRight(targetSelectedLine, ":")
	}
	spec := l.transformation.Buffer()
	return sourceEventType, targetComponent, targetEventType, spec, nil
}

func preloadRegistryData(componentsList map[string]meta, registryUrl string, cache map[string][]byte) error {
	cache["*"] = []byte("Not selected")

	for _, component := range componentsList {
		for _, eventType := range component.eventTypes {
			registryEndpoint, err := url.JoinPath(registryUrl, "schemagroups", component.kind, "schemas", eventType)
			if err != nil {
				return fmt.Errorf("registry path error: %v", err)
			}
			resp, err := http.Get(registryEndpoint)
			if err != nil {
				return fmt.Errorf("registry request error: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				cache[eventType] = []byte("Event schema not available: " + resp.Status)
				continue
			}
			responseData, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("registry response read error: %v", err)
			}
			data, err := json.MarshalIndent(schemaToData(responseData), "", "  ")
			if err != nil {
				return fmt.Errorf("sample error: %v", err)
			}
			cache[eventType] = data
		}
	}
	return nil
}

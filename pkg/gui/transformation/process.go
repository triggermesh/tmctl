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

	"github.com/jroimartin/gocui"
)

const (
	registryUrl = "http://localhost:8080/apis/registry/v2/groups/schema/artifacts/"
)

func ProcessKeystrokes(g *gocui.Gui, signals chan signal) error {
	for s := range signals {

		var outputView *gocui.View

		switch s.origin {
		case "sources":
			outputView, _ = g.View("sourceEvent")
		case "targets":
			outputView, _ = g.View("targetEvent")
		default:
			// debug
			debugOutput, _ := g.View("transformationContext")
			debugOutput.Autoscroll = true
			fmt.Fprintln(debugOutput, s)
			g.Update(func(g *gocui.Gui) error { return nil })
			continue
		}

		outputView.Clear()
		outputView.Wrap = true

		message := func() string {
			url := registryUrl + strings.TrimLeft(strings.TrimSpace(s.line), "-")
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
			schema, err := responseToSchema(responseData)
			if err != nil {
				return fmt.Sprintf("schema parse error: %v", err)
			}
			data, err := json.MarshalIndent(schemaToData(schema), "", "  ")
			if err != nil {
				return fmt.Sprintf("sample error: %v", err)
			}
			return string(data)
		}()

		fmt.Fprintln(outputView, message)
		g.Update(func(g *gocui.Gui) error { return nil })
	}
	return nil
}

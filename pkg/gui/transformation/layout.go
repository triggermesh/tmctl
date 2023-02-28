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
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if leftTop, err := g.SetView("source", 0, 0, int(0.1*float32(maxX)), maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		leftTop.Title = "Source (Ctrl+S)"
		leftTop.Highlight = true
		leftTop.SelBgColor = gocui.ColorGreen
		fmt.Fprintln(leftTop, "*")
		fmt.Fprintln(leftTop, "foo-awssqssource")
		fmt.Fprintln(leftTop, "foo-awss3source")
		fmt.Fprintln(leftTop, "foo-kafkasource")

		if _, err := g.SetCurrentView("source"); err != nil {
			return err
		}
	}

	if leftBottom, err := g.SetView("target", 0, maxY/2, int(0.1*float32(maxX)), maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		leftBottom.Title = "Target (Ctrl+T)"
		// leftBottom.Highlight = true
		leftBottom.SelBgColor = gocui.ColorGreen
		fmt.Fprintln(leftBottom, "*")
		fmt.Fprintln(leftBottom, "foo-solacetarget")
		fmt.Fprintln(leftBottom, "foo-sockeye")
	}

	if sourceEvent, err := g.SetView("sourceEvent", int(0.1*float32(maxX)), 0, maxX/2, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		sourceEvent.Title = "Produced event sample"
		sourceEvent.SelBgColor = gocui.ColorGreen
	}

	if targetEvent, err := g.SetView("targetEvent", int(0.1*float32(maxX)), maxY/2, maxX/2, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		targetEvent.Title = "Consumed event sample"

		transformationMain, err := g.SetView("transformationMain", maxX/2+1, 0, maxX-1, maxY-1)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
		transformationMain.Title = "Transformation (Ctrl+R)"
		// transformationMain.Editable = true
		// transformationMain.Wrap = true

		fmt.Fprintln(transformationMain, "Context:")
		_, vy := transformationMain.Size()
		fmt.Fprintf(transformationMain, "%sData:", strings.Repeat("\n", vy/2))
	}

	if transformationCtx, err := g.SetView("transformationContext", maxX/2+5, 2, maxX-2, int(0.5*float32(maxY))); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		transformationCtx.Frame = false
		transformationCtx.Editable = true
	}

	if transformationData, err := g.SetView("transformationData", maxX/2+5, int(0.5*float32(maxY))+2, maxX-2, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		transformationData.Frame = false
		transformationData.Editable = true
	}
	return nil
}

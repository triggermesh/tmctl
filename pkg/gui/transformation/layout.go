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

type layout struct {
	sources     *gocui.View
	sourcesSide *gocui.View

	targets     *gocui.View
	targetsSide *gocui.View

	transformationContext *gocui.View
	transformationData    *gocui.View
}

func NewLayout() *layout {
	return &layout{}
}

func (l *layout) draw(g *gocui.Gui) error {
	var err error
	maxX, maxY := g.Size()

	l.sources, err = sourcesView(g, 0, 0, int(0.15*float32(maxX)), maxY/2-1)
	if err != nil {
		return err
	}
	l.targets, err = targetsView(g, 0, maxY/2, int(0.15*float32(maxX)), maxY-1)
	if err != nil {
		return err
	}

	l.sourcesSide = genericViewOrPanic(g, "Event sample", "sourceEvent", int(0.15*float32(maxX)), 0, maxX/2, maxY/2-1)
	l.targetsSide = genericViewOrPanic(g, "Event sample", "targetEvent", int(0.15*float32(maxX)), maxY/2, maxX/2, maxY-1)
	transformation := genericViewOrPanic(g, "Transformation (Ctrl+R)", "transformation", maxX/2+1, 0, maxX-1, maxY-1)

	fmt.Fprintln(transformation, "Context:")
	_, vy := transformation.Size()
	fmt.Fprintf(transformation, "%sData:", strings.Repeat("\n", vy/2))

	l.transformationContext = genericViewOrPanic(g, "Transformation Context", "transformationContext", maxX/2+5, 2, maxX-2, int(0.5*float32(maxY)))
	l.transformationContext.Frame = false
	l.transformationContext.Editable = true

	l.transformationData = genericViewOrPanic(g, "Transformation Data", "transformationData", maxX/2+5, int(0.5*float32(maxY))+2, maxX-2, maxY-2)
	l.transformationData.Frame = false
	l.transformationData.Editable = true

	return nil
}

func sourcesView(g *gocui.Gui, x1, y1, x2, y2 int) (*gocui.View, error) {
	sources, err := g.SetView("sources", x1, y1, x2, y2)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return nil, err
		}
		sources.Title = "Source (Ctrl+S)"
		sources.Highlight = true
		sources.SelBgColor = gocui.ColorGreen

		fmt.Fprintln(sources, "*")

		if _, err := g.SetCurrentView("sources"); err != nil {
			return nil, err
		}
	}
	return sources, nil
}

func targetsView(g *gocui.Gui, x1, y1, x2, y2 int) (*gocui.View, error) {
	targets, err := g.SetView("targets", x1, y1, x2, y2)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return nil, err
		}
		targets.Title = "Target (Ctrl+T)"
		targets.SelBgColor = gocui.ColorGreen

		fmt.Fprintln(targets, "*")

	}
	return targets, nil
}

func genericViewOrPanic(g *gocui.Gui, title, name string, x1, y1, x2, y2 int) *gocui.View {
	v, err := g.SetView(name, x1, y1, x2, y2)
	if err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
		}
		v.Title = title
		v.SelBgColor = gocui.ColorGreen
		v.Wrap = true
	}
	return v
}

func popOperationsView(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	ops := genericViewOrPanic(g, "Operation", "transformationOperation", maxX/2-30, maxY/2-5, maxX/2+30, maxY/2+1)
	fmt.Fprintf(ops, "-add\n-delete\n-shift\n-store\n-parse\n")
	if _, err := g.SetCurrentView(ops.Name()); err != nil {
		return err
	}
	ops.Highlight = true
	ops.SelBgColor = gocui.ColorGreen
	return nil
}

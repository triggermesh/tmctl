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

	"github.com/jroimartin/gocui"
)

type layout struct {
	sources     *gocui.View
	sourcesSide *gocui.View

	targets     *gocui.View
	targetsSide *gocui.View

	transformation *gocui.View
}

func NewLayout() *layout {
	return &layout{}
}

func (l *layout) draw(g *gocui.Gui) error {
	var err error
	maxX, maxY := g.Size()

	l.sources, err = sourcesView(g, 0, 0, int(0.17*float32(maxX)), maxY/2-1)
	if err != nil {
		return err
	}
	l.targets, err = targetsView(g, 0, maxY/2, int(0.17*float32(maxX)), maxY-1)
	if err != nil {
		return err
	}

	l.sourcesSide = genericViewOrPanic(g, "Produced event", "sourceEvent", int(0.17*float32(maxX)), 0, maxX/2, maxY/2-1)
	l.targetsSide = genericViewOrPanic(g, "Expected event", "targetEvent", int(0.17*float32(maxX)), maxY/2, maxX/2, maxY-1)
	l.transformation = genericViewOrPanic(g, "Transformation (Ctrl+E)", "transformation", maxX/2+1, 0, maxX-1, int(0.8*float32(maxY)))
	l.transformation.Editable = true

	help := genericViewOrPanic(g, "Help", "help", maxX/2+1, int(0.8*float32(maxY))+1, maxX-1, maxY-1)
	help.Clear()
	fmt.Fprintln(help, "Ctrl+W - Add wipe event operation\t\t\tCtrl+C - Close active window")
	fmt.Fprintln(help, "Ctrl+R - Reset the transformation\t\t\tCtrl+S - Save and exit")
	return nil
}

func sourcesView(g *gocui.Gui, x1, y1, x2, y2 int) (*gocui.View, error) {
	sources, err := g.SetView("sources", x1, y1, x2, y2)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return nil, err
		}
		sources.Title = "Source (Ctrl+F)"
		sources.Highlight = true
		sources.Wrap = true
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
		targets.Wrap = true
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

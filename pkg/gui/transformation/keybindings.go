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

func applyKeybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	}); err != nil {
		return err
	}

	// switch to Sources
	if err := g.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == "source" {
			return nil
		}
		newV, err := g.SetCurrentView("source")
		if err != nil {
			return err
		}
		newV.Highlight = true
		v.Highlight = false
		return nil
	}); err != nil {
		return err
	}

	// switch to Targets
	if err := g.SetKeybinding("", gocui.KeyCtrlT, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == "target" {
			return nil
		}
		newV, err := g.SetCurrentView("target")
		if err != nil {
			return err
		}
		newV.Highlight = true
		v.Highlight = false
		return nil
	}); err != nil {
		return err
	}

	// switch to Transformation
	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if v.Name() == "transformationContext" || v.Name() == "transformationData" {
			return nil
		}
		if _, err := g.SetCurrentView("transformationContext"); err != nil {
			return err
		}
		v.Highlight = false
		return nil
	}); err != nil {
		return err
	}

	// select source - Up
	if err := g.SetKeybinding("source", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	// select source - Down
	if err := g.SetKeybinding("source", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	// switch to event
	if err := g.SetKeybinding("source", gocui.KeyArrowRight, gocui.ModNone, sourceCursorRight); err != nil {
		return err
	}
	// switch back to sources
	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowLeft, gocui.ModNone, sourceEventCursorLeft); err != nil {
		return err
	}

	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("sourceEvent", gocui.KeyEnter, gocui.ModNone, addOperation); err != nil {
		return err
	}
	if err := g.SetKeybinding("operation", gocui.KeyEnter, gocui.ModNone, delOperationView); err != nil {
		return err
	}

	// select target - Up
	if err := g.SetKeybinding("target", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	// select target - Down
	if err := g.SetKeybinding("target", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}

	// press Enter
	if err := g.SetKeybinding("source", gocui.KeyEnter, gocui.ModNone, nextView); err != nil {
		return err
	}
	// press Enter
	if err := g.SetKeybinding("target", gocui.KeyEnter, gocui.ModNone, nextView); err != nil {
		return err
	}

	if err := g.SetKeybinding("transformationContext", gocui.KeyCtrlR, gocui.ModNone, transformationNextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("transformationData", gocui.KeyCtrlR, gocui.ModNone, transformationNextView); err != nil {
		return err
	}
	return nil
}

func sourceCursorRight(g *gocui.Gui, v *gocui.View) error {
	newView, err := g.SetCurrentView("sourceEvent")
	if err != nil {
		return err
	}
	newView.Highlight = true
	v.Highlight = false
	return nil
}

func sourceEventCursorLeft(g *gocui.Gui, v *gocui.View) error {
	newView, err := g.SetCurrentView("source")
	if err != nil {
		return err
	}
	newView.Highlight = true
	v.Highlight = false
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if l, err := v.Line(cy + 1); err != nil || l == "" {
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}
	return getSelection(g, v)
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if l, err := v.Line(cy - 1); err != nil || l == "" {
		return nil
	}
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return getSelection(g, v)
}

func getSelection(g *gocui.Gui, v *gocui.View) error {
	var schemaView *gocui.View
	var err error

	switch v.Name() {
	case "source":
		if schemaView, err = g.View("sourceEvent"); err != nil {
			return err
		}
	case "target":
		if schemaView, err = g.View("targetEvent"); err != nil {
			return err
		}
	default:
		return nil
	}

	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return err
	}

	mockEvent := ""

	switch line {
	case "foo-awssqssource":
		mockEvent = `{
			"specversion" : "1.0",
			"type" : "custom-data",
			"source" : "awssqssource",
			"subject" : "123",
			"id" : "A234-1234-1234",
			"time" : "2018-04-05T17:31:00Z",
			"datacontenttype" : "application/json",
			"data" : {
					"id": 1,
					"first_name": "Nikolia",
					"last_name": "Mee",
					"email": "nmee0@wix.com",
					"gender": "Female",
					"ip_address": "67.17.181.181"
				}
		}`
	case "foo-awss3source":
		mockEvent = `{
			"specversion" : "1.0",
			"type" : "custom-data",
			"source" : "awss3source",
			"subject" : "123",
			"id" : "A234-1234-1234",
			"time" : "2018-04-05T17:31:00Z",
			"datacontenttype" : "application/json",
			"data" : {
					"id": 2,
					"first_name": "Ellis",
					"last_name": "Larmuth",
					"email": "elarmuth1@mtv.com",
					"gender": "Male",
					"ip_address": "23.13.105.133"
				}
		}`
	case "foo-kafkasource":
		mockEvent = `{
			"specversion" : "1.0",
			"type" : "custom-data",
			"source" : "kafkasource",
			"subject" : "123",
			"id" : "A234-1234-1234",
			"time" : "2018-04-05T17:31:00Z",
			"datacontenttype" : "application/json",
			"data" : {
					"id": 3,
					"first_name": "Cindra",
					"last_name": "Henryson",
					"email": "chenryson2@simplemachines.org",
					"gender": "Female",
					"ip_address": "222.81.4.118"
			  	}
		}`
	case "foo-solacetarget":
		mockEvent = `{
			"specversion" : "1.0",
			"type" : "custom-data",
			"source" : "solacetarget",
			"subject" : "123",
			"id" : "A234-1234-1234",
			"time" : "2018-04-05T17:31:00Z",
			"datacontenttype" : "application/json",
			"data" : {
					"name": "John",
					"ID": "18.98.36.4"
			  	}
		}`
	case "foo-sockeye":
		mockEvent = `-`
	}

	schemaView.Clear()
	fmt.Fprintf(schemaView, "%s", mockEvent)

	return nil
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "source":
		newView, err := g.SetCurrentView("target")
		if err != nil {
			return err
		}
		newView.Highlight = true
		v.Highlight = false
	case "target":
		_, err := g.SetCurrentView("transformationContext")
		if err != nil {
			return err
		}
		v.Highlight = false
	}
	return nil
}

func transformationNextView(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "transformationContext":
		if _, err := g.SetCurrentView("transformationData"); err != nil {
			return err
		}
	case "transformationData":
		if _, err := g.SetCurrentView("transformationContext"); err != nil {
			return err
		}
	}
	return nil
}

func addOperation(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("operation", maxX/2-30, maxY/2-5, maxX/2+30, maxY/2+1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintf(v, "-add\n-delete\n-shift\n-store\n-parse\n")
		if _, err := g.SetCurrentView("operation"); err != nil {
			return err
		}
	}
	return nil
}

func delOperationView(g *gocui.Gui, v *gocui.View) error {
	if err := g.DeleteView("operation"); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("sourceEvent"); err != nil {
		return err
	}
	return nil
}

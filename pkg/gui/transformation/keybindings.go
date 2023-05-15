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
	"strings"

	"github.com/jroimartin/gocui"
)

type keybindingHandler struct {
	signals chan signal
}

type signal struct {
	origin string
	line   string
}

var jsonSkipLines = []string{"{", "}", "[", "]", "],", "},"}

func NewKeybindingHandler() *keybindingHandler {
	return &keybindingHandler{
		signals: make(chan signal),
	}
}

func (h *keybindingHandler) Apply(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	}); err != nil {
		return err
	}

	// switch to Sources
	if err := g.SetKeybinding("", gocui.KeyCtrlS, gocui.ModNone, h.switchToSources); err != nil {
		return err
	}
	// switch to Targets
	if err := g.SetKeybinding("", gocui.KeyCtrlT, gocui.ModNone, h.switchToTargets); err != nil {
		return err
	}
	// switch to Transformation
	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, h.switchToTransformation); err != nil {
		return err
	}

	// select source - Up
	if err := g.SetKeybinding("sources", gocui.KeyArrowUp, gocui.ModNone, h.cursorUpWithSignal); err != nil {
		return err
	}
	// select source - Down
	if err := g.SetKeybinding("sources", gocui.KeyArrowDown, gocui.ModNone, h.cursorDownWithSignal); err != nil {
		return err
	}
	// switch to event
	if err := g.SetKeybinding("sources", gocui.KeyArrowRight, gocui.ModNone, h.sourceCursorRight); err != nil {
		return err
	}
	// press Enter
	if err := g.SetKeybinding("sources", gocui.KeyEnter, gocui.ModNone, h.nextView); err != nil {
		return err
	}
	// switch back to sources
	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowLeft, gocui.ModNone, h.sourceEventCursorLeft); err != nil {
		return err
	}
	// walk through the source schema
	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowUp, gocui.ModNone, h.cursorUpWithSignal); err != nil {
		return err
	}
	// walk through the source schema
	if err := g.SetKeybinding("sourceEvent", gocui.KeyArrowDown, gocui.ModNone, h.cursorDownWithSignal); err != nil {
		return err
	}
	// operation popup
	if err := g.SetKeybinding("sourceEvent", gocui.KeyEnter, gocui.ModNone, popOperationsView); err != nil {
		return err
	}
	// operation cancel
	if err := g.SetKeybinding("transformationOperation", gocui.KeyEsc, gocui.ModNone, h.cancelOperationView); err != nil {
		return err
	}
	// select operations
	if err := g.SetKeybinding("transformationOperation", gocui.KeyArrowUp, gocui.ModNone, h.cursorUp); err != nil {
		return err
	}
	// select operations
	if err := g.SetKeybinding("transformationOperation", gocui.KeyArrowDown, gocui.ModNone, h.cursorDown); err != nil {
		return err
	}
	// select operations
	if err := g.SetKeybinding("transformationOperation", gocui.KeyEnter, gocui.ModNone, h.selectOperation); err != nil {
		return err
	}

	// select target - Up
	if err := g.SetKeybinding("targets", gocui.KeyArrowUp, gocui.ModNone, h.cursorUpWithSignal); err != nil {
		return err
	}
	// select target - Down
	if err := g.SetKeybinding("targets", gocui.KeyArrowDown, gocui.ModNone, h.cursorDownWithSignal); err != nil {
		return err
	}
	// press Enter
	if err := g.SetKeybinding("targets", gocui.KeyEnter, gocui.ModNone, h.nextView); err != nil {
		return err
	}

	if err := g.SetKeybinding("transformationContext", gocui.KeyCtrlR, gocui.ModNone, h.transformationNextView); err != nil {
		return err
	}
	if err := g.SetKeybinding("transformationData", gocui.KeyCtrlR, gocui.ModNone, h.transformationNextView); err != nil {
		return err
	}
	return nil
}

func (h *keybindingHandler) selectOperation(g *gocui.Gui, v *gocui.View) error {
	_, cy := v.Cursor()
	operation, err := v.Line(cy)
	if err != nil {
		return err
	}

	switch operation {
	case "-delete", "-parse":
		if err := h.sendSignal(g); err != nil {
			return err
		}
		if err := g.DeleteView("transformationOperation"); err != nil {
			return err
		}
		if _, err := g.SetCurrentView("sourceEvent"); err != nil {
			return err
		}
	case "-add", "-store", "-shift":
		if err := popInputValueView(operation+":", g, v); err != nil {
			return err
		}
		if err := g.DeleteView("transformationOperation"); err != nil {
			return err
		}
		// read value
		if err := g.SetKeybinding("operationValue", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
			h.signals <- signal{
				origin: "transformationOperation",
				line:   operation + ":" + strings.TrimSpace(v.Buffer()),
			}
			if err := g.DeleteView("operationValue"); err != nil {
				return err
			}
			if _, err := g.SetCurrentView("sourceEvent"); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (h *keybindingHandler) switchToSources(g *gocui.Gui, v *gocui.View) error {
	if v.Name() == "sources" {
		return nil
	}
	newV, err := g.SetCurrentView("sources")
	if err != nil {
		return err
	}
	newV.Highlight = true
	v.Highlight = false
	return h.sendSignal(g)
}

func (h *keybindingHandler) switchToTargets(g *gocui.Gui, v *gocui.View) error {
	if v.Name() == "targets" {
		return nil
	}
	newV, err := g.SetCurrentView("targets")
	if err != nil {
		return err
	}
	newV.Highlight = true
	v.Highlight = false
	return h.sendSignal(g)
}

func (h *keybindingHandler) switchToTransformation(g *gocui.Gui, v *gocui.View) error {
	if v.Name() == "transformationContext" || v.Name() == "transformationData" {
		return nil
	}
	if _, err := g.SetCurrentView("transformationContext"); err != nil {
		return err
	}
	v.Highlight = false
	return h.sendSignal(g)
}

func (h *keybindingHandler) sourceCursorRight(g *gocui.Gui, v *gocui.View) error {
	newView, err := g.SetCurrentView("sourceEvent")
	if err != nil {
		return err
	}
	newView.Highlight = true
	v.Highlight = false
	return h.sendSignal(g)
}

func (h *keybindingHandler) sourceEventCursorLeft(g *gocui.Gui, v *gocui.View) error {
	newView, err := g.SetCurrentView("sources")
	if err != nil {
		return err
	}
	newView.Highlight = true
	v.Highlight = false
	v.SetCursor(0, 0)
	return h.sendSignal(g)
}

func (h *keybindingHandler) cursorDown(g *gocui.Gui, v *gocui.View) error {
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
	return nil
}

func (h *keybindingHandler) cursorUp(g *gocui.Gui, v *gocui.View) error {
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
	return nil
}

func isUseless(s string) bool {
	// s = strings.TrimSpace(s)
	// for _, v := range jsonSkipLines {
	// 	if s == v {
	// 		return true
	// 	}
	// }
	return false
}

func (h *keybindingHandler) cursorDownWithSignal(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	dy := cy + 1
	if l, err := v.Line(dy); err != nil || l == "" {
		return nil
	} else if strings.HasSuffix(l, ":") || isUseless(l) {
		v.SetCursor(cx, dy)
		return h.cursorDownWithSignal(g, v)
	}
	if err := v.SetCursor(cx, dy); err != nil {
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}
	return h.sendSignal(g)
}

func (h *keybindingHandler) cursorUpWithSignal(g *gocui.Gui, v *gocui.View) error {
	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	dy := cy - 1
	if l, err := v.Line(dy); err != nil || l == "" {
		return nil
	} else if strings.HasSuffix(l, ":") || isUseless(l) {
		v.SetCursor(cx, dy)
		return h.cursorUpWithSignal(g, v)
	}
	if err := v.SetCursor(cx, dy); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return h.sendSignal(g)
}

func (h *keybindingHandler) nextView(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "sources":
		newView, err := g.SetCurrentView("targets")
		if err != nil {
			return err
		}
		newView.Highlight = true
		v.Highlight = false
	case "targets":
		_, err := g.SetCurrentView("transformationContext")
		if err != nil {
			return err
		}
		v.Highlight = false
	}
	return h.sendSignal(g)
}

func (h *keybindingHandler) transformationNextView(g *gocui.Gui, v *gocui.View) error {
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
	return h.sendSignal(g)
}

func (h *keybindingHandler) cancelOperationView(g *gocui.Gui, v *gocui.View) error {
	if err := g.DeleteView("transformationOperation"); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("sourceEvent"); err != nil {
		return err
	}
	return nil
}

func (h *keybindingHandler) sendSignal(g *gocui.Gui) error {
	v := g.CurrentView()
	_, cy := v.Cursor()
	line, err := v.Line(cy)
	if err != nil {
		return err
	}
	h.signals <- signal{
		origin: v.Name(),
		line:   line,
	}
	return nil
}

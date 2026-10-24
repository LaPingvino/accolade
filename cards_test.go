package main

import (
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
)

// The card view: the scenes as cards under their section, dragged onto
// another card it moves its scene, a synopsis written on a card goes in the
// script, tapped it opens the scene in the editor.
func TestCards(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(900, 600))
	w.textEditor.SetText("# ACT ONE\n\nINT. BARN - DAY\n\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n\nEXT. FIELD - DAY\n\nWind.\n")
	w.toggleCards()
	c := w.cards
	if !w.cardsVisible || len(c.list.Objects) != 2 {
		t.Fatalf("groups: %d objects", len(c.list.Objects))
	}
	if _, ok := c.list.Objects[0].(*sectionHeader); !ok {
		t.Fatalf("no header for ACT ONE: %T", c.list.Objects[0])
	}
	grid := c.list.Objects[1].(*fyne.Container)
	if len(grid.Objects) != 3 { // two scenes and the new scene card
		t.Fatalf("cards: %d", len(grid.Objects))
	}
	if _, ok := grid.Objects[2].(*addCard); !ok {
		t.Errorf("the group does not end with the new scene card: %T", grid.Objects[2])
	}
	if p := os.Getenv("CARDS_PNG"); p != "" {
		f, _ := os.Create(p)
		_ = png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}

	// the barn dropped on the right half of the field: after it
	barn, field := grid.Objects[0].(*card), grid.Objects[1].(*card)
	d := fyne.CurrentApp().Driver()
	to := d.AbsolutePositionForObject(field).Add(fyne.NewPos(field.Size().Width*3/4, field.Size().Height/2))
	m := barn.middle()
	barn.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(to.X-m.X, to.Y-m.Y)})
	barn.DragEnd()
	if got := w.textEditor.Text(); got != "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n" {
		t.Fatalf("dragged: %q", got)
	}

	// a synopsis written on the field's card, now the first
	grid = c.list.Objects[1].(*fyne.Container)
	synopsisOf(t, grid.Objects[0].(*card)).SetText("The wind turns")
	if got := w.textEditor.Text(); got != "# ACT ONE\n\nEXT. FIELD - DAY\n= The wind turns\n\nWind.\n\nINT. BARN - DAY\n\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n" {
		t.Fatalf("synopsis: %q", got)
	}

	grid = c.list.Objects[1].(*fyne.Container)
	grid.Objects[1].(*card).Tapped(nil) // the barn, now second
	if w.cardsVisible {
		t.Error("still the cards")
	}
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 7 {
		t.Errorf("opened at line %d", line)
	}
}

// synopsisOf is a card's synopsis entry.
func synopsisOf(t *testing.T, k *card) *synopsisEntry {
	var found *synopsisEntry
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if s, ok := o.(*synopsisEntry); ok {
			found = s
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, ch := range c.Objects {
				walk(ch)
			}
		}
	}
	walk(k.body)
	if found == nil {
		t.Fatal("the card has no synopsis entry")
	}
	return found
}

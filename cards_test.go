package main

import (
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The card view: a card for each section and scene, dragged one place on
// it moves its scene, tapped it opens the scene in the editor.
func TestCards(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(900, 600))
	w.textEditor.SetText("# ACT ONE\n\nINT. BARN - DAY\n\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n\nEXT. FIELD - DAY\n\nWind.\n")
	w.toggleCards()
	c := w.cards
	if !w.cardsVisible || len(c.grid.Objects) != 3 {
		t.Fatalf("cards: %d", len(c.grid.Objects))
	}
	if p := os.Getenv("CARDS_PNG"); p != "" {
		f, _ := os.Create(p)
		_ = png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	cell := cardSize.Width + theme.Padding()
	k := c.grid.Objects[1].(*card) // the barn
	k.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: cell}})
	k.DragEnd()
	if got := w.textEditor.Text(); got != "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n" {
		t.Fatalf("dragged: %q", got)
	}
	c.grid.Objects[2].(*card).Tapped(nil) // the barn, now third
	if w.cardsVisible {
		t.Error("still the cards")
	}
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 6 {
		t.Errorf("opened at line %d", line)
	}
}

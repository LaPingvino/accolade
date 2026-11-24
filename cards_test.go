package main

import (
	"image/png"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// The card view: the scenes as cards under their section, dragged onto
// another card it moves its scene, a synopsis written on a card goes in the
// script, tapped it opens the scene in the editor.
func TestCards(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(900, 600))
	w.textEditor.SetText("# ACT ONE\n\nINT. BARN - DAY\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n\nEXT. FIELD - DAY\n\nWind.\n")
	w.toggleCards()
	c := w.cards
	if !w.cardsShown() || len(c.list.Objects) != 2 {
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
	if got := w.textEditor.Text(); got != "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n" {
		t.Fatalf("dragged: %q", got)
	}

	// a synopsis written on the field's card, now the first
	grid = c.list.Objects[1].(*fyne.Container)
	synopsisOf(t, grid.Objects[0].(*card)).SetText("The wind turns")
	if got := w.textEditor.Text(); got != "# ACT ONE\n\nEXT. FIELD - DAY\n= The wind turns\n\nWind.\n\nINT. BARN - DAY\n= Anna finds the tin.\n\nRain on the roof. Anna lifts the lid of the biscuit tin.\n" {
		t.Fatalf("synopsis: %q", got)
	}

	grid = c.list.Objects[1].(*fyne.Container)
	grid.Objects[1].(*card).Tapped(nil) // the barn, now second
	if w.cardsShown() || !w.mid.has("cards") {
		t.Error("still the cards, or their tab closed")
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

// cardsOf are the cards view's cards of type T, in order.
func cardsOf[T fyne.CanvasObject](c *cardsView) []T {
	var out []T
	for _, g := range c.list.Objects {
		if k, ok := g.(T); ok {
			out = append(out, k)
		}
		if grid, ok := g.(*fyne.Container); ok {
			for _, o := range grid.Objects {
				if k, ok := o.(T); ok {
					out = append(out, k)
				}
			}
		}
	}
	return out
}

// dragOnto drags d (its middle) onto target, at fx of its width.
func dragOnto(d *dragged, target fyne.CanvasObject, fx float32) {
	to := fyne.CurrentApp().Driver().AbsolutePositionForObject(target).Add(fyne.NewPos(target.Size().Width*fx, target.Size().Height/2))
	m := d.middle()
	d.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(to.X-m.X, to.Y-m.Y)})
	d.DragEnd()
}

// A synopsis card is dragged as a scene's card is (the same move of
// units as Alt+Up and Shift+Alt+Down): after another beat, into another
// act's start. A section with no scenes ends with a compact New Scene row,
// not an empty card; one with scenes with the card.
func TestSynopsisCardsDrag(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(900, 700))
	w.textEditor.SetText("# Act One\n= What it does.\n\n= Beat a.\n\n= Beat b.\n\n# Act Two\n= Two's.\n\n= Beat c.\n\nINT. BOAT - NIGHT\n\nWater.\n")
	w.toggleCards()
	c := w.cards
	if rows, adds := cardsOf[*addRow](c), cardsOf[*addCard](c); len(rows) != 1 || len(adds) != 1 || rows[0].at != 1 {
		t.Fatalf("Act One without scenes: %d rows (at %v), Act Two with one: %d cards", len(rows), rows, len(adds))
	}
	syn := cardsOf[*synopsisCard](c)
	if len(syn) != 3 {
		t.Fatalf("%d synopsis cards", len(syn))
	}
	dragOnto(&syn[0].dragged, syn[1], 0.75) // Beat a after Beat b
	if got, want := w.textEditor.Text(), "# Act One\n= What it does.\n\n= Beat b.\n\n= Beat a.\n\n# Act Two\n= Two's.\n\n= Beat c.\n\nINT. BOAT - NIGHT\n\nWater.\n"; got != want {
		t.Fatalf("after Beat b:\n%q\nwant\n%q", got, want)
	}
	syn = cardsOf[*synopsisCard](c)
	headers := cardsOf[*sectionHeader](c)
	dragOnto(&syn[2].dragged, headers[0], 0.5) // Beat c first in Act One, after its own
	if got, want := w.textEditor.Text(), "# Act One\n= What it does.\n\n= Beat c.\n\n= Beat b.\n\n= Beat a.\n\n# Act Two\n= Two's.\n\nINT. BOAT - NIGHT\n\nWater.\n"; got != want {
		t.Fatalf("into Act One:\n%q\nwant\n%q", got, want)
	}
	w.textEditor.Undo()
	if got := w.textEditor.Text(); !strings.Contains(got, "= Two's.\n\n= Beat c.") {
		t.Errorf("one undo step: %q", got)
	}
	// on the boat scene's right half: into the scene, at its end
	syn = cardsOf[*synopsisCard](c)
	boat := cardsOf[*card](c)[0]
	dragOnto(&syn[0].dragged, boat, 0.75)
	if got := w.textEditor.Text(); !strings.HasSuffix(got, "INT. BOAT - NIGHT\n\nWater.\n\n= Beat b.\n") {
		t.Errorf("into the boat scene:\n%q", got)
	}
	comboShot(t, w, "structure-drag-cards")
}

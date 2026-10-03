package editor

import (
	"testing"

	"fyne.io/fyne/v2/widget"
)

func bg(c widget.TextGridCell) bool {
	return c.Style != nil && c.Style.BackgroundColor() != nil
}

func TestHighlights(t *testing.T) {
	e := newEditor(t, "rain rain rain", 40)
	e.FocusLost() // no cursor cell
	e.SetHighlights([][2]int{{0, 4}, {10, 14}})
	cells := e.grid.Rows[0].Cells
	for i, want := range []bool{true, true, true, true, false, false, false, false, false, false, true, true, true, true} {
		if bg(cells[i]) != want {
			t.Errorf("cell %d highlighted %v, want %v", i, bg(cells[i]), want)
		}
	}
	// the selection wins over a highlight
	e.Buffer().Select(2, 6)
	e.changed(false)
	sel := e.grid.Rows[0].Cells[2].Style.BackgroundColor()
	hl := e.grid.Rows[0].Cells[0].Style.BackgroundColor()
	if sel == hl {
		t.Error("selected cell drawn as a highlight")
	}
	e.SetHighlights(nil)
	if bg(e.grid.Rows[0].Cells[12]) {
		t.Error("highlights not cleared")
	}
}

func TestSyntaxStyles(t *testing.T) {
	e := newEditor(t, "INT. BARN - DAY\n\nJOHN\n(beat)\nHi.\n\nRain.", 40)
	e.FocusLost()
	if e.grid.Rows[0].Cells[0].Style != nil {
		t.Fatal("styled without Syntax")
	}
	e.Syntax = true
	e.changed(false)
	style := func(row int) widget.TextGridStyle { return e.grid.Rows[row].Cells[0].Style }
	if s := style(0); s == nil || !s.Style().Bold {
		t.Error("scene heading not bold")
	}
	if s := style(2); s == nil || !s.Style().Bold || s.TextColor() == nil {
		t.Error("character not bold and coloured")
	}
	if s := style(3); s == nil || !s.Style().Italic {
		t.Error("parenthetical not italic")
	}
	if style(4) != nil || style(6) != nil {
		t.Error("dialogue and action should be plain")
	}
}

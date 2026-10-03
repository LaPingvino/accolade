package editor

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

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

func TestLineNumbers(t *testing.T) {
	text := "INT. BARN\n\nRain hammers the roof of the barn.\n" + strings.Repeat("x\n", 8)
	e := newEditor(t, text, 20)
	e.FocusLost()
	e.SetLineNumbers(true)
	rows := gridText(e)
	// 12 lines: two digits and a space; the long line wraps with an empty gutter
	want := []string{" 1 INT. BARN", " 2", " 3 Rain hammers the", "   roof of the barn."}
	for i, w := range want {
		if rows[i] != w {
			t.Errorf("row %d = %q, want %q", i, rows[i], w)
		}
	}
	if rows[len(rows)-1] != "12" {
		t.Errorf("last row %q", rows[len(rows)-1])
	}
	// clicks and the cursor account for the gutter
	e.Tapped(&fyne.PointEvent{Position: at(e, 3+4.2, 0)})
	if e.Buffer().Cursor() != 4 {
		t.Errorf("tap after the gutter: cursor %d, want 4", e.Buffer().Cursor())
	}
	if p := e.CursorPosition(); p.X != 7*e.cellWidth() {
		t.Errorf("cursor x %v, want 7 cells", p.X)
	}
	e.SetLineNumbers(false)
	if gridText(e)[0] != "INT. BARN" {
		t.Errorf("without line numbers: %q", gridText(e)[0])
	}
}

func TestSyntaxColoursFollowTheTheme(t *testing.T) {
	e := newEditor(t, "INT. BARN - DAY\n\nRain.", 40)
	e.FocusLost()
	e.Syntax = true
	e.changed(false)
	a := fyne.CurrentApp()
	for _, th := range []fyne.Theme{theme.LightTheme(), theme.DarkTheme()} {
		a.Settings().SetTheme(th)
		e.Refresh()
		want := th.Color(theme.ColorNameForeground, a.Settings().ThemeVariant())
		if got := e.grid.Rows[0].Cells[0].Style.TextColor(); got != want {
			t.Errorf("scene heading colour %v, want the theme's foreground %v", got, want)
		}
	}
}

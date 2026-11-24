package editor

import (
	"image/color"
	"reflect"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
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
	if s := style(2); s == nil || !s.Style().Bold {
		t.Error("character not bold")
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
	e := newEditor(t, "INT. BARN - DAY\n\nJOHN\nHello.", 40)
	e.FocusLost()
	e.Syntax = true
	e.changed(false)
	a := fyne.CurrentApp()
	for _, th := range []fyne.Theme{theme.LightTheme(), theme.DarkTheme()} {
		a.Settings().SetTheme(th)
		e.Refresh()
		want := th.Color(theme.ColorNameForeground, a.Settings().ThemeVariant())
		if got := e.grid.Rows[2].Cells[0].Style.TextColor(); got != want {
			t.Errorf("character name colour %v, want the theme's foreground %v", got, want)
		}
	}
}

// Markdown: headings bold, emphasis inside a line, and no screenplay
// layout even with a Format.
func TestMarkdownStyles(t *testing.T) {
	e := newEditor(t, "# One\n\nSome **bold** and *it*.\n\nJOHN\nHi.", 40)
	e.FocusLost()
	e.Syntax, e.Markdown = true, true
	e.Format = func(k syntax.Kind) wrap.Format { return wrap.Format{Indent: 10} }
	e.changed(false)
	cell := func(row, col int) widget.TextGridStyle { return e.grid.Rows[row].Cells[col].Style }
	if s := cell(0, 2); s == nil || !s.Style().Bold {
		t.Error("heading not bold")
	}
	if cell(2, 0) != nil {
		t.Error("plain text styled")
	}
	if s := cell(2, 7); s == nil || !s.Style().Bold {
		t.Error("**bold** not bold")
	}
	if s := cell(2, 19); s == nil || !s.Style().Italic {
		t.Error("*it* not italic")
	}
	if s := cell(4, 0); s != nil && s.Style().Bold {
		t.Error("a name in capitals is a character cue in Markdown")
	}
	if r := e.layout.Rows[4]; r.Indent != 0 {
		t.Errorf("laid out as a screenplay: indent %d", r.Indent)
	}
}

// Inks draw a line's words in a colour, underlined; the other lines and
// columns as they are.
func TestInks(t *testing.T) {
	e := newEditor(t, "first line\nthe revolver on the table", 40)
	e.FocusLost()
	red := color.NRGBA{R: 200, A: 255}
	e.Inks = func() map[int][]Ink { return map[int][]Ink{1: {{Start: 4, End: 12, Color: red}}} }
	e.RedrawMarks()
	cells := e.grid.Rows[1].Cells
	for i, c := range cells {
		inked := c.Style != nil && c.Style.Style().Underline && c.Style.TextColor() == color.Color(red)
		if want := i >= 4 && i < 12; inked != want {
			t.Errorf("cell %d (%q) inked %v", i, c.Rune, inked)
		}
	}
	for _, c := range e.grid.Rows[0].Cells {
		if c.Style != nil && c.Style.Style().Underline {
			t.Error("the first line inked")
		}
	}
}

// A layout after an edit keeps the emphasis and direction of the lines the
// edit left alone: what it has is what a layout from nothing finds, edit
// after edit (a code fence opened and closed, lines joined and split,
// right-to-left lines among them).
func TestLayoutKeepsUntouchedLines(t *testing.T) {
	e := newEditor(t, "# One\n\nSome **bold** and *it*.\n\nשלום *עולם*\n\n- a `code` item\n\n> a [link](x) quote\n\nlast *line*", 40)
	e.FocusLost()
	e.Syntax, e.Markdown = true, true
	e.changed(false)
	edits := []struct {
		at          int
		del, insert string
	}{
		{7, "", "*more* "}, {0, "", "```\n"}, {0, "```\n", ""}, {20, "", "\n"}, {20, "\n", ""},
		{5, "\n", ""}, {5, "", "\n\nnew **line** here"}, {40, "", "```"}, {40, "```", ""},
	}
	for n, ed := range edits {
		text := e.buf.Text()
		if ed.del != "" && !strings.HasPrefix(string([]rune(text)[ed.at:]), ed.del) {
			t.Fatalf("edit %d: %q is not at %d in %q", n, ed.del, ed.at, text)
		}
		e.buf.Replace(ed.at, ed.at+len([]rune(ed.del)), ed.insert)
		e.changed(false)
		got := e.laid
		e.laid = laidOut{}
		want := e.layOut(got.key)
		if !reflect.DeepEqual(got.inline, want.inline) || !reflect.DeepEqual(got.rtl, want.rtl) ||
			!reflect.DeepEqual(got.lineStart, want.lineStart) || !reflect.DeepEqual(got.kinds, want.kinds) {
			t.Errorf("edit %d: kept %v %v, from nothing %v %v", n, got.inline, got.rtl, want.inline, want.rtl)
		}
		e.laid = got
	}
}

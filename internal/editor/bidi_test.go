package editor

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"slices"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

func TestRTL(t *testing.T) {
	for line, want := range map[string]bool{
		"مرحبا بالعالم": true, "שלום": true, "@سارة": true, "Hello": false, "  (یک) سلام": true,
		"‏John قال": true, "‎سلام": false, "John قال": false, "123": false,
	} {
		if RTL(line) != want {
			t.Errorf("RTL(%q) = %v", line, !want)
		}
	}
	f := mirrored(wrap.Format{Indent: 10, Width: 30}, 60)
	if f.Indent != 20 || f.Width != 30 || f.Align != 'R' {
		t.Errorf("mirrored dialogue: %+v", f)
	}
	if f := mirrored(wrap.Format{Indent: 40, Align: 'R'}, 60); f.Indent != 0 || f.Width != 20 || f.Align != 'L' {
		t.Errorf("mirrored transition: %+v", f)
	}
}

// An Arabic line ends at the page's right edge, an English one starts at
// its left.
func TestRTLLayout(t *testing.T) {
	e := newEditor(t, "Hello\n\nسلام", 40)
	e.Format = func(syntax.Kind) wrap.Format { return wrap.Format{Width: 30} }
	e.PageColumns = 30
	e.changed(false)
	if r := e.layout.Rows[0]; r.Indent != 0 {
		t.Errorf("English row indent %d", r.Indent)
	}
	if r := e.layout.Rows[2]; r.Indent != 26 {
		t.Errorf("Arabic row indent %d, want 26", r.Indent)
	}
}

// A right-to-left row is drawn mirrored: its first letter at the right,
// the cursor drawn where its letter is, a click there lands there, and
// the arrows move the way the text is drawn.
func TestRTLMirrored(t *testing.T) {
	e := newEditor(t, "abc\nשלום", 40)
	e.FocusGained()
	e.Buffer().SetCursor(5, false) // before ל
	e.changed(false)
	if b, _, _ := e.order(1); b.From != e.gutter() || len(b.Levels) != 4 || !b.RTL || !slices.Equal(b.Order, []int{3, 2, 1, 0}) {
		t.Errorf("right-to-left row ordered %+v", b)
	}
	if b, _, _ := e.order(0); b.reordered() || b.RTL {
		t.Error("left-to-right row reordered")
	}
	if c := e.visualCol(1, 1); c != 2 { // ל: second from the right of 4
		t.Errorf("ל drawn at column %d", c)
	}
	pos := e.CursorPosition()
	if off := e.offsetAt(fyne.NewPos(pos.X+e.cellWidth()-1, pos.Y+1)); off != 5 {
		t.Errorf("click on the cursor's right edge: offset %d, want 5", off)
	}
	key(e, fyne.KeyLeft) // left on screen: on in the text
	if c := e.Buffer().Cursor(); c != 6 {
		t.Errorf("left arrow: cursor %d, want 6", c)
	}
}

// A line with no direction of its own ("…") takes the script language's
// direction, laid out mirrored; the others keep their own, and turning
// the language's direction lays the text out again.
func TestLanguageRTLLayout(t *testing.T) {
	e := newEditor(t, "Hello\n\n…\n\nשלום", 40)
	e.Format = func(syntax.Kind) wrap.Format { return wrap.Format{Width: 30} }
	e.PageColumns = 30
	e.changed(false)
	if r := e.layout.Rows[2]; r.Indent != 0 {
		t.Errorf("ltr language: … at indent %d", r.Indent)
	}
	e.LanguageRTL = true
	e.changed(false)
	if r := e.layout.Rows[2]; r.Indent != 29 || !e.rowRTL(2) {
		t.Errorf("rtl language: … at indent %d (rtl %v), want 29", r.Indent, e.rowRTL(2))
	}
	if r := e.layout.Rows[0]; r.Indent != 0 || e.rowRTL(0) {
		t.Errorf("rtl language: Hello at indent %d", r.Indent)
	}
	if !e.LineRTL("2.") || e.LineRTL("Hello") || e.LineRTL("‎…") || RTL("…") {
		t.Error("LineRTL")
	}
}

// In a right-to-left row Latin words and numbers keep their order, with
// what stands between them as the bidi algorithm has it.
func TestLTRJoins(t *testing.T) {
	pieces := func(s string) []string {
		rs := []rune(s)
		var out []string
		for k, j := range bidiJoins(rs, bidiLevels(rs, true)) {
			if j && k > 0 {
				out[len(out)-1] += string(rs[k])
			} else {
				out = append(out, string(rs[k]))
			}
		}
		return out
	}
	for s, want := range map[string][]string{
		"השעה 23:40.":        {"השעה", " ", "23:40", "."},
		"של Pieter Bruegel,": {"של", " ", "Pieter Bruegel", ","},
		"דני (O.S.)":         {"דני", " ", "(", "O.S", ".", ")"},
		"אולי 7, אולי 70.":   {"אולי", " ", "7", ",", " ", "אולי", " ", "70", "."},
		"وَجَدَ":             {"وَجَدَ"},
	} {
		if got := pieces(s); strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%q: %q", s, got)
		}
	}
	if mirrorBracket("(") != ")" || mirrorBracket("x") != "x" {
		t.Error("brackets")
	}
}

// shown is row i as drawn: its cells' runes placed at their columns.
func shown(e *ScriptEditor, i int) string {
	b, cells, cols := e.order(i)
	if cells == nil {
		var row widget.TextGridRow
		row, cols, _, _ = e.rowCells(i)
		cells = row.Cells
	}
	at := drawnCols(cols, b)
	out := []rune(strings.Repeat(" ", at[len(cells)]+2))
	for k, c := range cells {
		if c.Rune != wideFill && cols[k].W > 0 && at[k] >= 0 && at[k] < len(out) {
			r := c.Rune
			if b.level(k)%2 == 1 {
				r = []rune(mirrorBracket(string(r)))[0]
			}
			out[at[k]] = r
		}
	}
	return strings.TrimRight(string(out[e.gutter():]), " ")
}

// The bidi algorithm orders each row: right-to-left words in a
// left-to-right line and left-to-right ones in a right-to-left line
// drawn the way they read, numbers whole.
func TestBidiRows(t *testing.T) {
	for line, want := range map[string]string{
		"Title: המגדל":                  "Title: לדגמה",
		"Title: המגדל 2 abc":            "Title: 2 לדגמה abc",
		"Title: سمسم 2026 draft":        "Title: 2026 مسمس draft",
		"השעה 23:40 של Pieter Bruegel.": ".Pieter Bruegel לש 23:40 העשה",
		"דני (O.S.)":                    "(.O.S) ינד",
		"Hello":                         "Hello",
	} {
		e := newEditor(t, line, 60)
		e.changed(false)
		if got := shown(e, 0); strings.TrimSpace(got) != want {
			t.Errorf("%q drawn %q, want %q", line, got, want)
		}
	}
}

// The cursor moves through the text as it is written and is drawn on its
// character wherever that is drawn; a click on a character's cell lands
// before it (on its reading side), and a selection lights the cells of
// what is selected.
func TestBidiCursorAndClicks(t *testing.T) {
	for _, line := range []string{"השעה 23:40 של Pieter Bruegel.", "Title: המגדל and more", "مرحبا 12:30 يا"} {
		e := newEditor(t, line, 60)
		e.FocusGained()
		e.changed(false)
		rs := []rune(line)
		b, _, cols := e.order(0)
		at := drawnCols(cols, b)
		for off := range rs {
			e.Buffer().SetCursor(off, false)
			e.changed(false)
			pos := e.CursorPosition()
			k := e.gutter() + off // one cell a character in these lines
			if want := float32(at[k]) * e.cellWidth(); pos.X != want {
				t.Errorf("%q: cursor at %d drawn at x %v, want %v", line, off, pos.X, want)
			}
			// a click on the half of the cell its reading starts from
			x := pos.X + e.cellWidth()*0.25
			if b.level(k)%2 == 1 {
				x = pos.X + e.cellWidth()*0.75
			}
			if got := e.offsetAt(fyne.NewPos(x, 1)); got != off {
				t.Errorf("%q: click before %q lands at %d, want %d", line, rs[off], got, off)
			}
		}
		// a selection of "23:40" / "המגדל": its cells, and only they, lit
		s := strings.IndexAny(line, "2ה")
		if s < 0 {
			s = 0
		}
		from := len([]rune(line[:s]))
		e.Buffer().SetCursor(from, false)
		e.Buffer().SetCursor(from+5, true)
		e.FocusLost()
		e.changed(false)
		row := e.row(0)
		for k := range rs {
			sel := row.Cells[e.gutter()+k].Style != nil && row.Cells[e.gutter()+k].Style.BackgroundColor() != nil
			if want := k >= from && k < from+5; sel != want {
				t.Errorf("%q: cell of %q selected %v", line, rs[k], sel)
			}
		}
	}
}

// In a right-to-left line drawn in a font without its letters (each
// letter in its own cell) the signs between words stand each in its own
// cell: drawn together they would read left to right (": " for " :").
func TestBidiNeutralsAlone(t *testing.T) {
	e := newEditor(t, "כותרת: המגדל, 23:40.", 40)
	e.changed(false)
	for _, o := range test.WidgetRenderer(e.grid).Objects() {
		if tx, ok := o.(*canvas.Text); ok {
			if rs := []rune(tx.Text); len(rs) > 1 && !leads(rs[0], 1) && !leads(rs[0], 2) {
				t.Errorf("%q drawn in one piece at x %v", tx.Text, tx.Position().X)
			}
		}
	}
}

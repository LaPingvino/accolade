package editor

import (
	"bytes"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"

	lexfont "github.com/LaPingvino/lexington/font"

	"github.com/LaPingvino/accolade/internal/editor/cluster"
)

const lamAlefLine = "لا إله إلا الله، لأن لإ لآ السلام"

// shapedCells is how many cells Courier Badi draws s in, shaped right to
// left as Arabic (as the editor's text drawer does).
func shapedCells(t *testing.T, s string) int {
	t.Helper()
	face, err := font.ParseTTF(bytes.NewReader(lexfont.CourierBadiRegular))
	if err != nil {
		t.Fatal(err)
	}
	rs := []rune(s)
	out := (&shaping.HarfbuzzShaper{}).Shape(shaping.Input{Text: rs, RunEnd: len(rs),
		Direction: di.DirectionRTL, Face: face, Size: fixed.I(1000), Script: language.Arabic,
		Language: language.NewLanguage("ar")})
	return (out.Advance.Round() + 300) / 600
}

// An Arabic word with a lam-alef (لا, إلا, لأن) is drawn in as many cells
// as Courier Badi draws it in: the ligature one. Given two (a column each
// for its lam and its alef) the word's text was a cell wider than its
// glyphs, a blank cell in it, a gap in the word.
func TestLamAlefDrawnInItsCells(t *testing.T) {
	e := newDevaEditor(t, lamAlefLine, 60)
	e.changed(false)
	if got, want := e.layout.RowWidth(0), shapedCells(t, lamAlefLine); got != want {
		t.Errorf("row %d columns, the font draws it in %d", got, want)
	}
	n := 0
	for _, o := range test.WidgetRenderer(e.grid).(*runRenderer).Objects() {
		tx, ok := o.(*canvas.Text)
		if !ok || strings.TrimSpace(tx.Text) == "" || !tx.Visible() {
			continue
		}
		n++
		if got, want := tx.Size().Width/e.cellWidth(), float32(shapedCells(t, tx.Text)); got != want {
			t.Errorf("%q drawn %v cells wide, its glyphs take %v", tx.Text, got, want)
		}
	}
	if n == 0 {
		t.Error("nothing drawn")
	}
}

// The cursor goes round a lam-alef, which is one cluster in one column:
// the arrows step over it, a click lands at its edges, Shift+arrow lights
// its one column, Delete takes it whole and Backspace its alef, and no
// row breaks in it.
func TestLamAlefCursor(t *testing.T) {
	e := newDevaEditor(t, lamAlefLine, 60)
	e.FocusGained()
	rs := []rune(lamAlefLine)
	for _, off := range []int{1, 9} { // لا's alef, إلا's
		if e.isClusterEdge(off) {
			t.Errorf("the cursor can stand at %d, inside a lam-alef", off)
		}
	}
	var got []int
	for range 8 { // a right-to-left line: Left reads on
		key(e, fyne.KeyLeft)
		got = append(got, e.CursorOffset())
	}
	if want := []int{2, 3, 4, 5, 6, 7, 8, 10}; !equalInts(got, want) {
		t.Errorf("Left: %v, want %v", got, want)
	}
	got = got[:0]
	for range 3 {
		key(e, fyne.KeyRight)
		got = append(got, e.CursorOffset())
	}
	if want := []int{8, 7, 6}; !equalInts(got, want) {
		t.Errorf("Right: %v, want %v", got, want)
	}
	// the cursor's place: لا the line's first column from the right, one
	// column; the cursor after it one column on
	g := float32(e.gutter())
	x := func(off int) float32 {
		e.Buffer().SetCursor(off, false)
		e.changed(false)
		return e.CursorPosition().X/e.cellWidth() - g
	}
	if a, b := x(0), x(2); a-b != 1 {
		t.Errorf("cursor before لا at column %v, after it at %v: want one column apart", a, b)
	}
	// clicks: never inside a cluster
	w := float32(cluster.Width(lamAlefLine))
	for col := float32(0); col < 60; col += 0.25 {
		if off := e.offsetAt(fyne.NewPos((g+col)*e.cellWidth(), e.cellHeight()/2)); !cluster.Boundary(rs, off) {
			t.Errorf("click at column %v (of %v) lands inside a lam-alef (%d)", col, w, off)
		}
	}
	// Shift+Left over لا: one column lit
	e.Buffer().SetCursor(0, false)
	shift(e, true)
	key(e, fyne.KeyLeft)
	shift(e, false)
	if s, end := e.Buffer().Selection(); s != 0 || end != 2 {
		t.Errorf("Shift+Left selects %d–%d, want لا (0–2)", s, end)
	}
	e.FocusLost()
	e.changed(false)
	gr, cols, from, end := e.rowCells(0)
	lit := 0
	for k := from; k < end; k++ {
		if st := gr.Cells[k].Style; st != nil && st.BackgroundColor() != nil {
			lit += int(cols[k].W)
		}
	}
	if lit != 1 {
		t.Errorf("لا selected lights %d columns, want 1", lit)
	}
	// selected, لا is still drawn whole (its ligature), in its one cell
	for _, o := range test.WidgetRenderer(e.grid).(*runRenderer).Objects() {
		if tx, ok := o.(*canvas.Text); ok && (tx.Text == "ل" || tx.Text == "ا") {
			t.Errorf("a lam-alef drawn apart: %q", tx.Text)
		}
	}
	e.FocusGained()
	e.Buffer().SetCursor(0, false)
	key(e, fyne.KeyDelete)
	if got := e.Buffer().Text(); got != strings.TrimPrefix(lamAlefLine, "لا") {
		t.Errorf("Delete: %q", got)
	}
	e.Buffer().SetCursor(8, false) // after إلا (now at 5)
	key(e, fyne.KeyBackspace)
	if got := e.Buffer().Text(); !strings.HasPrefix(got, " إله إل ") {
		t.Errorf("Backspace: %q, want إلا's alef gone", got)
	}
}

// No row starts inside a lam-alef, and each row's cells take the columns
// the layout counts, at any width.
func TestLamAlefRowsAgree(t *testing.T) {
	text := "تلاشي إلى أسود لا لا لا إلا الله السلام عليكم"
	rs := []rune(text)
	for _, cols := range []int{3, 4, 5, 7, 9, 13, 40} {
		e := newDevaEditor(t, text, cols)
		for i, r := range e.layout.Rows {
			if !cluster.Boundary(rs, r.Start) {
				t.Errorf("%d columns: row %d starts inside a lam-alef", cols, i)
			}
			_, cc, from, end := e.rowCells(i)
			n := 0
			for k := from; k < end; k++ {
				n += int(cc[k].W)
			}
			if want := cluster.Width(string(rs[r.Start:r.End])); n != want || e.layout.RowWidth(i) != want {
				t.Errorf("%d columns, row %d: cells %d, layout %d, want %d", cols, i, n, e.layout.RowWidth(i), want)
			}
		}
	}
}

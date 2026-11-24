package editor

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	lexfont "github.com/LaPingvino/lexington/font"
	"github.com/LaPingvino/lexington/lekhan"
)

// courierTheme is the test theme with Courier Badi as its monospace font,
// as Accolade's: it has Devanagari.
type courierTheme struct{ fyne.Theme }

var courierBadi = fyne.NewStaticResource("CourierBadi-Regular.ttf", lexfont.CourierBadiRegular)

func (t courierTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		return courierBadi
	}
	return t.Theme.Font(s)
}

// newDevaEditor is newEditor in Courier Badi.
func newDevaEditor(t *testing.T, text string, columns int) *ScriptEditor {
	t.Helper()
	e := newEditor(t, "", columns)
	fyne.CurrentApp().Settings().SetTheme(courierTheme{theme.DefaultTheme()})
	e.SetText(text)
	e.Resize(fyne.NewSize(e.cellWidth()*float32(columns)+1, 300))
	return e
}

// The cursor goes round a Devanagari cluster (a conjunct, a letter and
// its signs), which is drawn whole: the arrows step over it, a click in
// it lands at its nearer edge, Delete takes it whole, Backspace a letter.
func TestDevanagariClusterCursor(t *testing.T) {
	text := "क्षत्रिय कि धर्म" // क्ष | त्रि | य | ␠ | कि | ␠ | ध | र्म
	e := newDevaEditor(t, text, 40)
	e.FocusGained()
	var got []int
	for range 9 {
		key(e, fyne.KeyRight)
		got = append(got, e.CursorOffset())
	}
	want := []int{3, 7, 8, 9, 11, 12, 13, 16, 16}
	if !equalInts(got, want) {
		t.Errorf("Right: %v, want %v", got, want)
	}
	got = got[:0]
	for range 8 {
		key(e, fyne.KeyLeft)
		got = append(got, e.CursorOffset())
	}
	if want := []int{13, 12, 11, 9, 8, 7, 3, 0}; !equalInts(got, want) {
		t.Errorf("Left: %v, want %v", got, want)
	}
	// a click in the middle of क्ष (its two cells): an edge, never inside
	for col := range 6 {
		pos := fyne.NewPos((float32(e.gutter()+col)+0.4)*e.cellWidth(), e.cellHeight()/2)
		if off := e.offsetAt(pos); !e.isClusterEdge(off) {
			t.Errorf("click at column %d: offset %d, inside a cluster", col, off)
		}
	}
	if s, end := e.wordAt(14); s != 12 || end != 16 { // in धर्म
		t.Errorf("the word at धर्म: %d–%d, want 12–16", s, end)
	}
	if s, end := e.wordAt(1); s != 0 || end != 8 { // at the virama of क्षत्रिय
		t.Errorf("the word at क्षत्रिय's virama: %d–%d, want 0–8", s, end)
	}
	shift(e, true)
	key(e, fyne.KeyRight)
	shift(e, false)
	if s, end := e.Buffer().Selection(); s != 0 || end != 3 {
		t.Errorf("Shift+Right selects %d–%d, want क्ष (0–3)", s, end)
	}
	e.Buffer().SetCursor(0, false)
	key(e, fyne.KeyDelete)
	if got := e.Buffer().Text(); got != strings.TrimPrefix(text, "क्ष") {
		t.Errorf("Delete: %q", got)
	}
	e.Buffer().SetCursor(4, false) // after त्रि
	key(e, fyne.KeyBackspace)
	if got := e.Buffer().Text(); !strings.HasPrefix(got, "त्रय") {
		t.Errorf("Backspace: %q, want the i-matra gone", got)
	}
}

// A style (the selection, the cursor's cell, syntax colouring) that
// changes inside a cluster does not split its text: the cluster is
// drawn in one piece, in its first cell's style.
func TestDevanagariClusterDrawnWhole(t *testing.T) {
	e := newDevaEditor(t, "क्षत्रिय कि धर्म", 40)
	e.FocusGained()
	for _, sel := range [][2]int{{1, 2}, {2, 5}, {0, 1}, {5, 6}, {10, 11}, {13, 15}} {
		e.Buffer().Select(sel[0], sel[1])
		e.changed(false)
		r := test.WidgetRenderer(e.grid).(*runRenderer)
		for _, o := range r.Objects() {
			tx, ok := o.(*canvas.Text)
			if !ok || tx.Text == "" {
				continue
			}
			if rs := []rune(tx.Text); len(rs) > 0 && strings.ContainsRune("्िी‍", rs[0]) {
				t.Errorf("selection %v: a text starts inside a cluster: %q", sel, tx.Text)
			}
			if rs := []rune(tx.Text); rs[len(rs)-1] == 0x094D {
				t.Errorf("selection %v: a text ends at a virama: %q", sel, tx.Text)
			}
		}
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Lekhan monospace in the editor's grid: a simple cluster (कि, आ, कः)
// takes one column, a conjunct (क्ष, स्क्री) two; the cursor stands
// after them there, a selection lights all of a cluster's cells, and a
// click on either cell of a conjunct lands at its nearer edge.
func TestDevanagariLekhanCells(t *testing.T) {
	text := "क्ष कि आ कः स्क्री x" // क्ष 2 | ␠ | कि 1 | ␠ | आ 1 | ␠ | कः 1 | ␠ | स्क्री 2 | ␠ | x
	e := newDevaEditor(t, text, 40)
	e.FocusGained()
	if w := e.layout.RowWidth(0); w != 13 || w != lekhan.Width(text) {
		t.Errorf("row width %d, want 13 (lekhan.Width %d)", w, lekhan.Width(text))
	}
	g := float32(e.gutter())
	for _, c := range []struct{ off, col int }{{0, 0}, {3, 2}, {4, 3}, {6, 4}, {7, 5}, {8, 6}, {9, 7}, {11, 8}, {12, 9}, {18, 11}, {19, 12}} {
		e.Buffer().SetCursor(c.off, false)
		e.changed(false)
		if x := e.CursorPosition().X; x != (g+float32(c.col))*e.cellWidth() {
			t.Errorf("cursor at %d drawn at column %v, want %d", c.off, x/e.cellWidth()-g, c.col)
		}
	}
	// the arrows step a cluster at a time
	e.Buffer().SetCursor(0, false)
	var got []int
	for range 6 {
		key(e, fyne.KeyRight)
		got = append(got, e.CursorOffset())
	}
	if want := []int{3, 4, 6, 7, 8, 9}; !equalInts(got, want) {
		t.Errorf("Right: %v, want %v", got, want)
	}
	// Shift+Right over क्ष: both its cells lit, nothing after
	e.Buffer().SetCursor(0, false)
	shift(e, true)
	key(e, fyne.KeyRight)
	shift(e, false)
	e.FocusLost()
	e.changed(false)
	gr, cols, from, end := e.rowCells(0)
	lit := 0
	for k := from; k < end; k++ {
		if st := gr.Cells[k].Style; st != nil && st.BackgroundColor() != nil {
			lit += int(cols[k].W)
			if k-from >= 3 {
				t.Errorf("cell %d (%q) lit", k-from, gr.Cells[k].Rune)
			}
		}
	}
	if lit != 2 {
		t.Errorf("Shift+Right over क्ष lights %d columns, want 2", lit)
	}
	// unlit, the row is one text over its 13 columns
	e.Buffer().SetCursor(0, false)
	e.changed(false)
	r := test.WidgetRenderer(e.grid).(*runRenderer)
	found := false
	for _, o := range r.Objects() {
		if tx, ok := o.(*canvas.Text); ok && strings.HasPrefix(tx.Text, "क्ष") {
			found = true
			if want := 13 * e.cellWidth(); tx.Size().Width != want || tx.Position().X != g*e.cellWidth() {
				t.Errorf("%q drawn %v wide at %v, want %v at %v", tx.Text, tx.Size().Width, tx.Position().X, want, g*e.cellWidth())
			}
		}
	}
	if !found {
		t.Error("no text starting with क्ष drawn")
	}
	// clicks: क्ष's left half and its first cell's right part before it,
	// its second cell after it; कि's cell (one) by its halves
	for _, c := range []struct {
		x    float32
		want int
	}{{0.2, 0}, {0.9, 0}, {1.1, 3}, {1.8, 3}, {3.2, 4}, {3.8, 6}, {9.6, 12}, {10.4, 18}} {
		if off := e.offsetAt(fyne.NewPos((g+c.x)*e.cellWidth(), e.cellHeight()/2)); off != c.want {
			t.Errorf("click at column %v: offset %d, want %d", c.x, off, c.want)
		}
	}
}

// Each row's text cells take the columns the layout counts, which are
// Lekhan monospace's (lekhan.Width of the row's text), wrapped at any
// width; no row starts inside a cluster.
func TestDevanagariRowsAgree(t *testing.T) {
	text := "क्षत्रिय स्क्री कि धर्म र्य त्न आ\nक्षक्षक्ष"
	rs := []rune(text)
	for _, cols := range []int{4, 5, 6, 7, 9, 13, 40} {
		e := newDevaEditor(t, text, cols)
		for i, r := range e.layout.Rows {
			if !lekhan.Boundary(rs, r.Start) {
				t.Errorf("%d columns: row %d starts inside a cluster", cols, i)
			}
			gr, cc, from, end := e.rowCells(i)
			n := 0
			for k := from; k < end; k++ {
				n += int(cc[k].W)
			}
			want := lekhan.Width(string(rs[r.Start:r.End]))
			if n != want || e.layout.RowWidth(i) != want {
				t.Errorf("%d columns, row %d %q: cells %d, layout %d, lekhan %d", cols, i, gr.Cells[from:end], n, e.layout.RowWidth(i), want)
			}
		}
	}
}

// In a row the bidi algorithm reorders (Hebrew in a Hindi line, Hindi in
// a right-to-left line) a conjunct still takes two columns, drawn left to
// right; a click on it lands at its nearer edge, and no click lands
// inside a cluster.
func TestDevanagariBidiRows(t *testing.T) {
	for _, line := range []string{"क्ष שלום कि", "שלום क्ष कि"} {
		e := newDevaEditor(t, line, 40)
		e.FocusGained()
		e.changed(false)
		b, cells, cols := e.order(0)
		if !b.reordered() && !b.RTL {
			t.Fatalf("%q: not reordered", line)
		}
		at := drawnCols(cols, b)
		rs := []rune(line)
		start := strings.IndexRune(string(rs), 'क') // in bytes
		start = len([]rune(line[:start]))
		k := b.From + start // क's cell
		if cells[k].Rune != 'क' || cells[k+2].Rune != 'ष' || at[k+2] != at[k]+1 || cols[k].W+cols[k+1].W+cols[k+2].W != 2 {
			t.Errorf("%q: क्ष at columns %d, %d, %d", line, at[k], at[k+1], at[k+2])
		}
		g := float32(e.gutter())
		click := func(x float32) int {
			return e.offsetAt(fyne.NewPos((g+x)*e.cellWidth(), e.cellHeight()/2))
		}
		l := float32(at[k]) - g
		for _, c := range []struct {
			dx   float32
			want int
		}{{0.1, start}, {0.9, start}, {1.1, start + 3}, {1.9, start + 3}} {
			if off := click(l + c.dx); off != c.want {
				t.Errorf("%q: click %v into क्ष: offset %d, want %d", line, c.dx, off, c.want)
			}
		}
		for x := float32(0); x < float32(lekhan.Width(line)); x += 0.25 {
			if off := click(x); !lekhan.Boundary(rs, off) {
				t.Errorf("%q: click at column %v lands inside a cluster (%d)", line, x, off)
			}
		}
	}
}

// A Devanagari line is drawn as one text, in the font the editor is
// drawn in (Accolade's: a theme override on the editor; the app's theme,
// Fyne's, has no Devanagari): its clusters shaped together, so the font's
// joins (the headline from cluster to cluster, Courier Badi's .h pads)
// are drawn. Drawn a cluster at a time, each headline stopped short of
// the next.
func TestDevanagariWordDrawnWhole(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	e := New("मगरमच्छ जामुन")
	w := test.NewWindow(container.NewThemeOverride(e, courierTheme{theme.DefaultTheme()}))
	t.Cleanup(w.Close)
	e.Resize(fyne.NewSize(e.cellWidth()*40+1, 300))
	e.changed(false)
	var got []string
	for _, o := range test.WidgetRenderer(e.grid).(*runRenderer).Objects() {
		if tx, ok := o.(*canvas.Text); ok && tx.Text != "" && tx.Visible() {
			got = append(got, tx.Text)
		}
	}
	if len(got) != 1 || got[0] != "मगरमच्छ जामुन" {
		t.Errorf("drawn as %q, want the line whole", got)
	}
}

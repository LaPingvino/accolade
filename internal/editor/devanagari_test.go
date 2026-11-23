package editor

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	lexfont "github.com/LaPingvino/lexington/font"
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

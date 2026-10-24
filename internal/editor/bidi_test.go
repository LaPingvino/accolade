package editor

import (
	"fyne.io/fyne/v2"

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
	if m := e.mirrored(1); m != [2]int{e.gutter(), e.gutter() + 4} {
		t.Errorf("mirrored cells %v", m)
	}
	if e.mirrored(0) != [2]int{} {
		t.Error("left-to-right row mirrored")
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

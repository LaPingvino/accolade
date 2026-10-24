package editor

import (
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

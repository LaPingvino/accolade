package main

import (
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// vimStructScript: two sections, three scenes, an end block that never
// moves.
const vimStructScript = "Title: X\n\n# One\n\nINT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n# Two\n\nEXT. C - NIGHT\n\nGamma.\n\n/* Notes\nkeep\n*/\n"

// vimAt is vimStructScript's editor in vim's normal mode, the cursor at
// the start of key.
func vimAt(t *testing.T, text, key string) (*editor.ScriptEditor, *vim) {
	t.Helper()
	e, v := vimOn(text)
	cursorTo(t, e, text, key)
	return e, v
}

// cursorTo puts e's cursor at the start of key.
func cursorTo(t *testing.T, e *editor.ScriptEditor, text, key string) {
	t.Helper()
	i := strings.Index(text, key)
	if i < 0 {
		t.Fatalf("no %q", key)
	}
	at := len([]rune(text[:i]))
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(at, false) })
}

// cursorLine is the cursor's line.
func vimCursorLine(e *editor.ScriptEditor) int {
	l, _ := e.Buffer().Position(e.CursorOffset())
	return l
}

// The motions ]] [[ ]s [s, with counts, in vim's normal and visual modes.
func TestVimStructureMotions(t *testing.T) {
	for _, c := range []struct {
		from, keys, to string // to: the cursor's line starts with it
	}{
		{"Title", "]]", "INT. A"},
		{"Alpha", "]]", "INT. B"},
		{"Alpha", "2]]", "EXT. C"},
		{"Gamma", "[[", "EXT. C"},
		{"EXT. C", "[[", "INT. B"},
		{"Gamma", "3[[", "INT. A"},
		{"Alpha", "]s", "# Two"},
		{"Gamma", "[s", "# Two"},
		{"Beta", "[s", "# One"},
		{"Gamma", "]]", "Gamma"}, // none after: stays
	} {
		e, _ := vimAt(t, vimStructScript, c.from)
		vimKeys(e, c.keys)
		l := strings.Split(e.Text(), "\n")[vimCursorLine(e)]
		if !strings.HasPrefix(l, c.to) {
			t.Errorf("%s %q: on %q, want %q", c.from, c.keys, l, c.to)
		}
	}
	e, _ := vimAt(t, vimStructScript, "Alpha")
	vimKeys(e, "V]]d")
	if strings.Contains(e.Text(), "Alpha") || strings.Contains(e.Text(), "INT. B") || !strings.Contains(e.Text(), "Beta") {
		t.Errorf("V]]d: %q", e.Text())
	}
}

// The objects as is aS iS with d c y and p, v, and m (the command line).
func TestVimStructureObjects(t *testing.T) {
	for _, c := range []struct {
		from, keys, want string
	}{
		{"Alpha", "das", strings.Replace(vimStructScript, "INT. A - DAY\n\nAlpha.\n\n", "", 1)},
		{"INT. A", "dis", strings.Replace(vimStructScript, "Alpha.\n", "", 1)},
		{"Beta", "2das", strings.Replace(vimStructScript, "INT. B - DAY\n\nBeta.\n\n", "", 1)}, // the next scene is another section's: B alone
		{"Alpha", "2das", strings.Replace(vimStructScript, "INT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n", "", 1)},
		{"Alpha", "daS", strings.Replace(vimStructScript, "# One\n\nINT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n", "", 1)},
		{"Gamma", "diS", strings.Replace(vimStructScript, "EXT. C - NIGHT\n\nGamma.\n", "", 1)},
		{"Alpha", "cisNew.<esc>", strings.Replace(vimStructScript, "Alpha.", "New.", 1)},
		// a scene moved by das p: after the cursor's scene, a heading still
		{"Alpha", "dasp", strings.Replace(vimStructScript, "INT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n", "INT. B - DAY\n\nBeta.\n\nINT. A - DAY\n\nAlpha.\n\n", 1)},
		// p from the middle of a scene: after it, not inside
		{"Alpha", "yasjjp", strings.Replace(vimStructScript, "Alpha.\n\n", "Alpha.\n\nINT. A - DAY\n\nAlpha.\n\n", 1)},
		{"Beta", "yasP", strings.Replace(vimStructScript, "INT. B - DAY\n\nBeta.\n\n", "INT. B - DAY\n\nBeta.\n\nINT. B - DAY\n\nBeta.\n\n", 1)},
		// the last scene: the end block stays where it is
		{"Gamma", "das", strings.Replace(vimStructScript, "EXT. C - NIGHT\n\nGamma.\n\n", "", 1)},
		{"Title", "das", vimStructScript}, // not in a scene
		// the last scene deleted, then put: before the end block, not in it
		{"Gamma", "dasp", vimStructScript},
		// a scene selected as one (vas), deleted, put: as das p
		{"Alpha", "vasdp", strings.Replace(vimStructScript, "INT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n", "INT. B - DAY\n\nBeta.\n\nINT. A - DAY\n\nAlpha.\n\n", 1)},
	} {
		e, _ := vimAt(t, vimStructScript, c.from)
		vimKeys(e, c.keys)
		if e.Text() != c.want {
			t.Errorf("%s %q:\n%q\nwant\n%q", c.from, c.keys, e.Text(), c.want)
		}
	}
	e, _ := vimAt(t, vimStructScript, "Alpha")
	vimKeys(e, "vas")
	if got := e.SelectedText(); got != "INT. A - DAY\n\nAlpha.\n" { // line-wise: up to the blank line's (empty) end
		t.Errorf("vas: %q", got)
	}
	e, v := vimAt(t, vimStructScript, "Beta")
	var lines [2]int
	var typed string
	v.exWith = func(l [2]int, s string) { lines, typed = l, s }
	vimKeys(e, "mas")
	if typed != "move " || lines != [2]int{8, 11} {
		t.Errorf("mas: %v %q", lines, typed)
	}
	if e.Text() != vimStructScript {
		t.Error("mas changed the text")
	}
}

// helix: ]] selects the next scene, mas the cursor's; d and p act on it.
func TestHelixStructure(t *testing.T) {
	e := helixOn(vimStructScript)
	vimKeys(e, "]]")
	if got := e.SelectedText(); got != "INT. A - DAY\n\nAlpha.\n" {
		t.Errorf("]]: %q", got)
	}
	vimKeys(e, "]s")
	if got := e.SelectedText(); !strings.HasPrefix(got, "# Two\n") || !strings.Contains(got, "Gamma") {
		t.Errorf("]s: %q", got)
	}
	e = helixOn(vimStructScript)
	cursorTo(t, e, vimStructScript, "Alpha")
	vimKeys(e, "mis")
	if got := e.SelectedText(); got != "Alpha.\n" {
		t.Errorf("mis: %q", got)
	}
	vimKeys(e, "mas")
	if got := e.SelectedText(); got != "INT. A - DAY\n\nAlpha.\n\n" {
		t.Errorf("mas: %q", got)
	}
	vimKeys(e, "d")
	if strings.Contains(e.Text(), "Alpha") {
		t.Errorf("mas d: %q", e.Text())
	}
	vimKeys(e, "p") // the cursor in B: after it
	want := strings.Replace(vimStructScript, "INT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\n", "INT. B - DAY\n\nBeta.\n\nINT. A - DAY\n\nAlpha.\n\n", 1)
	if e.Text() != want {
		t.Errorf("mas d p:\n%q\nwant\n%q", e.Text(), want)
	}
}

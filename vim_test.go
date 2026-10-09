package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// vimOn is an editor with vim mode on, in normal mode, the cursor at 0.
func vimOn(text string) (*editor.ScriptEditor, *vim) {
	e := editor.New(text)
	v := newVim(e)
	v.on = func() bool { return true }
	v.status = func(string) {}
	e.OnKey, e.OnShortcut = v.key, v.shortcut
	e.OnRune = v.rune
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(0, false) })
	v.setMode(vimNormal)
	return e, v
}

// vimKeys types keys: characters, and <esc> for Escape.
func vimKeys(e *editor.ScriptEditor, s string) {
	for len(s) > 0 {
		if len(s) >= 5 && s[:5] == "<esc>" {
			e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			s = s[5:]
			continue
		}
		r := []rune(s)[0]
		e.TypedRune(r)
		s = s[len(string(r)):]
	}
}

func TestVim(t *testing.T) {
	const text = "hello world foo\nsecond line\nthird line\n"
	for _, c := range []struct {
		keys, want string
		cursor     int // -1: not checked
	}{
		{"w", text, 6},
		{"3w", text, 16}, // over the line break
		{"$", text, 15},  //
		{"dw", "world foo\nsecond line\nthird line\n", 0}, //
		{"cwHi<esc>", "Hi world foo\nsecond line\nthird line\n", 1},
		{"dd", "second line\nthird line\n", 0},
		{"2dd", "third line\n", 0},
		{"ddp", "second line\nhello world foo\nthird line\n", 12},
		{"yyP", "hello world foo\n" + text, 0},
		{"xxu", "ello world foo\nsecond line\nthird line\n", -1},
		{"J", "hello world foo second line\nthird line\n", 15},
		{"rJ", "Jello world foo\nsecond line\nthird line\n", 0},
		{"oNew<esc>", "hello world foo\nNew\nsecond line\nthird line\n", -1},
		{"AX<esc>", "hello world fooX\nsecond line\nthird line\n", -1},
		{"ved", " world foo\nsecond line\nthird line\n", 0},
		{"Vjd", "third line\n", 0},
		{"wvey$p", "hello world fooworld\nsecond line\nthird line\n", -1},
		{"D", "\nsecond line\nthird line\n", 0},
		{"jdk", "third line\n", 0},
	} {
		e, _ := vimOn(text)
		vimKeys(e, c.keys)
		if e.Text() != c.want || c.cursor >= 0 && e.CursorOffset() != c.cursor {
			t.Errorf("%q: %q cursor %d, want %q cursor %d", c.keys, e.Text(), e.CursorOffset(), c.want, c.cursor)
		}
	}
}

// Undo takes back a whole command; Ctrl+R redoes it; vim off types.
func TestVimUndoAndOff(t *testing.T) {
	e, v := vimOn("one two\n")
	vimKeys(e, "dw")
	vimKeys(e, "u")
	if e.Text() != "one two\n" {
		t.Errorf("after u: %q", e.Text())
	}
	e.TypedShortcut(&desktopCtrlR)
	if e.Text() != "two\n" {
		t.Errorf("after Ctrl+R: %q", e.Text())
	}
	v.on = func() bool { return false }
	vimKeys(e, "x")
	if e.Text() != "xtwo\n" {
		t.Errorf("vim off: %q", e.Text())
	}
	if v.mode == vimInsert {
		t.Log("mode unchanged while off, as intended")
	}
}

var desktopCtrlR = desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}

func helixOn(text string) *editor.ScriptEditor {
	e, v := vimOn(text)
	v.helix = func() bool { return true }
	v.setMode(vimNormal)
	return e
}

func TestHelix(t *testing.T) {
	const text = "hello world foo\nsecond line\nthird line\n"
	for _, c := range []struct{ keys, want, sel string }{
		{"w", text, "hello "},                              // w selects to the next word
		{"wd", "world foo\nsecond line\nthird line\n", ""}, // and d deletes it
		{"e", text, "hello"},                               //
		{"x", text, "hello world foo\n"},                   // the line
		{"xx", text, "hello world foo\nsecond line\n"},     // and the next
		{"xd", "second line\nthird line\n", ""},            //
		{"xyp", "hello world foo\nhello world foo\nsecond line\nthird line\n", "hello world foo\n"},
		{"ecHi<esc>", "Hi world foo\nsecond line\nthird line\n", ""}, // change the selection
		{"%d", "", ""},                //
		{"wdu", text, "hello "},       // undo: the text back, selected
		{"jx", text, "second line\n"}, // move, then select
		{"ve", text, "hello"},         // v extends
	} {
		e := helixOn(text)
		vimKeys(e, c.keys)
		if e.Text() != c.want || e.SelectedText() != c.sel {
			t.Errorf("%q: %q selected %q, want %q selected %q", c.keys, e.Text(), e.SelectedText(), c.want, c.sel)
		}
	}
}

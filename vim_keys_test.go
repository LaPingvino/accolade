package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// modeState is what a key can change: the text, the cursor and
// selections, the mode and what is pending, and the hooks it called.
func modeState(e *editor.ScriptEditor, v *vim, called string) string {
	s, t := e.Buffer().Selection()
	return fmt.Sprintf("%q|%d|%d-%d|%d|%v|%d|%c|%v|%v|%v|%c|%c|%v|%s|%q", e.Text(), e.CursorOffset(), s, t, len(e.Buffer().Spans()),
		v.mode, v.count, v.op, v.g, v.z, v.replace, v.bracket, v.obj, v.match, called, v.reg)
}

// modeEditor is an editor in mode's normal mode, every hook a recorder,
// the cursor in the middle of a scene.
func modeEditor(mode string) (*editor.ScriptEditor, *vim, *string) {
	text := vimStructScript
	e, v := vimOn(text)
	if mode == "helix" {
		v.helix = func() bool { return true }
		v.setMode(vimNormal)
	}
	called := new(string)
	rec := func(name string) { *called += name + ";" }
	v.search = func() { rec("search") }
	v.ex = func([2]int) { rec("ex") }
	v.exWith = func([2]int, string) { rec("exWith") }
	v.prompt = func(string, func(string)) { rec("prompt") }
	v.fold = func(r rune) { // as the window's: these alone fold (editor.go)
		if strings.ContainsRune("acoRM", r) {
			rec("fold" + string(r))
		}
	}
	v.org = func(c string) bool { rec("org" + c); return true }
	at := len([]rune(text[:strings.Index(text, "Beta.")])) + 2
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(at, false) })
	return e, v, called
}

// documented are the key sequences mode's table has.
func documented(mode string) []string {
	var out []string
	for _, k := range keysOfMode(mode) {
		out = append(out, k.seq...)
	}
	return out
}

// Every key that does something in vim's or helix's normal mode, alone
// or after a key that waits for another (g z ] [ m d c y, an object's i
// or a), has a line in the table the help is made from; every sequence
// the table names does something.
func TestModeKeysDocumented(t *testing.T) {
	for _, mode := range []string{"vim", "helix"} {
		docs := documented(mode)
		first := map[string]bool{}
		for _, d := range docs {
			first[string([]rune(d)[0])] = true
		}
		does := func(seq string) bool {
			e, v, called := modeEditor(mode)
			before := modeState(e, v, *called)
			vimKeys(e, seq)
			return modeState(e, v, *called) != before
		}
		for r := rune(33); r < 127; r++ {
			if k := string(r); does(k) && !first[k] {
				t.Errorf("%s: %q does something, and the help has no line for it", mode, k)
			}
		}
		// after a key that waits: the second keys
		for _, p := range []string{"g", "z", "]", "[", "m", "d", "da", "di", "v", "va"} {
			if !waits(mode, p) {
				continue
			}
			for r := rune(33); r < 127; r++ {
				seq := p + string(r)
				e, v, called := modeEditor(mode)
				vimKeys(e, p)
				mid := modeState(e, v, *called)
				vimKeys(e, string(r))
				now := modeState(e, v, *called)
				dropped := false // the pending key dropped: as after a shorter beginning
				for i := 0; i < len(p); i++ {
					fe, fv, fc := modeEditor(mode)
					vimKeys(fe, p[:i])
					dropped = dropped || now == modeState(fe, fv, *fc)
				}
				if strings.ContainsRune("123456789", r) || now == mid || dropped {
					continue
				}
				if !slices.ContainsFunc(docs, func(d string) bool {
					return d == seq || len(d) > 1 && strings.HasPrefix(seq, d) || strings.HasPrefix(d, seq)
				}) && !genericSecond(p, r) {
					t.Errorf("%s: %q does something, and the help has no line for it", mode, seq)
				}
			}
		}
		// what the help names does something, after what it needs (a
		// selection, a register, a change to undo, two selections)
		for _, d := range docs {
			ok := false
			for _, primer := range []string{"", "x", "yy", "xy", "xd", "xdu", "C"} {
				e, v, called := modeEditor(mode)
				vimKeys(e, primer)
				before := modeState(e, v, *called)
				vimKeys(e, d)
				ok = ok || modeState(e, v, *called) != before
			}
			if !ok {
				t.Errorf("%s: the help's %q does nothing", mode, d)
			}
		}
	}
}

// waits reports whether p leaves a key pending in mode (its second key
// then means something of its own).
func waits(mode, p string) bool {
	e, v, _ := modeEditor(mode)
	vimKeys(e, p)
	return v.g || v.z || v.bracket != 0 || v.obj != 0 || v.match || v.op != 0
}

// genericSecond: the keys after p that the table writes as one line ("d
// c y + a motion", "a motion in visual mode", "r + a character").
func genericSecond(p string, r rune) bool {
	switch p {
	case "d", "c", "y", "v":
		return strings.ContainsRune("hjklwbe0^$Gg", r) || p == "v"
	}
	return false
}

// docs/help/vim.md is what the table makes:
//
//	UPDATE_HELP=1 go test -run TestVimHelpPage .
func TestVimHelpPage(t *testing.T) {
	want := vimHelpPage()
	for _, mode := range []string{"vim", "helix"} {
		for _, k := range keysOfMode(mode) {
			if !strings.Contains(want, "**"+k.keys+"**") {
				t.Errorf("%s: %q not on the page", mode, k.keys)
			}
		}
	}
	path := filepath.Join("docs", "help", "vim.md")
	if os.Getenv("UPDATE_HELP") != "" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != want {
		t.Errorf("%s is not what the table makes: UPDATE_HELP=1 go test -run TestVimHelpPage .", path)
	}
}

// modeTokens are the keys a mode's table names, one by one ("Alt+j",
// "Tab", "Ctrl+R").
func modeTokens(mode string) map[string]bool {
	out := map[string]bool{}
	for _, k := range keysOfMode(mode) {
		for _, f := range strings.FieldsFunc(k.keys, func(r rune) bool { return r == ' ' }) {
			if strings.HasSuffix(f, ",") && !strings.HasSuffix(f, "+,") {
				f = strings.TrimSuffix(f, ",")
			}
			out[f] = true
		}
	}
	return out
}

// tableToken is a shortcut as the modes' table writes it: Alt+j, and
// with Shift the capital, Alt+J.
func tableToken(s *desktop.CustomShortcut) string {
	k := string(s.KeyName)
	if s.Modifier&fyne.KeyModifierShift == 0 && len(k) == 1 {
		k = strings.ToLower(k)
	}
	switch {
	case s.Modifier&fyne.KeyModifierControl != 0:
		return "Ctrl+" + string(s.KeyName)
	case s.Modifier&fyne.KeyModifierAlt != 0:
		return "Alt+" + k
	}
	return k
}

// Every key with a modifier a mode takes, and Tab, Enter and Backspace,
// are in its table and in modeShortcuts (what the clash test looks at);
// and every key modeShortcuts names, the mode takes.
func TestModeShortcutsListed(t *testing.T) {
	for _, mode := range []string{"vim", "helix"} {
		tokens := modeTokens(mode)
		listed := map[string]bool{}
		for _, k := range modeShortcuts[mode] {
			listed[k] = true
		}
		took := map[string]bool{}
		var keys []fyne.KeyName
		for r := 'A'; r <= 'Z'; r++ {
			keys = append(keys, fyne.KeyName(string(r)))
		}
		keys = append(keys, fyne.KeyComma, fyne.KeyPeriod, fyne.KeySlash)
		for _, k := range keys {
			for _, mod := range []fyne.KeyModifier{fyne.KeyModifierAlt, fyne.KeyModifierAlt | fyne.KeyModifierShift, fyne.KeyModifierControl, fyne.KeyModifierControl | fyne.KeyModifierShift} {
				_, v, _ := modeEditor(mode)
				s := &desktop.CustomShortcut{KeyName: k, Modifier: mod}
				if !v.shortcut(s) {
					continue
				}
				took[keyName(s)] = true
				if !listed[keyName(s)] {
					t.Errorf("%s takes %s: modeShortcuts has no %s (the clash test would not see it)", mode, keyName(s), keyName(s))
				}
				if !tokens[tableToken(s)] {
					t.Errorf("%s takes %s: its table has no %s", mode, keyName(s), tableToken(s))
				}
			}
		}
		for k := range listed {
			if !took[k] {
				t.Errorf("modeShortcuts lists %s for %s, which does not take it", k, mode)
			}
		}
		for _, k := range []fyne.KeyName{fyne.KeyTab, fyne.KeyReturn, fyne.KeyBackspace} {
			_, v, _ := modeEditor(mode)
			if v.key(&fyne.KeyEvent{Name: k}) && !tokens[keyLabel(k)] {
				t.Errorf("%s takes %s: its table has no %s", mode, k, keyLabel(k))
			}
		}
	}
}

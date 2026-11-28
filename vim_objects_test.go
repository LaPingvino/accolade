// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

// vimObjScript: speeches (one with an extension and a parenthetical, a
// pair side by side), a note, a beat in a scene and a section marked a
// beat, an end block that is no scene's.
const vimObjScript = "Title: X\n\n# One\n\nINT. A - DAY\n\nAlpha walks in. [[check the door]]\n\n" +
	"BOB (V.O.)\n(quietly)\nHello there.\nAgain.\n\nALICE\nHi.\n\n= Bob leaves angry\n\n" +
	"INT. B - DAY\n\nBRICK\nScrew retirement.\n\nSTEEL ^\nScrew it.\n\nBeta.\n\nBOB\nBye.\n\n" +
	"# Two [[beat]]\n\n= the turn\n\n/* Notes\nkeep\n*/\n"

func cut(s, part string) string { return strings.Replace(s, part, "", 1) }

// The objects ad id ac ic aB iB an in aa ia with d c y p, in vim's keys.
func TestVimSceneObjects(t *testing.T) {
	bob := "BOB (V.O.)\n(quietly)\nHello there.\nAgain.\n\n"
	pair := "BRICK\nScrew retirement.\n\nSTEEL ^\nScrew it.\n\n"
	for _, c := range []struct {
		from, keys, want string
	}{
		{"Hello", "dad", cut(vimObjScript, bob)},
		{"BOB (V.O.)", "dad", cut(vimObjScript, bob)},
		{"(quietly)", "did", cut(vimObjScript, "Hello there.\nAgain.\n")},
		{"Hello", "2dad", cut(vimObjScript, bob+"ALICE\nHi.\n\n")},
		{"Hi.", "2dad", cut(vimObjScript, "ALICE\nHi.\n\n")}, // a beat after: one alone
		{"Again", "cicROBERT<esc>", strings.Replace(vimObjScript, "BOB (V.O.)", "ROBERT (V.O.)", 1)},
		{"Again", "dac", strings.Replace(vimObjScript, "BOB (V.O.)", "", 1)},
		// a pair side by side is one speech to ad; id takes a half's lines
		{"Screw it", "dad", cut(vimObjScript, pair)},
		{"BRICK", "dad", cut(vimObjScript, pair)},
		{"Screw it", "did", cut(vimObjScript, "Screw it.\n")},
		{"Screw it", "cicIRON<esc>", strings.Replace(vimObjScript, "STEEL ^", "IRON ^", 1)},
		{"check", "dan", strings.Replace(vimObjScript, " [[check the door]]", " ", 1)},
		{"check", "cinlock it<esc>", strings.Replace(vimObjScript, "check the door", "lock it", 1)},
		{"Alpha", "dan", strings.Replace(vimObjScript, " [[check the door]]", " ", 1)}, // the next on the line
		{"Alpha", "daa", cut(vimObjScript, "Alpha walks in. [[check the door]]\n\n")},
		{"Alpha", "dia", cut(vimObjScript, "Alpha walks in. [[check the door]]\n")},
		{"= Bob", "daB", cut(vimObjScript, "= Bob leaves angry\n\n")},
		{"= the turn", "daB", cut(vimObjScript, "# Two [[beat]]\n\n= the turn\n\n")}, // a section a beat: all of it
		{"# Two", "diB", cut(vimObjScript, "= the turn\n")},
		// a speech yanked as one goes after the cursor's block, never inside it
		{"ALICE", "yadjp", strings.Replace(vimObjScript, "ALICE\nHi.\n\n", "ALICE\nHi.\n\nALICE\nHi.\n\n", 1)},
		{"Hello", "dadjjp", strings.Replace(cut(vimObjScript, bob), "ALICE\nHi.\n\n", "ALICE\nHi.\n\n"+bob, 1)},
		{"Beta", "yaaP", strings.Replace(vimObjScript, "Beta.\n\n", "Beta.\n\nBeta.\n\n", 1)},
		// not there: nothing changes
		{"Title", "dad", vimObjScript},
		{"Beta", "dac", vimObjScript},
		{"Beta", "daB", vimObjScript},
		{"Beta", "din", vimObjScript},
	} {
		e, _ := vimAt(t, vimObjScript, c.from)
		vimKeys(e, c.keys)
		if e.Text() != c.want {
			t.Errorf("%s %q:\n%q\nwant\n%q", c.from, c.keys, e.Text(), c.want)
		}
	}
	// v: line-wise for a speech, by characters for a cue
	e, _ := vimAt(t, vimObjScript, "Hi.")
	vimKeys(e, "vad")
	if got := e.SelectedText(); got != "ALICE\nHi.\n" {
		t.Errorf("vad: %q", got)
	}
	e, _ = vimAt(t, vimObjScript, "Hello")
	vimKeys(e, "vic")
	if got := e.SelectedText(); got != "BOB" {
		t.Errorf("vic: %q", got)
	}
	vimKeys(e, "<esc>vac")
	if got := e.SelectedText(); got != "BOB (V.O.)" {
		t.Errorf("vac: %q", got)
	}
	// m: a speech selected for Alt+j / Alt+k; a section beat to :move
	e, v := vimAt(t, vimObjScript, "Hi.")
	vimKeys(e, "mad")
	if v.mode != vimVisualLine || e.SelectedText() != "ALICE\nHi.\n" {
		t.Errorf("mad: %v %q", v.mode, e.SelectedText())
	}
	var typed string
	e, v = vimAt(t, vimObjScript, "= the turn")
	v.exWith = func(_ [2]int, s string) { typed = s }
	vimKeys(e, "maB")
	if typed != "move " {
		t.Errorf("maB on a section beat: %q", typed)
	}
}

// The motions ]d [d ]B [B ]n [n, with counts.
func TestVimSceneMotions(t *testing.T) {
	for _, c := range []struct{ from, keys, to string }{
		{"Alpha", "]d", "BOB (V.O.)"},
		{"Alpha", "2]d", "ALICE"},
		{"Hi.", "[d", "ALICE"},
		{"ALICE", "[d", "BOB (V.O.)"},
		{"BRICK", "]d", "STEEL ^"}, // each of a pair
		{"Alpha", "]B", "= Bob"},
		{"Alpha", "2]B", "# Two"},
		{"Beta", "[B", "= Bob"},
		{"Title", "]n", "[[check"},
		{"Beta", "[n", "[[check"},
		{"Beta", "]n", "[[beat]]"},         // a heading's note is a note too
		{"= the turn", "]n", "= the turn"}, // none after: stays
	} {
		e, _ := vimAt(t, vimObjScript, c.from)
		vimKeys(e, c.keys)
		at := e.CursorOffset()
		if got := string([]rune(e.Text())[at:]); !strings.HasPrefix(got, c.to) {
			t.Errorf("%s %q: at %.20q, want %q", c.from, c.keys, got, c.to)
		}
	}
	e, _ := vimAt(t, vimObjScript, "Alpha")
	vimKeys(e, "V]dd")
	if strings.Contains(e.Text(), "Alpha") || strings.Contains(e.Text(), "BOB (V.O.)") || !strings.Contains(e.Text(), "(quietly)") {
		t.Errorf("V]dd: %q", e.Text())
	}
}

// A speech at the script's very end, with no blank line or block after.
func TestVimSpeechAtEnd(t *testing.T) {
	const text = "INT. A - DAY\n\nRain.\n\nCARL\nBye."
	e, _ := vimAt(t, text, "Bye")
	vimKeys(e, "dad")
	if e.Text() != "INT. A - DAY\n\nRain.\n" {
		t.Errorf("dad at the end: %q", e.Text())
	}
	e, _ = vimAt(t, text, "Rain")
	vimKeys(e, "]d")
	if l := strings.Split(e.Text(), "\n")[vimCursorLine(e)]; l != "CARL" {
		t.Errorf("]d to the last speech: %q", l)
	}
	e, _ = vimAt(t, text, "CARL")
	vimKeys(e, "did")
	if e.Text() != "INT. A - DAY\n\nRain.\n\nCARL" { // the last line: with the line break before it, as dd
		t.Errorf("did at the end: %q", e.Text())
	}
}

// helix: m + an object selects it, the motions select the unit, M every
// speech of the cue's character; d then deletes them all.
func TestHelixSceneObjects(t *testing.T) {
	e := helixOn(vimObjScript)
	cursorTo(t, e, vimObjScript, "Hi.")
	vimKeys(e, "mad")
	if got := e.SelectedText(); got != "ALICE\nHi.\n\n" {
		t.Errorf("mad: %q", got)
	}
	vimKeys(e, "mic")
	if got := e.SelectedText(); got != "ALICE" {
		t.Errorf("mic: %q", got)
	}
	e = helixOn(vimObjScript)
	cursorTo(t, e, vimObjScript, "Alpha")
	vimKeys(e, "]d")
	if got := e.SelectedText(); got != "BOB (V.O.)\n(quietly)\nHello there.\nAgain.\n" {
		t.Errorf("]d: %q", got)
	}
	vimKeys(e, "]B")
	if got := e.SelectedText(); got != "= Bob leaves angry\n" {
		t.Errorf("]B: %q", got)
	}
	vimKeys(e, "[n")
	if got := e.SelectedText(); got != "[[check the door]]" {
		t.Errorf("[n: %q", got)
	}
	vimKeys(e, "man")
	if got := e.SelectedText(); got != "[[check the door]]" {
		t.Errorf("man: %q", got)
	}
	e = helixOn(vimObjScript)
	cursorTo(t, e, vimObjScript, "Hello")
	vimKeys(e, "M")
	if n := len(e.Buffer().Spans()); n != 2 {
		t.Fatalf("M: %d selections, want BOB's 2", n)
	}
	vimKeys(e, "d")
	want := cut(cut(vimObjScript, "BOB (V.O.)\n(quietly)\nHello there.\nAgain.\n\n"), "BOB\nBye.\n\n")
	if e.Text() != want {
		t.Errorf("M d:\n%q\nwant\n%q", e.Text(), want)
	}
	e = helixOn(vimObjScript)
	cursorTo(t, e, vimObjScript, "Beta")
	vimKeys(e, "M")
	if n := len(e.Buffer().Spans()); n != 1 || e.Text() != vimObjScript {
		t.Errorf("M outside a speech: %d selections", n)
	}
}

// In an Esperanto script, read the window's way: its scene headings and
// cues, in vim's keys and helix's.
func TestVimObjectsEsperanto(t *testing.T) {
	const text = "EN. KUIREJO - TAGO\n\nĈiu iras.\n\nĴOZEFO (DAŬR.)\nSaluton.\n\n= Ĵozefo foriras\n\nEKST. KAMPO - NOKTO\n\nĴOZEFO\nĜis!\n"
	w := newTestWindow(t, "")
	w.setLanguage("eo")
	w.textEditor.SetText(text)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.vim.reset(true)
	e := w.textEditor
	cursorTo(t, e, text, "Saluton")
	vimKeys(e, "cicPETRO<esc>")
	if !strings.Contains(e.Text(), "PETRO (DAŬR.)\nSaluton.") {
		t.Errorf("cic in Esperanto: %q", e.Text())
	}
	cursorTo(t, e, e.Text(), "Ĉiu")
	vimKeys(e, "]B")
	if l := strings.Split(e.Text(), "\n")[vimCursorLine(e)]; l != "= Ĵozefo foriras" {
		t.Errorf("]B in Esperanto: on %q", l)
	}
	cursorTo(t, e, e.Text(), "Ĝis")
	vimKeys(e, "das") // the scene, by the language's headings
	if strings.Contains(e.Text(), "EKST.") || !strings.Contains(e.Text(), "EN. KUIREJO") {
		t.Errorf("das in Esperanto: %q", e.Text())
	}
	vimKeys(e, "u")
	cursorTo(t, e, e.Text(), "Saluton")
	vimKeys(e, "dad")
	if strings.Contains(e.Text(), "Saluton") || !strings.Contains(e.Text(), "Ĝis!") {
		t.Errorf("dad in Esperanto: %q", e.Text())
	}
	w.settings.SetString("editor-keys", "helix")
	w.vim.reset(true)
	cursorTo(t, e, e.Text(), "Ĝis")
	vimKeys(e, "M")
	if n := len(e.Buffer().Spans()); n != 1 || e.SelectedText() != "ĴOZEFO\nĜis!\n" {
		t.Errorf("M in Esperanto: %d, %q", n, e.SelectedText())
	}
}

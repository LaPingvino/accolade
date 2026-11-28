// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor"
)

// helixAsk is text's editor in helix keys whose s and S prompt answers
// with answer.
func helixAsk(text string, answer *string) (*editor.ScriptEditor, *vim) {
	e, v := vimOn(text)
	v.helix = func() bool { return true }
	v.prompt = func(_ string, _ []string, done func(string)) { done(*answer) }
	v.setMode(vimNormal)
	return e, v
}

// spanTexts are the texts of e's selections.
func spanTexts(e *editor.ScriptEditor) []string {
	t := []rune(e.Text())
	var out []string
	for _, s := range e.Buffer().Spans() {
		out = append(out, string(t[s.Start():s.End()]))
	}
	slices.Sort(out)
	return out
}

// % s @kind for each kind, in vimObjScript (a pair side by side, a
// note, beats, the end block no kind's).
func TestHelixSelectKinds(t *testing.T) {
	for _, c := range []struct {
		kind string
		want []string
	}{
		{"@cue", []string{"ALICE", "BOB", "BOB (V.O.)", "BRICK", "STEEL ^"}},
		{"@speech", []string{"ALICE\nHi.", "BOB (V.O.)\n(quietly)\nHello there.\nAgain.", "BOB\nBye.", "BRICK\nScrew retirement.", "STEEL ^\nScrew it."}},
		{"@dialogue", []string{"Bye.", "Hello there.\nAgain.", "Hi.", "Screw it.", "Screw retirement."}},
		{"@paren", []string{"(quietly)"}},
		{"@action", []string{"Alpha walks in. [[check the door]]", "Beta."}},
		{"@heading", []string{"INT. A - DAY", "INT. B - DAY"}},
		{"@section", []string{"# One", "# Two [[beat]]"}},
		{"@note", []string{"[[beat]]", "[[check the door]]"}},
		{"@beat", []string{"= Bob leaves angry", "= the turn"}},
		{"@CUE", []string{"ALICE", "BOB", "BOB (V.O.)", "BRICK", "STEEL ^"}}, // any case
	} {
		answer := c.kind
		e, _ := helixAsk(vimObjScript, &answer)
		vimKeys(e, "%s")
		slices.Sort(c.want)
		if got := spanTexts(e); !slices.Equal(got, c.want) {
			t.Errorf("%% s %s: %q, want %q", c.kind, got, c.want)
		}
	}
	answer := "@transition"
	text := "INT. A - DAY\n\nRain.\n\nCUT TO:\n\nINT. B - DAY\n\nSun.\n"
	e, _ := helixAsk(text, &answer)
	vimKeys(e, "%s")
	if got := spanTexts(e); !slices.Equal(got, []string{"CUT TO:"}) {
		t.Errorf("@transition: %q", got)
	}
}

// Kinds inside a selection: a scene's dialogue; S splits at the cues;
// a regex as ever, @@ a literal @; an unknown kind selects nothing.
func TestHelixKindsCompose(t *testing.T) {
	answer := "@dialogue"
	e, _ := helixAsk(vimObjScript, &answer)
	cursorTo(t, e, vimObjScript, "Beta")
	vimKeys(e, "mass") // scene B, then s @dialogue
	if got := spanTexts(e); !slices.Equal(got, []string{"Bye.", "Screw it.", "Screw retirement."}) {
		t.Errorf("mas s @dialogue: %q", got)
	}
	vimKeys(e, "d")
	if strings.Contains(e.Text(), "Screw") || !strings.Contains(e.Text(), "Hello there.") {
		t.Errorf("mas s @dialogue d: %q", e.Text())
	}
	answer = "@paren"
	e, _ = helixAsk(vimObjScript, &answer)
	cursorTo(t, e, vimObjScript, "Alpha")
	vimKeys(e, "maSsd") // a section's parentheticals gone
	if strings.Contains(e.Text(), "(quietly)") {
		t.Errorf("maS s @paren d: %q", e.Text())
	}
	answer = "@cue"
	e, _ = helixAsk(vimObjScript, &answer)
	cursorTo(t, e, vimObjScript, "Hi.")
	vimKeys(e, "madS") // ALICE's speech split at its cue: the rest
	if got := spanTexts(e); !slices.Equal(got, []string{"\nHi.\n\n"}) {
		t.Errorf("mad S @cue: %q", got)
	}
	// rename a character everywhere: % s @cue, s BOB, c
	answer = "@cue"
	e, v := helixAsk(vimObjScript, &answer)
	vimKeys(e, "%s")
	answer = "BOB"
	vimKeys(e, "scROBERT<esc>")
	if strings.Contains(e.Text(), "BOB") || strings.Count(e.Text(), "ROBERT") != 2 || !strings.Contains(e.Text(), "ROBERT (V.O.)") {
		t.Errorf("rename by cues: %q", e.Text())
	}
	_ = v
	// a regex, and @@
	const at = "INT. A - DAY\n\nMail @home and @work.\n"
	answer = "@@\\w+"
	e, _ = helixAsk(at, &answer)
	vimKeys(e, "%s")
	if got := spanTexts(e); !slices.Equal(got, []string{"@home", "@work"}) {
		t.Errorf("s @@\\w+: %q", got)
	}
	answer = "@nothing"
	e, _ = helixAsk(at, &answer)
	vimKeys(e, "%s")
	if len(e.Buffer().Spans()) != 1 || e.SelectedText() != e.Text() {
		t.Errorf("an unknown kind changed the selection: %q", spanTexts(e))
	}
	// the prompt completes the names, with what each is
	c := promptCompletion(elementKindNames(), "@d")
	if len(c.choices) != 1 || c.choices[0].text != "@dialogue" || c.choices[0].desc == "" {
		t.Errorf("@d completes to %v", c.choices)
	}
	if c := promptCompletion(elementKindNames(), "INT"); len(c.choices) != 0 {
		t.Errorf("a regex completed: %v", c.choices)
	}
}

// In an Esperanto window: the language's headings and cues; and Select
// All Character Cues in standard keys.
func TestHelixKindsEsperanto(t *testing.T) {
	const text = "EN. KUIREJO - TAGO\n\nĈiu iras.\n\nĴOZEFO (DAŬR.)\nSaluton.\n\nEKST. KAMPO - NOKTO\n\nMARIA\nĜis!\n"
	w := newTestWindow(t, "")
	w.setLanguage("eo")
	w.textEditor.SetText(text)
	w.vim.prompt = func(_ string, _ []string, done func(string)) { done("@heading") }
	w.settings.SetString("editor-keys", "helix")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.vim.reset(true)
	vimKeys(w.textEditor, "%s")
	if got := spanTexts(w.textEditor); !slices.Equal(got, []string{"EKST. KAMPO - NOKTO", "EN. KUIREJO - TAGO"}) {
		t.Errorf("@heading in Esperanto: %q", got)
	}
	w.settings.SetString("editor-keys", "standard")
	w.vim.reset(false)
	cmd("select.cue").run(w)
	if got := spanTexts(w.textEditor); !slices.Equal(got, []string{"MARIA", "ĴOZEFO (DAŬR.)"}) {
		t.Errorf("Select All Character Cues in Esperanto: %q", got)
	}
}

// The key hints after ] and an object's a, and the mode's line after.
func TestModeKeyHints(t *testing.T) {
	var said string
	e, v := vimOn(vimObjScript)
	v.status = func(s string) { said = s }
	vimKeys(e, "]")
	if !strings.Contains(said, "d speech") || !strings.Contains(said, "B beat") {
		t.Errorf("] hint: %q", said)
	}
	vimKeys(e, "d")
	if said != "-- NORMAL --" {
		t.Errorf("after ]d: %q", said)
	}
	vimKeys(e, "da")
	if !strings.Contains(said, "c cue") || !strings.Contains(said, "a action paragraph") {
		t.Errorf("da hint: %q", said)
	}
}

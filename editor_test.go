package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// typeScript types each line followed by Enter, like a writer would.
func typeScript(w *MainWindow, lines ...string) {
	w.fyneWindow.Canvas().Focus(w.textEditor)
	for _, l := range lines {
		test.Type(w.textEditor, l)
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	}
}

func pad(n int, s string) string { return strings.Repeat(" ", n) + s }

func TestTypingAScene(t *testing.T) {
	w := newTestWindow(t, "")
	typeScript(w,
		"FADE IN:",
		"",
		"int. barn - day",
		"",
		"Rain hammers the roof.",
		"",
		"JOHN",
		"(beat)",
		"It's coming.",
		"",
		"CUT TO:",
	)
	want := strings.Join([]string{
		"FADE IN:",
		"",
		"INT. BARN - DAY",
		"",
		"Rain hammers the roof.",
		"",
		pad(CharacterIndent, "JOHN"),
		pad(ParentheticalIndent, "(beat)"),
		pad(DialogueIndent, "It's coming."),
		"",
		pad(TransitionIndent, "CUT TO:"),
		"",
	}, "\n")
	if got := w.textEditor.Text; got != want {
		t.Errorf("script:\n%q\nwant:\n%q", got, want)
	}
}

func TestSpaceAfterSceneHeadingPrefixSurvives(t *testing.T) {
	w := newTestWindow(t, "")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	test.Type(w.textEditor, "INT. BARN")
	if got := w.textEditor.Text; got != "INT. BARN" {
		t.Errorf("typed %q", got)
	}
}

func TestFormattingKeepsUndoHistory(t *testing.T) {
	w := newTestWindow(t, "")
	typeScript(w, "", "john")
	if got := w.textEditor.Text; got != "\n"+pad(CharacterIndent, "john")+"\n" {
		// lowercase is not a character cue: left alone
		if got != "\njohn\n" {
			t.Fatalf("text = %q", got)
		}
	}
	undoAll(w)
	if w.textEditor.Text != "" {
		t.Errorf("undo could not get back to the empty script: %q", w.textEditor.Text)
	}

	w2 := newTestWindow(t, "")
	typeScript(w2, "int. barn")
	if got := w2.textEditor.Text; got != "INT. BARN\n" {
		t.Fatalf("text = %q", got)
	}
	// Fyne records the rewritten line word by word, so stepping back
	// through the formatting takes a few undos, but the typed text is
	// still in the history.
	seen := undoAll(w2)
	if !seen["int. barn"] {
		t.Errorf("undo never returned to the text as typed; states: %v", seen)
	}
	if w2.textEditor.Text != "" {
		t.Errorf("undo stopped at %q", w2.textEditor.Text)
	}
}

// undoAll undoes until nothing changes and returns every state passed.
func undoAll(w *MainWindow) map[string]bool {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		before := w.textEditor.Text
		w.textEditor.Undo()
		if w.textEditor.Text == before {
			break
		}
		seen[w.textEditor.Text] = true
	}
	return seen
}

func TestEnterMidLineDoesNotFormat(t *testing.T) {
	w := newTestWindow(t, "int. barn")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	setCursorOffset(&w.textEditor.Entry, 4)
	w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if got := w.textEditor.Text; got != "int.\n barn" {
		t.Errorf("text = %q", got)
	}
}

func TestLoadingDoesNotReformat(t *testing.T) {
	text := "INT. HOUSE - DAY\n\nJOHN\n  indented action stays put"
	w := newTestWindow(t, text)
	if w.textEditor.Text != text {
		t.Errorf("loading changed the text: %q", w.textEditor.Text)
	}
}

func TestFormatCompletedLine(t *testing.T) {
	cases := []struct{ prev, line, want, kind string }{
		{"", "FADE IN:", "FADE IN:", "Action"},
		{"", "ext. field - night", "EXT. FIELD - NIGHT", "Scene Heading"},
		{"Some action.", "int. barn", "int. barn", "Action"}, // needs a blank line before
		{"", "MARY (V.O.)", pad(CharacterIndent, "MARY (V.O.)"), "Character"},
		{"", "BANG!", "BANG!", "Action"},
		{"", "OK.", "OK.", "Action"},
		{"", "I", "I", "Action"},
		{"", "smash cut to:", pad(TransitionIndent, "SMASH CUT TO:"), "Transition"},
		{"", "fade out.", pad(TransitionIndent, "> FADE OUT."), "Transition"},
		{"", "> fade to black.", pad(TransitionIndent, "> FADE TO BLACK."), "Transition"},
		{pad(CharacterIndent, "JOHN"), "(quietly)", pad(ParentheticalIndent, "(quietly)"), "Parenthetical"},
		{"", "  He waits.", "  He waits.", "Action"},
	}
	for _, c := range cases {
		got, kind := formatCompletedLine(c.prev, c.line)
		if got != c.want || kind != c.kind {
			t.Errorf("formatCompletedLine(%q, %q) = %q, %s; want %q, %s", c.prev, c.line, got, kind, c.want, c.kind)
		}
	}
}

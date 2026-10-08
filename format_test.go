package main

import (
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

func TestAsElement(t *testing.T) {
	ind := func(n int, s string) string { return strings.Repeat(" ", n) + s }
	for _, c := range []struct{ line, element, want string }{
		{"int. kitchen - day", elementScene, "INT. KITCHEN - DAY"},
		{"the kitchen", elementScene, ".THE KITCHEN"},
		{"anna", elementCharacter, ind(CharacterIndent, "ANNA")},
		{"Dr. Who", elementCharacter, ind(CharacterIndent, "@DR. WHO")},
		{"quietly", elementParenthetical, ind(ParentheticalIndent, "(quietly)")},
		{ind(ParentheticalIndent, "(quietly)"), elementDialogue, ind(DialogueIndent, "quietly")},
		{"cut to:", elementTransition, ind(TransitionIndent, "CUT TO:")},
		{"fade out.", elementTransition, ind(TransitionIndent, "> FADE OUT.")},
		{ind(CharacterIndent, "ANNA"), elementAction, "!ANNA"}, // would read as a name
		{".THE KITCHEN", elementAction, "!THE KITCHEN"},        // and as a heading, upper case
		{"Anna walks in.", elementAction, "Anna walks in."},
		{"THE END", elementCentered, "> THE END <"},
		{"> THE END <", elementCharacter, ind(CharacterIndent, "THE END")},
	} {
		if got := asElement(c.line, c.element, nil); got != c.want {
			t.Errorf("asElement(%q, %s) = %q, want %q", c.line, c.element, got, c.want)
		}
	}
}

func edited(text string, sel [2]int, f func(e *editor.ScriptEditor)) (string, string) {
	e := editor.New(text)
	e.Navigate(func(b *buffer.Buffer) { b.Select(sel[0], sel[1]) })
	f(e)
	return e.Text(), e.SelectedText()
}

func TestToggleEmphasis(t *testing.T) {
	bold := func(e *editor.ScriptEditor) { toggleEmphasis(e, "**") }
	italic := func(e *editor.ScriptEditor) { toggleEmphasis(e, "*") }
	if text, sel := edited("a word here", [2]int{2, 6}, bold); text != "a **word** here" || sel != "word" {
		t.Errorf("bold: %q %q", text, sel)
	}
	if text, sel := edited("a **word** here", [2]int{4, 8}, bold); text != "a word here" || sel != "word" {
		t.Errorf("unbold around: %q %q", text, sel)
	}
	if text, _ := edited("a **word** here", [2]int{2, 10}, bold); text != "a word here" {
		t.Errorf("unbold inside: %q", text)
	}
	// italic of a bold word is added, not taken for an italic marker
	if text, _ := edited("a **word** here", [2]int{4, 8}, italic); text != "a ***word*** here" {
		t.Errorf("italic in bold: %q", text)
	}
	if text, _ := edited("a *word* here", [2]int{3, 7}, italic); text != "a word here" {
		t.Errorf("unitalic: %q", text)
	}
	e := editor.New("ab")
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(1, false) })
	toggleEmphasis(e, "_")
	if e.Text() != "a__b" || e.CursorOffset() != 2 {
		t.Errorf("empty selection: %q at %d", e.Text(), e.CursorOffset())
	}
}

func TestMoveLines(t *testing.T) {
	e := editor.New("one\ntwo\nthree")
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(5, false) }) // "two", column 1
	moveLines(e, -1)
	if e.Text() != "two\none\nthree" || e.CursorOffset() != 1 {
		t.Errorf("up: %q at %d", e.Text(), e.CursorOffset())
	}
	moveLines(e, -1) // at the top: nothing
	if e.Text() != "two\none\nthree" {
		t.Errorf("up at the top: %q", e.Text())
	}
	e.Navigate(func(b *buffer.Buffer) { b.Select(0, 7) }) // "two" and "one"
	moveLines(e, 1)
	if e.Text() != "three\ntwo\none" || e.SelectedText() != "two\none" {
		t.Errorf("down: %q, selected %q", e.Text(), e.SelectedText())
	}
	e.Buffer().Undo()
	if e.Text() != "two\none\nthree" {
		t.Errorf("undo: %q", e.Text())
	}
}

// The shortcuts are on the main menu, where Fyne looks first.
func TestFormatMenu(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	found := map[string]bool{}
	for _, m := range w.buildMainMenu().Items {
		if m.Label != "Format" {
			continue
		}
		for _, it := range m.Items {
			if it.Shortcut != nil {
				found[it.Label] = true
			}
		}
	}
	for _, l := range []string{"Scene Heading", "Character", "Dialogue", "Bold", "Italic", "Underline", "Move Line Up"} {
		if !found[l] {
			t.Errorf("Format menu: no %s shortcut", l)
		}
	}
}

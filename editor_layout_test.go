package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/rules"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

func TestEditorFormatFollowsLexington(t *testing.T) {
	format, page := editorFormat(rules.Default)
	for k, want := range map[syntax.Kind]wrap.Format{
		syntax.Action:     {Indent: 0, Width: 60, Align: 'L'},
		syntax.Character:  {Indent: 22, Width: 33, Align: 'L'},
		syntax.Dialogue:   {Indent: 10, Width: 45, Align: 'L'},
		syntax.Transition: {Indent: 0, Width: 60, Align: 'R'},
	} {
		if got := format(k); got != want {
			t.Errorf("kind %d: %+v, want %+v", k, got, want)
		}
	}
	if page != 60 {
		t.Errorf("page: %d columns", page)
	}
}

// Scripts Accolade saved with layout spaces open without them, read the
// same by Lexington.
func TestOpeningDropsLayoutSpaces(t *testing.T) {
	pad := func(n int, s string) string { return strings.Repeat(" ", n) + s }
	old := strings.Join([]string{
		"INT. BARN - DAY",
		"",
		"  Rain, indented on purpose.",
		"",
		pad(37, "JOHN"),
		pad(31, "(beat)"),
		pad(25, "It's coming."),
		"",
		pad(60, "CUT TO:"),
		"",
	}, "\n")
	path := filepath.Join(t.TempDir(), "old.fountain")
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	w := newTestWindow(t, "")
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	got := w.textEditor.Text()
	want := "INT. BARN - DAY\n\n  Rain, indented on purpose.\n\nJOHN\n(beat)\nIt's coming.\n\nCUT TO:\n"
	if got != want {
		t.Errorf("opened as\n%q\nwant\n%q", got, want)
	}
	if !reflect.DeepEqual(fountain.Parse(nil, strings.NewReader(got)), fountain.Parse(nil, strings.NewReader(old))) {
		t.Error("Lexington reads the script differently")
	}
}

// The editor shows dialogue in its column without spaces in the text, the
// cursor there as soon as Enter follows a name.
func TestDialogueColumnWithoutSpaces(t *testing.T) {
	w := newTestWindow(t, "")
	typeScript(w, "INT. BARN - DAY", "", "John waits.", "", "JOHN")
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nJohn waits.\n\nJOHN\n" {
		t.Fatalf("text %q", got)
	}
	x := w.textEditor.CursorPosition().X
	w.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\nJ"))
	cell := w.textEditor.CursorPosition().X
	format, _ := editorFormat(currentScriptFormat())
	if want := format(syntax.Dialogue).Indent; cell <= 0 || int(x/cell+0.5) != want {
		t.Errorf("cursor at column %.1f, want the dialogue's %d", x/cell, want)
	}
}

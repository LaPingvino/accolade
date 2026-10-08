package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestNotesBlock(t *testing.T) {
	script := "INT. A - DAY\n\nHi.\n"
	with := strings.TrimRight(script, "\n") + notesText("Call Bram.\n# not a section")
	if notesOf(with) != "Call Bram.\n# not a section" {
		t.Errorf("notes %q", notesOf(with))
	}
	if notesStart(with) != len([]rune("INT. A - DAY\n\nHi.")) {
		t.Errorf("start %d", notesStart(with))
	}
	if notesText("a */ b") != "\n\n/* Notes\na * / b\n*/\n" {
		t.Errorf("*/ in notes: %q", notesText("a */ b"))
	}
	if notesText("  \n") != "" || notesOf(script) != "" {
		t.Error("empty notes")
	}
	if items := outlineOf(with); len(items) != 1 {
		t.Errorf("outline reads the notes: %+v", items)
	}
}

func TestNotesPanel(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. A - DAY\n\nHi.\n")
	w.toggleNotes()
	test.Type(w.notes.entry, "Call Bram.")
	if got := w.textEditor.Text(); got != "INT. A - DAY\n\nHi.\n\n/* Notes\nCall Bram.\n*/\n" {
		t.Fatalf("script with notes: %q", got)
	}
	// typing on in the script keeps the notes at the end
	w.textEditor.SetText(strings.Replace(w.textEditor.Text(), "Hi.", "Hello.", 1))
	if w.notes.entry.Text != "Call Bram." {
		t.Errorf("notes after an edit: %q", w.notes.entry.Text)
	}
	w.notes.entry.SetText("")
	if got := w.textEditor.Text(); got != "INT. A - DAY\n\nHello.\n" {
		t.Errorf("notes removed: %q", got)
	}
	if w.outlineVisible {
		t.Error("outline and notes at once")
	}
	// the preview and the exports do not print them
	w.textEditor.SetText("INT. A - DAY\n\nHi.\n\n/* Notes\nsecret\n*/\n")
	if strings.Contains(previewText(previewSegments(w.textEditor.Text(), []string{"INT"}, nil)), "secret") {
		t.Error("notes in the preview")
	}
}

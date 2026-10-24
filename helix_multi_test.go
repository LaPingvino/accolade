package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

// Helix's several selections in a window: % then s selects the names, c
// changes them all at once (one undo step), Alt-s splits into lines, C
// adds one below, , keeps the primary.
func TestHelixSelections(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("editor-keys", "helix")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	const text = "ANNA sits.\nBRAM sits.\nCEES sits.\n"
	w.textEditor.SetText(text)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	e := w.textEditor
	e.TypedRune('%')
	e.TypedRune('s')
	test.Type(w.exEntry, "[A-Z]{4}")
	w.exEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if n := len(e.Buffer().Spans()); n != 3 {
		t.Fatalf("%d selections", n)
	}
	e.TypedRune('c')
	test.Type(e, "X")
	if got := e.Text(); got != "X sits.\nX sits.\nX sits.\n" {
		t.Fatalf("changed: %q", got)
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	e.TypedRune('u')
	e.TypedRune('u')
	if got := e.Text(); got != text {
		t.Errorf("undone: %q", got)
	}

	e.TypedRune('%')
	e.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierAlt})
	if n := len(e.Buffer().Spans()); n != 3 {
		t.Errorf("Alt-s: %d lines", n)
	}
	e.TypedRune(',')
	if e.Buffer().HasOthers() {
		t.Error(", kept the others")
	}
	e.TypedRune('C')
	if n := len(e.Buffer().Spans()); n != 2 {
		t.Errorf("C: %d selections", n)
	}
}

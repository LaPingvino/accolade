package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestUndoRedoButtons(t *testing.T) {
	w := newTestWindow(t, "")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	test.Type(w.textEditor, "Rain")
	if w.textEditor.Text() != "Rain" {
		t.Fatalf("typed text = %q", w.textEditor.Text())
	}

	if w.headerBar.undoButton.Disabled() || w.headerBar.redoButton.Disabled() {
		t.Fatal("undo/redo buttons are disabled")
	}
	test.Tap(w.headerBar.undoButton)
	if w.textEditor.Text() != "" {
		t.Errorf("after Undo: %q", w.textEditor.Text())
	}
	test.Tap(w.headerBar.redoButton)
	if w.textEditor.Text() != "Rain" {
		t.Errorf("after Redo: %q", w.textEditor.Text())
	}
	if !w.hasChanges {
		t.Error("undo/redo should mark the document changed")
	}
}

func TestUndoRedoMenuFollowsFocus(t *testing.T) {
	w := newTestWindow(t, "")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	test.Type(w.textEditor, "barn")

	menuItemByLabel(t, w, "Find…").Action() // focuses the search field
	test.Type(w.searchBar.searchEntry, "owl")

	menuItemByLabel(t, w, "Undo").Action()
	if got := w.searchBar.searchEntry.Text; got != "" {
		t.Errorf("Undo with the search field focused: search = %q", got)
	}
	if w.textEditor.Text() != "barn" {
		t.Errorf("Undo in the search field touched the editor: %q", w.textEditor.Text())
	}

	w.hideFindReplace() // focus back to the editor
	menuItemByLabel(t, w, "Undo").Action()
	if w.textEditor.Text() != "" {
		t.Errorf("Undo in the editor: %q", w.textEditor.Text())
	}
	redo := menuItemByLabel(t, w, "Redo")
	if redo.Shortcut == nil || redo.Shortcut.ShortcutName() != shortcut(fyne.KeyZ, fyne.KeyModifierShift).ShortcutName() {
		t.Errorf("Redo shortcut = %v, want Ctrl+Shift+Z", redo.Shortcut)
	}
	redo.Action()
	if w.textEditor.Text() != "barn" {
		t.Errorf("Redo in the editor: %q", w.textEditor.Text())
	}
}

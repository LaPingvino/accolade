package main

import (
	"testing"

	"fyne.io/fyne/v2"
)

// Folds in the editor with vim keys: zc hides a scene's lines (the cursor
// to its heading), zo shows them, zM folds all, and the cursor going into
// a fold opens it.
func TestEditorFolds(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	e := w.textEditor
	e.SetText("INT. BARN - DAY\n\nRain on the roof.\n\nMore rain.\n\nEXT. FIELD - DAY\n\nWind.\n")
	all := w.editorFoldRanges
	e.SetCursorOffset(len("INT. BARN - DAY\n\nR"))
	e.TypedKey(escape())
	e.TypedRune('z')
	e.TypedRune('c')
	if r := all(); len(r) != 1 || r[0] != [2]int{1, 5} {
		t.Fatalf("zc hid %v", r)
	}
	if line, _ := e.Buffer().Position(e.CursorOffset()); line != 0 {
		t.Errorf("cursor on line %d, not the heading", line)
	}
	e.TypedRune('z')
	e.TypedRune('o')
	if len(all()) != 0 {
		t.Errorf("zo left %v", all())
	}
	e.TypedRune('z')
	e.TypedRune('M')
	if len(all()) != 2 {
		t.Fatalf("zM: %v", all())
	}
	e.SetCursorOffset(len("INT. BARN - DAY\n\nR")) // into the barn's fold
	e.Refresh()
	if len(all()) != 1 {
		t.Errorf("the fold the cursor went into: %v", all())
	}
}

func escape() *fyne.KeyEvent { return &fyne.KeyEvent{Name: fyne.KeyEscape} }

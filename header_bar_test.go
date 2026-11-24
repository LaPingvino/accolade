package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

func TestHeaderShowsDocumentState(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY")
	w.updateTitle()
	if got := w.headerBar.titleLabel.Text; got != "Untitled" {
		t.Errorf("new document: %q", got)
	}
	w.textEditor.SetText("INT. BARN - NIGHT")
	if got := w.headerBar.titleLabel.Text; got != "• Untitled" {
		t.Errorf("after an edit: %q", got)
	}
	w.currentFile = "/tmp/barn.fountain"
	w.hasChanges = false
	w.updateTitle()
	if got := w.headerBar.titleLabel.Text; got != "barn.fountain" {
		t.Errorf("saved file: %q", got)
	}
}

func TestHeaderButtonsHaveTips(t *testing.T) {
	w := newTestWindow(t, "")
	hb := w.headerBar
	for name, tip := range map[string]string{
		"new": hb.button("file.new").Tip, "open": hb.button("file.open").Tip, "save": hb.button("file.save").Tip,
		"undo": hb.button("edit.undo").Tip, "redo": hb.button("edit.redo").Tip, "find": hb.button("edit.findReplace").Tip,
		"preview": hb.button("view.preview").Tip, "export": hb.button("file.export").Tip, "preferences": hb.button("app.preferences").Tip,
		"outline": hb.button("view.outline").Tip, "cards": hb.button("view.cards").Tip, // Notes has none: Ctrl+Shift+K is Add Note
	} {
		if !strings.Contains(tip, "(") {
			t.Errorf("%s button tip %q has no shortcut", name, tip)
		}
	}
}

// Alt+Enter toggles full screen, beside Ctrl+Shift+F; Enter alone is the
// editor's.
func TestAltEnterFullScreen(t *testing.T) {
	w := newTestWindow(t, "")
	if w.keyPreview(fyne.KeyReturn, 0) {
		t.Error("Enter alone was taken from the editor")
	}
	if !w.keyPreview(fyne.KeyReturn, fyne.KeyModifierAlt) || !w.isFullscreen {
		t.Fatal("Alt+Enter: not full screen")
	}
	w.keyPreview(fyne.KeyReturn, fyne.KeyModifierAlt)
	if w.isFullscreen {
		t.Error("Alt+Enter again: still full screen")
	}
}

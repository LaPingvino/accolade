package main

import (
	"strings"
	"testing"
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
		"new": hb.newButton.Tip, "open": hb.openButton.Tip, "save": hb.saveButton.Tip,
		"undo": hb.undoButton.Tip, "redo": hb.redoButton.Tip, "find": hb.findButton.Tip,
		"preview": hb.previewButton.Tip, "export": hb.exportButton.Tip, "preferences": hb.preferencesButton.Tip,
	} {
		if !strings.Contains(tip, "(") {
			t.Errorf("%s button tip %q has no shortcut", name, tip)
		}
	}
}

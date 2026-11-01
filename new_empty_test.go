package main

import "testing"

func TestNewEmptyFile(t *testing.T) {
	w := newTestWindow(t, "INT. ROOM - DAY")
	w.NewEmptyFile()
	if got := w.textEditor.Text(); got != "" {
		t.Errorf("New Empty: the script has %q", got)
	}
	w.NewFile()
	if w.IsEmpty() {
		t.Error("New: no sample script")
	}
}

func TestNewEmptyWindow(t *testing.T) {
	w := newTestWindow(t, "")
	app := w.app
	if got := app.newScriptWindow(true).textEditor.Text(); got != "" {
		t.Errorf("New Empty Window: the script has %q", got)
	}
	if app.newScriptWindow(false).IsEmpty() {
		t.Error("New Window: no sample script")
	}
}

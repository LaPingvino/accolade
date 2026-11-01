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

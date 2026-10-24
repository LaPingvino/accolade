package main

import (
	"strings"
	"testing"
)

// The characters, who speaks most first, without extensions; one picked
// goes in as a cue with the cursor on the dialogue's line.
func TestCharacters(t *testing.T) {
	const script = "INT. BARN - DAY\n\nANNA\nRain.\n\nBRAM (V.O.)\nYes.\n\nANNA (CONT'D)\nMore rain.\n\nBRAM ^\nNo.\n\nANNA\nStill.\n"
	if got := strings.Join(scriptCharacters(script, sceneStarts("en")), ","); got != "ANNA,BRAM" {
		t.Errorf("characters %q", got)
	}
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. BARN - DAY\n\nRain.\n")
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	w.insertCue("ANNA")
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nRain.\n\nANNA\n" {
		t.Errorf("cue: %q", got)
	}
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 5 {
		t.Errorf("cursor on line %d", line)
	}
}

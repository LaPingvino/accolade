package main

import (
	"strings"
	"testing"
)

const outlineScript = `Title: T

# ACT ONE

= Anna wants the biscuit.

INT. KITCHEN - DAY #1#

= She finds the tin empty.

Anna looks.

## The garden

.THE GARDEN

EXT. ROAD - NIGHT
Action right after the heading.
`

func TestOutlineOf(t *testing.T) {
	var got []string
	for _, it := range outlineOf(outlineScript, nil) {
		got = append(got, strings.Join([]string{strings.Repeat(">", it.Level), it.Text, it.Synopsis}, "|"))
	}
	want := []string{
		">|ACT ONE|Anna wants the biscuit.",
		">>|INT. KITCHEN - DAY|She finds the tin empty.",
		">>|The garden|",
		">>>|THE GARDEN|",
		">>>|EXT. ROAD - NIGHT|",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("outline:\n%s", strings.Join(got, "\n"))
	}
}

func TestOutlinePanelGoesToTheScene(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.toggleOutline()
	if !w.outlineVisible || len(w.outline.items) != 5 {
		t.Fatalf("outline: %v, %d items", w.outlineVisible, len(w.outline.items))
	}
	w.outline.list.Select(1) // INT. KITCHEN
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 6 {
		t.Errorf("cursor on line %d, want 6", line)
	}
	w.textEditor.SetText(outlineScript + "\nINT. HALL - DAY\n")
	if len(w.outline.items) != 6 {
		t.Errorf("outline not updated: %d items", len(w.outline.items))
	}
	w.togglePreview() // both at once
	if !w.previewVisible || !w.outlineVisible {
		t.Error("preview and outline together")
	}
	capture(t, w.fyneWindow.Canvas(), "outline.png")
}

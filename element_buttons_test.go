package main

import (
	"image/png"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// The elements as buttons with their Fountain marks: @ makes the line a
// character, and the line's element stands out.
func TestElementButtons(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetBoolean("element-buttons", false) })
	w.fyneWindow.Resize(fyne.NewSize(1000, 300))
	w.settings.SetBoolean("element-buttons", true)
	hb := w.headerBar
	hb.showElementKind()
	if hb.elementBox.Objects[0] != hb.elementMarks {
		t.Fatal("not the buttons")
	}
	w.textEditor.SetText("INT. BARN - DAY\n\nanna\n")
	w.textEditor.SetCursorOffset(len("INT. BARN - DAY\n\n"))
	test.Tap(hb.elementButtons[elementCharacter])
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nANNA\n" {
		t.Errorf("@: %q", got)
	}
	w.textEditor.SetCursorOffset(0)
	if hb.elementButtons[elementScene].Importance != widget.HighImportance {
		t.Error("the scene heading's button does not stand out")
	}
	if p := os.Getenv("BUTTONS_PNG"); p != "" {
		f, _ := os.Create(p)
		_ = png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
}

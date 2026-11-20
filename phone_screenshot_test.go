package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// TestPhoneScreenshots renders the window at a phone's size (380×800):
// the script, the source pane open on a story with a wide table, the
// Notes tab, and the script again (SHOT_DIR as in screenshot_test.go).
func TestPhoneScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, sourceScript, wideStory())
	w.toggleSource() // closed again: the script alone first
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "phone-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	snap("1-script")
	w.toggleSource()
	s := w.sourcePane()
	s.follow()
	snap("2-source")
	y := float32(0) // the table's row at the top
	for i, p := range s.passages {
		if len(p.Markdown) > 0 && p.Markdown[0] == '|' {
			s.list.ScrollToOffset(y)
			break
		}
		y += s.heights[i] + theme.Padding()
	}
	snap("3-source-table")
	w.notes.tabs.Select(w.notes.notesTab)
	snap("4-notes")
	w.notes.tabs.Select(w.notes.sourceTab)
	w.toggleSource()
	snap("5-script-again")
}

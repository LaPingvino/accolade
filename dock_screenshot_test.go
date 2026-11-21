package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
)

// TestDockScreenshots renders the side dock's arrangements, driven only
// through the registry's commands (SHOT_DIR as in screenshot_test.go):
// the outline alone, two slots, a tab moved to the other slot, the dock
// hidden, the same on a phone, and every pane in at once (the tab rows
// wrapping).
func TestDockScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, sourceScript, wideStory())
	keepDockSettings(t, w)
	run := func(id string) {
		t.Helper()
		if c, ok := commands().byID[id]; ok {
			c.run(w)
		} else {
			t.Logf("no command %s", id)
		}
	}
	run("view.source") // sourceWindow showed it: hidden again
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "dock-"+name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	run("view.outline")
	snap("1-outline")
	run("view.notes")
	run("view.source")
	snap("2-two-slots")
	if _, ok := commands().byID["view.paneTop"]; ok {
		run("view.paneTop") // the source, picked last, up beside the outline
		snap("3-moved")
		run("view.paneBottom")
	}
	run("view.outline")
	run("view.source")
	run("view.notes")
	snap("4-hidden")

	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	run("view.outline")
	run("view.notes")
	run("view.source")
	snap("5-phone")
	for _, id := range []string{"view.production", "view.help", "view.changes"} {
		run(id)
	}
	snap("7-phone-all")
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	snap("6-all-tabs")
}

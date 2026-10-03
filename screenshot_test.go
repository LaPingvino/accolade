package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// TestScreenshots renders the main window in each colour scheme to PNG
// files for reviewing the design. It only runs when SHOT_DIR is set:
//
//	SHOT_DIR=/tmp/shots go test -run TestScreenshots .
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)
	w := NewMainWindow(&Application{fyneApp: a})
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) / 3)

	for _, scheme := range []string{"light", "dark", "sepia"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		img := w.fyneWindow.Canvas().Capture()
		f, err := os.Create(filepath.Join(dir, "accolade-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

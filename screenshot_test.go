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

// TestMarkdownScreenshots renders a Markdown document open, with its
// outline and preview, in the light and dark schemes (SHOT_DIR as above).
func TestMarkdownScreenshots(t *testing.T) {
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
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	md := filepath.Join(t.TempDir(), "story.md")
	if err := os.WriteFile(md, []byte(story), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(md); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetCursorOffset(len([]rune(story)) / 3)
	w.toggleOutline()
	w.preview()
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "accolade-markdown-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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
	t.Cleanup(w.stopTimers)
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
	t.Cleanup(w.stopTimers)
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

// TestDocumentTypeScreenshot renders View > Document Type open over a
// Markdown document (SHOT_DIR as above). The test window draws no main
// menu, so the View menu and its submenu are shown as pop-ups where they
// would open.
func TestDocumentTypeScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	p := filepath.Join(t.TempDir(), "story.txt")
	if err := os.WriteFile(p, []byte(story), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.setDocumentType(true)
	c := w.fyneWindow.Canvas()
	var view *fyne.Menu
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "View" {
			view = m
		}
	}
	main := widget.NewPopUpMenu(view, c)
	main.ShowAtPosition(fyne.NewPos(120, 4))
	types := documentTypeItems(t, w)
	sub := widget.NewPopUpMenu(fyne.NewMenu("", types...), c)
	sub.ShowAtPosition(fyne.NewPos(120+main.MinSize().Width, main.MinSize().Height-40)) // beside its row
	f, err := os.Create(filepath.Join(dir, "accolade-document-type.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, c.Capture()); err != nil {
		t.Fatal(err)
	}
}

// TestImportExportScreenshots renders, for a Markdown document, the File
// menu with Import, and the Export dialog (SHOT_DIR as above); the
// Export dialog with pandoc's formats too, and without them.
func TestImportExportScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shot := func(w *MainWindow, name string) {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	open := func() *MainWindow {
		w, _ := newDialogTestWindow(t)
		w.fyneWindow.Resize(fyne.NewSize(1000, 650))
		p := filepath.Join(t.TempDir(), "story.md")
		if err := os.WriteFile(p, []byte(story), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		return w
	}
	w := open()
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "File" {
			widget.NewPopUpMenu(m, w.fyneWindow.Canvas()).ShowAtPosition(fyne.NewPos(4, 4))
		}
	}
	shot(w, "accolade-markdown-import-menu.png")

	w = open()
	ed := newMarkdownExportDialog(w)
	ed.format.SetSelected("Word (DOCX)")
	ed.Show()
	shot(w, "accolade-markdown-export.png")
	ed.dialog.Hide()
	ed.format.OnChanged = nil
	ed.format.SetSelected("PDF")
	ed.formatChanged()
	ed.Show()
	ed.format.Tapped(&fyne.PointEvent{}) // its list open
	shot(w, "accolade-markdown-export-formats.png")

	old := pandocPath
	pandocPath = func() string { return "" }
	defer func() { pandocPath = old }()
	w = open()
	ed = newMarkdownExportDialog(w)
	ed.Show()
	ed.format.Tapped(&fyne.PointEvent{})
	shot(w, "accolade-markdown-export-no-pandoc.png")
}

// TestSourceScreenshots renders a script with its source beside it, the
// cursor in a scene whose linked passage is lit, another picked, in the
// light and dark schemes (SHOT_DIR as above).
func TestSourceScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	s := w.sourcePane()
	s.selected = passageNamed(t, s.passages, "The next morning")
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "ANNA")))
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		s.list.Refresh()
		f, err := os.Create(filepath.Join(dir, "accolade-source-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// a scene with an ambiguous and a missing link
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "EXT. GARDEN - DAY")))
	f, err := os.Create(filepath.Join(dir, "accolade-source-doubt.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, w.fyneWindow.Canvas().Capture())
	// the source kept in the script
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) - 40)
	s.follow()
	g, err := os.Create(filepath.Join(dir, "accolade-source-inside.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	png.Encode(g, w.fyneWindow.Canvas().Capture())
}

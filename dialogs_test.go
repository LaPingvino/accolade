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

func newDialogTestWindow(t *testing.T) (*MainWindow, *Application) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)
	app := &Application{fyneApp: a}
	w := NewMainWindow(app)
	app.windows = append(app.windows, w)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	return w, app
}

func capture(t *testing.T, c fyne.Canvas, name string) {
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		f, _ := os.Create(filepath.Join(dir, name))
		png.Encode(f, c.Capture())
		f.Close()
	}
}



// No dialog has an empty dismiss button any more, and Preferences is a
// window of its own.
func TestDialogsHaveNoEmptyButton(t *testing.T) {
	w, app := newDialogTestWindow(t)
	empty := func(o fyne.CanvasObject) int {
		n := 0
		var walk func(fyne.CanvasObject)
		walk = func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == "" && b.Icon == nil && b.Visible() {
				n++
			}
			if c, ok := o.(*fyne.Container); ok {
				for _, x := range c.Objects {
					walk(x)
				}
			}
			if wr, ok := o.(fyne.Widget); ok {
				r := test.WidgetRenderer(wr)
				for _, x := range r.Objects() {
					walk(x)
				}
			}
		}
		walk(o)
		return n
	}
	tp := NewTitlePageDialog(w)
	tp.Show()
	if n := empty(w.fyneWindow.Canvas().Overlays().Top()); n != 0 {
		t.Errorf("title page dialog: %d empty buttons", n)
	}
	capture(t, w.fyneWindow.Canvas(), "titlepage.png")
	tp.dialog.Hide()
	ed := NewExportDialog(w)
	ed.Show()
	if n := empty(w.fyneWindow.Canvas().Overlays().Top()); n != 0 {
		t.Errorf("export dialog: %d empty buttons", n)
	}
	capture(t, w.fyneWindow.Canvas(), "export.png")
	ed.dialog.Hide()
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.Show()
	if pd.win == w.fyneWindow || w.fyneWindow.Canvas().Overlays().Top() != nil {
		t.Error("preferences is not a window of its own")
	}
	pd.win.Resize(fyne.NewSize(640, 420)) // smaller than its content: it scrolls, the buttons stay
	capture(t, pd.win.Canvas(), "preferences.png")
	pd.cancel()
}





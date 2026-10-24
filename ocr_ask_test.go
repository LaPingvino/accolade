package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// The language OCR reads in is guessed from the file's name, else the
// preferences, else English, among those it can read.
func TestGuessOCRLanguage(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	prev := w.settings.GetString("script-language")
	t.Cleanup(func() { w.settings.SetString("script-language", prev) })
	w.settings.SetString("script-language", "nl")
	langs := []string{"en", "he", "nl"}
	for name, want := range map[string]string{"he": "he", "": "nl", "ru": "nl"} {
		if got := w.guessOCRLanguage(name, langs); got != want {
			t.Errorf("file says %q: %q, want %q", name, got, want)
		}
	}
	w.settings.SetString("script-language", "eo")
	if got := w.guessOCRLanguage("", langs); got != "en" {
		t.Errorf("nothing it can read: %q", got)
	}
	if got := languageLabel("he"); got != "עברית (he)" {
		t.Errorf("label %q", got)
	}
}

// Before a scanned PDF is read, a dialog asks its language, with the one
// guessed chosen.
func TestScanAsksLanguage(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	if langs, _ := ocrLanguages(); len(langs) == 0 {
		t.Skip("no OCR")
	}
	w.importPDFAsync("testdata/scanned-tv-episode.pdf")
	top := w.fyneWindow.Canvas().Overlays().Top()
	var pick *widget.Select
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if s, ok := o.(*widget.Select); ok {
			pick = s
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, x := range c.Objects {
				walk(x)
			}
		}
		if wr, ok := o.(fyne.Widget); ok {
			for _, x := range test.WidgetRenderer(wr).Objects() {
				walk(x)
			}
		}
	}
	if top != nil {
		walk(top)
	}
	if pick == nil {
		t.Fatal("no language question before OCR")
	}
	if pick.Selected == "" {
		t.Error("no language guessed")
	}
	if w.imported != nil {
		t.Error("reading started before the question was answered")
	}
}

// tapDialogButton taps the button labelled label in the window's dialog.
func tapDialogButton(t *testing.T, w *MainWindow, label string) {
	t.Helper()
	var found *widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == label {
			found = b
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, x := range c.Objects {
				walk(x)
			}
		}
		if wr, ok := o.(fyne.Widget); ok {
			for _, x := range test.WidgetRenderer(wr).Objects() {
				walk(x)
			}
		}
	}
	if top := w.fyneWindow.Canvas().Overlays().Top(); top != nil {
		walk(top)
	}
	if found == nil {
		t.Fatalf("no %q button in a dialog", label)
	}
	test.Tap(found)
}

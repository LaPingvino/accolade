package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// garbledScanPDF is a one-page scan (a blank page-sized image) with lines
// over it as its text layer, as a poor OCR makes it: stray marks of the
// scan at the lines' ends, glued punctuation.
func garbledScanPDF(t *testing.T) string {
	t.Helper()
	const w, h = 850, 1100
	var pix bytes.Buffer
	z := zlib.NewWriter(&pix)
	z.Write(bytes.Repeat([]byte{0xff}, w*h))
	z.Close()
	var content strings.Builder
	content.WriteString("q 612 0 0 792 0 0 cm /Im0 Do Q\nBT 3 Tr /F1 12 Tf 14 TL 72 700 Td\n")
	for _, l := range []string{
		"_ MARTA ol ruen |", "The boat is late again. We wait by the pier. |", "(she folds the map } ond puts it away) ~",
		"CUT TO: -", "a THE OLD`MILL AT DUSK :", "Rain on the roof. Ivo and the dog come ~", "in from the yard, shaking off the wa- |",
		"ter, and sit down by the stove. _", "Ivo_(pointing at the clock on the wall)", "Half past six and still no word from her |",
	} {
		fmt.Fprintf(&content, "(%s) Tj T*\n", strings.NewReplacer("(", `\(`, ")", `\)`).Replace(l))
	}
	content.WriteString("ET")
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> /XObject << /Im0 6 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>",
		fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
			w, h, pix.Len(), pix.String()),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offs := make([]int, len(objs))
	for i, o := range objs {
		offs[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	path := filepath.Join(t.TempDir(), "garbled.pdf")
	if err := os.WriteFile(path, b.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A scan whose own text looks garbled: Accolade asks whether to read it
// with OCR instead, never on its own; Keep the Text reads the text as it
// is. A PDF Accolade prints is never asked about.
func TestGarbledTextOffersOCR(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	if langs, _ := ocrLanguages(); len(langs) == 0 {
		t.Skip("no OCR")
	}
	path := garbledScanPDF(t)
	if n := pdfPagesOf(path); n != (pdfPages{scanned: 0, garbled: 1, pages: 1}) {
		t.Fatalf("pages: %+v", n)
	}
	w.importPDFAsync(path)
	if w.imported != nil {
		t.Fatal("reading started before the question was answered")
	}
	tapDialogButton(t, w, "Keep the Text")
	if text := waitForText(t, w); !strings.Contains(text, "stove") {
		t.Errorf("the text kept: %q", text)
	}

	clean := filepath.Join(t.TempDir(), "clean.pdf")
	if err := exportAs("INT. A - DAY\n\nPia waits by the window, the kettle singing.\n", clean, "pdf", exportJob{}); err != nil {
		t.Fatal(err)
	}
	if n := pdfPagesOf(clean); n.garbled != 0 || n.scanned != 0 {
		t.Errorf("printed: %+v", n)
	}
}

// Read with OCR reads the garbled pages with OCR; with scans as well, a
// tick says whether the garbled pages are read too.
func TestGarbledAskReads(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	var gotLang string
	var gotText, kept bool
	read := func(lang string, ocrText bool) { gotLang, gotText = lang, ocrText }
	w.askOCRLanguage(pdfPages{garbled: 3, pages: 4}, "he", []string{"en", "he"}, true, read, func() { kept = true })
	tapDialogButton(t, w, "Read with OCR")
	if gotLang != "he" || !gotText || kept {
		t.Errorf("read with OCR: %q %v (kept %v)", gotLang, gotText, kept)
	}
	gotText = false
	w.askOCRLanguage(pdfPages{scanned: 1, garbled: 3, pages: 4}, "en", []string{"en", "he"}, true, read, func() { kept = true })
	tapDialogButton(t, w, "Read")
	if gotLang != "en" || !gotText || kept {
		t.Errorf("scans and garbled: %q %v (kept %v)", gotLang, gotText, kept)
	}
	// the tick taken off: the scans are read, the text kept
	w.askOCRLanguage(pdfPages{scanned: 1, garbled: 3, pages: 4}, "en", []string{"en", "he"}, true, read, func() { kept = true })
	var tick *widget.Check
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if c, ok := o.(*widget.Check); ok {
			tick = c
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
	walk(w.fyneWindow.Canvas().Overlays().Top())
	if tick == nil {
		t.Fatal("no tick for the garbled pages")
	}
	tick.SetChecked(false)
	tapDialogButton(t, w, "Read")
	if gotText || kept {
		t.Errorf("tick off: ocrText %v (kept %v)", gotText, kept)
	}
}

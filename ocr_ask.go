// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/pdfin"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// guessOCRLanguage is the language a scan is probably in: the file name's
// (script.he.pdf), else the preferences' script language, else English;
// one of langs, those OCR can read.
func (w *MainWindow) guessOCRLanguage(fromName string, langs []string) string {
	for _, l := range []string{fromName, w.settings.GetString("script-language"), "en"} {
		if l != "" && slices.Contains(langs, l) {
			return l
		}
	}
	return langs[0]
}

// selfNames are languages' own names that the display tables lack.
var selfNames = map[string]string{"tok": "toki pona", "hekaya": "Hekaya, العربية",
	// Hindi (Lexington's hi: English Fountain's words first, reading all)
	// and its variants per script, hi-Latn and hi-Deva
	"hi-Latn": "Hindi (Latin)", "hi-Deva": "Hindi (Devanagari)", "hi": "Hindi"}

// languageLabel names a language in itself, with its code: "עברית (he)".
// A name that ends in brackets takes the code inside them: "Hindi
// (Latin, hi-Latn)".
func languageLabel(code string) string {
	name := selfName(code)
	switch {
	case name == "":
		return code
	case strings.HasSuffix(name, ")"):
		return fmt.Sprintf("%s, %s)", strings.TrimSuffix(name, ")"), code)
	}
	return fmt.Sprintf("%s (%s)", name, code)
}

// selfName is a language's name in itself ("עברית", "Hindi (Latin)"),
// "" for a code with none.
func selfName(code string) string {
	if name := selfNames[code]; name != "" {
		return name
	}
	return display.Self.Name(language.Make(code))
}

// pdfPages are what OCR could read in a PDF: its scanned pages (no text),
// the scans whose own text layer looks garbled (a poor OCR's, made with
// the scan), of all its pages.
type pdfPages struct{ scanned, garbled, pages int }

// pdfPagesOf counts a PDF's scanned and garbled pages (nothing to read
// with OCR if it cannot be read).
func pdfPagesOf(path string) pdfPages {
	layers, err := pdfin.TextLayers(path)
	if err != nil {
		return pdfPages{}
	}
	n := pdfPages{garbled: pdfin.CountGarbled(layers), pages: len(layers)}
	for _, l := range layers {
		if l.Quality.Lines == 0 {
			n.scanned++
		}
	}
	return n
}

// readPDFAsking reads a PDF: if it has pages OCR should read (scans, or
// scans whose text looks garbled) and there is OCR, it asks first what
// language they are in (guessing nameLang, the file name's) and, for the
// garbled ones, whether to read them with OCR at all, never on its own;
// then read gets the language and whether OCR reads the garbled pages in
// place of their text. Else plain reads it as it is.
func (w *MainWindow) readPDFAsking(path, nameLang string, read func(lang string, ocrText bool), plain func()) {
	n := pdfPagesOf(path)
	langs, tesseract := ocrLanguages()
	if n.scanned+n.garbled == 0 || len(langs) == 0 {
		plain()
		return
	}
	w.askOCRLanguage(n, w.guessOCRLanguage(nameLang, langs), langs, tesseract, read, plain)
}

// askOCRLanguage asks, before scanned pages are read, what language they
// are in, with guess chosen; read gets the language (Lexington's code)
// and whether OCR reads the scanned pages with text of their own again
// too (asked when some of that text looks garbled). With
// garbled pages and no scans, it asks whether to read them with OCR at
// all: plain keeps the text.
func (w *MainWindow) askOCRLanguage(n pdfPages, guess string, langs []string, tesseract bool, read func(lang string, ocrText bool), plain func()) {
	labels := make([]string, len(langs))
	for i, l := range langs {
		labels[i] = languageLabel(l)
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelectedIndex(slices.Index(langs, guess))
	fyne.SetAccessibleLabel(pick, "The script's language")
	title, yes, no := "Read the Scanned Pages", "Read", "Cancel"
	what := fmt.Sprintf("%d of the %d pages are scans: text recognition (OCR) reads them, a page at a time, which takes a while.", n.scanned, n.pages)
	if n.scanned == n.pages {
		what = fmt.Sprintf("This PDF is scanned: text recognition (OCR) reads its %d pages, a page at a time, which takes a while.", n.pages)
	}
	if n.scanned == 0 {
		title, yes, no = "The PDF's Text Looks Garbled", "Read with OCR", "Keep the Text"
		what = fmt.Sprintf("This PDF's text looks garbled — read it with OCR instead? It is a scan with text of its own "+
			"(made by the scanner), and on %d of its %d pages that text has stray marks and broken lines. Text recognition (OCR) "+
			"reads the scanned pages again, a page at a time, which takes a while. Keep the Text reads the text as it is.", n.garbled, n.pages)
	}
	intro := widget.NewLabel(what)
	intro.Wrapping = fyne.TextWrapWord
	items := []fyne.CanvasObject{intro}
	also := widget.NewCheck(fmt.Sprintf("Read the scanned pages with text again too (on %d of them it looks garbled)", n.garbled), nil)
	also.SetChecked(true)
	if n.scanned > 0 && n.garbled > 0 {
		items = append(items, also)
	}
	more := "Only the languages whose data Tesseract has are listed: install more (tesseract-data-heb, …) to read others."
	if !tesseract {
		more = "Without Tesseract installed, the built-in OCR reads English only: install Tesseract and its languages' data to read others."
	}
	note := widget.NewLabel(more)
	note.Wrapping = fyne.TextWrapWord
	note.Importance = widget.LowImportance
	items = append(items, widget.NewForm(widget.NewFormItem("The script's language", pick)), note)
	d := dialog.NewCustomConfirm(title, yes, no, container.NewVBox(items...), func(ok bool) {
		switch {
		case ok && pick.SelectedIndex() >= 0:
			read(langs[pick.SelectedIndex()], n.garbled > 0 && (n.scanned == 0 || also.Checked))
		case !ok && n.scanned == 0:
			plain()
		}
	}, w.fyneWindow)
	d.Resize(fyne.NewSize(480, 320))
	d.Show()
}

package main

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
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

// languageLabel names a language in itself, with its code: "עברית (he)".
func languageLabel(code string) string {
	if name := display.Self.Name(language.Make(code)); name != "" {
		return fmt.Sprintf("%s (%s)", name, code)
	}
	return code
}

// askOCRLanguage asks, before scanned pages are read, what language they
// are in, with guess chosen; read gets the language (Lexington's code).
func (w *MainWindow) askOCRLanguage(path string, scanned, pages int, guess string, langs []string, tesseract bool, read func(lang string)) {
	labels := make([]string, len(langs))
	for i, l := range langs {
		labels[i] = languageLabel(l)
	}
	pick := widget.NewSelect(labels, nil)
	pick.SetSelectedIndex(slices.Index(langs, guess))
	fyne.SetAccessibleLabel(pick, "The script's language")
	what := fmt.Sprintf("%d of the %d pages are scans: text recognition (OCR) reads them, a page at a time, which takes a while.", scanned, pages)
	if scanned == pages {
		what = fmt.Sprintf("This PDF is scanned: text recognition (OCR) reads its %d pages, a page at a time, which takes a while.", pages)
	}
	intro := widget.NewLabel(what)
	intro.Wrapping = fyne.TextWrapWord
	more := "Only the languages whose data Tesseract has are listed: install more (tesseract-data-heb, …) to read others."
	if !tesseract {
		more = "Without Tesseract installed, the built-in OCR reads English only: install Tesseract and its languages' data to read others."
	}
	note := widget.NewLabel(more)
	note.Wrapping = fyne.TextWrapWord
	note.Importance = widget.LowImportance
	content := container.NewVBox(intro, widget.NewForm(widget.NewFormItem("The script's language", pick)), note)
	d := dialog.NewCustomConfirm("Read the Scanned Pages", "Read", "Cancel", content, func(ok bool) {
		if ok && pick.SelectedIndex() >= 0 {
			read(langs[pick.SelectedIndex()])
		}
	}, w.fyneWindow)
	d.Resize(fyne.NewSize(480, 300))
	d.Show()
}

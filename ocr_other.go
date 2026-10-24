//go:build !(js && wasm)

package main

import (
	"context"

	"github.com/LaPingvino/lexington/ocrwasm"
	"github.com/LaPingvino/lexington/pdfin"
)

// pdfOCR reads scanned pages in lang (Lexington's code, "" English):
// tesseract if it is installed (with lang's data if it has it), else the
// built-in one (slower, English only); done frees it.
func pdfOCR(lang string) (pdfin.OCR, func()) {
	if t, ok := pdfin.InstalledTesseract(); ok {
		if lang != "" && lang != "en" {
			t.UseLanguage(lang)
		}
		return t, func() {}
	}
	o := ocrwasm.New()
	return o, func() { o.Close(context.Background()) }
}

// ocrLanguages are the languages scanned pages can be read in, and whether
// tesseract reads them (the built-in OCR only reads English).
func ocrLanguages() ([]string, bool) {
	if t, ok := pdfin.InstalledTesseract(); ok {
		return t.Languages(), true
	}
	return []string{"en"}, false
}

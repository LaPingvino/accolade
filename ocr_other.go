//go:build !(js && wasm)

package main

import (
	"context"

	"github.com/LaPingvino/lexington/ocrwasm"
	"github.com/LaPingvino/lexington/pdfin"
)

// pdfOCR reads scanned pages: tesseract if it is installed, else the
// built-in one (slower); done frees it.
func pdfOCR() (pdfin.OCR, func()) {
	if t, ok := pdfin.InstalledTesseract(); ok {
		return t, func() {}
	}
	o := ocrwasm.New()
	return o, func() { o.Close(context.Background()) }
}

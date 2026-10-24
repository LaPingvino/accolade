//go:build js && wasm

package main

import "github.com/LaPingvino/lexington/pdfin"

// pdfOCR: none in the web version (OCR is a WebAssembly program itself,
// several megabytes); scanned pages are left out, the desktop app reads
// them.
func pdfOCR(string) (pdfin.OCR, func()) { return nil, nil }

// ocrLanguages: none, as there is no OCR.
func ocrLanguages() ([]string, bool) { return nil, false }

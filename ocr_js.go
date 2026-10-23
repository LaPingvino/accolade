//go:build js && wasm

package main

import "github.com/LaPingvino/lexington/pdfin"

// pdfOCR: none in the web version (OCR is a WebAssembly program itself,
// several megabytes); scanned pages are left out, the desktop app reads
// them.
func pdfOCR() (pdfin.OCR, func()) { return nil, nil }

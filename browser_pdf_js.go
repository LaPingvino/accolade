//go:build js

package main

import "errors"

// In the browser, printing is the browser's own (browser_js.go): it makes
// no PDFs in the background.
func localBrowser() string { return "" }

func browserPDF(string, string, string) error { return errors.New("no browser to print with") }

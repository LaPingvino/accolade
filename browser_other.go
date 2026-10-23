//go:build !(js && wasm)

package main

// inBrowser: the web version (browser_js.go); not this one.
const inBrowser = false

const (
	openLabel = "Open…"
	saveLabel = "Save"
)

func pickFile(func(string)) {}
func offerDownload(string)  {}
func keepDraft(string)      {}
func draft() (string, bool) { return "", false }

func pageLink() string { return "" }

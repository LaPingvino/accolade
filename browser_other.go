//go:build !(js && wasm)

package main

// inBrowser: the web version (browser_js.go); not this one.
const inBrowser = false

const (
	openLabel = "Open…"
	saveLabel = "Save"
)

func pickFile([]string, func(string)) {}
func offerDownload(string)            {}
func keepDraft(kept)                  {}
func draft() (kept, bool)             { return kept{}, false }

func pageLink() string               { return "" }
func pageBase() string               { return "" }
func setPageKey(string)              {}
func onPageKey(func(string))         {}
func watchdog(*Application)          {}
func keyArrived()                    {}
func browserFullscreen(bool, func()) {}
func onPresence(func(), func())      {}

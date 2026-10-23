//go:build !(js && wasm)

package main

// inBrowser: the web version (browser_js.go); not this one.
const inBrowser = false

func pickFile(func(string)) {}
func offerDownload(string)  {}

//go:build js && wasm

package main

import (
	"os"
	"path"
	"syscall/js"

	"fyne.io/fyne/v2"
)

// In the web version files live in memory (web/memfs.js): opening takes
// a file from the browser's file picker, and what is saved or exported is
// also handed to the browser as a download.
const inBrowser = true

// The File menu says what Open and Save do here.
const (
	openLabel = "Open from Your Computer…"
	saveLabel = "Save (Kept Here, and Downloaded)"
)

// draftPath keeps an untitled script in this browser (the settings
// folder is kept: web/index.html), to come back to after a reload.
func draftPath() string { return path.Join(getConfigDir(), "draft.fountain") }

func keepDraft(text string) { _ = os.WriteFile(draftPath(), []byte(text), 0o644) }

// draft is the untitled script kept, if any.
func draft() (string, bool) {
	b, err := os.ReadFile(draftPath())
	return string(b), err == nil && len(b) > 0
}

// pickFile lets the user choose a file in the browser; open gets its
// path in memory.
func pickFile(open func(path string)) {
	doc := js.Global().Get("document")
	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", ".fountain,.spmd,.txt,.fdx,.pdf")
	var onChange js.Func
	onChange = js.FuncOf(func(js.Value, []js.Value) any {
		onChange.Release()
		files := input.Get("files")
		if files.Length() == 0 {
			return nil
		}
		file := files.Index(0)
		name := path.Base(file.Get("name").String())
		var then js.Func
		then = js.FuncOf(func(_ js.Value, args []js.Value) any {
			then.Release()
			data := make([]byte, args[0].Get("byteLength").Int())
			js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(args[0]))
			p := path.Join("/files", name)
			if os.MkdirAll("/files", 0o755) == nil && os.WriteFile(p, data, 0o644) == nil {
				fyne.Do(func() { open(p) })
			}
			return nil
		})
		file.Call("arrayBuffer").Call("then", then)
		return nil
	})
	input.Call("addEventListener", "change", onChange)
	input.Call("click")
}

// offerDownload hands a file in memory to the browser as a download.
func offerDownload(p string) {
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := js.Global().Get("Blob").New([]any{arr})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	a := js.Global().Get("document").Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", path.Base(p))
	a.Call("click")
	js.Global().Get("URL").Call("revokeObjectURL", url)
}

// pageLink is the page's address (with a shared script's #k=...).
func pageLink() string { return js.Global().Get("location").Get("href").String() }

//go:build js && wasm

package main

import (
	"encoding/json"
	"mime"
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

// The script in the window is kept in this browser (the settings folder
// is kept: web/index.html), to come back to after a reload: its text,
// its language and its file, if it has one.
func draftPath() string { return path.Join(getConfigDir(), "draft.json") }

func keepDraft(d kept) {
	if b, err := json.Marshal(d); err == nil {
		_ = os.WriteFile(draftPath(), b, 0o644)
	}
}

// draft is the script kept, if any.
func draft() (kept, bool) {
	var d kept
	if b, err := os.ReadFile(draftPath()); err == nil && json.Unmarshal(b, &d) == nil {
		return d, d.Text != ""
	}
	b, err := os.ReadFile(path.Join(getConfigDir(), "draft.fountain")) // kept before languages were
	return kept{Text: string(b)}, err == nil && len(b) > 0
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

// fileURL is a link to a file in memory, for the browser.
func fileURL(p, kind string) (js.Value, bool) {
	data, err := os.ReadFile(p)
	if err != nil {
		return js.Undefined(), false
	}
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	blob := js.Global().Get("Blob").New([]any{arr}, map[string]any{"type": kind})
	return js.Global().Get("URL").Call("createObjectURL", blob), true
}

// openFile shows a file (the PDF preview) in a pane beside the editor.
func init() {
	openFile = func(p string) {
		url, ok := fileURL(p, mime.TypeByExtension(path.Ext(p)))
		if !ok {
			return
		}
		frame := previewPane()
		if old := frame.Get("src").String(); old != "" {
			js.Global().Get("URL").Call("revokeObjectURL", old)
		}
		frame.Set("src", url)
	}
}

// previewPane is the preview's frame, in a pane made the first time.
func previewPane() js.Value {
	doc := js.Global().Get("document")
	if f := doc.Call("getElementById", "preview-frame"); f.Truthy() {
		f.Get("parentElement").Get("style").Set("display", "flex")
		return f
	}
	pane := doc.Call("createElement", "div")
	pane.Set("id", "preview-pane")
	pane.Get("style").Set("cssText", "position:fixed;top:0;right:0;bottom:0;width:50vw;display:flex;flex-direction:column;background:#fff;box-shadow:0 0 12px rgba(0,0,0,.3);z-index:10")
	stop := js.FuncOf(func(_ js.Value, args []js.Value) any { args[0].Call("stopPropagation"); return nil })
	for _, ev := range []string{"mousedown", "mouseup", "keydown", "keyup", "wheel"} {
		pane.Call("addEventListener", ev, stop) // the pane's, not the editor's
	}
	bar := doc.Call("createElement", "div")
	bar.Get("style").Set("cssText", "display:flex;justify-content:space-between;align-items:center;padding:4px 8px;font:14px sans-serif")
	bar.Set("textContent", "PDF preview")
	closeBtn := doc.Call("createElement", "button")
	closeBtn.Set("textContent", "Close")
	closeBtn.Call("addEventListener", "click", js.FuncOf(func(js.Value, []js.Value) any {
		pane.Get("style").Set("display", "none")
		return nil
	}))
	bar.Call("appendChild", closeBtn)
	frame := doc.Call("createElement", "iframe")
	frame.Set("id", "preview-frame")
	frame.Get("style").Set("cssText", "flex:1;border:0;width:100%")
	pane.Call("appendChild", bar)
	pane.Call("appendChild", frame)
	doc.Get("body").Call("appendChild", pane)
	return frame
}

// offerDownload hands a file in memory to the browser as a download.
func offerDownload(p string) {
	url, ok := fileURL(p, "")
	if !ok {
		return
	}
	a := js.Global().Get("document").Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", path.Base(p))
	a.Call("click")
	js.Global().Get("URL").Call("revokeObjectURL", url)
}

// pageLink is the page's address (with a shared script's #k=...).
func pageLink() string { return js.Global().Get("location").Get("href").String() }

// browserFullscreen puts the page in the browser's full screen, or takes
// it out; left is called when the browser leaves it on its own (Escape).
func browserFullscreen(on bool, left func()) {
	doc := js.Global().Get("document")
	if on {
		if !doc.Get("fullscreenElement").Truthy() {
			doc.Get("documentElement").Call("requestFullscreen").Call("catch", js.FuncOf(func(js.Value, []js.Value) any { return nil }))
		}
		var change js.Func
		change = js.FuncOf(func(js.Value, []js.Value) any {
			if !doc.Get("fullscreenElement").Truthy() {
				doc.Call("removeEventListener", "fullscreenchange", change)
				change.Release()
				fyne.Do(left)
			}
			return nil
		})
		doc.Call("addEventListener", "fullscreenchange", change)
		return
	}
	if doc.Get("fullscreenElement").Truthy() {
		doc.Call("exitFullscreen")
	}
}

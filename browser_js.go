// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

//go:build js && wasm

package main

import (
	"encoding/json"
	"math"
	"mime"
	"os"
	"path"
	"strings"
	"syscall/js"
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

// pickFile lets the user choose a file with one of exts in the browser;
// open gets its path in memory.
func pickFile(exts []string, open func(path string)) {
	doc := js.Global().Get("document")
	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", strings.Join(exts, ","))
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
			go func() { // not in the page's event: file writes wait
				p := path.Join("/files", name)
				if os.MkdirAll("/files", 0o755) == nil && os.WriteFile(p, data, 0o644) == nil {
					uiDo(func() { open(p) })
				}
			}()
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
	// printFile shows an HTML file in the pane and opens the browser's
	// print dialog for it (Save as PDF among its printers): the browser's
	// fonts, so every script prints
	printFile = func(p string) {
		openFile(p)
		frame := previewPane()
		var loaded js.Func
		loaded = js.FuncOf(func(js.Value, []js.Value) any {
			frame.Call("removeEventListener", "load", loaded)
			loaded.Release()
			frame.Get("contentWindow").Call("print")
			return nil
		})
		frame.Call("addEventListener", "load", loaded)
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

// pageBase is the page's address without its fragment: a shared script's
// link is this page, wherever it is served.
func pageBase() string {
	l := js.Global().Get("location")
	return l.Get("origin").String() + l.Get("pathname").String() + l.Get("search").String()
}

// setPageKey puts a shared script's key in the page's address (#k=...),
// so that a reload comes back to it; "" takes it away. The history gets
// no step for it.
func setPageKey(key string) {
	frag := ""
	if key != "" {
		frag = "#k=" + key
	}
	if js.Global().Get("location").Get("hash").String() == frag {
		return
	}
	js.Global().Get("history").Call("replaceState", js.Null(), "", pageBase()+frag)
}

// onPageKey calls join with the page's address when its fragment changes
// to a shared script's link (one pasted into a page already open).
func onPageKey(join func(link string)) {
	js.Global().Call("addEventListener", "hashchange", js.FuncOf(func(js.Value, []js.Value) any {
		if link := pageLink(); strings.Contains(link, "#k=") {
			go uiDo(func() { join(link) })
		}
		return nil
	}))
}

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
				go uiDo(left)
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

// onPresence calls away when the page is hidden, back when it shows or
// gets the focus again.
func onPresence(away, back func()) {
	doc := js.Global().Get("document")
	doc.Call("addEventListener", "visibilitychange", js.FuncOf(func(js.Value, []js.Value) any {
		if doc.Get("visibilityState").String() == "hidden" {
			go away() // not in the page's event
		} else {
			go back()
		}
		return nil
	}))
	js.Global().Call("addEventListener", "focus", js.FuncOf(func(js.Value, []js.Value) any {
		go back()
		return nil
	}))
}

// Two fingers on the page view zoom it (a pinch): the browser's touches,
// which Fyne passes on one finger at a time.
func init() {
	var on bool
	var startDist float64
	var startScale float32
	dist := func(ts js.Value) float64 {
		a, b := ts.Index(0), ts.Index(1)
		dx := a.Get("clientX").Float() - b.Get("clientX").Float()
		dy := a.Get("clientY").Float() - b.Get("clientY").Float()
		return dx*dx + dy*dy
	}
	doc := js.Global().Get("document")
	opts := map[string]any{"passive": true}
	doc.Call("addEventListener", "touchstart", js.FuncOf(func(_ js.Value, args []js.Value) any {
		ts := args[0].Get("touches")
		if v := pagesInSight.Load(); v != nil && ts.Length() == 2 {
			on, startDist, startScale = true, dist(ts), v.scale
		}
		return nil
	}), opts)
	doc.Call("addEventListener", "touchmove", js.FuncOf(func(_ js.Value, args []js.Value) any {
		ts := args[0].Get("touches")
		if !on || ts.Length() != 2 || startDist <= 0 {
			return nil
		}
		factor := float32(sqrt(dist(ts) / startDist))
		uiDo(func() {
			if v := pagesInSight.Load(); v != nil {
				v.pinch(startScale, factor)
			}
		})
		return nil
	}), opts)
	end := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if args[0].Get("touches").Length() < 2 && on {
			on = false
			uiDo(func() {
				if v := pagesInSight.Load(); v != nil {
					v.pinchEnd()
				}
			})
		}
		return nil
	})
	doc.Call("addEventListener", "touchend", end, opts)
	doc.Call("addEventListener", "touchcancel", end, opts)
}

func sqrt(x float64) float64 { return math.Sqrt(x) }

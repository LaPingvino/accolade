//go:build js && wasm

package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall/js"
	"time"

	"fyne.io/fyne/v2"
)

// The web version's watchdog (web/index.html): the app's main loop marks
// itself alive every second; when it stops, the page offers a restart and
// a bug report, with what fixing it takes (accoladeReport).

var appState atomic.Value // what the windows were doing, as of the last beat

func watchdog(app *Application) {
	go func() {
		for range time.Tick(time.Second) {
			fyne.Do(func() {
				appState.Store(stateOf(app))
				js.Global().Set("accoladeAlive", js.Global().Get("Date").Call("now"))
			})
		}
	}()
	js.Global().Set("accoladeReport", js.FuncOf(func(js.Value, []js.Value) any { return report() }))
}

// stateOf is the windows' state, without the script's text.
func stateOf(app *Application) string {
	s := fmt.Sprintf("windows: %d\n", len(app.windows))
	for _, w := range app.windows {
		name := "untitled"
		if w.currentFile != "" {
			name = filepath.Base(w.currentFile)
		}
		mode := "-"
		if w.vim != nil {
			mode = w.vim.mode.String()
		}
		peers := "not shared"
		if w.shared != nil {
			peers = fmt.Sprintf("shared, %d peers, other version: %v", w.shared.peers, w.shared.other)
		}
		s += fmt.Sprintf("- file %q, %d runes, changed %v, keys %s (vim mode %v), format %s, language %q, %s, outline %v, notes %v, fullscreen %v\n",
			name, len([]rune(w.textEditor.Text())), w.hasChanges, w.editorKeys(), mode,
			w.settings.GetString("script-format"), w.language, peers, w.outlineVisible, w.notesVisible, w.fyneWindow.FullScreen())
	}
	return s
}

// report is the app's side of a bug report: its version, its state as of
// the last beat, and where every goroutine is.
func report() string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	state, _ := appState.Load().(string)
	return fmt.Sprintf("version: %s (%s)\ngoroutines: %d\n\nstate (last beat):\n%s\nstacks:\n%s",
		appVersion(), runtime.Version(), runtime.NumGoroutine(), state, buf)
}

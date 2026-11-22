package main

// Keystrokes in the script with Accolade's own blocks folded and its
// source links drawn as chips (block_folds.go): the per-key cost must
// stay flat. Run with
//
//	go test -run '^$' -bench 'BenchmarkScriptKey' -benchmem -count 6

import (
	"strings"
	"testing"
)

func BenchmarkScriptKey(b *testing.B) {
	// the sample: tail blocks, an OMITTED scene, links on most lines
	b.Run("sample", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		w.openSample()
		if !strings.HasPrefix(w.textEditor.Text(), "Title: A Chameleon") {
			b.Fatal("the sample is not open")
		}
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		at := strings.Index(e.Text(), "Noon heat.")
		e.SetCursorOffset(at)
		for b.Loop() {
			e.TypedRune('x')
		}
	})
	// the novel's screenplay, its 200 links, the novel kept in it
	b.Run("novel", func(b *testing.B) {
		w, _, at := benchWindow(b, true)
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		e.SetCursorOffset(at)
		for b.Loop() {
			e.TypedRune('x')
		}
	})
	// a cursor move down and up again in the novel's screenplay
	b.Run("novel-move", func(b *testing.B) {
		w, _, at := benchWindow(b, true)
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		e.SetCursorOffset(at)
		for b.Loop() {
			e.Key("Down", false)
			e.Key("Up", false)
		}
	})
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

// Keystrokes in the script with Accolade's own blocks folded and its
// source links drawn as chips (block_folds.go): the per-key cost must
// stay flat. Run with
//
//	go test -run '^$' -bench 'BenchmarkScriptKey' -benchmem -count 6

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
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
	// the sample with the timeline strip over the editor (timeline_strip.go)
	b.Run("sample-strip", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		w.openSample()
		w.toggleStrip()
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		e.SetCursorOffset(strings.Index(e.Text(), "Noon heat."))
		for b.Loop() {
			e.TypedRune('x')
		}
	})
	// the Devanagari sample (Lekhan monospace: its clusters shaped by
	// go-text with Courier Badi's psts width lookups), a letter typed in
	// a line of dialogue
	b.Run("hi-Deva", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		sample, err := os.ReadFile(filepath.Join("examples", "sample.hi-Deva.fountain"))
		if err != nil {
			b.Fatal(err)
		}
		w.setLanguage("hi-Deva")
		w.textEditor.SetText(string(sample))
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		e.SetCursorOffset(len([]rune(e.Text()[:strings.Index(e.Text(), "वह मेरा दोस्त है!")])))
		for b.Loop() {
			e.TypedRune('क')
		}
	})
	// the same, with the editor's texts measured after each key as a
	// painter measures and draws them: Fyne keeps both by the text, so
	// what is shaped is what the key changed (the test driver paints
	// nothing, so without this the grid's Devanagari is never shaped)
	b.Run("hi-Deva-drawn", func(b *testing.B) {
		w, _ := newBenchWindow(b)
		sample, err := os.ReadFile(filepath.Join("examples", "sample.hi-Deva.fountain"))
		if err != nil {
			b.Fatal(err)
		}
		w.setLanguage("hi-Deva")
		w.textEditor.SetText(string(sample))
		realTimers(b)
		e := w.textEditor
		w.fyneWindow.Canvas().Focus(e)
		e.SetCursorOffset(len([]rune(e.Text()[:strings.Index(e.Text(), "वह मेरा दोस्त है!")])))
		measure := func() {
			for _, o := range test.LaidOutObjects(e) {
				if t, ok := o.(*canvas.Text); ok && t.Visible() && t.Text != "" {
					t.MinSize() // the driver measures it in its FontSource (the editor theme's)
				}
			}
		}
		measure()
		for b.Loop() {
			e.TypedRune('क')
			measure()
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

// A cursor move in the bench novel's script, the novel kept in it (its
// block folded): what it allocates, apart from timing (BenchmarkScriptKey
// times it). The folds' rows reuse their room.
func TestFoldMoveAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the bench novel")
	}
	holdUI(t)
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	md, script := benchFixture()
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(script), 0o644)
	os.WriteFile(filepath.Join(dir, "story.md"), []byte(md), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	e := w.textEditor
	if len(w.scriptFolds.blockRanges()) == 0 {
		t.Fatal("the source block is not folded")
	}
	e.SetCursorOffset(strings.Index(e.Text(), "INT. PLACE 100"))
	n := testing.AllocsPerRun(5, func() {
		e.Key("Down", false)
		e.Key("Up", false)
	})
	t.Logf("a Down and Up: %.0f allocations", n)
}

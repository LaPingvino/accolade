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

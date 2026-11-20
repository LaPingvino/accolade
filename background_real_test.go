//go:build !race

// The debounce and the background work as the app has them, not at
// once as the other tests run them (background_test.go). Under the test
// driver their functions run on a timer's or a goroutine's goroutine,
// beside the test's, which the race detector rightly flags; the real
// drivers run them on the UI goroutine. So not under -race.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LaPingvino/accolade/internal/prose"
)

func realPaths(t *testing.T) {
	debounceAtOnce, backgroundAtOnce = false, false
	t.Cleanup(func() {
		time.Sleep(100 * time.Millisecond) // what was under way when the test ended is done
		debounceAtOnce, backgroundAtOnce = true, true
	})
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("waited in vain for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Typing puts the pane's follow off until a pause, and it runs once.
func TestFollowAfterTypingPause(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	s := w.sourcePane()
	realPaths(t)
	e := w.textEditor
	at := len([]rune(strings.SplitAfter(e.Text(), "Anna at the door.")[0]))
	for _, r := range " Rain." {
		e.Replace(at, at, string(r))
		at++
		if w.followTimer == nil {
			t.Fatal("no follow pending after a keystroke")
		}
	}
	if sameString(s.modelFor, e.Text()) {
		t.Error("the model was made during typing")
	}
	text := e.Text()
	waitFor(t, "the follow after the pause", func() bool { return w.followTimer == nil && sameString(s.modelFor, text) })
	time.Sleep(50 * time.Millisecond) // the follow done
	// a cursor move alone follows at once
	e.SetCursorOffset(0)
	if w.followTimer != nil || len(s.lit) != 0 {
		t.Errorf("cursor move: pending %v, lit %v", w.followTimer != nil, s.lit)
	}
}

// A long source is parted into passages in the background; the pane
// shows them when they are in.
func TestLongSourceReadInBackground(t *testing.T) {
	long := sourceStory + strings.Repeat("Another paragraph of the story, long enough to count.\n\n", 2000)
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(sourceScript), 0o644)
	os.WriteFile(filepath.Join(dir, "story.md"), []byte(long), 0o644)
	w.LoadFile(p)
	realPaths(t)
	w.toggleSource()
	s := w.sourcePane()
	waitFor(t, "the passages", func() bool { return s.loading == nil && len(s.passages) > 2000 })
	waitFor(t, "every row measured", func() bool {
		if s.measureT != nil || len(s.exact) != len(s.passages) {
			return false
		}
		for _, ok := range s.exact {
			if !ok {
				return false
			}
		}
		return true
	})
}

// Choose Source and Import read in the background.
func TestChooseAndImportInBackground(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	realPaths(t)
	var b bytes.Buffer
	if err := prose.FromMarkdown("# Novel\n\nIt was day.\n").WriteDOCX(&b); err != nil {
		t.Fatal(err)
	}
	docx := filepath.Join(t.TempDir(), "novel.docx")
	os.WriteFile(docx, b.Bytes(), 0o644)

	w.chooseSourceFile(docx)
	waitFor(t, "the source", func() bool { return w.sourceRef().Origin == sourceKept })

	w.importDocument(docx)
	waitFor(t, "the import", func() bool { return w.markdown && strings.Contains(w.textEditor.Text(), "It was day.") })
}

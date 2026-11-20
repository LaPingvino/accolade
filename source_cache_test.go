package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The source pane keeps what it found while nothing changed (the tail
// blocks per text, the links' matches per passages, the stats per text,
// the editor's layout per revision). None of it may outlive a change:
// these change the script, the source, undo and redo, choose another
// source, move it into the script and back, and open another script,
// and check what the pane shows against a pane made anew.

// fresh is what a pane made anew would find for the window now.
func fresh(w *MainWindow) *sourceModel {
	md, _ := w.sourceText(w.sourceRef())
	return newSourceModel(passagesOf(md), w.textEditor.Text(), w.sceneStarts())
}

func sameModel(t *testing.T, what string, got, want *sourceModel) {
	t.Helper()
	if len(got.passages) != len(want.passages) || len(got.matches) != len(want.matches) {
		t.Fatalf("%s: %d passages %d links, want %d and %d", what, len(got.passages), len(got.matches), len(want.passages), len(want.matches))
	}
	for i := range got.matches {
		g, x := got.matches[i], want.matches[i]
		if g.Kind != x.Kind || len(g.Passages) != len(x.Passages) || (len(g.Passages) > 0 && g.Passages[0] != x.Passages[0]) {
			t.Errorf("%s: link %q %+v, want %+v", what, got.links[i].Quote, g, x)
		}
	}
	for i := range got.status {
		if got.status[i] != want.status[i] {
			t.Errorf("%s: passage %d %v, want %v", what, i, got.status[i], want.status[i])
		}
	}
}

func TestSourceCachesFollowChanges(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	s := w.sourcePane()
	e := w.textEditor
	check := func(what string) {
		t.Helper()
		s.follow()
		sameModel(t, what, s.current(), fresh(w))
		if got, want := computeStats(e.Text(), e.CursorOffset(), w.sceneStarts()), countFresh(e.Text(), e.CursorOffset(), w.sceneStarts()); got != want {
			t.Errorf("%s: stats %+v, want %+v", what, got, want)
		}
	}
	check("opened")
	missing := passageCount(s.current(), linkMissing)

	// the script: a missing link edited to quote a passage
	at := strings.Index(e.Text(), "Nothing like this")
	at = len([]rune(e.Text()[:at]))
	e.Replace(at, at+len("Nothing like this is in the story at all"), "The next morning the garden was white with frost")
	check("link edited")
	if passageCount(s.current(), linkMissing) != missing-1 {
		t.Errorf("the edited link is still missing")
	}
	e.Undo()
	check("undone")
	if passageCount(s.current(), linkMissing) != missing {
		t.Errorf("after undo: %d missing, want %d", passageCount(s.current(), linkMissing), missing)
	}
	e.Redo()
	check("redone")

	// the source on disk: the passage the link found is gone
	os.WriteFile(s.ref.Path, []byte("# Only\n\nThe rain kept falling.\n"), 0o644)
	s.modTime = s.modTime.Add(-1)
	check("source changed")
	if len(s.passages) != 2 {
		t.Errorf("source changed: %d passages", len(s.passages))
	}

	// another source chosen
	other := filepath.Join(t.TempDir(), "novel.md")
	os.WriteFile(other, []byte(sourceStory), 0o644)
	if err := w.useSource(other); err != nil {
		t.Fatal(err)
	}
	w.showSource()
	check("another source")

	// kept in the script, then edited there, then out again
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	check("moved in")
	if s.ref.Origin != sourceInside {
		t.Fatalf("not inside: %+v", s.ref)
	}
	at = strings.Index(e.Text(), "The next morning")
	at = len([]rune(e.Text()[:at]))
	e.Replace(at, at+len("The next morning"), "That morning")
	check("source edited in the script")
	if src, _, _, _ := sourceBlock(e.Text()); src == 0 {
		t.Error("no block")
	}
	e.Undo()
	check("source edit undone")
	if err := w.keepSourceInside(false); err != nil {
		t.Fatal(err)
	}
	check("moved out")
	if s.ref.Origin == sourceInside {
		t.Error("still inside")
	}

	// another script
	lone := filepath.Join(filepath.Dir(p), "lone.fountain")
	os.WriteFile(lone, []byte("INT. ROOM - DAY\n\nNothing.\n"), 0o644)
	w.LoadFile(lone)
	s.follow()
	if len(s.passages) != 0 || computeStats(e.Text(), 0, w.sceneStarts()).Scenes != 1 {
		t.Errorf("another script: %d passages, %+v", len(s.passages), computeStats(e.Text(), 0, w.sceneStarts()))
	}
}

// The stats of the same text with other scene starts are counted again.
func TestStatsCacheKeysOnStarts(t *testing.T) {
	text := "INT. A\n\nx\n\nBIN. B\n\ny\n"
	en := computeStats(text, 0, []string{"INT."})
	nl := computeStats(text, 0, []string{"INT.", "BIN."})
	if en.Scenes == nl.Scenes {
		t.Errorf("same scenes with English and Dutch starts: %d", en.Scenes)
	}
	if again := computeStats(text, 0, []string{"INT."}); again != en {
		t.Errorf("English again: %+v, want %+v", again, en)
	}
}

// countFresh is computeStats with nothing remembered.
func countFresh(text string, cursor int, starts []string) scriptStats {
	counted.Lock()
	counted.ok = false
	counted.Unlock()
	return computeStats(text, cursor, starts)
}

func passageCount(m *sourceModel, k matchKind) int { return m.count(k) }

// What Choose Source read for one script is not given to another opened
// meanwhile, nor kept once another source was chosen since.
func TestChosenSourceForItsScript(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	novel := filepath.Join(t.TempDir(), "novel.md")
	gen, script, bundle := w.sourceGen, w.currentFile, w.bundle
	other := filepath.Join(filepath.Dir(p), "other.fountain")
	os.WriteFile(other, []byte("INT. X - DAY\n"), 0o644)
	w.LoadFile(other)
	if w.takeChosenSource(gen, script, bundle, novel, "# Novel\n", nil) || w.chosenSource(other) != "" || w.chosenSource(p) != "" {
		t.Error("given to another script")
	}
	w.LoadFile(p)
	w.sourceGen++
	if w.takeChosenSource(gen, script, bundle, novel, "# Novel\n", nil) {
		t.Error("kept after another was chosen")
	}
	if !w.takeChosenSource(w.sourceGen, script, bundle, novel, "# Novel\n", nil) || w.sourceRef().Origin != sourceKept {
		t.Errorf("not kept: %+v", w.sourceRef())
	}
}

// A source chosen again replaces the copy kept of the one before: its
// folder goes (the browser's storage filled with old copies).
func TestOldKeptCopyDropped(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	os.WriteFile(a, []byte("# A\n"), 0o644)
	os.WriteFile(b, []byte("# B\n"), 0o644)
	if err := w.useSource(a); err != nil {
		t.Fatal(err)
	}
	first := w.sourceRef().Path
	if err := w.useSource(b); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(first)); !os.IsNotExist(err) {
		t.Errorf("the old copy's folder is still there: %v", err)
	}
	if _, err := os.Stat(w.sourceRef().Path); err != nil {
		t.Errorf("the new copy: %v", err)
	}
	if _, err := os.Stat(a); err != nil {
		t.Errorf("the chosen file itself went: %v", err)
	}
}

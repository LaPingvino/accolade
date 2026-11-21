package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// sourcePair is a's script with its source beside it, shared, and b
// joining from nothing (not yet connected).
func sourcePair(t *testing.T) (a, b *MainWindow, key collab.Key, script string) {
	t.Helper()
	noTrackers(t)
	a, script = sourceWindow(t, sourceScript, sourceStory)
	holdUI(t)
	b = a.app.newWindow()
	key, _ = collab.NewKey()
	a.startSharing(key, false)
	_ = os.Remove(keptSharedPath(key)) // b starts from nothing, as on another computer
	_ = os.Remove(ownMarkPath(key))
	b.startSharing(key, true)
	_ = os.WriteFile(ownMarkPath(key), nil, 0o644)
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	return a, b, key, script
}

// docText is a window's source document's text ("" with none open).
func docText(w *MainWindow) string {
	if w.srcDoc == nil {
		return ""
	}
	return w.srcDoc.Text()
}

// The source beside the sharer's script goes to the peer who joins, kept
// for the room there; editing it in either window reaches the other,
// outside the other's undo history; the sharer's Save writes the file
// beside the script, with the peer's edit in it.
func TestSourceTogether(t *testing.T) {
	a, b, key, script := sourcePair(t)
	want, _ := a.sourceText(a.sourceRef())
	if a.sharedSource() == "" {
		t.Fatal("a's source is not shared")
	}
	connect(t, a, b)
	textIs(t, b, "b", a.textEditor.Text())
	waitFor(t, "b has the source", func() bool { return docText(b) == want })
	if r := b.sourceRef(); r.Origin != sourceKept || r.Path != keptSharedSource(key) {
		t.Errorf("b keeps it as %+v", r)
	}
	if b.srcDoc.sharedAs() != a.sharedSource() || a.srcDoc.sharedAs() != a.sharedSource() {
		t.Errorf("shared as %q and %q", b.srcDoc.sharedAs(), a.srcDoc.sharedAs())
	}
	// a types at the end of its source, b at its start
	a.srcDoc.ed.Edit(func(e *buffer.Buffer) { e.SetCursor(e.Len(), false); e.Insert("\nTyped by a.\n") })
	waitFor(t, "b has a's typing", func() bool { return strings.Contains(docText(b), "Typed by a.") })
	b.srcDoc.ed.Edit(func(e *buffer.Buffer) { e.SetCursor(0, false); e.Insert("B: ") })
	waitFor(t, "a has b's typing", func() bool { return strings.HasPrefix(docText(a), "B: ") })
	if !strings.Contains(docText(a), "Typed by a.") || !a.srcDoc.dirty() {
		t.Errorf("a's source %q, dirty %v", docText(a), a.srcDoc.dirty())
	}
	// Undo undoes one's own only
	for a.srcDoc.ed.Buffer().CanUndo() {
		a.srcDoc.ed.Undo()
	}
	waitFor(t, "b without a's typing", func() bool { return !strings.Contains(docText(b), "Typed by a.") })
	if !strings.HasPrefix(docText(a), "B: ") {
		t.Errorf("a's Undo took b's typing: %q", docText(a)[:20])
	}
	// the pane's passages are the shared text
	if p := b.source; p != nil && !strings.HasPrefix(p.src, "B: ") {
		t.Errorf("b's pane reads %q", p.src[:20])
	}
	if err := a.SaveFile(); err != nil {
		t.Fatal(err)
	}
	beside, _ := os.ReadFile(filepath.Join(filepath.Dir(script), "story.md"))
	if !strings.HasPrefix(string(beside), "B: ") {
		t.Errorf("saved beside the script: %q", string(beside)[:20])
	}
}

// A third peer joins while a types in the source, key by key: it ends
// with what a and b have.
func TestSourceJoinMidEdit(t *testing.T) {
	a, b, key, _ := sourcePair(t)
	connect(t, a, b)
	waitFor(t, "b has the source", func() bool { return docText(b) != "" && docText(b) == docText(a) })
	c := a.app.newWindow()
	c.startSharing(key, true)
	t.Cleanup(c.stopSharing)
	for i := 0; i < 40; i++ {
		if i == 10 {
			connect(t, b, c) // through b
		}
		a.srcDoc.ed.Edit(func(e *buffer.Buffer) { e.SetCursor(i*7, false); e.Insert("k") })
		uiSleep(2e6)
	}
	waitFor(t, "c has the source as a and b", func() bool {
		return docText(c) != "" && docText(c) == docText(a) && docText(b) == docText(a)
	})
	if strings.Count(docText(c), "k") < 40 {
		t.Errorf("c has %d keys", strings.Count(docText(c), "k"))
	}
}

// A peer of protocol 4 (an older Accolade) is told apart: the status bar
// says it is older, and nothing it sends is taken.
func TestSourceOldPeer(t *testing.T) {
	a, b, key, _ := sourcePair(t)
	connect(t, a, b)
	waitFor(t, "b has the source", func() bool { return docText(b) != "" })
	hello, _ := key.Seal([]byte(`{"hello":true,"p":4,"app":"1.4.0"}`))
	ops, _ := key.Seal([]byte(`{"r":[{"id":{"c":1,"r":"old"},"t":"OLD"}]}`))
	session := b.shared.session
	go func() { // as its data channel does: what it does waits for the UI turn
		_ = session.Receive("old", hello)
		_ = session.Receive("old", ops)
	}()
	waitFor(t, "b says the other is older", func() bool { return strings.Contains(b.sharingStatus(), "older than yours") })
	uiSleep(20e6)
	if strings.Contains(b.textEditor.Text(), "OLD") || strings.Contains(a.textEditor.Text(), "OLD") {
		t.Error("an old peer's edit was taken")
	}
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The single-writer rule (docs/COLLAB.md): an automatic write that every
// peer would make is made by the writer of the line it follows from.

// A locked script's new scene, typed by a, is numbered once, by a: b,
// who has edited too and sees the heading come in key by key, never
// numbers it (nor a's half-typed heading).
func TestSceneNumbersTogether(t *testing.T) {
	a, b, _ := sharedPair(t, lockScript)
	a.lockSceneNumbers()
	connect(t, a, b)
	textIs(t, b, "b", a.textEditor.Text())
	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(len([]rune(b.textEditor.Text()[:strings.Index(b.textEditor.Text(), "Dry.")])))
	test.Type(b.textEditor, "Very ") // b can undo: it would number too
	textIs(t, a, "a", b.textEditor.Text())

	text := a.textEditor.Text()
	at := len([]rune(text[:strings.Index(text, "EXT. GARDEN")]))
	a.textEditor.Edit(func(buf *buffer.Buffer) { // a pasted scene, its heading still being written
		buf.Replace(at, at, "INT. SHED - DAY\n\nTools on the wall.\n\n")
		buf.SetCursor(at+3, false)
	})
	textIs(t, b, "b", a.textEditor.Text())
	if strings.Contains(b.textEditor.Text(), "#1A#") {
		t.Fatalf("b numbered a's heading:\n%s", b.textEditor.Text())
	}
	a.textEditor.SetCursorOffset(0) // a leaves the heading: a numbers it
	waitFor(t, "a's new scene numbered", func() bool { return strings.Contains(a.textEditor.Text(), "INT. SHED - DAY #1A#\n") })
	textIs(t, b, "b", a.textEditor.Text())
	got := a.textEditor.Text()
	if strings.Count(got, "#1A#") != 1 || strings.Contains(got, "#1B#") || strings.Contains(got, "##") {
		t.Errorf("numbered more than once:\n%s", got)
	}
	// the number is a's writing, not b's
	at = len([]rune(got[:strings.Index(got, "#1A#")]))
	var who, mine string
	a.shared.session.Do(func(d *collab.Doc) { who, mine = d.Writer(at), d.Replica() })
	if who != mine {
		t.Errorf("the number was written by %s, not a (%s)", who, mine)
	}
}

// A lesson (a heading written for a linked scene) is learned by the
// heading's writer only: b, whose cursor goes through a's new scene, does
// not learn from it; a does, once its cursor leaves the heading.
func TestLearnTogether(t *testing.T) {
	rs := []rune(suggestScript)
	start, end, block := withSourceBlock(suggestScript, learnStory)
	script := string(rs[:start]) + block + string(rs[end:])
	a, b, _ := sharedPair(t, script)
	connect(t, a, b)
	textIs(t, b, "b", script)
	for _, w := range []*MainWindow{a, b} {
		w.togglePane("source")
		waitFor(t, "the source read", func() bool {
			return w.source != nil && w.source.loading == nil && len(w.source.passages) > 0
		})
		w.learnFromWriting() // what the script has is known
	}
	at := len([]rune(script[:strings.Index(script, "/* Source")]))
	a.textEditor.Edit(func(buf *buffer.Buffer) {
		buf.Replace(at, at, "EXT. KOVAČ ROAD - MORNING\n\n[[from: \"The next morning the street was white\"]]\n\n")
		buf.SetCursor(at+3, false) // still writing the heading
	})
	if l := learnedBlock(a); l != "" {
		t.Fatalf("a learned while writing the heading: %q", l)
	}
	textIs(t, b, "b", a.textEditor.Text())
	moveTo(t, b, "The next morning the street was white")
	moveTo(t, b, "Title")
	b.learnFromWriting()
	if l := learnedBlock(b); l != "" {
		t.Fatalf("b learned from a's heading: %q", l)
	}
	moveTo(t, a, "Title")
	waitFor(t, "a's lesson at b", func() bool { return learnedBlock(b) == "places: street = EXT. KOVAČ ROAD" })
	textIs(t, a, "a", b.textEditor.Text())
}

// A number written twice in a script not shared (or by one writer) is
// left for Renumber: nothing changes by itself.
func TestSceneNumberTwiceAlone(t *testing.T) {
	w := newTestWindow(t, strings.Replace(lockScript, "EXT. GARDEN - DAY [[red]]", "EXT. GARDEN - DAY #2# [[red]]", 1))
	w.lockSceneNumbers()
	text := strings.Replace(w.textEditor.Text(), "INT. HOUSE - NIGHT #3#", "INT. HOUSE - NIGHT #2#", 1)
	w.textEditor.SetText(text)
	w.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(0, 0, ""); b.Insert("") })
	w.textEditor.SetCursorOffset(len([]rune(text)) - 1)
	w.numberNewScenes()
	if got := w.textEditor.Text(); got != text {
		t.Errorf("renumbered by itself:\n%s", got)
	}
}

// Two peers add a scene at one place before either hears of the other's:
// each numbers its own 1A. When they meet, the later of the two in the
// text is numbered again (1B), by its writer only, so the script ends
// with one 1A and one 1B on both sides.
func TestSceneNumberCollision(t *testing.T) {
	a, b, _ := sharedPair(t, lockScript)
	a.lockSceneNumbers()
	cut := connect(t, a, b)
	textIs(t, b, "b", a.textEditor.Text())
	cut()
	add := func(w *MainWindow, scene string) {
		text := w.textEditor.Text()
		at := len([]rune(text[:strings.Index(text, "EXT. GARDEN")]))
		w.textEditor.Edit(func(buf *buffer.Buffer) { buf.Replace(at, at, scene) })
		w.textEditor.SetCursorOffset(0)
		w.numberNewScenes()
		if !strings.Contains(w.textEditor.Text(), "#1A#") {
			t.Fatalf("not numbered alone:\n%s", w.textEditor.Text())
		}
	}
	add(a, "INT. SHED - DAY\n\nTools.\n\n")
	add(b, "INT. BARN - DAY\n\nHay.\n\n")
	connect(t, a, b)
	waitFor(t, "one 1A and one 1B on both", func() bool {
		ta, tb := a.textEditor.Text(), b.textEditor.Text()
		return ta == tb && strings.Count(ta, "#1A#") == 1 && strings.Count(ta, "#1B#") == 1
	})
	got := a.textEditor.Text()
	if i, j := strings.Index(got, "#1A#"), strings.Index(got, "#1B#"); i > j {
		t.Errorf("1B before 1A:\n%s", got)
	}
	uiSleep(2 * sceneNumberPause) // nothing more to number: it stays
	if a.textEditor.Text() != got || b.textEditor.Text() != got {
		t.Errorf("numbered on:\n%s\n%s", a.textEditor.Text(), b.textEditor.Text())
	}
}

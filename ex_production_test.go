package main

import (
	"slices"
	"strings"
	"testing"
)

// exScript has two sections and four scenes, numbered by hand (1A after
// 1), locked.
const exScript = "Title: X\n\n# One\n\nINT. A - DAY #1#\n\nAlpha.\n\nINT. B - DAY #1A#\n\nBeta.\n\nINT. C - DAY #2#\n\nGamma.\n\n# Two\n\nEXT. D - NIGHT #3#\n\nDelta.\n\n/* Production\nscene numbers: locked, 10A\n*/\n"

// exWin is a window with text, working as Production, its status kept.
func exWin(t *testing.T, text string) (*MainWindow, *string) {
	t.Helper()
	w := draftsWindow(t)
	keepAllSettings(t, w)
	w.setOwnRole(roleProduction)
	w.textEditor.SetText(text)
	said := new(string)
	return w, said
}

// numbersOf are the script's scene labels, in order.
func numbersNow(w *MainWindow) []string {
	text := w.textEditor.Text()
	var out []string
	for _, h := range sceneHeads(text[:bodyEndByte(text)], w.sceneStarts()) {
		out = append(out, h.num)
	}
	return out
}

// order is the scenes' first words after the place, in order (A B C D).
func sceneOrder(w *MainWindow) string {
	var out []string
	for _, l := range strings.Split(w.textEditor.Text(), "\n") {
		if strings.HasPrefix(l, "INT. ") || strings.HasPrefix(l, "EXT. ") {
			out = append(out, strings.Fields(l)[1])
		}
	}
	return strings.Join(out, " ")
}

func TestExRenumber(t *testing.T) {
	w, _ := exWin(t, exScript)
	w.runEx("renumber keep 1A")
	if got := numbersNow(w); !slices.Equal(got, []string{"1", "1A", "2", "3"}) {
		t.Errorf("keep 1A: %q", got)
	}
	w.runEx("renumber fold")
	if got := numbersNow(w); !slices.Equal(got, []string{"1", "2", "3", "4"}) {
		t.Errorf("fold: %q (%s)", got, w.statusLabel.Text)
	}
	w.undo()
	if got := numbersNow(w); !slices.Equal(got, []string{"1", "1A", "2", "3"}) {
		t.Errorf("one undo: %q", got)
	}
	w.runEx("renumber from 2 to 3")
	if got := numbersNow(w); got[1] != "1A" || got[0] != "1" {
		t.Errorf("from 2 to 3 renumbered before it: %q", got)
	}
	before := w.textEditor.Text()
	for _, bad := range []string{"renumber from 3 to 1", "renumber bogus", "renumber keep", "renumber omitted maybe", "renumber from 9"} {
		w.runEx(bad)
		if w.textEditor.Text() != before {
			t.Errorf(":%s changed the script", bad)
		}
		if s := w.statusLabel.Text; s == "" || strings.HasPrefix(s, "Scenes renumbered") {
			t.Errorf(":%s said %q", bad, s)
		}
	}
	if p, err := parseRenumber("only 2 letters omitted runs", exRefs{}); err != nil || p.List != listOnly || p.Numbers != "2" || !p.KeepLetters || p.Omitted != omitPrintRuns {
		t.Errorf("parse: %+v %v", p, err)
	}
}

func TestExLockOmitRestore(t *testing.T) {
	plain := "INT. A - DAY\n\nAlpha.\n\nINT. B - DAY\n\nBeta.\n\nINT. C - DAY\n\nGamma.\n"
	w, _ := exWin(t, plain)
	w.runEx("omit 2")
	if strings.Contains(w.textEditor.Text(), "OMITTED") {
		t.Error("omitted without numbers")
	}
	w.runEx("lock")
	if got := numbersNow(w); !slices.Equal(got, []string{"1", "2", "3"}) {
		t.Fatalf("lock: %q", got)
	}
	w.runEx("omit 2..3")
	text := w.textEditor.Text()
	if strings.Count(text, "OMITTED") != 2 || !strings.Contains(text, "Alpha.") {
		t.Fatalf("omit 2..3: %q", text)
	}
	w.undo()
	if strings.Contains(w.textEditor.Text(), "OMITTED") {
		t.Errorf("one undo: %q", w.textEditor.Text())
	}
	w.runEx("omit 2..3")
	w.runEx("restore 3")
	if strings.Count(w.textEditor.Text(), "OMITTED") != 1 || !strings.Contains(w.textEditor.Text(), "Gamma.") {
		t.Errorf("restore 3: %q", w.textEditor.Text())
	}
	w.runEx("restore 1..3")
	if strings.Contains(w.textEditor.Text(), "OMITTED") {
		t.Errorf("restore 1..3: %q", w.textEditor.Text())
	}
	w.runEx("unlock remove")
	if got := numbersNow(w); !slices.Equal(got, []string{"", "", ""}) || sceneLockOf(w.textEditor.Text()).locked {
		t.Errorf("unlock remove: %q", got)
	}
}

func TestExMoveSwap(t *testing.T) {
	w, _ := exWin(t, exScript)
	w.runEx("move 1..1A after 3")
	if got := sceneOrder(w); got != "C D A B" { // A and B leave section One for after D, in section Two
		t.Errorf("move 1..1A after 3: %s (%s)", got, w.statusLabel.Text)
	}
	if got := numbersNow(w); !slices.Equal(got, []string{"2", "3", "1", "1A"}) {
		t.Errorf("locked numbers go with their scenes: %q", got)
	}
	w.undo()
	if got := sceneOrder(w); got != "A B C D" {
		t.Errorf("one undo: %s", got)
	}
	w.runEx("swap 1 with 2")
	if got := sceneOrder(w); got != "C B A D" {
		t.Errorf("swap 1 with 2: %s", got)
	}
	w.undo()
	w.runEx("swap #One with #Two")
	if got := sceneOrder(w); got != "D A B C" {
		t.Errorf("swap sections: %s", got)
	}
	w.undo()
	w.runEx("move 3 before 1")
	if got := sceneOrder(w); got != "D A B C" { // into section One, before A
		t.Errorf("move 3 before 1: %s (%s)", got, w.statusLabel.Text)
	}
	w.textEditor.SetText(exScript)
	// a range of lines: the units they are in
	r, _ := w.scriptRefs()
	from := r.heads[1].line + 2 // Beta
	w.exVisual = [2]int{from, from}
	w.runEx("'<,'>move after 2")
	if got := sceneOrder(w); got != "A C B D" {
		t.Errorf("'<,'>move after 2: %s (%s)", got, w.statusLabel.Text)
	}
	// :m with an address still moves lines
	w.textEditor.SetText("one\ntwo\nthree")
	w.exVisual = [2]int{-1, -1}
	w.textEditor.SetCursorOffset(0)
	w.runEx("m$")
	if got := w.textEditor.Text(); got != "two\nthree\none" {
		t.Errorf("m$: %q", got)
	}
	before := exScript
	w.textEditor.SetText(before)
	for _, bad := range []string{"move 3..1 after 1A", "move 9 after 1", "move 1 after", "swap 1", "move 1 after 1"} {
		w.runEx(bad)
		if w.textEditor.Text() != before {
			t.Errorf(":%s changed the script", bad)
		}
	}
}

func TestExTagDraft(t *testing.T) {
	w, _ := exWin(t, exScript)
	selectWords(t, w, "Gamma", 0)
	w.runEx("tag prop")
	f, _, _ := w.breakdownNowFound()
	if len(f.Tags) != 1 {
		t.Errorf("tag prop: %d tags (%s)", len(f.Tags), w.statusLabel.Text)
	}
	w.runEx("draft Before the move")
	w.flushBundleWrites()
	waitFor(t, "the draft saved", func() bool {
		ds, _ := readDrafts(w.extrasHome())
		return slices.ContainsFunc(ds, func(d savedDraft) bool { return d.Name == "Before the move" })
	})
}

// The new commands complete: their words, the scenes' labels and the
// sections, the last word typed only.
func TestExProductionComplete(t *testing.T) {
	w, _ := exWin(t, exScript)
	ctx := w.exCtx()
	texts := func(line string) []string {
		var out []string
		for _, c := range exCommands().complete(ctx, line).choices {
			out = append(out, c.text)
		}
		return out
	}
	if got := texts("move 1.."); !slices.Contains(got, "1A") || !slices.Contains(got, "#One") {
		t.Errorf("move 1..: %q", got)
	}
	if got := texts("move 1..2 a"); len(got) == 0 || got[0] != "after " {
		t.Errorf("move 1..2 a: %q", got)
	}
	if c := exCommands().complete(ctx, "move 1..2 a"); c.start != len("move 1..2 ") {
		t.Errorf("start %d: only the last word is completed", c.start)
	}
	if got := texts("swap 1 w"); len(got) == 0 || got[0] != "with " {
		t.Errorf("swap 1 w: %q", got)
	}
	if got := texts("renumber k"); len(got) == 0 || got[0] != "keep " {
		t.Errorf("renumber k: %q", got)
	}
	if got := texts("renumber omitted "); !slices.Equal(got, []string{"drop ", "keep ", "runs "}) {
		t.Errorf("renumber omitted: %q", got)
	}
	if got := texts("unlock "); !slices.Equal(got, []string{"keep", "remove"}) {
		t.Errorf("unlock: %q", got)
	}
	if got := texts("tag pr"); len(got) == 0 || got[0] != "Props" {
		t.Errorf("tag pr: %q", got)
	}
	for _, n := range []string{"renumber", "lock", "unlock", "omit", "restore", "swap", "issue", "tag", "draft"} {
		if c, _ := exCommands().lookup(n); c == nil || c.name != n {
			t.Errorf(":%s names %v", n, c)
		}
	}
}

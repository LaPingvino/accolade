package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Keep Mine holds across sessions: a line with the dismissed findings,
// beside the document (or in its notes, as that setting says); a
// finding dismissed afterwards does not let it go; deleting it brings
// the difference back.
func TestScreenplaySourceKeepMineKept(t *testing.T) {
	for _, inDoc := range []bool{false, true} {
		w := spWindow(t)
		if inDoc {
			dismissInDocFor(t, w)
		}
		s := w.sourcePane()
		deck := lineWith(t, w, "EXT. FERRY DECK")
		ids, _ := s.sp.inScene(s.sp.current(), deck)
		s.sp.keepMine(s.sp.current(), ids[:1])
		s.follow()
		var line string
		for k := range w.allDismissed() {
			if strings.HasPrefix(k, keptMineHead) {
				line = k
			}
		}
		if line == "" {
			t.Fatalf("in the document %v: Keep Mine not kept: %v", inDoc, w.allDismissed())
		}
		side, err := os.ReadFile(dismissedSidecar(absPath(w.currentFile)))
		if inDoc == (err == nil) || !inDoc && !strings.Contains(string(side), line) {
			t.Errorf("in the document %v: sidecar %q (%v)", inDoc, side, err)
		}
		if inDoc && !strings.Contains(w.textEditor.Text(), line) {
			t.Errorf("not in the notes:\n%s", w.textEditor.Text())
		}
		// a finding dismissed after it keeps it, the source open as a
		// document (the findings' lines let go then)
		w.openMiddle("source")
		if w.srcEditor() == nil {
			t.Fatal("the source is not open as a document")
		}
		w.dismissFinding(openItem{Kind: openCheck, Text: "A finding", Raw: "x"})
		if !w.allDismissed()[line] {
			t.Errorf("in the document %v: dismissing a finding let Keep Mine go", inDoc)
		}
		// the next session: a new pane on the file as saved
		if inDoc {
			if err := w.SaveFile(); err != nil {
				t.Fatal(err)
			}
		}
		w.closeMiddle("source")
		w.dismissed = nil // read again, as a new session does
		s.sp = newScreenplayView(s)
		s.sp.follow() // the pane, shown again after the tab, follows by itself
		if !strings.Contains(s.count.Text, "1 difference kept as yours") {
			t.Errorf("in the document %v: after reopening %q", inDoc, s.count.Text)
		}
		if left, _ := s.sp.inScene(s.sp.current(), deck); len(left) != len(ids)-1 {
			t.Errorf("%d open after reopening, want %d", len(left), len(ids)-1)
		}
	}
}

// A source chosen as a PDF keeps its pages as printed beside it: the
// pane shows the scene's page in its layout, and ticking it folds its
// differences away (kept as Keep Mine is); unmarking brings them back.
func TestScreenplaySourcePageTick(t *testing.T) {
	w, _ := sourceWindow(t, spConverted, "")
	os.Remove(filepath.Join(filepath.Dir(w.currentFile), "story.md"))
	pdf := filepath.Join(t.TempDir(), "ferry.pdf")
	if err := exportAs(spOriginal, pdf, "pdf", exportJob{}); err != nil {
		t.Fatal(err)
	}
	holdUI(t)
	w.chooseSourceFile(pdf)
	waitForText(t, w)
	fyne.DoAndWait(func() {
		s := w.sourcePane()
		s.follow()
		if p := keptPagesFile(s.ref.Path); p == "" {
			t.Fatalf("kept as %s, not in a folder of its own", s.ref.Path)
		} else if data, err := os.ReadFile(p); err != nil || !strings.Contains(string(data), "Rain on the rail.") {
			t.Fatalf("pages %q (%v)", data, err)
		}
		deck := lineWith(t, w, "EXT. FERRY DECK")
		spCursorTo(t, w, "EXT. FERRY DECK")
		sc := s.sp.current()
		if sc == nil || sc.pages == nil {
			t.Fatal("no pages with the comparison")
		}
		pg := sc.pageOfScene(deck)
		if pg < 0 || !strings.Contains(sc.pages.pages[pg], "Rain on the rail.") {
			t.Fatalf("the deck scene's page %d", pg)
		}
		if !strings.Contains(sc.pages.pages[pg], "\n"+strings.Repeat(" ", 20)) {
			t.Errorf("the page not in its layout (cues indented):\n%s", sc.pages.pages[pg])
		}
		box := s.sp.rows.Objects[len(s.sp.rows.Objects)-1].(*widget.Accordion)
		if !strings.Contains(box.Items[0].Title, "as printed") {
			t.Errorf("the page box %q", box.Items[0].Title)
		}
		before, _ := s.sp.inScene(sc, deck)
		if len(before) == 0 {
			t.Fatal("the deck scene has no differences")
		}
		s.sp.tickPage(sc, pg, true)
		if left, _ := s.sp.inScene(s.sp.current(), deck); len(left) != 0 {
			t.Errorf("%d left on a page that looks right", len(left))
		}
		if !strings.Contains(s.count.Text, "on pages that look right") {
			t.Errorf("summary %q", s.count.Text)
		}
		side, _ := os.ReadFile(dismissedSidecar(absPath(w.currentFile)))
		if !strings.Contains(string(side), pageRightHead) {
			t.Errorf("the tick not kept beside: %q", side)
		}
		s.sp.tickPage(s.sp.current(), pg, false)
		if left, _ := s.sp.inScene(s.sp.current(), deck); len(left) != len(before) {
			t.Errorf("%d back after unmarking, want %d", len(left), len(before))
		}
	})
}

// A page is found by its words, a word broken over a line too; another
// print's page is not ticked by this one's.
func TestPageIndexFind(t *testing.T) {
	p := newPageIndex("INT. A\n\nThe mo-\nney is gone for good.\n\f\nINT. B\n\nRain.\n")
	if pg, _, sure := p.find("The money is gone for good.", 0); pg != 0 || !sure {
		t.Errorf("page %d, sure %v", pg, sure)
	}
	if pg, _, sure := p.find("Rain.", 0); pg != 1 || sure {
		t.Errorf("page %d, sure %v", pg, sure)
	}
	if pg, _, _ := p.find("Snow.", 0); pg != -1 {
		t.Errorf("page %d", pg)
	}
	// a few words are looked for only near the last sure place
	far := newPageIndex("Rain.\n\f\n" + strings.Repeat("word ", nearWords+10) + "\n\f\nINT. B\n")
	if pg, _, _ := far.find("INT. B", 0); pg != -1 {
		t.Errorf("a short text found %d pages away", pg)
	}
	if q := newPageIndex("INT. A\n\nOther.\n\f\nINT. B\n\nRain.\n"); q.rightLine(0) == p.rightLine(0) || q.rightLine(1) != p.rightLine(1) {
		t.Error("ticks not by the page's text")
	}
}

// A scene with no differences shows its heading's page after the
// differences before it: a place that comes back is not its first
// page's.
func TestPageOfSceneRecurring(t *testing.T) {
	script := "INT. KITCHEN - DAY\n\nAnna waits by the window for the bus.\n\nINT. HALL - DAY\n\nBram leaves with the dog and the keys.\n\nINT. KITCHEN - DAY\n\nAnna sits.\n"
	src := "INT. KITCHEN - DAY\n\nAnna waits by the window for the late bus.\n\nINT. HALL - DAY\n\nBram leaves with the dog and the old keys.\n\nINT. KITCHEN - DAY\n\nAnna sits.\n"
	pages := newPageIndex("INT. KITCHEN - DAY\n\nAnna waits by the window for the late bus.\n\f\nINT. HALL - DAY\n\nBram leaves with the dog and the old keys.\n\f\nINT. KITCHEN - DAY\n\nAnna sits.\n")
	w, _ := newDialogTestWindow(t)
	sc := compareWithSource(script, src, w.sourceOptions(src), pages)
	if len(sc.m.Changes) != 2 {
		t.Fatalf("%d changes", len(sc.m.Changes))
	}
	if got := sc.pageOfScene(1); got != 0 {
		t.Errorf("the first kitchen scene on page %d", got)
	}
	if got := sc.pageOfScene(9); got != 2 {
		t.Errorf("the kitchen again on page %d, want 2", got)
	}
}

// A bundle keeps its source's pages inside it: read at once (before its
// queue wrote them) and from the file after; a source chosen without
// pages takes them away.
func TestSourcePagesInBundle(t *testing.T) {
	holdUI(t)
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	realPaths(t)
	flush := func() {
		done := false
		w.afterBundleWrites(func() { done = true })
		waitFor(t, "the writes", func() bool { return done && !w.bundleBusy() })
	}
	ref := sourceRef{Origin: sourceBundle}
	w.keepSourcePages("INT. A\n\f\nINT. B\n")
	if got := w.readSourcePages(ref); !strings.Contains(got, "INT. B") {
		t.Errorf("before the write: %q", got)
	}
	flush()
	w.freshPages = nil
	if got := w.readSourcePages(ref); !strings.Contains(got, "INT. B") {
		t.Errorf("from the bundle: %q", got)
	}
	w.keepSourcePages("")
	flush()
	w.freshPages = nil
	if got := w.readSourcePages(ref); got != "" {
		t.Errorf("still there: %q", got)
	}
}

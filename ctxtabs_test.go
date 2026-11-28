// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/collab"
)

// The Characters tab: forward as the pane comes, its commands on the
// character tapped; Write chosen beside the pane stays chosen while
// typing; gone with the pane, the tab before back.
func TestCharactersTab(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, charsScript)
	r := ribbonOnFor(t, w)
	r.selectTab("View")
	w.showPane("characters", false)
	w.ribbonRefresh()
	if !tabButton(r, "Characters").Visible() || r.selected != "Characters" {
		t.Fatalf("with the pane: shown %v, %s chosen", tabButton(r, "Characters").Visible(), r.selected)
	}
	got := dumpRibbon(r)
	for _, want := range []string{"## Character", "L Rename icon=foreground_document-create.svg (disabled)", "S Merge", "## Notes", "## Places", "S Scenes", "## Close"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q:\n%s", want, got)
		}
	}
	p := w.charactersPane()
	p.next("BEN", 1) // tapped
	w.ribbonRefresh()
	if b := buttonOf(r, "chars.rename"); b == nil || b.disabled {
		t.Fatal("Rename off with BEN tapped")
	}
	if name := cmd("chars.rename").nameFor(w); !strings.Contains(name, "BEN") {
		t.Errorf("its tip %q", name)
	}
	if m := w.theirScenes(); len(m.Items) != 2 || !strings.Contains(m.Items[1].Label, "STREET") {
		t.Errorf("BEN's scenes: %v", m.Items)
	}
	w.do(cmd("chars.note"))()
	if p.open != "BEN" || w.fyneWindow.Canvas().Focused() != p.detail.note {
		t.Errorf("Note: open %q", p.open)
	}
	// Write chosen beside the pane, typing: Write stays
	r.selectTab("Write")
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(len(w.textEditor.Text()))
	test.Type(w.textEditor, "x")
	w.ribbonRefresh()
	if r.selected != "Write" {
		t.Errorf("typing beside the pane: %s chosen", r.selected)
	}
	r.selectTab("View")
	w.do(cmd("chars.close"))()
	w.ribbonRefresh()
	if tabButton(r, "Characters").Visible() || w.paneInSight("characters") {
		t.Error("still there after Close")
	}
}

// The Session tab: there while shared, not forward; who is here, in
// their colours; Follow moves the cursor with theirs until you type;
// Leave takes the tab away.
func TestSessionTab(t *testing.T) {
	a, b, _ := sharedPair(t, "INT. BARN - DAY\n\nRain.\n\nINT. LOFT - NIGHT\n\nHay.\n")
	r := ribbonOnFor(t, a)
	a.settings.SetString("your-name", "Ann")
	b.settings.SetString("your-name", "Bob")
	r.selectTab("Write")
	a.ribbonRefresh()
	if !tabButton(r, "Session").Visible() || r.selected != "Write" {
		t.Fatalf("shared: shown %v, %s chosen", tabButton(r, "Session").Visible(), r.selected)
	}
	a.sayHere(true)
	b.sayHere(true)
	connect(t, a, b)
	waitFor(t, "a sees Bob", func() bool { h := a.hereNow(); return len(h) == 1 && h[0].Name == "Bob" })
	textIs(t, b, "b", a.textEditor.Text()) // the script there before b's cursor goes into it
	r.selectTab("Session")
	got := dumpRibbon(r)
	if !strings.Contains(got, "## Here") || !strings.Contains(got, "S Bob icon=peer-") || !strings.Contains(got, "Copy Link") {
		t.Fatalf("the tab:\n%s", got)
	}
	// follow Bob: a's cursor goes where his goes
	a.follow(b.shared.replica)
	b.textEditor.SetCursorOffset(strings.Index(b.textEditor.Text(), "Hay."))
	b.onCursorChanged()
	waitFor(t, "a hears Bob on line 6", func() bool { h := a.hereNow(); return len(h) == 1 && h[0].Line == 6 })
	waitFor(t, "a follows Bob to line 6", func() bool { return a.scriptCursorLine() == 6 })
	a.fyneWindow.Canvas().Focus(a.textEditor)
	test.Type(a.textEditor, "Wet ")
	if a.shared.follow != "" {
		t.Error("still following after typing")
	}
	// the link
	a.do(cmd("session.copyLink"))()
	if clip := fyne.CurrentApp().Clipboard().Content(); !strings.Contains(clip, "#k=") {
		t.Errorf("copied %q", clip)
	}
	// b leaves: gone from a's list, b's tab gone
	rb := ribbonOnFor(t, b)
	b.ribbonRefresh()
	if !tabButton(rb, "Session").Visible() {
		t.Fatal("b has no Session tab")
	}
	b.do(cmd("session.leave"))()
	b.ribbonRefresh()
	if tabButton(rb, "Session").Visible() {
		t.Error("b's tab there after Leave")
	}
	waitFor(t, "Bob gone from a's list", func() bool { return len(a.hereNow()) == 0 })
	a.ribbonRefresh()
	if got := dumpRibbon(r); strings.Contains(got, "Bob") || !strings.Contains(got, "## Here") { // "Only you so far", a label
		t.Errorf("nobody here:\n%s", dumpRibbon(r))
	}
	_ = collab.HereFor
}

// The Compare tab: Compare is a middle tab now, its tab forward; Next
// goes round the changed scenes; Close, the tab before back.
func TestCompareTab(t *testing.T) {
	w := draftsShotWindow(t, fyne.NewSize(1100, 700))
	r := ribbonOnFor(t, w)
	r.selectTab("Edit")
	dd, _ := newDraftsDialog(w)
	cv := showCompare(w, dd.drafts, dd.drafts[0].ID, nowID)
	waitFor(t, "compared", func() bool { return cv.rows != nil })
	w.ribbonRefresh()
	if !tabButton(r, "Compare").Visible() || r.selected != "Compare" || w.mid.shown != "compare" {
		t.Fatalf("comparing: shown %v, %s chosen, %q in front", tabButton(r, "Compare").Visible(), r.selected, w.mid.shown)
	}
	if cv.bar.Visible() {
		t.Error("the Menus look's bar shows with the ribbon")
	}
	if len(cv.changes) != 3 {
		t.Fatalf("%d changes", len(cv.changes))
	}
	w.do(cmd("compare.next"))()
	w.do(cmd("compare.next"))()
	if cv.at != 1 || !strings.Contains(w.statusLabel.Text, "Change 2 of 3") {
		t.Errorf("at %d: %q", cv.at, w.statusLabel.Text)
	}
	w.do(cmd("compare.previous"))()
	w.do(cmd("compare.previous"))()
	if cv.at != 2 {
		t.Errorf("round the back: at %d", cv.at)
	}
	w.do(cmd("compare.close"))()
	w.ribbonRefresh()
	if tabButton(r, "Compare").Visible() || r.selected != "Edit" || w.compare != nil {
		t.Errorf("closed: shown %v, %s chosen", tabButton(r, "Compare").Visible(), r.selected)
	}
}

// TestContextTabsScreenshots renders the Characters, Session and Compare
// tabs on a desktop and a phone (SHOT_DIR as TestScreenshots).
func TestContextTabsScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	snap := func(w *MainWindow, name string) {
		t.Helper()
		w.ribbonRefresh()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	sizes := []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1200, 720)}, {"phone", fyne.NewSize(380, 800)}}
	for _, size := range sizes {
		// Characters
		w, _ := scriptWindow(t, charsScript)
		ribbonOnFor(t, w)
		w.fyneWindow.Resize(size.size)
		w.onWindowWidth(size.size.Width)
		w.showPane("characters", false)
		w.charactersPane().next("ANNA", 1)
		snap(w, "ctx-characters-"+size.name)
		// Compare
		cw := draftsShotWindow(t, size.size)
		ribbonOnFor(t, cw)
		cw.onWindowWidth(size.size.Width)
		dd, _ := newDraftsDialog(cw)
		cv := showCompare(cw, dd.drafts, dd.drafts[0].ID, nowID)
		waitFor(t, "compared", func() bool { return cv.rows != nil })
		cw.do(cmd("compare.next"))()
		snap(cw, "ctx-compare-"+size.name)
	}
	// Session (one pair: the sessions are the process's)
	a, b, _ := sharedPair(t, "INT. BARN - DAY\n\nRain.\n\nINT. LOFT - NIGHT\n\nHay.\n")
	r := ribbonOnFor(t, a)
	a.settings.SetString("your-name", "Ann")
	b.settings.SetString("your-name", "Bob")
	b.sayHere(true)
	connect(t, a, b)
	waitFor(t, "a sees Bob", func() bool { return len(a.hereNow()) == 1 })
	textIs(t, b, "b", a.textEditor.Text())
	for _, size := range sizes {
		a.fyneWindow.Resize(size.size)
		a.onWindowWidth(size.size.Width)
		r.selectTab("Session")
		a.follow(b.shared.replica)
		snap(a, "ctx-session-"+size.name)
		a.stopFollowing()
	}
}

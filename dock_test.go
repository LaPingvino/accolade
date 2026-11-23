package main

import (
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// dockTitles are the titles of the tabs shown in slot, in order.
func dockTitles(w *MainWindow, slot int) string {
	var titles []string
	for _, p := range w.dock.visible(slot) {
		titles = append(titles, p.cmd.label)
	}
	return strings.Join(titles, " ")
}

// keepDockSettings puts the dock's settings (shared by the tests) back
// as they were when the test ends.
func keepDockSettings(t *testing.T, w *MainWindow) {
	s := w.settings
	panes, width, split := s.GetStringSlice(dockPanesPref), s.GetFloat(dockWidthPref), s.GetFloat(dockSplitPref)
	t.Cleanup(func() {
		s.SetStringSlice(dockPanesPref, panes)
		s.SetFloat(dockWidthPref, width)
		s.SetFloat(dockSplitPref, split)
	})
}

// A pane's command brings it in (in front of its slot), to the front if
// it is behind another tab, and out when it is in sight; the dock goes
// when nothing is in it.
func TestDockToggles(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	run := func(id string) { cmd(id).run(w) }
	run("view.outline")
	if dockTitles(w, dockTop) != "Outline" || dockTitles(w, dockBottom) != "" || w.dock.split != nil {
		t.Fatalf("outline alone: top %q, bottom %q", dockTitles(w, dockTop), dockTitles(w, dockBottom))
	}
	if w.sideSplit == nil {
		t.Fatal("no dock beside the script")
	}
	run("view.notes")
	run("view.help")
	if got := dockTitles(w, dockBottom); got != "Notes Help" || !w.paneInSight("help") || w.paneInSight("notes") {
		t.Fatalf("bottom %q, help in sight %v", got, w.paneInSight("help"))
	}
	if w.dock.split == nil {
		t.Error("two slots and no split between them")
	}
	if !cmd("view.notes").checked(w) {
		t.Error("Notes is in the dock but its toggle is not checked")
	}
	run("view.notes") // behind Help: to the front, not out
	if !w.paneInSight("notes") || !w.paneShown("help") {
		t.Errorf("Notes not brought to the front (help still in: %v)", w.paneShown("help"))
	}
	if c := w.fyneWindow.Canvas().Focused(); c != w.notes.entry {
		t.Errorf("the keyboard on %T, not the notes", c)
	}
	run("view.notes") // in sight: out
	if w.paneShown("notes") || !w.paneInSight("help") {
		t.Errorf("Notes not out, or Help not in front after it")
	}
	run("view.outline")
	run("view.help")
	if w.sideSplit != nil || len(w.dock.visible(dockSheet)) != 0 {
		t.Error("the dock is still there with nothing in it")
	}
	// Side Panes: all out, then back as they were
	run("view.outline")
	run("view.notes")
	run("view.dock")
	if len(w.dock.visible(dockSheet)) != 0 || w.sideSplit != nil {
		t.Error("Side Panes left panes in sight")
	}
	run("view.dock")
	if dockTitles(w, dockTop) != "Outline" || dockTitles(w, dockBottom) != "Notes" {
		t.Errorf("Side Panes brought back top %q, bottom %q", dockTitles(w, dockTop), dockTitles(w, dockBottom))
	}
}

// A tab moves to the other slot (its menu, or View > Move Pane to …);
// the arrangement is kept and read back by a new window; an older
// settings file's side-pane-offset becomes the dock's width.
func TestDockMoveRemembered(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	keepDockSettings(t, w)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	w.togglePane("notes")
	w.togglePane("changes")
	cmd("view.paneTop").run(w) // Changes, picked last
	if got := dockTitles(w, dockTop); got != "Outline Changes" || !w.paneInSight("changes") {
		t.Fatalf("top %q after the move", got)
	}
	if dockTitles(w, dockBottom) != "Notes" {
		t.Errorf("bottom %q", dockTitles(w, dockBottom))
	}
	// from the tab's menu, with the keyboard on it
	tab := w.dock.tabs["outline"]
	m := w.dock.menu("outline")
	if m.Items[0].Label != "Move to Bottom" || m.Items[1].Label != "Open in the Middle" || m.Items[2].Label != "Hide Outline" {
		t.Fatalf("the tab's menu: %q, %q, %q", m.Items[0].Label, m.Items[1].Label, m.Items[2].Label)
	}
	w.fyneWindow.Canvas().Focus(tab)
	m.Items[0].Action()
	if dockTitles(w, dockTop) != "Changes" || dockTitles(w, dockBottom) != "Notes Outline" {
		t.Errorf("after Move to Bottom: top %q, bottom %q", dockTitles(w, dockTop), dockTitles(w, dockBottom))
	}
	if f, ok := w.fyneWindow.Canvas().Focused().(*dockTab); !ok || f.id != "outline" {
		t.Errorf("the keyboard did not go with the tab: %T", w.fyneWindow.Canvas().Focused())
	}
	if got := w.settings.GetStringSlice(dockPanesPref); !slices.Contains(got, "changes:top") || !slices.Contains(got, "outline:bottom") {
		t.Errorf("kept %q", got)
	}
	w.dock.split.SetOffset(0.3)
	w.sideSplit.SetOffset(0.4)
	w.dock.keepSizes()

	// another window reads it back
	w2 := NewMainWindow(w.app)
	t.Cleanup(w2.stopTimers)
	w2.fyneWindow.Resize(fyne.NewSize(1100, 650))
	w2.togglePane("outline")
	w2.togglePane("notes")
	w2.togglePane("changes")
	if dockTitles(w2, dockTop) != "Changes" || dockTitles(w2, dockBottom) != "Notes Outline" {
		t.Errorf("read back: top %q, bottom %q", dockTitles(w2, dockTop), dockTitles(w2, dockBottom))
	}
	if w2.sideSplit.Offset != 0.4 || w2.dock.split.Offset != 0.3 {
		t.Errorf("sizes read back: width %v, split %v", w2.sideSplit.Offset, w2.dock.split.Offset)
	}
}

// The width the side panes had before the dock is the dock's.
func TestDockMigratesWidth(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings = NewSettings() // an older settings file of its own, not the tests' shared one
	w.settings.SetFloat(oldWidthPref, 0.35)
	w.settings.SetStringSlice(dockPanesPref, []string{"notes:top", "gone:top", "outline:bottom"}) // a pane this version has not
	w.dock = newDock(w)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	w.togglePane("outline")
	if w.sideSplit == nil || w.sideSplit.Offset != 0.35 {
		t.Errorf("the old width not taken: %v", w.sideSplit)
	}
	w.togglePane("notes")
	w.togglePane("help") // not in the file: where it goes at first
	if dockTitles(w, dockTop) != "Notes" || dockTitles(w, dockBottom) != "Outline Help" {
		t.Errorf("top %q, bottom %q", dockTitles(w, dockTop), dockTitles(w, dockBottom))
	}
}

// Production is put away as a Writer and a screenplay's panes in a
// Markdown document, kept for when they apply again.
func TestDockFollowsRoleAndKind(t *testing.T) {
	w := newTestWindow(t, lockedScript)
	w.togglePane("notes")
	w.togglePane("production")
	if dockTitles(w, dockBottom) != "Notes Production" {
		t.Fatalf("bottom %q", dockTitles(w, dockBottom))
	}
	workAsFor(t, w, roleWriter)
	if dockTitles(w, dockBottom) != "Notes" || !w.paneInSight("notes") {
		t.Errorf("as a Writer: %q", dockTitles(w, dockBottom))
	}
	w.togglePane("production") // its command is away; nothing happens
	workAsFor(t, w, roleProduction)
	if dockTitles(w, dockBottom) != "Notes" {
		t.Errorf("Production again: %q (the pane was put away)", dockTitles(w, dockBottom))
	}
	w.togglePane("production") // asked for again (the palette switches the role, then runs it): in, not out
	if !w.paneInSight("production") {
		t.Error("Production's command after Writer did not bring the pane in")
	}
	w.setMarkdown(true)
	if dockTitles(w, dockBottom) != "Notes" {
		t.Errorf("Markdown: %q", dockTitles(w, dockBottom))
	}
}

// On a phone the dock is the sheet under the script: one row of tabs,
// the top's first; Tab reaches the row, the arrows go along it.
func TestDockPhoneSheet(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.togglePane("notes")
	w.togglePane("outline")
	if !w.narrow || w.paneSplit == nil || w.paneSplit.Horizontal {
		t.Fatalf("narrow %v, sheet %v", w.narrow, w.paneSplit)
	}
	if got := dockTitles(w, dockSheet); got != "Outline Notes" || !w.paneInSight("outline") || w.paneInSight("notes") {
		t.Errorf("sheet %q, outline in sight %v", got, w.paneInSight("outline"))
	}
	if mw := w.paneSplit.Trailing.MinSize().Width; mw > 380 {
		t.Errorf("the sheet is %v wide", mw)
	}
	tab := w.dock.tabs["outline"]
	w.fyneWindow.Canvas().Focus(tab)
	tab.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if !w.paneInSight("notes") || w.fyneWindow.Canvas().Focused() != w.dock.tabs["notes"] {
		t.Errorf("Right: notes in sight %v, focus %T", w.paneInSight("notes"), w.fyneWindow.Canvas().Focused())
	}
	w.dock.tabs["notes"].TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if w.fyneWindow.Canvas().Focused() != w.notes.entry {
		t.Errorf("Down: focus %T", w.fyneWindow.Canvas().Focused())
	}
	// F6: the editor, the pane in front
	if got := w.panes(); len(got) != 2 || got[1] != w.notes.entry {
		t.Errorf("F6 goes to %d panes", len(got))
	}
}

// F6 goes to each slot's pane in front, the top's first; Shift+F10 on a
// tab opens its menu.
func TestDockKeyboard(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	w.togglePane("notes")
	if got := w.panes(); len(got) != 3 || got[1] != w.outline.list || got[2] != w.notes.entry {
		t.Fatalf("F6's panes: %d", len(got))
	}
	c := w.fyneWindow.Canvas()
	c.Focus(w.dock.tabs["notes"])
	if !w.keyPreview(fyne.KeyF10, fyne.KeyModifierShift) || c.Overlays().Top() == nil {
		t.Error("Shift+F10 on a tab: no menu")
	}
	for c.Overlays().Top() != nil {
		c.Overlays().Remove(c.Overlays().Top())
	}
	// Tab reaches only the tab in front of a row
	if w.dock.tabs["outline"].SkipsFocusChain() {
		t.Error("the top row's only tab is skipped")
	}
	w.togglePane("help")
	if !w.dock.tabs["notes"].SkipsFocusChain() || w.dock.tabs["help"].SkipsFocusChain() {
		t.Error("Tab should reach Help (in front), not Notes")
	}
}

// Everything Tab reaches in the dock shows that it has the focus: the
// tabs in front of each row, the hide buttons, the panes; on a phone the
// sheet's row too.
func TestDockFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race (as TestFocusIsVisible)")
	}
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.togglePane("outline")
	w.togglePane("changes")
	w.togglePane("notes") // two tabs in the bottom row (not Help: its page's links run off the screen)
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("phone: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
}

// A press held half a second on a tab opens its menu (a finger in the
// browser); the tap after it, the menu dismissed, picks the tab again.
func TestDockLongPress(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.togglePane("notes")
	w.togglePane("changes")
	tab := w.dock.tabs["notes"]
	c := w.fyneWindow.Canvas()
	ev := &desktop.MouseEvent{Button: desktop.MouseButtonPrimary}
	var fire func()
	oldAfter := longPressAfter
	longPressAfter = func(d time.Duration, f func()) *time.Timer {
		fire = f
		return time.NewTimer(time.Hour) // stopped by MouseUp, never fires
	}
	t.Cleanup(func() { longPressAfter = oldAfter })
	tab.MouseDown(ev)
	fire() // the half second is up, on this goroutine
	if c.Overlays().Top() == nil {
		t.Fatal("no menu after a long press")
	}
	for c.Overlays().Top() != nil { // the release went to the menu: no Tapped
		c.Overlays().Remove(c.Overlays().Top())
	}
	tab.MouseDown(ev)
	tab.MouseUp(ev)
	tab.Tapped(&fyne.PointEvent{})
	if !w.paneInSight("notes") {
		t.Error("the tap after a long press was swallowed")
	}
	if c.Overlays().Top() != nil {
		t.Error("a short press opened the menu")
	}
}

// Hiding the pane in front brings the next one up to date.
func TestDockHideRefreshesFront(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	w.togglePane("notes")
	keepDockSettings(t, w)
	w.dock.move("notes", dockTop)
	w.outline.all = nil // stale
	w.togglePane("notes")
	if !w.paneInSight("outline") || len(w.outline.all) == 0 {
		t.Errorf("the outline, in front again, not brought up to date (%d items)", len(w.outline.all))
	}
}

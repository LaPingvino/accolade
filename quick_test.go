package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// quickIDs are the Quick tab's buttons' commands, in order.
func quickIDs(r *ribbonBar) []string {
	var out []string
	for _, b := range r.buttons {
		if b.quick && b.c != nil {
			out = append(out, b.c.id)
		}
	}
	return out
}

// restartQuick is the app started again: the counts read anew.
func restartQuick(w *MainWindow) {
	w.quick().save()
	w.app.quick = nil
	if w.ribbon != nil {
		w.ribbon.builtFor = ""
	}
	w.ribbonRefresh()
}

// toPhone lays w out at a phone's width.
func toPhone(t *testing.T, w *MainWindow) {
	t.Helper()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.fyneWindow.Canvas().Capture()
	if !w.narrow {
		t.Fatal("not narrow")
	}
}

// The score: the built-in priority per kind and role, the use as the
// app started (log of the count), the counts halving every two weeks.
func TestQuickScoring(t *testing.T) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	old := quickNow
	quickNow = func() time.Time { return now }
	t.Cleanup(func() { quickNow = old })
	s := loadQuickStore(filepath.Join(t.TempDir(), quickFileName))
	el := elementCommandID
	want := []string{el(elementScene), "edit.undo", el(elementAction), el(elementCharacter), el(elementDialogue),
		"file.save", "edit.find", "view.preview", el(elementParenthetical), "edit.redo"}
	if got := s.picks(cmdEnv{writer: true}); !slices.Equal(got, want) {
		t.Errorf("a screenplay's picks:\n got %q\nwant %q", got, want)
	}
	src := s.picks(cmdEnv{source: true})
	if !slices.Contains(src, "source.next") || !slices.Contains(src, "source.adapt") || len(src) != quickSize {
		t.Errorf("adapting: %q", src)
	}
	if prod := s.picks(cmdEnv{}); !slices.Contains(prod, "production.tag") {
		t.Errorf("production's picks without Tag: %q", prod)
	}
	md := s.picks(cmdEnv{markdown: true})
	if md[0] != "md.headings" || slices.Contains(md, el(elementScene)) || !slices.Contains(md, "format.bold") {
		t.Errorf("Markdown's picks %q", md)
	}
	for _, env := range []cmdEnv{{}, {writer: true}} {
		for _, id := range s.picks(env) {
			if strings.HasPrefix(id, "preview.") || id == "app.palette" || id == "view.menu" || id == "view.ribbonLock" {
				t.Errorf("%s among the picks", id)
			}
		}
	}
	// use: twenty uses of Move Line Up bring it in, Redo (the last) out
	for range 20 {
		s.used("format.lineUp")
	}
	s.save()
	s2 := loadQuickStore(s.path)
	if got := s2.picks(cmdEnv{writer: true}); !slices.Contains(got, "format.lineUp") || slices.Contains(got, "edit.redo") {
		t.Errorf("after twenty uses: %q", got)
	}
	if got := s2.frozen["format.lineUp"]; got != 20 {
		t.Errorf("count %v", got)
	}
	now = now.Add(quickHalfLife) // two weeks on: half
	s3 := loadQuickStore(s.path)
	if got := s3.frozen["format.lineUp"]; got < 9.99 || got > 10.01 {
		t.Errorf("after two weeks: %v, not 10", got)
	}
	if u := s3.data.Uses["format.lineUp"]; u.at(now.Add(quickHalfLife)) > 5.01 {
		t.Errorf("after four weeks: %v", u.at(now.Add(quickHalfLife)))
	}
}

// Stable in a session: uses change nothing until the app starts again,
// a source's commands come in at the end, nothing before them moves.
func TestQuickStableInSession(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	toPhone(t, w)
	if !w.cmdEnv().smart || r.selected != quickTab || r.shownTabs()[0].title != quickTab {
		t.Fatalf("a phone's ribbon: smart=%v, %s chosen", w.cmdEnv().smart, r.selected)
	}
	first := quickIDs(r)
	if len(first) < 8 || len(first) > quickMax {
		t.Fatalf("Quick holds %d: %q", len(first), first)
	}
	for range 40 {
		w.noteUse(cmd("format.lineDown"))
	}
	r.builtFor = ""
	r.refresh()
	if got := quickIDs(r); !slices.Equal(got, first) {
		t.Errorf("moved within the session:\n %q\n %q", first, got)
	}
	r.selectTab("Edit")
	r.selectTab(quickTab)
	if got := quickIDs(r); !slices.Equal(got, first) {
		t.Errorf("moved after another tab: %q", got)
	}
	// a source: Next Passage and Adapt at the end
	p := filepath.Join(t.TempDir(), "story.md")
	if err := os.WriteFile(p, []byte(suggestStory), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.useSource(p); err != nil {
		t.Fatal(err)
	}
	w.ribbonRefresh()
	got := quickIDs(r)
	if !slices.Equal(got[:len(first)], first) || !slices.Contains(got[len(first):], "source.next") {
		t.Errorf("a source's commands not at the end:\n %q\n %q", first, got)
	}
	if b := buttonOf(r, "source.next"); b == nil || b.label != "Next Passage" {
		t.Errorf("Next Source Passage on Quick: %+v", b)
	}
	// started again: the uses count
	restartQuick(w)
	if got := quickIDs(r); !slices.Contains(got, "format.lineDown") {
		t.Errorf("after a restart, Move Line Down (40 uses) is not on Quick: %q", got)
	}
}

// Pinning: a long press or right-click on any button, Pin to Quick puts
// it at the end, kept over a restart (pins first then); Unpin takes it
// off; Remove takes a pick off for good.
func TestQuickPin(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	toPhone(t, w)
	before := quickIDs(r)
	r.selectTab("Edit")
	b := buttonOf(r, "edit.selectAll")
	ev := &desktop.MouseEvent{Button: desktop.MouseButtonPrimary}
	var fire func()
	oldAfter := longPressAfter
	longPressAfter = func(d time.Duration, f func()) *time.Timer {
		fire = f
		return time.NewTimer(time.Hour)
	}
	t.Cleanup(func() { longPressAfter = oldAfter })
	c := w.fyneWindow.Canvas()
	b.MouseDown(ev)
	fire()
	if c.Overlays().Top() == nil {
		t.Fatal("no menu after a long press")
	}
	if m := r.quickMenu(b); m.Items[0].Label != "Pin to Quick" {
		t.Errorf("the menu: %q", m.Items[0].Label)
	}
	for c.Overlays().Top() != nil {
		c.Overlays().Remove(c.Overlays().Top())
	}
	b.Tapped(&fyne.PointEvent{}) // the release after the long press: no tap
	if b.pressed {
		t.Error("the long press's tap not swallowed")
	}
	r.quickMenu(b).Items[0].Action()
	r.selectTab(quickTab)
	got := quickIDs(r)
	if !slices.Equal(got[:len(before)], before) || got[len(got)-1] != "edit.selectAll" {
		t.Errorf("pinned not at the end:\n %q\n %q", before, got)
	}
	restartQuick(w)
	if got := quickIDs(r); got[0] != "edit.selectAll" {
		t.Errorf("after a restart the pin is not first: %q", got)
	}
	sa := buttonOf(r, "edit.selectAll")
	if m := r.quickMenu(sa); m.Items[0].Label != "Unpin from Quick" {
		t.Errorf("a pin's menu: %q", m.Items[0].Label)
	}
	r.quickMenu(sa).Items[0].Action()
	if slices.Contains(quickIDs(r), "edit.selectAll") {
		t.Error("still there after Unpin")
	}
	// a pick taken off stays off
	find := buttonOf(r, "edit.find")
	if m := r.quickMenu(find); m.Items[0].Label != "Remove from Quick" {
		t.Fatalf("a pick's menu: %q", m.Items[0].Label)
	}
	r.quickMenu(find).Items[0].Action()
	restartQuick(w)
	if slices.Contains(quickIDs(r), "edit.find") {
		t.Error("Find back after a restart")
	}
	// the tab row's buttons and a format's own have no menu
	if r.quickMenu(r.order) != nil || r.quickMenu(r.palette) != nil {
		t.Error("the tab row's buttons have the Quick menu")
	}
}

// The lock: Fixed on the desktop, Smart on a phone, one tap switching in
// place (the tab in front kept if both have it), per window; flipping
// twice shows the same ribbon; Smart orders the groups by use.
func TestQuickToggle(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.fyneWindow.Canvas().Capture()
	if w.cmdEnv().smart || tabButton(r, quickTab).Visible() {
		t.Fatal("the desktop starts Smart")
	}
	if r.order.icon != lockIcon || !r.order.checked || r.order.Disabled() {
		t.Error("the lock is not shut in Fixed")
	}
	for range 30 { // Edit > Lines, used a lot before the app started
		w.noteUse(cmd("format.lineUp"))
	}
	restartQuick(w)
	r.selectTab("Edit")
	fixed := dumpRibbon(r)
	if !strings.Contains(fixed, "## Undo") || strings.Index(fixed, "## Undo") > strings.Index(fixed, "## Lines") {
		t.Errorf("Fixed is not the standard order:\n%s", fixed)
	}
	r.order.Tapped(&fyne.PointEvent{})
	if !w.cmdEnv().smart || r.selected != "Edit" || r.order.icon != unlockIcon {
		t.Fatalf("after the tap: smart=%v, %s chosen", w.cmdEnv().smart, r.selected)
	}
	smart := dumpRibbon(r)
	if strings.Index(smart, "## Lines") > strings.Index(smart, "## Undo") {
		t.Errorf("Smart does not put the used group first:\n%s", smart)
	}
	if r.shownTabs()[0].title != quickTab {
		t.Error("no Quick tab first in Smart")
	}
	// another window keeps its own (a new one starts as the lock was last
	// set on the desktop: Smart now)
	w2, _ := scriptWindow(t, revScript)
	ribbonOnFor(t, w2)
	w.app.windows = append(w.app.windows, w2)
	before := w2.ribbonSmart()
	r.order.Tapped(&fyne.PointEvent{})
	if w2.ribbonSmart() != before || w2.ribbonOrder == "" {
		t.Error("the other window switched too")
	}
	w.app.windows = slices.DeleteFunc(w.app.windows, func(x *MainWindow) bool { return x == w2 })
	if got := dumpRibbon(r); got != fixed {
		t.Errorf("Fixed again differs:\n%s", lineDiff(fixed, got))
	}
	r.order.Tapped(&fyne.PointEvent{})
	if got := dumpRibbon(r); got != smart {
		t.Errorf("Smart again differs:\n%s", lineDiff(smart, got))
	}
	// on Quick, Fixed has no Quick tab: Write; back to Smart keeps Write
	r.selectTab(quickTab)
	w.toggleRibbonOrder()
	if r.selected != "Write" {
		t.Errorf("Fixed from Quick: %s", r.selected)
	}
	w.toggleRibbonOrder()
	if r.selected != "Write" {
		t.Errorf("Smart from Write: %s", r.selected)
	}
	// the lock's key tip, unique with the tabs'; it switches
	w.keyPreview(fyne.KeyF10, 0)
	tips := []string{r.order.tip, r.palette.tip, r.menus.tip}
	for _, tb := range r.shownTabs() {
		tips = append(tips, tb.tip)
	}
	checkTips(t, "Smart's tabs", tips)
	for _, ch := range r.order.tip {
		w.fyneWindow.Canvas().Focused().TypedRune(ch)
	}
	if w.cmdEnv().smart {
		t.Error("the lock's key tip did not switch")
	}
	// in the ☰ menu (View), checked when Fixed
	var found bool
	for _, m := range w.menusMenu().Items {
		for _, it := range m.ChildMenu.Items {
			if it.Label == "Lock Ribbon Order" {
				found = it.Checked
			}
		}
	}
	if !found {
		t.Error("no checked Lock Ribbon Order in ☰")
	}
	// a phone: Smart and Quick at first
	toPhone(t, w2)
	if !w2.cmdEnv().smart || w2.ribbon.selected != quickTab {
		t.Errorf("a phone: smart=%v, %s", w2.cmdEnv().smart, w2.ribbon.selected)
	}
	// remembered per phone or desktop: Fixed on the phone, a new window
	// (or a restart: the settings) starts Fixed there, the desktop as it was
	w2.toggleRibbonOrder()
	w2.ribbonOrder = "" // as a new window has it
	if w2.ribbonSmart() {
		t.Error("the phone's Fixed not remembered")
	}
	if got := w2.settings.GetString(ribbonOrderPref(true)); got != "fixed" {
		t.Errorf("the phone's setting %q", got)
	}
	last := w.ribbonSmart() // w, on the desktop, toggled last
	w.ribbonOrder = ""      // as a new desktop window has it
	if w.ribbonSmart() != last || w.settings.GetString(ribbonOrderPref(false)) == "" {
		t.Error("a new desktop window does not take the desktop's last choice")
	}
}

// A window turning narrow with the page view open keeps the Preview tab
// in front, and a tab the user chose.
func TestQuickNarrowKeepsPreview(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.fyneWindow.Canvas().Capture()
	previewAndWait(t, w)
	toPhone(t, w)
	if r.selected != "Preview" {
		t.Errorf("narrow with the preview: %s in front", r.selected)
	}
	w.closePreview()
	if r.selected != "Write" && r.selected != quickTab {
		t.Errorf("after the preview: %s", r.selected)
	}
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.fyneWindow.Canvas().Capture()
	r.selectTab("Edit")
	toPhone(t, w)
	if r.selected != "Edit" {
		t.Errorf("narrow took the chosen tab: %s", r.selected)
	}
}

// The Preview tab and Quick: the preview comes forward over Quick and
// gives it back when it closes.
func TestQuickAndPreview(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	toPhone(t, w)
	if r.selected != quickTab {
		t.Fatalf("%s chosen", r.selected)
	}
	previewAndWait(t, w)
	if r.selected != "Preview" {
		t.Errorf("the preview did not come forward: %s", r.selected)
	}
	w.closePreview()
	if r.selected != quickTab {
		t.Errorf("after the preview: %s", r.selected)
	}
}

// The desktop: the Quick tab (Smart) and the Quick row (Fixed, off at
// first) fit 1000 px; the row's buttons have level-one key tips.
func TestQuickFitsDesktop(t *testing.T) {
	holdUI(t)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	if r.quickRowBox.Visible() {
		t.Error("the Quick row on at first")
	}
	for _, md := range []bool{false, true} {
		w.setDocumentType(md)
		w.ribbonOrder = "smart"
		r.selectTab(quickTab)
		w.fyneWindow.Canvas().Capture()
		if got, room := r.groupBox.MinSize().Width, r.groups.Size().Width; got > room {
			t.Errorf("md=%v Quick: %v wide, more than %v", md, got, room)
		}
		w.ribbonOrder = "fixed"
		cmd("set.quickRow").run(w)
		w.fyneWindow.Canvas().Capture()
		if !r.quickRowBox.Visible() || len(r.rowButtons) < 8 {
			t.Fatalf("md=%v: no Quick row (%d)", md, len(r.rowButtons))
		}
		room := r.quickRowBox.Size().Width
		got := r.quickRow.MinSize().Width
		if testing.Verbose() {
			t.Logf("md=%v the Quick row: %v of %v", md, got, room)
		}
		if got > room {
			t.Errorf("md=%v the Quick row %v wide, more than %v", md, got, room)
		}
		w.keyPreview(fyne.KeyF10, 0)
		var tips []string
		for _, b := range append([]*ribbonButton{r.order, r.palette, r.menus}, r.rowButtons...) {
			tips = append(tips, b.tip)
		}
		for _, tb := range r.shownTabs() {
			tips = append(tips, tb.tip)
		}
		checkTips(t, "the Quick row's", tips)
		r.leaveKeyTips(false)
		cmd("set.quickRow").run(w)
	}
	w.setDocumentType(false)
}

// TestQuickScreenshots renders Smart and Fixed on a phone and the
// desktop, the Quick row, a button's Pin menu (SHOT_DIR=… go test -run
// TestQuickScreenshots).
func TestQuickScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	for range 30 { // Edit > Find and Lines used a lot: first in Smart
		w.noteUse(cmd("edit.findNext"))
		w.noteUse(cmd("format.lineUp"))
	}
	restartQuick(w)
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	}
	toPhone(t, w)
	snap("quick-phone-smart")
	r.selectTab("Edit")
	snap("quick-phone-smart-edit")
	w.toggleRibbonOrder()
	snap("quick-phone-fixed")
	w.toggleRibbonOrder()
	r.selectTab(quickTab)
	r.showQuickMenu(r.buttons[3], fyne.Position{})
	snap("quick-phone-pin-menu")
	for c := w.fyneWindow.Canvas(); c.Overlays().Top() != nil; {
		c.Overlays().Remove(c.Overlays().Top())
	}
	w.fyneWindow.Resize(fyne.NewSize(1000, 600))
	w.fyneWindow.Canvas().Capture()
	w.ribbonOrder = ""
	r.selectTab("Write")
	snap("quick-desktop-fixed")
	w.toggleRibbonOrder()
	r.selectTab(quickTab)
	snap("quick-desktop-smart")
	w.toggleRibbonOrder()
	cmd("set.quickRow").run(w)
	snap("quick-desktop-fixed-row")
	w.keyPreview(fyne.KeyF10, 0)
	snap("quick-desktop-row-keytips")
	r.leaveKeyTips(false)
	cmd("set.quickRow").run(w)
}

// A phone's Quick tab: its buttons in two balanced rows, every one inside
// the window and not cut; the script below gets the height left, a row
// less on Quick, and going to another tab and back lays it out as before.
func TestQuickPhoneTwoRows(t *testing.T) {
	holdUI(t)
	w := suggestWindow(t, suggestScript)
	r := ribbonOnFor(t, w)
	toPhone(t, w)
	if r.selected != quickTab {
		t.Fatalf("%s in front", r.selected)
	}
	d := fyne.CurrentApp().Driver()
	type geo struct{ top, height, ribbon float32 }
	at := func() geo {
		w.fyneWindow.Canvas().Capture() // laid out
		var e fyne.CanvasObject = w.textEditor
		if w.narrow && w.paneSplit != nil { // the script over the panes: their split shares the height
			e = w.paneSplit
		}
		return geo{d.AbsolutePositionForObject(e).Y, e.Size().Height, d.AbsolutePositionForObject(r.box).Y + r.box.Size().Height}
	}
	quick := at()
	bs := r.buttons
	if len(bs) < 8 {
		t.Fatalf("Quick holds %d", len(bs))
	}
	rows := map[float32][]string{}
	for _, b := range bs {
		p, s, m := d.AbsolutePositionForObject(b), b.Size(), b.MinSize()
		if p.X < 0 || p.X+s.Width > 380 || p.Y+s.Height > quick.ribbon {
			t.Errorf("%s at %v, %v: outside", b.label, p, s)
		}
		if s.Width < m.Width || s.Height < m.Height {
			t.Errorf("%s is %v, less than its %v", b.label, s, m)
		}
		rows[p.Y] = append(rows[p.Y], b.label)
	}
	if len(rows) != 2 {
		t.Errorf("Quick in %d rows: %v", len(rows), rows)
	}
	var counts []int
	for _, row := range rows {
		counts = append(counts, len(row))
	}
	if counts[0]-counts[1] > 1 || counts[1]-counts[0] > 1 {
		t.Errorf("rows not balanced: %v", rows)
	}
	if r.groups.Offset.X != 0 || r.groupBox.MinSize().Width > r.groups.Size().Width {
		t.Errorf("Quick scrolls: %v wide in %v", r.groupBox.MinSize().Width, r.groups.Size().Width)
	}
	if quick.top < quick.ribbon {
		t.Errorf("the script at %v, under the ribbon's foot at %v", quick.top, quick.ribbon)
	}
	r.selectTab("Write")
	write := at()
	if write.top >= quick.top || write.height <= quick.height {
		t.Errorf("Write's script not taller: %+v vs Quick's %+v", write, quick)
	}
	if write.top < write.ribbon || write.top-write.ribbon != quick.top-quick.ribbon {
		t.Errorf("a gap or overlap: Write %+v, Quick %+v", write, quick)
	}
	if d := (quick.top - write.top) - (write.height - quick.height); d > 0.5 || d < -0.5 {
		t.Errorf("the script and panes did not get the height left: %+v vs %+v", write, quick)
	}
	r.selectTab(quickTab)
	if again := at(); again != quick {
		t.Errorf("Quick again: %+v, was %+v", again, quick)
	}
	r.selectTab("Write")
	if again := at(); again != write {
		t.Errorf("Write again: %+v, was %+v", again, write)
	}
	// the desktop keeps Quick in three-row columns, one ribbon height
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.ribbonOrder = "smart"
	r.selectTab("Write")
	dw := at()
	r.selectTab(quickTab)
	if dq := at(); dq != dw {
		t.Errorf("the desktop's Quick changes the height: %+v vs %+v", dq, dw)
	}
}

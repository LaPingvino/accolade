package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// a script with one number typed by hand before the lock (#7#), a note
// after a heading and a stray number
const unlockScript = "Title: Rain\n\nINT. HOUSE - DAY\n\nRain.\n\nEXT. GARDEN - DAY [[red]]\n\nMore rain.\n\nINT. HOUSE - NIGHT #7#\n\nDry.\n\nINT. SHED - DAY\n\nTools.\n\n/* Notes\nkeep\n*/\n"

// buttonIn is the button labelled so in the dialog's content (nil none).
func buttonIn(o fyne.CanvasObject, label string) *widget.Button {
	switch x := o.(type) {
	case *widget.Button:
		if x.Text == label {
			return x
		}
	case *fyne.Container:
		for _, c := range x.Objects {
			if b := buttonIn(c, label); b != nil {
				return b
			}
		}
	case fyne.Widget:
		for _, c := range test.WidgetRenderer(x).Objects() {
			if b := buttonIn(c, label); b != nil {
				return b
			}
		}
	}
	return nil
}

// overlayButton is the button labelled so on the window's top overlay.
func overlayButton(t *testing.T, w *MainWindow, label string) *widget.Button {
	t.Helper()
	for _, o := range w.fyneWindow.Canvas().Overlays().List() {
		if b := buttonIn(o, label); b != nil {
			return b
		}
	}
	t.Fatalf("no button %q", label)
	return nil
}

// Remove takes out what the lock wrote, the number typed before it kept:
// the script as it was, one undo step; Keep leaves every number.
func TestUnlockRemoveAndKeep(t *testing.T) {
	w := newTestWindow(t, unlockScript)
	w.lockSceneNumbers()
	locked := w.textEditor.Text()
	if !strings.Contains(locked, "numbered before the lock: 7\n") {
		t.Fatalf("the hand number not noted:\n%s", locked)
	}
	if got := strings.Join(headsOf(w), "|"); got != "INT. HOUSE - DAY #A7#|EXT. GARDEN - DAY #B7# [[red]]|INT. HOUSE - NIGHT #7#|INT. SHED - DAY #8#" {
		t.Fatalf("locked %q", got)
	}
	w.unlockNow(true)
	if got := w.textEditor.Text(); got != unlockScript {
		t.Errorf("Remove is not the script before the lock:\n%s", got)
	}
	w.undo()
	if got := w.textEditor.Text(); got != locked {
		t.Errorf("Undo of Remove:\n%s", got)
	}
	w.redo()
	if got := w.textEditor.Text(); got != unlockScript {
		t.Errorf("Redo of Remove:\n%s", got)
	}
	w.undo()
	w.unlockNow(false)
	text := w.textEditor.Text()
	if strings.Contains(text, "Production") || !strings.Contains(text, "INT. SHED - DAY #8#") {
		t.Errorf("Keep:\n%s", text)
	}
}

// A lock made before the hand numbers were noted: Remove takes every
// number, and the dialog says so.
func TestUnlockOldLock(t *testing.T) {
	old := strings.Replace(unlockScript, "/* Notes", "/* Production\nscene numbers: locked, 10A\n*/\n\n/* Notes", 1)
	old = strings.Replace(old, "INT. HOUSE - DAY\n", "INT. HOUSE - DAY #1#\n", 1)
	w := newTestWindow(t, old)
	w.unlockDialog().Show()
	if !dialogShows(w, "made before Accolade noted") || !dialogShows(w, "every number goes (2 numbers)") {
		t.Error("the dialog does not say every number goes")
	}
	clearOverlays(w)
	w.unlockNow(true)
	text := w.textEditor.Text()
	if strings.Contains(text, "#") || strings.Contains(text, "Production") {
		t.Errorf("numbers left:\n%s", text)
	}
}

// The dialog's default: Remove while nothing has been issued, Keep once
// it has; the menu item opens it.
func TestUnlockDialogDefault(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, unlockScript)
	w.lockSceneNumbers()
	w.unlockSceneNumbers()
	if !dialogShows(w, "Nothing has been issued") || !dialogShows(w, "the ones you typed before locking stay (7)") {
		t.Error("the dialog before an issue")
	}
	if overlayButton(t, w, "Remove the Numbers").Importance != widget.HighImportance ||
		overlayButton(t, w, "Keep the Numbers").Importance == widget.HighImportance {
		t.Error("Remove is not the default before an issue")
	}
	test.Tap(overlayButton(t, w, "Cancel"))
	if !sceneLockOf(w.textEditor.Text()).locked {
		t.Error("Cancel unlocked")
	}
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	clearOverlays(w)
	w.unlockSceneNumbers()
	if !dialogShows(w, "The script was issued: White (1 Nov 2026)") {
		t.Error("the dialog after an issue")
	}
	if overlayButton(t, w, "Keep the Numbers").Importance != widget.HighImportance ||
		overlayButton(t, w, "Remove the Numbers").Importance == widget.HighImportance {
		t.Error("Keep is not the default after an issue")
	}
	test.Tap(overlayButton(t, w, "Keep the Numbers"))
	text := w.textEditor.Text()
	if sceneLockOf(text).locked || !strings.Contains(text, "INT. SHED - DAY #8#") {
		t.Errorf("Keep:\n%s", text)
	}
}

// a locked script with A-numbers, a gap and an OMITTED scene
const renumberScript = "INT. HOUSE - DAY #1#\n\nRain.\n\nINT. HALL - DAY #1A#\n\nCoats.\n\nEXT. GARDEN - DAY #3# [[red]]\n\nMore rain.\n\n.OMITTED #4#\n\n/* Omitted 4\nINT. CELLAR - NIGHT #4#\n\nDark.\n*/\n\nINT. HOUSE - NIGHT #5#\n\nDry.\n\n/* Production\nscene numbers: locked, 10A\nnumbered before the lock: none\n*/\n"

// Renumber: 1…n in order, A-numbers and gaps gone, the OMITTED scene's
// number too (its text kept as boneyard); one undo step; at once before
// an issue.
func TestRenumber(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, renumberScript)
	w.renumberScenes()
	if len(w.fyneWindow.Canvas().Overlays().List()) > 0 {
		t.Error("Renumber asked before an issue")
	}
	want := "INT. HOUSE - DAY #1#\n\nRain.\n\nINT. HALL - DAY #2#\n\nCoats.\n\nEXT. GARDEN - DAY #3# [[red]]\n\nMore rain.\n\n/* Omitted scene\nINT. CELLAR - NIGHT\n\nDark.\n*/\n\nINT. HOUSE - NIGHT #4#\n\nDry.\n\n/* Production\nscene numbers: locked, 10A\nnumbered before the lock: none\n*/\n"
	if got := w.textEditor.Text(); got != want {
		t.Errorf("renumbered:\n%s", got)
	}
	w.undo()
	if got := w.textEditor.Text(); got != renumberScript {
		t.Errorf("Undo:\n%s", got)
	}
	// Remove: the OMITTED scene's number goes the same way
	w.unlockNow(true)
	text := w.textEditor.Text()
	if strings.Contains(text, "#") || !strings.Contains(text, "/* Omitted scene\nINT. CELLAR - NIGHT\n") || strings.Contains(text, "OMITTED") {
		t.Errorf("Remove with OMITTED:\n%s", text)
	}
	w.undo()
	// an OMITTED scene restored after Undo is whole
	w.textEditor.SetCursorOffset(len([]rune(renumberScript[:strings.Index(renumberScript, ".OMITTED")])))
	w.restoreScene()
	if !strings.Contains(w.textEditor.Text(), "INT. CELLAR - NIGHT #4#\n\nDark.\n\nINT. HOUSE - NIGHT #5#") {
		t.Errorf("restore after undo:\n%s", w.textEditor.Text())
	}
}

// After an issue Renumber warns, and offers a draft first.
func TestRenumberAfterIssue(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, renumberScript)
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	issued := w.textEditor.Text()
	w.renumberScenes()
	if !dialogShows(w, "Sides, schedules and printed pages that went out use the old numbers") {
		t.Fatal("no warning")
	}
	test.Tap(overlayButton(t, w, "Cancel"))
	if w.textEditor.Text() != issued {
		t.Fatal("Cancel renumbered")
	}
	w.renumberScenes()
	test.Tap(overlayButton(t, w, "Save a Draft, Then Renumber"))
	waitFor(t, "renumbered", func() bool { return strings.Contains(w.textEditor.Text(), "INT. HALL - DAY #2#") })
	ds, err := readDrafts(w.extrasHome())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range ds {
		names = append(names, d.Name)
	}
	if !strings.Contains(strings.Join(names, "|"), "Before renumbering") {
		t.Errorf("drafts %v", names)
	}
}

// a script whose words recur: "revolver" in both kitchens
const followScript = `Title: The Shed

INT. KITCHEN - NIGHT

Anna puts the revolver on the table.

EXT. STREET - NIGHT

The taxi waits. A dog barks at it.

INT. KITCHEN - NIGHT

The revolver is gone. Nobody speaks.
`

// tagLine is the line the tag of name is found on, -1 lost.
func tagLine(w *MainWindow, name string) int {
	w.updateBreakdown()
	_, bt, ps := w.breakdownNowFound()
	for _, p := range ps {
		if p.element.Name == name && !p.lost {
			line, _ := bt.lineCol(p.start)
			return line
		}
	}
	return -1
}

// Tags, a scene marked done and strips keep their scenes through Renumber
// and Remove, and through Undo and Redo of them; the stored numbers follow.
func TestRenumberKeepsTagsAndStrips(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, followScript)
	w.lockSceneNumbers()
	// a scene put in while locked: 1A
	replaceText(t, w, "EXT. STREET", "INT. HALL - NIGHT #1A#\n\nCoats.\n\nEXT. STREET")
	// the second kitchen's revolver, stored with its number (3)
	tagNow(t, w, "revolver", 1, "Props")
	if f := w.breakdownNow(); len(f.Tags) != 1 || f.Tags[0].Scene != "3" {
		t.Fatalf("tag %+v", f.Tags)
	}
	w.markSceneDone(3)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(sceneWith(t, w, "INT. KITCHEN - NIGHT"), false) // the first kitchen
	sb.moveSelected(0, 0)
	sb.selectScene(3, false)
	sb.moveSelected(0, 1)
	line := func() int { return editorLineWith(t, w, "The revolver is gone") }
	check := func(what, num string) {
		t.Helper()
		if got := tagLine(w, "revolver"); got != line() {
			t.Errorf("%s: the tag on line %d, not %d", what, got, line())
		}
		f, bt, p := w.scheduleFound()
		got := p.scenesOf(0)
		if len(got) != 2 || got[0] != 0 || got[1] != 3 {
			t.Errorf("%s: day 1 %v", what, got)
		}
		if s := f.Days[0].Strips[1]; s.Scene != num {
			t.Errorf("%s: the strip's number %q, not %q", what, s.Scene, num)
		}
		for _, sc := range got {
			if changed, _ := f.changed(bt, p, sc); changed {
				t.Errorf("%s: scene %d flagged as changed", what, sc)
			}
		}
		bf := w.breakdownNow()
		if bf.Tags[0].Scene != num || !bf.isDone(bt.scenes[3]) || bf.isDone(bt.scenes[0]) {
			t.Errorf("%s: tag %+v, done %+v", what, bf.Tags[0], bf.Done)
		}
	}
	check("locked", "3")
	w.renumberScenes()
	if !strings.Contains(w.textEditor.Text(), "INT. KITCHEN - NIGHT #4#") {
		t.Fatal("not renumbered")
	}
	check("renumbered", "4")
	w.undo()
	check("undo", "3")
	w.redo()
	check("redo", "4")
	// Undo in the editor itself (Ctrl+Z there): followed once typing pauses
	w.textEditor.Buffer().Undo()
	w.textEditor.Refresh()
	w.updateBreakdown()
	check("the editor's undo", "3")
	w.unlockNow(true)
	if strings.Contains(w.textEditor.Text(), "#") {
		t.Fatal("numbers left")
	}
	check("removed", "")
	w.undo()
	check("remove undone", "3")
}

// The locked pages' scenes follow Renumber in the same undo step.
func TestRenumberFollowsLockedPages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	w.lockSceneNumbers()
	text := w.textEditor.Text()
	// make the second scene an A-scene, so Renumber changes the numbers after it
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	if len(heads) < 3 {
		t.Fatalf("heads %v", heads)
	}
	replaceText(t, w, "#2#", "#1A#")
	w.lockPages()
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	before := pageLockOf(w.textEditor.Text())
	w.renumberNow()
	after := pageLockOf(w.textEditor.Text())
	if after == nil || len(after.lock.Pages) != len(before.lock.Pages) {
		t.Fatalf("the page lock: %+v", after)
	}
	for i, p := range before.lock.Pages {
		want := p.Scene
		if want == "1A" {
			want = "2"
		}
		if after.lock.Pages[i].Scene != want {
			t.Errorf("page %s: scene %q, want %q", p.Label, after.lock.Pages[i].Scene, want)
		}
	}
	w.undo()
	if got := pageLockOf(w.textEditor.Text()); got.lock.Pages[len(got.lock.Pages)-1].Scene != before.lock.Pages[len(before.lock.Pages)-1].Scene {
		t.Error("Undo did not bring the page lock's numbers back")
	}
}

// The pane's Lock stage: Unlock and Renumber beside Lock once locked.
func TestProductionUnlockRenumberButtons(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	p := prodPane(t, w)
	if p.numbersRow.Visible() && len(p.buttons.Objects) > 0 && p.buttons.Objects[0] == p.numbersRow {
		t.Error("Unlock and Renumber shown before the lock")
	}
	test.Tap(p.lockNumbers)
	p.refresh()
	if p.buttons.Objects[0] != p.numbersRow {
		t.Fatal("no Unlock / Renumber beside the lock")
	}
	if !strings.HasPrefix(p.count.Text, "Numbers locked, not issued · ") {
		t.Errorf("count %q", p.count.Text)
	}
	replaceText(t, w, "#2#", "#1A#")
	test.Tap(p.renumber)
	if !strings.Contains(w.textEditor.Text(), "#2#") || strings.Contains(w.textEditor.Text(), "#1A#") {
		t.Errorf("Renumber:\n%s", w.textEditor.Text())
	}
	test.Tap(p.unlockNumbers)
	if !dialogShows(w, "Unlock Scene Numbers") {
		t.Fatal("Unlock… shows no dialog")
	}
	test.Tap(overlayButton(t, w, "Remove the Numbers"))
	p.refresh()
	if sceneLockOf(w.textEditor.Text()).locked || p.buttons.Objects[0] != p.lockNumbers || p.lockNumbers.Disabled() {
		t.Error("after Unlock: still locked, or Lock not back")
	}
}

// SHOT_DIR renders the Unlock dialog and the locked Lock stage, wide and
// on a phone.
func TestUnlockDialogScreenshots(t *testing.T) {
	holdUI(t)
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, unlockScript)
	w.lockSceneNumbers()
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		clearOverlays(w)
		w.fyneWindow.Resize(size.size)
		w.unlockDialog().Show()
		capture(t, w.fyneWindow.Canvas(), "unlock-"+size.name+".png")
	}
	clearOverlays(w)
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	// the pane's Lock stage, locked: Unlock and Renumber beside it
	prodPane(t, w)
	comboShot(t, w, "prodpane-locked")
}

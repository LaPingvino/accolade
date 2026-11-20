package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
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

// Renumber, Clean 1…n (the preset before an issue): 1…n in order,
// A-numbers and gaps gone, the OMITTED scene's number too (its text kept
// as boneyard); one undo step.
func TestRenumber(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, renumberScript)
	w.renumberScenes()
	if !dialogShows(w, presetClean) {
		t.Error("no dialog, or not the Clean preset, before an issue")
	}
	test.Tap(overlayButton(t, w, "Renumber"))
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
	if w.renumbering.preset.Selected != presetSent {
		t.Errorf("after an issue the preset is %q", w.renumbering.preset.Selected)
	}
	if !overlayButton(t, w, "Save a Draft, Then Renumber").Disabled() {
		t.Error("Keep sent scenes: every number went out, yet something to renumber")
	}
	w.renumbering.preset.SetSelected(presetClean)
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
	test.Tap(overlayButton(t, w, "Renumber"))
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

// threeLongScenes is three scenes, each over a page long.
func threeLongScenes() string {
	var b strings.Builder
	b.WriteString("Title: Pages\n\n")
	for _, h := range []string{"INT. HOUSE - DAY", "EXT. GARDEN - DAY", "INT. HOUSE - NIGHT"} {
		b.WriteString(h + "\n\n")
		for i := range 40 {
			fmt.Fprintf(&b, "Line %d of %s goes on for a while, long enough to wrap once on the page.\n\n", i, strings.ToLower(h))
		}
	}
	return b.String()
}

// The locked pages' scenes follow Renumber and Remove in the same undo
// step.
func TestRenumberFollowsLockedPages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, threeLongScenes())
	w.lockSceneNumbers()
	// the second scene an A-scene, so Renumber changes the numbers from it
	replaceText(t, w, "#2#", "#1A#")
	replaceText(t, w, "#3#", "#2#")
	w.lockPages()
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	before := pageLockOf(w.textEditor.Text())
	var scenes []string
	for _, p := range before.lock.Pages {
		scenes = append(scenes, p.Scene)
	}
	if !slices.Contains(scenes, "1A") || !slices.Contains(scenes, "2") {
		t.Fatalf("pages' scenes %v: none starts in 1A or 2", scenes)
	}
	w.renumberNow()
	after := pageLockOf(w.textEditor.Text())
	if after == nil || len(after.lock.Pages) != len(before.lock.Pages) {
		t.Fatalf("the page lock: %+v", after)
	}
	to := map[string]string{"1": "1", "1A": "2", "2": "3", "": ""}
	for i, p := range before.lock.Pages {
		if got := after.lock.Pages[i].Scene; got != to[p.Scene] {
			t.Errorf("page %s: scene %q, want %q", p.Label, got, to[p.Scene])
		}
	}
	w.undo()
	if got := pageLockOf(w.textEditor.Text()); !slices.EqualFunc(got.lock.Pages, before.lock.Pages, func(a, b lex.LockedPage) bool { return a == b }) {
		t.Error("Undo did not bring the page lock back")
	}
	w.unlockNow(true)
	removed := pageLockOf(w.textEditor.Text())
	if removed == nil {
		t.Fatal("Remove took the page lock out")
	}
	for _, p := range removed.lock.Pages {
		if p.Scene != "" {
			t.Errorf("page %s still names scene %q", p.Label, p.Scene)
		}
	}
	// every page still in the block (its scene "none": found by its words)
	if got := len(lex.ParsePageLock(mustProductionBlock(t, w)).Pages); got != len(before.lock.Pages) {
		t.Errorf("%d pages after Remove, %d before", got, len(before.lock.Pages))
	}
}

// mustProductionBlock is the script's production block's inside.
func mustProductionBlock(t *testing.T, w *MainWindow) string {
	t.Helper()
	inside, ok := productionBlock(w.textEditor.Text())
	if !ok {
		t.Fatal("no production block")
	}
	return inside
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
	test.Tap(overlayButton(t, w, "Renumber"))
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

// a locked script whose third scene will be omitted
const omitFollowScript = "INT. HOUSE - DAY #1#\n\nRain.\n\nINT. HALL - DAY #1A#\n\nCoats.\n\nINT. CELLAR - NIGHT #2#\n\nDark cellar stairs creak.\n\nINT. HOUSE - NIGHT #3#\n\nDry stairs creak too.\n\n/* Production\nscene numbers: locked, 10A\nnumbered before the lock: none\n*/\n"

// What named an OMITTED scene that Renumber or Remove takes away does not
// move to the scene that gets its number; Undo keeps it off it too.
func TestRenumberDropsOmittedKeys(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, omitFollowScript)
	tagNow(t, w, "stairs creak", 0, "Props")
	w.markSceneDone(2)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(2, false)
	sb.moveSelected(0, 0)
	w.textEditor.SetCursorOffset(len([]rune(omitFollowScript[:strings.Index(omitFollowScript, "Dark cellar")])))
	w.omitScene()
	if !strings.Contains(w.textEditor.Text(), ".OMITTED #2#") {
		t.Fatalf("not omitted:\n%s", w.textEditor.Text())
	}
	// none of it on another scene: the hall gets the cellar's 2, the house
	// at night shares its words
	notOnHouseNight := func(what string) {
		t.Helper()
		others := []int{editorLineWith(t, w, "INT. HALL - DAY"), editorLineWith(t, w, "INT. HOUSE - NIGHT")}
		dry := editorLineWith(t, w, "Dry stairs creak")
		if got := tagLine(w, "stairs creak"); got == dry {
			t.Errorf("%s: the cellar's tag moved to the house at night", what)
		}
		f, bt, p := w.scheduleFound()
		for _, sc := range p.scenesOf(0) {
			if slices.Contains(others, bt.scenes[sc].line) {
				t.Errorf("%s: the cellar's strip moved to %q (%+v)", what, bt.scenes[sc].heading, f.Days[0].Strips)
			}
		}
		for _, s := range bt.scenes {
			if slices.Contains(others, s.line) && w.breakdownNow().isDone(s) {
				t.Errorf("%s: %q marked done", what, s.heading)
			}
		}
	}
	w.renumberNow()
	if !strings.Contains(w.textEditor.Text(), "INT. HOUSE - NIGHT #3#") || strings.Contains(w.textEditor.Text(), "OMITTED") {
		t.Fatalf("renumbered:\n%s", w.textEditor.Text())
	}
	notOnHouseNight("renumbered")
	w.undo()
	notOnHouseNight("undone")
	w.unlockNow(true)
	notOnHouseNight("removed")
}

// A number retyped through a number twice (#2# → #1# → #5#) and a
// Renumber from a number twice keep what names a scene on it.
func TestFollowThroughDuplicateNumbers(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, followScript)
	w.lockSceneNumbers()
	tagNow(t, w, "revolver", 1, "Props") // the second kitchen, 3
	w.markSceneDone(0)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(1, false) // the street, 2
	sb.moveSelected(0, 0)
	keys := func() (string, string) {
		f := w.scheduleNow()
		return f.Days[0].Strips[0].Scene, w.breakdownNow().Tags[0].Scene
	}
	replaceText(t, w, "EXT. STREET - NIGHT #2#", "EXT. STREET - NIGHT #1#")
	w.updateBreakdown()
	replaceText(t, w, "EXT. STREET - NIGHT #1#", "EXT. STREET - NIGHT #5#")
	w.updateBreakdown()
	if s, tag := keys(); s != "5" || tag != "3" {
		t.Errorf("retyped: strip %q, tag %q", s, tag)
	}
	_, _, p := w.scheduleFound()
	if got := p.scenesOf(0); len(got) != 1 || got[0] != 1 {
		t.Errorf("retyped: the strip on %v", got)
	}
	// a number twice, then Renumber
	replaceText(t, w, "EXT. STREET - NIGHT #5#", "EXT. STREET - NIGHT #1#")
	w.updateBreakdown()
	w.renumberNow()
	if !strings.Contains(w.textEditor.Text(), "EXT. STREET - NIGHT #2#") {
		t.Fatalf("renumbered:\n%s", w.textEditor.Text())
	}
	_, bt, p := w.scheduleFound()
	if got := p.scenesOf(0); len(got) != 1 || got[0] != 1 {
		t.Errorf("renumbered: the strip on %v", got)
	}
	bf := w.breakdownNow()
	if !bf.isDone(bt.scenes[0]) || bf.isDone(bt.scenes[1]) || bf.isDone(bt.scenes[2]) {
		t.Errorf("done marks %+v", bf.Done)
	}
	if got := tagLine(w, "revolver"); got != editorLineWith(t, w, "The revolver is gone") {
		t.Errorf("the tag on line %d", got)
	}
}

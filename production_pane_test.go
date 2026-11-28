// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

// checkScript has one of each thing the check before the lock finds,
// its lines known.
func checkScript() string {
	long := strings.Repeat("She looks around the long room, then at the windows, then at the door again and waits.\n\n", 120)
	return strings.Join([]string{
		"Title: Checks", // 0
		"",
		"INT. KITCHEN - DAY #1#", // 2
		"",
		"MRS. BENNET", // 4
		"Tea.",
		"",
		"EXT. GARDEN #2#", // 7: no time
		"",
		"MRS. BENNET", // 9
		"More tea.",
		"",
		".THE ROOF #3#", // 12: no INT./EXT., no time
		"",
		"MRS BENNET", // 14: the other spelling
		"Coffee.",
		"",
		"ANNIE", // 17: Anna's other name, learned
		"Hello.",
		"",
		"INT. HALL - NIGHT #2#", // 20: number twice
		"",
		"ANNA", // 22
		"Hi.",
		"",
		"INT. ATTIC - DAY", // 25: no number
		"",
		"[[ a note left open", // 27
		"",
		"INT. CELLAR - NIGHT #5#", // 29: very long
		"",
		long + "/* the end, not closed", // 31 + 240
		"",
		"/* Adaptation",
		"characters: ANNA = Anna, Annie",
		"*/",
		"",
	}, "\n")
}

// A check of each kind, on its line; the adaptation block is not read as
// the script (its /* is closed).
func TestPreLockChecks(t *testing.T) {
	holdUI(t)
	text := checkScript()
	got := preLockChecks(text, sceneStarts("en"))
	want := []struct {
		line int
		kind string
		has  string
	}{
		{7, checkTime, "EXT. GARDEN"},
		{12, checkPlace, "No INT./EXT. and no time"},
		{14, checkCue, "MRS BENNET (1 cue) and MRS. BENNET (2 cues)"},
		{17, checkCue, "ANNIE (1 cue) is ANNA's other name"},
		{20, checkNumber, "Number 2 twice (also line 8)"},
		{25, checkNumber, "No number"},
		{27, checkNote, "[[ not closed"},
		{29, checkLong, "Very long"},
		{271, checkBoneyard, "/* not closed"},
	}
	for _, w := range want {
		found := false
		for _, c := range got {
			if c.line == w.line && c.kind == w.kind && strings.Contains(c.text, w.has) {
				found = true
			}
		}
		if !found {
			t.Errorf("no %s check on line %d (%q)", w.kind, w.line, w.has)
		}
	}
	if len(got) != len(want) {
		for _, c := range got {
			t.Logf("%d %s %s", c.line, c.kind, c.text)
		}
		t.Errorf("%d checks, want %d", len(got), len(want))
	}
	for i := 1; i < len(got); i++ {
		if got[i].line < got[i-1].line {
			t.Errorf("not in the order of the lines: %v", got)
		}
	}
}

// A clean script: a note that the numbers are still to be written, no
// problem; without a title page, that; stray marks found.
func TestPreLockChecksClean(t *testing.T) {
	holdUI(t)
	got := preLockChecks(revScript, sceneStarts("en"))
	if len(got) != 1 || !got[0].info || got[0].kind != checkNumber || problems(got) != 0 {
		t.Errorf("clean script: %+v", got)
	}
	got = preLockChecks("INT. A - DAY #1#\n\nRain. */ and ]] here.\n\nOMITTED #2#\n", sceneStarts("en"))
	kinds := map[string]bool{}
	for _, c := range got {
		kinds[c.kind] = true
	}
	if !kinds[checkTitle] || !kinds[checkBoneyard] || !kinds[checkNote] || len(got) != 3 {
		t.Errorf("stray closes, no title page: %+v", got)
	}
	// an omitted scene's heading is no heading to check; its boneyard holds a cue
	omitted := "Title: X\n\nINT. A - DAY #1#\n\nANNA\nHi.\n\n.OMITTED #2#\n\n/* Omitted 2\nINT. B\n\nANNA.\nGone.\n*/\n\nINT. C - DAY #3#\n\nRain.\n"
	if got := preLockChecks(omitted, sceneStarts("en")); len(got) != 0 {
		t.Errorf("omitted scene: %+v", got)
	}
}

// changedScenes: by number, the title page's own, in order.
func TestChangedScenes(t *testing.T) {
	holdUI(t)
	text := "Title: X\n\nINT. A - DAY #1#\n\nRain.\n\nINT. B - DAY #2A#\n\nSun.\n\nMore sun.\n"
	marks := make([]string, strings.Count(text, "\n")+1)
	marks[0], marks[8], marks[10] = "Blue", "Blue", "Blue"
	marks[4] = "White" // another set's: not this revision's
	got := changedScenes(text, sceneStarts("en"), marks, "Blue")
	if len(got) != 2 || got[0].label != "Title page" || got[1].label != "2A" || got[1].line != 6 {
		t.Fatalf("changed %+v", got)
	}
	if s := sceneList(got[1:]); s != "scene 2A" {
		t.Errorf("list %q", s)
	}
	var many []changedScene
	for i := 1; i <= 10; i++ {
		many = append(many, changedScene{label: fmt.Sprint(i)})
	}
	if s := sceneList(many); s != "scenes 1, 2, 3, 4, 5, 6, 7, 8, and 2 more" {
		t.Errorf("list %q", s)
	}
}

// prodPane is the window's Production pane, shown.
func prodPane(t *testing.T, w *MainWindow) *productionPane {
	t.Helper()
	w.togglePane("production")
	p := w.productionPane()
	if !p.shown() {
		t.Fatal("the pane is not shown")
	}
	return p
}

// rowLabels are the pane's list's rows' texts (the tap labels).
func rowLabels(p *productionPane) []*tapLabel {
	var out []*tapLabel
	for _, o := range p.rows.Objects {
		if c, ok := o.(*fyne.Container); ok {
			for _, x := range c.Objects {
				if l, ok := x.(*tapLabel); ok {
					out = append(out, l)
				}
			}
		}
	}
	return out
}

// The stages: Lock before an issue, Revise after; the counts follow the
// edits; Lock can be looked at again.
func TestProductionStages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	p := prodPane(t, w)
	if p.stage != stageLock || p.view() != stageLock {
		t.Fatalf("stage %d", p.stage)
	}
	if got := p.count.Text; got != "Not locked yet · 3 scenes · nothing to check" {
		t.Errorf("count %q", got)
	}
	if p.stages[stageLock].Importance != widget.HighImportance || p.stages[stageRevise].Importance != widget.LowImportance {
		t.Error("the stage is not highlighted")
	}
	// an edit makes a problem: the count follows
	replaceText(t, w, "EXT. GARDEN - DAY", "EXT. GARDEN")
	if got := p.count.Text; got != "Not locked yet · 3 scenes · 1 to check" {
		t.Errorf("after an edit %q", got)
	}
	replaceText(t, w, "EXT. GARDEN", "EXT. GARDEN - DAY")

	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	if p.stage != stageRevise || p.view() != stageRevise {
		t.Fatalf("after White: stage %d", p.stage)
	}
	if got := p.count.Text; got != "White issued 1 Nov 2026 · no changes since" {
		t.Errorf("after White %q", got)
	}
	if p.issueNext.Text != "Issue Blue…" || !p.issueNext.Disabled() || !p.revisedPDF.Disabled() {
		t.Errorf("Issue %q (disabled %v)", p.issueNext.Text, p.issueNext.Disabled())
	}
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	if got := p.count.Text; got != "Blue (open) · 1 page changed · scene 2" {
		t.Errorf("after an edit %q", got)
	}
	if p.issueNext.Disabled() || p.revisedPDF.Disabled() {
		t.Error("Issue Blue or the PDF disabled with a change")
	}
	replaceText(t, w, "Dry at last.", "Dry at last, almost.")
	if got := p.count.Text; got != "Blue (open) · 1 page changed · scenes 2, 3" {
		t.Errorf("after a second edit %q", got)
	}
	if ls := rowLabels(p); len(ls) != 2 || !strings.HasPrefix(ls[0].Text, "2  EXT. GARDEN") {
		t.Errorf("rows %d", len(ls))
	}
	// all issues' marks in the margin: the count is still the revision's
	w.settings.SetString(revisionMarksPref, "all")
	t.Cleanup(func() { w.settings.SetString(revisionMarksPref, "last") })
	issueNow(t, w, "Blue", time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local))
	if got := p.count.Text; got != "Blue issued 20 Nov 2026 · 1 page changed · scenes 2, 3" {
		t.Errorf("Blue issued %q", got)
	}
	replaceText(t, w, "Rain on the roof.", "Rain on the roof, loud.")
	if got := p.count.Text; got != "Pink (open) · 1 page changed · scene 1" {
		t.Errorf("Pink %q", got)
	}
	// Lock again, looked at: done, the count of the lock
	test.Tap(p.stages[stageLock])
	if p.view() != stageLock || p.stages[stageLock].Icon == nil || p.stages[stageRevise].Importance != widget.MediumImportance {
		t.Error("Lock not shown as done")
	}
	if got := p.count.Text; !strings.HasPrefix(got, "Locked 1 Nov 2026 · ~1 page · 3 scenes") {
		t.Errorf("lock's count %q", got)
	}
	if !p.issueWhite.Disabled() || !p.lockNumbers.Disabled() {
		t.Error("Lock's buttons after the lock")
	}
	test.Tap(p.stages[stageRevise])
	if p.view() != stageRevise || p.picked != -1 {
		t.Error("back to Revise")
	}
}

// A tap on a check or a changed scene goes to its line.
func TestProductionRowsGo(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, checkScript())
	p := prodPane(t, w)
	ls := rowLabels(p)
	if len(ls) != len(p.checks) {
		t.Fatalf("%d rows for %d checks", len(ls), len(p.checks))
	}
	for i, c := range p.checks {
		if c.kind == checkTitle {
			continue
		}
		test.Tap(ls[i])
		if got := cursorLine(w); got != c.line {
			t.Errorf("%q: went to line %d, want %d", c.text, got, c.line)
		}
	}
	// the keyboard's button does the same
	w.goToLine(0)
	row := p.rows.Objects[len(p.rows.Objects)-1].(*fyne.Container)
	tapped := false
	for _, o := range row.Objects {
		if b, ok := o.(*goButton); ok {
			test.Tap(b)
			tapped = true
		}
	}
	if !tapped {
		t.Fatal("no go button")
	}
	if got := cursorLine(w); got != p.checks[len(p.checks)-1].line {
		t.Errorf("the go button went to %d", got)
	}
	if w.fyneWindow.Canvas().Focused() != w.textEditor {
		t.Error("the editor has not the focus")
	}

	w2, _ := scriptWindow(t, revScript)
	p2 := prodPane(t, w2)
	issueNow(t, w2, "White", time.Now())
	replaceText(t, w2, "Dry at last.", "Dry at last, almost.")
	ls = rowLabels(p2)
	if len(ls) != 1 {
		t.Fatalf("%d changed rows", len(ls))
	}
	test.Tap(ls[0])
	if got := cursorLine(w2); got != editorLineWith(t, w2, "INT. HOUSE - NIGHT") {
		t.Errorf("went to %d", got)
	}
	// a stale line (the text shorter now) goes to the end
	w2.goToLine(10000)
	if got, n := cursorLine(w2), strings.Count(w2.textEditor.Text(), "\n"); got != n {
		t.Errorf("stale line: %d of %d", got, n)
	}
}

// The buttons are the menu's actions.
func TestProductionButtons(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	p := prodPane(t, w)
	test.Tap(p.lockNumbers)
	if !sceneLockOf(w.textEditor.Text()).locked {
		t.Error("Lock Scene Numbers did not lock")
	}
	if !p.lockNumbers.Disabled() || p.lockNumbers.Text != "Scene Numbers Locked" {
		t.Errorf("after the lock: %q", p.lockNumbers.Text)
	}
	test.Tap(p.issueWhite)
	if !dialogShows(w, "Issue White") || !dialogShows(w, "Lock the pages as White prints them") {
		t.Error("Issue White… shows no dialog offering the lock")
	}
	clearOverlays(w)
	test.Tap(p.lockPages)
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	p.refresh()
	if p.lockPages.Text != "Lock Pages Again" {
		t.Errorf("Lock Pages: %q", p.lockPages.Text)
	}
	issueWait(t, w, "White", 1, "")
	replaceText(t, w, "Mud everywhere.", "Mud.")
	test.Tap(p.issueNext)
	if !dialogShows(w, "Issue Blue") {
		t.Error("Issue Blue… shows no dialog")
	}
	clearOverlays(w)
	test.Tap(p.revisedPDF)
	if !dialogShows(w, "revised pages only") { // its page setup line
		t.Error("Revised Pages PDF… shows no export")
	}
	clearOverlays(w)
	test.Tap(p.compare)
	if w.compare == nil || w.mid.shown != "compare" || !strings.Contains(w.compare.from.Selected, "White") { // a middle tab now
		t.Error("Compare with White shows no comparison")
	}
	w.closeCompare()
}

// Revised Pages PDF… sets the export up: PDF, the marks, revised pages
// only (this export's, the page setup left as it is, its line saying so),
// a name of its own.
func TestRevisedPagesExport(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	issueNow(t, w, "White", time.Now())
	replaceText(t, w, "Mud everywhere.", "Mud.")
	got := w.showRevisedPagesExport("Blue")
	if got == nil {
		t.Fatal("no export dialog")
	}
	j := got.job()
	if exportFormatByLabel(got.formatSelect.Selected).id != "pdf" || j.revisions == nil || !j.revisions.PagesOnly {
		t.Errorf("format %q, marks %+v", got.formatSelect.Selected, j.revisions)
	}
	if w.pageSetup().revisedOnly || !strings.Contains(got.pageLine.summary.Text, "revised pages only") {
		t.Errorf("this export's revised pages: the page setup's %v, the line %q", w.pageSetup().revisedOnly, got.pageLine.summary.Text)
	}
	if !strings.HasSuffix(got.filenameEntry.Text, "-blue-pages.pdf") {
		t.Errorf("name %q", got.filenameEntry.Text)
	}
}

// clearOverlays closes the window's dialogs.
func clearOverlays(w *MainWindow) {
	for _, o := range w.fyneWindow.Canvas().Overlays().List() {
		w.fyneWindow.Canvas().Overlays().Remove(o)
	}
}

// The changed pages are those revised pages only prints: with locked
// pages, the page and its A-page.
func TestProductionPageCount(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, pagesScript())
	p := prodPane(t, w)
	done := false
	w.issueDraft(lex.RevisionSet{Name: "White", Date: time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)}, "", true, func(err error) {
		if err != nil {
			t.Error(err)
		}
		done = true
	})
	waitFor(t, "White issued with its pages locked", func() bool { return done })
	if pageLockOf(w.textEditor.Text()) == nil {
		t.Fatal("Issue White with the lock did not lock the pages")
	}
	replaceText(t, w, "R3P2 ", "R3P2 "+strings.Repeat("Blue words go on and on here. ", 40))
	only := revisedOnly(t, w)
	if len(only) != 2 || !strings.HasSuffix(only[1], "A") {
		t.Fatalf("revised pages only %v", only)
	}
	if want := fmt.Sprintf("Blue (open) · %d pages changed · scene 3", len(only)); p.count.Text != want {
		t.Errorf("count %q, want %q", p.count.Text, want)
	}
}

// The pane on a phone: under the script, no wider than the window; the
// toolbar's button in the menu, View > Production there; a Markdown
// document has neither.
func TestProductionNarrow(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, checkScript())
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	p := prodPane(t, w)
	if got := w.fyneWindow.Canvas().Capture().Bounds().Dx(); got != 380 {
		t.Errorf("the window is %d wide", got)
	}
	if mw := w.paneSplit.Trailing.MinSize().Width; mw > sourceWidth {
		t.Errorf("the panes are at least %v wide", mw)
	}
	if p.box.MinSize().Width > sourceWidth {
		t.Errorf("the production pane is at least %v wide", p.box.MinSize().Width)
	}
	if w.paneSplit == nil {
		t.Error("the panes are not under the script")
	}
	if !p.narrow || len(p.inList.Objects) != 2 || len(p.bottom.Objects) != 0 {
		t.Error("on a phone the buttons are not at the list's end")
	}
	if w.headerBar.button("view.production").Visible() {
		t.Error("the toolbar's Production button stays on a phone's row")
	}
	if !w.headerBar.button("view.source").Visible() {
		t.Error("the source button went")
	}
	menuItemByLabel(t, w, "Production")
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	w.fyneWindow.Content().Refresh()
	if !w.headerBar.button("view.production").Visible() {
		t.Error("the toolbar's Production button is not on a wide row")
	}
	if p.narrow || len(p.inList.Objects) != 1 || len(p.bottom.Objects) != 1 {
		t.Error("beside the script the buttons are not under the list")
	}

	md, _ := scriptWindow(t, revScript)
	md.setMarkdown(true)
	for _, m := range md.fyneWindow.MainMenu().Items {
		for _, it := range m.Items {
			if it.Label == "Production" {
				t.Error("Markdown's View menu has Production")
			}
		}
	}
	if !md.headerBar.button("view.production").Disabled() {
		t.Error("Markdown's toolbar has Production")
	}
}

// SHOT_DIR renders the Lock and Revise stages wide and on a phone.
func TestProductionPaneScreenshots(t *testing.T) {
	holdUI(t)
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, checkScript())
	prodPane(t, w)
	w.goToLine(0)
	comboShot(t, w, "prodpane-lock")

	w2, _ := scriptWindow(t, revScript)
	p := prodPane(t, w2)
	issueNow(t, w2, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w2, "Mud everywhere.", "Mud everywhere, and a dog digging in it.")
	replaceText(t, w2, "Dry at last.", "Dry at last, almost.")
	w2.goToLine(editorLineWith(t, w2, "Mud everywhere"))
	p.refresh()
	comboShot(t, w2, "prodpane-revise")
}

// Everything Tab reaches in the pane shows its focus.
func TestProductionFocusVisible(t *testing.T) {
	holdUI(t)
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race")
	}
	w, _ := scriptWindow(t, checkScript())
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	prodPane(t, w)
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
}

package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// strategyScript is a locked script: an A-scene, a gap, two OMITTED
// scenes one after the other, another A-scene and a gap at the end.
func strategyScript() string {
	o4, _ := fountain.OmitScene("INT. D - DAY #4#\n\nDee.\n\n")
	o5, _ := fountain.OmitScene("INT. E - DAY #5#\n\nEe.\n\n")
	return "INT. A - DAY #1#\n\nAy.\n\nINT. B - DAY #1A#\n\nBee.\n\nINT. C - DAY #3#\n\nSee.\n\n" + o4 + o5 +
		"INT. F - DAY #6#\n\nEff.\n\nINT. G - DAY #6A#\n\nGee.\n\nINT. H - DAY #9#\n\nAitch.\n\n" +
		"/* Production\nscene numbers: locked, 10A\nnumbered before the lock: 9\n*/\n"
}

// headNums are the script's headings' numbers, an OMITTED one's with O.
func headNums(w *MainWindow) string {
	text := w.textEditor.Text()
	var out []string
	for _, h := range sceneHeads(text[:bodyEndByte(text)], w.sceneStarts()) {
		n := h.num
		if isOmitted(h) {
			n = "O" + n
		}
		out = append(out, n)
	}
	return strings.Join(out, " ")
}

// expected is what the preview says the headings will be: the same
// notation as headNums.
func expected(heads []sceneHead, p numberPlan) string {
	rows, _ := previewRows(heads, p)
	var out []string
	for i, r := range rows {
		if p.drop[i] || p.merge[i] {
			continue
		}
		n := r.new
		if !r.changed {
			n = heads[i].num
		}
		if isOmitted(heads[i]) {
			n = "O" + n
		}
		out = append(out, n)
	}
	return strings.Join(out, " ")
}

// lineOf is the rune offset of the line holding s.
func offsetOf(t *testing.T, w *MainWindow, s string) int {
	t.Helper()
	text := w.textEditor.Text()
	i := strings.Index(text, s)
	if i < 0 {
		t.Fatalf("no %q", s)
	}
	return len([]rune(text[:i]))
}

// Every strategy and the combinations: the script gets what the preview
// said, in one undo step.
func TestRenumberStrategies(t *testing.T) {
	holdUI(t)
	script := strategyScript()
	w, _ := scriptWindow(t, script)
	cursorIn := func(s string) func() {
		return func() { w.textEditor.SetCursorOffset(offsetOf(t, w, s)) }
	}
	selecting := func(from, to string) func() {
		return func() {
			a, b := offsetOf(t, w, from), offsetOf(t, w, to)
			w.textEditor.Navigate(func(bf *buffer.Buffer) { bf.Select(a, b) })
		}
	}
	for _, c := range []struct {
		name  string
		setup func()
		c     renumberChoice
		want  string
	}{
		{"clean", nil, cleanChoice(), "1 2 3 4 5 6"},
		{"letters kept", nil, renumberChoice{KeepLetters: true}, "1 1A 2 3 3A 4"},
		{"OMITTED kept", nil, renumberChoice{Omitted: lex.OmittedKeep}, "1 2 3 O4 O5 6 7 8"},
		{"OMITTED merged", nil, renumberChoice{Omitted: lex.OmittedMerge}, "1 2 3 O4-5 6 7 8"},
		{"keep 9", nil, renumberChoice{List: listKeep, Numbers: "9"}, "1 2 3 4 5 9"},
		{"keep 1A and 9", nil, renumberChoice{List: listKeep, Numbers: "1A, 9"}, "1 1A 2 3 4 9"},
		{"keep a range", nil, renumberChoice{List: listKeep, Numbers: "6-9"}, "1 2 3 6 6A 9"},
		{"only", nil, renumberChoice{List: listOnly, Numbers: "6A, 9"}, "1 1A 3 O4 O5 6 7 8"},
		{"range, OMITTED kept", nil, renumberChoice{Scope: scopeRange, From: 2, To: 6, Omitted: lex.OmittedKeep}, "1 1A 2 O4 O5 6 7 9"},
		{"range backwards", nil, renumberChoice{Scope: scopeRange, From: 6, To: 2, Omitted: lex.OmittedKeep}, "1 1A 2 O4 O5 6 7 9"},
		{"keep and range", nil, renumberChoice{Scope: scopeRange, From: 1, To: 6, List: listKeep, Numbers: "3"}, "1 2 3 4 5 9"},
		{"from the cursor on", cursorIn("Eff."), renumberChoice{Scope: scopeCursor}, "1 1A 3 O4 O5 6 7 8"},
		{"selected", selecting("See.", "Eff."), renumberChoice{Scope: scopeSelected}, "1 1A 2 3 6A 9"},
		{"selected up to a heading's start: not that scene", selecting("See.", "INT. F"), renumberChoice{Scope: scopeSelected}, "1 1A 2 6 6A 9"},
		{"selected, none: the cursor's scene", cursorIn("Gee."), renumberChoice{Scope: scopeSelected}, "1 1A 3 O4 O5 6 7 9"},
	} {
		w.textEditor.SetCursorOffset(0)
		if c.setup != nil {
			c.setup()
		}
		text := w.textEditor.Text()
		heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
		ctx := w.renumberContextNow(text, heads)
		want := expected(heads, strategyPlan(heads, c.c, ctx))
		if want != c.want {
			t.Errorf("%s: the preview says %q, want %q", c.name, want, c.want)
		}
		w.renumberWith(c.c, ctx)
		if got := headNums(w); got != want {
			t.Errorf("%s: renumbered %q, the preview said %q", c.name, got, want)
		}
		w.undo()
		if w.textEditor.Text() != script {
			t.Fatalf("%s: Undo is not one step:\n%s", c.name, w.textEditor.Text())
		}
	}
}

// The lock's "numbered before" line keeps the hand-typed numbers whose
// scene kept its number; Remove then still leaves them.
func TestRenumberHandNumbers(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, strategyScript())
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	w.renumberWith(renumberChoice{List: listKeep, Numbers: "9"}, w.renumberContextNow(text, heads))
	if hand, _ := handNumbersOf(w.textEditor.Text()); !hand["9"] || len(hand) != 1 {
		t.Errorf("kept 9: hand numbers %v", hand)
	}
	w.undo()
	w.renumberNow()
	if hand, known := handNumbersOf(w.textEditor.Text()); !known || len(hand) != 0 {
		t.Errorf("clean: hand numbers %v", hand)
	}
}

// A merged OMITTED run: one heading with the range, both blocks under
// it; Restore Scene splits it, and then brings each scene back.
func TestRenumberMergeAndRestore(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, strategyScript())
	w.renumberWith(renumberChoice{Omitted: lex.OmittedMerge}, w.renumberContextNow(w.textEditor.Text(), nil))
	text := w.textEditor.Text()
	if strings.Count(text, ".OMITTED") != 1 || !strings.Contains(text, ".OMITTED #4-5#\n\n/* Omitted 4\nINT. D - DAY #4#\n\nDee.\n*/\n\n/* Omitted 5\nINT. E - DAY #5#") {
		t.Fatalf("merged:\n%s", text)
	}
	// a new scene in the locked script never gets a number in the range
	if got := sceneNumbersFor([]sceneHead{{num: "3"}, {num: "4-5"}, {}}, lex.NumberOptions{Locked: true}); got[2] == "4" || got[2] == "5" {
		t.Errorf("a new scene after 4-5 got %s", got[2])
	}
	// Clean on the merged run: both blocks plain boneyard, no number left
	w.renumberNow()
	if text := w.textEditor.Text(); strings.Contains(text, "OMITTED") || strings.Contains(text, "/* Omitted 5") ||
		strings.Count(text, "/* Omitted scene\nINT. ") != 2 {
		t.Errorf("clean after a merge:\n%s", text)
	}
	w.undo()
	w.textEditor.SetCursorOffset(offsetOf(t, w, ".OMITTED #4-5#"))
	w.restoreScene()
	text = w.textEditor.Text()
	if !strings.Contains(text, ".OMITTED #4#\n\n/* Omitted 4\n") || !strings.Contains(text, ".OMITTED #5#\n\n/* Omitted 5\n") {
		t.Fatalf("split:\n%s", text)
	}
	w.textEditor.SetCursorOffset(offsetOf(t, w, ".OMITTED #5#"))
	w.restoreScene()
	if !strings.Contains(w.textEditor.Text(), "INT. E - DAY #5#\n\nEe.\n\nINT. F") {
		t.Errorf("restored:\n%s", w.textEditor.Text())
	}
}

// The presets: Clean before an issue; after one, Keep sent scenes keeps
// every number that went out (a cut scene's too: never given again) and
// numbers only what came since; Last used is the dialog's last choice.
func TestRenumberPresets(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, strategyScript())
	w.renumberScenes()
	r := w.renumbering
	if r.preset.Selected != presetClean || r.choice() != cleanChoice() {
		t.Errorf("before an issue: %q %+v", r.preset.Selected, r.choice())
	}
	for _, o := range r.preset.Options {
		if o == presetLast {
			t.Error("Last used offered before any use")
		}
	}
	// a choice of one's own: from the cursor on, letters kept
	r.scope.SetSelected(scopeOptions[scopeCursor])
	r.letters.SetSelected(letterOptions[1])
	if r.preset.Selected != "" {
		t.Errorf("a control changed, the preset still %q", r.preset.Selected)
	}
	want := expected(r.heads, r.plan())
	var shown []string
	for _, row := range r.rows {
		shown = append(shown, row.old+">"+row.new)
	}
	if len(r.rows) != r.preview.Length() {
		t.Errorf("the list shows %d rows of %d", r.preview.Length(), len(r.rows))
	}
	test.Tap(overlayButton(t, w, "Renumber"))
	if got := headNums(w); got != want {
		t.Errorf("Apply gave %q, the preview %q (%v)", got, want, shown)
	}
	w.undo()
	w.renumberScenes()
	r = w.renumbering
	r.preset.SetSelected(presetLast)
	if c := r.choice(); c.Scope != scopeCursor || !c.KeepLetters {
		t.Errorf("Last used: %+v", c)
	}
	clearOverlays(w)

	// an issue, then a scene cut (3) and one written (unnumbered)
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "INT. C - DAY #3#\n\nSee.\n\n", "INT. N - NIGHT\n\nNew.\n\n")
	w.renumberScenes()
	r = w.renumbering
	if r.preset.Selected != presetSent || r.numbers.Text != "1, 3-6, 9, 1A, 6A" {
		t.Errorf("after an issue: %q, numbers %q", r.preset.Selected, r.numbers.Text)
	}
	if !dialogShows(w, "Sides, schedules and printed pages that went out use the old numbers") {
		t.Error("no warning after the issue")
	}
	if got := expected(r.heads, r.plan()); got != "1 1A 2 O4 O5 6 6A 9" {
		t.Errorf("keep sent: %q", got)
	}
	r.preset.SetSelected(presetClean)
	if got := expected(r.heads, r.plan()); got != "1 2 3 4 5 6" {
		t.Errorf("clean after the issue: %q", got)
	}
}

// The script changing under the open dialog (the lock numbering a new
// scene at a pause, say): Renumber shows the new preview first, and
// applies only on the next tap.
func TestRenumberStalePreview(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, strategyScript())
	w.renumberScenes()
	replaceText(t, w, "INT. H - DAY #9#", "INT. G2 - DAY #8#\n\nMore.\n\nINT. H - DAY #9#")
	changed := w.textEditor.Text()
	test.Tap(overlayButton(t, w, "Renumber"))
	if w.textEditor.Text() != changed || !dialogShows(w, "The script changed meanwhile") {
		t.Fatal("renumbered from a stale preview")
	}
	want := expected(w.renumbering.heads, w.renumbering.plan())
	test.Tap(overlayButton(t, w, "Renumber"))
	if got := headNums(w); got != want || got != "1 2 3 4 5 6 7" {
		t.Errorf("renumbered %q, the preview said %q", got, want)
	}
}

// Tags, a done mark, strips and the locked pages follow a partial
// strategy (a range with a number kept), and Undo.
func TestRenumberStrategyFollows(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, followScript)
	w.lockSceneNumbers()
	replaceText(t, w, "EXT. STREET", "INT. HALL - NIGHT #1A#\n\nCoats.\n\nEXT. STREET")
	replaceText(t, w, "INT. KITCHEN - NIGHT #3#", "INT. KITCHEN - NIGHT #7#")
	tagNow(t, w, "revolver", 1, "Props")
	w.markSceneDone(3)
	sb := board(t, w)
	w.addShootDay()
	sb.selectScene(3, false)
	sb.moveSelected(0, 0)
	check := func(what, num string) {
		t.Helper()
		if got := tagLine(w, "revolver"); got != editorLineWith(t, w, "The revolver is gone") {
			t.Errorf("%s: the tag on line %d", what, got)
		}
		f, bt, _ := w.scheduleFound()
		if s := f.Days[0].Strips[0]; s.Scene != num {
			t.Errorf("%s: the strip's number %q, not %q", what, s.Scene, num)
		}
		bf := w.breakdownNow()
		if bf.Tags[0].Scene != num || !bf.isDone(bt.scenes[3]) {
			t.Errorf("%s: tag %+v, done %+v", what, bf.Tags[0], bf.Done)
		}
	}
	check("before", "7")
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	w.renumberWith(renumberChoice{Scope: scopeRange, From: 1, To: 3, List: listKeep, Numbers: "2"}, w.renumberContextNow(text, heads))
	if got := headNums(w); got != "1 1A 2 3" {
		t.Fatalf("renumbered %q", got)
	}
	check("renumbered", "3")
	w.undo()
	check("undo", "7")
	w.redo()
	check("redo", "3")
}

// The locked pages' scenes follow a strategy from the cursor on.
func TestRenumberStrategyLockedPages(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, threeLongScenes())
	w.lockSceneNumbers()
	replaceText(t, w, "#2#", "#1A#")
	replaceText(t, w, "#3#", "#2#")
	w.lockPages()
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	scenes := func() string {
		var s []string
		for _, p := range pageLockOf(w.textEditor.Text()).lock.Pages {
			s = append(s, p.Scene)
		}
		return strings.Join(s, " ")
	}
	before := scenes()
	w.textEditor.SetCursorOffset(offsetOf(t, w, "EXT. GARDEN"))
	text := w.textEditor.Text()
	w.renumberWith(renumberChoice{Scope: scopeCursor}, w.renumberContextNow(text, sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())))
	want := strings.NewReplacer("1A", "2", "2", "3").Replace(before)
	if got := scenes(); got != want {
		t.Errorf("pages' scenes %q, want %q (were %q)", got, want, before)
	}
	w.undo()
	if got := scenes(); got != before {
		t.Errorf("undo: %q", got)
	}
}

// SHOT_DIR renders the dialog, wide and on a phone, after an issue (its
// warning and both buttons).
func TestRenumberDialogScreenshots(t *testing.T) {
	holdUI(t)
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, strategyScript())
	for _, issued := range []bool{false, true} {
		if issued {
			issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
			replaceText(t, w, "INT. C - DAY #3#\n\nSee.\n\n", "INT. N - NIGHT\n\nNew.\n\n")
		}
		for _, size := range []struct {
			name string
			size fyne.Size
		}{{"wide", fyne.NewSize(1100, 700)}, {"phone", fyne.NewSize(380, 800)}} {
			clearOverlays(w)
			w.fyneWindow.Resize(size.size)
			w.renumberScenes()
			if !issued {
				w.renumbering.omitted.SetSelected(omittedOptions[lex.OmittedMerge])
			}
			name := "renumber-" + size.name
			if issued {
				name += "-issued"
			}
			capture(t, w.fyneWindow.Canvas(), name+".png")
		}
	}
}

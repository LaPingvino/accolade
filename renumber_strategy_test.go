// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"os/exec"
	"regexp"
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
	rows, _ := previewRows(heads, p, nil)
	var out []string
	for i, r := range rows {
		if p.drop[i] {
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
		{"OMITTED kept", nil, renumberChoice{Omitted: omitKeep}, "1 2 3 O4 O5 6 7 8"},
		{"OMITTED kept, printed as one: still one heading a scene", nil, renumberChoice{Omitted: omitPrintRuns}, "1 2 3 O4 O5 6 7 8"},
		{"keep 9", nil, renumberChoice{List: listKeep, Numbers: "9"}, "1 2 3 4 5 9"},
		{"keep 1A and 9", nil, renumberChoice{List: listKeep, Numbers: "1A, 9"}, "1 1A 2 3 4 9"},
		{"keep a range", nil, renumberChoice{List: listKeep, Numbers: "6-9"}, "1 2 3 6 6A 9"},
		{"only", nil, renumberChoice{List: listOnly, Numbers: "6A, 9"}, "1 1A 3 O4 O5 6 7 8"},
		{"range, OMITTED kept", nil, renumberChoice{Scope: scopeRange, From: 2, To: 6, Omitted: omitKeep}, "1 1A 2 O4 O5 6 7 9"},
		{"keep a range by the script's order", nil, renumberChoice{List: listKeep, Numbers: "1A .. 4"}, "1 1A 3 O4 5 6 7"},
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
	// the wrong way round: an error, nothing swapped, nothing written
	text := w.textEditor.Text()
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	for _, c := range []struct {
		c    renumberChoice
		want string
	}{
		{renumberChoice{Scope: scopeRange, From: 6, To: 2}, "From 6A to 3: scene 3 comes before scene 6A in the script"},
		{renumberChoice{List: listKeep, Numbers: "9-6"}, "9-6: scene 6 comes before scene 9 in the script"},
		{renumberChoice{List: listOnly, Numbers: "1, 4-99"}, "4-99: no scene 99"},
	} {
		if got := strings.Join(c.c.problems(heads), "; "); got != c.want {
			t.Errorf("%+v: problems %q, want %q", c.c, got, c.want)
		}
		if msg := w.renumberWith(c.c, w.renumberContextNow(text, heads)); !strings.HasPrefix(msg, "Not renumbered: ") || w.textEditor.Text() != script {
			t.Errorf("%+v applied: %q", c.c, msg)
		}
	}
}

// Labels out of Lexington's scheme (Fountain's I-1-A, 1.1, PROLOGUE …)
// stay through lock, every strategy and Unlock's Remove; the scheme's
// numbers are given around them, never theirs.
func TestFixedLabelsStay(t *testing.T) {
	holdUI(t)
	o, _ := fountain.OmitScene("INT. OLD - DAY #1-A#\n\nGone.\n\n")
	script := "INT. A - DAY #I-1-A#\n\nAy.\n\nINT. B - DAY\n\nBee.\n\nINT. C - DAY #1.#\n\nSee.\n\n" + o +
		"INT. D - DAY #110A#\n\nDee.\n\nINT. E - DAY #PROLOGUE#\n\nEe.\n\nINT. F - DAY\n\nEff.\n"
	w, _ := scriptWindow(t, script)
	fixed := "I-1-A 1. O1-A PROLOGUE"
	only := func() string {
		var out []string
		for _, n := range strings.Fields(headNums(w)) {
			if !lex.InNumberScheme(strings.TrimPrefix(n, "O")) {
				out = append(out, n)
			}
		}
		return strings.Join(out, " ")
	}
	w.lockSceneNumbers()
	// B has no scheme number before it: before 110A, as before a first
	if got := headNums(w); got != "I-1-A A110A 1. O1-A 110A PROLOGUE 111" {
		t.Errorf("locked: %q", got)
	}
	for _, c := range []renumberChoice{cleanChoice(), {KeepLetters: true}, {Omitted: omitKeep}, {Omitted: omitPrintRuns},
		{List: listKeep, Numbers: "110A"}, {List: listOnly, Numbers: "I-1-A – PROLOGUE"}, {Scope: scopeRange, From: 0, To: 5}} {
		text := w.textEditor.Text()
		heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
		w.renumberWith(c, w.renumberContextNow(text, heads))
		if got := only(); got != fixed {
			t.Errorf("%+v: fixed labels %q → %q (%s)", c, fixed, got, headNums(w))
		}
		w.undo()
	}
	w.renumberNow()
	if got := headNums(w); got != "I-1-A 1 1. O1-A 2 PROLOGUE 3" {
		t.Errorf("clean: %q", got)
	}
	w.unlockNow(true)
	if got := only(); got != fixed {
		t.Errorf("unlock, remove: %q (%s)", got, headNums(w))
	}
	w.undo()
	// the page lock names fixed labels as any other, and reads them back
	w.lockPages()
	waitFor(t, "the pages locked", func() bool { return pageLockOf(w.textEditor.Text()) != nil })
	l := pageLockOf(w.textEditor.Text())
	if len(l.lock.Pages) == 0 || l.lock.Pages[0].Scene != "I-1-A" {
		t.Errorf("locked pages %+v", l.lock.Pages)
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

// ".OMITTED #4-5#" is one scene labelled "4-5", a fixed label: a new
// scene after 3 in a locked script gets 4.
func TestOmittedDashedLabelIsFixed(t *testing.T) {
	heads := []sceneHead{{num: "3", text: "INT. C - DAY #3#"}, {num: "4-5", text: ".OMITTED #4-5#"}, {}}
	if got := sceneNumbersFor(heads, lex.NumberOptions{Locked: true}); strings.Join(got, " ") != "3 4-5 4" {
		t.Errorf("numbers %q", got)
	}
}

// "Print runs as one": the dialog's preview says how the run prints,
// Apply writes the setting (and nothing else when no number changes),
// and the print and the PDF follow it; FDX keeps one heading a number.
func TestPrintOmittedRunsAsOne(t *testing.T) {
	holdUI(t)
	script := strategyScript()
	w, _ := scriptWindow(t, script)
	w.renumberScenes()
	r := w.renumbering
	r.omitted.SetSelected(omittedOptions[omitPrintRuns])
	r.letters.SetSelected(letterOptions[1])
	r.list.SetSelected(listOptions[listOnly])
	r.numbers.SetText("9")
	var printed []string
	for _, row := range r.rows {
		if strings.HasSuffix(row.heading, "OMITTED · prints as 4-5") {
			printed = append(printed, row.old)
		}
	}
	if strings.Join(printed, " ") != "4 5" || !strings.Contains(r.summary.Text, "Printed as one: 4-5 OMITTED") ||
		!strings.Contains(r.summary.Text, "OMITTED runs will print as one") {
		t.Errorf("preview: %v, summary %q", printed, r.summary.Text)
	}
	if r.apply.Disabled() {
		t.Error("Apply disabled: the setting changes")
	}
	// the list read under the field
	r.numbers.SetText("6 .. 9, 99")
	if !r.read.Visible() || r.read.Text != "Read as: 6 to 9 (3 scenes) · 99 (no such scene)\nNo scene has 99 (nothing to renumber)" {
		t.Errorf("read: %q", r.read.Text)
	}
	r.numbers.SetText("9-6")
	if !r.bad.Visible() || r.bad.Text != "9-6: scene 6 comes before scene 9 in the script" || !r.apply.Disabled() {
		t.Errorf("a range backwards: %q, apply disabled %v", r.bad.Text, r.apply.Disabled())
	}
	r.numbers.SetText("9")
	r.scope.SetSelected(scopeOptions[scopeRange])
	r.from.SetSelectedIndex(6)
	r.to.SetSelectedIndex(1)
	if !r.bad.Visible() || !strings.Contains(r.bad.Text, "scene 1A comes before scene 6A") || !r.apply.Disabled() {
		t.Errorf("From … To backwards: %q", r.bad.Text)
	}
	r.scope.SetSelected(scopeOptions[scopeAll])
	test.Tap(overlayButton(t, w, "Renumber"))
	text := w.textEditor.Text()
	if !omittedRunsPrinted(text) || strings.Count(text, ".OMITTED #") != 2 {
		t.Fatalf("applied:\n%s", text)
	}
	// the print, the PDF and the FDX
	dir := t.TempDir()
	job := w.layoutJob()
	if err := exportAs(text, dir+"/s.html", "html", job); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dir + "/s.html"); strings.Count(string(b), "OMITTED") != 1 || !strings.Contains(string(b), "4-5") {
		t.Errorf("the print has %d OMITTED", strings.Count(string(b), "OMITTED"))
	}
	if err := exportAs(text, dir+"/s.fdx", "fdx", job); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dir + "/s.fdx"); !strings.Contains(string(b), `Number="4"`) || !strings.Contains(string(b), `Number="5"`) {
		t.Errorf("the FDX is not one heading a number")
	}
	if _, err := exec.LookPath("pdftotext"); err == nil {
		if err := exportAs(text, dir+"/s.pdf", "pdf", job); err != nil {
			t.Fatal(err)
		}
		out, _ := exec.Command("pdftotext", "-enc", "UTF-8", "-layout", dir+"/s.pdf", "-").Output()
		if !regexp.MustCompile(`(?m)^\s*4-5\s+OMITTED\s+4-5\s*$`).Match(out) {
			t.Errorf("the PDF:\n%s", out)
		}
	}
	// keeping them one a heading takes the setting out
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	w.renumberWith(renumberChoice{List: listOnly, Numbers: "9", Omitted: omitKeep}, w.renumberContextNow(text, heads))
	if omittedRunsPrinted(w.textEditor.Text()) {
		t.Error("keep: the setting stayed")
	}
	w.undo()
	w.undo()
	if w.textEditor.Text() != script {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	w.settings.SetString(renumberLastPref, "") // no Last used for the tests after
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
	if r.preset.Selected != presetSent || r.numbers.Text != "1-1A, 4-9, 3" {
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
				w.renumbering.omitted.SetSelected(omittedOptions[omitPrintRuns])
				w.renumbering.list.SetSelected(listOptions[listKeep])
				w.renumbering.numbers.SetText("1A .. 4, 99")
			}
			name := "renumber-" + size.name
			if issued {
				name += "-issued"
			}
			capture(t, w.fyneWindow.Canvas(), name+".png")
		}
	}
}

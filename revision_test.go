package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The checks' findings: the writing's heuristics (not the lock's) and the
// linter's, each on its line, a title page above them counted right.
func TestCheckItems(t *testing.T) {
	text := "Title: Rain\nAuthor: Me\n\nINT. KITCHEN\n\nAnna waits.\n\n@\nHello.\n"
	items := checkItemsOf(text, sceneStarts("en"))
	want := map[int]string{3: checkTime, 7: checkLint}
	got := map[int]string{}
	for _, it := range items {
		got[it.Line] = it.Check
		if it.Kind != openCheck || it.Shape != shapeFinding || !it.still(text) {
			t.Errorf("%+v: not a finding on its line", it)
		}
	}
	for l, k := range want {
		if got[l] != k {
			t.Errorf("line %d: %q, want %q (all: %v)", l, got[l], k, got)
		}
	}
	for _, it := range items {
		if it.Check == checkNumber || it.Check == checkTitle {
			t.Errorf("a lock's check: %+v", it)
		}
	}
}

// Keep writes a finding at its line's end as a revision note; the note
// is then the item, the finding gone; a heading keeps reading as one.
func TestKeepFinding(t *testing.T) {
	text := "INT. KITCHEN\n\nAnna waits.\n"
	en := sceneStarts("en")
	items := checkItemsOf(text, en)
	if len(items) != 1 || items[0].Check != checkTime {
		t.Fatalf("findings: %+v", items)
	}
	kept := items[0].keepEdit().apply(text)
	if want := "INT. KITCHEN [[revision: " + items[0].Text + "]]\n\nAnna waits.\n"; kept != want {
		t.Fatalf("kept: %q", kept)
	}
	if f := checkItemsOf(kept, en); len(f) != 0 {
		t.Errorf("the finding still there beside its note: %+v", f)
	}
	notes := openItemsOf(kept, en, false)
	if len(notes) != 1 || notes[0].Kind != openRevision || notes[0].Text != items[0].Text || !notes[0].open() {
		t.Errorf("the revision note: %+v", notes)
	}
	if o := outlineOf(kept, en); len(o) != 1 || !o[0].Scene {
		t.Errorf("the heading with its note: %+v", o)
	}
}

// Done to-dos: [[done: …]] and [[todo: [x] …]]; a done one is not open,
// and Done turns one into the other.
func TestDoneTodos(t *testing.T) {
	for in, want := range map[string]bool{"todo: x": false, "done: x": true, "todo: [x] x": true, "TODO [X] x": true, "done with it": false} {
		_, rest, done, ok := todoNote(in)
		if want && (!ok || !done || rest != "x") || !want && done {
			t.Errorf("%q: %q done %v ok %v", in, rest, done, ok)
		}
	}
	text := "INT. A - DAY\n\nHi. [[todo: call Bram]]\n"
	it := openItemsOf(text, nil, false)[0]
	done := it.doneEdit(false).apply(text)
	if done != "INT. A - DAY\n\nHi. [[done: call Bram]]\n" {
		t.Fatalf("done: %q", done)
	}
	d := openItemsOf(done, nil, false)[0]
	if d.Kind != openTodo || !d.Done || d.open() {
		t.Errorf("a done to-do: %+v", d)
	}
	if again := d.doneEdit(false).apply(done); again != text {
		t.Errorf("to do again: %q", again)
	}
}

// The findings go among the notes by their place, the unsorted notes last.
func TestMergeFindings(t *testing.T) {
	items := []openItem{{Kind: openNote, Start: 5}, {Kind: openNote, Start: 20}, {Kind: openUnsorted, Start: 90}}
	found := []openItem{{Kind: openCheck, Start: 0}, {Kind: openCheck, Start: 5}, {Kind: openCheck, Start: 50}}
	var got []int
	for _, it := range mergeFindings(items, found) {
		got = append(got, it.Start)
	}
	if want := []int{0, 5, 5, 20, 50, 90}; !slices.Equal(got, want) {
		t.Errorf("merged: %v, want %v", got, want)
	}
}

// The Revision tab steps through the items shown from the cursor, round
// at the ends; the kinds hidden are skipped; Keep, Dismiss and the bulk
// actions are each one undo step.
func TestRevisionStepping(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dismissInDocFor(t, w) // Dismiss in the notes block, one undo step
	text := "INT. A - DAY\n\nHi. [[todo: call Bram]]\n\nINT. B\n\nBye. [[done: write it]] [[done: and this]]\n\n/* INT. C - NIGHT\n\nGone.\n*/\n"
	w.textEditor.SetText(text)
	w.textEditor.SetCursorOffset(0)
	var kinds []openKind
	for range 5 {
		w.stepOpen(1)
		items := w.shownOpenItems()
		kinds = append(kinds, items[currentOpen(items, w.textEditor.CursorOffset())].Kind)
	}
	if want := []openKind{openTodo, openCheck, openTodo, openTodo, openCut}; !slices.Equal(kinds, want) {
		t.Fatalf("stepped: %v, want %v", kinds, want)
	}
	w.stepOpen(1) // round to the first
	if items := w.shownOpenItems(); items[currentOpen(items, w.textEditor.CursorOffset())].Kind != openTodo {
		t.Error("not round to the first")
	}
	w.stepOpen(-1)
	if items := w.shownOpenItems(); items[currentOpen(items, w.textEditor.CursorOffset())].Kind != openCut {
		t.Error("Previous not round to the last")
	}
	// the filter: the to-dos hidden
	w.toggleOpenKind(0)
	w.textEditor.SetCursorOffset(0)
	w.stepOpen(1)
	if items := w.shownOpenItems(); items[currentOpen(items, w.textEditor.CursorOffset())].Kind != openCheck {
		t.Error("a hidden kind stepped to")
	}
	w.toggleOpenKind(0)
	// Keep the finding
	if e := w.cmdEnv(); !e.rev.finding {
		t.Fatal("the finding at the cursor not seen")
	}
	if it, ok := w.currentItem("Keep"); ok {
		w.keepFinding(it)
	}
	if got := w.textEditor.Text(); !strings.Contains(got, "INT. B [[revision: No time of day") {
		t.Fatalf("kept: %q", got)
	}
	w.undo()
	if w.textEditor.Text() != text {
		t.Fatalf("Keep not one undo step: %q", w.textEditor.Text())
	}
	// Dismiss it: a line in the notes block, out of sight; one undo step
	w.textEditor.SetCursorOffset(strings.Index(text, "INT. B"))
	w.dismissCurrent()
	if got := w.textEditor.Text(); !strings.HasPrefix(got, text) || !strings.Contains(got, "/* Notes\n"+dismissedHead+"\n- No time of day") {
		t.Errorf("Dismiss wrote: %q", got)
	}
	for _, it := range w.openItems() {
		if it.Kind == openCheck {
			t.Error("a dismissed finding still listed")
		}
	}
	w.undo()
	if w.textEditor.Text() != text {
		t.Fatalf("Dismiss not one undo step: %q", w.textEditor.Text())
	}
	// Remove Done To-dos, one undo step
	if e := w.cmdEnv(); e.rev.doneTodos != 2 {
		t.Fatalf("done to-dos: %d", e.rev.doneTodos)
	}
	var done []textEdit
	for _, it := range w.openItems() {
		if it.Kind == openTodo && it.Done {
			done = append(done, it.removal(text))
		}
	}
	w.applyTextEdits(done)
	if got := w.textEditor.Text(); !strings.Contains(got, "\n\nBye.\n\n") || strings.Contains(got, "done:") {
		t.Errorf("done removed: %q", got)
	}
	w.undo()
	if w.textEditor.Text() != text {
		t.Errorf("not one undo step: %q", w.textEditor.Text())
	}
}

// Restore Cuts in This Scene: the cuts under the cursor's heading, not
// another scene's.
func TestSceneCuts(t *testing.T) {
	text := "INT. A - DAY\n\n/* INT. X - DAY\n\nOne.\n*/\n\n/* INT. Y - DAY\n\nTwo.\n*/\n\nINT. B - DAY\n\n/* INT. Z - DAY\n\nThree.\n*/\n"
	items := openItemsOf(text, nil, false)
	if got := sceneCuts(items, text, 3, nil, false); len(got) != 2 {
		t.Errorf("A's cuts: %+v", got)
	}
	if got := sceneCuts(items, text, len([]rune(text))-2, nil, false); len(got) != 1 || got[0].Title != "INT. Z - DAY" {
		t.Errorf("B's cuts: %+v", got)
	}
}

// Where the beats land: each section on the page its next printed line
// is on, its place over the script's pages (the title page not counted),
// a structure's ranges matched by the sections' names.
func TestBeatPlaces(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	var b strings.Builder
	b.WriteString("Title: X\n\n# Act One\n\nINT. A - DAY\n\n")
	for range 120 {
		b.WriteString("Anna waits and waits, a long line of action here.\n\n")
	}
	b.WriteString("# Midpoint\n\nINT. B - DAY\n\nHi.\n\n# Act Three\n\nINT. C - DAY\n\nBye.\n")
	text := b.String()
	j := w.outputJob(w.pageSetup(), "pdf", text, nil)
	d, err := drawer(text, j)(nil)
	if err != nil {
		t.Fatal(err)
	}
	places, structure := placesOf(text, sceneStarts("en"), d)
	if structure != "Three Acts" || len(places) != 3 {
		t.Fatalf("%q: %+v", structure, places)
	}
	if p := places[0]; p.page != 1 || p.pct != 0 || p.has {
		t.Errorf("Act One: %+v", p)
	}
	mid := places[1]
	if mid.page != mid.pages || !mid.has || mid.lo != 45 || !mid.off() || !strings.Contains(mid.advice(), "late") {
		t.Errorf("the Midpoint, on the last page: %+v %s", mid, mid.advice())
	}
	// the window's: the layout in the background, then the readout
	w.textEditor.SetText(text)
	w.goToLine(strings.Count(text[:strings.Index(text, "# Midpoint")], "\n"))
	waitFor(t, "the layout", func() bool { _, _, ok := w.beatPlaces(); return ok })
	if got := w.beatsReadout(); !strings.HasPrefix(got, "Midpoint · p. ") || !strings.HasSuffix(got, "⚑") {
		t.Errorf("readout %q", got)
	}
	w.goToLine(0)
	w.stepOff(1)
	if w.scriptCursorLine() != mid.line {
		t.Errorf("Next Off went to line %d", w.scriptCursorLine())
	}
}

// TestRevisionTabScreenshots renders the Revision tab (SHOT_DIR): on a
// finding with the Notes pane beside, the beats' readout, and on a phone.
func TestRevisionTabScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w := newTestWindow(t, "")
	keepDockSettings(t, w)
	r := ribbonOnFor(t, w)
	var b strings.Builder
	b.WriteString("Title: The Long Rain\n\n# Act One\n\n= Anna waits for a letter.\n\nINT. KITCHEN\n\nAnna stands at the window. [[todo: why does she not go out?]]\n\n")
	for range 40 {
		b.WriteString("The rain goes on, and Anna watches the road for the postman.\n\n")
	}
	b.WriteString("# Act Two\n\n= Bram brings the letter himself.\n\nEXT. ROAD - DAY\n\nBram drives. [[done: name the car]]\n\n/* INT. GARDEN - NIGHT\n\nANNA\nGoodbye, then.\n*/\n\n")
	for range 30 {
		b.WriteString("The wipers beat, the road shines, Bram does not slow down.\n\n")
	}
	b.WriteString("# Midpoint\n\nINT. HALL - NIGHT\n\nThe letter on the mat.\n\n# Act Three\n\nEXT. LAKE - DAWN\n\nThey read it together.\n")
	w.textEditor.SetText(b.String())
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	r.selectTab("Revision")
	w.togglePane("notes")
	w.textEditor.SetCursorOffset(0)
	w.stepOpen(1) // the finding: INT. KITCHEN has no time
	waitFor(t, "the layout", func() bool { _, _, ok := w.beatPlaces(); return ok })
	w.ribbonRefresh()
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		capture(t, w.fyneWindow.Canvas(), "revision-"+name+".png")
	}
	snap("1-finding")
	w.goToLine(strings.Count(b.String()[:strings.Index(b.String(), "# Midpoint")], "\n"))
	w.ribbonRefresh()
	snap("2-beats")
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.ribbonRefresh()
	snap("3-phone")
}

// F8 through the window's keys steps; Restore Cuts in This Scene brings
// an omitted scene back with its number and a plain cut, one undo step.
func TestRevisionKeysAndRestoreCuts(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	text := "INT. A - DAY #1#\n\n/* INT. X - DAY\n\nOne.\n*/\n\n.OMITTED #2#\n\n/* Omitted 2\nINT. CELLAR - DAY #2#\n\nDark.\n*/\n\nINT. B - DAY #3#\n\nHi. [[todo: x]]\n"
	w.textEditor.SetText(text)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(0)
	if !w.keyPreview(fyne.KeyF8, 0) {
		t.Fatal("F8 not taken")
	}
	items := w.shownOpenItems()
	if i := currentOpen(items, w.textEditor.CursorOffset()); i < 0 || items[i].Kind != openCut || items[i].Title != "INT. X - DAY" {
		t.Fatalf("F8 went to %d", w.textEditor.CursorOffset())
	}
	w.textEditor.SetCursorOffset(3)
	cuts := sceneCuts(w.openItems(), text, 3, w.sceneStarts(), false)
	if len(cuts) != 1 { // the omitted scene has its own heading: not A's
		t.Fatalf("A's cuts: %+v", cuts)
	}
	at := strings.Index(text, ".OMITTED")
	cuts = sceneCuts(w.openItems(), text, at+2, w.sceneStarts(), false)
	if len(cuts) != 1 || !cuts[0].Kept {
		t.Fatalf("the omitted scene's: %+v", cuts)
	}
	var edits []textEdit
	for _, c := range append(sceneCuts(w.openItems(), text, 3, w.sceneStarts(), false), cuts...) {
		e, ok := c.restore(text, false)
		if !ok {
			t.Fatalf("restore %+v", c)
		}
		edits = append(edits, e)
	}
	w.applyTextEdits(edits)
	got := w.textEditor.Text()
	if strings.Contains(got, "/*") || !strings.Contains(got, "INT. CELLAR - DAY #2#") || !strings.Contains(got, "INT. X - DAY") || strings.Contains(got, "OMITTED") {
		t.Errorf("restored: %q", got)
	}
	w.undo()
	if w.textEditor.Text() != text {
		t.Errorf("not one undo step: %q", w.textEditor.Text())
	}
}

// Keep is offered only where a note on the line leaves the line as it is.
func TestKeepOnlyWhereTheLineStays(t *testing.T) {
	en := sceneStarts("en")
	for _, s := range []string{"INT. A - DAY #12#\n\nHi.\n\nINT. B #13#\n\nHo.\n", "INT. A - DAY\n\n@\nHi.\n",
		"INT. A - DAY\n\nMRS. BENNET\nHi.\n\nMRS BENNET\nHo.\n\nMRS. BENNET\nAgain.\n", "INT. A - DAY\n\nHi. ]]\n\nHo.\n"} {
		for _, f := range checkItemsOf(s, en) {
			if !f.keepable() {
				continue
			}
			k := f.keepEdit().apply(s)
			if a, b := fmt.Sprint(syntax.ClassifyIn(strings.Split(s, "\n"), en, dialectForStarts(en))), fmt.Sprint(syntax.ClassifyIn(strings.Split(k, "\n"), en, dialectForStarts(en))); a != b {
				t.Errorf("%s: Keep changed the lines' kinds", f.Text)
			}
			for _, g := range checkItemsOf(k, en) {
				if g.Text == f.Text {
					t.Errorf("%s: still found after Keep", f.Text)
				}
			}
		}
	}
	if (openItem{Kind: openCheck, Check: checkCue}).keepable() {
		t.Error("a cue's finding keepable")
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
	"github.com/LaPingvino/lexington/fountain"
)

// runCmd runs the registry's command id as its menu item does.
func runCmd(w *MainWindow, id string) { w.do(commands().byID[id])() }

// mergeScript has what an outside copy may not carry: an outline, a
// source link, locked numbers, a note, boneyard and Accolade's blocks.
const mergeScript = `Title: The Barn
Author: Me

# Act One

= The storm comes.

INT. BARN - DAY #1#

Rain hammers the roof. [[from: "The rain kept falling."]]

[[a note of mine]]

JOHN
(beat)
It's coming.

/* an old line */

EXT. FIELD - NIGHT #2#

Mud. *Lots* of it.

MARY (V.O.)
Run!

> THE END <

/* Notes
keep
*/
`

// mergeTheirs merges the copy at path into w's script, from base.
func mergeTheirs(t *testing.T, w *MainWindow, path, base string) *scriptdiff.Merge {
	t.Helper()
	oc, err := w.readOutside(path)
	if err != nil {
		t.Fatal(err)
	}
	return scriptdiff.NewMerge(base, w.textEditor.Text(), oc.text, w.mergeOptions(oc.text, oc.lossy))
}

// An FDX made from the script and back is no change: what Final Draft
// cannot carry is not theirs to cut. Their edits are, and taking them
// keeps the link, the locked numbers, the outline and the blocks.
func TestMergeFDXRoundTrip(t *testing.T) {
	w, _ := scriptWindow(t, mergeScript)
	dir := t.TempDir()
	same := filepath.Join(dir, "same.fdx")
	if err := exportAs(mergeScript, same, "fdx", exportJob{}); err != nil {
		t.Fatal(err)
	}
	if m := mergeTheirs(t, w, same, mergeScript); len(m.Changes) != 0 {
		t.Errorf("the round trip is %d changes: %+v", len(m.Changes), m.Changes)
	}
	edited := strings.Replace(strings.Replace(mergeScript, "Rain hammers the roof.", "Rain drums on the roof.", 1),
		"INT. BARN - DAY #1#", "INT. BARN - NIGHT", 1)
	p := filepath.Join(dir, "theirs.fdx")
	if err := exportAs(edited, p, "fdx", exportJob{}); err != nil {
		t.Fatal(err)
	}
	m := mergeTheirs(t, w, p, mergeScript)
	if len(m.Changes) != 2 {
		t.Fatalf("%d changes: %+v", len(m.Changes), m.Changes)
	}
	got, _, _ := m.Apply(map[int]bool{0: true, 1: true}, nil)
	want := strings.Replace(strings.Replace(mergeScript, "Rain hammers the roof.", "Rain drums on the roof.", 1),
		"INT. BARN - DAY #1#", "INT. BARN - NIGHT #1#", 1)
	if got != want {
		t.Errorf("taken:\n%s\nwant:\n%s", got, want)
	}
}

// A Clean Fountain has no links or blocks: no change for those either.
func TestMergeCleanFountain(t *testing.T) {
	w, _ := scriptWindow(t, mergeScript)
	clean := w.cleanText(mergeScript, fountain.StripOptions{})
	p := filepath.Join(t.TempDir(), "clean.fountain")
	os.WriteFile(p, []byte(strings.Replace(clean, "Run!", "Run, John!", 1)), 0o644)
	m := mergeTheirs(t, w, p, mergeScript)
	if len(m.Changes) != 1 || m.Changes[0].Theirs != "MARY (V.O.)\nRun, John!" {
		t.Fatalf("changes %+v", m.Changes)
	}
	if got, _, _ := m.Apply(map[int]bool{0: true}, nil); got != strings.Replace(mergeScript, "Run!", "Run, John!", 1) {
		t.Errorf("taken:\n%s", got)
	}
}

// File > Merge Changes…: the copy's base found among the drafts, its
// changes in Compare, only what is taken merged, as one undo step and a
// new draft, the script before kept as a draft first; the tags follow.
func TestMergeChanges(t *testing.T) {
	holdUI(t)
	const script = "INT. KITCHEN - NIGHT\n\nAnna loads the revolver.\n\nINT. HALL - NIGHT\n\nBen hides the knife.\n\nEXT. STREET - DAY\n\nRain.\n"
	w, _ := scriptWindow(t, script)
	saveDraftNow(t, w, "Sent to Anna")
	saveDraftNow(t, w, "Unrelated") // the same text: stored once
	w.textEditor.SetText(script + "\nINT. ATTIC - DAY\n\nDust.\n")
	saveDraftNow(t, w, "With the attic")
	ours := strings.Replace(w.textEditor.Text(), "Rain.", "Rain falls.", 1) // ours' own change since
	w.textEditor.SetText(ours)
	tagNow(t, w, "knife", 0, "Props")
	// theirs: the street before the hall, a line changed, the kitchen's
	// line changed
	theirs := "INT. KITCHEN - NIGHT\n\nAnna cleans the revolver.\n\nEXT. STREET - DAY\n\nRain.\n\nINT. HALL - NIGHT\n\nBen hides the knife.\n\nTOM\nWho's there?\n"
	p := filepath.Join(t.TempDir(), "anna.fountain")
	os.WriteFile(p, []byte(theirs), 0o644)
	was := chooseFileIn
	chooseFileIn = func(_ fyne.Window, _ []string, then func(string)) { then(p) }
	t.Cleanup(func() { chooseFileIn = was })
	runCmd(w, "file.mergeChanges")
	cv := w.compare
	if cv == nil || cv.merge == nil || w.mid.shown != "compare" {
		t.Fatal("no merge in front")
	}
	ds := draftsOf(t, w)
	if base := cv.idOf(cv.from.Selected); base != ds[0].ID && base != ds[1].ID {
		t.Errorf("base %q, not the draft sent", cv.from.Selected)
	}
	m := cv.merge.m
	var kinds []string
	for _, c := range m.Changes {
		kinds = append(kinds, changeWhat(c))
	}
	if len(m.Changes) != 3 {
		t.Fatalf("changes: %q", kinds)
	}
	if m.Kept != 1 {
		t.Errorf("kept %d", m.Kept)
	}
	if !strings.Contains(cv.summary.Text, "3 changes from anna.fountain") {
		t.Errorf("summary %q", cv.summary.Text)
	}
	if !cv.mergeBtn.Disabled() || w.cmdEnv().mergeTaken {
		t.Error("Merge on with nothing taken")
	}
	// Take the first (the kitchen's line) by the tab's command, leave the
	// move, take Tom's line by its row
	runCmd(w, "compare.take")
	if !cv.merge.take[cv.changeIDs[0]] || cv.at != 1 {
		t.Errorf("take: %v at %d", cv.merge.take, cv.at)
	}
	runCmd(w, "compare.leave")
	var tom int = -1
	for id, c := range m.Changes {
		if strings.Contains(c.Theirs, "Who's there?") {
			tom = id
		}
	}
	cv.choices[tom].SetSelected(choiceTake)
	if cv.mergeBtn.Disabled() || !strings.Contains(cv.mergeBtn.Text, "2 Taken") {
		t.Errorf("merge button %q", cv.mergeBtn.Text)
	}
	before := len(draftsOf(t, w))
	runCmd(w, "compare.merge")
	want := strings.Replace(strings.Replace(ours, "Anna loads", "Anna cleans", 1), "Ben hides the knife.\n", "Ben hides the knife.\n\nTOM\nWho's there?\n", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
	ds = draftsOf(t, w)
	if len(ds) != before+2 || ds[len(ds)-2].Name != "Before merging anna.fountain" || ds[len(ds)-1].Name != "Merged from anna.fountain" ||
		ds[len(ds)-1].MergedFrom != "anna.fountain" || ds[len(ds)-1].Base == "" {
		t.Errorf("drafts %+v", ds[before:])
	}
	if got, want := tagLine(w, "knife"), editorLineWith(t, w, "Ben hides the knife."); got != want {
		t.Errorf("the knife's tag on line %d, not %d", got, want)
	}
	// what was left is offered still
	if cv.merge.m == nil || len(cv.merge.m.Changes) != 1 || cv.merge.m.Changes[0].Kind != scriptdiff.SceneMoved {
		t.Errorf("left: %+v", cv.merge.m)
	}
	w.undo()
	if got := w.textEditor.Text(); got != ours {
		t.Errorf("one undo:\n%s", got)
	}
}

// Merging into a script changed since it was compared is refused.
func TestMergeRefusesChangedScript(t *testing.T) {
	const script = "INT. KITCHEN - NIGHT\n\nAnna loads the revolver.\n"
	w, _ := scriptWindow(t, script)
	p := filepath.Join(t.TempDir(), "x.fountain")
	os.WriteFile(p, []byte(strings.Replace(script, "loads", "cleans", 1)), 0o644)
	oc, _ := w.readOutside(p)
	cv := showMerge(w, nil, nowID, oc)
	cv.choices[0].SetSelected(choiceTake)
	w.textEditor.SetText(script + "\nMore.\n")
	var got error
	w.applyMerge(cv.merge, func(err error) { got = err })
	if got == nil || !strings.Contains(got.Error(), "changed since") {
		t.Errorf("err %v", got)
	}
}

// A conflict shows both ways, and Edit writes the writer's own text.
func TestMergeConflictEdit(t *testing.T) {
	const script = "INT. KITCHEN - NIGHT\n\nAnna loads the revolver.\n"
	w, _ := scriptWindow(t, script)
	saveDraftNow(t, w, "Sent")
	w.textEditor.SetText(strings.Replace(script, "loads", "hides", 1))
	p := filepath.Join(t.TempDir(), "x.fountain")
	os.WriteFile(p, []byte(strings.Replace(script, "loads", "cleans", 1)), 0o644)
	oc, _ := w.readOutside(p)
	cv := showMerge(w, readDraftsNewest(t, w), draftsOf(t, w)[0].ID, oc)
	if len(cv.merge.m.Changes) != 1 || !cv.merge.m.Changes[0].Conflict {
		t.Fatalf("%+v", cv.merge.m.Changes)
	}
	r := cv.choices[0]
	if len(r.Options) != 3 {
		t.Errorf("options %v", r.Options)
	}
	r.SetSelected(choiceEdit)
	cv.merge.edits[0] = "Anna hides and cleans the revolver."
	var err error
	w.applyMerge(cv.merge, func(e error) { err = e })
	if err != nil || w.textEditor.Text() != "INT. KITCHEN - NIGHT\n\nAnna hides and cleans the revolver.\n" {
		t.Errorf("%v %q", err, w.textEditor.Text())
	}
}

func readDraftsNewest(t *testing.T, w *MainWindow) []savedDraft {
	ds := draftsOf(t, w)
	for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {
		ds[i], ds[j] = ds[j], ds[i]
	}
	return ds
}

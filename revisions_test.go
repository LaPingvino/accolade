package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdfin"
)

// The WGA's order (lex.RevisionName): White, Blue … Cherry, then Second
// Blue … Second Cherry, Third Blue.
func TestRevisionColourOrder(t *testing.T) {
	want := []string{"White", "Blue", "Pink", "Yellow", "Green", "Goldenrod", "Buff", "Salmon", "Cherry",
		"Second Blue", "Second Pink", "Second Yellow", "Second Green", "Second Goldenrod", "Second Buff",
		"Second Salmon", "Second Cherry", "Third Blue"}
	for n, name := range want {
		if got := lex.RevisionName(n); got != name {
			t.Errorf("set %d: %q, want %q", n, got, name)
		}
		if lex.RevisionIndex(name) != n {
			t.Errorf("index of %q: %d", name, lex.RevisionIndex(name))
		}
	}
	if c := lex.RevisionColor("Second Goldenrod"); c != "Goldenrod" {
		t.Errorf("colour %q", c)
	}
	c := &revisionChain{issues: []issue{{set: lex.RevisionSet{Name: "Cherry"}}}}
	if n := c.nextSet(time.Now()).Name; n != "Second Blue" {
		t.Errorf("after Cherry: %q", n)
	}
}

// carry: a line kept keeps its mark, a new or changed one the set's, the
// line after a deletion too; blank lines never.
func TestCarryMarks(t *testing.T) {
	old := []string{"INT. A - DAY", "", "Rain.", "", "BOB", "Hi.", "", "Gone.", "", "End."}
	marks := []string{"", "", "Blue", "", "", "", "", "", "", ""}
	new := []string{"INT. A - DAY", "", "Rain.", "", "BOB", "Hello.", "", "", "End."}
	got := carry(old, marks, new, "Pink")
	// "Gone." went next to a changed line: that line's mark is the place
	want := []string{"", "", "Blue", "", "", "Pink", "", "", ""}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("marks %q, want %q", got, want)
	}
	// a deletion alone marks the line after it
	new = []string{"INT. A - DAY", "", "Rain.", "", "BOB", "Hi.", "", "End."}
	got = carry(old, marks, new, "Pink")
	want = []string{"", "", "Blue", "", "", "", "", "Pink"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("deletion's marks %q, want %q", got, want)
	}
}

const revScript = "Title: Rain\n\nINT. HOUSE - DAY\n\nRain on the roof.\n\nANNA\nIt never stops.\n\nEXT. GARDEN - DAY\n\nMud everywhere.\n\nINT. HOUSE - NIGHT\n\nDry at last.\n"

// lineWith is the line of the editor's text holding s.
func editorLineWith(t *testing.T, w *MainWindow, s string) int {
	t.Helper()
	for i, l := range strings.Split(w.textEditor.Text(), "\n") {
		if strings.Contains(l, s) {
			return i
		}
	}
	t.Fatalf("no line %q in\n%s", s, w.textEditor.Text())
	return -1
}

// replaceText changes the first old in the editor to new.
func replaceText(t *testing.T, w *MainWindow, old, new string) {
	t.Helper()
	text := w.textEditor.Text()
	i := strings.Index(text, old)
	if i < 0 {
		t.Fatalf("no %q", old)
	}
	at := len([]rune(text[:i]))
	w.textEditor.Replace(at, at+len([]rune(old)), new)
}

func issueNow(t *testing.T, w *MainWindow, name string, day time.Time) {
	t.Helper()
	called := false
	w.issueDraft(lex.RevisionSet{Name: name, Date: day}, "", func(err error) {
		if err != nil {
			t.Fatal(err)
		}
		called = true
	})
	if !called {
		t.Fatal("not issued at once")
	}
}

// marked are the editor's marks, by the text of their lines.
func marked(w *MainWindow) map[string]string {
	w.updateRevisionMarks()
	out := map[string]string{}
	lines := strings.Split(w.textEditor.Text(), "\n")
	for i, m := range w.revisions.marks {
		if m != "" {
			out[strings.TrimSpace(lines[i])] = m
		}
	}
	return out
}

// Issue White (the lock), edit, issue Blue, edit: the marks in the last
// issue's and in all issues' colours.
func TestIssueEditMarks(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	if got := marked(w); len(got) != 0 {
		t.Fatalf("marks before any issue: %v", got)
	}
	white := time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local)
	issueNow(t, w, "White", white)
	text := w.textEditor.Text()
	if !sceneLockOf(text).locked || !strings.Contains(text, "INT. HOUSE - DAY #1#") {
		t.Fatalf("the first issue did not lock the numbers:\n%s", text)
	}
	if got := currentRevisionOf(text); got != "White, 2026-11-01" {
		t.Errorf("production block's revision %q", got)
	}
	ds := draftsOf(t, w)
	if len(ds) != 1 || ds[0].Kind != kindIssue || ds[0].Colour != "White" || ds[0].Issued != "2026-11-01" {
		t.Fatalf("store %+v", ds)
	}
	if got := marked(w); len(got) != 0 {
		t.Errorf("marks right after White: %v", got)
	}
	// an edit is Blue, the open set
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	if got := marked(w); len(got) != 1 || got["Mud everywhere, and a dog."] != "Blue" {
		t.Errorf("after an edit: %v", got)
	}
	blue := time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local)
	issueNow(t, w, "Blue", blue)
	if got := currentRevisionOf(w.textEditor.Text()); got != "Blue, 2026-11-20" {
		t.Errorf("revision line %q", got)
	}
	// just issued: its marks are Blue's (what its pages print)
	if got := marked(w); len(got) != 1 || got["Mud everywhere, and a dog."] != "Blue" {
		t.Errorf("right after Blue: %v", got)
	}
	// edit again: Pink; last issue shows Pink only, all issues both
	replaceText(t, w, "It never stops.", "It never ever stops.")
	if got := marked(w); len(got) != 1 || got["It never ever stops."] != "Pink" {
		t.Errorf("last issue's marks: %v", got)
	}
	w.settings.SetString(revisionMarksPref, "all")
	t.Cleanup(func() { w.settings.SetString(revisionMarksPref, "last") })
	got := marked(w)
	if len(got) != 2 || got["It never ever stops."] != "Pink" || got["Mud everywhere, and a dog."] != "Blue" {
		t.Errorf("all issues' marks: %v", got)
	}
	// the editor's margin has them, in their colours
	if ink := w.revisionMarksShown(); len(ink) != 2 || ink[editorLineWith(t, w, "ever stops")] != revisionInk("Pink") {
		t.Errorf("margin %v", ink)
	}
	// the export's sets: White, Blue and the open Pink, dated
	r, marks := w.exportRevisions(w.textEditor.Text(), true)
	if len(r.Sets) != 3 || r.Sets[1].Name != "Blue" || !r.Sets[1].Date.Equal(blue) || r.Sets[2].Name != "Pink" {
		t.Errorf("sets %+v", r.Sets)
	}
	if marks[editorLineWith(t, w, "ever stops")] != "Pink" {
		t.Error("export marks")
	}
	// reopened: the issues are found again
	w2 := againWindow(t, w)
	w.saveToFile(w.currentFile)
	if err := w2.LoadFile(w.currentFile); err != nil {
		t.Fatal(err)
	}
	if got := marked(w2); len(got) != 2 {
		t.Errorf("reopened: %v", got)
	}
}

// The menu has Issue Draft…, and its dialog names the next set.
func TestIssueDraftMenu(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	menuItemByLabel(t, w, "Issue Draft…")
	w.showIssueDraft()
	if !dialogShows(w, "Issue White") {
		t.Error("the dialog does not issue White")
	}
}

// dialogShows tells whether a text is on the window's overlays.
func dialogShows(w *MainWindow, s string) bool {
	found := false
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch x := o.(type) {
		case *widget.Label:
			found = found || strings.Contains(x.Text, s)
		case *canvas.Text:
			found = found || strings.Contains(x.Text, s)
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			r := test.WidgetRenderer(x)
			for _, c := range r.Objects() {
				walk(c)
			}
		}
	}
	for _, o := range w.fyneWindow.Canvas().Overlays().List() {
		walk(o)
	}
	return found
}

// The PDF: asterisks on the changed lines, the header on their pages,
// revised pages only, and the FDX out and back in as imported marks.
func TestRevisionExports(t *testing.T) {
	var b strings.Builder
	b.WriteString(revScript)
	for i := 1; i <= 40; i++ {
		b.WriteString("\nEXT. FIELD " + strings.Repeat("I", i%5+1) + " - DAY\n\nThe wind moves the grass, row after row, as far as the eye can see.\n")
	}
	w, _ := scriptWindow(t, b.String())
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Dry at last.", "Dry at last, and warm.")
	dir := t.TempDir()
	job := w.withRevisions(w.savedExportJob(), w.textEditor.Text(), false, false)
	full := filepath.Join(dir, "full.pdf")
	if err := exportAs(w.textEditor.Text(), full, "pdf", job); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(full)
	if err != nil {
		t.Fatal(err)
	}
	today := time.Now().Format("02/01/06")
	changed := 0
	for _, p := range pages {
		star, head := false, false
		for _, l := range p {
			star = star || strings.Contains(l.Text, "Dry at last, and warm.") && strings.HasSuffix(l.Text, "*")
			head = head || strings.Contains(l.Text, "Blue Revision – "+today)
		}
		if star != head {
			t.Errorf("a page with mark %v and header %v", star, head)
		}
		if star {
			changed++
		}
	}
	if changed != 1 || len(pages) < 3 {
		t.Fatalf("%d changed pages of %d", changed, len(pages))
	}
	only := filepath.Join(dir, "only.pdf")
	job = w.withRevisions(w.savedExportJob(), w.textEditor.Text(), true, false)
	if err := exportAs(w.textEditor.Text(), only, "pdf", job); err != nil {
		t.Fatal(err)
	}
	if p, err := pdfin.Pages(only); err != nil || len(p) != 1 {
		t.Errorf("revised pages only: %d pages (%v)", len(p), err)
	}

	// FDX out: the sets and the RevisionID (every issue's, whatever the
	// margin shows)
	fdxFile := filepath.Join(dir, "rev.fdx")
	job = w.withAllRevisions(w.savedExportJob(), w.textEditor.Text())
	if err := exportAs(w.textEditor.Text(), fdxFile, "fdx", job); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(fdxFile)
	if !bytes.Contains(data, []byte(`Name="Blue Rev. `)) || !bytes.Contains(data, []byte(`<Text RevisionID="1">Dry at last, and warm.</Text>`)) {
		t.Fatalf("FDX:\n%s", data)
	}
	// and in: an imported issue of Blue with its mark, the next edit Pink
	w2, _ := newDialogTestWindow(t)
	if err := w2.importFile(fdxFile); err != nil {
		t.Fatal(err)
	}
	ds := draftsOf(t, w2)
	if len(ds) != 1 || ds[0].Kind != kindIssue || ds[0].Colour != "Blue" {
		t.Fatalf("imported store %+v", ds)
	}
	if got := marked(w2); len(got) != 1 || got["Dry at last, and warm."] != "Blue" {
		t.Errorf("imported marks %v", got)
	}
	replaceText(t, w2, "Mud everywhere.", "Mud, mud.")
	if got := marked(w2); len(got) != 1 || got["Mud, mud."] != "Pink" {
		t.Errorf("after the import, last issue's: %v", got)
	}
	w2.settings.SetString(revisionMarksPref, "all")
	t.Cleanup(func() { w2.settings.SetString(revisionMarksPref, "last") })
	if got := marked(w2); len(got) != 2 || got["Dry at last, and warm."] != "Blue" {
		t.Errorf("after the import, all: %v", got)
	}
	if !strings.Contains(currentRevisionOf(w2.textEditor.Text()), "imported from rev.fdx") {
		t.Errorf("revision line %q", currentRevisionOf(w2.textEditor.Text()))
	}
}

// The editor with its margin's colours, at a desktop's width and a
// phone's (SHOT_DIR=… go test -run TestRevisionScreenshots).
func TestRevisionScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := scriptWindow(t, revScript)
	w.settings.SetString(revisionMarksPref, "all")
	t.Cleanup(func() { w.settings.SetString(revisionMarksPref, "last") })
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog digging in it.")
	issueNow(t, w, "Blue", time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "It never stops.", "It never stops. Not once.")
	marked(w)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	test.Type(w.textEditor, "")
	comboShot(t, w, "revisions")
	r, marks := w.exportRevisions(w.textEditor.Text(), true)
	job := w.savedExportJob()
	job.revisions, job.marks = r, marks
	if err := exportAs(w.textEditor.Text(), filepath.Join(os.Getenv("SHOT_DIR"), "revisions.pdf"), "pdf", job); err != nil {
		t.Fatal(err)
	}
}

// New after an unsaved script's issue: the new script has no marks, and
// Issue Draft… offers White again.
func TestNewScriptForgetsIssues(t *testing.T) {
	w := draftsWindow(t)
	w.textEditor.SetText(revScript)
	issueNow(t, w, "White", time.Now())
	replaceText(t, w, "Mud everywhere.", "Mud.")
	if got := marked(w); len(got) != 1 {
		t.Fatalf("marks %v", got)
	}
	w.hasChanges = false
	w.newFile(true)
	w.textEditor.SetText(revScript)
	replaceText(t, w, "Mud everywhere.", "Mud.")
	if got := marked(w); len(got) != 0 {
		t.Errorf("a new script with the old one's marks: %v", got)
	}
	if c, _ := w.revisionChainNow(); c != nil && len(c.issues) != 0 {
		t.Errorf("a new script with issues %+v", c.issues)
	}
}

// In an .accolade: White, an edit, Blue, an edit, saved and opened again.
func TestIssuesInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Rain.accolade")
	w.textEditor.SetText(revScript)
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename("", "untitled.fountain", p, p)
	w.currentFile = p
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Mud everywhere.", "Mud, and a dog.")
	issueNow(t, w, "Blue", time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Dry at last.", "Dry, at last.")
	if got := marked(w); len(got) != 1 || got["Dry, at last."] != "Pink" {
		t.Errorf("marks %v", got)
	}
	if err := w.saveToFile(p); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	w2 := againWindow(t, w)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	ds := draftsOf(t, w2)
	if len(ds) != 2 || ds[0].Colour != "White" || ds[1].Colour != "Blue" {
		t.Fatalf("issues in the bundle %+v", ds)
	}
	w2.settings.SetString(revisionMarksPref, "all")
	t.Cleanup(func() { w2.settings.SetString(revisionMarksPref, "last") })
	if got := marked(w2); len(got) != 2 || got["Mud, and a dog."] != "Blue" || got["Dry, at last."] != "Pink" {
		t.Errorf("reopened marks %v", got)
	}
}

// After an import whose file names its sets its own way, the next set is
// one it has not used.
func TestNextSetSkipsImportedNames(t *testing.T) {
	c := &revisionChain{issues: []issue{{set: lex.RevisionSet{Name: "Pink"}, sets: []lex.RevisionSet{{Name: "Blue"}, {Name: "Yellow"}}}}}
	if n := c.nextSet(time.Now()).Name; n != "Green" {
		t.Errorf("next %q, want Green", n)
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The stage as the facts tell it.
func TestStageOf(t *testing.T) {
	for _, c := range []struct {
		f    stageFacts
		want storyStage
	}{
		{stageFacts{markdown: true}, stageNone},
		{stageFacts{}, stageWriting},
		{stageFacts{hasSource: true}, stageAtSource},
		{stageFacts{hasSource: true, linked: true}, stageWriting},
		{stageFacts{hasSource: true, sourceLocked: true}, stageWriting},
		{stageFacts{issues: 1}, stageLocked},
		{stageFacts{issues: 1, changed: true}, stageShooting},
		{stageFacts{issues: 2}, stageShooting},
		{stageFacts{hasSource: true, issues: 1}, stageLocked},
	} {
		if got := stageOf(c.f); got != c.want {
			t.Errorf("%+v: %v, want %v", c.f, got, c.want)
		}
	}
}

const stagesStory = "# The Garden\n\nShe opened the door to the garden and stepped out into the rain.\n\n" +
	"The rain kept falling on the roses.\n\nAnna went back inside.\n"

const stagesScript = "Title: The Garden\n\nINT. HOUSE - NIGHT\n\nAnna at the door.\n\nEXT. GARDEN - NIGHT\n\nRain.\n"

// stagesWindow is the script with the story beside it, the pane shown.
func stagesWindow(t *testing.T) (*MainWindow, string) {
	t.Helper()
	w, p := sourceWindow(t, stagesScript, stagesStory)
	w.sourcePane().follow()
	return w, filepath.Join(filepath.Dir(p), "story.md")
}

// sourceLocksOf are the store's source locks.
func sourceLocksOf(t *testing.T, w *MainWindow) []savedDraft {
	t.Helper()
	var out []savedDraft
	for _, d := range draftsOf(t, w) {
		if d.Kind == kindSource {
			out = append(out, d)
		}
	}
	return out
}

// Source → Writing: the first link locks the source as adapted (a
// source draft in the store, the status line saying so); the title
// names the stage; Unlock takes it out, Lock puts that one back.
func TestSourceLockedAtFirstLink(t *testing.T) {
	w, _ := stagesWindow(t)
	if got := w.stageShown(); got != stageAtSource || !strings.Contains(w.fyneWindow.Title(), "story.fountain · Source") {
		t.Fatalf("before a link: %v, title %q", got, w.fyneWindow.Title())
	}
	if n := len(sourceLocksOf(t, w)); n != 0 {
		t.Fatalf("locked before a link: %d", n)
	}
	replaceText(t, w, "Anna at the door.", `Anna at the door. [[from: "She opened the door to the garden"]]`)
	w.sourcePane().follow()
	locks := sourceLocksOf(t, w)
	if len(locks) != 1 || locks[0].Name != "Source 1" {
		t.Fatalf("not locked at the first link: %+v", locks)
	}
	if !strings.Contains(w.statusText(), "locked as adapted") || !strings.Contains(w.statusText(), "Unlock Source") {
		t.Errorf("status %q", w.statusText())
	}
	if !strings.Contains(w.fyneWindow.Title(), "· Writing") {
		t.Errorf("title %q", w.fyneWindow.Title())
	}
	src, err := readSourceLock(w.extrasHome(), locks[0])
	if err != nil || src != stagesStory {
		t.Errorf("locked source %q (%v)", src, err)
	}
	// a second link, a later follow: no second lock
	replaceText(t, w, "Rain.", `Rain. [[from: "The rain kept falling"]]`)
	w.sourcePane().follow()
	if n := len(sourceLocksOf(t, w)); n != 1 {
		t.Errorf("%d locks after the second link", n)
	}

	w.unlockSource()
	if n := len(sourceLocksOf(t, w)); n != 0 || !strings.Contains(w.statusText(), "unlocked") {
		t.Fatalf("unlock: %d locks, %q", n, w.statusText())
	}
	w.lockSource(false, nil)
	back := sourceLocksOf(t, w)
	if len(back) != 1 || back[0].ID != locks[0].ID || !strings.Contains(w.statusText(), "locked again") {
		t.Errorf("locked again: %+v, %q", back, w.statusText())
	}
	w.lockSource(false, nil)
	if n := len(sourceLocksOf(t, w)); n != 1 || !strings.Contains(w.statusText(), "already") {
		t.Errorf("locked twice: %d, %q", n, w.statusText())
	}
}

// A linked script with no lock (linked before locks were kept) is not
// locked on its own on opening: Lock Source does.
func TestLinkedScriptNotLockedOnOpening(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.sourcePane().follow()
	if n := len(sourceLocksOf(t, w)); n != 0 {
		t.Fatalf("locked on opening: %d", n)
	}
	if !cmd("source.lock").enabled(w.cmdEnv()) {
		t.Error("Lock Source not offered")
	}
	w.lockSource(false, nil)
	if n := len(sourceLocksOf(t, w)); n != 1 {
		t.Errorf("Lock Source: %d", n)
	}
	if cmd("source.lock").enabled(w.cmdEnv()) || !cmd("source.unlock").enabled(w.cmdEnv()) {
		t.Error("Lock/Unlock not following the lock")
	}
}

// The source edited after the lock: the linked passage whose words
// changed shows the notice (and the picked one Relink From Here), the
// count says so; Relink From Here links to the source now and locks it
// as Source 2, the notices gone; the first lock stays.
func TestSourceChangedSinceLinked(t *testing.T) {
	w, story := stagesWindow(t)
	replaceText(t, w, "Anna at the door.", `Anna at the door. [[from: "She opened the door to the garden"]]`)
	s := w.sourcePane()
	s.follow()
	if len(sourceLocksOf(t, w)) != 1 {
		t.Fatal("not locked")
	}
	if len(s.changed) != 0 {
		t.Fatalf("changed right after the lock: %v", s.changed)
	}
	edited := strings.Replace(stagesStory, "stepped out into the rain", "stepped out into the storm", 1)
	os.WriteFile(story, []byte(edited), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(story, later, later)
	s.follow()
	if len(s.changed) != 1 || !s.changed[1] {
		t.Fatalf("changed %v (passages %d)", s.changed, len(s.passages))
	}
	if !strings.Contains(s.count.Text, "1 changed since linked") {
		t.Errorf("count %q", s.count.Text)
	}
	if a := s.rowActs(1); !a.changed || a.relink {
		t.Errorf("not picked: %+v", a)
	}
	s.setPicked(1)
	if a := s.rowActs(1); !a.changed || !a.relink {
		t.Errorf("picked: %+v", a)
	}
	if !cmd("source.relinkFromHere").enabled(w.cmdEnv()) {
		t.Error("Relink From Here not offered")
	}
	// the row shows the notice over its text
	row := newPassageRow()
	fill(rowParts(row), s.passages[1].Markdown, false, s.rowActs(1))
	if segs := rowParts(row).text.Segments; len(segs) == 0 || !strings.Contains(segs[0].Textual(), "Changed since the script was linked") {
		t.Errorf("no notice in the row")
	}
	if a, b := s.measured(s.passages[1].Markdown, false, lineActs(rowShape{line: true, note: true}), 300),
		s.measured(s.passages[1].Markdown, false, lineActs(rowShape{line: true}), 300); a <= b {
		t.Errorf("the notice takes no room: %v, %v", a, b)
	}

	_, lockSrc := w.sourceLock()
	done := false
	w.relinkNow(lockSrc, func(ok bool) { done = ok })
	if !done {
		t.Fatal("Relink From Here did not lock")
	}
	locks := sourceLocksOf(t, w)
	if len(locks) != 2 || locks[1].Name != "Source 2" {
		t.Fatalf("locks %+v", locks)
	}
	s.follow()
	if len(s.changed) != 0 || strings.Contains(s.count.Text, "changed since") {
		t.Errorf("after Relink From Here: %v, %q", s.changed, s.count.Text)
	}
}

// Writing → Locked → Shooting: White locks the writer's draft; an edit
// after it, or a later issue, is the shooting script.
func TestStagesThroughProduction(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	if got := w.stageNow(); got != stageWriting {
		t.Fatalf("at first %v", got)
	}
	if strings.Contains(w.fyneWindow.Title(), "·") {
		t.Errorf("a plain script's title names a stage: %q", w.fyneWindow.Title())
	}
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	if got := w.stageNow(); got != stageLocked || !strings.Contains(w.fyneWindow.Title(), "· Locked") {
		t.Errorf("after White: %v, %q", got, w.fyneWindow.Title())
	}
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	w.updateRevisionMarks()
	if got := w.stageNow(); got != stageShooting || !strings.Contains(w.fyneWindow.Title(), "· Shooting") {
		t.Errorf("edited after White: %v, %q", got, w.fyneWindow.Title())
	}
	issueNow(t, w, "Blue", time.Date(2026, 11, 20, 0, 0, 0, 0, time.Local))
	if got := w.stageNow(); got != stageShooting {
		t.Errorf("after Blue: %v", got)
	}
	ds := draftsOf(t, w)
	d := w.stagesDialog(ds)
	if got := lockedStages(ds); len(got) != 2 || stageLabel(got[0], true) != "Writer's draft locked: White" ||
		stageLabel(got[1], false) != "Revision issued: Blue" {
		t.Errorf("stages %+v", got)
	}
	d.Show()
	d.Hide()
}

// The writer's version is White's text without production's layers; the
// shooting version the script now with them; the Export dialog has both,
// File's items open it with one chosen.
func TestWriterAndShootingVersions(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	before, from := w.writerVersion()
	if from != "" || strings.Contains(before, "#1#") {
		t.Errorf("before an issue: from %q:\n%s", from, before)
	}
	issueNow(t, w, "White", time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local))
	replaceText(t, w, "Mud everywhere.", "Mud everywhere, and a dog.")
	text, from := w.writerVersion()
	if from != "White" || strings.Contains(text, "and a dog") || strings.Contains(text, "#1#") ||
		strings.Contains(text, "/* Production") || !strings.Contains(text, "Mud everywhere.") {
		t.Errorf("writer's version (from %q):\n%s", from, text)
	}
	ed := NewExportDialog(w)
	ed.setVersion(versionWriter)
	if ed.content() != text || !strings.Contains(ed.filenameEntry.Text, " - writer") ||
		!strings.Contains(ed.versionNote.Text, "White") {
		t.Errorf("dialog, writer: %q, note %q", ed.filenameEntry.Text, ed.versionNote.Text)
	}
	ed.formatSelect.SetSelected(exportFormatLabel("pdf"))
	if j := ed.job(); j.revisions != nil || j.lock != nil || j.numberScenes {
		t.Errorf("writer's job has production's layers: %+v", j.revisions)
	}
	ed.setVersion(versionShooting)
	if ed.content() != w.textEditor.Text() || strings.Contains(ed.filenameEntry.Text, " - writer") {
		t.Errorf("dialog, shooting: %q", ed.filenameEntry.Text)
	}
	if j := ed.job(); j.revisions == nil {
		t.Error("shooting job without its revisions")
	}
}

// The screenshots: the stage in the title, the notice, View > Stages,
// on the desktop and a phone (SHOT_DIR).
func TestStagesScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w, story := stagesWindow(t)
		w.fyneWindow.Resize(size.size)
		replaceText(t, w, "Anna at the door.", `Anna at the door. [[from: "She opened the door to the garden"]]`)
		s := w.sourcePane()
		s.follow()
		edited := strings.Replace(stagesStory, "stepped out into the rain", "stepped out into the storm", 1)
		os.WriteFile(story, []byte(edited), 0o644)
		later := time.Now().Add(time.Minute)
		os.Chtimes(story, later, later)
		s.follow()
		s.setPicked(1)
		s.list.Refresh()
		s.view.close.OnTapped() // room for the list on a phone
		s.list.ScrollToOffset(s.heights[0] + theme.Padding())
		shotTo(t, w, "stages-notice-"+size.name)
		w.afterBundleWrites(func() {})
		w.stagesDialog(draftsOf(t, w)).Show()
		shotTo(t, w, "stages-dialog-"+size.name)
	}
}

// In an .accolade the lock is written into the bundle; a source kept in
// the script is locked as its Source block, read back from the lock's
// text.
func TestSourceLockInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	story := filepath.Join(t.TempDir(), "novel.md")
	os.WriteFile(story, []byte("# Novel\n\nNight falls on the town.\n"), 0o644)
	if err := w.useSource(story); err != nil {
		t.Fatal(err)
	}
	w.lockSource(false, nil)
	locks := sourceLocksOf(t, w)
	if len(locks) != 1 || locks[0].Source == "" {
		t.Fatalf("bundle lock %+v (%q)", locks, w.statusText())
	}
	if lock, src := w.sourceLock(); lock == nil || !strings.Contains(src, "Night falls") {
		t.Errorf("read back %q", src)
	}
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	replaceText(t, w, "Night falls", "Night came")
	w.lockSource(false, nil)
	locks = sourceLocksOf(t, w)
	if len(locks) != 2 || locks[1].Source != "" {
		t.Fatalf("inside: %+v (%q)", locks, w.statusText())
	}
	if _, src := w.sourceLock(); !strings.Contains(src, "Night came") {
		t.Errorf("inside, read back %q", src)
	}
}

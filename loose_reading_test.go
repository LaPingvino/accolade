package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// looseScript relied on Lexington's old reading in two places: a heading
// right under action, a name right under action.
const looseScript = "INT. KITCHEN - DAY\n\nBram cooks.\nEXT. GARDEN - DAY\n\nAnna waits.\nANNA\nHe's late.\n"

const keptScript = "INT. KITCHEN - DAY\n\nBram cooks.\n\nEXT. GARDEN - DAY\n\nAnna waits.\n\nANNA\nHe's late.\n"

func TestLooseReadingKeep(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "old.fountain")
	os.WriteFile(p, []byte(looseScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if !w.loose.box.Visible() || !strings.Contains(w.loose.text.Text, "in 2 places") {
		t.Fatalf("notice %v %q", w.loose.box.Visible(), w.loose.text.Text)
	}
	steps := w.textEditor.UndoSteps()
	test.Tap(w.loose.keep)
	if got := w.textEditor.Text(); got != keptScript {
		t.Fatalf("kept:\n%q\nwant\n%q", got, keptScript)
	}
	if w.loose.box.Visible() || w.textEditor.UndoSteps() != steps+1 {
		t.Errorf("notice %v, undo steps %d (was %d)", w.loose.box.Visible(), w.textEditor.UndoSteps(), steps)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != looseScript {
		t.Errorf("one undo: %q", w.textEditor.Text())
	}
	// never asked twice: not even before it is saved
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.loose.box.Visible() || w.looseAnswer(p) != "keep" {
		t.Errorf("asked again (%q)", w.looseAnswer(p))
	}
}

func TestLooseReadingNewWay(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "old.fountain")
	os.WriteFile(p, []byte(looseScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	test.Tap(w.loose.new)
	if w.textEditor.Text() != looseScript || w.loose.box.Visible() {
		t.Fatalf("changed: %q, notice %v", w.textEditor.Text(), w.loose.box.Visible())
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.loose.box.Visible() {
		t.Error("asked again")
	}
	// another script is asked about, a spec-valid one is not
	q := filepath.Join(dir, "other.fountain")
	os.WriteFile(q, []byte(looseScript), 0o644)
	if err := w.LoadFile(q); err != nil {
		t.Fatal(err)
	}
	if !w.loose.box.Visible() {
		t.Error("another script not asked about")
	}
	r := filepath.Join(dir, "spec.fountain")
	os.WriteFile(r, []byte(keptScript), 0o644)
	if err := w.LoadFile(r); err != nil {
		t.Fatal(err)
	}
	if w.loose.box.Visible() {
		t.Error("a script that reads the same either way asked about")
	}
	// a Markdown document is never asked about
	m := filepath.Join(dir, "notes.md")
	os.WriteFile(m, []byte("Bram cooks.\nEXT. GARDEN - DAY\n"), 0o644)
	if err := w.LoadFile(m); err != nil {
		t.Fatal(err)
	}
	if w.loose.box.Visible() {
		t.Error("Markdown asked about")
	}
}

// A bundle is asked about as a plain file is, its answer kept by its
// path; a rename carries it.
func TestLooseReadingBundle(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := filepath.Join(t.TempDir(), "Story.textbundle")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "info.json"), []byte(highlandInfo), 0o644)
	os.WriteFile(filepath.Join(dir, "text.fountain"), []byte(looseScript), 0o644)
	if err := w.LoadFile(dir); err != nil {
		t.Fatal(err)
	}
	if !w.loose.box.Visible() {
		t.Fatal("bundle not asked about")
	}
	test.Tap(w.loose.new)
	if w.looseAnswer(dir) != "new" {
		t.Errorf("answer %q", w.looseAnswer(dir))
	}
	to := filepath.Join(filepath.Dir(dir), "Renamed.textbundle")
	w.looseAfterRename(dir, to)
	if w.looseAnswer(to) != "new" {
		t.Errorf("after a rename: %q", w.looseAnswer(to))
	}
}

// A script not saved yet is asked once while it is open, and the
// answer goes with it when it is saved.
func TestLooseReadingUnsaved(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(looseScript)
	w.checkLooseReading()
	if !w.loose.box.Visible() {
		t.Fatal("not asked")
	}
	test.Tap(w.loose.new)
	w.checkLooseReading()
	if w.loose.box.Visible() {
		t.Error("asked twice")
	}
	p := filepath.Join(t.TempDir(), "new.fountain")
	w.looseAfterRename("", p)
	if w.looseAnswer(p) != "new" || w.looseUnsaved != "" {
		t.Errorf("saved: %q, unsaved %q", w.looseAnswer(p), w.looseUnsaved)
	}
}

// TestLooseReadingScreenshots renders the notice on the desktop and on a
// phone (SHOT_DIR as in screenshot_test.go): strict-desktop.png,
// strict-phone.png.
func TestLooseReadingScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		p := filepath.Join(t.TempDir(), "old.fountain")
		os.WriteFile(p, []byte(looseScript), 0o644)
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		w.fyneWindow.Resize(c.size)
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "strict-"+c.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

// A rename carries the answer (File > Rename, through afterNewName).
func TestLooseReadingRename(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(looseScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	test.Tap(w.loose.new)
	if err := w.renameTo("film.fountain"); err != nil {
		t.Fatal(err)
	}
	if got := w.looseAnswer(filepath.Join(dir, "film.fountain")); got != "new" {
		t.Errorf("after the rename: %q", got)
	}
}

// A speech's line of two spaces survives opening and saving, in a plain
// file and in a bundle.
func TestTwoSpacesSurviveSaving(t *testing.T) {
	const script = "INT. HOUSE - DAY\n\nBOB\nHello.\n  \nGoodbye.\n"
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "two.fountain")
	os.WriteFile(p, []byte(script), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.textEditor.Text() != script || w.loose.box.Visible() == false {
		t.Fatalf("opened %q, notice %v", w.textEditor.Text(), w.loose.box.Visible())
	}
	w.hasChanges = true
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != script {
		t.Errorf("saved %q", b)
	}
	dir := filepath.Join(t.TempDir(), "Two.textbundle")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "info.json"), []byte(highlandInfo), 0o644)
	os.WriteFile(filepath.Join(dir, "text.fountain"), []byte(script), 0o644)
	if err := w.LoadFile(dir); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetText(w.textEditor.Text() + "\nBob leaves.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	w.flushBundleWrites()
	if b, _ := os.ReadFile(filepath.Join(dir, "text.fountain")); !strings.HasPrefix(string(b), script) {
		t.Errorf("bundle saved %q", b)
	}
}

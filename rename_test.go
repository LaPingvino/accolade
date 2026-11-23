package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenamedName(t *testing.T) {
	for _, c := range []struct{ typed, old, want string }{
		{"draft", "scene.fountain", "draft.fountain"},
		{" draft.eo.fountain ", "scene.fountain", "draft.eo.fountain"},
		{"notes", "", "notes.fountain"},
		{"a/b", "scene.fountain", ""},
		{"  ", "scene.fountain", ""},
	} {
		if got := renamedName(c.typed, c.old); got != c.want {
			t.Errorf("%q from %q: got %q, want %q", c.typed, c.old, got, c.want)
		}
	}
}

func TestRenameTo(t *testing.T) {
	w := newTestWindow(t, "EN. KUIRO - TAGO")
	if err := w.renameTo("kuiro.eo.fountain"); err != nil {
		t.Fatal(err)
	}
	if w.suggestedName != "kuiro.eo.fountain" || w.language != "eo" || w.headerBar.titleLabel.Text != "• kuiro.eo.fountain (not saved yet)" {
		t.Errorf("unsaved: name %q, language %q, title %q", w.suggestedName, w.language, w.headerBar.titleLabel.Text)
	}
	dir := t.TempDir()
	w.currentFile = filepath.Join(dir, "a.fountain")
	os.WriteFile(w.currentFile, []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "taken.fountain"), []byte("y"), 0o644)
	if err := w.renameTo("taken.fountain"); err == nil {
		t.Error("renamed over another file")
	}
	if err := w.renameTo("b.fountain"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "b.fountain")); err != nil || filepath.Base(w.currentFile) != "b.fountain" || w.headerBar.titleLabel.Text != "b.fountain" {
		t.Errorf("saved: file %v, current %q, title %q", err, w.currentFile, w.headerBar.titleLabel.Text)
	}
}

func TestMarkLanguageRetitles(t *testing.T) {
	w := newTestWindow(t, "EN. KUIRO")
	w.changeLanguage("eo", markLanguage, nil)
	if got := w.headerBar.titleLabel.Text; got != "• untitled.eo.fountain (not saved yet)" {
		t.Errorf("title after marking: %q", got)
	}
}

// A script not saved yet renamed to an Accolade folder is one: Rename and
// Save As show it so, and its first save writes the folder, not a zip.
func TestRenameUnsavedToFolder(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain falls.\n")
	w.renameScript()
	a := w.renameAsked
	a.base.SetText("film")
	a.kind.SetSelected(typeAccoladeFolder.label + " (" + typeAccoladeFolder.shownExt() + ")")
	a.confirm()
	if w.suggestedName != "film.accolade" || w.suggestedForm != bundleFolder {
		t.Fatalf("renamed: %q, form %v", w.suggestedName, w.suggestedForm)
	}
	w.renameScript()
	if got := w.renameAsked.kind.Selected; got != typeAccoladeFolder.label+" (.accolade)" {
		t.Errorf("Rename again shows %q", got)
	}
	w.renameAsked.confirm() // the same again: still the folder
	if w.suggestedForm != bundleFolder {
		t.Errorf("renamed again: form %v", w.suggestedForm)
	}
	w.showSaveAs()
	s := w.saveAsAsked
	if s.chosen().id != typeAccoladeFolder.id {
		t.Errorf("Save As has %q chosen", s.chosen().id)
	}
	p := filepath.Join(t.TempDir(), "film.accolade")
	r := s.target()
	w.saveAsWith(p, &r)
	waitFor(t, "the first save", func() bool { return w.currentFile == p && !w.bundleBusy() })
	if st, err := os.Stat(p); err != nil || !st.IsDir() {
		t.Fatalf("first save: %v, a folder: %v", err, err == nil && st.IsDir())
	}
	if _, err := os.Stat(filepath.Join(p, "text.fountain")); err != nil {
		t.Error(err)
	}
	if w.currentFile != p || w.suggestedForm != notBundle {
		t.Errorf("after: %q, form %v", w.currentFile, w.suggestedForm)
	}
}

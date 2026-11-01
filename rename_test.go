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

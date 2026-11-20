package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A kept script comes back with its file and its language, and its
// changes not saved; an untitled one with its language.
func TestRestoreKeepsLanguage(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	file := filepath.Join(t.TempDir(), "scene.eo.fountain")
	if err := os.WriteFile(file, []byte("EN. DOMO - TAGO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.restore(kept{Text: "EN. DOMO - NOKTO\n", File: file})
	if w.language != "eo" || w.currentFile != file || w.textEditor.Text() != "EN. DOMO - NOKTO\n" || !w.hasChanges {
		t.Errorf("with a file: %q %q %q changed %v", w.language, w.currentFile, w.textEditor.Text(), w.hasChanges)
	}
	w.restore(kept{Text: "ИНТ. ДОМ - ДЕНЬ\n", Language: "ru", File: "/gone.fountain", Name: "dom.ru.fountain"})
	if w.language != "ru" || w.textEditor.Text() != "ИНТ. ДОМ - ДЕНЬ\n" || w.suggestedName != "dom.ru.fountain" {
		t.Errorf("not saved: %q %q %q", w.language, w.textEditor.Text(), w.suggestedName)
	}
}

// An unsaved script's source comes back with it after a reload: the
// draft names its kept copy.
func TestRestoreKeepsSource(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	w.createNewFile(true)
	story := filepath.Join(t.TempDir(), "story.md")
	os.WriteFile(story, []byte("# Story\n\nIt was night.\n"), 0o644)
	if err := w.useSource(story); err != nil {
		t.Fatal(err)
	}
	d := w.draft()
	if d.Source == "" {
		t.Fatal("the draft has no source")
	}
	w2, _ := newDialogTestWindow(t)
	w2.restore(d)
	if r := w2.sourceRef(); r.Origin != sourceKept || r.Path != d.Source {
		t.Errorf("restored: %+v", r)
	}
	w2.restore(kept{Text: "x", Source: "/gone/story.md"}) // gone: none
	if r := w2.sourceRef(); r.Origin != sourceNone {
		t.Errorf("a draft whose source is gone: %+v", r)
	}
}

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

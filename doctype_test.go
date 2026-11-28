// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
)

// documentTypeItems are View > Document Type's items.
func documentTypeItems(t *testing.T, w *MainWindow) []*fyne.MenuItem {
	t.Helper()
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label != "View" {
			continue
		}
		for _, it := range m.Items {
			if it.Label == "Document Type" && it.ChildMenu != nil {
				return it.ChildMenu.Items
			}
		}
	}
	t.Fatal("no View > Document Type")
	return nil
}

// checkedType is the type checked in View > Document Type.
func checkedType(t *testing.T, w *MainWindow) string {
	t.Helper()
	var on []string
	for _, it := range documentTypeItems(t, w) {
		if it.Checked {
			on = append(on, it.Label)
		}
	}
	if len(on) != 1 {
		t.Fatalf("checked: %v", on)
	}
	return on[0]
}

// pickType picks a type in View > Document Type.
func pickType(t *testing.T, w *MainWindow, label string) {
	t.Helper()
	for _, it := range documentTypeItems(t, w) {
		if it.Label == label {
			it.Action()
			return
		}
	}
	t.Fatalf("no %s", label)
}

// Switching keeps the name and the text exactly (Markdown's indents too,
// no Fountain clean-up), both ways, and the menu follows.
func TestDocumentTypeSwitch(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "story.fountain")
	os.WriteFile(p, []byte(story), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	text := w.textEditor.Text()
	if w.markdown || checkedType(t, w) != "Screenplay" {
		t.Fatal("a .fountain file is not a screenplay")
	}
	pickType(t, w, "Markdown")
	if !w.markdown || !w.textEditor.Markdown || checkedType(t, w) != "Markdown" {
		t.Fatal("not Markdown")
	}
	if !slices.Contains(menuLabels(w, "View"), "Live Preview") {
		t.Error("the menu is still a screenplay's")
	}
	if w.currentFile != p || w.textEditor.Text() != text || w.hasChanges {
		t.Errorf("switching changed the document: %q %v\n%q", w.currentFile, w.hasChanges, w.textEditor.Text())
	}
	pickType(t, w, "Screenplay")
	if w.markdown || w.textEditor.Format == nil || checkedType(t, w) != "Screenplay" {
		t.Fatal("not a screenplay again")
	}
	if w.textEditor.Text() != text || w.hasChanges {
		t.Errorf("switching back changed the text:\n%q", w.textEditor.Text())
	}
	// an indented Markdown list stays indented through a screenplay
	w.NewMarkdownFile()
	w.textEditor.SetText(story)
	pickType(t, w, "Screenplay")
	pickType(t, w, "Markdown")
	if w.textEditor.Text() != story {
		t.Errorf("Markdown changed by a round trip:\n%q", w.textEditor.Text())
	}
}

// The choice is remembered for the file (in settings.json, so after a
// restart too); without one, the extension says.
func TestDocumentTypeRemembered(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	txt := filepath.Join(dir, "notes.txt")
	fountain := filepath.Join(dir, "draft.fountain")
	md := filepath.Join(dir, "plain.md")
	for _, p := range []string{txt, fountain, md} {
		os.WriteFile(p, []byte("# Heading\n\n- item\n    more\n"), 0o644)
	}
	open := func(p string) {
		t.Helper()
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
	}
	// the extension is the default
	for p, want := range map[string]bool{txt: false, fountain: false, md: true} {
		open(p)
		if w.markdown != want {
			t.Errorf("%s opens as Markdown %v", filepath.Base(p), w.markdown)
		}
	}
	open(txt)
	pickType(t, w, "Markdown")
	open(fountain)
	pickType(t, w, "Markdown")
	open(md)
	pickType(t, w, "Screenplay")

	open(md)
	if w.markdown || checkedType(t, w) != "Screenplay" {
		t.Error("the .md file chosen a screenplay reopens as Markdown")
	}
	for _, p := range []string{txt, fountain} {
		open(p)
		if !w.markdown || checkedType(t, w) != "Markdown" {
			t.Errorf("%s chosen Markdown reopens as a screenplay", filepath.Base(p))
		}
		if w.textEditor.Text() != "# Heading\n\n- item\n    more\n" {
			t.Errorf("%s reopened with its text changed: %q", filepath.Base(p), w.textEditor.Text())
		}
	}

	// kept on disk: what a restart reads
	s := NewSettings()
	got := s.GetStringSlice("document-types")
	for _, want := range []string{"markdown:" + txt, "markdown:" + fountain, "screenplay:" + md} {
		if !slices.Contains(got, want) {
			t.Errorf("settings.json has no %s: %q", want, got)
		}
	}

	// choosing what the extension says forgets the choice
	open(md)
	pickType(t, w, "Markdown")
	if slices.Contains(w.settings.GetStringSlice("document-types"), "screenplay:"+md) {
		t.Error("the .md file's choice is still kept")
	}
}

// Renaming: a new .md/.fountain extension decides; otherwise the type and
// its choice go with the file.
func TestDocumentTypeRename(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	os.WriteFile(p, []byte("# A\n"), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	pickType(t, w, "Markdown")
	if err := w.renameTo("b.txt"); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(dir, "b.txt")
	if !w.markdown {
		t.Error("renamed to b.txt, no longer Markdown")
	}
	if w.LoadFile(b); !w.markdown {
		t.Error("b.txt does not reopen as Markdown")
	}
	if slices.Contains(w.settings.GetStringSlice("document-types"), "markdown:"+p) {
		t.Error("the old name's choice is kept")
	}
	if err := w.renameTo("b.fountain"); err != nil || w.markdown {
		t.Errorf("renamed .fountain: still Markdown (%v)", err)
	}
	if err := w.renameTo("b.md"); err != nil || !w.markdown {
		t.Errorf("renamed .md: not Markdown (%v)", err)
	}
	// an unsaved screenplay made Markdown stays so by a name saying nothing
	w.NewFile()
	pickType(t, w, "Markdown")
	if w.untitledName() != "untitled.md" {
		t.Errorf("saved as %s", w.untitledName())
	}
	if err := w.renameTo("story.txt"); err != nil || !w.markdown {
		t.Errorf("named story.txt: not Markdown (%v)", err)
	}
}

// Notes go along in the new type's form: a boneyard becomes an HTML
// comment and back, the notes and the rest of the text as they were.
func TestDocumentTypeCarriesNotes(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.NewEmptyFile()
	body := "INT. BARN - DAY\n\nRain."
	screenplay := body + fountainNotes.text("Call Bram.\n-- > not the end")
	w.textEditor.SetText(screenplay)
	w.hasChanges = false
	w.togglePane("notes")
	pickType(t, w, "Markdown")
	got := w.textEditor.Text()
	if want := body + "\n\n<!-- Notes\nCall Bram.\n-- > not the end\n-->\n"; got != want {
		t.Fatalf("notes in Markdown:\n%q, want\n%q", got, want)
	}
	w.notesPane().read()
	if its := w.notesPane().all; markdownNotes.of(got) != "Call Bram.\n-- > not the end" || len(its) != 1 || its[0].Text != "Call Bram.\n--> not the end" {
		t.Errorf("the Notes pane shows %+v", its)
	}
	if !w.hasChanges {
		t.Error("the notes moved, and the document is not changed")
	}
	pickType(t, w, "Screenplay")
	if got := w.textEditor.Text(); got != screenplay {
		t.Errorf("back as a screenplay:\n%q, want\n%q", got, screenplay)
	}
	// a document with notes in its own form already keeps them as they are
	w.setMarkdown(true)
	both := "Text." + fountainNotes.text("old") + markdownNotes.text("new")
	w.textEditor.SetText(both)
	w.setMarkdown(false)
	pickType(t, w, "Markdown")
	if w.textEditor.Text() != both {
		t.Errorf("notes changed: %q", w.textEditor.Text())
	}
}

// Editing together, a switch reaches the peer as the mode, not as a
// choice of the peer's.
func TestDocumentTypeShared(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	p := filepath.Join(t.TempDir(), "s.fountain")
	os.WriteFile(p, []byte("Text.\n"), 0o644)
	w.LoadFile(p)
	w.applyShared("markdown", "true")
	if !w.markdown || checkedType(t, w) != "Markdown" {
		t.Error("the peer's Markdown is not shown")
	}
	if _, chosen := w.chosenType(p); chosen {
		t.Error("the peer's switch is kept as this user's choice")
	}
}

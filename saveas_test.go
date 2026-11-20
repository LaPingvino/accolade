package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSaveAsName(t *testing.T) {
	for _, c := range []struct {
		current, suggested string
		production, md     bool
		want               string
	}{
		{"", "", false, false, "untitled.fountain"},
		{"", "", true, false, "untitled.accolade"},
		{"/a/story.fountain", "", true, false, "story.accolade"},
		{"/a/story.fountain", "", false, false, "story.fountain"},
		{"/a/story.highland", "", true, false, "story.highland"},
		{"/a/story.textpack", "", true, false, "story.textpack"},
		{"", "imported.fountain", true, false, "imported.accolade"},
		{"/a/notes.md", "", true, true, "notes.md"},
	} {
		untitled := "untitled.fountain"
		if c.md {
			untitled = "untitled.md"
		}
		if got := saveAsName(c.current, c.suggested, untitled, c.production, c.md); got != c.want {
			t.Errorf("%+v: %s", c, got)
		}
	}
}

// Save As offers .accolade once the script has production data: a draft,
// or a lock in its production block.
func TestOfferedName(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "story.fountain")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if got := w.offeredName(); got != "story.fountain" {
		t.Errorf("plain: %s", got)
	}
	w.textEditor.SetText(bundleScript + "\n/* Production\nscene numbers: locked, 10A\n*/\n")
	if got := w.offeredName(); got != "story.accolade" {
		t.Errorf("locked: %s", got)
	}
	w.textEditor.SetText(bundleScript)
	saveDraftNow(t, w, "One")
	if got := w.offeredName(); got != "story.accolade" {
		t.Errorf("with a draft: %s", got)
	}
	// Save keeps the file plain: the drafts where a plain script keeps them
	w.textEditor.SetText(bundleScript + "More.\n")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != bundleScript+"More.\n" || w.currentFile != p {
		t.Errorf("Save wrote %q to %s", b, w.currentFile)
	}
}

// Save As .fountain of a bundle with drafts asks where they go; nothing
// is written before the answer; Cancel writes nothing.
func TestSaveAsFlattenAsks(t *testing.T) {
	w, p := bundleWithDraft(t)
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	w.saveAs(to)
	ask := w.flattenAsked
	if ask == nil {
		t.Fatal("not asked")
	}
	if _, err := os.Stat(to); err == nil {
		t.Fatal("written before the answer")
	}
	ask.choose(flattenFolder)
	if _, err := os.Stat(filepath.Join(productionFolderOf(to), "assets", "drafts", "drafts.json")); err != nil {
		t.Errorf("no folder: %v", err)
	}
	if w.currentFile != to || w.bundle != nil || !strings.Contains(w.statusLabel.Text, "Script.production") {
		t.Errorf("after: %s, bundle %v, status %q", w.currentFile, w.bundle, w.statusLabel.Text)
	}
	// left out: asked again first
	w2, p2 := bundleWithDraft(t)
	to2 := filepath.Join(filepath.Dir(p2), "Bare.fountain")
	w2.saveAs(to2)
	w2.flattenAsked.choose(flattenDrop)
	if _, err := os.Stat(to2); err == nil {
		t.Fatal("left out before it was confirmed")
	}
	w2.flattenAsked.confirmDrop(false)
	if _, err := os.Stat(to2); err == nil {
		t.Fatal("written though not confirmed")
	}
	w2.flattenAsked.choose(flattenDrop)
	w2.flattenAsked.confirmDrop(true)
	if b, err := os.ReadFile(to2); err != nil || string(b) != bundleScript {
		t.Errorf("plain %q %v", b, err)
	}
	if w2.extrasHome().kind != extrasNone {
		t.Error("drafts kept though left out")
	}
	// the bundle still has them
	w3 := againWindow(t, w2)
	if err := w3.LoadFile(p2); err != nil || len(draftsOf(t, w3)) != 1 {
		t.Errorf("the bundle lost its drafts: %v", err)
	}
	// a bundle without drafts is not asked about
	w4 := draftsWindow(t)
	p4 := filepath.Join(t.TempDir(), "Plain.accolade")
	os.WriteFile(p4, makeZip(t, bundleItems("")), 0o644)
	w4.LoadFile(p4)
	w4.saveAs(filepath.Join(filepath.Dir(p4), "Plain.fountain"))
	if w4.flattenAsked != nil || w4.bundle != nil {
		t.Error("asked without drafts")
	}
}

func TestFlattenChoicesBrowser(t *testing.T) {
	if c := flattenChoices(true); slices.Contains(c, flattenFolder) || len(c) != 2 {
		t.Errorf("browser %v", c)
	}
	if c := flattenChoices(false); len(c) != 3 {
		t.Errorf("desktop %v", c)
	}
	if _, err := browserSavePath("x.textbundle"); err == nil {
		t.Error("a folder bundle in the browser")
	}
	if _, err := browserSavePath("  "); err == nil {
		t.Error("no name")
	}
}

// Saved as .highland with drafts: the warning, once for the file (the
// file says so: a later session does not warn again); without
// Accolade's files, none.
func TestHighlandWarningOnce(t *testing.T) {
	w, p := bundleWithDraft(t)
	h := filepath.Join(filepath.Dir(p), "Script.highland")
	w.saveAs(h)
	if !w.highlandWarned[absPath(h)] {
		t.Fatal("no warning")
	}
	data, _ := os.ReadFile(h)
	if info, _ := zipFile(t, data, "Script.textbundle/info.json"); !strings.Contains(info, `"highlandWarned":true`) {
		t.Errorf("info.json %s", info)
	}
	if got, _ := zipFile(t, data, "Script.textbundle/assets/drafts/01-one/text.fountain"); got != bundleScript {
		t.Errorf("drafts not in the .highland: %q", got)
	}
	// opened again: saved and a draft added, no warning
	w2 := againWindow(t, w)
	if err := w2.LoadFile(h); err != nil {
		t.Fatal(err)
	}
	w2.textEditor.SetText(bundleScript + "x\n")
	w2.SaveFile()
	saveDraftNow(t, w2, "Two")
	if len(w2.highlandWarned) != 0 {
		t.Error("warned again")
	}
	// a .highland of Highland's own: no warning on save; the first
	// draft warns
	w3 := draftsWindow(t)
	h3 := filepath.Join(t.TempDir(), "Theirs.highland")
	os.WriteFile(h3, makeZip(t, bundleItems("")), 0o644)
	w3.LoadFile(h3)
	w3.SaveFile()
	if len(w3.highlandWarned) != 0 {
		t.Error("warned without Accolade's files")
	}
	saveDraftNow(t, w3, "Mine")
	if !w3.highlandWarned[absPath(h3)] {
		t.Error("the first draft in a .highland did not warn")
	}
	data, _ = os.ReadFile(h3)
	if info, _ := zipFile(t, data, "info.json"); !strings.Contains(info, `"highlandWarned":true`) {
		t.Errorf("info.json %s", info)
	}
}

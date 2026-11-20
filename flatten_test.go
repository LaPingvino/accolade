package main

import (
	"os"
	"path/filepath"
	"testing"
)

// bundleWithDraft is a window with Script.accolade open, holding a draft.
func bundleWithDraft(t *testing.T) (*MainWindow, string) {
	t.Helper()
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	saveDraftNow(t, w, "One")
	return w, p
}

// saveAsPlain saves the window's script as a plain file, as Save As does.
func saveAsPlain(t *testing.T, w *MainWindow, to string, how flattenChoice) {
	t.Helper()
	from := w.currentFile
	w.flattenTo = how
	if err := w.saveToFile(to); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename(from, from, to, to)
	w.currentFile = to
}

// Flattened to a kept copy: the plain file finds its drafts again by its
// path; saved as .accolade again they go back in.
func TestFlattenKept(t *testing.T) {
	w, p := bundleWithDraft(t)
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	saveAsPlain(t, w, to, flattenKept)
	if w.bundle != nil || w.flattenTo != flattenKept {
		t.Fatal("still a bundle")
	}
	if b, _ := os.ReadFile(to); string(b) != bundleScript {
		t.Errorf("plain %q", b)
	}
	if _, err := os.Stat(productionFolderOf(to)); err == nil {
		t.Error("a folder beside")
	}
	w2 := againWindow(t, w)
	if err := w2.LoadFile(to); err != nil {
		t.Fatal(err)
	}
	if h := w2.extrasHome(); h.kind != extrasKept {
		t.Errorf("home %+v", h)
	}
	if ds := draftsOf(t, w2); len(ds) != 1 || ds[0].Name != "One" {
		t.Errorf("reopened %+v", ds)
	}
	// and back into a bundle
	back := filepath.Join(filepath.Dir(p), "Back.accolade")
	if err := w2.saveToFile(back); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(back)
	if got, _ := zipFile(t, data, "Back.textbundle/assets/drafts/01-one/text.fountain"); got != bundleScript {
		t.Errorf("back in: %q", got)
	}
}

// Flattened to Name.production/ beside the file: the bundle's assets
// unzipped; found again from the folder.
func TestFlattenFolder(t *testing.T) {
	w, p := bundleWithDraft(t)
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	saveAsPlain(t, w, to, flattenFolder)
	folder := filepath.Join(filepath.Dir(p), "Script.production")
	if b, err := os.ReadFile(filepath.Join(folder, "assets", "drafts", "01-one", "text.fountain")); err != nil || string(b) != bundleScript {
		t.Fatalf("folder: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(folder, "assets", "drafts", "drafts.json")); err != nil {
		t.Error("no index in the folder")
	}
	if w.chosenExtras(to) != "" {
		t.Error("a kept copy remembered too")
	}
	w2 := againWindow(t, w)
	if err := w2.LoadFile(to); err != nil {
		t.Fatal(err)
	}
	if h := w2.extrasHome(); h.kind != extrasBeside || string(h.dir) != folder {
		t.Errorf("home %+v", h)
	}
	// a draft saved now goes into the folder
	saveDraftNow(t, w2, "Two")
	if _, err := os.Stat(filepath.Join(folder, "assets", "drafts", "02-two")); err == nil {
		t.Error("the same text stored twice")
	}
	if ds := draftsOf(t, w2); len(ds) != 2 {
		t.Errorf("drafts %+v", ds)
	}
	// renamed: the folder follows
	if err := w2.renameTo("Renamed.fountain"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(p), "Renamed.production", "assets", "drafts", "drafts.json")); err != nil {
		t.Errorf("the folder did not follow the rename: %v", err)
	}
}

// Left out: nothing kept anywhere.
func TestFlattenDrop(t *testing.T) {
	w, p := bundleWithDraft(t)
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	saveAsPlain(t, w, to, flattenDrop)
	if h := w.extrasHome(); h.kind != extrasNone {
		t.Errorf("home %+v", h)
	}
	if w.hasProductionData() {
		t.Error("production data left")
	}
}

// Flattened again under the same name with another choice: the choice
// made last is what the plain file finds.
func TestFlattenTwice(t *testing.T) {
	w, p := bundleWithDraft(t)
	to := filepath.Join(filepath.Dir(p), "Script.fountain")
	saveAsPlain(t, w, to, flattenKept)
	kept := w.chosenExtras(to)
	if kept == "" {
		t.Fatal("not kept")
	}
	for _, how := range []flattenChoice{flattenFolder, flattenDrop} {
		w2 := againWindow(t, w)
		if err := w2.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		saveAsPlain(t, w2, to, how)
		if w2.chosenExtras(to) != "" {
			t.Errorf("%d: the old kept copy is still remembered", how)
		}
		if _, err := os.Stat(kept); err == nil {
			t.Errorf("%d: the old kept copy is still there", how)
		}
	}
	w3 := againWindow(t, w)
	w3.LoadFile(to)
	if h := w3.extrasHome(); h.kind != extrasBeside { // the folder made by the first of the two
		t.Errorf("home %+v", h)
	}
}

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// draftsWindow is a window with XDG_DATA_HOME of its own (the kept
// copies).
func draftsWindow(t *testing.T) *MainWindow {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	return w
}

// againWindow is another window with the same settings (a script opened
// again later).
func againWindow(t *testing.T, w *MainWindow) *MainWindow {
	t.Helper()
	w2 := NewMainWindow(w.app)
	t.Cleanup(w2.stopTimers)
	return w2
}

// saveDraftNow saves a draft and fails the test on an error.
func saveDraftNow(t *testing.T, w *MainWindow, name string) (savedDraft, bool) {
	t.Helper()
	var got savedDraft
	var added, called bool
	w.addDraft(name, false, func(d savedDraft, ok bool, err error) {
		if err != nil {
			t.Fatalf("draft %q: %v", name, err)
		}
		got, added, called = d, ok, true
	})
	if !called {
		t.Fatal("the draft was not saved at once")
	}
	return got, added
}

func draftsOf(t *testing.T, w *MainWindow) []savedDraft {
	t.Helper()
	ds, err := readDrafts(w.extrasHome())
	if err != nil {
		t.Fatal(err)
	}
	return ds
}

// In an .accolade: drafts under assets/drafts/, an index naming each
// one's text; a text (and a source) stored once; the bundle's other
// entries copied through byte for byte; info.json's key names the index.
func TestDraftsInBundle(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	orig := makeZip(t, bundleItems("Script.textbundle/"))
	os.WriteFile(p, orig, 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.hasProductionData() {
		t.Error("production data before any draft")
	}
	d1, _ := saveDraftNow(t, w, "First Draft!")
	if d1.ID != "01-first-draft" || d1.Text != "assets/drafts/01-first-draft/text.fountain" || d1.Source != "" || d1.Kind != kindDraft {
		t.Errorf("first: %+v", d1)
	}
	// the same text again: no new file
	d2, _ := saveDraftNow(t, w, "Same")
	if d2.Text != d1.Text || d2.ID != "02-same" {
		t.Errorf("same text stored again: %+v", d2)
	}
	// with a source: kept with it; once only
	story := filepath.Join(t.TempDir(), "novel.md")
	os.WriteFile(story, []byte("# Novel\n\nNight.\n"), 0o644)
	if err := w.useSource(story); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetText(bundleScript + "Bram eats.\n")
	d3, _ := saveDraftNow(t, w, "Über Café")
	if d3.ID != "03-uber-cafe" || d3.Source != "assets/drafts/03-uber-cafe/source.md" || d3.Text != "assets/drafts/03-uber-cafe/text.fountain" {
		t.Errorf("third: %+v", d3)
	}
	w.textEditor.SetText(bundleScript + "Bram sleeps.\n")
	d4, _ := saveDraftNow(t, w, "Fourth")
	if d4.Source != d3.Source || d4.Text == d3.Text {
		t.Errorf("fourth: %+v", d4)
	}
	data, _ := os.ReadFile(p)
	// what was there, as it was
	before := rawEntries(t, orig)
	after := rawEntries(t, data)
	for _, b := range before {
		i := slices.IndexFunc(after, func(a rawEntry) bool { return a.name == b.name })
		if i < 0 {
			t.Errorf("%s gone", b.name)
			continue
		}
		if strings.HasSuffix(b.name, "info.json") {
			continue
		}
		if a := after[i]; a.crc != b.crc || string(a.raw) != string(b.raw) {
			t.Errorf("%s changed", b.name)
		}
	}
	var names []string
	for _, a := range after {
		if strings.Contains(a.name, "assets/drafts/") {
			names = append(names, strings.TrimPrefix(a.name, "Script.textbundle/"))
		}
	}
	want := []string{"assets/drafts/01-first-draft/text.fountain", "assets/drafts/drafts.json",
		"assets/drafts/03-uber-cafe/text.fountain", "assets/drafts/03-uber-cafe/source.md", "assets/drafts/04-fourth/text.fountain"}
	slices.Sort(names)
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("drafts' files %v", names)
	}
	if got, _ := zipFile(t, data, "Script.textbundle/assets/drafts/01-first-draft/text.fountain"); got != bundleScript {
		t.Errorf("first's text %q", got)
	}
	info, _ := zipFile(t, data, "Script.textbundle/info.json")
	if !strings.Contains(info, `"drafts":"assets/drafts/drafts.json"`) {
		t.Errorf("info.json %s", info)
	}
	// the text in the bundle is as it was saved last: a draft is no save
	if got, _ := zipFile(t, data, "Script.textbundle/text.fountain"); got != bundleScript {
		t.Errorf("text %q", got)
	}
	// opened again: the drafts are there, in order
	w2 := againWindow(t, w)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	ds := draftsOf(t, w2)
	if len(ds) != 4 || ds[0].Name != "First Draft!" || ds[3].Name != "Fourth" || !w2.hasProductionData() {
		t.Errorf("reopened: %+v", ds)
	}
	// a save keeps them
	w2.textEditor.SetText("INT. X - DAY\n")
	if err := w2.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if ds := draftsOf(t, w2); len(ds) != 4 {
		t.Errorf("after a save: %d drafts", len(ds))
	}
}

func TestDraftSlug(t *testing.T) {
	for in, want := range map[string]string{
		"First Draft!":          "first-draft",
		"  ":                    "draft",
		"Über Café":             "uber-cafe",
		"Черновик 2":            "2",
		"a/../b":                "a-b",
		strings.Repeat("x", 60): strings.Repeat("x", 40),
	} {
		if got := draftSlug(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

// A plain script's drafts are kept by Accolade, found again by the
// script's path, carried into a bundle saved from it; Save As .fountain
// elsewhere takes their memory along.
func TestDraftsOfPlainScript(t *testing.T) {
	w := draftsWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	saveDraftNow(t, w, "One")
	if h := w.extrasHome(); h.kind != extrasKept || !isKeptExtras(string(h.dir)) {
		t.Fatalf("home %+v", h)
	}
	if !w.hasProductionData() {
		t.Error("no production data with a draft")
	}
	w2 := againWindow(t, w)
	if err := w2.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if ds := draftsOf(t, w2); len(ds) != 1 || ds[0].Name != "One" {
		t.Errorf("reopened: %+v", ds)
	}
	// Save As another .fountain: the drafts follow
	q := filepath.Join(dir, "story2.fountain")
	if err := w.saveToFile(q); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename(p, p, q, q)
	w.currentFile = q
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("after Save As: %+v", ds)
	}
	// Save As .accolade: carried in
	a := filepath.Join(dir, "story.accolade")
	if err := w.saveToFile(a); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename(q, q, a, a)
	w.currentFile = a
	if w.bundle == nil || len(w.bundle.extras) != 2 {
		t.Fatalf("bundle %+v", w.bundle)
	}
	if ds := draftsOf(t, w); len(ds) != 1 || ds[0].Name != "One" {
		t.Errorf("in the bundle: %+v", ds)
	}
	data, _ := os.ReadFile(a)
	if info, _ := zipFile(t, data, "story.textbundle/info.json"); !strings.Contains(info, `"drafts":"assets/drafts/drafts.json"`) {
		t.Errorf("info.json %s", info)
	}
}

// An unsaved script's drafts: kept, carried into its first save, let go of
// by New.
func TestDraftsOfUnsavedScript(t *testing.T) {
	w := draftsWindow(t)
	w.textEditor.SetText(bundleScript)
	saveDraftNow(t, w, "Early")
	kept := w.unsavedExtras
	if kept == "" {
		t.Fatal("no kept copy")
	}
	a := filepath.Join(t.TempDir(), "new.accolade")
	if err := w.saveToFile(a); err != nil {
		t.Fatal(err)
	}
	w.typeAfterRename("", "untitled.fountain", a, a)
	w.currentFile = a
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("carried: %+v", ds)
	}
	if _, err := os.Stat(kept); err == nil || w.unsavedExtras != "" {
		t.Error("the unsaved copy is still there")
	}
	// New lets go of an unsaved one's
	w3 := draftsWindow(t)
	saveDraftNow(t, w3, "Gone")
	kept = w3.unsavedExtras
	w3.newFile(true)
	if _, err := os.Stat(kept); err == nil {
		t.Error("New kept the unsaved drafts")
	}
}

// Restore: the script now is kept first, the draft's text comes back in
// one undo step, and is kept as a new draft; nothing is overwritten.
func TestRestoreDraft(t *testing.T) {
	w := draftsWindow(t)
	p := filepath.Join(t.TempDir(), "Script.accolade")
	os.WriteFile(p, makeZip(t, bundleItems("")), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	d1, _ := saveDraftNow(t, w, "Old")
	w.textEditor.SetText(bundleScript + "New line.\n")
	var rerr error
	w.restoreDraft(d1, func(err error) { rerr = err })
	if rerr != nil {
		t.Fatal(rerr)
	}
	if w.textEditor.Text() != bundleScript {
		t.Errorf("restored %q", w.textEditor.Text())
	}
	ds := draftsOf(t, w)
	if len(ds) != 3 || ds[1].Name != "Before restoring Old" || ds[2].Name != "Old (restored)" || ds[2].Text != d1.Text {
		t.Fatalf("drafts %+v", ds)
	}
	if got, _ := w.extrasHome().readExtra(ds[1].Text); string(got) != bundleScript+"New line.\n" {
		t.Errorf("the text before: %q", got)
	}
	w.undo()
	if w.textEditor.Text() != bundleScript+"New line.\n" {
		t.Errorf("undone: %q", w.textEditor.Text())
	}
	// restoring when the script is a draft already: no "before" draft
	w.restoreDraft(ds[1], nil)
	if n := len(draftsOf(t, w)); n != 4 {
		t.Errorf("%d drafts", n)
	}
}

// newDraft with skipSame adds nothing for a text stored already.
func TestNewDraftSkipSame(t *testing.T) {
	dir := extrasFolder(t.TempDir())
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	_, files, ok, err := newDraft(dir, "", "", []byte("A"), ".fountain", nil, now, false)
	if err != nil || !ok || len(files) != 2 {
		t.Fatalf("%v %v %v", files, ok, err)
	}
	dir.write(files)
	if _, files, ok, _ := newDraft(dir, "x", "", []byte("A"), ".fountain", nil, now, true); ok || files != nil {
		t.Errorf("same text added: %v", files)
	}
	if _, _, ok, _ := newDraft(dir, "x", "", []byte("A"), ".fountain", []byte("src"), now, true); !ok {
		t.Error("a new source not added")
	}
	ds, _ := readDrafts(dir)
	if len(ds) != 1 || ds[0].Name != "Draft 1" || !ds[0].Date.Equal(now) {
		t.Errorf("%+v", ds)
	}
}

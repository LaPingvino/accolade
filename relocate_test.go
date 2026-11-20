package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// A name's parts: base, language, extension; a full name typed is taken
// as it is.
func TestScriptNameParts(t *testing.T) {
	for _, c := range []struct {
		name string
		want nameParts
	}{
		{"film.fountain", nameParts{"film", "", ".fountain"}},
		{"film.eo.fountain", nameParts{"film", "eo", ".fountain"}},
		{"Kuiro.NL.accolade", nameParts{"Kuiro", "nl", ".accolade"}},
		{"Mr. Smith", nameParts{"Mr. Smith", "", ""}},
		{"notes.txt", nameParts{"notes", "", ".txt"}},
		{"a.b.textbundle", nameParts{"a.b", "", ".textbundle"}},
	} {
		if got := splitScriptName(c.name); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
		if got := splitScriptName(c.name).String(); !strings.EqualFold(got, c.name) {
			t.Errorf("%s comes back as %s", c.name, got)
		}
	}
	if got := composedName("film", typeAccolade); got != "film.accolade" {
		t.Errorf("composed %s", got)
	}
	if got := composedName("film", dialectType("eo")); got != "film.eo.fountain" {
		t.Errorf("a dialect: %s", got)
	}
	if got := composedName(" story.md ", typeAccolade); got != "story.md" {
		t.Errorf("a full name typed: %s", got)
	}
	if got := composedName("film.eo.accolade", typeAccolade); got != "film.accolade" {
		t.Errorf("a bundle's name says no language: %s", got)
	}
	if got := composedName("Mr. Smith", typeFountain); got != "Mr. Smith.fountain" {
		t.Errorf("a dot in the name: %s", got)
	}
}

// The types offered: the document's own first, the bundles, the other
// kind last; no folder in the browser; another plain extension kept as it
// is; each says what it keeps.
func TestScriptTypes(t *testing.T) {
	ids := func(ts []fileType) string {
		var s []string
		for _, t := range ts {
			s = append(s, t.id)
		}
		return strings.Join(s, " ")
	}
	if got := ids(scriptTypes(false, false, "a.fountain", "en")); got != "fountain accolade highland textpack textbundle markdown" {
		t.Errorf("screenplay: %s", got)
	}
	if got := ids(scriptTypes(true, true, "a.md", "eo")); got != "markdown accolade highland textpack fountain" {
		t.Errorf("Markdown in the browser: %s", got)
	}
	ts := scriptTypes(false, false, "a.txt", "en")
	if ts[0].id != "asis" || ts[0].ext != ".txt" {
		t.Errorf("a .txt: %+v", ts[0])
	}
	for _, md := range []bool{false, true} {
		for _, ty := range scriptTypes(md, false, "a.txt", "eo") {
			if ty.keeps(md, false, "eo") == "" {
				t.Errorf("%s says nothing", ty.id)
			}
		}
	}
	if !strings.Contains(typeHighland.keeps(false, false, "en"), "may drop") {
		t.Error("Highland's warning is missing")
	}
	// a script in Esperanto: standard Fountain and its dialect, each named
	eo := scriptTypes(false, false, "film.eo.fountain", "eo")
	if got := ids(eo); got != "fountain dialect accolade highland textpack textbundle markdown" {
		t.Errorf("Esperanto: %s", got)
	}
	if eo[1].label != "Esperanto Fountain" || eo[1].shownExt() != ".eo.fountain" || typeOfName("film.eo.fountain", false) != eo[1] {
		t.Errorf("the dialect: %+v", eo[1])
	}
	if !strings.Contains(eo[1].keeps(false, false, "eo"), "won't see") || !strings.Contains(eo[0].keeps(false, false, "eo"), "/* Language: eo */") {
		t.Errorf("their lines: %q, %q", eo[1].keeps(false, false, "eo"), eo[0].keeps(false, false, "eo"))
	}
	if recommendedType(false, true) != typeAccolade || recommendedType(false, false) != typeFountain || recommendedType(true, true) != typeMarkdown {
		t.Error("recommended")
	}
}

// relocWindow is story.fountain with story.md beside it and its drafts in
// story.production/ beside it.
func relocWindow(t *testing.T) (*MainWindow, string) {
	t.Helper()
	w := draftsWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	os.WriteFile(filepath.Join(dir, "story.md"), []byte(sourceStory), 0o644)
	if err := os.MkdirAll(productionFolderOf(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	saveDraftNow(t, w, "One")
	if h := w.extrasHome(); h.kind != extrasBeside {
		t.Fatalf("drafts %+v", h)
	}
	return w, p
}

// visible are a folder's entries, the hidden (parked) ones left out.
func visible(t *testing.T, dir string) string {
	t.Helper()
	es, _ := os.ReadDir(dir)
	var names []string
	for _, e := range es {
		if !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return strings.Join(names, " ")
}

// A plain rename takes the script's files along: story.md and
// story.production/ follow the name; Undo Rename puts all back.
func TestRenameTakesFilesAlong(t *testing.T) {
	w, p := relocWindow(t)
	dir := filepath.Dir(p)
	rows := w.planRows(relocation{rename: true, from: p, to: filepath.Join(dir, "film.fountain")})
	var said []string
	for _, r := range rows {
		said = append(said, r.what+": "+r.now+" → "+r.after)
	}
	plan := strings.Join(said, "\n")
	for _, want := range []string{"Script: story.fountain → film.fountain", "Source: story.md beside it → film.md beside it", "1 draft: story.production/ beside it → film.production/ beside it"} {
		if !strings.Contains(plan, want) {
			t.Errorf("the plan lacks %q:\n%s", want, plan)
		}
	}
	if err := w.renameTo("film.fountain"); err != nil {
		t.Fatal(err)
	}
	if got := visible(t, dir); got != "film.fountain film.md film.production" {
		t.Errorf("after the rename: %s", got)
	}
	if r := w.sourceRef(); r.Origin != sourceBeside || filepath.Base(r.Path) != "film.md" {
		t.Errorf("source %+v", r)
	}
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("drafts %+v", ds)
	}
	if !w.cmdEnv().renamed {
		t.Fatal("Undo Rename cannot be given")
	}
	w.undoRename()
	if got := visible(t, dir); got != "story.fountain story.md story.production" {
		t.Errorf("after Undo Rename: %s", got)
	}
	if w.currentFile != p || w.cmdEnv().renamed {
		t.Errorf("window at %s", w.currentFile)
	}
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("drafts after undo %+v", ds)
	}
}

// Keep the source's own name: story.md stays, remembered for film.fountain.
func TestRenameKeepsSourceName(t *testing.T) {
	w, p := relocWindow(t)
	dir := filepath.Dir(p)
	if err := w.relocate(relocation{rename: true, from: p, to: filepath.Join(dir, "film.fountain"), keepBesideName: true}, nil); err != nil {
		t.Fatal(err)
	}
	if got := visible(t, dir); got != "film.fountain film.production story.md" {
		t.Errorf("after the rename: %s", got)
	}
	if r := w.sourceRef(); filepath.Base(r.Path) != "story.md" {
		t.Errorf("source %+v", r)
	}
}

// Renamed into a bundle: the source and drafts move inside it; the files
// left behind are parked, and Undo Rename brings them back.
func TestRenameIntoBundle(t *testing.T) {
	w, p := relocWindow(t)
	dir := filepath.Dir(p)
	to := filepath.Join(dir, "film.accolade")
	plan := w.planRows(relocation{rename: true, from: p, to: to})
	for _, r := range plan {
		if r.what == bundleSourceName && (!r.fixed || r.after != "inside film.accolade") {
			t.Errorf("source row %+v", r)
		}
	}
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	if got := visible(t, dir); got != "film.accolade" {
		t.Errorf("after the rename: %s", got)
	}
	data, _ := os.ReadFile(to)
	if src, ok := zipFile(t, data, "film.textbundle/assets/source.md"); !ok || src != sourceStory {
		t.Errorf("source in the bundle: %v %q", ok, src)
	}
	if r := w.sourceRef(); r.Origin != sourceBundle {
		t.Errorf("source %+v", r)
	}
	if ds := draftsOf(t, w); len(ds) != 1 || w.extrasHome().kind != extrasBundle {
		t.Errorf("drafts %+v", ds)
	}
	w.undoRename()
	if got := visible(t, dir); got != "story.fountain story.md story.production" {
		t.Errorf("after Undo Rename: %s", got)
	}
	if w.bundle != nil || w.currentFile != p || w.sourceRef().Origin != sourceBeside {
		t.Errorf("window: %s, bundle %v, source %+v", w.currentFile, w.bundle, w.sourceRef())
	}
}

// Renamed out of a bundle, its drafts to the folder beside: film.fountain,
// film.md and film.production/; Undo Rename brings the bundle back with
// every entry of another app's.
func TestRenameOutOfBundle(t *testing.T) {
	w, p := bundleWithDraft(t)
	if err := w.useSourceText(filepath.Join(t.TempDir(), "novel.md"), sourceStory); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(p)
	before, _ := os.ReadFile(p)
	r := relocation{rename: true, from: p, to: filepath.Join(dir, "film.fountain"), how: flattenFolder}
	var others bool
	for _, row := range w.planRows(r) {
		if strings.Contains(row.what, "other files") && row.warn {
			others = true
		}
	}
	if !others {
		t.Error("the plan does not say another app's files are left out")
	}
	if err := w.relocate(r, nil); err != nil {
		t.Fatal(err)
	}
	if got := visible(t, dir); got != "film.fountain film.md film.production" {
		t.Errorf("after the rename: %s", got)
	}
	if md, _ := os.ReadFile(filepath.Join(dir, "film.md")); string(md) != sourceStory {
		t.Errorf("source beside: %q", md)
	}
	if ds := draftsOf(t, w); len(ds) != 1 || w.extrasHome().kind != extrasBeside {
		t.Errorf("drafts %+v", ds)
	}
	w.undoRename()
	if got := visible(t, dir); got != "Script.accolade" {
		t.Errorf("after Undo Rename: %s", got)
	}
	if after, _ := os.ReadFile(p); string(after) != string(before) {
		t.Error("the bundle did not come back as it was")
	}
	if w.bundle == nil || len(w.bundle.extras) == 0 || w.sourceRef().Origin != sourceBundle {
		t.Errorf("window: bundle %v, source %+v", w.bundle, w.sourceRef())
	}
}

// A zip renamed to a folder bundle and back: converted, never refused.
func TestRenameZipToFolder(t *testing.T) {
	w, p := bundleWithDraft(t)
	dir := filepath.Dir(p)
	if err := w.renameTo("Script.textbundle"); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(dir, "Script.textbundle", "sprints.json")); err != nil || st.IsDir() {
		t.Errorf("another app's file not carried: %v", err)
	}
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("drafts %+v", ds)
	}
	if err := w.renameTo("Script.textpack"); err != nil {
		t.Fatal(err)
	}
	if got := visible(t, dir); got != "Script.textpack" {
		t.Errorf("after the renames: %s", got)
	}
	if ds := draftsOf(t, w); len(ds) != 1 {
		t.Errorf("drafts %+v", ds)
	}
}

// All or nothing: a rename that fails half way leaves everything as it
// was (here the drafts' folder cannot be made: a file has its name).
func TestRenameFailsWhole(t *testing.T) {
	w, p := bundleWithDraft(t)
	if err := w.useSourceText(filepath.Join(t.TempDir(), "novel.md"), sourceStory); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(p)
	os.WriteFile(filepath.Join(dir, "film.production"), []byte("in the way"), 0o644)
	before, _ := os.ReadFile(p)
	err := w.relocate(relocation{rename: true, from: p, to: filepath.Join(dir, "film.fountain"), how: flattenFolder}, nil)
	if err == nil {
		t.Fatal("no error")
	}
	if got := visible(t, dir); got != "Script.accolade film.production" {
		t.Errorf("after the failure: %s", got)
	}
	if after, _ := os.ReadFile(p); string(after) != string(before) {
		t.Error("the bundle changed")
	}
	if w.currentFile != p || w.bundle == nil || w.sourceRef().Origin != sourceBundle || w.cmdEnv().renamed {
		t.Errorf("window: %s, bundle %v", w.currentFile, w.bundle)
	}
}

// Save As copies: the old file keeps its own files, the new one gets
// copies of them.
func TestSaveAsCopiesFiles(t *testing.T) {
	w, p := relocWindow(t)
	dir := filepath.Dir(p)
	w.finishSaveAs(filepath.Join(dir, "film.fountain"), relocation{})
	if got := visible(t, dir); got != "film.fountain film.md film.production story.fountain story.md story.production" {
		t.Errorf("after Save As: %s", got)
	}
	if w.cmdEnv().renamed {
		t.Error("a Save As to take back")
	}
}

// The rename editor: base, language and type make the name, live; a full
// name typed sets the pickers; the plan follows; Rename does it.
func TestRenameEditor(t *testing.T) {
	w, p := relocWindow(t)
	w.renameScript()
	a := w.renameAsked
	if a == nil || a.base.Text != "story" || a.kind.Selected != "Fountain (.fountain)" || a.lang.Selected != languageLabel("en") {
		t.Fatalf("opened as %+v", a)
	}
	a.base.SetText("film")
	a.lang.SetSelected(languageLabel("eo"))
	if !slices.Contains(a.kind.Options, "Esperanto Fountain (.eo.fountain)") || a.kind.Selected != "Fountain (.fountain)" {
		t.Errorf("types in Esperanto: %v, %q", a.kind.Options, a.kind.Selected)
	}
	a.kind.SetSelected("Accolade (.accolade)")
	if a.result.Text != "Becomes: film.accolade  · Esperanto, in info.json, its text as text.eo.fountain" {
		t.Errorf("result %q", a.result.Text)
	}
	plan := planText(a.plan)
	for _, want := range []string{"assets/source.md", "Language: English → Esperanto", "text.eo.fountain, text.fountain (standard Fountain)"} {
		if !strings.Contains(plan, want) {
			t.Errorf("plan has no %q:\n%s", want, plan)
		}
	}
	a.base.SetText("draft.md")
	if a.kind.Selected != "Markdown (.md)" || a.result.Text != "Becomes: draft.md" {
		t.Errorf("typed a full name: %q %q", a.kind.Selected, a.result.Text)
	}
	a.base.SetText("film.nl.fountain") // a dialect's full name: its language picked
	if a.lang.Selected != languageLabel("nl") || a.kind.Selected != "Nederlands Fountain (.nl.fountain)" {
		t.Errorf("typed a dialect's name: %q %q", a.lang.Selected, a.kind.Selected)
	}
	a.base.SetText("film")
	a.lang.SetSelected(languageLabel("eo"))
	if a.kind.Selected != "Esperanto Fountain (.eo.fountain)" {
		t.Errorf("the dialect follows the language: %q", a.kind.Selected)
	}
	a.confirm()
	if w.currentFile != filepath.Join(filepath.Dir(p), "film.eo.fountain") || w.language != "eo" {
		t.Errorf("renamed to %s (%s)", w.currentFile, w.language)
	}
}

// planText is what a plan panel shows.
func planText(p *planPanel) string {
	var s []string
	for _, o := range p.rows.Objects {
		for _, l := range labelsIn(o) {
			s = append(s, l)
		}
	}
	return strings.Join(s, "\n")
}

// The Save As chooser: the type now chosen, the recommended one marked,
// each saying what it keeps; the plan follows the type.
func TestSaveAsChooser(t *testing.T) {
	w, _ := relocWindow(t)
	w.showSaveAs()
	a := w.saveAsAsked
	if a == nil || a.chosen().id != "fountain" {
		t.Fatalf("chosen %+v", a)
	}
	var marked []string
	for _, r := range a.radios {
		if strings.Contains(r.Options[0], "recommended") {
			marked = append(marked, r.Options[0])
		}
	}
	if !slices.Equal(marked, []string{"Accolade (.accolade) — recommended"}) { // it has drafts
		t.Errorf("marked %v", marked)
	}
	if a.radios[0].Selected == "" || a.radios[1].Selected != "" {
		t.Error("the chosen radio")
	}
	a.choose(1)
	if a.chosen().id != "accolade" || a.radios[0].Selected != "" || !strings.Contains(planText(a.plan), "inside story.accolade") {
		t.Errorf("chose Accolade:\n%s", planText(a.plan))
	}
}

// labelsIn are the labels' texts in o and what it holds.
func labelsIn(o fyne.CanvasObject) []string {
	switch v := o.(type) {
	case *widget.Label:
		return []string{v.Text}
	case *fyne.Container:
		var s []string
		for _, c := range v.Objects {
			s = append(s, labelsIn(c)...)
		}
		return s
	}
	return nil
}

// With the bundle written in the background as in the app: the rename
// into a bundle waits for a save under way, finishes once the bundle is
// written, and Undo Rename then puts the plain files back.
func TestRenameIntoBundleInBackground(t *testing.T) {
	holdUI(t)
	w, p := relocWindow(t)
	dir := filepath.Dir(p)
	realPaths(t)
	w.textEditor.SetText(bundleScript + "Bram eats.\n")
	w.queueBundle(bundleJob{run: func(b *textBundle) (*textBundle, error) { // a write under way
		time.Sleep(50 * time.Millisecond)
		return b, nil
	}})
	var outcome error
	finished := false
	if err := w.relocate(relocation{rename: true, from: w.currentFile, to: filepath.Join(dir, "film.accolade")}, func(err error) {
		outcome, finished = err, true
	}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the rename", func() bool { return finished && !w.bundleBusy() })
	if outcome != nil {
		t.Fatal(outcome)
	}
	if got := visible(t, dir); got != "film.accolade" {
		t.Errorf("after the rename: %s", got)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "film.accolade"))
	if got, _ := zipFile(t, data, "film.textbundle/text.fountain"); got != bundleScript+"Bram eats.\n" {
		t.Errorf("text %q", got)
	}
	w.undoRename()
	waitFor(t, "the undo", func() bool { return !w.bundleBusy() && w.renameUndo == nil })
	if got := visible(t, dir); got != "story.fountain story.md story.production" {
		t.Errorf("after Undo Rename: %s", got)
	}
	if !w.hasChanges {
		t.Error("the text written into the bundle is not marked unsaved again")
	}
}

// Screenshots of the Save As chooser and the rename editor, on a desktop
// and a phone (SHOT_DIR=… go test -run TestSaveAsScreenshots): saveas-*.png.
func TestSaveAsScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		// Save As of an .accolade with a draft and a source, Fountain chosen
		w, _ := bundleWithDraft(t)
		w.useSourceText(filepath.Join(t.TempDir(), "novel.md"), sourceStory)
		w.fyneWindow.Resize(size.size)
		w.showSaveAs()
		w.saveAsAsked.choose(indexOfType(w.saveAsAsked.types, typeFountain))
		capture(t, w.fyneWindow.Canvas(), "saveas-chooser-"+size.name+".png")
		w.saveAsAsked.scroll.ScrollToBottom() // its files
		capture(t, w.fyneWindow.Canvas(), "saveas-chooser-files-"+size.name+".png")
		// the rename editor: story.fountain to film.eo.accolade
		w2, _ := relocWindow(t)
		w2.fyneWindow.Resize(size.size)
		w2.renameScript()
		a := w2.renameAsked
		a.base.SetText("film")
		a.lang.SetSelected(languageLabel("eo"))
		a.kind.SetSelected("Accolade (.accolade)")
		capture(t, w2.fyneWindow.Canvas(), "saveas-rename-"+size.name+".png")
		a.scroll.ScrollToBottom()
		capture(t, w2.fyneWindow.Canvas(), "saveas-rename-files-"+size.name+".png")
	}
}

// Save As's help link opens File Formats at its section on saving.
func TestSaveAsHelpLink(t *testing.T) {
	w, _ := relocWindow(t)
	w.showSaveAs()
	w.showHelp("formats", "save-as-and-rename")
	if w.notes == nil || w.notes.help == nil || w.notes.help.anchors["save-as-and-rename"] == nil {
		t.Error("no section save-as-and-rename in File Formats")
	}
}

// copyPath copies a folder with what is in it: movePath's way across file
// systems, where a rename cannot go.
func TestCopyPath(t *testing.T) {
	src := filepath.Join(t.TempDir(), "story.production")
	os.MkdirAll(filepath.Join(src, "assets", "drafts", "01-one"), 0o755)
	os.WriteFile(filepath.Join(src, "assets", "drafts", "01-one", "text.fountain"), []byte(bundleScript), 0o644)
	dst := filepath.Join(t.TempDir(), "film.production")
	if err := copyPath(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "assets", "drafts", "01-one", "text.fountain")); string(got) != bundleScript {
		t.Errorf("copied %q", got)
	}
	if err := copyPath(src, dst); err == nil {
		t.Error("copied over what is there")
	}
}

// A bare name from the save dialog gets the type's extension, the empty
// file the dialog made gone; a name with one stays.
func TestWithTypeExt(t *testing.T) {
	dir := t.TempDir()
	bare := filepath.Join(dir, "film")
	os.WriteFile(bare, nil, 0o644)
	if got := withTypeExt(bare, typeAccolade); got != bare+".accolade" || exists(bare) {
		t.Errorf("bare: %s, left %v", got, exists(bare))
	}
	if got := withTypeExt(filepath.Join(dir, "film.md"), typeAccolade); got != filepath.Join(dir, "film.md") {
		t.Errorf("with one: %s", got)
	}
}

// Undo Rename after a conversion will not lose a draft saved since.
func TestUndoRenameKeepsLaterWork(t *testing.T) {
	w, p := relocWindow(t)
	if err := w.renameTo("film.accolade"); err != nil {
		t.Fatal(err)
	}
	saveDraftNow(t, w, "Two")
	w.undoRename()
	if w.currentFile == p || len(draftsOf(t, w)) != 2 {
		t.Errorf("taken back over a later draft: %s, %d drafts", w.currentFile, len(draftsOf(t, w)))
	}
}

// A .highland of plain text renamed to another .highland stays plain.
func TestRenamePlainHighland(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "Plain.highland")
	os.WriteFile(p, []byte(bundleScript), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if err := w.renameTo("Other.highland"); err != nil {
		t.Fatal(err)
	}
	q := filepath.Join(dir, "Other.highland")
	if got, _ := os.ReadFile(q); string(got) != bundleScript || !w.isPlainBundle(q) || w.bundle != nil {
		t.Errorf("not plain any more: %q", got)
	}
}

// Undo Rename puts back only the script's own remembered entries: what
// another window remembered since stays.
func TestUndoRenameKeepsOthersSettings(t *testing.T) {
	w, p := relocWindow(t)
	if err := w.renameTo("film.fountain"); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "other.fountain")
	novel := filepath.Join(t.TempDir(), "novel.md") // absolute on every system (Windows made "/somewhere" D:\somewhere)
	w.rememberSource(other, novel)
	w.undoRename()
	if w.currentFile != p || w.chosenSource(other) != novel {
		t.Errorf("at %s; the other's source %q", w.currentFile, w.chosenSource(other))
	}
}

// Everything Tab reaches in both dialogs shows its focus.
func TestSaveAsFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race")
	}
	w, _ := relocWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1100, 1500)) // all of it in sight: the scroll does not follow the focus
	w.showSaveAs()
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("Save As: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	w2, _ := relocWindow(t)
	w2.fyneWindow.Resize(fyne.NewSize(1100, 1500))
	w2.renameScript()
	if u := invisibleFocus(w2.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("Rename: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
}

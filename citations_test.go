package main

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
)

const citeBib = `@article{goedel31,
  author = {G{\"o}del, Kurt},
  title = {{\"U}ber formal unentscheidbare S{\"a}tze},
  journal = {Monatshefte f{\"u}r Mathematik and Physik},
  volume = 38, pages = {173--198}, year = 1931,
}
@book{goethe08, author = {Johann Wolfgang von Goethe}, title = {Faust}, year = {1808}, publisher = {Cotta}}
@book{knuth84, author = {Donald E. Knuth}, title = {The {\TeX}book}, year = {1984}}
`

// citeWindow is a Markdown window on doc.md in a folder with refs.bib
// beside it; the cursor at | in text.
func citeWindow(t *testing.T, text string) (*MainWindow, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "refs.bib"), []byte(citeBib), 0o644); err != nil {
		t.Fatal(err)
	}
	w := mdWindow(t, text, -1, -1)
	w.currentFile = filepath.Join(dir, "doc.md")
	return w, dir
}

func citeKeyPress(w *MainWindow, k fyne.KeyName) { w.textEditor.TypedKey(&fyne.KeyEvent{Name: k}) }

// @ lists the library's entries as the key is typed; ↓ and Tab put the
// chosen one in, inside the brackets closed as [ was typed.
func TestCiteCompletion(t *testing.T) {
	w, _ := citeWindow(t, "See |.")
	w.settings.SetBoolean("auto-close-brackets", true)
	typeText(w, "[@go")
	p := w.citePop
	if !p.open() {
		t.Fatal("no list after [@go")
	}
	var keys []string
	for _, e := range p.entries {
		keys = append(keys, e.Key)
	}
	if !reflect.DeepEqual(keys, []string{"goedel31", "goethe08"}) {
		t.Fatalf("listed %v", keys)
	}
	if r := p.shown[0]; r.name.Text != "goedel31" || !strings.Contains(r.desc.Text, "Gödel 1931 · Über formal") {
		t.Errorf("the row: %q %q", r.name.Text, r.desc.Text)
	}
	citeKeyPress(w, fyne.KeyDown)
	citeKeyPress(w, fyne.KeyTab)
	if got := w.textEditor.Text(); got != "See [@goethe08]." {
		t.Fatalf("after Tab: %q", got)
	}
	if off := w.textEditor.CursorOffset(); off != len("See [@goethe08") {
		t.Errorf("cursor at %d", off)
	}
	if p.open() {
		t.Error("the list stays open after a key is put in")
	}
	// on: a locator, then another key
	typeText(w, ", p. 3; @kn")
	if !p.open() || p.entries[0].Key != "knuth84" {
		t.Fatalf("second key: open %v", p.open())
	}
	citeKeyPress(w, fyne.KeyReturn)
	if got := w.textEditor.Text(); got != "See [@goethe08, p. 3; @knuth84]." {
		t.Errorf("after Enter: %q", got)
	}
}

// Esc puts the list away until another @; typing on does not bring it
// back. A tap on a row takes it. In vim's normal mode nothing opens.
func TestCiteCompletionEscTapVim(t *testing.T) {
	w, _ := citeWindow(t, "Text |")
	typeText(w, "@")
	p := w.citePop
	if !p.open() || len(p.entries) != 3 {
		t.Fatalf("@ alone: open %v, %d entries", p.open(), len(p.entries))
	}
	citeKeyPress(w, fyne.KeyEscape)
	typeText(w, "k")
	if p.open() {
		t.Error("the list came back after Esc")
	}
	typeText(w, " and @")
	if !p.open() {
		t.Fatal("no list at another @")
	}
	p.shown[2].Tapped(nil)
	if got := w.textEditor.Text(); got != "Text @k and @knuth84" {
		t.Errorf("after a tap: %q", got)
	}
	// an e-mail address is no citation
	typeText(w, " me@go")
	if p.open() {
		t.Error("listed after me@")
	}
	// vim's normal mode: a change there opens nothing
	w.settings.SetString("editor-keys", "vim")
	w.textEditor.SetText("x @go")
	w.textEditor.SetCursorOffset(5)
	w.vim.setMode(vimNormal)
	w.citeFollow(true)
	if p.open() {
		t.Error("listed in normal mode")
	}
	w.vim.setMode(vimInsert)
	w.citeFollow(true)
	if !p.open() {
		t.Error("not listed in insert mode")
	}
	// the cursor put in a citation opens nothing; moved out, the list closes
	w.settings.SetString("editor-keys", "standard")
	w.textEditor.SetText("x @go y")
	w.textEditor.SetCursorOffset(4)
	if p.open() {
		t.Error("listed for a cursor put in a key")
	}
	typeText(w, "o") // x @go|o: what is before the cursor
	if !p.open() {
		t.Fatal("not listed while typing in a key")
	}
	tabKey(w)
	if got := w.textEditor.Text(); got != "x @goedel31 y" { // the whole key replaced, after the cursor too
		t.Errorf("taken in a key: %q", got)
	}
	typeText(w, "o")
	w.textEditor.SetCursorOffset(0)
	if p.open() {
		t.Error("the list stays open with the cursor away")
	}
}

// The pointer over a citation shows its entry; over one key of several,
// that key's; a key the library lacks says so.
func TestCiteHoverTip(t *testing.T) {
	text := "As [@goedel31, p. 3; @knuth84] and @nobody say."
	w, _ := citeWindow(t, text+"|")
	at := strings.Index(text, "goedel")
	tip := w.citeTipAt(text, at)
	if !strings.Contains(tip, "Kurt Gödel (1931)\nÜber formal unentscheidbare Sätze\nIn Monatshefte für Mathematik and Physik 38: 173–198") || strings.Contains(tip, "Knuth") {
		t.Errorf("over goedel31: %q", tip)
	}
	if tip := w.citeTipAt(text, strings.Index(text, "[")); !strings.Contains(tip, "Gödel") || !strings.Contains(tip, "Knuth") {
		t.Errorf("over the bracket: %q", tip)
	}
	if tip := w.citeTipAt(text, strings.Index(text, "nobody")); tip != "@nobody is not in refs.bib." {
		t.Errorf("unknown: %q", tip)
	}
	if tip := w.citeTipAt(text, strings.Index(text, "say")); tip != "" {
		t.Errorf("off a citation: %q", tip)
	}
	w.noteHover(at, fyne.NewPos(10, 10), fyne.NewSize(8, 16))
	if !strings.Contains(w.noteTipShown, "Gödel") {
		t.Errorf("the hover shows %q", w.noteTipShown)
	}
	// a phone: a tap on a citation shows it, typing does not
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.showEditorTip("", fyne.Position{}, fyne.Size{})
	w.textEditor.SetCursorOffset(at)
	if !strings.Contains(w.noteTipShown, "Gödel") {
		t.Errorf("a tap shows %q", w.noteTipShown)
	}
	typeText(w, "x")
	if w.noteTipShown != "" {
		t.Errorf("typing in a key shows %q", w.noteTipShown)
	}
	// with no library at all
	w.currentFile = filepath.Join(t.TempDir(), "alone.md")
	if tip := w.citeTipAt(text, at); !strings.Contains(tip, "No library") {
		t.Errorf("no library: %q", tip)
	}
}

// The front matter's bibliography comes first; the library is read
// again when its file changes, and a half-written file keeps the one
// read before.
func TestCiteLibraryWatched(t *testing.T) {
	w, dir := citeWindow(t, "|")
	other := filepath.Join(dir, "sub", "zotero.json")
	os.MkdirAll(filepath.Dir(other), 0o755)
	os.WriteFile(other, []byte(`[{"id": "zot1", "title": "From Zotero", "author": [{"family": "Zee"}], "issued": {"date-parts": [[2020]]}}]`), 0o644)
	w.textEditor.SetText("---\nbibliography: sub/zotero.json\n---\n\nText @")
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().Len())
	lib := w.library()
	if lib == nil || lib.Get("zot1") == nil || lib.Get("goedel31") != nil {
		t.Fatalf("the front matter's library is not read: %v", w.libraryNote())
	}
	if n := w.libraryNote(); n != "1 entry in zotero.json (the front matter's bibliography)." {
		t.Errorf("note %q", n)
	}
	again := func() {
		w.citations().checked = time.Time{} // a second has passed
		w.library()
	}
	// Better BibTeX exports again
	os.WriteFile(other, []byte(`[{"id": "zot1", "title": "From Zotero"}, {"id": "zot2", "title": "Second"}]`), 0o644)
	again()
	if w.library().Get("zot2") == nil {
		t.Fatal("the export again is not read")
	}
	// half written: the library before is kept
	os.WriteFile(other, []byte(`[{"id": "zot1", "tit`), 0o644)
	again()
	if lib := w.library(); lib == nil || lib.Get("zot2") == nil {
		t.Error("a half-written file lost the library")
	}
	// a .bib cut short: the same
	bib := filepath.Join(dir, "refs.bib")
	w.textEditor.SetText("---\nbibliography: refs.bib\n---\n\nText @")
	w.library()
	os.WriteFile(bib, []byte(citeBib[:strings.Index(citeBib, "@book{knuth84")+20]), 0o644)
	again()
	if lib := w.library(); lib == nil || lib.Get("goethe08") == nil || len(lib.Entries) != 3 {
		t.Errorf("a .bib cut short lost the library: %s", w.libraryNote())
	}
	os.WriteFile(bib, []byte(citeBib), 0o644)
	// the front matter gone: the .bib beside the document
	w.textEditor.SetText("Text @")
	w.library()
	if lib := w.library(); lib == nil || lib.Get("goedel31") == nil {
		t.Errorf("not the .bib beside it: %s", w.libraryNote())
	}
	// none beside it: Preferences' main library
	os.Remove(filepath.Join(dir, "refs.bib"))
	w.settings.SetString(citeLibraryPref, other)
	t.Cleanup(func() { w.settings.SetString(citeLibraryPref, "") }) // the settings are shared: no main library left for the next tests
	os.WriteFile(other, []byte(`[{"id": "main1", "title": "Main"}]`), 0o644)
	w.library()
	if lib := w.library(); lib == nil || lib.Get("main1") == nil {
		t.Errorf("not the main library: %s", w.libraryNote())
	}
}

// A bundle's library is in its assets/.
func TestCiteLibraryInBundle(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := filepath.Join(t.TempDir(), "Essay.textbundle")
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "info.json"), []byte(`{"version":2,"type":"net.daringfireball.markdown"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "text.md"), []byte("# Essay\n\nAs [@goedel31].\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "library.bib"), []byte(citeBib), 0o644)
	if err := w.LoadFile(filepath.Join(dir, "text.md")); err != nil {
		t.Fatal(err)
	}
	if lib := w.library(); lib == nil || lib.Get("knuth84") == nil {
		t.Fatalf("the bundle's library: %s", w.libraryNote())
	}
	if n := w.libraryNote(); !strings.Contains(n, "library.bib (in the bundle)") {
		t.Errorf("note %q", n)
	}
	// saved: the same library, not read again from nothing
	lib := w.library()
	w.textEditor.SetText("# Essay\n\nAs [@goedel31] and @kn")
	if err := w.SaveFile(); err != nil {
		t.Fatal(err)
	}
	if w.library() != lib {
		t.Error("saving the bundle dropped its library")
	}
}

func TestCitedKeys(t *testing.T) {
	text := "---\nnocite: |\n  @extra, @goedel31\n---\n\n[@goedel31; @knuth84] and @goethe08.\n\n```\n@code\n```\n\nAgain [-@knuth84].\n"
	cited, nocite := citedKeys(text)
	if !reflect.DeepEqual(cited, []string{"goedel31", "knuth84", "goethe08"}) || !reflect.DeepEqual(nocite, []string{"extra"}) {
		t.Errorf("cited %v, nocite %v", cited, nocite)
	}
}

// Pandoc's arguments: as before for a document citing nothing; with
// --citeproc, the library, the style and Pandoc's own Markdown for one
// that cites.
func TestPandocArgs(t *testing.T) {
	d := t.TempDir()
	src, out := filepath.Join(d, "document.md"), filepath.Join(d, "out.docx")
	plain := pandocArgs("docx", "T", src, out, d, citeExport{})
	if want := []string{"-f", "gfm", "-t", "docx", "--standalone", "--metadata", "title=T", "-o", out, src}; !slices.Equal(plain, want) {
		t.Errorf("plain: %v", plain)
	}
	c := citeExport{bibName: "library.bib", bib: []byte("x"), cslName: "style.csl", csl: []byte("y"), nocite: []string{"a"}}
	got := strings.Join(pandocArgs("docx", "T", src, out, d, c), " ")
	want := "-f " + citeReader + " -t docx --standalone --metadata title=T --citeproc --bibliography " + filepath.Join(d, "library.bib") +
		" --csl " + filepath.Join(d, "style.csl") + " --metadata-file " + filepath.Join(d, "nocite.yaml") + " -o " + out + " " + src
	if got != want {
		t.Errorf("citing:\n%s\nwant\n%s", got, want)
	}
	c.csl = nil
	c.cslName = "apa"
	if got := strings.Join(pandocArgs("odt", "T", "s", "o", d, c), " "); !strings.Contains(got, "--csl apa ") {
		t.Errorf("a style Pandoc finds itself: %s", got)
	}
}

// The export finds the library and the front matter's csl: from the
// document's folder; nothing is cited without citations; with pandoc
// there, the citations come out as citations and a bibliography.
func TestCiteExport(t *testing.T) {
	w, dir := citeWindow(t, "|")
	os.WriteFile(filepath.Join(dir, "my.csl"), []byte("<style/>"), 0o644)
	text := "---\ncsl: my.csl\n---\n\nAs [@goedel31, p. 3] shows.\n"
	c, err := citeExportFor(text, w.citeInputsFor(text))
	if err != nil || c.bibName != "library.bib" || string(c.bib) != citeBib || c.cslName != "style.csl" || string(c.csl) != "<style/>" {
		t.Errorf("export: %+v %v", c, err)
	}
	if c, _ := citeExportFor("No citations.", w.citeInputsFor("No citations.")); c.on() {
		t.Error("a document citing nothing cites")
	}
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("no pandoc")
	}
	text = "---\nnocite: |\n  @goethe08\n---\n\n# Essay\n\nAs [@goedel31, p. 3] shows, and @knuth84 says. It costs $5 or $6.\n"
	c, err = citeExportFor(text, w.citeInputsFor(text))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "essay.tex")
	if err := exportMarkdownWith(text, "Essay", out, "pandoc-latex", mdPageSetup{cite: c}, nil); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	for _, want := range []string{"(Gödel 1931, 3)", "Knuth (1984)", "Monatshefte", "Faust", `\$5 or \$6`} { // Faust: nocite's
		if !strings.Contains(string(data), want) {
			t.Errorf("the LaTeX has no %q", want)
		}
	}
}

const citeShotText = `---
title: On Incompleteness
bibliography: refs.bib
---

# On Incompleteness

As Gödel showed [@goedel31, pp. 173–175], some truths cannot be proved.
Faust's despair [see @goethe08; also -@goedel31] is older than logic,
and @knuth84 [ch. 1] set it all in type. Mail me@example.org; the
` + "`@code`" + ` is no citation, and a [link](https://example.org) stays a link.

Typing a citation: `

// TestCiteScreenshots renders the colouring, the @ list and the hover,
// desktop and phone (SHOT_DIR).
func TestCiteScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w, _ := newDialogTestWindow(t)
	docs := t.TempDir()
	os.WriteFile(filepath.Join(docs, "refs.bib"), []byte(citeBib), 0o644)
	w.setDocumentType(true)
	w.currentFile = filepath.Join(docs, "essay.md")
	w.textEditor.SetText(citeShotText)
	w.fyneWindow.Canvas().Focus(w.textEditor)
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	}
	hoverAt := func(sub string) {
		off := len([]rune(citeShotText[:strings.Index(citeShotText, sub)]))
		w.textEditor.SetCursorOffset(off)
		w.noteHover(off, w.textEditor.CursorPosition(), w.textEditor.CellSize())
	}
	for _, phone := range []bool{false, true} {
		name := "cite-desktop"
		if phone {
			name = "cite-phone"
			w.togglePane("references") // the desktop's pane put away: the phone's script alone first
			w.fyneWindow.Resize(fyne.NewSize(380, 800))
		}
		w.textEditor.SetText(citeShotText)
		toEnd(w)
		snap(name + "-1-colours")
		typeText(w, "[@go")
		snap(name + "-2-list")
		citeKeyPress(w, fyne.KeyEscape)
		if phone { // a tap on a citation: its entry
			w.textEditor.SetCursorOffset(len([]rune(citeShotText[:strings.Index(citeShotText, "goedel31, pp")])))
			w.citeTipAtCursor()
		} else {
			hoverAt("goedel31, pp")
		}
		snap(name + "-3-hover")
		w.noteHover(-1, fyne.Position{}, fyne.Size{})
		w.showEditorTip("", fyne.Position{}, fyne.Size{})
		w.textEditor.SetText(citeShotText + "[@nobody]")
		if !w.paneShown("references") {
			w.togglePane("references")
		}
		snap(name + "-4-references")
	}
	w.runEx("theme light") // the green on a light page
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
	w.textEditor.SetText(citeShotText)
	toEnd(w)
	typeText(w, "[@go")
	snap("cite-light-desktop")
}

// The References pane lists what is cited, first citation first, then
// nocite's; a key the library lacks says so; a tap goes to the citation.
func TestReferencesPane(t *testing.T) {
	text := "---\nnocite: |\n  @goethe08\n---\n\nFirst [@knuth84].\n\nThen @goedel31 and @nobody, and [-@knuth84] again.\n"
	w, _ := citeWindow(t, text+"|")
	w.togglePane("references")
	if !w.paneShown("references") {
		t.Fatal("no References pane")
	}
	p := w.referencesPane()
	var got []string
	for _, it := range p.items {
		s := it.key
		if it.entry == nil {
			s += " (missing)"
		}
		if it.line < 0 {
			s += " (nocite)"
		}
		got = append(got, s)
	}
	if want := []string{"knuth84", "goedel31", "nobody (missing)", "goethe08 (nocite)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("items %v, want %v", got, want)
	}
	if !strings.Contains(p.note.Text, "3 entries in refs.bib (beside the document)") {
		t.Errorf("note %q", p.note.Text)
	}
	p.list.Select(1)
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 7 {
		t.Errorf("the tap went to line %d, want 7", line)
	}
	toEnd(w)
	typeText(w, "\nAnd [@goethe08].")
	if last := p.items[len(p.items)-1]; last.key != "goethe08" || last.line != 9 {
		t.Errorf("cited now, not nocite: %+v", last)
	}
}

// With the library read in the background (as the app reads it): the
// first @ is typed before it is read, and the list opens once it is.
func TestCiteListAfterBackgroundRead(t *testing.T) {
	holdUI(t)
	w, _ := citeWindow(t, "See |")
	realPaths(t)
	typeText(w, "@go")
	waitFor(t, "the list", func() bool { return w.citePop.open() })
	if k := w.citePop.entries[0].Key; k != "goedel31" {
		t.Errorf("first %q", k)
	}
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/cite"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Citations in a Markdown document, as Zettlr has them (LOOP.md,
// 2026-11-22): Pandoc's syntax ([@key, p. 12], @key, [-@key]) coloured
// in the editor (internal/editor/syntax/cite.go); the document's library
// (internal/cite) read from its front matter's bibliography:, else one
// in its bundle's assets/ or beside it, else Preferences' main library;
// @ completes a key from it (cite_complete.go); the pointer over a
// citation (a tap on a phone) shows the entry; Pandoc's exports cite
// with --citeproc and the front matter's csl:.
//
// The library is watched: whenever it is looked at (the @ list, a
// hover, the References pane) and a second has passed, the file's time
// and size are compared, and a changed file (Zotero's Better BibTeX
// exporting again) is read again in the background. A file read while
// it is being written does not parse: the library read before is kept
// until it does.

// citeLibraryPref is Preferences' main library (a .json or .bib path).
const citeLibraryPref = "citation-library"

// citeCheckEvery is how often the library's file is looked at again.
const citeCheckEvery = time.Second

// citeState is the window's library: what it was found for, read from,
// and what it holds.
type citeState struct {
	key     string      // the document and its front matter's bibliography it was found for
	src     cite.Source // where it was read
	stamp   string      // the file's time and size when read
	lib     *cite.Library
	err     error // why it could not be read ("" with lib: read)
	checked time.Time
	loading bool
	changed []func() // told when what it holds changes (the @ list, the References pane)
	tipRev  uint64   // the text's revision when it last changed: a cursor moved at another is typing's, no tap (citeTipAtCursor)
}

// citeInputs is what finding the library needs, taken on the UI
// goroutine for the background's work.
type citeInputs struct {
	key    string
	place  cite.Place
	bundle *textBundle
	csl    string // the front matter's csl:
}

// citeInputsFor gathers where text (the window's document) is.
func (w *MainWindow) citeInputsFor(text string) citeInputs {
	front := cite.FrontOf(text)
	in := citeInputs{place: cite.Place{Front: front.Bibliography}, bundle: w.bundle, csl: front.CSL}
	if w.settings != nil {
		in.place.Main = strings.TrimSpace(w.settings.GetString(citeLibraryPref))
	}
	if !inBrowser {
		in.place.Home, _ = os.UserHomeDir()
	}
	file := w.currentFile
	if b := w.bundle; b != nil {
		file = b.Path
	}
	if file != "" && !inBrowser {
		in.place.Dir = filepath.Dir(file)
		in.place.Stem = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	} else if file == "" && w.unsavedDir != "" && !inBrowser {
		in.place.Dir = w.unsavedDir
	}
	in.key = file + "|" + front.Bibliography + "|" + in.place.Main + "|" + front.CSL // a bundle saved again is the same (its stamp says if the library changed)
	return in
}

// read gives a source's bytes: the bundle's entry, or the file.
func (in citeInputs) read(s cite.Source) ([]byte, error) {
	if s.Bundle {
		return in.bundle.readExtra(s.Name)
	}
	if inBrowser || !filepath.IsAbs(s.Name) {
		return nil, fmt.Errorf("%s: not found (the document is not saved yet)", s.Name)
	}
	return os.ReadFile(s.Name)
}

// stampOf is a source's time and size ("" if it cannot be seen): a
// bundle's entry is the bundle's, as read.
func (in citeInputs) stampOf(s cite.Source) string {
	if s.Bundle {
		if b := in.bundle; b != nil && !b.Zipped {
			if st, err := os.Stat(filepath.Join(b.Path, filepath.FromSlash(s.Name))); err == nil {
				return st.ModTime().String() + "/" + fmt.Sprint(st.Size())
			}
		}
		return fmt.Sprintf("%p", in.bundle)
	}
	if st, err := os.Stat(s.Name); err == nil {
		return st.ModTime().String() + "/" + fmt.Sprint(st.Size())
	}
	return ""
}

// listed fills the place's files: the folder's, the bundle's entries.
func (in *citeInputs) listed() {
	if d := in.place.Dir; d != "" {
		if es, err := os.ReadDir(d); err == nil {
			for _, e := range es {
				if !e.IsDir() {
					in.place.DirFiles = append(in.place.DirFiles, e.Name())
				}
			}
		}
	}
	if b := in.bundle; b != nil {
		in.place.Bundle = []string{}
		if es, err := b.Entries(); err == nil {
			for _, e := range es {
				in.place.Bundle = append(in.place.Bundle, strings.TrimPrefix(e.Name, b.Prefix))
			}
		}
	}
}

// citeFound is what the background found.
type citeFound struct {
	in    citeInputs
	src   cite.Source
	stamp string
	lib   *cite.Library
	err   error
	same  bool // the source as before, unchanged: nothing read
}

// find reads the library: the file read before again if its time or
// size changed (the same document: what it cites from is known), else
// it is found anew.
func (in citeInputs) find(prevSrc cite.Source, prevStamp string) citeFound {
	f := citeFound{in: in}
	if prevSrc.Name != "" {
		f.src, f.stamp = prevSrc, in.stampOf(prevSrc)
		if f.stamp == prevStamp {
			f.same = true
			return f
		}
		data, err := in.read(prevSrc)
		if err == nil {
			f.lib, err = cite.Parse(prevSrc.Name, data)
		}
		f.err = err
		in.readStyle(f.lib)
		return f
	}
	in.listed()
	f.src, f.lib, f.err = cite.Resolve(in.place, in.read)
	f.stamp = in.stampOf(f.src)
	in.readStyle(f.lib)
	return f
}

// cslSources are where the front matter's csl: file may be: the
// bundle's, beside the document, under the home folder, as written.
func (in citeInputs) cslSources(name string) []cite.Source {
	var tries []cite.Source
	if in.bundle != nil {
		tries = append(tries, cite.Source{Bundle: true, Name: name}, cite.Source{Bundle: true, Name: "assets/" + name})
	}
	if in.place.Dir != "" && !filepath.IsAbs(name) {
		tries = append(tries, cite.Source{Name: filepath.Join(in.place.Dir, filepath.FromSlash(name))})
	}
	if strings.HasPrefix(name, "~/") && in.place.Home != "" {
		tries = append(tries, cite.Source{Name: filepath.Join(in.place.Home, name[2:])})
	}
	if filepath.IsAbs(name) {
		tries = append(tries, cite.Source{Name: name})
	}
	return tries
}

// cslFile is the csl: file's bytes, nil when there is none.
func (in citeInputs) cslFile(name string) []byte {
	for _, s := range in.cslSources(name) {
		if d, err := in.read(s); err == nil {
			return d
		}
	}
	return nil
}

// readStyle reads the document's csl: file into lib (Accolade's own
// styles by name need none; one that cannot be read is left out:
// Accolade's style cites, StyleOf).
func (in citeInputs) readStyle(lib *cite.Library) {
	if lib == nil || in.csl == "" {
		return
	}
	data := in.cslFile(in.csl)
	switch {
	case data != nil:
		st, err := cite.ParseStyle(data)
		if err != nil {
			lib.StyleNote = fmt.Sprintf("The style %s cannot be used (%v): cited in Accolade's own.", in.csl, err)
			return
		}
		lib.Style, lib.StyleFor = st, in.csl
	case cite.BuiltinCSL(in.csl) == nil:
		lib.StyleNote = "No style " + in.csl + " here: cited in Accolade's own (Pandoc's export looks for it itself)."
	}
}

// citations is the window's library state, made when first asked for.
func (w *MainWindow) citations() *citeState {
	if w.cites == nil {
		w.cites = &citeState{}
		// a phone's tip on a citation said "Reading the library…": the entry now
		w.cites.changed = append(w.cites.changed, func() {
			if w.narrow && w.noteTipShown != "" {
				w.noteTipShown = "" // shown again, as it is now
				w.citeTipAtCursor()
			}
		}, func() { // the preview's citations and references, from the library as read now
			if p := w.previewPane; w.previewVisible && p != nil {
				w.updateLivePreview()
				if p.showsPages() && p.pages != nil && !w.followEdits() && p.pages.view.d != nil { // not made again by the update
					p.pages.update(p.last)
				}
			}
		})
	}
	return w.cites
}

// library is the document's library as last read (nil for none), and
// looks again in the background when it is time: the document, its
// front matter's bibliography or the file changed.
func (w *MainWindow) library() *cite.Library {
	if !w.markdown {
		return nil
	}
	st := w.citations()
	in := w.citeInputsFor(w.textEditor.Text())
	if st.loading || in.key == st.key && time.Since(st.checked) < citeCheckEvery {
		return st.lib
	}
	if in.key != st.key { // another document or library: the old one is not this one's
		st.lib, st.err, st.src, st.stamp = nil, nil, cite.Source{}, ""
	}
	st.loading, st.checked = true, time.Now()
	prevSrc, prevStamp := st.src, st.stamp
	inBackground(func() citeFound { return in.find(prevSrc, prevStamp) }, func(f citeFound) {
		st.loading, st.checked = false, time.Now()
		w.citeApply(f)
	})
	return st.lib
}

// citeApply takes what the background found.
func (w *MainWindow) citeApply(f citeFound) {
	st := w.citations()
	if cur := w.citeInputsFor(w.textEditor.Text()); cur.key != f.in.key {
		st.key = "" // the document changed meanwhile: look again next time
		return
	}
	st.key = f.in.key
	if f.same && f.src == st.src {
		return
	}
	if f.src == st.src && st.lib != nil && (f.err != nil || f.lib != nil && f.lib.Warning != "" && len(f.lib.Entries) < len(st.lib.Entries)) {
		return // being written again (half a file: unreadable, or a .bib cut short): the library before is kept
	}
	st.src, st.stamp, st.lib, st.err = f.src, f.stamp, f.lib, f.err
	for _, c := range st.changed {
		c()
	}
}

// libraryNote says where the library is, or why there is none, for the
// status bar and the References pane.
func (w *MainWindow) libraryNote() string {
	st := w.citations()
	switch {
	case st.err != nil:
		return "The library cannot be read: " + st.err.Error()
	case st.lib == nil && st.loading:
		return "Reading the library…"
	case st.lib == nil:
		return "No library: name one in the front matter (bibliography: refs.bib), put a .bib or CSL-JSON .json beside the document, or choose a main library in Preferences > Editor."
	}
	n := len(st.lib.Entries)
	s := fmt.Sprintf("%d entries in %s.", n, st.src)
	if n == 1 {
		s = fmt.Sprintf("1 entry in %s.", st.src)
	}
	if st.lib.Warning != "" {
		s += " " + st.lib.Warning
	}
	if st.lib.StyleNote != "" {
		s += " " + st.lib.StyleNote
	}
	return s
}

// citeTipAt is the tip for offset of a Markdown text: the entries of the
// citation there ("" if offset is on none).
func (w *MainWindow) citeTipAt(text string, offset int) string {
	line, col := lineOfOffset(text, offset)
	for _, c := range syntax.Citations(line) {
		if col < c.Start || col >= c.End {
			continue
		}
		keys := c.Keys
		for _, k := range c.Keys { // over one key of several: that one
			if col >= k.Start && col < k.End {
				keys = []syntax.CiteKey{k}
			}
		}
		lib := w.library()
		var parts []string
		for _, k := range keys {
			e := lib.Get(k.Key)
			switch {
			case e != nil:
				parts = append(parts, e.Tip())
			case lib == nil:
				parts = append(parts, "@"+k.Key+": "+w.libraryNote())
			default:
				parts = append(parts, "@"+k.Key+" is not in "+w.citations().src.Base()+".")
			}
		}
		return strings.Join(parts, "\n\n")
	}
	return ""
}

// lineOfOffset is the line holding the rune offset and offset's column in it.
func lineOfOffset(text string, offset int) (string, int) {
	start, n := 0, 0
	for i, r := range text {
		if n == offset {
			break
		}
		if r == '\n' {
			start = i + 1
		}
		n++
	}
	end := strings.IndexByte(text[start:], '\n')
	if end < 0 {
		end = len(text) - start
	}
	line := text[start : start+end]
	col := offset - utf8.RuneCountInString(text[:start])
	return line, col
}

// citeTipAtCursor shows the citation's entry under the cursor on a
// phone, which has no pointer to hover with: a tap on a citation shows
// it, a tap elsewhere puts it away.
func (w *MainWindow) citeTipAtCursor() {
	if !w.markdown || !w.narrow || w.citePop != nil && w.citePop.open() {
		return
	}
	if w.textEditor.Buffer().Rev() != w.citations().tipRev { // typing moved it, not a tap (citeFollow keeps the revision)
		if w.noteTipShown != "" {
			w.showEditorTip("", fyne.Position{}, fyne.Size{})
		}
		return
	}
	tip := w.citeTipAt(w.textEditor.Text(), w.textEditor.CursorOffset())
	if tip == "" && w.noteTipShown == "" {
		return
	}
	w.showEditorTip(tip, w.textEditor.CursorPosition(), w.textEditor.CellSize())
}

// citedKeys are the keys a Markdown text cites, in order, each once;
// with the front matter's nocite after them.
func citedKeys(text string) (cited, nocite []string) {
	seen := map[string]bool{}
	cite.EachCitation(text, func(_ int, c syntax.Citation) {
		for _, k := range c.Keys {
			if !seen[k.Key] {
				seen[k.Key] = true
				cited = append(cited, k.Key)
			}
		}
	})
	for _, k := range cite.FrontOf(text).Nocite {
		if !seen[k] {
			seen[k] = true
			nocite = append(nocite, k)
		}
	}
	return cited, nocite
}

// citedBody is the part of a Markdown document that is shown and
// printed (markdownBody), its citations in its style and its
// bibliography added when it has a library (lib; nil: the citations as
// written) — what the Live Preview, the page view and Accolade's own
// PDF, Word, ODT and web page show. The style is the front matter's
// csl: (a .csl file, or apa, chicago-author-date, ieee, vancouver:
// internal/cite/csl.go), else Accolade's own author-date one
// (style.go). Pandoc's exports cite with --citeproc and the same .csl.
func citedBody(text string, lib *cite.Library) string {
	front := cite.FrontOf(text)
	return cite.RenderWith(markdownBody(text), lib, front.Nocite, cite.StyleOf(front.CSL, lib))
}

// libraryNow is the document's library, found and read at once (the
// background's work, for an export: the window's may not be read yet).
func libraryNow(text string, in citeInputs) *cite.Library {
	if !strings.ContainsRune(text, '@') {
		return nil
	}
	in.listed()
	_, lib, _ := cite.Resolve(in.place, in.read)
	in.readStyle(lib)
	return lib
}

// citeExport is what Pandoc's export needs to cite: the library's file
// (its bytes, written beside the Markdown it reads), the style's, the
// keys listed without being cited. Empty: the document is exported as
// before.
type citeExport struct {
	bibName string
	bib     []byte
	cslName string // the front matter's csl:, as written when it is no file here (Pandoc's data folder has it)
	csl     []byte
	nocite  []string
}

// on: the export cites.
func (c citeExport) on() bool { return len(c.bib) > 0 }

// citeExportFor finds the library and style text's export cites with,
// in.listed's work included (the background's).
func citeExportFor(text string, in citeInputs) (citeExport, error) {
	var c citeExport
	cited, nocite := citedKeys(text)
	if len(cited) == 0 && len(nocite) == 0 {
		return c, nil
	}
	in.listed()
	src, lib, err := cite.Resolve(in.place, in.read)
	if err != nil {
		return c, err
	}
	if lib == nil {
		return c, nil // nothing to cite from: the citations stay as written
	}
	data, err := in.read(src)
	if err != nil {
		return c, err
	}
	c.bibName, c.bib, c.nocite = "library"+filepath.Ext(src.Base()), data, nocite
	if name := cite.FrontOf(text).CSL; name != "" {
		c.cslName = name
		if d := in.cslFile(name); d != nil {
			c.cslName, c.csl = "style.csl", d
		} else if d := cite.BuiltinCSL(name); d != nil { // one of Accolade's: Pandoc cites with the same file
			c.cslName, c.csl = "style.csl", d
		}
	}
	return c, nil
}

// citeReader is the Markdown Pandoc reads a citing document as: its own
// Markdown (GitHub's has no citations), made like GitHub's where they
// differ (compared on headings, lists and quotes under a line of text, a
// rule before text, backslashes, dollars, [Heading] references): bare
// addresses are links, lists need no blank line before them nor headings,
// no curly quotes of its own, no figures, ~ ^ \word and $ as written,
// [Heading] no link, no YAML block in the text (its front matter is not
// sent; the nocite list goes in a file). Left: a quote right under a
// list item joins the item.
const citeReader = "markdown+autolink_bare_uris+gfm_auto_identifiers+lists_without_preceding_blankline-blank_before_header" +
	"-smart-implicit_figures-subscript-superscript-implicit_header_references-raw_tex-tex_math_dollars-yaml_metadata_block"

// pandocArgs are the arguments of Pandoc's export of the Markdown at src
// to format into tmp, dir holding the library and style c names (written
// there by pandocFiles).
func pandocArgs(format, title, src, tmp, dir string, c citeExport) []string {
	if !c.on() {
		return []string{"-f", "gfm", "-t", format, "--standalone", "--metadata", "title=" + title, "-o", tmp, src}
	}
	args := []string{"-f", citeReader, "-t", format, "--standalone", "--metadata", "title=" + title,
		"--citeproc", "--bibliography", filepath.Join(dir, c.bibName)}
	switch {
	case len(c.csl) > 0:
		args = append(args, "--csl", filepath.Join(dir, c.cslName))
	case c.cslName != "":
		args = append(args, "--csl", c.cslName)
	}
	if len(c.nocite) > 0 {
		args = append(args, "--metadata-file", filepath.Join(dir, "nocite.yaml"))
	}
	return append(args, "-o", tmp, src)
}

// pandocFiles writes the library, the style and the nocite list into dir.
func pandocFiles(dir string, c citeExport) error {
	if !c.on() {
		return nil
	}
	if err := os.WriteFile(filepath.Join(dir, c.bibName), c.bib, 0o644); err != nil {
		return err
	}
	if len(c.csl) > 0 {
		if err := os.WriteFile(filepath.Join(dir, c.cslName), c.csl, 0o644); err != nil {
			return err
		}
	}
	if len(c.nocite) > 0 {
		var ks []string
		for _, k := range c.nocite {
			ks = append(ks, "@"+k)
		}
		meta := "nocite: |\n  " + strings.Join(ks, ", ") + "\n"
		if err := os.WriteFile(filepath.Join(dir, "nocite.yaml"), []byte(meta), 0o644); err != nil {
			return err
		}
	}
	return nil
}

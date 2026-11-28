// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"archive/zip"
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdfin"
)

// The kinds of file the file dialogs show one at a time (the fork's
// FileDialog.SetFileKinds): what a file is for the writer, not its format.
const (
	kindScreenplay = "Screenplay"
	kindNovel      = "Novel"
	kindOutline    = "Outline"
)

// pickKinds are the kinds in the dialogs' order; "" is any other file.
var pickKinds = []string{kindScreenplay, kindNovel, kindOutline, ""}

// kindGroup is the name of a kind's "All …" entry in the file type select.
var kindGroup = map[string]string{
	kindScreenplay: "All screenplay formats",
	kindNovel:      "All novel formats",
	kindOutline:    "All outline formats",
	"":             "All other formats",
}

// pickFormat is a format a file dialog offers: its name in the file type
// select, the kind its files are in, its extensions.
type pickFormat struct {
	name, kind string
	exts       []string
}

// extFormat is the format of an extension: its name and kind. A PDF is
// a screenplay's or a novel's (pickFormats offers both); a plain text or
// Markdown file is a novel's by its name, either by its content
// (fileKindOf).
func extFormat(ext string) (name, kind string) {
	switch ext {
	case ".fountain", ".spmd":
		return "Fountain", kindScreenplay
	case ".fdx":
		return "Final Draft (FDX)", kindScreenplay
	case ".highland":
		return "Highland", kindScreenplay
	case ".accolade":
		return "Accolade", kindScreenplay
	case ".textpack":
		return "TextPack", kindScreenplay
	case ".textbundle":
		return "TextBundle", kindScreenplay
	case ".pdf":
		return "Screenplay (PDF)", kindScreenplay
	case ".md", ".markdown":
		return "Markdown", kindNovel
	case ".txt":
		return "Plain text", kindNovel
	case ".docx":
		return "Word (DOCX)", kindNovel
	case ".odt":
		return "OpenDocument (ODT)", kindNovel
	case ".html", ".htm", ".xhtml":
		return "Web page (HTML)", kindNovel
	case ".epub":
		return "EPUB", kindNovel
	case ".rtf":
		return "Rich Text (RTF)", kindNovel
	case ".tex", ".latex":
		return "LaTeX", kindNovel
	case ".org":
		return "Org", kindNovel
	case ".rst":
		return "reStructuredText", kindNovel
	case ".fb2":
		return "FictionBook", kindNovel
	case ".textile":
		return "Textile", kindNovel
	case ".dbk":
		return "DocBook", kindNovel
	case ".typ":
		return "Typst", kindNovel
	case ".opml":
		return "Outline (OPML)", kindOutline
	case ".toml":
		return "Lexington settings (TOML)", ""
	}
	for _, t := range langTypes() { // a language's own extension: its Fountain
		if t.ext == ext {
			return t.label + " Fountain", kindScreenplay
		}
	}
	if f, ok := otherFormat("x" + ext); ok { // another program's, as Lexington names it
		return f.Name, kindScreenplay
	}
	return strings.ToUpper(strings.TrimPrefix(ext, ".")), ""
}

// pickFormats are the formats of a dialog's extensions, by kind, in the
// order the extensions come in; a PDF is offered as a novel's too when
// the dialog offers novels.
func pickFormats(exts []string) []pickFormat {
	var out []pickFormat
	add := func(name, kind, ext string) {
		for i := range out {
			if out[i].name == name {
				if !slices.Contains(out[i].exts, ext) {
					out[i].exts = append(out[i].exts, ext)
				}
				return
			}
		}
		out = append(out, pickFormat{name: name, kind: kind, exts: []string{ext}})
	}
	novels := false
	for _, e := range exts {
		if _, kind := extFormat(strings.ToLower(e)); kind == kindNovel {
			novels = true
		}
	}
	for _, e := range exts {
		e = strings.ToLower(e)
		name, kind := extFormat(e)
		add(name, kind, e)
		if e == ".pdf" && novels {
			add("Novel (PDF)", kindNovel, e)
		}
	}
	return out
}

// pickFilters are the file type select's entries for a dialog's
// extensions: each kind's "All … formats", then its formats one by one
// (set in under it), kind by kind; the dialog adds "All files". first is
// the index of the preferred kind's "All …" entry (0 if it has none).
func pickFilters(exts []string, prefer string) (filters []dialog.NamedFilter, kinds []string, first int) {
	formats := pickFormats(exts)
	if prefer == "" && len(formats) > 0 { // the kind of the list's first format
		prefer = formats[0].kind
	}
	for _, kind := range pickKinds {
		var of []pickFormat
		var union []string
		for _, f := range formats {
			if f.kind == kind {
				of = append(of, f)
				for _, e := range f.exts {
					if !slices.Contains(union, e) {
						union = append(union, e)
					}
				}
			}
		}
		if len(of) == 0 {
			continue
		}
		if kind == kindScreenplay || kind == kindNovel {
			// a file its content tells the kind of may be of this one
			// (a screenplay in a .txt): the kind toggles sort them
			for _, e := range exts {
				if e = strings.ToLower(e); readsKind(e) && !slices.Contains(union, e) {
					union = append(union, e)
				}
			}
		}
		if kind != "" {
			kinds = append(kinds, kind)
		}
		if kind == prefer && kind != "" {
			first = len(filters)
		}
		indent := ""
		if len(of) > 1 || len(formats) > 1 {
			group := kindGroup[kind]
			if len(of) == len(formats) && kind == "" { // pictures, fonts: nothing else
				group = "All supported formats"
			}
			filters = append(filters, dialog.NamedFilter{Name: group, Filter: storage.NewExtensionFileFilter(union), Kind: kind})
			indent = " " // set in under its group: the select has no headers
		}
		for _, f := range of {
			filters = append(filters, dialog.NamedFilter{Name: indent + f.name, Filter: storage.NewExtensionFileFilter(f.exts), Kind: kind})
		}
	}
	return filters, kinds, first
}

// pickerKinds turns on the dialogs' kind toggles and their background
// classification. Tests turn it off: under Fyne's test driver fyne.Do runs
// on the classifier's own goroutine, racing the test's rendering.
var pickerKinds = true

// setPickFilters gives an open dialog its file types and, when they have
// kinds, the kind toggles, prefer's (or the first's) chosen.
func setPickFilters(d *dialog.FileDialog, exts []string, prefer string) {
	filters, kinds, first := pickFilters(exts, prefer)
	d.SetFilters(filters)
	if len(kinds) > 0 && pickerKinds {
		d.SetFileKinds(kinds, func(u fyne.URI) string { return fileKindOf(u.Path()) })
	}
	d.SelectFilter(first)
}

// preferredKind is the kind a window's dialogs show first.
func (w *MainWindow) preferredKind() string {
	if w != nil && w.markdown {
		return kindNovel
	}
	return kindScreenplay
}

// The kinds of files found by their content: by path, changed time and
// size, so a folder is read once.
type kindKey struct {
	path string
	mod  time.Time
	size int64
}

var kindCache = struct {
	sync.Mutex
	m       map[kindKey]string
	reading map[kindKey]chan struct{} // closed when the kind is known
}{m: map[kindKey]string{}, reading: map[kindKey]chan struct{}{}}

// sniffing are as many files as are read for their kinds at once.
var sniffing = make(chan struct{}, 4)

// sniffPages are the pages a PDF is read for its kind: past a title
// page and a book's front matter (measured on 22 PDFs of Joop's: with 3
// pages a book's opening poem read as a script, with 10 none did; 51
// files' kinds took 2.8 s, 1.3 s the longest).
const sniffPages = 10

// sniffLimit is as long as a PDF is read for its kind; one that takes
// longer (a broken file) is of no kind. The file dialog asks in the
// background, so nothing waits.
const sniffLimit = 5 * time.Second

// readsKind: a file with the extension ext is a screenplay's or a
// novel's by what it holds.
func readsKind(ext string) bool {
	switch ext {
	case ".pdf", ".txt", ".md", ".markdown", ".accolade", ".textpack":
		return true
	}
	return false
}

// fileKindOf is the kind of the file at path: by its name where that
// says it, else by its content (a PDF's first pages, a text's first
// lines, a bundle's text file), which can take seconds: never on the
// UI goroutine.
func fileKindOf(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if !readsKind(ext) {
		_, kind := extFormat(ext)
		return kind
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return ""
	}
	key := kindKey{path, st.ModTime(), st.Size()}
	kindCache.Lock()
	kind, ok := kindCache.m[key]
	done, reading := kindCache.reading[key]
	if !ok && !reading {
		done = make(chan struct{})
		kindCache.reading[key] = done
		go func() {
			sniffing <- struct{}{}
			k := sniffKind(path, ext)
			<-sniffing
			kindCache.Lock()
			kindCache.m[key] = k
			delete(kindCache.reading, key)
			kindCache.Unlock()
			close(done)
		}()
	}
	kindCache.Unlock()
	if ok {
		return kind
	}
	<-done
	kindCache.Lock()
	defer kindCache.Unlock()
	return kindCache.m[key]
}

// sniffKind reads enough of a file to tell its kind.
func sniffKind(path, ext string) string {
	switch ext {
	case ".pdf":
		return sniffPDF(path)
	case ".accolade", ".textpack":
		return sniffBundle(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	head := make([]byte, 8192)
	n, _ := f.Read(head)
	if textIsScreenplay(string(head[:n])) {
		return kindScreenplay
	}
	return kindNovel
}

// sniffBundle tells a zipped bundle's kind by its text file.
func sniffBundle(path string) string {
	z, err := zip.OpenReader(path)
	if err != nil {
		return ""
	}
	defer z.Close()
	for _, f := range z.File {
		switch strings.ToLower(filepath.Ext(f.Name)) {
		case ".md", ".markdown":
			if !strings.Contains(f.Name, "/assets/") {
				return kindNovel
			}
		case ".fountain":
			return kindScreenplay
		}
	}
	return kindScreenplay
}

var sniffSceneStart = sync.OnceValue(func() *regexp.Regexp {
	var words []string
	for _, h := range allSceneHeaders() {
		words = append(words, regexp.QuoteMeta(strings.ToUpper(strings.TrimRight(h, ". "))))
	}
	slices.SortFunc(words, func(a, b string) int { return len(b) - len(a) })
	return regexp.MustCompile(`^(?:` + strings.Join(slices.Compact(words), "|") + `)[ .]`)
})

// isCue is a character's name as a script has it over a speech: in
// capitals, short, a name's letters.
func isCue(s string) bool {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '('); i > 0 { // NAME (V.O.)
		s = strings.TrimSpace(s[:i])
	}
	if s == "" || len([]rune(s)) > 30 || sniffSceneStart().MatchString(s) || strings.HasSuffix(s, ":") {
		return false
	}
	letters := 0
	for _, r := range s {
		switch {
		case unicode.IsLower(r):
			return false
		case unicode.IsLetter(r):
			letters++
		case r != ' ' && r != '.' && r != '\'' && r != '-' && !unicode.IsDigit(r):
			return false
		}
	}
	return letters >= 2
}

// textIsScreenplay tells from a text's first lines whether it is a
// screenplay in Fountain: scene headings, a title page, or names in
// capitals over speeches.
func textIsScreenplay(text string) bool {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		lines = append(lines, strings.TrimRight(sc.Text(), " \t\r"))
	}
	if len(lines) > 0 && strings.HasPrefix(strings.ToLower(lines[0]), "title:") {
		return true
	}
	headings, cues, headers := 0, 0, 0
	for i, l := range lines {
		t := strings.TrimSpace(l)
		up := strings.ToUpper(t)
		switch {
		case strings.HasPrefix(t, "#"):
			headers++
		case sniffSceneStart().MatchString(up) || (strings.HasPrefix(t, ".") && len(t) > 1 && unicode.IsLetter([]rune(t)[1])):
			if i == 0 || strings.TrimSpace(lines[i-1]) == "" {
				headings++
			}
		case isCue(t) && (i == 0 || strings.TrimSpace(lines[i-1]) == "") && i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "":
			cues++
		}
	}
	return headings >= 1 && headings+cues >= 2 || cues >= 3 && cues > headers
}

// sniffPDF tells a PDF's kind as Lexington reads its first pages as a screenplay:
// a script's lines come out as scene headings, names and their speeches;
// a novel's mostly as action. A PDF with no text (a scan) is of no kind.
func sniffPDF(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), sniffLimit)
	defer cancel()
	sp, err := pdfin.ReadFileWith(path, pdfin.Options{Context: ctx, MaxPages: sniffPages, Scenes: allSceneHeaders()})
	if err != nil {
		return ""
	}
	return screenplayKind(sp)
}

// screenplayKind is the kind of what Lexington read as a screenplay.
func screenplayKind(sp lex.Screenplay) string {
	var speakers, spoken, lines int
	for _, l := range sp {
		switch l.Type {
		case lex.TypeEmpty, lex.TypeTitle:
			continue
		case lex.TypeSpeaker:
			speakers++
		case lex.TypeDialog, lex.TypeParen:
			spoken++
		}
		lines++
	}
	if lines < 5 {
		return ""
	}
	// measured on 20 PDFs of Joop's: scripts speak in a third to three
	// quarters of their lines, books in under 2% (an indented poem, a
	// list read as names); scene words alone mislead (a book's chapters)
	switch {
	case speakers >= 1 && (spoken+speakers)*5 >= lines:
		return kindScreenplay
	case lines >= 20: // enough to tell a book by
		return kindNovel
	}
	return ""
}

// setSaveFilters gives a save dialog the file types it can save as, the
// one of ext chosen: choosing another gives the name its extension.
func setSaveFilters(d *dialog.FileDialog, exts []string, ext string) {
	filters, _, first := pickFilters(exts, "")
	d.SetFilters(filters)
	for i, f := range filters {
		if strings.HasPrefix(f.Name, " ") && f.Filter.(*storage.ExtensionFileFilter).Extensions[0] == ext {
			first = i
			break
		}
	}
	d.SelectFilter(first)
}

// saveExtensions are the extensions Save As saves a script in (a folder
// bundle is chosen as a folder).
func saveExtensions() []string {
	var out []string
	for _, t := range knownTypes() {
		if t.kind != bundleFolder && !slices.Contains(out, t.ext) {
			out = append(out, t.ext)
		}
	}
	// other programs' formats: a copy written through Lexington
	// (other_formats.go)
	for _, e := range otherFormatExts() {
		if _, ok := otherSaveFormat("x" + e); ok && !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	return out
}

// setNamedFileFilter gives the save dialog of a file named name (a
// report, a page, an export) its type, by the name's extension.
func setNamedFileFilter(d *dialog.FileDialog, name string) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return
	}
	format, _ := extFormat(ext)
	if ext == ".pdf" {
		format = "PDF" // a report's or a page's too, not only a screenplay's
	}
	d.SetFilters([]dialog.NamedFilter{{Name: format, Filter: storage.NewExtensionFileFilter([]string{ext})}})
}

// showPicker shows a file dialog as large as the window allows, up to a
// size where its list has room: Fyne's own is a few rows tall.
func showPicker(d *dialog.FileDialog, win fyne.Window) {
	d.Show()
	d.Resize(fitDialog(win, 860, 620))
}

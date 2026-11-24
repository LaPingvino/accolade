package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The documents a window edits (docs/MIDDLE.md). A window has one script,
// its file (w.textEditor, saved by File > Save, shared when editing
// together); beside it, documents of their own the main editor can hold
// instead (a middle tab, middle.go): the script's source and its notes.
// Whatever needs to know what the writer is typing in asks the main
// document (w.mainDoc) rather than taking the script's editor for it.
//
// Each document has its kind, where it is kept (its home), its own editor
// (so its own undo history) and the Doc it is shared through when the
// script is shared:
//
//	kind    home                                    collab Doc
//	script  the window's file                       the session's (named "script")
//	notes   the notes block at the script's end     the script's (it is in its text)
//	source  in the script (/* Source */)            the script's
//	        in the bundle, a kept copy, beside it   its own, the one the setting "source" names (collab_source.go)
//
// A document kept in the script is a stretch of the script's text: each
// change is written into the script at once, as the Notes pane does, and
// a change to that stretch made in the script (Undo there, a co-writer's
// edit) comes into the document's editor as Buffer.Remote edits, outside
// its undo history. A source kept elsewhere is written to its home when
// the script is saved (File > Save, auto-save), when it leaves the main
// editor, and on the Source pane's Done; until then it is unsaved, as the
// title and the close question say.

// docKind is what a document is.
type docKind int

const (
	docScript docKind = iota
	docSource
	docNotes
)

func (k docKind) String() string {
	switch k {
	case docSource:
		return "Source"
	case docNotes:
		return "Notes"
	}
	return "Script"
}

// paneOfKind is the dock pane a kind of document belongs to.
func paneOfKind(k docKind) string {
	switch k {
	case docSource:
		return "source"
	case docNotes:
		return "notes"
	}
	return ""
}

// document is one of the window's documents.
type document struct {
	w    *MainWindow
	kind docKind
	ed   *editor.ScriptEditor
	vim  *vim
	// folds are its Markdown headings' folds (editor_folds.go); the
	// script's are the window's (w.scriptFolds)
	folds *folds
	// view is the editor in its scroll with the editor's fonts, as the
	// main area or a pane shows it (the script's: w.editorView)
	view *container.ThemeOverride
	// ref is a source's home, as opened
	ref sourceRef
	// stored is the text as last read from its home or written to it;
	// homeSeen is the home's text then (a home in the script keeps a
	// "*/" as "* /", and its block's new line: it reads back otherwise)
	stored, homeSeen string
	// fromHome: its text is being set from its home, not by the writer;
	// writing: it is being written into the script (whose change is its own);
	// moving: its home is moving (Keep Source in the Script), not going
	fromHome, writing, moving bool
	// linkBase is a source's text as its links were last looked at
	// (relink.go): what an edit of the source is measured from; atPassage the passage its cursor was in (-1 none),
	// seenRev its text's revision then
	linkBase  string
	atPassage int
	atLine    int // the line its cursor was on as the passage was looked at
	seenRev   uint64
	// span is where a document kept in the script starts in the script's
	// text (runes), as its last write left the script at revision rev: a
	// key typed then goes into the script as the changed span alone
	// (spanInto)
	span struct {
		at  int
		rev uint64
		ok  bool
	}
}

// newDoc opens the window's document of a kind (not the script,
// which the window makes itself): read from its home.
func newDoc(w *MainWindow, kind docKind) *document {
	d := &document{w: w, kind: kind}
	d.ed = editor.New("")
	d.ed.Markdown, d.ed.Syntax = true, true
	d.ed.PageColumns = markdownColumns
	d.ed.AccessibleName = kind.String() + " text"
	d.ed.OnEnter = func() bool {
		if w.settings != nil && !w.settings.GetBoolean("auto-indent") {
			return false
		}
		return continueList(d.ed) // prose: lists go on
	}
	d.folds = newFolds(w, d.ed, markdownOutline)
	d.ed.OnTab = d.folds.cycleHere // at a heading's start, as in the script: its fold
	d.vim = attachVim(w, d.ed, false)
	d.ed.OnChanged = d.changed
	d.ed.OnCursorChanged = d.cursorMoved
	family, size := "", 0
	if w.settings != nil {
		family, size = w.settings.GetString("font-family"), w.settings.GetInt("font-size")
		d.ed.SetLineNumbers(w.settings.GetBoolean("show-line-numbers"))
	}
	d.view = container.NewThemeOverride(editor.NewPageScroll(d.ed, 0), newEditorTheme(family, size))
	d.ed.SetOffstage(true) // out of sight until shown (docsOnStage)
	d.reload()
	return d
}

// scriptDoc is the window's script as a document: its editor is
// w.textEditor, its home the window's file.
func (w *MainWindow) scriptDoc() *document {
	if w.script == nil || w.script.ed != w.textEditor {
		w.script = &document{w: w, kind: docScript, ed: w.textEditor, vim: w.vim, view: w.editorView}
	}
	return w.script
}

// mainEditor is the editor in the main area, nil when a view is there.
func (w *MainWindow) mainEditor() *editor.ScriptEditor {
	if d := w.main(); d != nil {
		return d.ed
	}
	return nil
}

// writeEditor is the editor the writer's commands act on: the main
// editor, the script's if a view is there.
func (w *MainWindow) writeEditor() *editor.ScriptEditor {
	if e := w.mainEditor(); e != nil {
		return e
	}
	return w.textEditor
}

// editorMarkdown reports whether the main editor holds Markdown: a
// Markdown file, or the source or the notes in the middle.
func (w *MainWindow) editorMarkdown() bool {
	if d := w.main(); d != nil && d.kind != docScript {
		return true
	}
	return w.markdown
}

// mainVim is the main editor's vim state (the script's when a view is
// there).
func (w *MainWindow) mainVim() *vim {
	if d := w.main(); d != nil && d.vim != nil {
		return d.vim
	}
	return w.vim
}

// markdown reports whether the document is Markdown (its editor's mode).
func (d *document) markdown() bool {
	if d.kind == docScript {
		return d.w.markdown
	}
	return true
}

// inScript reports whether the document is a stretch of the script's
// text: saved with it, shared with it.
func (d *document) inScript() bool {
	return d.kind == docNotes || d.kind == docSource && d.ref.Origin == sourceInside
}

// sharedAs is the Doc of the session the document is shared through:
// "script" for the script and what it holds; for a source kept outside
// the script, its own Doc (collab_source.go); "" for none (not shared).
func (d *document) sharedAs() string {
	if d.kind == docScript || d.inScript() {
		return collab.Script
	}
	if d.kind == docSource && d.ref == d.w.sourceRef() {
		return d.w.sharedSource()
	}
	return ""
}

// where is where the document is kept, as the status line says it.
func (d *document) where() string {
	switch d.kind {
	case docNotes:
		return "kept at the script's end"
	case docSource:
		return d.ref.describe()
	}
	if f := d.w.currentFile; f != "" {
		return filepath.Base(f)
	}
	return "not saved yet"
}

// title is how the window's title names the document.
func (d *document) title() string {
	switch d.kind {
	case docSource:
		switch {
		case d.ref.Path != "":
			return filepath.Base(d.ref.Path)
		case d.ref.Entry != "":
			return path.Base(d.ref.Entry)
		}
		return "Source"
	case docNotes:
		return "Notes"
	}
	return d.w.scriptName()
}

// Text is the document's text as it is now.
func (d *document) Text() string { return d.ed.Text() }

// SetText puts text in the document's editor with no undo history, the
// cursor at its start (the writer's edit: a home in the script follows).
func (d *document) SetText(text string) {
	d.ed.SetText(text)
}

// dirty reports whether the document has changes its home has not.
func (d *document) dirty() bool {
	return d.kind != docScript && !sameString(d.ed.Text(), d.stored)
}

// home is the document's text as its home has it now.
func (d *document) home() (string, error) {
	w := d.w
	switch d.kind {
	case docNotes:
		return w.notesMarks().of(w.textEditor.Text()), nil
	case docSource:
		return w.homeSourceText(d.ref)
	}
	return w.textEditor.Text(), nil
}

// reload reads the document from its home again, its undo history gone.
func (d *document) reload() {
	if d.kind == docSource {
		d.ref = d.w.sourceRef()
	}
	text, err := d.home()
	if err != nil {
		d.w.setStatus("Could not read the " + d.kind.String() + ": " + err.Error())
	}
	d.fromHome = true
	d.ed.SetText(text)
	d.fromHome = false
	d.ed.Buffer().TakeChange()
	d.span.ok = false
	d.stored, d.homeSeen = text, text
	d.linkBase, d.atPassage, d.atLine, d.seenRev = text, -1, -1, d.ed.Buffer().Rev()
}

// rehome is a source moved to another home (Keep Source in the Script):
// the document stays open, with its history, kept there now.
func (d *document) rehome() {
	if d.kind != docSource {
		return
	}
	d.ref = d.w.sourceRef()
	if home, err := d.home(); err == nil {
		d.homeSeen = home
	}
	d.stored = d.Text()
	d.ed.Buffer().TakeChange()
	d.span.ok = false
}

// store writes the document to its home if it changed.
func (d *document) store() error {
	text := d.ed.Text()
	if sameString(text, d.stored) {
		d.ed.Buffer().TakeChange() // back as it was: nothing to write
		return nil
	}
	w := d.w
	if d.inScript() {
		start, end, oldEnd, ok := d.ed.Buffer().TakeChange()
		if ok && d.spanInto(text, start, end, oldEnd) {
			w.updateTitle()
			return nil
		}
	}
	switch d.kind {
	case docNotes:
		var from int
		var block string
		d.intoScript(func() { from, block = w.writeNotesAt(text) })
		d.spanFrom(from, block, w.notesMarks())
	case docSource:
		switch d.ref.Origin {
		case sourceInside: // into the script's block
			e := w.textEditor
			start, end, block := withSourceBlock(e.Text(), text)
			d.intoScript(func() { e.Edit(func(b *buffer.Buffer) { replaceLeast(b, start, end, block) }) })
			d.spanFrom(start, block, sourceMarks)
		case sourceBundle:
			if err := w.saveBundleSource(text); err != nil {
				return err
			}
		case sourceKept, sourceBeside:
			if err := writeFileAtomic(d.ref.Path, []byte(text)); err != nil {
				return fmt.Errorf("could not save the source: %v", err)
			}
		default:
			return fmt.Errorf("the script has no source to save it to")
		}
		if w.source != nil {
			w.source.modTime = time.Time{} // read again
		}
	}
	d.stored = text
	if home, err := d.home(); err == nil {
		d.homeSeen = home // what the home holds now (a "*/" kept as "* /")
	}
	w.updateTitle()
	return nil
}

// marks are the marks of the block a document kept in the script is in.
func (d *document) marks() notesMarks {
	if d.kind == docNotes {
		return d.w.notesMarks()
	}
	return sourceMarks
}

// spanFrom notes where the document's text starts in the script, its
// block just written at from: the block's opening line before it.
func (d *document) spanFrom(from int, block string, m notesMarks) {
	d.span.ok = false
	open := m.open + "\n"
	i := strings.Index(block, open)
	if i < 0 {
		return
	}
	d.span.at = from + utf8.RuneCountInString(block[:i+len(open)])
	d.span.rev, d.span.ok = d.w.textEditor.Buffer().Rev(), true
}

// spanInto writes the document's change, [start, end) of text that was
// [start, oldEnd) of d.stored, into its stretch of the script as that
// span alone: what a key costs is the key, not the stretch (a novel kept
// in the script). It reports false when the whole stretch is to be
// written instead (replaceLeast, which finds the same span the long way):
// the script changed since the document last wrote there, or the text
// holds what its block writes otherwise (a "*/" kept as "* /", a mark of
// a block), or the change is in its last new lines (the block keeps
// none) or makes it empty (no block).
func (d *document) spanInto(text string, start, end, oldEnd int) bool {
	e := d.w.textEditor
	old := d.stored
	if !d.span.ok || d.span.rev != e.Buffer().Rev() {
		return false
	}
	m := d.marks()
	for _, s := range []string{old, text} {
		if m.otherwise(s) || strings.TrimSpace(s) == "" {
			return false
		}
	}
	n := d.ed.Buffer().Len()
	nOld := n - end + oldEnd
	trail := func(s string) int { return len(s) - len(strings.TrimRight(s, "\n")) }
	if oldEnd+trail(old) > nOld || end+trail(text) > n {
		return false
	}
	bs := byteAt(old, start)
	bo := bs + byteAt(old[bs:], oldEnd-start)
	if !strings.HasPrefix(d.homeSeen, old[:bo]) { // the home as last seen: the same text up to the change
		return false
	}
	ins := d.ed.Buffer().Slice(start, end)
	from, to := d.span.at+start, d.span.at+oldEnd
	d.intoScript(func() { e.Edit(func(b *buffer.Buffer) { replaceKeeping(b, from, to, ins) }) })
	d.homeSeen = d.homeSeen[:bs] + ins + d.homeSeen[bo:]
	d.stored = text
	d.span.rev = e.Buffer().Rev()
	return true
}

// otherwise reports whether s holds what the block writes as something
// else ("*/" as "* /"), or would read as a mark of a block: then the
// block's text is not s rune for rune.
func (m notesMarks) otherwise(s string) bool {
	opener := m.open
	if i := strings.IndexByte(opener, ' '); i > 0 {
		opener = opener[:i] // "/*", "<!--"
	}
	return strings.Contains(s, m.close) || strings.Contains(s, m.escaped) || strings.Contains(s, opener)
}

// byteAt is where the rune at offset r of s starts, in bytes.
func byteAt(s string, r int) int {
	if r <= 0 {
		return 0
	}
	n := 0
	for i := range s {
		if n == r {
			return i
		}
		n++
	}
	return len(s)
}

// intoScript makes write, a change of the script, one undo step with the
// document's writes just before it (nothing else wrote there between):
// the script's step on top is tagged with the document while it is its.
func (d *document) intoScript(write func()) {
	e := d.w.textEditor
	ours := e.UndoTag() == d
	n := e.UndoSteps()
	d.stored = d.ed.Text()
	d.writing = true // the script's change comes back to followHome: its own
	write()
	d.writing = false
	if e.UndoSteps() > n {
		if ours && n > 0 {
			e.JoinStepsSince(n - 1)
		}
		e.TagLastStep(d)
	}
}

// replaceLeast replaces [start, end) with text, changing only what
// differs: a co-writer's edit elsewhere in the stretch, and the undo
// steps around it, are not touched.
func replaceLeast(b *buffer.Buffer, start, end int, text string) {
	old, nw := []rune(b.Slice(start, end)), []rune(text)
	p := 0
	for p < len(old) && p < len(nw) && old[p] == nw[p] {
		p++
	}
	q := 0
	for q < len(old)-p && q < len(nw)-p && old[len(old)-1-q] == nw[len(nw)-1-q] {
		q++
	}
	if p == len(old)-q && p == len(nw)-q {
		return
	}
	replaceKeeping(b, start+p, end-q, string(nw[p:len(nw)-q]))
}

// replaceKeeping replaces [from, to) with text, the cursor and the
// selection kept where they were, with their text: a document writing
// into the script leaves the script's cursor in its place.
func replaceKeeping(b *buffer.Buffer, from, to int, text string) {
	cursor := b.Cursor()
	anchor, s1 := b.Selection()
	if anchor == cursor {
		anchor = s1
	}
	grow := utf8.RuneCountInString(text) - (to - from)
	keep := func(off int) int {
		switch {
		case off >= to:
			return off + grow
		case off > from:
			return from
		}
		return off
	}
	b.Replace(from, to, text)
	b.SetCursor(keep(anchor), false)
	b.SetCursor(keep(cursor), keep(anchor) != keep(cursor))
}

// changed is a change in the document's editor.
func (d *document) changed(string) {
	w := d.w
	if !d.fromHome && d.inScript() {
		_ = d.store() // a stretch of the script: written into it at once
	}
	if !d.fromHome && d.kind == docSource && !d.inScript() {
		if name := d.sharedAs(); name != "" { // a source shared in a Doc of its own
			w.shareSourceEdit(name, d.ed.Text())
		}
	}
	if w.main() == d {
		w.mainChanged()
	}
	if !d.inScript() {
		w.updateTitle() // unsaved
	}
}

// cursorMoved is the document's cursor moving.
func (d *document) cursorMoved() {
	if d.w.main() == d {
		d.w.mainCursorMoved()
	}
}

// followHome brings a document kept in the script up to date after the
// script changed (Undo there, a co-writer): the stretch's changes as
// edits outside its undo history, the cursor kept with its text.
func (d *document) followHome() {
	if !d.inScript() || d.writing || d.moving {
		return
	}
	home, err := d.home()
	if err != nil || sameString(home, d.homeSeen) { // not changed there since
		return
	}
	old := d.ed.Text()
	edits := collab.Edits(old, home)
	d.fromHome = true
	d.ed.Edit(func(b *buffer.Buffer) {
		for i := len(edits) - 1; i >= 0; i-- { // from the end: the positions before stay
			e := edits[i]
			b.Remote(e.Pos, e.Pos+e.Del, e.Text)
		}
	})
	d.fromHome = false
	d.ed.Buffer().TakeChange() // the script's change: not written back
	d.span.ok = false
	d.stored, d.homeSeen = d.ed.Text(), home
}

// The window's open documents (besides the script).

// docWriting reports whether an open document is writing into the
// script just now: the script's change is its own.
func (w *MainWindow) docWriting() bool {
	for _, d := range []*document{w.srcDoc, w.notesDoc} {
		if d != nil && d.writing {
			return true
		}
	}
	return false
}

// openDocs are the documents open besides the script.
func (w *MainWindow) openDocs() []*document {
	var out []*document
	for _, d := range []*document{w.srcDoc, w.notesDoc} {
		if d != nil {
			out = append(out, d)
		}
	}
	return out
}

// sourceDoc is the script's source as a document, opened (read) at the
// first need; nil without a source.
func (w *MainWindow) sourceDoc() *document {
	ref := w.sourceRef()
	if ref.Origin == sourceNone {
		return nil
	}
	if d := w.srcDoc; d != nil && d.ref == ref {
		return d
	}
	if w.srcDoc != nil { // another source now (a rename, Keep Source in the Script)
		_ = w.srcDoc.store()
	}
	w.srcDoc = newDoc(w, docSource)
	w.srcDoc.followShared() // shared: the Doc's text, the peers' edits in it
	return w.srcDoc
}

// docsOnStage tells each open document's editor whether it is in sight.
// One out of sight (behind another middle tab, a source open only to
// follow a shared Doc, the Source pane behind another pane or folded
// away) does not wrap the whole text again at each key, its own or a
// co-writer's: it is laid out when shown, or when something reads its
// layout (internal/editor: SetOffstage). Called wherever what is shown
// changes: the window laid out again, a middle tab or a pane to the front,
// the Source pane's Edit.
func (w *MainWindow) docsOnStage() {
	for _, d := range w.openDocs() {
		d.ed.SetOffstage(!w.docInSight(d))
	}
}

// docInSight reports whether the document's editor is on the screen: in
// front of the middle, or the source edited in the Source pane, that pane
// in front of its place and not folded down under a phone's board.
func (w *MainWindow) docInSight(d *document) bool {
	if id := w.mid.shown; id != "" && w.mid.docs[id] == d {
		return true
	}
	return d == w.srcDoc && w.source != nil && w.source.editing && w.dock != nil &&
		w.dock.inSight("source") && !(w.narrow && w.sheetDown)
}

// notesDoc is the script's notes as a document, opened at the first need.
func (w *MainWindow) notesDocument() *document {
	if w.notesDoc == nil {
		w.notesDoc = newDoc(w, docNotes)
	}
	return w.notesDoc
}

// closeDoc stores a document and lets it go (it is read again when next
// opened); the source pane's Edit with it.
func (w *MainWindow) closeDoc(d *document) error {
	if d == nil {
		return nil
	}
	if err := d.store(); err != nil {
		return err
	}
	switch d {
	case w.srcDoc:
		w.srcDoc = nil
		if w.source != nil && w.source.editing {
			w.source.endEdit()
		}
	case w.notesDoc:
		w.notesDoc = nil
	}
	return nil
}

// storeDocs writes every open document with changes to its home (File >
// Save, auto-save, closing); the first error is returned.
func (w *MainWindow) storeDocs() error {
	var first error
	for _, d := range w.openDocs() {
		if err := d.store(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// unsavedDocs reports whether an open document has changes its home has
// not (the title's mark, the close question).
func (w *MainWindow) unsavedDocs() bool {
	for _, d := range w.openDocs() {
		if d.dirty() {
			return true
		}
	}
	return false
}

// docsFollowScript brings the documents kept in the script up to date
// after it changed. A source whose block went (Undo of Keep Source in the
// Script, a draft restored) is let go as it is, not written back.
func (w *MainWindow) docsFollowScript() {
	for _, d := range w.openDocs() {
		if d.writing || d.moving { // its own change, or its home moving
			continue
		}
		if d.kind == docSource && d.inScript() && w.sourceRef().Origin != sourceInside {
			d.stored = d.Text()
			if id := w.mid.docTab(d); id != "" {
				w.closeMiddle(id)
			} else {
				_ = w.closeDoc(d)
			}
			w.setStatus("The source is no longer kept in the script: it was put back")
			continue
		}
		d.followHome()
	}
}

// leaveDocs is the window about to take another script (Open, New, an
// import, a join, another document type): the main editor holds the
// script again and the open documents are written and let go, while they
// still belong to the script they were opened for.
func (w *MainWindow) leaveDocs() {
	w.closeMiddleDocs()
	for _, d := range w.openDocs() {
		if err := w.closeDoc(d); err != nil && w.fyneWindow != nil {
			dialog.ShowError(err, w.fyneWindow)
		}
	}
}

// applyDocFonts gives the documents' editors the editor's font settings.
func (w *MainWindow) applyDocFonts(theme fyne.Theme) {
	for _, d := range w.openDocs() {
		d.view.Theme = theme
		d.view.Refresh()
		d.ed.SetLineNumbers(w.settings.GetBoolean("show-line-numbers"))
	}
}

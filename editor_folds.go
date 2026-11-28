// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Folding in the editor, as in the outline: a section or scene folded
// shows its heading with … after it, what it holds hidden (up to the blank
// line before the next). A heading's mark in the gutter (▾, folded ▸)
// folds and opens it when tapped; as in org-mode, Tab at the start of a
// heading cycles it (folded, its headings, all) and Shift+Tab the script
// (the sections, all headings, everything). View > Fold or Unfold Here,
// Fold All, Headings Only, Unfold All; vim keys za zc zo zR zM. The cursor
// getting into a fold opens it.

// folds are an editor's folds: the script's, or a document's in the
// main editor (the source, the notes: their Markdown headings), each its
// own (docs/MIDDLE.md).
type folds struct {
	w  *MainWindow
	ed *editor.ScriptEditor
	// items are the headings of a text that fold (the outline's)
	items func(text string) []outlineItem
	on    map[string]bool // folded, by foldKey
	state int             // where Shift+Tab is: foldShown, foldOverview or foldContents
	// blocks: Accolade's own blocks fold too (the script's editor:
	// block_folds.go), folded unless opened by hand (opened, by key)
	blocks bool
	opened map[string]bool
	bl     struct {
		rev  uint64
		ok   bool
		list []blockFold
	}
	labels map[string]blockLabel // the blocks' summaries, by key
	// laid is the outline as read for the text at rev, and its headings
	// that can fold: the editor asks at every layout (each key, each
	// cursor move), the text read once a change
	laid struct {
		rev      uint64
		markdown bool     // the file's kind then: the script's outline is Markdown's or a screenplay's
		starts   []string // the scene headings' starts then (the script's language)
		ok       bool
		items    []outlineItem
		headings map[int]bool
	}
}

// newFolds are the folds of an editor whose headings items reads.
func newFolds(w *MainWindow, e *editor.ScriptEditor, items func(string) []outlineItem) *folds {
	f := &folds{w: w, ed: e, items: items}
	e.Folds = f.ranges
	e.FoldLabel = f.label
	e.OnUnfold = f.unfoldAt
	e.Headings = f.headingLines
	e.OnFoldTapped = f.foldLine
	e.OnShiftTab = f.cycleAll
	return f
}

// outline is the editor's text's headings, read once for each text.
func (f *folds) outline() []outlineItem {
	l, rev, starts := &f.laid, f.ed.Buffer().Rev(), f.w.sceneStarts()
	if !l.ok || l.rev != rev || l.markdown != f.w.markdown || !slices.Equal(l.starts, starts) {
		l.items, l.headings = f.items(f.ed.Text()), nil
		l.rev, l.markdown, l.starts, l.ok = rev, f.w.markdown, starts, true
	}
	return f.laid.items
}

// The fold commands (View, vim's z keys, Tab and Shift+Tab) act on the
// main editor's folds; with a view in the main area there are none.

// mainFolds are the main editor's folds; nil (and the status says why)
// with a view there.
func (w *MainWindow) mainFolds() *folds {
	if d := w.main(); d != nil && d.folds != nil {
		return d.folds
	}
	if w.scriptInMain() {
		return w.scriptFolds
	}
	w.setStatus("Folding works in an editor: the Script tab of the middle has the script")
	return nil
}

// foldsOf are the folds of the editor e (the script's or a document's).
func (w *MainWindow) foldsOf(e *editor.ScriptEditor) *folds {
	for _, d := range w.openDocs() {
		if d.ed == e {
			return d.folds
		}
	}
	return w.scriptFolds
}

func (w *MainWindow) fold(open, toggle bool) {
	if f := w.mainFolds(); f != nil {
		f.fold(open, toggle)
	}
}

func (w *MainWindow) foldAll(open bool) {
	if f := w.mainFolds(); f != nil {
		f.foldAll(open)
	}
}

func (w *MainWindow) foldSections() {
	if f := w.mainFolds(); f != nil {
		f.foldOverview(f.outline())
	}
}

func (w *MainWindow) foldHeadings() {
	if f := w.mainFolds(); f != nil {
		f.foldHeadings()
	}
}

// outlineExtent is where item i of items runs in the script's lines: from
// its line to the next item at its level or higher (or the end).
func outlineExtent(items []outlineItem, i, lines int) (from, to int) {
	from, to = items[i].Line, lines
	for _, it := range items[i+1:] {
		if it.Level <= items[i].Level {
			return from, it.Line
		}
	}
	return from, to
}

// ranges are the folded items' hidden lines, and the blocks folded.
func (f *folds) ranges() []editor.Fold {
	out := f.blockRanges()
	if len(f.on) == 0 {
		return out
	}
	text := f.ed.Text()
	lines := strings.Count(text, "\n") + 1
	items := f.outline()
	all := strings.Split(text, "\n")
	for i, it := range items {
		if !f.on[foldKey(it)] {
			continue
		}
		_, to := outlineExtent(items, i, lines)
		if to-1 > it.Line+1 && strings.TrimSpace(all[to-1]) == "" {
			to-- // the blank line before the next stays
		}
		if to > it.Line+1 {
			out = append(out, editor.Fold{From: it.Line + 1, To: to})
		}
	}
	return out
}

// cursorItem is the outline item the cursor is in (-1: before any).
func (f *folds) cursorItem(items []outlineItem) int {
	line, _ := f.ed.Buffer().Position(f.ed.CursorOffset())
	at := -1
	for i, it := range items {
		if it.Line <= line {
			at = i
		}
	}
	return at
}

// fold folds (or with open false, opens; toggle: either) the section or
// scene the cursor is in.
func (f *folds) fold(open, toggle bool) {
	line, _ := f.ed.Buffer().Position(f.ed.CursorOffset())
	if inc, ok := f.w.includeOnLine(line); ok && (open || toggle) { // an include: its file shown (zo, za; highland_directives.go)
		f.w.showInclude(inc)
		return
	}
	if b := f.blockAt(line); b != nil { // in a block: it is shown, the cursor in it
		f.openBlock(b, open && !toggle)
		return
	}
	items := f.outline()
	i := f.cursorItem(items)
	if i < 0 {
		f.w.setStatus("Nothing to fold here: no section or scene above")
		return
	}
	key := foldKey(items[i])
	if toggle {
		open = f.on[key]
	}
	if open {
		delete(f.on, key)
	} else {
		if f.on == nil {
			f.on = map[string]bool{}
		}
		f.on[key] = true
		line := items[i].Line // the cursor to the heading, out of the fold
		f.ed.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	}
	f.ed.Refresh()
}

// foldAll folds every section and scene and Accolade's blocks, or opens
// them all (Unfold All, vim's zR).
func (f *folds) foldAll(open bool) {
	f.allBlocks(open)
	f.foldOutline(open)
}

// foldOutline folds every section and scene, or opens them all: the
// blocks as they are (Shift+Tab's cycle).
func (f *folds) foldOutline(open bool) {
	f.on = nil
	f.state = foldShown
	if !open {
		f.on = map[string]bool{}
		for _, it := range f.outline() {
			f.on[foldKey(it)] = true
		}
		items := f.outline()
		if i := f.cursorItem(items); i >= 0 {
			line := items[i].Line
			f.ed.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
		}
	}
	f.ed.Refresh()
}

// unfoldAt opens the fold of the heading on line (the cursor went in).
func (f *folds) unfoldAt(line int) {
	for _, it := range f.outline() {
		if it.Line == line {
			delete(f.on, foldKey(it))
		}
	}
}

// headingLines are the lines that can fold: the outline's headings with
// something under them.
func (f *folds) headingLines() map[int]bool {
	items := f.outline()
	if f.laid.headings != nil {
		return f.laid.headings
	}
	lines := f.ed.Buffer().LineCount()
	out := map[int]bool{}
	for i, it := range items {
		if _, to := outlineExtent(items, i, lines); to > it.Line+1 {
			out[it.Line] = true
		}
	}
	for _, b := range f.blocksOf() {
		out[b.first] = true
	}
	f.laid.headings = out
	return out
}

// foldLine folds or opens the heading on line (its fold mark tapped).
func (f *folds) foldLine(line int) {
	for _, b := range f.blocksOf() {
		if b.first == line {
			cur, _ := f.ed.Buffer().Position(f.ed.CursorOffset())
			shown := f.opened[b.key] || cur >= b.first && cur <= b.last
			f.openBlock(&b, !shown)
			return
		}
	}
	for _, it := range f.outline() {
		if it.Line != line {
			continue
		}
		key := foldKey(it)
		if f.on[key] {
			delete(f.on, key)
		} else {
			if f.on == nil {
				f.on = map[string]bool{}
			}
			f.on[key] = true
		}
		f.ed.Refresh()
		return
	}
}

// cycleHere is Tab at the start of a heading, as in org-mode: a section
// goes folded, then its headings shown (what they hold folded), then all
// shown; a scene folded or shown. It reports whether the cursor was there.
func (f *folds) cycleHere() bool {
	e := f.ed
	if s, t := e.Buffer().Selection(); s != t {
		return false
	}
	line, col := e.Buffer().Position(e.CursorOffset())
	return col == 0 && f.cycleAt(line)
}

// cycleAt cycles the fold of the heading on line, as Tab at its start; it
// reports whether line is a heading with something to fold.
func (f *folds) cycleAt(line int) bool {
	e := f.ed
	text := e.Text()
	items := f.outline()
	i := slices.IndexFunc(items, func(it outlineItem) bool { return it.Line == line })
	if i < 0 || !f.headingLines()[line] {
		return false
	}
	_, end := outlineExtent(items, i, strings.Count(text, "\n")+1)
	var inside []outlineItem // what the item holds
	for _, it := range items[i+1:] {
		if it.Line >= end {
			break
		}
		inside = append(inside, it)
	}
	if f.on == nil {
		f.on = map[string]bool{}
	}
	key := foldKey(items[i])
	anyFolded := slices.ContainsFunc(inside, func(it outlineItem) bool { return f.on[foldKey(it)] })
	switch {
	case f.on[key] && len(inside) > 0: // folded: its headings
		delete(f.on, key)
		for _, it := range inside {
			f.on[foldKey(it)] = true
		}
		f.w.setStatus("Headings shown")
	case f.on[key] || anyFolded: // its headings: all of it
		delete(f.on, key)
		for _, it := range inside {
			delete(f.on, foldKey(it))
		}
		f.w.setStatus("All shown")
	default: // all of it: folded
		f.on[key] = true
		f.w.setStatus("Folded")
	}
	e.Refresh()
	return true
}

// cycleAll is Shift+Tab, as in org-mode: the sections only (the scenes
// when there are none), then all headings, then everything.
func (f *folds) cycleAll() bool {
	items := f.outline()
	switch {
	case len(f.on) == 0:
		f.foldOverview(items)
		f.w.setStatus("Overview: the sections")
	case f.state == foldOverview:
		f.foldHeadings()
		f.w.setStatus("Contents: all headings")
	default:
		f.foldOutline(true)
		f.w.setStatus("Everything shown")
	}
	return true
}

// The fold states Shift+Tab goes through.
const (
	foldShown = iota
	foldOverview
	foldContents
)

// foldOverview folds the sections at the top level, or every scene in a
// script without sections.
func (f *folds) foldOverview(items []outlineItem) {
	f.on = map[string]bool{}
	top := 0
	for _, it := range items {
		if !it.Scene && (top == 0 || it.Level < top) {
			top = it.Level
		}
	}
	for _, it := range items {
		if top == 0 && it.Scene || !it.Scene && it.Level == top {
			f.on[foldKey(it)] = true
		}
	}
	f.state = foldOverview
	f.cursorToHeading(items)
	f.ed.Refresh()
}

// foldHeadings shows every heading with what they hold folded: the
// scenes folded, the sections open.
func (f *folds) foldHeadings() {
	items := f.outline()
	f.on = map[string]bool{}
	for _, it := range items {
		if it.Scene {
			f.on[foldKey(it)] = true
		}
	}
	f.state = foldContents
	f.cursorToHeading(items)
	f.ed.Refresh()
}

// cursorToHeading puts the cursor on the heading of what it is in, out of
// a fold.
func (f *folds) cursorToHeading(items []outlineItem) {
	if i := f.cursorItem(items); i >= 0 {
		line := items[i].Line
		f.ed.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	}
}

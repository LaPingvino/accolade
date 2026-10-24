package main

import (
	"slices"
	"strings"

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

// editorFoldRanges are the folded items' hidden lines.
func (w *MainWindow) editorFoldRanges() [][2]int {
	if len(w.editorFolds) == 0 {
		return nil
	}
	text := w.textEditor.Text()
	lines := strings.Count(text, "\n") + 1
	items := outlineOf(text, w.sceneStarts())
	all := strings.Split(text, "\n")
	var out [][2]int
	for i, it := range items {
		if !w.editorFolds[foldKey(it)] {
			continue
		}
		_, to := outlineExtent(items, i, lines)
		if to-1 > it.Line+1 && strings.TrimSpace(all[to-1]) == "" {
			to-- // the blank line before the next stays
		}
		if to > it.Line+1 {
			out = append(out, [2]int{it.Line + 1, to})
		}
	}
	return out
}

// cursorItem is the outline item the cursor is in (-1: before any).
func (w *MainWindow) cursorItem(items []outlineItem) int {
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
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
func (w *MainWindow) fold(open, toggle bool) {
	items := outlineOf(w.textEditor.Text(), w.sceneStarts())
	i := w.cursorItem(items)
	if i < 0 {
		w.setStatus("Nothing to fold here: no section or scene above")
		return
	}
	key := foldKey(items[i])
	if toggle {
		open = w.editorFolds[key]
	}
	if open {
		delete(w.editorFolds, key)
	} else {
		if w.editorFolds == nil {
			w.editorFolds = map[string]bool{}
		}
		w.editorFolds[key] = true
		line := items[i].Line // the cursor to the heading, out of the fold
		w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	}
	w.textEditor.Refresh()
}

// foldAll folds every section and scene, or opens them all.
func (w *MainWindow) foldAll(open bool) {
	w.editorFolds = nil
	w.foldState = foldShown
	if !open {
		w.editorFolds = map[string]bool{}
		for _, it := range outlineOf(w.textEditor.Text(), w.sceneStarts()) {
			w.editorFolds[foldKey(it)] = true
		}
		items := outlineOf(w.textEditor.Text(), w.sceneStarts())
		if i := w.cursorItem(items); i >= 0 {
			line := items[i].Line
			w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
		}
	}
	w.textEditor.Refresh()
}

// unfoldAt opens the fold of the heading on line (the cursor went in).
func (w *MainWindow) unfoldAt(line int) {
	for _, it := range outlineOf(w.textEditor.Text(), w.sceneStarts()) {
		if it.Line == line {
			delete(w.editorFolds, foldKey(it))
		}
	}
}

// headingLines are the lines that can fold: the outline's headings with
// something under them.
func (w *MainWindow) headingLines() map[int]bool {
	text := w.textEditor.Text()
	items := outlineOf(text, w.sceneStarts())
	lines := strings.Count(text, "\n") + 1
	out := map[int]bool{}
	for i, it := range items {
		if _, to := outlineExtent(items, i, lines); to > it.Line+1 {
			out[it.Line] = true
		}
	}
	return out
}

// foldLine folds or opens the heading on line (its fold mark tapped).
func (w *MainWindow) foldLine(line int) {
	for _, it := range outlineOf(w.textEditor.Text(), w.sceneStarts()) {
		if it.Line != line {
			continue
		}
		key := foldKey(it)
		if w.editorFolds[key] {
			delete(w.editorFolds, key)
		} else {
			if w.editorFolds == nil {
				w.editorFolds = map[string]bool{}
			}
			w.editorFolds[key] = true
		}
		w.textEditor.Refresh()
		return
	}
}

// cycleHere is Tab at the start of a heading, as in org-mode: a section
// goes folded, then its headings shown (what they hold folded), then all
// shown; a scene folded or shown. It reports whether the cursor was there.
func (w *MainWindow) cycleHere() bool {
	e := w.textEditor
	if s, t := e.Buffer().Selection(); s != t {
		return false
	}
	line, col := e.Buffer().Position(e.CursorOffset())
	return col == 0 && w.cycleAt(line)
}

// cycleAt cycles the fold of the heading on line, as Tab at its start; it
// reports whether line is a heading with something to fold.
func (w *MainWindow) cycleAt(line int) bool {
	e := w.textEditor
	text := e.Text()
	items := outlineOf(text, w.sceneStarts())
	i := slices.IndexFunc(items, func(it outlineItem) bool { return it.Line == line })
	if i < 0 || !w.headingLines()[line] {
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
	if w.editorFolds == nil {
		w.editorFolds = map[string]bool{}
	}
	key := foldKey(items[i])
	anyFolded := slices.ContainsFunc(inside, func(it outlineItem) bool { return w.editorFolds[foldKey(it)] })
	switch {
	case w.editorFolds[key] && len(inside) > 0: // folded: its headings
		delete(w.editorFolds, key)
		for _, it := range inside {
			w.editorFolds[foldKey(it)] = true
		}
		w.setStatus("Headings shown")
	case w.editorFolds[key] || anyFolded: // its headings: all of it
		delete(w.editorFolds, key)
		for _, it := range inside {
			delete(w.editorFolds, foldKey(it))
		}
		w.setStatus("All shown")
	default: // all of it: folded
		w.editorFolds[key] = true
		w.setStatus("Folded")
	}
	e.Refresh()
	return true
}

// cycleAll is Shift+Tab, as in org-mode: the sections only (the scenes
// when there are none), then all headings, then everything.
func (w *MainWindow) cycleAll() bool {
	items := outlineOf(w.textEditor.Text(), w.sceneStarts())
	switch {
	case len(w.editorFolds) == 0:
		w.foldOverview(items)
		w.setStatus("Overview: the sections")
	case w.foldState == foldOverview:
		w.foldHeadings()
		w.setStatus("Contents: all headings")
	default:
		w.foldAll(true)
		w.setStatus("Everything shown")
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
func (w *MainWindow) foldOverview(items []outlineItem) {
	w.editorFolds = map[string]bool{}
	top := 0
	for _, it := range items {
		if !it.Scene && (top == 0 || it.Level < top) {
			top = it.Level
		}
	}
	for _, it := range items {
		if top == 0 && it.Scene || !it.Scene && it.Level == top {
			w.editorFolds[foldKey(it)] = true
		}
	}
	w.foldState = foldOverview
	w.cursorToHeading(items)
	w.textEditor.Refresh()
}

// foldHeadings shows every heading with what they hold folded: the
// scenes folded, the sections open.
func (w *MainWindow) foldHeadings() {
	items := outlineOf(w.textEditor.Text(), w.sceneStarts())
	w.editorFolds = map[string]bool{}
	for _, it := range items {
		if it.Scene {
			w.editorFolds[foldKey(it)] = true
		}
	}
	w.foldState = foldContents
	w.cursorToHeading(items)
	w.textEditor.Refresh()
}

// cursorToHeading puts the cursor on the heading of what it is in, out of
// a fold.
func (w *MainWindow) cursorToHeading(items []outlineItem) {
	if i := w.cursorItem(items); i >= 0 {
		line := items[i].Line
		w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	}
}

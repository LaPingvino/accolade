package main

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Folding in the editor, as in the outline: a section or scene folded
// shows its heading with … after it, what it holds hidden (up to the blank
// line before the next). View > Fold or Unfold Here, Unfold All; vim keys
// za zc zo zR zM. The cursor getting into a fold opens it.

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

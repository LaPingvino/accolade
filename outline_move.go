package main

import "strings"

// Reordering the script by its outline: a scene goes with everything up to
// the next scene or a section around it, a section with all it holds.

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

// moveOutline moves item from (with what it holds) before item to, or to
// the end when to is len(items); it reports whether the text changed.
func moveOutline(text string, starts []string, from, to int) (string, bool) {
	items := outlineOf(text, starts)
	if from < 0 || from >= len(items) || to < 0 || to > len(items) {
		return text, false
	}
	lines := strings.Split(text, "\n")
	a, b := outlineExtent(items, from, len(lines))
	at := len(lines)
	if to < len(items) {
		at = items[to].Line
	}
	if at >= a && at <= b { // into itself, or where it is
		return text, false
	}
	block := append([]string(nil), lines[a:b]...)
	if len(block) > 0 && block[len(block)-1] != "" { // the last part: ends with a blank line
		block = append(block, "")
	}
	rest := append(append([]string(nil), lines[:a]...), lines[b:]...)
	if at > a {
		at -= b - a
	}
	if at == len(rest) && len(rest) > 0 && rest[len(rest)-1] != "" { // to the end: after a blank line
		rest = append(rest, "")
		at++
	}
	out := append(append(append([]string(nil), rest[:at]...), block...), rest[at:]...)
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + trailingNewlines(text), true
}

// trailingNewlines are the line breaks text ends with.
func trailingNewlines(text string) string { return text[len(strings.TrimRight(text, "\n")):] }

// moveOutlineItem moves an outline item in the window's script, as one undo
// step; to is an index in the outline (len: the end).
func (w *MainWindow) moveOutlineItem(from, to int) {
	text := w.textEditor.Text()
	if nw, ok := moveOutline(text, w.sceneStarts(), from, to); ok {
		w.replaceText(text, nw)
		w.setStatus("Moved")
	}
}

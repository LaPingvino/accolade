// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"

	"github.com/LaPingvino/lexington/fountain"
)

// The edits the Notes pane makes to an item (open_items.go): each an
// edit of the text, made as one undo step (notes_pane.go). They are
// worked out here, from the text and the item as it was read.

// textEdit replaces [Start, End) (runes) with Text.
type textEdit struct {
	Start, End int
	Text       string
}

// apply is the text with the edit made.
func (e textEdit) apply(text string) string {
	r := []rune(text)
	return string(r[:e.Start]) + e.Text + string(r[e.End:])
}

// still tells whether the item is where it was read, as it was: the
// text may have changed since (typing, a co-writer).
func (it openItem) still(text string) bool {
	r := []rune(text)
	return it.Start >= 0 && it.End <= len(r) && it.Start <= it.End && string(r[it.Start:it.End]) == it.Raw
}

// commentMarks are a boneyard's marks in the document's kind, and the
// close mark as it is kept inside.
func commentMarks(markdown bool) (open, close, escaped string) {
	if markdown {
		return "<!--", "-->", "-- >"
	}
	return "/*", "*/", "* /"
}

// written is the item as written with text as what it says.
func (it openItem) written(text string, markdown bool) string {
	text = strings.TrimRight(text, " \t\n")
	switch it.Shape {
	case shapeNote:
		text = strings.ReplaceAll(text, "]]", "] ]")
		for strings.Contains(text, "\n\n") { // a blank line would end it
			text = strings.ReplaceAll(text, "\n\n", "\n")
		}
		inner := it.Prefix + text
		if it.Prefix == "" && strings.HasPrefix(it.Raw, "[[ ") && strings.HasSuffix(it.Raw, " ]]") {
			inner = " " + text + " "
		}
		return "[[" + inner + "]]"
	case shapeBoneyard:
		open, close, escaped := commentMarks(markdown)
		text = strings.ReplaceAll(it.Prefix+text, close, escaped)
		if it.Single && !strings.Contains(text, "\n") && it.Label == "" {
			return open + " " + text + " " + close
		}
		label := ""
		if it.Label != "" {
			label = " " + it.Label
		}
		return open + label + "\n" + text + "\n" + close
	case shapeSynopsis:
		var lines []string
		for _, l := range strings.Split(text, "\n") {
			if l = strings.TrimSpace(l); l != "" { // a blank line would make two
				lines = append(lines, "= "+l)
			}
		}
		return strings.Join(lines, "\n")
	case shapePara:
		_, close, escaped := commentMarks(markdown)
		return strings.ReplaceAll(text, close, escaped)
	}
	return it.Raw
}

// editTo is the edit that makes the item say text; an empty text takes
// it out.
func (it openItem) editTo(text, now string, markdown bool) textEdit {
	if strings.TrimSpace(text) == "" {
		return it.removal(now)
	}
	return textEdit{it.Start, it.End, it.written(text, markdown)}
}

// removal is the edit that takes the item out of the text: a note with
// the space before it, a whole line with the line, and no blank line
// left doubled.
func (it openItem) removal(text string) textEdit {
	r := []rune(text)
	s, e := it.Start, it.End
	ls := s
	for ls > 0 && r[ls-1] != '\n' {
		ls--
	}
	le := e
	for le < len(r) && r[le] != '\n' {
		le++
	}
	blank := func(a, b int) bool { return strings.TrimSpace(string(r[a:b])) == "" }
	if !blank(ls, s) || !blank(e, le) { // on a line with more: only it, and a space before
		for s > ls && (r[s-1] == ' ' || r[s-1] == '\t') {
			s--
		}
		return textEdit{s, e, ""}
	}
	s, e = ls, le
	if e < len(r) {
		e++ // its new line
	}
	// a blank line before and after it: one goes
	prevBlank := s == 0 || s >= 2 && r[s-1] == '\n' && r[s-2] == '\n'
	if prevBlank {
		for e < len(r) && r[e] == '\n' {
			e++
		}
		if s == 0 {
			return textEdit{s, e, ""}
		}
		if e < len(r) {
			return textEdit{s, e, ""} // the blank line before it stays
		}
		for s > 0 && r[s-1] == '\n' { // at the end: no blank lines left after the text
			s--
		}
		return textEdit{s, e, "\n"}
	}
	return textEdit{s, e, ""}
}

// restorable tells whether Restore is offered: on every boneyard (Joop:
// "err on the side of allowing": prose restores as action).
func (it openItem) restorable() bool { return it.Shape == shapeBoneyard }

// restore is the edit that takes the boneyard's marks away, its inside
// back in the script: a scene kept under its OMITTED heading comes back
// as Restore Scene brings it (its number with it); ok false if that
// fails (its block is not as Restore Scene wrote it).
func (it openItem) restore(text string, markdown bool) (textEdit, bool) {
	r := []rune(text)
	if it.Kept {
		out, ok := fountain.RestoreScene(string(r[it.HeadStart:it.End]))
		if !ok {
			return textEdit{}, false
		}
		return textEdit{it.HeadStart, it.End, out}, true
	}
	inside := it.Prefix + it.Text
	if it.Label != "" { // Accolade's label: not the script's
		inside = it.Text
	}
	s, e := it.Start, it.End
	ls := s
	for ls > 0 && r[ls-1] != '\n' {
		ls--
	}
	le := e
	for le < len(r) && r[le] != '\n' {
		le++
	}
	if strings.TrimSpace(string(r[ls:s])) != "" || strings.TrimSpace(string(r[e:le])) != "" {
		return textEdit{s, e, inside}, true // within a line: its words in place
	}
	// lines of their own: blank lines around them, so that a heading or a
	// cue reads as one
	s, e = ls, le
	pre, post := "", ""
	if s > 0 && !(s >= 2 && r[s-1] == '\n' && r[s-2] == '\n') {
		pre = "\n"
	}
	if e < len(r) && !(e+1 < len(r) && r[e+1] == '\n') {
		post = "\n"
	}
	return textEdit{s, e, pre + inside + post}, true
}

// newUnsorted is the edit that adds a paragraph to the unsorted notes
// (the notes block at the end), making the block if there is none.
func newUnsorted(text, note string, m notesMarks) textEdit {
	note = strings.TrimSpace(note)
	notes := strings.TrimRight(m.of(text), "\n")
	if strings.TrimSpace(notes) != "" {
		notes += "\n\n"
	}
	from, block := m.edit(text, strings.ReplaceAll(notes, m.escaped, m.close)+note)
	return textEdit{from, len([]rune(text)), block}
}

// removeUnsorted is the edit that takes an unsorted note out: the block
// written again without it (gone with the last).
func removeUnsorted(text string, it openItem, m notesMarks) textEdit {
	r := []rune(text)
	from := m.start(text)
	inside := m.of(text)
	if from > it.Start || it.End > len(r) {
		return textEdit{it.Start, it.End, ""}
	}
	// the block's inside with the paragraph out, unescaped (edit escapes)
	var kept []string
	gone := false
	for _, p := range strings.Split(inside, "\n\n") {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if !gone && strings.Trim(p, " \t\n") == strings.Trim(it.Raw, " \t\n") {
			gone = true // the first one so: the item (another the same stays)
			continue
		}
		kept = append(kept, strings.Trim(p, "\n"))
	}
	notes := strings.ReplaceAll(strings.Join(kept, "\n\n"), m.escaped, m.close)
	f, block := m.edit(text, notes)
	return textEdit{f, len(r), block}
}

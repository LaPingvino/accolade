// Package buffer is the text model of Accolade's editor: the text, the
// cursor and selection, edits and their undo history. It has no GUI;
// positions are rune offsets into the text. See docs/EDITOR_WIDGET.md.
package buffer

import (
	"strings"
	"unicode"
)

// change is one replacement: at pos, removed was replaced by inserted.
type change struct {
	pos               int
	removed, inserted []rune
}

// step is what one Undo undoes: a group of changes plus where the cursor
// and selection were before and after.
type step struct {
	changes                    []change
	beforeCursor, beforeAnchor int
	afterCursor, afterAnchor   int
	// typing marks a step that further typed letters may join
	typing bool
}

// Buffer holds the text being edited.
type Buffer struct {
	text []rune
	// the selection runs between anchor and cursor; equal means none
	cursor, anchor int

	undo, redo []step
	group      *step // the step being built by Group
	depth      int
}

// New creates a buffer holding text, with the cursor at the start.
func New(text string) *Buffer { return &Buffer{text: []rune(text)} }

// Text is the whole text.
func (b *Buffer) Text() string { return string(b.text) }

// Len is the length of the text in runes.
func (b *Buffer) Len() int { return len(b.text) }

// Slice is the text between two offsets.
func (b *Buffer) Slice(start, end int) string {
	start, end = b.clamp(start), b.clamp(end)
	if start > end {
		start, end = end, start
	}
	return string(b.text[start:end])
}

// SetText replaces the whole text and forgets the undo history (loading a
// file).
func (b *Buffer) SetText(text string) {
	b.text = []rune(text)
	b.cursor, b.anchor = 0, 0
	b.undo, b.redo = nil, nil
}

// Cursor is the cursor offset.
func (b *Buffer) Cursor() int { return b.cursor }

// Selection returns the selected range, start <= end; equal when nothing
// is selected.
func (b *Buffer) Selection() (start, end int) {
	if b.anchor < b.cursor {
		return b.anchor, b.cursor
	}
	return b.cursor, b.anchor
}

// HasSelection reports whether some text is selected.
func (b *Buffer) HasSelection() bool { return b.cursor != b.anchor }

// SelectedText is the selected text.
func (b *Buffer) SelectedText() string { s, e := b.Selection(); return string(b.text[s:e]) }

// SetCursor moves the cursor; with extend the selection grows from its
// anchor, otherwise the selection is dropped.
func (b *Buffer) SetCursor(offset int, extend bool) {
	b.cursor = b.clamp(offset)
	if !extend {
		b.anchor = b.cursor
	}
	b.breakTyping()
}

// Select selects [start, end) with the cursor at end.
func (b *Buffer) Select(start, end int) {
	b.anchor, b.cursor = b.clamp(start), b.clamp(end)
	b.breakTyping()
}

// SelectAll selects the whole text.
func (b *Buffer) SelectAll() { b.Select(0, len(b.text)) }

// Insert types text: it replaces the selection, or goes in at the cursor.
// Consecutive typed letters undo together, up to a word separator.
func (b *Buffer) Insert(text string) {
	start, end := b.Selection()
	r := []rune(text)
	if !b.HasSelection() && len(r) == 1 && !isSeparator(r[0]) && b.joinTyping(start, r[0]) {
		return
	}
	b.replace(start, end, r, len(r) == 1 && !isSeparator(r[0]))
}

// Replace replaces the text in [start, end) with text as one undo step,
// leaving the cursor after the new text.
func (b *Buffer) Replace(start, end int, text string) {
	start, end = b.clamp(start), b.clamp(end)
	if start > end {
		start, end = end, start
	}
	b.replace(start, end, []rune(text), false)
}

// DeleteBackward deletes the selection, or the rune before the cursor.
func (b *Buffer) DeleteBackward() {
	start, end := b.Selection()
	if start == end {
		if start == 0 {
			return
		}
		start--
	}
	b.replace(start, end, nil, false)
}

// DeleteForward deletes the selection, or the rune after the cursor.
func (b *Buffer) DeleteForward() {
	start, end := b.Selection()
	if start == end {
		if end == len(b.text) {
			return
		}
		end++
	}
	b.replace(start, end, nil, false)
}

// Group makes all edits done by f one undo step (formatting a line,
// replacing all matches).
func (b *Buffer) Group(f func()) {
	if b.depth == 0 {
		b.breakTyping()
		b.group = &step{beforeCursor: b.cursor, beforeAnchor: b.anchor}
	}
	b.depth++
	f()
	b.depth--
	if b.depth == 0 {
		g := b.group
		b.group = nil
		if len(g.changes) > 0 {
			g.afterCursor, g.afterAnchor = b.cursor, b.anchor
			b.undo = append(b.undo, *g)
			b.redo = nil
		}
	}
}

// CanUndo and CanRedo report whether there is something to undo or redo.
func (b *Buffer) CanUndo() bool { return len(b.undo) > 0 }
func (b *Buffer) CanRedo() bool { return len(b.redo) > 0 }

// Undo undoes the last step.
func (b *Buffer) Undo() bool {
	if len(b.undo) == 0 {
		return false
	}
	s := b.undo[len(b.undo)-1]
	b.undo = b.undo[:len(b.undo)-1]
	for i := len(s.changes) - 1; i >= 0; i-- {
		c := s.changes[i]
		b.splice(c.pos, c.pos+len(c.inserted), c.removed)
	}
	b.cursor, b.anchor = s.beforeCursor, s.beforeAnchor
	b.redo = append(b.redo, s)
	return true
}

// Redo redoes the last undone step.
func (b *Buffer) Redo() bool {
	if len(b.redo) == 0 {
		return false
	}
	s := b.redo[len(b.redo)-1]
	b.redo = b.redo[:len(b.redo)-1]
	for _, c := range s.changes {
		b.splice(c.pos, c.pos+len(c.removed), c.inserted)
	}
	b.cursor, b.anchor = s.afterCursor, s.afterAnchor
	s.typing = false
	b.undo = append(b.undo, s)
	return true
}

// replace records and applies one change.
func (b *Buffer) replace(start, end int, inserted []rune, typing bool) {
	if start == end && len(inserted) == 0 {
		return
	}
	c := change{pos: start, removed: append([]rune(nil), b.text[start:end]...), inserted: inserted}
	beforeCursor, beforeAnchor := b.cursor, b.anchor
	b.splice(start, end, inserted)
	b.cursor = start + len(inserted)
	b.anchor = b.cursor

	if b.group != nil {
		b.group.changes = append(b.group.changes, c)
		return
	}
	b.undo = append(b.undo, step{
		changes:      []change{c},
		beforeCursor: beforeCursor, beforeAnchor: beforeAnchor,
		afterCursor: b.cursor, afterAnchor: b.anchor,
		typing: typing,
	})
	b.redo = nil
}

// joinTyping adds a typed letter to the last undo step if that step was
// plain typing that ended exactly at the cursor.
func (b *Buffer) joinTyping(at int, r rune) bool {
	if b.group != nil || len(b.undo) == 0 {
		return false
	}
	last := &b.undo[len(b.undo)-1]
	if !last.typing || last.afterCursor != at || len(last.changes) != 1 {
		return false
	}
	c := &last.changes[0]
	if len(c.removed) != 0 || c.pos+len(c.inserted) != at {
		return false
	}
	b.splice(at, at, []rune{r})
	c.inserted = append(c.inserted, r)
	b.cursor, b.anchor = at+1, at+1
	last.afterCursor, last.afterAnchor = b.cursor, b.anchor
	b.redo = nil
	return true
}

func (b *Buffer) breakTyping() {
	if n := len(b.undo); n > 0 {
		b.undo[n-1].typing = false
	}
}

func (b *Buffer) splice(start, end int, inserted []rune) {
	b.text = append(b.text[:start], append(append([]rune(nil), inserted...), b.text[end:]...)...)
}

func (b *Buffer) clamp(off int) int { return max(0, min(off, len(b.text))) }

func isSeparator(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }

// Lines.

// LineCount is the number of lines (an empty text has one).
func (b *Buffer) LineCount() int { return strings.Count(string(b.text), "\n") + 1 }

// LineStart is the offset where line i (0-based) begins.
func (b *Buffer) LineStart(i int) int {
	if i <= 0 {
		return 0
	}
	n := 0
	for off, r := range b.text {
		if r == '\n' {
			n++
			if n == i {
				return off + 1
			}
		}
	}
	return len(b.text)
}

// LineEnd is the offset where line i ends (before its newline).
func (b *Buffer) LineEnd(i int) int {
	off := b.LineStart(i)
	for off < len(b.text) && b.text[off] != '\n' {
		off++
	}
	return off
}

// Line is the text of line i, without its newline.
func (b *Buffer) Line(i int) string { return string(b.text[b.LineStart(i):b.LineEnd(i)]) }

// Position is the line and column (both 0-based) of an offset.
func (b *Buffer) Position(offset int) (line, col int) {
	offset = b.clamp(offset)
	start := 0
	for i := 0; i < offset; i++ {
		if b.text[i] == '\n' {
			line++
			start = i + 1
		}
	}
	return line, offset - start
}

// Offset is the offset of a line and column, the column clamped to the
// line's length.
func (b *Buffer) Offset(line, col int) int {
	if line >= b.LineCount() {
		return len(b.text)
	}
	start, end := b.LineStart(line), b.LineEnd(line)
	return start + max(0, min(col, end-start))
}

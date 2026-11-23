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
	// tag is what the step means beyond its text, if anything (Tag):
	// the editor tells its owner when such a step is undone or redone
	tag any
}

// Buffer holds the text being edited.
type Buffer struct {
	text []rune
	// the selection runs between anchor and cursor; equal means none
	cursor, anchor int
	// others are more selections than that one, the primary (spans.go)
	others []Span
	// marks are places that move with the text, not selections (SetMarks)
	marks []int

	undo, redo []step
	group      *step // the step being built by Group
	depth      int

	// rev counts the text's changes; str is the text as a string, made
	// once per change (Text is asked for many times per keystroke)
	rev    uint64
	str    string
	strRev uint64
	strOK  bool

	// what changed since TakeChange: [chStart, chEnd) of the text now
	// was [chStart, chOld) then; chAll: replaced whole (SetText)
	chStart, chEnd, chOld int
	chSome, chAll         bool
}

// New creates a buffer holding text, with the cursor at the start.
func New(text string) *Buffer { return &Buffer{text: []rune(text)} }

// Text is the whole text. It is made once per change: the same string
// comes back until the text changes.
func (b *Buffer) Text() string {
	if !b.strOK || b.strRev != b.rev {
		b.str, b.strRev, b.strOK = string(b.text), b.rev, true
	}
	return b.str
}

// Rev is the text's revision: it changes whenever the text does.
func (b *Buffer) Rev() uint64 { return b.rev }

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
	b.rev++
	b.chAll = true
	b.cursor, b.anchor = 0, 0
	b.undo, b.redo = nil, nil
	for i, m := range b.marks {
		b.marks[i] = b.clamp(m)
	}
}

// At is the rune at offset i (0 outside the text).
func (b *Buffer) At(i int) rune {
	if i < 0 || i >= len(b.text) {
		return 0
	}
	return b.text[i]
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
	b.others = nil // moving keeps the primary selection only
	b.cursor = b.clamp(offset)
	if !extend {
		b.anchor = b.cursor
	}
	b.breakTyping()
}

// Select selects [start, end) with the cursor at end.
func (b *Buffer) Select(start, end int) {
	b.others = nil
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

// Amend replaces [start, end) with text as part of the last undo step:
// undone and redone with it, not a step of its own. The cursor and the
// selections stay where they were (moved with the text). It is for what
// follows from the last edit by itself (Accolade's learned notes): an
// Undo of the edit takes it away too. Nothing to undo, it is no step at
// all. Redo is forgotten, as by any edit.
func (b *Buffer) Amend(start, end int, text string) {
	start, end = b.clamp(start), b.clamp(end)
	if start > end {
		start, end = end, start
	}
	inserted := []rune(text)
	if start == end && len(inserted) == 0 {
		return
	}
	c := change{pos: start, removed: append([]rune(nil), b.text[start:end]...), inserted: inserted}
	cursor, anchor := b.cursor, b.anchor
	b.splice(start, end, inserted)
	b.cursor = shifted(cursor, start, end, len(inserted))
	b.anchor = shifted(anchor, start, end, len(inserted))
	b.redo = nil
	switch {
	case b.group != nil:
		b.group.changes = append(b.group.changes, c)
	case len(b.undo) > 0:
		last := &b.undo[len(b.undo)-1]
		last.changes = append(last.changes, c)
		last.typing = false
		last.afterCursor = shifted(last.afterCursor, start, end, len(inserted))
		last.afterAnchor = shifted(last.afterAnchor, start, end, len(inserted))
	}
}

// Steps is how many steps there are to undo: a mark for JoinSince.
func (b *Buffer) Steps() int { return len(b.undo) }

// JoinSince makes the steps after the first n one step (n is an earlier
// Steps), undone and redone as one: an action whose edits come in turns
// (Accolade's Issue, which waits for the pages between its edits). False
// if there is no step after n, or n is no longer a mark (undone past).
func (b *Buffer) JoinSince(n int) bool {
	if n < 0 || n >= len(b.undo) || b.group != nil {
		return false
	}
	first, last := b.undo[n], b.undo[len(b.undo)-1]
	j := step{beforeCursor: first.beforeCursor, beforeAnchor: first.beforeAnchor,
		afterCursor: last.afterCursor, afterAnchor: last.afterAnchor, tag: last.tag,
		changes: first.changes} // grown in place: a step joined at every key stays linear
	for _, s := range b.undo[n+1:] {
		j.changes = append(j.changes, s.changes...)
	}
	b.undo = append(b.undo[:n], j)
	return true
}

// Tag gives the last step a tag (nil takes it away): UndoTag and RedoTag
// report it while the step is next to be undone or redone. False if there
// is no step.
func (b *Buffer) Tag(tag any) bool {
	if len(b.undo) == 0 || b.group != nil {
		return false
	}
	b.undo[len(b.undo)-1].tag = tag
	b.undo[len(b.undo)-1].typing = false
	return true
}

// UndoTag is the tag of the step Undo would undo (nil: none, or no tag).
func (b *Buffer) UndoTag() any {
	if len(b.undo) == 0 {
		return nil
	}
	return b.undo[len(b.undo)-1].tag
}

// RedoTag is the tag of the step Redo would redo.
func (b *Buffer) RedoTag() any {
	if len(b.redo) == 0 {
		return nil
	}
	return b.redo[len(b.redo)-1].tag
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
	b.others = nil
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
	b.others = nil
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
	b.noteChange(start, end, len(inserted))
	b.text = append(b.text[:start], append(append([]rune(nil), inserted...), b.text[end:]...)...)
	b.rev++
	for i, m := range b.marks {
		b.marks[i] = markMoved(m, start, end, len(inserted))
	}
	for i := range b.others { // the other selections move with the text
		b.others[i].Anchor = shifted(b.others[i].Anchor, start, end, len(inserted))
		b.others[i].Cursor = shifted(b.others[i].Cursor, start, end, len(inserted))
	}
}

// noteChange adds [start, end) replaced by n runes to what changed
// since TakeChange.
func (b *Buffer) noteChange(start, end, n int) {
	if !b.chSome {
		b.chStart, b.chEnd, b.chOld, b.chSome = start, start+n, end, true
		return
	}
	e := b.chEnd // the changed stretch's end, in the text after this change
	switch {
	case e >= end:
		e += n - (end - start)
	case e > start:
		e = start + n
	}
	b.chOld += max(0, end-b.chEnd) // what this change takes beyond it was there before too
	b.chStart, b.chEnd = min(b.chStart, start), max(e, start+n)
}

// TakeChange is what changed in the text since the last TakeChange, and
// starts over: the runes [start, end) of the text now were [start, oldEnd)
// then. ok is false when nothing changed or the text was replaced whole
// (SetText): all of it is new.
func (b *Buffer) TakeChange() (start, end, oldEnd int, ok bool) {
	start, end, oldEnd, ok = b.chStart, b.chEnd, b.chOld, b.chSome && !b.chAll
	b.chSome, b.chAll = false, false
	return start, end, oldEnd, ok
}

func (b *Buffer) clamp(off int) int { return max(0, min(off, len(b.text))) }

// isSeparator: not of a word (a letter, a digit, a letter's mark).
func isSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r)
}

// Lines.

// LineCount is the number of lines (an empty text has one).
func (b *Buffer) LineCount() int { return strings.Count(b.Text(), "\n") + 1 }

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

// Remote replaces [start, end) with text for someone else (a peer editing
// the same script): not an undo step. The cursor and the selections keep
// their place in the text around it, and the undo and redo steps are
// moved past it, so Undo undoes only this buffer's own edits, where they
// now are. A step the change overlaps cannot be undone faithfully any
// more: it goes from the history, with every step before it (or, for
// Redo, after it).
func (b *Buffer) Remote(start, end int, text string) {
	start, end = b.clamp(start), b.clamp(end)
	if start > end {
		start, end = end, start
	}
	inserted := []rune(text)
	if start == end && len(inserted) == 0 {
		return
	}
	n := len(inserted)
	move := func(off int) int { return remoteMove(off, start, end, n) }
	b.cursor, b.anchor = move(b.cursor), move(b.anchor)
	others := b.others
	b.others = nil
	b.splice(start, end, inserted)
	for i := range others {
		others[i].Anchor, others[i].Cursor = move(others[i].Anchor), move(others[i].Cursor)
	}
	b.others = others
	if b.group != nil { // not between edits: nothing to keep in step with
		b.undo, b.redo = nil, nil
		return
	}
	b.undo = rebaseUndo(b.undo, start, end, n)
	b.redo = rebaseRedo(b.redo, start, end, n)
}

// rebaseUndo moves the undo steps past a change of [start, end) to n runes
// made after them (in the text as it is now), newest first. It returns the
// steps that are left: those from the newest one the change overlaps back
// are dropped.
func rebaseUndo(undo []step, start, end, n int) []step {
	for i := len(undo) - 1; i >= 0; i-- {
		s := &undo[i]
		// the change in the text as the step left it: the step's after
		s.afterCursor, s.afterAnchor = remoteMove(s.afterCursor, start, end, n), remoteMove(s.afterAnchor, start, end, n)
		for j := len(s.changes) - 1; j >= 0; j-- {
			c := &s.changes[j]
			cEnd := c.pos + len(c.inserted)
			switch {
			case end <= c.pos:
				c.pos += n - (end - start) // the change is before it
			case start >= cEnd:
				d := len(c.removed) - len(c.inserted) // after it: where it was before c
				start, end = start+d, end+d
			default:
				return append([]step(nil), undo[i+1:]...)
			}
		}
		s.beforeCursor, s.beforeAnchor = remoteMove(s.beforeCursor, start, end, n), remoteMove(s.beforeAnchor, start, end, n)
		s.typing = false
	}
	return undo
}

// rebaseRedo does the same for the redo steps, whose changes would apply
// to the text as it is now: the next to redo first.
func rebaseRedo(redo []step, start, end, n int) []step {
	for i := len(redo) - 1; i >= 0; i-- {
		s := &redo[i]
		s.beforeCursor, s.beforeAnchor = remoteMove(s.beforeCursor, start, end, n), remoteMove(s.beforeAnchor, start, end, n)
		for j := range s.changes {
			c := &s.changes[j]
			cEnd := c.pos + len(c.removed)
			switch {
			case end <= c.pos:
				c.pos += n - (end - start)
			case start >= cEnd:
				d := len(c.inserted) - len(c.removed)
				start, end = start+d, end+d
			default:
				return append([]step(nil), redo[i+1:]...) // it and the ones redone after it go
			}
		}
		s.afterCursor, s.afterAnchor = remoteMove(s.afterCursor, start, end, n), remoteMove(s.afterAnchor, start, end, n)
	}
	return redo
}

// remoteMove is where offset off goes when [start, end) becomes n runes
// (an insertion at off leaves off before it).
func remoteMove(off, start, end, n int) int {
	switch {
	case off <= start:
		return off
	case off >= end:
		return off + n - (end - start)
	}
	return start + n
}

// SetMarks keeps places in the text that move with it as it is edited,
// and are nothing else to it (a co-writer's cursor and selection, drawn
// by the editor): text typed at a mark goes after it, a mark in text
// deleted goes to where it was.
func (b *Buffer) SetMarks(marks []int) {
	b.marks = b.marks[:0]
	for _, m := range marks {
		b.marks = append(b.marks, b.clamp(m))
	}
}

// Marks are the places SetMarks keeps, moved with the text since.
func (b *Buffer) Marks() []int { return b.marks }

func markMoved(off, start, end, n int) int {
	switch {
	case off <= start:
		return off
	case off >= end:
		return off + n - (end - start)
	}
	return start
}

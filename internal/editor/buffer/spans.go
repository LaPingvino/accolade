package buffer

import "sort"

// Several selections at once (helix keys): the buffer's own, the primary,
// and others. Edits move them all with the text; EditSpans edits at each
// as one undo step; moving the cursor keeps the primary one only.

// Span is a selection from Anchor to Cursor (equal: a cursor alone).
type Span struct{ Anchor, Cursor int }

// Start and End are the span's offsets in order.
func (s Span) Start() int { return min(s.Anchor, s.Cursor) }
func (s Span) End() int   { return max(s.Anchor, s.Cursor) }

// Spans are the selections, the primary first.
func (b *Buffer) Spans() []Span {
	return append([]Span{{b.anchor, b.cursor}}, b.others...)
}

// SetSpans makes spans the selections, the first the primary; one that
// overlaps another is left out.
func (b *Buffer) SetSpans(spans []Span) {
	if len(spans) == 0 {
		return
	}
	var kept []Span
	for _, s := range spans {
		s = Span{b.clamp(s.Anchor), b.clamp(s.Cursor)}
		overlaps := false
		for _, k := range kept {
			if s.Start() < k.End() && k.Start() < s.End() || s.Start() == k.Start() && s.End() == k.End() {
				overlaps = true
				break
			}
		}
		if !overlaps {
			kept = append(kept, s)
		}
	}
	b.anchor, b.cursor = kept[0].Anchor, kept[0].Cursor
	b.others = kept[1:]
	b.breakTyping()
}

// KeepPrimary forgets the other selections.
func (b *Buffer) KeepPrimary() { b.others = nil }

// HasOthers reports whether there is more than one selection.
func (b *Buffer) HasOthers() bool { return len(b.others) > 0 }

// EditSpans edits at each selection, as one undo step: f says what to
// replace there (start to end) with what; afterwards each is a cursor
// after what it inserted.
func (b *Buffer) EditSpans(f func(s Span) (start, end int, insert string)) {
	spans := b.Spans()
	order := make([]int, len(spans))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool { return spans[order[i]].Start() > spans[order[j]].Start() })
	carets := make([]int, len(spans))
	b.Group(func() {
		var done []int
		for _, i := range order { // the last first: an edit moves what is after it only
			start, end, insert := f(spans[i])
			start, end = b.clamp(start), b.clamp(end)
			r := []rune(insert)
			for _, j := range done {
				carets[j] = shifted(carets[j], start, end, len(r))
			}
			b.replace(start, end, r, false)
			carets[i] = start + len(r)
			done = append(done, i)
		}
		b.cursor, b.anchor = carets[0], carets[0]
		b.others = b.others[:0]
		for _, c := range carets[1:] {
			b.others = append(b.others, Span{c, c})
		}
	})
}

// shifted is where off is after start..end became n runes: after them it
// moves with the end, inside them it goes to the new end.
func shifted(off, start, end, n int) int {
	switch {
	case off >= end:
		return off + n - (end - start)
	case off > start:
		return start + n
	}
	return off
}

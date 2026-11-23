// Package wrap lays the lines of a text out in rows of a fixed number of
// columns, for Accolade's grid-based editor (docs/EDITOR_WIDGET.md, step
// 2). A rune takes one column (the editor uses a monospace font), a wide
// one (CJK, emoji) two: see Width.
package wrap

import (
	"sort"
	"unicode"
)

// Width is how many columns r takes: 2 for East Asian wide and fullwidth
// characters and emoji, 0 for marks drawn on the character before them
// (combining accents, joiners, variation selectors, Hangul's medial and
// final jamo), 1 for the others.
func Width(r rune) int {
	switch {
	case r < 0x300:
		return 1
	case zeroWidth(r):
		return 0
	case r < 0x1100:
		return 1
	case r <= 0x115F, // Hangul jamo
		r >= 0x2E80 && r <= 0xA4CF && r != 0x303F,              // CJK, kana, Yi
		r >= 0xAC00 && r <= 0xD7A3,                             // Hangul
		r >= 0xF900 && r <= 0xFAFF,                             // CJK compatibility
		r >= 0xFE30 && r <= 0xFE4F,                             // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, r >= 0xFFE0 && r <= 0xFFE6, // fullwidth
		r >= 0x1F300 && r <= 0x1F64F, r >= 0x1F900 && r <= 0x1F9FF, // emoji
		r >= 0x20000 && r <= 0x3FFFD: // CJK extensions
		return 2
	}
	return 1
}

func zeroWidth(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Me) || (r >= 0x200B && r <= 0x200F) ||
		(r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || // direction marks
		(r >= 0x1160 && r <= 0x11FF)
}

// JoinsBefore tells whether b belongs to the cluster of a, the character
// before it (a cluster is drawn whole, and the cursor goes round it): b a
// mark (a vowel sign, a virama, a nukta, an accent) or a joiner, a after
// a zero-width joiner, or a Devanagari consonant after a virama (क्ष).
func JoinsBefore(a, b rune) bool {
	if a < 0x20 { // a line's start (a mark there has no letter)
		return false
	}
	return unicode.In(b, unicode.Mn, unicode.Mc, unicode.Me) || b == 0x200C || b == 0x200D || a == 0x200D ||
		a == 0x094D && b >= 0x0900 && b <= 0x097F
}

// Span is a stretch of the text, [Start, End) in runes, shown as Cols
// columns in one piece (a chip drawn in its place): its first rune takes
// the columns, the others none, and no row breaks inside it.
type Span struct {
	Start, End, Cols int
}

// line is one line being laid out: its runes, and for a line with spans
// (nil without) each rune's columns and the spans, from the line's start.
type line struct {
	r     []rune
	ws    []int
	spans [][2]int
}

// newLine is the line [base, base+len(r)) of a text with spans (sorted):
// those in it taken in.
func newLine(r []rune, base int, spans []Span) line {
	l := line{r: r}
	for _, s := range spans {
		if s.Start >= base+len(r) {
			break
		}
		if s.End <= base {
			continue
		}
		if l.ws == nil {
			l.ws = make([]int, len(r))
			for i, c := range r {
				l.ws[i] = Width(c)
			}
		}
		a, b := max(s.Start-base, 0), min(s.End-base, len(r))
		for i := a; i < b; i++ {
			l.ws[i] = 0
		}
		l.ws[a] = s.Cols
		l.spans = append(l.spans, [2]int{a, b})
	}
	return l
}

// w is how many columns rune i takes.
func (l line) w(i int) int {
	if l.ws != nil {
		return l.ws[i]
	}
	return Width(l.r[i])
}

// cols is how many columns runes [a, b) take.
func (l line) cols(a, b int) int {
	n := 0
	for i := a; i < b; i++ {
		n += l.w(i)
	}
	return n
}

// inside reports whether a row may not break before rune i: it is inside
// a span.
func (l line) inside(i int) bool {
	for _, s := range l.spans {
		if i > s[0] && i < s[1] {
			return true
		}
	}
	return false
}

// whole moves a break at end out of a span: before it, or after it when
// it starts the row (pos).
func (l line) whole(end, pos int) int {
	for _, s := range l.spans {
		if end > s[0] && end < s[1] {
			if s[0] > pos {
				return s[0]
			}
			return s[1]
		}
	}
	return end
}

// fit is the end of the runes from pos that fit in room columns (at least
// one rune).
func (l line) fit(pos, room int) int {
	end, used := pos, 0
	for end < len(l.r) && used+l.w(end) <= room {
		used += l.w(end)
		end++
	}
	return max(end, pos+1)
}

// Row is one visual row: the runes [Start, End) of logical line Line,
// shown after Indent columns of padding. Offsets are into the whole text.
type Row struct {
	Line       int
	Start, End int
	Indent     int
	// Last reports whether this is the last row of its line.
	Last bool
}

// Layout is a wrapped text.
type Layout struct {
	Rows  []Row
	Width int
	text  []rune // for the columns of wide characters
	spans []Span // shown in one piece (WrapSpans), sorted
}

// Wrap lays text out in rows of at most width columns. A line breaks after
// the last space that fits (the space stays at the end of the row) or, if
// a word is longer than the row, in the word. Continuation rows keep the
// line's leading indentation so wrapped dialogue stays in its column,
// unless that indentation takes more than half the width.
func Wrap(text []rune, width int) Layout {
	return WrapSpans(text, width, nil)
}

// WrapSpans is Wrap with spans shown in one piece (sorted, apart).
func WrapSpans(text []rune, width int, spans []Span) Layout {
	width = max(width, 1)
	l := Layout{Width: width, text: text, spans: spans}
	line, start, k := 0, 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		for k < len(spans) && spans[k].End <= start {
			k++
		}
		l.Rows = append(l.Rows, wrapLine(newLine(text[start:i], start, spans[k:]), start, line, width)...)
		line++
		start = i + 1
	}
	return l
}

func wrapLine(ln line, base, number, width int) []Row {
	line := ln.r
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	// never break inside the leading indentation (a row of only spaces)
	leading := indent
	if indent > width/2 {
		indent = 0
	}

	var rows []Row
	pos, pad := 0, 0
	for {
		room := width - pad
		if ln.cols(pos, len(line)) <= room {
			return append(rows, Row{Line: number, Start: base + pos, End: base + len(line), Indent: pad, Last: true})
		}
		end := ln.fit(pos, room)
		// break after the last space that fits; keep at least one rune
		for b := end; b > pos; b-- {
			if line[b-1] == ' ' && b-1 > pos && b-1 >= leading && !ln.inside(b-1) && !ln.inside(b) {
				end = b
				break
			}
		}
		// spaces right at the break stay on this row
		for end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.cols(pos, end) < room {
			end++
		}
		end = ln.whole(end, pos)
		rows = append(rows, Row{Line: number, Start: base + pos, End: base + end, Indent: pad})
		pos, pad = end, indent
		if pad >= width {
			pad = 0
		}
	}
}

// Format is how a line is laid out: from column Indent, Width columns
// wide (0: to the edge), aligned 'L' (or 0), 'C' or 'R' in them.
type Format struct {
	Indent, Width int
	Align         byte
}

// WrapFormatted lays text out in rows of at most width columns, each line
// in its format (formats[i] for line i; lines past the end are laid out
// from column 0). Leading spaces are text here, not an indent.
func WrapFormatted(text []rune, width int, formats []Format) Layout {
	return WrapFormattedSpans(text, width, formats, nil)
}

// WrapFormattedSpans is WrapFormatted with spans shown in one piece
// (sorted, apart).
func WrapFormattedSpans(text []rune, width int, formats []Format, spans []Span) Layout {
	width = max(width, 1)
	l := Layout{Width: width, text: text, spans: spans}
	line, start, k := 0, 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		var f Format
		if line < len(formats) {
			f = formats[line]
		}
		for k < len(spans) && spans[k].End <= start {
			k++
		}
		l.Rows = append(l.Rows, formatLine(newLine(text[start:i], start, spans[k:]), start, line, width, f)...)
		line++
		start = i + 1
	}
	return l
}

func formatLine(ln line, base, number, width int, f Format) []Row {
	line := ln.r
	col, room := f.Indent, f.Width
	if col < 0 || col >= width {
		col = 0
	}
	if room <= 0 || col+room > width {
		room = width - col
	}
	var rows []Row
	for pos := 0; ; {
		end := len(line)
		if ln.cols(pos, len(line)) > room {
			end = ln.fit(pos, room)
			for b := end; b > pos+1; b-- {
				if line[b-1] == ' ' && !ln.inside(b-1) && !ln.inside(b) {
					end = b
					break
				}
			}
			for end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.cols(pos, end) < room {
				end++
			}
			end = ln.whole(end, pos)
		}
		last := end
		for last > pos && line[last-1] == ' ' && !ln.inside(last-1) {
			last--
		}
		shown := ln.cols(pos, last)
		pad := col
		switch f.Align {
		case 'R':
			pad += max(0, room-shown)
		case 'C':
			pad += max(0, room-shown) / 2
		}
		done := end == len(line)
		rows = append(rows, Row{Line: number, Start: base + pos, End: base + end, Indent: pad, Last: done})
		if done {
			return rows
		}
		pos = end
	}
}

// RowCol is where an offset is shown: its row, and its column including
// the row's indent. An offset at a wrap point is shown at the start of
// the next row.
func (l Layout) RowCol(offset int) (row, col int) {
	for i, r := range l.Rows {
		if offset >= r.Start && (offset < r.End || (offset == r.End && r.Last)) {
			return i, r.Indent + l.cols(r.Start, offset)
		}
	}
	last := len(l.Rows) - 1
	return last, l.Rows[last].Indent + l.cols(l.Rows[last].Start, l.Rows[last].End)
}

// RowWidth is how many columns row's text takes (without its indent).
func (l Layout) RowWidth(row int) int {
	r := l.Rows[row]
	return l.cols(r.Start, r.End)
}

// cols is how many columns the text [start, end) takes.
func (l Layout) cols(start, end int) int {
	if l.text == nil || end > len(l.text) {
		return end - start
	}
	n := 0
	if len(l.spans) == 0 {
		for _, r := range l.text[start:end] {
			n += Width(r)
		}
		return n
	}
	for off := start; off < end; off++ {
		n += l.width(off)
	}
	return n
}

// width is how many columns the rune at off takes: a span's first all of
// its columns, its others none.
func (l Layout) width(off int) int {
	if len(l.spans) > 0 {
		if s, ok := l.SpanAt(off); ok {
			if off == s.Start {
				return s.Cols
			}
			return 0
		}
	}
	return Width(l.text[off])
}

// SpanAt is the span that holds the rune at off, if one does.
func (l Layout) SpanAt(off int) (Span, bool) {
	i := sort.Search(len(l.spans), func(i int) bool { return l.spans[i].End > off })
	if i < len(l.spans) && l.spans[i].Start <= off {
		return l.spans[i], true
	}
	return Span{}, false
}

// Offset is the text offset for a row and column (clicking or moving the
// cursor up and down); columns in the indent or past the end of the row
// snap to the row's start or end.
func (l Layout) Offset(row, col int) int {
	if row < 0 {
		return 0
	}
	if row >= len(l.Rows) {
		return l.Rows[len(l.Rows)-1].End
	}
	r := l.Rows[row]
	off := r.Start
	for c := r.Indent; off < r.End && off < len(l.text) && c+l.width(off) <= col; off++ {
		c += l.width(off)
	}
	if l.text == nil {
		off = r.Start + max(0, col-r.Indent)
	}
	end := r.End
	if !r.Last {
		// the last position of a wrapped row belongs to the next row
		end--
	}
	return min(off, max(end, r.Start))
}

// RowText is the text shown in a row, without its indent.
func (l Layout) RowText(text []rune, row int) string {
	r := l.Rows[row]
	return string(text[r.Start:r.End])
}

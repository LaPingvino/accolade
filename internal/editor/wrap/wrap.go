// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Package wrap lays the lines of a text out in rows of a fixed number of
// columns, for Accolade's grid-based editor (docs/EDITOR_WIDGET.md, step
// 2). The editor's font is monospace, and a text takes the cells Lekhan
// monospace gives it (lexington/lekhan, through cluster): a character one
// column, a wide one (CJK, emoji) two, a mark none, a Devanagari cluster
// one, a conjunct two, an Arabic lam-alef one. A row never breaks inside a
// cluster.
package wrap

import (
	"sort"

	"github.com/LaPingvino/accolade/internal/editor/cluster"
)

// Span is a stretch of the text, [Start, End) in runes, shown as Cols
// columns in one piece (a chip drawn in its place): its first rune takes
// the columns, the others none, and no row breaks inside it.
type Span struct {
	Start, End, Cols int
}

// cellsOf are the columns each rune of text takes in its cluster
// (cluster.Cells), nil for a text all of whose runes are below U+0300 (one
// column each, every rune its own cluster).
func cellsOf(text []rune) []int8 {
	plain := true
	for _, r := range text {
		if r >= 0x300 {
			plain = false
			break
		}
	}
	if plain {
		return nil
	}
	out := make([]int8, len(text))
	for start := 0; start <= len(text); {
		end := start
		hard := false
		for end < len(text) && text[end] != '\n' {
			hard = hard || text[end] >= 0x300
			end++
		}
		if hard {
			for k, c := range cluster.Cells(text[start:end]) {
				out[start+k] = int8(c)
			}
		} else {
			for k := start; k < end; k++ {
				out[k] = 1
			}
		}
		if end < len(text) {
			out[end] = 1 // the newline
		}
		start = end + 1
	}
	return out
}

// line is one line being laid out: its runes, the columns each takes
// (cells, nil for plain text; ws with the spans counted, nil without
// spans) and the spans, from the line's start.
type line struct {
	r     []rune
	cells []int8
	ws    []int
	spans [][2]int
}

// newLine is the line [base, base+len(r)) of a text with spans (sorted)
// and cells (cellsOf the text): those in it taken in.
func newLine(r []rune, base int, cells []int8, spans []Span) line {
	l := line{r: r}
	if cells != nil {
		l.cells = cells[base : base+len(r)]
	}
	for _, s := range spans {
		if s.Start >= base+len(r) {
			break
		}
		if s.End <= base {
			continue
		}
		if l.ws == nil {
			ws := make([]int, len(r))
			for i := range r {
				ws[i] = l.w(i)
			}
			l.ws = ws
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
	switch {
	case l.ws != nil:
		return l.ws[i]
	case l.cells != nil:
		return int(l.cells[i])
	}
	return 1
}

// cols is how many columns runes [a, b) take.
func (l line) cols(a, b int) int {
	n := 0
	for i := a; i < b; i++ {
		n += l.w(i)
	}
	return n
}

// fits reports whether runes [a, b) take at most room columns, looking
// no further than room (a long paragraph's rest is not counted again at
// each of its rows).
func (l line) fits(a, b, room int) bool {
	if l.ws == nil && l.cells == nil {
		return b-a <= room
	}
	n := 0
	for i := a; i < b; i++ {
		if n += l.w(i); n > room {
			return false
		}
	}
	return true
}

// boundary tells whether a cluster starts at rune i (a row may break
// there, spans aside).
func (l line) boundary(i int) bool {
	return l.cells == nil || cluster.Boundary(l.r, i)
}

// next is where the cluster at rune i ends.
func (l line) next(i int) int {
	if l.cells == nil {
		return i + 1
	}
	return cluster.End(l.r, i)
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

// fit is the end of the clusters from pos that fit in room columns (at
// least one cluster).
func (l line) fit(pos, room int) int {
	end, used := pos, 0
	for end < len(l.r) {
		next := l.next(end)
		w := l.cols(end, next)
		if used+w > room {
			break
		}
		used += w
		end = next
	}
	if end == pos {
		end = l.next(pos)
	}
	return end
}

// breakable tells whether a row may break after the space at rune b-1:
// not in a span, and a cluster starting at b.
func (l line) breakable(b int) bool {
	return l.r[b-1] == ' ' && !l.inside(b-1) && !l.inside(b) && l.boundary(b)
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
	text  []rune // for the columns of wide characters and clusters
	cells []int8 // the columns each rune takes (cellsOf; nil: one each)
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
	l := Layout{Width: width, text: text, cells: cellsOf(text), spans: spans, Rows: make([]Row, 0, rowsGuess(text, width))}
	line, start, k := 0, 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		for k < len(spans) && spans[k].End <= start {
			k++
		}
		l.Rows = wrapLine(l.Rows, newLine(text[start:i], start, l.cells, spans[k:]), start, line, width)
		line++
		start = i + 1
	}
	return l
}

// rowsGuess is about how many rows text takes at width: room for them
// at once, not grown row by row.
func rowsGuess(text []rune, width int) int {
	return len(text)/width + lineBreaks(text) + 1
}

// lineBreaks is how many line breaks text has.
func lineBreaks(text []rune) int {
	n := 0
	for _, r := range text {
		if r == '\n' {
			n++
		}
	}
	return n
}

// wrapLine appends the line's rows to rows.
func wrapLine(rows []Row, ln line, base, number, width int) []Row {
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

	pos, pad := 0, 0
	for {
		room := width - pad
		if ln.fits(pos, len(line), room) {
			return append(rows, Row{Line: number, Start: base + pos, End: base + len(line), Indent: pad, Last: true})
		}
		end := ln.fit(pos, room)
		// break after the last space that fits; keep at least one rune
		for b := end; b > pos; b-- {
			if b-1 > pos && b-1 >= leading && ln.breakable(b) {
				end = b
				break
			}
		}
		// spaces right at the break stay on this row
		for used := ln.cols(pos, end); end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.boundary(end+1) && used < room; end++ {
			used += ln.w(end)
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
	l := Layout{Width: width, text: text, cells: cellsOf(text), spans: spans, Rows: make([]Row, 0, rowsGuess(text, width))}
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
		l.Rows = formatLine(l.Rows, newLine(text[start:i], start, l.cells, spans[k:]), start, line, width, f)
		line++
		start = i + 1
	}
	return l
}

// formatLine appends the line's rows to rows.
func formatLine(rows []Row, ln line, base, number, width int, f Format) []Row {
	line := ln.r
	col, room := f.Indent, f.Width
	if col < 0 || col >= width {
		col = 0
	}
	if room <= 0 || col+room > width {
		room = width - col
	}
	for pos := 0; ; {
		end := len(line)
		if !ln.fits(pos, len(line), room) {
			end = ln.fit(pos, room)
			for b := end; b > pos+1; b-- {
				if ln.breakable(b) {
					end = b
					break
				}
			}
			for used := ln.cols(pos, end); end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.boundary(end+1) && used < room; end++ {
				used += ln.w(end)
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
	return l.RuneCols(off)
}

// RuneCols is how many columns the rune at off takes in its cluster
// (cluster.Cells: a Devanagari cluster's first rune and a conjunct's
// second consonant one each, its other runes none; a lam-alef's lam one,
// its alef none), spans aside.
func (l Layout) RuneCols(off int) int {
	if l.cells == nil {
		return 1 // a text all below U+0300: a column each
	}
	return int(l.cells[off])
}

// clusterEnd is where the cluster (or span) at off ends, in row r.
func (l Layout) clusterEnd(off int, r Row) int {
	if s, ok := l.SpanAt(off); ok && s.Start == off {
		return min(s.End, r.End)
	}
	if l.cells == nil {
		return off + 1
	}
	return min(cluster.End(l.text, off), r.End)
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
	if l.text == nil {
		off = r.Start + max(0, col-r.Indent)
	} else {
		// by clusters: a column in one (a conjunct's or a wide
		// character's second) gives its start
		for c := r.Indent; off < r.End && off < len(l.text); {
			next := l.clusterEnd(off, r)
			w := l.cols(off, next)
			if c+w > col {
				break
			}
			c += w
			off = next
		}
	}
	end := r.End
	if !r.Last && end > r.Start {
		// the last position of a wrapped row belongs to the next row: the
		// start of the row's last cluster
		end--
		if l.text != nil && l.cells != nil {
			for end > r.Start && !cluster.Boundary(l.text, end) {
				end--
			}
		}
	}
	return min(off, max(end, r.Start))
}

// RowText is the text shown in a row, without its indent.
func (l Layout) RowText(text []rune, row int) string {
	r := l.Rows[row]
	return string(text[r.Start:r.End])
}

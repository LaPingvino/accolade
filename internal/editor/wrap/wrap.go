// Package wrap lays the lines of a text out in rows of a fixed number of
// columns, for Accolade's grid-based editor (docs/EDITOR_WIDGET.md, step
// 2). A rune takes one column (the editor uses a monospace font), a wide
// one (CJK, emoji) two: see Width.
package wrap

import "unicode"

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

// cols is how many columns runes take.
func cols(runes []rune) int {
	n := 0
	for _, r := range runes {
		n += Width(r)
	}
	return n
}

// fit is the end of the runes from pos that fit in room columns (at least
// one rune).
func fit(line []rune, pos, room int) int {
	end, used := pos, 0
	for end < len(line) && used+Width(line[end]) <= room {
		used += Width(line[end])
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
}

// Wrap lays text out in rows of at most width columns. A line breaks after
// the last space that fits (the space stays at the end of the row) or, if
// a word is longer than the row, in the word. Continuation rows keep the
// line's leading indentation so wrapped dialogue stays in its column,
// unless that indentation takes more than half the width.
func Wrap(text []rune, width int) Layout {
	width = max(width, 1)
	l := Layout{Width: width, text: text}
	line, start := 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		l.Rows = append(l.Rows, wrapLine(text[start:i], start, line, width)...)
		line++
		start = i + 1
	}
	return l
}

func wrapLine(line []rune, base, number, width int) []Row {
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
		if cols(line[pos:]) <= room {
			return append(rows, Row{Line: number, Start: base + pos, End: base + len(line), Indent: pad, Last: true})
		}
		end := fit(line, pos, room)
		// break after the last space that fits; keep at least one rune
		for b := end; b > pos; b-- {
			if line[b-1] == ' ' && b-1 > pos && b-1 >= leading {
				end = b
				break
			}
		}
		// spaces right at the break stay on this row
		for end < len(line) && line[end] == ' ' && cols(line[pos:end]) < room {
			end++
		}
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
	width = max(width, 1)
	l := Layout{Width: width, text: text}
	line, start := 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		var f Format
		if line < len(formats) {
			f = formats[line]
		}
		l.Rows = append(l.Rows, formatLine(text[start:i], start, line, width, f)...)
		line++
		start = i + 1
	}
	return l
}

func formatLine(line []rune, base, number, width int, f Format) []Row {
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
		if cols(line[pos:]) > room {
			end = fit(line, pos, room)
			for b := end; b > pos+1; b-- {
				if line[b-1] == ' ' {
					end = b
					break
				}
			}
			for end < len(line) && line[end] == ' ' && cols(line[pos:end]) < room {
				end++
			}
		}
		last := end
		for last > pos && line[last-1] == ' ' {
			last--
		}
		shown := cols(line[pos:last])
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

// cols is how many columns the text [start, end) takes.
func (l Layout) cols(start, end int) int {
	if l.text == nil || end > len(l.text) {
		return end - start
	}
	return cols(l.text[start:end])
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
	for c := r.Indent; off < r.End && off < len(l.text) && c+Width(l.text[off]) <= col; off++ {
		c += Width(l.text[off])
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

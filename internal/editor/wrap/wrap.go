// Package wrap lays the lines of a text out in rows of a fixed number of
// columns, for Accolade's grid-based editor (docs/EDITOR_WIDGET.md, step
// 2). Every rune takes one column (the editor uses a monospace font;
// double-width characters are not accounted for).
package wrap

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
}

// Wrap lays text out in rows of at most width columns. A line breaks after
// the last space that fits (the space stays at the end of the row) or, if
// a word is longer than the row, in the word. Continuation rows keep the
// line's leading indentation so wrapped dialogue stays in its column,
// unless that indentation takes more than half the width.
func Wrap(text []rune, width int) Layout {
	width = max(width, 1)
	l := Layout{Width: width}
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
		if len(line)-pos <= room {
			return append(rows, Row{Line: number, Start: base + pos, End: base + len(line), Indent: pad, Last: true})
		}
		end := pos + room
		// break after the last space that fits; keep at least one rune
		for b := end; b > pos; b-- {
			if line[b-1] == ' ' && b-1 > pos && b-1 >= leading {
				end = b
				break
			}
		}
		// spaces right at the break stay on this row
		for end < len(line) && line[end] == ' ' && end-pos < room {
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
	l := Layout{Width: width}
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
		if end-pos > room {
			end = pos + room
			for b := end; b > pos+1; b-- {
				if line[b-1] == ' ' {
					end = b
					break
				}
			}
			for end < len(line) && line[end] == ' ' && end-pos < room {
				end++
			}
		}
		shown := end - pos
		for shown > 0 && line[pos+shown-1] == ' ' {
			shown--
		}
		pad := col
		switch f.Align {
		case 'R':
			pad += max(0, room-shown)
		case 'C':
			pad += max(0, room-shown) / 2
		}
		last := end == len(line)
		rows = append(rows, Row{Line: number, Start: base + pos, End: base + end, Indent: pad, Last: last})
		if last {
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
			return i, r.Indent + offset - r.Start
		}
	}
	last := len(l.Rows) - 1
	return last, l.Rows[last].Indent + l.Rows[last].End - l.Rows[last].Start
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
	off := r.Start + max(0, col-r.Indent)
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

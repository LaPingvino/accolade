// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package wrap

// The wrap as it was before a row stopped counting the rest of its line
// (fits): kept as the oracle that the rows are the same.

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func oldWrapSpans(text []rune, width int, spans []Span) Layout {
	width = max(width, 1)
	l := Layout{Width: width, text: text, cells: cellsOf(text), spans: spans}
	line, start, k := 0, 0, 0
	for i := 0; i <= len(text); i++ {
		if i < len(text) && text[i] != '\n' {
			continue
		}
		for k < len(spans) && spans[k].End <= start {
			k++
		}
		l.Rows = append(l.Rows, oldWrapLine(newLine(text[start:i], start, l.cells, spans[k:]), start, line, width)...)
		line++
		start = i + 1
	}
	return l
}

func oldWrapLine(ln line, base, number, width int) []Row {
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
			if b-1 > pos && b-1 >= leading && ln.breakable(b) {
				end = b
				break
			}
		}
		// spaces right at the break stay on this row
		for end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.boundary(end+1) && ln.cols(pos, end) < room {
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

func oldWrapFormattedSpans(text []rune, width int, formats []Format, spans []Span) Layout {
	width = max(width, 1)
	l := Layout{Width: width, text: text, cells: cellsOf(text), spans: spans}
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
		l.Rows = append(l.Rows, oldFormatLine(newLine(text[start:i], start, l.cells, spans[k:]), start, line, width, f)...)
		line++
		start = i + 1
	}
	return l
}

func oldFormatLine(ln line, base, number, width int, f Format) []Row {
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
				if ln.breakable(b) {
					end = b
					break
				}
			}
			for end < len(line) && line[end] == ' ' && !ln.inside(end) && ln.boundary(end+1) && ln.cols(pos, end) < room {
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

func TestWrapAsBefore(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	words := []string{"a", "word", "longerwordthanmost", "  ", " ", "वह", "मेरा", "दोस्त", "क्षत्रिय", "漢字", "日本語", "שלום", "é", "x́", "🙂", "supercalifragilisticexpialidocious"}
	for n := 0; n < 3000; n++ {
		ws := words
		if n%2 == 0 { // plain text: a column a rune
			ws = []string{"a", "word", "longerwordthanmost", "  ", " ", "supercalifragilisticexpialidocious"}
		}
		var b strings.Builder
		lines := 1 + rng.Intn(6)
		for l := 0; l < lines; l++ {
			if l > 0 {
				b.WriteByte('\n')
			}
			if rng.Intn(3) == 0 {
				b.WriteString(strings.Repeat(" ", rng.Intn(30)))
			}
			for k := rng.Intn(80); k > 0; k-- {
				b.WriteString(ws[rng.Intn(len(ws))])
				if rng.Intn(4) > 0 {
					b.WriteByte(' ')
				}
			}
		}
		text := []rune(b.String())
		var spans []Span
		for at := rng.Intn(20); n%4 != 0 && at < len(text)-3 && rng.Intn(3) > 0; at += 5 + rng.Intn(40) {
			if text[at] == '\n' {
				continue
			}
			end := at + 1 + rng.Intn(3)
			if strings.ContainsRune(string(text[at:end]), '\n') {
				continue
			}
			spans = append(spans, Span{Start: at, End: end, Cols: 1 + rng.Intn(8)})
		}
		width := 1 + rng.Intn(70)
		if got, want := WrapSpans(text, width, spans), oldWrapSpans(text, width, spans); !reflect.DeepEqual(got.Rows, want.Rows) {
			t.Fatalf("WrapSpans(%q, %d, %v):\n got %v\nwant %v", string(text), width, spans, got.Rows, want.Rows)
		}
		formats := make([]Format, lines)
		for i := range formats {
			formats[i] = Format{Indent: rng.Intn(20) - 2, Width: rng.Intn(50), Align: "LCR\x00"[rng.Intn(4)]}
		}
		if got, want := WrapFormattedSpans(text, width, formats, spans), oldWrapFormattedSpans(text, width, formats, spans); !reflect.DeepEqual(got.Rows, want.Rows) {
			t.Fatalf("WrapFormattedSpans(%q, %d, %v, %v):\n got %v\nwant %v", string(text), width, formats, spans, got.Rows, want.Rows)
		}
	}
}

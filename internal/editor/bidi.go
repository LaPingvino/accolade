package editor

import (
	"slices"

	"fyne.io/fyne/v2/widget"
	gobidi "github.com/go-text/typesetting/bidi"
	"golang.org/x/text/unicode/bidi"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// A line's direction can be set with a mark at its start, which other
// apps read the same way: right to left (RLM) or left to right (LRM).
const (
	LRM = lex.LRM
	RLM = lex.RLM
)

// RTL reports whether a line reads right to left (Arabic, Hebrew,
// Persian…): by its direction mark, or else its first strong letter
// (lex.OwnDirection); left to right for a line with neither.
func RTL(line string) bool { return lex.LineRTL(line, false) }

// LineRTL is RTL in the editor's script: a line with no direction of its
// own ("…", "2.") takes the script language's (LanguageRTL), the last
// resort only.
func (e *ScriptEditor) LineRTL(line string) bool { return lex.LineRTL(line, e.LanguageRTL) }

// mirrored is a format laid out from the other side of a page cols wide,
// for a right-to-left line: its column mirrored, its alignment turned.
func mirrored(f wrap.Format, cols int) wrap.Format {
	if f.Width <= 0 || f.Indent+f.Width > cols {
		f.Width = max(cols-f.Indent, 1)
	}
	f.Indent = max(cols-f.Indent-f.Width, 0)
	switch f.Align {
	case 'R':
		f.Align = 'L'
	case 'C':
	default:
		f.Align = 'R'
	}
	return f
}

// rowBidi is how a row's text is drawn, as the Unicode bidi algorithm
// orders it: its text cells [From, From+len(Levels)) each at its
// embedding level, drawn in Order (left to right, indexes into Levels).
// The zero value draws a row as written, left to right.
type rowBidi struct {
	From   int
	Levels []int8
	Order  []int
	// RTL: the row reads right to left; the cells after its text (the
	// cursor at the line's end) go left of it
	RTL bool
}

// reordered reports whether the row's text is drawn other than as written.
func (b rowBidi) reordered() bool { return len(b.Order) > 0 }

// level is cell i's embedding level, -1 outside the text.
func (b rowBidi) level(i int) int {
	if k := i - b.From; k >= 0 && k < len(b.Levels) {
		return int(b.Levels[k])
	}
	return -1
}

// newRowBidi orders the text cells [from, end) of a row; rtl is its
// line's direction. A row with no right-to-left letter or Arabic digit
// in a line read left to right is drawn as written, unlooked at.
func newRowBidi(cells []widget.TextGridCell, from, end int, rtl bool) rowBidi {
	end = min(end, len(cells))
	if from >= end {
		return rowBidi{From: from, RTL: rtl}
	}
	rs := make([]rune, end-from)
	need := rtl
	for k := range rs {
		r := cells[from+k].Rune
		if r == wideFill && k > 0 { // a wide character's second cell goes with it
			r = rs[k-1]
		}
		if r >= 0x0590 && !need {
			switch bidiClass(r) {
			case bidi.R, bidi.AL, bidi.AN, bidi.RLE, bidi.RLO, bidi.RLI, bidi.FSI:
				need = true
			}
		}
		rs[k] = r
	}
	if !need {
		return rowBidi{}
	}
	lv := bidiLevels(rs, rtl)
	if !rtl && !slices.ContainsFunc(lv, func(l int8) bool { return l != 0 }) {
		return rowBidi{}
	}
	return rowBidi{From: from, Levels: lv, Order: visualOrder(lv), RTL: rtl}
}

func bidiClass(r rune) bidi.Class { p, _ := bidi.LookupRune(r); return p.Class() }

// bidiLevels are the embedding levels of a paragraph's characters in the
// direction given (rules P–I and L1 of the bidi algorithm). go-text gives
// a run's level for all of it; in a left-to-right paragraph an even run
// can hold numbers at level 2 (after a right-to-left letter) beside
// letters at level 0, told apart here.
func bidiLevels(rs []rune, rtl bool) []int8 {
	dir := gobidi.LeftToRight
	if rtl {
		dir = gobidi.RightToLeft
	}
	var p gobidi.Paragraph
	runs := p.Segment(rs, dir)
	lv := make([]int8, len(rs))
	for i := 0; i < runs.NumRuns(); i++ {
		r := runs.Run(i)
		for k := r.Start; k < r.End; k++ {
			lv[k] = int8(r.Level)
		}
	}
	if rtl {
		return lv
	}
	// a number in a left-to-right paragraph: level 2 when the last strong
	// letter before it reads right to left (W2, W7, I1), else 0
	lastRTL := false
	for k := 0; k < len(rs); {
		c := bidiClass(rs[k])
		switch c {
		case bidi.L:
			lastRTL = false
		case bidi.R, bidi.AL:
			lastRTL = true
		}
		if lv[k]%2 != 0 || (c != bidi.EN && c != bidi.AN) {
			if lv[k]%2 == 0 {
				lv[k] = 0 // with the letters
			}
			k++
			continue
		}
		// the number: digits, one separator between two, terminators, marks
		j := k + 1
		for j < len(rs) && lv[j] == lv[k] {
			cj := bidiClass(rs[j])
			if cj == bidi.EN || cj == bidi.AN || cj == bidi.ET || cj == bidi.NSM {
				j++
				continue
			}
			if (cj == bidi.CS || cj == bidi.ES) && j+1 < len(rs) && lv[j+1] == lv[k] {
				if cn := bidiClass(rs[j+1]); cn == bidi.EN || cn == bidi.AN {
					j++
					continue
				}
			}
			break
		}
		l := int8(0)
		if lastRTL || c == bidi.AN {
			l = 2
		}
		for m := k; m < j; m++ {
			lv[m] = l
		}
		k = j
	}
	return lv
}

// visualOrder is the order characters at these levels are drawn in, left
// to right (rule L2): from the highest level down to the lowest odd one,
// every run at that level or higher reversed.
func visualOrder(lv []int8) []int {
	order := make([]int, len(lv))
	high, lowOdd := int8(0), int8(127)
	for i, l := range lv {
		order[i] = i
		high = max(high, l)
		if l%2 == 1 {
			lowOdd = min(lowOdd, l)
		}
	}
	for l := high; l >= lowOdd && l > 0; l-- {
		for i := 0; i < len(order); {
			if lv[order[i]] < l {
				i++
				continue
			}
			j := i
			for j < len(order) && lv[order[j]] >= l {
				j++
			}
			slices.Reverse(order[i:j])
			i = j
		}
	}
	return order
}

// bidiJoins tells for each character of a row's text at these levels
// whether it is drawn in one piece with the one before (the text drawer
// orders a piece itself): one level, and above a left-to-right
// paragraph's own level only letters of the level's direction (a word,
// "השעה"), their marks, and in left-to-right text in a right-to-left one
// a neutral between two of them ("Pieter Bruegel", "23:40"). Every other
// neutral stands in its own cell.
func bidiJoins(rs []rune, lv []int8) []bool {
	joins := make([]bool, len(rs))
	strong := func(k int) bool { return leads(rs[k], lv[k]) }
	mark := func(k int) bool { return bidiClass(rs[k]) == bidi.NSM || rs[k] == wideFill }
	for k := 1; k < len(rs); k++ {
		switch {
		case lv[k] != lv[k-1]:
		case lv[k] == 0:
			joins[k] = true
		case !joins[k-1] && !strong(k-1):
			// the cell before starts no piece
		case strong(k) || mark(k):
			joins[k] = true
		case lv[k]%2 == 1:
			// a space or sign in right-to-left text stands in its own cell:
			// the words keep to their cells (a font's own spaces need not)
		default: // a neutral: joined if a letter of the piece's way follows
			m := k + 1
			for m < len(rs) && lv[m] == lv[k] && !strong(m) {
				m++
			}
			joins[k] = m < len(rs) && lv[m] == lv[k]
		}
	}
	return joins
}

// leads tells whether a character at a level can start a piece drawn
// whole: a letter (or digit) of the level's direction. A neutral starts
// none: drawn in a piece by itself the text drawer would put it left to
// right, where the bidi algorithm has it.
func leads(r rune, level int8) bool {
	c := bidiClass(r)
	if level%2 == 1 {
		return c == bidi.R || c == bidi.AL
	}
	return c == bidi.L || c == bidi.EN || c == bidi.AN
}

// drawnCols are the columns each cell of a row is drawn at (marks take
// none), and the column after the last: cells in a reordered text where
// the order puts them, after a right-to-left row's text left of it.
func drawnCols(cells []widget.TextGridCell, b rowBidi) []int {
	x := make([]int, len(cells)+1)
	for i, c := range cells {
		x[i+1] = x[i] + cellWidth(c.Rune)
	}
	from, end := b.From, min(b.From+len(b.Levels), len(cells))
	if from >= end && !b.reordered() {
		return x
	}
	at := slices.Clone(x)
	if b.reordered() {
		acc := x[from]
		for _, k := range b.Order {
			if i := from + k; i < end {
				at[i] = acc
				acc += x[i+1] - x[i]
			}
		}
	}
	if b.RTL && from < end {
		for i := end; i < len(cells); i++ {
			at[i] = x[from] - (x[i+1] - x[end])
		}
	}
	at[len(cells)] = x[len(cells)]
	return at
}

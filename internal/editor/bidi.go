package editor

import (
	"golang.org/x/text/unicode/bidi"

	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// A line's direction can be set with a mark at its start, which other
// apps read the same way: right to left (RLM) or left to right (LRM).
const (
	LRM = '‎'
	RLM = '‏'
)

// RTL reports whether a line reads right to left (Arabic, Hebrew,
// Persian…): by its direction mark, or else its first strong letter.
func RTL(line string) bool {
	for _, r := range line {
		switch r {
		case RLM:
			return true
		case LRM:
			return false
		}
		p, _ := bidi.LookupRune(r)
		switch p.Class() {
		case bidi.R, bidi.AL:
			return true
		case bidi.L:
			return false
		}
	}
	return false
}

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

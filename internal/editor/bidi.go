package editor

import (
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

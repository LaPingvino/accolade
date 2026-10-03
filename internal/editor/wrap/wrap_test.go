package wrap

import (
	"reflect"
	"strings"
	"testing"
)

// rows renders a layout as the rows would look on screen.
func rows(text string, width int) []string {
	t := []rune(text)
	l := Wrap(t, width)
	var out []string
	for i, r := range l.Rows {
		out = append(out, strings.Repeat(" ", r.Indent)+l.RowText(t, i))
	}
	return out
}

func TestWrap(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  []string
	}{
		{"", 10, []string{""}},
		{"short", 10, []string{"short"}},
		{"Rain hammers the roof.", 10, []string{"Rain ", "hammers ", "the roof."}},
		{"a\n\nb", 10, []string{"a", "", "b"}},
		{"supercalifragilistic", 8, []string{"supercal", "ifragili", "stic"}},
		// dialogue keeps its indentation on continuation rows
		{"    I can't believe it is raining.", 16, []string{"    I can't ", "    believe it ", "    is raining."}},
		// too much indentation for the width: continuation starts at 0
		{"          AAAA BBBB", 12, []string{"          AA", "AA BBBB"}},
		{"exactly ten", 11, []string{"exactly ten"}},
	}
	for _, c := range cases {
		if got := rows(c.text, c.width); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Wrap(%q, %d) = %q, want %q", c.text, c.width, got, c.want)
		}
	}
}

func TestRowsCoverTextAndFitWidth(t *testing.T) {
	text := []rune("FADE IN:\n\nINT. BARN - DAY\n\nRain hammers the roof of the old barn while the wind howls.\n" +
		strings.Repeat(" ", 25) + "It's coming, and there is nothing we can do about it now.")
	for width := 1; width <= 70; width++ {
		l := Wrap(text, width)
		next := 0
		for i, r := range l.Rows {
			if r.Start != next && !(i > 0 && l.Rows[i-1].Last && r.Start == next+1) {
				t.Fatalf("width %d: row %d starts at %d, expected %d", width, i, r.Start, next)
			}
			if r.Indent+r.End-r.Start > width {
				t.Fatalf("width %d: row %d is %d columns", width, i, r.Indent+r.End-r.Start)
			}
			next = r.End
		}
		if next != len(text) {
			t.Fatalf("width %d: rows end at %d of %d", width, next, len(text))
		}
	}
}

func TestOffsetRoundTrip(t *testing.T) {
	text := []rune("Rain hammers the roof.\n    I can't believe it is raining.\n")
	for _, width := range []int{8, 10, 16, 40} {
		l := Wrap(text, width)
		for off := 0; off <= len(text); off++ {
			row, col := l.RowCol(off)
			if got := l.Offset(row, col); got != off {
				t.Errorf("width %d: offset %d -> (%d,%d) -> %d", width, off, row, col, got)
			}
		}
	}
}

func TestWrapPointAndSnapping(t *testing.T) {
	text := []rune("Rain hammers the roof.")
	l := Wrap(text, 10) // "Rain " | "hammers " | "the roof."
	if row, col := l.RowCol(5); row != 1 || col != 0 {
		t.Errorf("offset at the wrap point shown at (%d,%d), want (1,0)", row, col)
	}
	if got := l.Offset(0, 99); got != 4 {
		t.Errorf("past the end of a wrapped row: %d, want 4", got)
	}
	if got := l.Offset(2, 99); got != len(text) {
		t.Errorf("past the end of the last row: %d", got)
	}
	ind := Wrap([]rune("    I can't believe it"), 12)
	if got := ind.Offset(1, 1); got != ind.Rows[1].Start {
		t.Errorf("click in the indent snaps to the row start: %d", got)
	}
}

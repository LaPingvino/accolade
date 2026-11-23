package structure

import "strings"

// Refusal is why a move is not made, for the writer (the status line).
type Refusal string

func (r Refusal) Error() string { return string(r) }

// Result is a move made: the new lines, where each old line went, and
// where what moved is now.
type Result struct {
	Lines []string
	// Map is each old line's new line (-1: gone, or its text changed)
	Map []int
	// First and Last are the lines of what moved, in the new text
	First, Last int
	Msg         string
	// moved are the units that moved, which may change parent (the
	// check's); pos is where each old line went, its text changed or not
	moved map[*Node]bool
	pos   []int
	// kept: the text's lines are the same ones, only in another order
	kept bool
}

// piece is a part of the new text: old lines [from, to], or new ones.
type piece struct {
	from, to int
	lit      []string
}

func span(from, to int) piece { return piece{from: from, to: to} }
func lit(l ...string) piece    { return piece{from: 0, to: -1, lit: l} }

// assemble puts pieces together: where two pieces meet with something on
// both sides and sep says units meet there, a blank line between them.
// It gives the lines, where each old line went, and where each piece
// starts.
func (t *Tree) assemble(pieces []piece, sep bool) (lines []string, at []int, starts []int) {
	old := t.S.Lines
	at = make([]int, len(old))
	for i := range at {
		at[i] = -1
	}
	blank := func(s string) bool { return strings.TrimSpace(s) == "" }
	for _, p := range pieces {
		var add []string
		var from []int
		if p.lit != nil {
			add = p.lit
			for range p.lit {
				from = append(from, -1)
			}
		} else {
			for i := p.from; i <= p.to; i++ {
				add = append(add, old[i])
				from = append(from, i)
			}
		}
		if len(add) == 0 {
			starts = append(starts, len(lines))
			continue
		}
		if sep && len(lines) > 0 && !blank(lines[len(lines)-1]) && !blank(add[0]) {
			lines = append(lines, "")
		}
		starts = append(starts, len(lines))
		for k, l := range add {
			if from[k] >= 0 {
				at[from[k]] = len(lines)
			}
			lines = append(lines, l)
		}
	}
	return lines, at, starts
}

// swap puts lines [a0, a1] where [b0, b1] are and the other way round
// (a before b).
func (t *Tree) swap(a0, a1, b0, b1 int, sep bool) ([]string, []int, [2]int, [2]int) {
	n := len(t.S.Lines)
	lines, at, st := t.assemble([]piece{span(0, a0-1), span(b0, b1), span(a1+1, b0-1), span(a0, a1), span(b1+1, n-1)}, sep)
	return lines, at, [2]int{st[3], st[3] + a1 - a0}, [2]int{st[1], st[1] + b1 - b0}
}

// move takes lines [a0, a1] (with the blank lines after them, or before
// at the body's end) to before line to (after: after line to), and gives
// where they are now.
func (t *Tree) move(a0, a1, to int, after bool, sep bool) ([]string, []int, [2]int) {
	n := len(t.S.Lines)
	r0, r1 := a0, a1 // what goes away: the unit and a gap beside it
	if sep {
		for r1+1 < n && r1+1 < t.S.End && t.sep(r1+1) && t.bone[r1+1] == false {
			r1++
		}
		if r1+1 >= t.S.End || r1+1 >= n { // at the body's end: the gap before
			r1 = a1
			for r0-1 >= t.Front && t.sep(r0-1) {
				r0--
			}
		}
	}
	ins := to
	if after {
		ins = to + 1
	}
	var pieces []piece
	var unit int
	if ins <= r0 {
		pieces = []piece{span(0, ins-1), span(a0, a1), span(ins, r0-1), span(r1+1, n-1)}
		unit = 1
	} else {
		pieces = []piece{span(0, r0-1), span(r1+1, ins-1), span(a0, a1), span(ins, n-1)}
		unit = 2
	}
	lines, at, st := t.assemble(pieces, sep)
	return lines, at, [2]int{st[unit], st[unit] + a1 - a0}
}

// replaced is the old lines with some lines' text changed (by line).
func (t *Tree) replaced(change map[int]string) []string {
	out := append([]string(nil), t.S.Lines...)
	for i, s := range change {
		out[i] = s
	}
	return out
}

// result makes a Result of new lines: lines whose text changed map to -1.
func (t *Tree) result(lines []string, at []int, where [2]int, msg string) Result {
	m := append([]int(nil), at...)
	for i, j := range m {
		if j >= 0 && (j >= len(lines) || lines[j] != t.S.Lines[i]) {
			m[i] = -1
		}
	}
	return Result{Lines: lines, Map: m, First: where[0], Last: where[1], Msg: msg, pos: at, moved: map[*Node]bool{}}
}

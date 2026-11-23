package structure

import (
	"fmt"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Range is units side by side named by their headings' lines: a scene,
// a run of scenes (the first's heading to the last's), a section with
// all it holds, or sections side by side.
type Range struct{ First, Last int }

// runOf is the units r names.
func (t *Tree) runOf(r Range) (run, error) {
	var a, b *Node
	for _, c := range t.containers() {
		if c.First == r.First {
			a = c
		}
		if c.First == r.Last {
			b = c
		}
	}
	if a == nil || b == nil {
		return run{}, Refusal("No scene or section there")
	}
	if a.Parent != b.Parent || a.Kind != b.Kind || index(a) > index(b) {
		return run{}, Refusal("A range is scenes or sections side by side, in one section, the first before the last")
	}
	return run{a.Parent, index(a), index(b)}, nil
}

// Units are the headings of r's units (for a preview).
func (t *Tree) Units(r Range) ([]int, error) {
	x, err := t.runOf(r)
	if err != nil {
		return nil, err
	}
	var out []int
	for _, n := range x.nodes() {
		out = append(out, n.First)
	}
	return out, nil
}

// Swap puts the units of x where y's are and y's where x's were.
func (t *Tree) Swap(x, y Range, read Reader) (Result, error) {
	rx, err := t.runOf(x)
	if err != nil {
		return Result{}, err
	}
	ry, err := t.runOf(y)
	if err != nil {
		return Result{}, err
	}
	x0, x1 := rx.lines()
	y0, y1 := ry.lines()
	if y0 < x0 {
		rx, ry = ry, rx
		x0, x1, y0, y1 = y0, y1, x0, x1
	}
	if y0 <= x1 {
		return Result{}, Refusal("The two overlap: one is in the other")
	}
	lines, at, _, where := t.swap(x0, x1, y0, y1, true)
	res := t.result(lines, at, where, fmt.Sprintf("Swapped %s and %s", t.runName(rx), t.runName(ry)))
	res.moved, res.kept = t.movedKeys(append(rx.nodes(), ry.nodes()...)...), true
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

// Move puts the units of x before the scene or section whose heading is
// on line to (after: after it, with all it holds).
func (t *Tree) Move(x Range, to int, after bool, read Reader) (Result, error) {
	rx, err := t.runOf(x)
	if err != nil {
		return Result{}, err
	}
	var target *Node
	for _, c := range t.containers() {
		if c.First == to {
			target = c
		}
	}
	if target == nil {
		return Result{}, Refusal("No scene or section there to move it by")
	}
	a0, a1 := rx.lines()
	if target.First >= a0 && target.First <= a1 {
		return Result{}, Refusal("Not into itself")
	}
	line, where := target.First, "before"
	if after {
		line, where = target.Last, "after"
	}
	res, err := t.moveTo(rx, line, after, fmt.Sprintf("Moved %s %s %s", t.runName(rx), where, t.Name(target)))
	if err != nil {
		return Result{}, err
	}
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

// runName names units side by side: the one, or the first to the last.
func (t *Tree) runName(r run) string {
	if r.i == r.j {
		return t.Name(r.first())
	}
	return t.Name(r.first()) + " to " + t.Name(r.last())
}

// Expand is the unit around lines a to b that holds more than them: a
// line, its speech or paragraph, its scene, its section, the sections
// around that, the whole body. ok is false when there is none bigger.
func (t *Tree) Expand(a, b int) (first, last int, ok bool) {
	if a > b {
		a, b = b, a
	}
	p := t.path(a)
	for i := len(p) - 1; i >= 0; i-- {
		n := p[i]
		if n.First <= a && n.Last >= b && (n.First < a || n.Last > b) {
			return n.First, n.Last, true
		}
	}
	return a, b, false
}

// DefaultHeads are a text's headings as its kinds have them (outside
// boneyards): the reader's when the window's outline is not at hand.
func DefaultHeads(lines []string, kinds []syntax.Kind) []Head {
	var out []Head
	inBone := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if inBone || strings.HasPrefix(t, "/*") {
			inBone = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		switch kinds[i] {
		case syntax.Section:
			out = append(out, Head{Line: i, Level: len(t) - len(strings.TrimLeft(t, "#"))})
		case syntax.SceneHeading:
			out = append(out, Head{Line: i, Scene: true})
		}
	}
	return out
}

// Plain reads lines with English scene headings and no blocks at the
// end (tests, and a text without a window).
func Plain(lines []string) Script {
	kinds := syntax.Classify(lines)
	return Script{Lines: lines, Kinds: kinds, Heads: DefaultHeads(lines, kinds), End: len(lines)}
}

// Lines are text's lines; Text is lines as text again.
func Lines(text string) []string { return strings.Split(text, "\n") }
func Text(lines []string) string { return strings.Join(lines, "\n") }

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package structure

import (
	"fmt"
	"slices"
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

// Kin are the headings of the units r's last can run on to (its kind,
// side by side with it, after it: a range's possible ends), with r's own.
func (t *Tree) Kin(first int) []int {
	x, err := t.runOf(Range{first, first})
	if err != nil {
		return nil
	}
	_, hi := x.sibs()
	var out []int
	for _, n := range x.parent.Kids[x.i : hi+1] {
		out = append(out, n.First)
	}
	return out
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
	res := t.result(lines, at, where, fmt.Sprintf("%s and %s change places", t.runName(rx), t.runName(ry)))
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
	res, err := t.moveTo(rx, line, after, fmt.Sprintf("%s goes %s %s", t.runName(rx), where, t.Name(target)))
	if err != nil {
		return Result{}, err
	}
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

// Unit is a section or scene as a picker lists it.
type Unit struct {
	Line  int
	Scene bool
	Depth int // how many sections it is in
	Name  string
}

// Outline is the sections and scenes in the text's order.
func (t *Tree) Outline() []Unit {
	var out []Unit
	for _, c := range t.containers() {
		d := 0
		for p := c.Parent; p.Kind != Body; p = p.Parent {
			d++
		}
		out = append(out, Unit{Line: c.First, Scene: c.Kind == Scene, Depth: d, Name: t.Name(c)})
	}
	return out
}

// At is the innermost section or scene holding line i (-1: none).
func (t *Tree) At(i int) int {
	at := -1
	for _, n := range t.path(i) {
		if n.container() {
			at = n.First
		}
	}
	return at
}

// InSpeech tells whether line i is in a block (ok), and whether that is
// a speech's line under its cue (speech).
func (t *Tree) InSpeech(i int) (speech, ok bool) {
	p := t.path(i)
	n := p[len(p)-1]
	if n.Kind == Line {
		n = n.Parent
	}
	if n.Kind != Block {
		return false, false
	}
	return n.Speech && !n.Dual && i > n.HeadEnd, true
}

// LastTop is the heading line of the last section or scene at the top
// (-1: none): what "to the end" goes after.
func (t *Tree) LastTop() int {
	for i := len(t.Root.Kids) - 1; i >= 0; i-- {
		if t.Root.Kids[i].container() {
			return t.Root.Kids[i].First
		}
	}
	return -1
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

// Span is the scene (k Scene) or section (k Section) holding line i: its
// heading's line and its last line with something on it (a section: all
// it holds). ok is false when line i is in none (the title page, a
// section's own lines for a scene, the blocks at the end).
func (t *Tree) Span(i int, k Kind) (first, last int, ok bool) {
	var in *Node
	for _, n := range t.path(i) {
		if n.Kind == k {
			in = n
		}
	}
	if in == nil {
		return 0, 0, false
	}
	return in.First, in.Last, true
}

// SpanOf is the units of kind k from the one holding line i on, n of
// them in all, side by side in one parent (a scene in another section
// is not taken, nor that section's heading): first, the first's
// heading, and last, the last's last line.
func (t *Tree) SpanOf(i int, k Kind, n int) (first, last int, ok bool) {
	first, last, ok = t.Span(i, k)
	if !ok {
		return 0, 0, false
	}
	var at *Node
	for _, c := range t.containers() {
		if c.First == first && c.Kind == k {
			at = c
		}
	}
	units := t.ofKind(k)
	j := slices.Index(units, at)
	for m := 1; m < n && j+1 < len(units); m++ {
		nx := units[j+1]
		if nx.Parent != at.Parent {
			break
		}
		j++
		last = nx.Last
	}
	return first, last, true
}

// ofKind are the units of kind k in the text's order.
func (t *Tree) ofKind(k Kind) []*Node {
	var out []*Node
	for _, c := range t.containers() {
		if c.Kind == k {
			out = append(out, c)
		}
	}
	return out
}

// Trail is the last of the blank lines after line last, up to the next
// unit or the body's end: what "a scene" takes with it, as vim's "a
// paragraph" takes the blank lines after it.
func (t *Tree) Trail(last int) int {
	i := last // blank lines are no unit's: past the body's end too, up to an end block
	for i+1 < len(t.S.Lines) && t.sep(i+1) && strings.TrimSpace(t.S.Lines[i+1]) == "" {
		i++
	}
	return i
}

// HeadAfter is the heading line of the n-th unit of kind k after line i
// (before it: back), and whether there is one; past the last (first),
// the last (first) there is.
func (t *Tree) HeadAfter(i int, k Kind, n int, back bool) (int, bool) {
	units := t.ofKind(k)
	at := -1
	for j, u := range units {
		if back && u.First < i || !back && u.First > i {
			if back {
				at = j
			} else if at < 0 {
				at = j
			}
		}
	}
	if at < 0 {
		return 0, false
	}
	if back {
		at = max(at-(n-1), 0)
	} else {
		at = min(at+(n-1), len(units)-1)
	}
	return units[at].First, true
}

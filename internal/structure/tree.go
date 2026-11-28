// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Package structure moves the parts of a script as units, keeping it
// well-formed (LOOP.md: STRUCTURAL MOVES, paredit for scripts; swapping
// and moving parts of the script).
//
// A script's body nests: sections (# ## …) hold scenes and sections,
// scenes hold blocks (a speech: cue, parentheticals and dialogue as one;
// an action paragraph; a transition; a boneyard), and blocks hold lines.
// Fountain never closes a section: a scene belongs to the last heading
// above it, so a section's own blocks come first, then its scenes, then
// its sections, and every move here keeps that order. Each move works on
// the lines alone and is checked afterwards by reading the text again: the
// same units, the same lines, nothing else moved into another parent
// (else it is refused, and the text is left as it was).
//
// The title page and the blocks at the script's end (Characters, Source,
// Adaptation, Production, Notes) are never moved: the body is what lies
// between them.
package structure

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// Head is a section or scene heading, as the outline reads it.
type Head struct {
	Line  int
	Level int // a section's #s (a scene's: ignored)
	Scene bool
}

// Script is a text as lines, with what each line is (the editor's
// reading) and its headings (the outline's).
type Script struct {
	Lines []string
	Kinds []syntax.Kind
	Heads []Head
	// End is where the body ends: the lines from End on are the blocks
	// at the script's end, never moved
	End int
}

// Reader reads a text's lines as a Script, the same way before and after
// a move (the window's: its language's scene headings and cues).
type Reader func(lines []string) Script

// Kind is what a unit is.
type Kind int

const (
	Body Kind = iota
	Section
	Scene
	Block
	Line
)

func (k Kind) String() string {
	return [...]string{"script", "section", "scene", "block", "line"}[k]
}

// Node is a unit of the script.
type Node struct {
	Kind  Kind
	Level int // a section's #s
	// First and Last are its lines: from its first to its last line with
	// something on it (a section or scene: all it holds)
	First, Last int
	// HeadEnd is the last line of its heading's own lines (the heading
	// and the lines right under it: a synopsis, a note); a speech's: its
	// cue. Before First: none (the body, a block that is not a speech).
	HeadEnd int
	Speech  bool // a block: a speech (a cue and its lines)
	Dual    bool // a speech: two, side by side (the second cue's ^)
	Parent  *Node
	Kids    []*Node
}

// Tree is a script read as its units.
type Tree struct {
	S     Script
	Root  *Node
	Front int // the first line after the title page
	bone  []bool
}

// sep tells whether line i is a blank line between units.
func (t *Tree) sep(i int) bool {
	if i < 0 || i >= len(t.S.Lines) {
		return true
	}
	if t.bone[i] {
		return false
	}
	if i < len(t.S.Kinds) {
		return t.S.Kinds[i] == syntax.Empty
	}
	return strings.TrimSpace(t.S.Lines[i]) == ""
}

// lastContent is the last line in [from, to] with something on it (from-1:
// none).
func (t *Tree) lastContent(from, to int) int {
	for i := to; i >= from; i-- {
		if !t.sep(i) {
			return i
		}
	}
	return from - 1
}

// Parse reads s as its units.
func Parse(s Script) *Tree {
	n := len(s.Lines)
	end := min(max(s.End, 0), n)
	t := &Tree{S: s, bone: make([]bool, n)}
	// boneyards: /* … */, never split (their blank lines are theirs)
	for i := 0; i < n; i++ {
		tr := strings.TrimSpace(s.Lines[i])
		if !strings.HasPrefix(tr, "/*") {
			continue
		}
		j := i
		if !strings.Contains(strings.TrimPrefix(tr, "/*"), "*/") {
			for j = i + 1; j < n-1 && !strings.Contains(s.Lines[j], "*/"); j++ {
			}
		}
		for k := i; k <= min(j, n-1); k++ {
			t.bone[k] = true
		}
		i = j
	}
	// the title page: its fields up to the first blank line after them
	front := 0
	for i := 0; i < n && i < len(s.Kinds); i++ {
		if s.Kinds[i] == syntax.TitlePage {
			front = i + 1
		}
		if s.Kinds[i] != syntax.TitlePage && s.Kinds[i] != syntax.Empty {
			break
		}
	}
	for front < end && t.sep(front) {
		front++
	}
	t.Front = front
	t.Root = &Node{Kind: Body, First: front, Last: t.lastContent(front, end-1), HeadEnd: front - 1}

	// the sections and scenes, each in the last open one that can hold it
	var heads []Head
	for _, h := range s.Heads {
		if h.Line >= front && h.Line < end {
			heads = append(heads, h)
		}
	}
	var conts []*Node
	stack := []*Node{t.Root}
	for _, h := range heads {
		c := &Node{Kind: Section, Level: h.Level, First: h.Line}
		if h.Scene {
			c.Kind = Scene
		}
		for len(stack) > 1 {
			top := stack[len(stack)-1]
			if top.Kind == Scene || (c.Kind == Section && top.Level >= c.Level) {
				stack = stack[:len(stack)-1]
				continue
			}
			break
		}
		p := stack[len(stack)-1]
		c.Parent = p
		p.Kids = append(p.Kids, c)
		stack = append(stack, c)
		conts = append(conts, c)
	}
	// where each runs: to the next heading not inside it
	for k, c := range conts {
		stop := end
		for _, d := range conts[k+1:] {
			if c.Kind == Scene || d.Kind == Section && d.Level <= c.Level {
				stop = d.First
				break
			}
		}
		c.Last = t.lastContent(c.First, stop-1)
		h := c.First
		for h+1 <= c.Last && !t.sep(h+1) && !t.isHead(h+1, heads) {
			h++
		}
		c.HeadEnd = h
	}
	// the blocks of each: its own lines, before what it holds
	var blocks func(c *Node)
	blocks = func(c *Node) {
		from, to := c.HeadEnd+1, c.Last
		if len(c.Kids) > 0 {
			to = c.Kids[0].First - 1
		}
		var bs []*Node
		for i := from; i <= to; {
			if t.sep(i) {
				i++
				continue
			}
			j := i
			for j+1 <= to && !t.sep(j+1) {
				j++
			}
			b := &Node{Kind: Block, First: i, Last: j, HeadEnd: i - 1, Parent: c}
			b.Speech = j > i && !t.bone[i] && i < len(s.Kinds) && s.Kinds[i] == syntax.Character
			if b.Speech {
				b.HeadEnd = i
			}
			if b.Speech && strings.HasSuffix(strings.TrimSpace(s.Lines[i]), "^") && len(bs) > 0 && bs[len(bs)-1].Speech && !bs[len(bs)-1].Dual {
				p := bs[len(bs)-1] // the second of two side by side: one speech
				p.Last, p.Dual = j, true
			} else {
				bs = append(bs, b)
			}
			i = j + 1
		}
		for _, b := range bs {
			if !b.Dual && !t.bone[b.First] {
				for l := b.HeadEnd + 1; l <= b.Last; l++ {
					b.Kids = append(b.Kids, &Node{Kind: Line, First: l, Last: l, HeadEnd: l - 1, Parent: b})
				}
			}
		}
		c.Kids = append(bs, c.Kids...)
		for _, k := range c.Kids {
			if k.Kind == Section || k.Kind == Scene {
				blocks(k)
			}
		}
	}
	blocks(t.Root)
	return t
}

// isHead tells whether line i is one of heads'.
func (t *Tree) isHead(i int, heads []Head) bool {
	for _, h := range heads {
		if h.Line == i {
			return true
		}
	}
	return false
}

// path is the units holding line i, the body first.
func (t *Tree) path(i int) []*Node {
	out := []*Node{t.Root}
	n := t.Root
	for {
		var in *Node
		for _, k := range n.Kids {
			if k.First <= i && i <= k.Last {
				in = k
				break
			}
		}
		if in == nil {
			return out
		}
		out = append(out, in)
		n = in
	}
}

// index is n's place among its parent's.
func index(n *Node) int {
	for i, k := range n.Parent.Kids {
		if k == n {
			return i
		}
	}
	return -1
}

// container tells whether n holds units under a heading.
func (n *Node) container() bool { return n.Kind == Section || n.Kind == Scene }

// inHead tells whether line i is in n's heading (a container's, a
// speech's cue).
func (n *Node) inHead(i int) bool { return i >= n.First && i <= n.HeadEnd }

// all is every unit of the tree, in the text's order.
func (t *Tree) all() []*Node {
	var out []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		out = append(out, n)
		for _, k := range n.Kids {
			walk(k)
		}
	}
	walk(t.Root)
	return out
}

// containers are the sections and scenes in the text's order.
func (t *Tree) containers() []*Node {
	var out []*Node
	for _, n := range t.all() {
		if n.container() {
			out = append(out, n)
		}
	}
	return out
}

// Name is how a unit is named to the writer: its heading (a speech: its
// cue), or its first line.
func (t *Tree) Name(n *Node) string {
	if n.Kind == Body {
		return "the script's start"
	}
	s := strings.TrimSpace(t.S.Lines[n.First])
	if n.Kind == Section {
		s = strings.TrimSpace(strings.TrimLeft(s, "#"))
	}
	if r := []rune(s); len(r) > 40 {
		s = string(r[:39]) + "…"
	}
	return s
}

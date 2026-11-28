// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package structure

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The units inside a scene as vim's and helix's text objects read them
// (vim_objects.go): a speech, its cue, an action paragraph.

// Speech is one character's speech: its cue's line and its last line.
// Two side by side (the second cue's ^) are each a speech of their own,
// and Block is the pair's lines, the engine's one unit (a speech alone:
// its own lines).
type Speech struct {
	Cue, Last  int
	Block      [2]int
	Dual       bool
	Name       string // the cue's name: no @, extension or ^
	InnerFirst int    // its first line after the cue and the parentheticals right under it (Last+1: none)
}

// Speeches are the script's speeches in the text's order (outside the
// title page and the blocks at the end).
func (t *Tree) Speeches() []Speech {
	var out []Speech
	for _, n := range t.all() {
		if n.Kind != Block || !n.Speech || t.bone[n.First] {
			continue
		}
		cues := []int{n.First}
		if n.Dual {
			for l := n.First + 1; l <= n.Last; l++ {
				if l < len(t.S.Kinds) && t.S.Kinds[l] == syntax.Character && t.sep(l-1) && strings.HasSuffix(strings.TrimSpace(t.S.Lines[l]), "^") {
					cues = append(cues, l)
					break
				}
			}
		}
		for k, c := range cues {
			last := n.Last
			if k+1 < len(cues) {
				last = t.lastContent(c, cues[k+1]-1)
			}
			s := Speech{Cue: c, Last: last, Block: [2]int{n.First, n.Last}, Dual: n.Dual, Name: CueName(t.S.Lines[c])}
			s.InnerFirst = c + 1
			for s.InnerFirst <= last && s.InnerFirst < len(t.S.Kinds) && t.S.Kinds[s.InnerFirst] == syntax.Parenthetical {
				s.InnerFirst++
			}
			if s.InnerFirst > last { // parentheticals alone: they are what it says
				s.InnerFirst = c + 1
			}
			out = append(out, s)
		}
	}
	return out
}

// SpeechAt is the speech holding line i (a blank line in a pair: the
// first's).
func (t *Tree) SpeechAt(i int) (Speech, bool) {
	var in *Speech
	ss := t.Speeches()
	for k := range ss {
		s := &ss[k]
		if i >= s.Cue && i <= s.Last {
			return *s, true
		}
		if i >= s.Block[0] && i <= s.Block[1] && in == nil {
			in = s
		}
	}
	if in != nil {
		return *in, true
	}
	return Speech{}, false
}

// Paragraph is the action paragraph holding line i: its first and last
// lines (a paragraph whose first line is action, outside a boneyard).
func (t *Tree) Paragraph(i int) (first, last int, ok bool) {
	for _, n := range t.Paragraphs() {
		if i >= n[0] && i <= n[1] {
			return n[0], n[1], true
		}
	}
	return 0, 0, false
}

// Paragraphs are the action paragraphs' first and last lines, in order.
func (t *Tree) Paragraphs() [][2]int {
	var out [][2]int
	for _, n := range t.all() {
		if n.Kind == Block && !n.Speech && !t.bone[n.First] && n.First < len(t.S.Kinds) && t.S.Kinds[n.First] == syntax.Action {
			out = append(out, [2]int{n.First, n.Last})
		}
	}
	return out
}

// BlockAt is the block holding line i: a speech (a pair: both), a
// paragraph, a transition, a boneyard.
func (t *Tree) BlockAt(i int) (first, last int, ok bool) {
	for _, n := range t.path(i) {
		if n.Kind == Block {
			return n.First, n.Last, true
		}
	}
	return 0, 0, false
}

// BlockAfter is the first line of the first block that starts after
// line i (ok false: none before the body's end).
func (t *Tree) BlockAfter(i int) (int, bool) {
	best := -1
	for _, n := range t.all() {
		if n.Kind == Block && n.First > i && (best < 0 || n.First < best) {
			best = n.First
		}
	}
	return best, best >= 0
}

// InBoneyard tells whether line i is in a boneyard (/* … */).
func (t *Tree) InBoneyard(i int) bool { return i >= 0 && i < len(t.bone) && t.bone[i] }

// CueParts is a cue's line in three: what comes before the name (its
// indent, an @), the name, and what comes after (extensions, a ^).
func CueParts(l string) (pre, name, post string) { return cueParts(l) }

// CueName is a cue's name: no @, extension such as (V.O.) or (CONT'D),
// or ^.
func CueName(l string) string {
	_, n, _ := cueParts(l)
	return n
}

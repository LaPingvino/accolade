// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package structure

import (
	"fmt"
	"slices"
	"strings"
)

// check reads the moved text again and refuses a move that broke the
// script: the title page and the end blocks as they were; every section
// and scene still there, where its heading went, in the same parent
// unless it is what moved; a move of units the same lines, and the same
// speeches whole.
func (t *Tree) check(res Result, read Reader) error {
	broken := func(why string) error {
		return Refusal("Not moved: that would break the script's structure (" + why + ")")
	}
	s := read(res.Lines)
	u := Parse(s)
	if t.Front > len(res.Lines) || !slices.Equal(t.S.Lines[:t.Front], res.Lines[:t.Front]) {
		return broken("the title page")
	}
	if s.End < 0 || s.End > len(res.Lines) || !slices.Equal(t.S.Lines[t.S.End:], res.Lines[s.End:]) {
		return broken("the blocks at the end")
	}
	before, after := t.containers(), u.containers()
	if len(before) != len(after) {
		return broken(fmt.Sprintf("%d headings, then %d", len(before), len(after)))
	}
	at := map[int]*Node{} // the new units by their heading's line
	for _, c := range after {
		at[c.First] = c
	}
	where := func(c *Node) *Node { // c in the new text (the body: the body)
		if c.Kind == Body {
			return u.Root
		}
		if c.First < 0 || c.First >= len(res.Pos) || res.Pos[c.First] < 0 {
			return nil
		}
		return at[res.Pos[c.First]]
	}
	for _, c := range before {
		n := where(c)
		if n == nil || n.Kind != c.Kind {
			return broken("a heading would be lost: " + t.Name(c))
		}
		if !res.moved[c] && where(c.Parent) != n.Parent {
			return broken(t.Name(c) + " would land in another section")
		}
	}
	if !res.kept {
		return nil
	}
	if !slices.Equal(t.bodyLines(), u.bodyLines()) {
		return broken("the lines would change")
	}
	if !slices.Equal(t.speeches(), u.speeches()) {
		return broken("a speech would come apart")
	}
	return nil
}

// bodyLines are the body's lines with something on them, sorted.
func (t *Tree) bodyLines() []string {
	var out []string
	for i := t.Front; i < t.S.End && i < len(t.S.Lines); i++ {
		if s := strings.TrimSpace(t.S.Lines[i]); s != "" {
			out = append(out, s)
		}
	}
	slices.Sort(out)
	return out
}

// speeches are the speeches as their cue and their lines (in any
// order: a line moves within its speech), sorted.
func (t *Tree) speeches() []string {
	var out []string
	for _, n := range t.all() {
		if n.Kind == Block && n.Speech {
			var ls []string
			for l := n.HeadEnd + 1; l <= n.Last; l++ {
				if s := strings.TrimSpace(t.S.Lines[l]); s != "" {
					ls = append(ls, s)
				}
			}
			slices.Sort(ls)
			out = append(out, strings.TrimSpace(t.S.Lines[n.First])+"\n"+strings.Join(ls, "\n"))
		}
	}
	slices.Sort(out)
	return out
}

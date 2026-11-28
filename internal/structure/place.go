// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package structure

import (
	"fmt"
	"strings"
)

// Where is where Place puts a paragraph, by the unit it is put by.
type Where int

const (
	Before Where = iota // before the unit
	After               // after it, with all it holds
	Start               // in it (a section or scene), first after its heading's own lines
	End                 // in it, at the end of its own paragraphs (a section's: before its scenes); at line -1: the body's end
)

// Place moves the synopsis or note paragraph at line from by the unit
// whose first line is to (a section's or scene's heading, or a
// paragraph): before it, after it, or into it at its start or the end of
// its own paragraphs; with to -1 and End, to the end of the body. It is
// the cards' drag of a synopsis card, the same move of units as Alt+Up
// and Shift+Alt+Down, checked with read.
func (t *Tree) Place(from, to int, where Where, read Reader) (Result, error) {
	if from < t.Front || from >= t.S.End {
		return Result{}, Refusal("Moves work in the script's body")
	}
	n := t.unitAt(from)
	if n == nil || n.Kind != Block || !t.planning(one(n)) {
		return Result{}, Refusal("Only a synopsis or a note is placed so")
	}
	r := one(n)
	var target *Node
	if to < 0 {
		target = t.Root
	} else {
		for _, u := range t.all() {
			if u.Kind != Line && u.Kind != Body && u.First == to {
				target = u
				break
			}
		}
	}
	if target == nil {
		return Result{}, Refusal("Nothing there to put it by")
	}
	if target == n {
		return Result{}, InPlace
	}
	if (where == Start || where == End) && !target.container() && target.Kind != Body {
		where = After // a paragraph holds no other: after it
	}
	name := t.Name(target)
	var line int
	var after bool
	var msg string
	switch where {
	case Before:
		line, after, msg = target.First, false, "Before "+name
	case After:
		line, after, msg = target.Last, true, "After "+name
	case Start:
		msg = fmt.Sprintf("Into %s, at its start", name)
		if k := target.Kids; len(k) > 0 && k[0].Kind == Block && t.synopsis(k[0]) && !t.headSynopsis(target) && k[0] != n {
			line, after = k[0].Last, true // its own synopsis (the first) stays its own
		} else if len(target.Kids) > 0 {
			line, after = target.Kids[0].First, false
		} else {
			line, after = target.HeadEnd, true
		}
	case End:
		msg = fmt.Sprintf("Into %s, at its end", name)
		var last *Node // its own paragraphs, before what it holds
		for _, k := range target.Kids {
			if k.Kind != Block {
				break
			}
			last = k
		}
		switch {
		case target.Kind == Body:
			line, after, msg = target.Last, true, "To the end"
		case last != nil:
			line, after = last.Last, true
		case len(target.Kids) > 0:
			line, after = target.Kids[0].First, false
		default:
			line, after = target.HeadEnd, true
		}
	}
	if line >= n.First && line <= n.Last {
		return Result{}, InPlace
	}
	res, err := t.moveTo(r, line, after, msg)
	if err != nil {
		return Result{}, err
	}
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

// headSynopsis tells whether container c's heading lines (those touching
// it) hold a synopsis: its own, else its first synopsis paragraph is.
func (t *Tree) headSynopsis(c *Node) bool {
	for l := c.First + 1; l <= c.HeadEnd; l++ {
		if isSynopsisText(t.S.Lines[l]) {
			return true
		}
	}
	return false
}

// isSynopsisText tells whether line l is a synopsis line ("= …").
func isSynopsisText(l string) bool {
	l = strings.TrimSpace(l)
	return strings.HasPrefix(l, "=") && !strings.HasPrefix(l, "==")
}

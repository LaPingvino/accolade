package structure

import (
	"cmp"
	"fmt"
	"slices"
)

// Reorder puts the scenes of each section (and those before the first
// section) in the order rank gives them, as one move: rank is a scene's
// place by its heading's line (lower first); scenes it leaves out, and
// scenes of the same rank, keep their order. Scenes never leave their
// section, and the sections, the blank lines between the scenes and
// everything else stay where they are (the cards' "Make It the Order").
func (t *Tree) Reorder(rank map[int]int, read Reader) (Result, error) {
	var pieces []piece
	at := 0 // the next old line not yet in a piece
	moved := 0
	var first, last = -1, -1
	keys := map[*Node]bool{}
	for _, c := range append([]*Node{t.Root}, t.containers()...) {
		if c.Kind != Body && c.Kind != Section {
			continue
		}
		var scenes []*Node
		for _, k := range c.Kids {
			if k.Kind == Scene {
				scenes = append(scenes, k)
			}
		}
		if len(scenes) < 2 {
			continue
		}
		order := slices.Clone(scenes)
		key := func(n *Node) int {
			if r, ok := rank[n.First]; ok {
				return r
			}
			return int(^uint(0) >> 1)
		}
		slices.SortStableFunc(order, func(a, b *Node) int { return cmp.Compare(key(a), key(b)) })
		if slices.Equal(order, scenes) {
			continue
		}
		// the scenes' places, each filled by the one the order puts there;
		// the gaps between them stay
		for i, s := range scenes {
			if s.First > at {
				pieces = append(pieces, span(at, s.First-1))
			}
			n := order[i]
			pieces = append(pieces, span(n.First, n.Last))
			if n != s {
				moved++
				keys[n] = true
				if first < 0 || s.First < first {
					first = s.First
				}
				last = max(last, s.Last)
			}
			at = s.Last + 1
		}
	}
	if moved == 0 {
		return Result{}, InPlace
	}
	pieces = append(pieces, span(at, len(t.S.Lines)-1))
	lines, pos, _ := t.assemble(pieces, false)
	// what moved is all between the first and the last place changed (the
	// lengths are kept there: the same scenes, in another order)
	res := t.result(lines, pos, [2]int{first, last}, fmt.Sprintf("%d scenes in their new order", moved))
	res.moved, res.kept = keys, true
	if err := t.check(res, read); err != nil {
		return Result{}, err
	}
	return res, nil
}

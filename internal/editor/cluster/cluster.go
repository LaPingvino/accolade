// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Package cluster counts the columns a text takes in the editor's grid
// and where its clusters (drawn whole, the cursor round them) start: by
// Lekhan monospace (lekhan), and Arabic's lam-alef one cluster of one
// column, also with vowel marks on the lam (لَا, لّا, لّٰا), as Courier
// Badi draws it (one glyph a cell wide, from 1.109 over the marks) and the
// PDF, shaped first, prints it. Counted as two, lam-alef left a blank cell
// beside its glyph, a gap in the word.
//
// Every count of the editor's columns and clusters goes through here, so
// the wrap, the drawing and the cursor agree.
package cluster

import (
	"unicode"

	"github.com/LaPingvino/lexington/lekhan"
)

const lam = 0x0644

// alef: an alef that makes a ligature with a lam before it in Courier
// Badi (its rlig): alef, with madda, with hamza above or below. Not alef
// wasla (ٱ) nor the extended ones, which the font draws apart.
func alef(r rune) bool { return r == 0x0627 || r == 0x0622 || r == 0x0623 || r == 0x0625 }

// LamAlef tells whether rs[i] is the alef of a lam-alef: the first
// character before it that is no combining mark (Mn, Me) a lam. The font
// ligates over the marks (لَا); a tatweel or a joiner between ends the
// look-back, and the font draws the two apart (لـا, ل‍ا).
func LamAlef(rs []rune, i int) bool {
	if i <= 0 || i >= len(rs) || !alef(rs[i]) {
		return false
	}
	for k := i - 1; k >= 0; k-- {
		if !combining(rs[k]) {
			return rs[k] == lam
		}
	}
	return false
}

// combining tells whether r is a combining mark a lam-alef's look-back
// passes over: nonspacing or enclosing.
func combining(r rune) bool { return unicode.In(r, unicode.Mn, unicode.Me) }

// Joins tells whether b belongs to the cluster of a, the character before
// it, judged by the two alone: lekhan.Joins, or an alef after a lam or a
// mark (a lam-alef's if a lam is before the marks: Boundary decides). It
// is a quick filter: false is exact, true asks Boundary.
func Joins(a, b rune) bool { return alef(b) && (a == lam || combining(a)) || lekhan.Joins(a, b) }

// Boundary tells whether a cluster starts at rs[i].
func Boundary(rs []rune, i int) bool { return !LamAlef(rs, i) && lekhan.Boundary(rs, i) }

// End is where the cluster that starts at rs[i] ends.
func End(rs []rune, i int) int {
	j := i + 1
	for j < len(rs) && !Boundary(rs, j) {
		j++
	}
	return min(j, len(rs))
}

// Cells is the columns each rune of rs takes (lekhan.Cells; a lam-alef's
// alef none, its lam the one).
func Cells(rs []rune) []int {
	out := lekhan.Cells(rs)
	for i := 1; i < len(rs); i++ {
		if LamAlef(rs, i) {
			out[i] = 0
		}
	}
	return out
}

// Width is how many columns s takes.
func Width(s string) int {
	n := 0
	for _, c := range Cells([]rune(s)) {
		n += c
	}
	return n
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package buffer

import (
	"math/rand"
	"testing"
)

// TakeChange covers what changed since it was last taken, by any way in
// (edits, typing, Undo, Redo, Remote): the text before, with [start,
// oldEnd) replaced by [start, end) of the text now, is the text now.
func TestTakeChange(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	b := New("The quick brown fox jumps over the lazy dog.")
	b.TakeChange()
	before := []rune(b.Text())
	for i := 0; i < 2000; i++ {
		n := b.Len()
		from := r.Intn(n + 1)
		to := min(n, from+r.Intn(4))
		switch r.Intn(6) {
		case 0:
			b.Replace(from, to, []string{"", "x", "yz", "日本"}[r.Intn(4)])
		case 1:
			b.SetCursor(from, false)
			b.Insert("ab")
		case 2:
			b.Undo()
		case 3:
			b.Redo()
		case 4:
			b.Remote(from, to, "R")
		default:
			b.SetCursor(from, false)
			b.DeleteBackward()
		}
		if r.Intn(3) > 0 { // several changes at a time, too
			continue
		}
		now := []rune(b.Text())
		start, end, oldEnd, ok := b.TakeChange()
		if !ok {
			if string(now) != string(before) {
				t.Fatalf("step %d: no change said, the text changed", i)
			}
			continue
		}
		got := string(before[:start]) + string(now[start:end]) + string(before[oldEnd:])
		if got != string(now) {
			t.Fatalf("step %d: [%d,%d) for [%d,%d): %q, want %q", i, start, end, start, oldEnd, got, string(now))
		}
		before = now
	}
	b.SetText("new")
	if _, _, _, ok := b.TakeChange(); ok {
		t.Error("SetText: a span said, not the whole text")
	}
}

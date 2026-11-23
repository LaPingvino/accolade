package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The links looked at again whenever the source changes (Joop
// 2026-11-22: "checking shortest unique for start and finish … You could
// auto-fix that when new text breaks this, and when impossible signal
// this with a warning"). A new link is as short as finds its passages
// alone (textfrag.go's fragmentFor), so an edit elsewhere in the source
// can make it ambiguous, and an edit of the words it quotes can lose it.
// After a change (the source edited in Accolade and the cursor leaving
// the passage, the Source pane's Done, Save; the source file read again;
// a script opened, or another source chosen; Update Source Links) every
// link is looked at.
//
// The script is written only when that is certain (Joop 2026-11-23: the
// first source tool "had wildly inaccurate suggestions and started
// breaking the document"): a link whose passages are paired for sure
// (alignPassages: the same words, or the only passage between two that
// are) and that does not find exactly them now is written again for them,
// lengthened or re-anchored, one undo step, the status line saying how
// many. Anything weaker (passages paired by likeness and place, a link
// that only nearly finds its passage, the change not known) is not
// written: the link keeps its words, its chip warns ("↗ check?") with the
// guess in its tip, and a tap on it offers Move It There (one undo step)
// or Keep As It Is. A link that cannot be told apart (two passages alike,
// and their neighbours), or whose passages are gone, warns too: "↗ 2
// places", "↗ source?" (also when its few words now fit another passage).
//
// The adaptation block's entries (cut:, skipped links:) are written the
// same way, when certain; never guessed.

// relinkCount is what relinkFixes found.
type relinkCount struct {
	lengthened, reanchored int  // links written again (certain)
	kept                   int  // links that find their passages where they are now
	check                  int  // links with a guess to check (their chips)
	lost                   int  // links that find no passage of their own now
	entries                int  // the adaptation block's written again
	swap                   bool // another source for the script (said otherwise)
}

func (c relinkCount) any() bool { return c.lengthened+c.reanchored+c.check+c.entries > 0 }

// String is the status line's words for it.
func (c relinkCount) String() string {
	var parts []string
	if c.swap {
		parts = append(parts, groupDigits(c.lengthened+c.reanchored+c.kept)+" links moved", groupDigits(c.check)+" to check")
		if c.lost > 0 {
			parts = append(parts, groupDigits(c.lost)+" not found")
		}
	} else {
		if c.lengthened > 0 {
			parts = append(parts, plural(c.lengthened, "link")+" lengthened")
		}
		if c.reanchored > 0 {
			s := plural(c.reanchored, "link") + " re-anchored"
			if c.lengthened > 0 {
				s = fmt.Sprint(c.reanchored, " re-anchored")
			}
			parts = append(parts, s)
		}
		if c.check > 0 {
			s := plural(c.check, "link") + " to check"
			if c.lengthened+c.reanchored > 0 {
				s = fmt.Sprint(c.check, " to check")
			}
			parts = append(parts, s+" (its chip: ↗ check?)")
		}
	}
	if c.entries > 0 {
		e := "1 entry"
		if c.entries > 1 {
			e = fmt.Sprint(c.entries, " entries")
		}
		parts = append(parts, e+" of the adaptation block written again")
	}
	return strings.Join(parts, ", ")
}

// linkGuess is what the relinking thinks of a link it did not write: the
// passages [I, J] it seems to be about now and the directive for them
// (Target), or none; and why it is not sure.
type linkGuess struct {
	Target bool
	I, J   int
	Dir    textDirective
	Why    string
}

// relinkFixes are the edits that write the script's links again for
// the source as it is now (nw) where that is certain, the last first;
// old is the source before the change, nil if not known (a script
// opened: only guesses then). moved is each link written again: its key
// before (sourceLink.Key, or an old form's words) to its key after;
// guesses the links not written, by their rawKey.
func relinkFixes(script string, old, nw []passage) (edits []linkEdit, c relinkCount, moved map[string]string, guesses map[string]linkGuess) {
	if len(nw) == 0 {
		return nil, c, nil, nil
	}
	after := newLinkMatcher(nw)
	var before *linkMatcher
	var to []int
	var sure []bool
	if len(old) > 0 {
		before = newLinkMatcher(old)
		to, sure = alignPassagesSure(old, nw)
	}
	moved, guesses = map[string]string{}, map[string]linkGuess{}
	links := parseLinks(script)
	for a := 0; a < len(links); {
		b := a // the note's links: one note may hold several directives
		for b < len(links) && links[b].Start == links[a].Start {
			b++
		}
		note := links[a:b]
		a = b
		var dirs []textDirective
		changed := false
		for _, l := range note {
			dir := l.Ref.Dir
			if l.Ref.Old {
				dir = textDirective{Start: l.Ref.Quote}
			}
			now := after.matchRef(l.Ref)
			d, how, g := relinkLook(l.Ref, now, before, after, to, sure)
			switch how {
			case relinkFix:
				dir, changed = d, true
				if now.Kind == linkAmbiguous || before != nil && sameWords(before, after, l.Ref, d) {
					c.lengthened++
				} else {
					c.reanchored++
				}
				key := l.Key
				if l.Ref.Old {
					key = strings.Join(l.Words, " ")
				}
				moved[key] = linkRef{Dir: d}.rawKey()
				moved[l.Ref.rawKey()] = linkRef{Dir: d}.rawKey() // (an old form's other key: its guess goes with it)
			case relinkKept:
				c.kept++
			case relinkGuess:
				guesses[l.Ref.rawKey()] = g
				if g.Target {
					c.check++
				} else {
					c.lost++
				}
			default:
				if !now.found() && l.Ref.Bad == "" {
					c.lost++
				}
			}
			dirs = append(dirs, dir)
		}
		if changed {
			edits = append(edits, linkEdit{note[0].Start, note[0].End, fragmentNote(note[0].Ref.File, dirs)})
		}
	}
	// the adaptation block's entries, when certain: one edit of the block
	// (a boneyard, where parseLinks finds no links)
	if inside, ok := adaptationBlock(script); ok && before != nil {
		nu := inside
		fix := func(e cutMark) (cutMark, bool) {
			r := e.ref()
			if r.Bad != "" {
				return e, false
			}
			mt := before.matchRef(r)
			if !(mt.found() || e.Nth > 0 && mt.Kind == linkAmbiguous) {
				return e, false
			}
			was := cutAt(e, mt)
			if was.first < 0 {
				return e, false
			}
			i, j := to[was.first], to[was.last]
			if i < 0 || j < i || !sure[was.first] || !sure[was.last] {
				return e, false
			}
			nowMt := after.matchRef(r)
			if now := cutAt(e, nowMt); now.first == i && now.last == j && nowMt.Kind != linkNear {
				return e, false // it names them still
			}
			d, unique := fragmentFor(after, i, j)
			if !unique {
				return e, false
			}
			c.entries++
			return cutOf(linkRef{Dir: d}), true
		}
		cuts, changed := parseCuts(inside), false
		for k, e := range cuts {
			if e2, ok := fix(e); ok {
				cuts[k], changed = e2, true
			}
		}
		if changed {
			nu = withCuts(nu, cuts)
		}
		l, changed := parseLearned(nu), false
		for k, s := range l.Skipped {
			if e2, ok := fix(s.Link); ok {
				l.Skipped[k].Link, changed = e2, true
			}
		}
		if changed {
			nu = withLearned(nu, l)
		}
		if nu != inside {
			start, end, block := withAdaptationBlock(script, nu)
			edits = append(edits, linkEdit{start, end, block})
		}
	}
	slices.SortStableFunc(edits, func(x, y linkEdit) int { return y.Start - x.Start })
	return edits, c, moved, guesses
}

// sameWords reports whether the passages a link found before have the
// same words in after where d finds them (a link lengthened, its passage
// as it was).
func sameWords(before, after *linkMatcher, r linkRef, d textDirective) bool {
	was, now := before.matchRef(r), after.matchRef(linkRef{Dir: d})
	if !was.found() || !now.found() || was.last()-was.Passages[0] != now.last()-now.Passages[0] {
		return false
	}
	for k := 0; k <= was.last()-was.Passages[0]; k++ {
		if before.ps[was.Passages[0]+k].Plain != after.ps[now.Passages[0]+k].Plain {
			return false
		}
	}
	return true
}

// relinkHow is what is done with a link.
type relinkHow int

const (
	relinkNone  relinkHow = iota // it stays as it is, nothing to say
	relinkFix                    // written again for its passages (certain)
	relinkGuess                  // not written: a guess to check, or a warning
	relinkKept                   // it finds its passages, where they are now
)

// relinkLook is what becomes of a link after the change: the passages it
// found before the change (before; nil if not known) where they are now
// (to; sure where certain), written again for them if that is certain
// and it does not find exactly them now; else a guess. The change not
// known, a link that only nearly finds a passage gets a guess for it.
func relinkLook(r linkRef, now linkMatch, before, after *linkMatcher, to []int, sure []bool) (textDirective, relinkHow, linkGuess) {
	if r.Bad != "" {
		return textDirective{}, relinkNone, linkGuess{}
	}
	guess := func(i, j int, why string) (textDirective, relinkHow, linkGuess) {
		if d, unique := fragmentFor(after, i, j); unique {
			return textDirective{}, relinkGuess, linkGuess{Target: true, I: i, J: j, Dir: d, Why: why}
		}
		return textDirective{}, relinkNone, linkGuess{}
	}
	if before != nil {
		was := before.matchRef(r)
		switch {
		case was.found():
			i, j := to[was.Passages[0]], to[was.last()]
			switch {
			case i < 0 || j < i:
				if now.found() {
					return textDirective{}, relinkGuess, linkGuess{Why: "its passage was taken out of the source; its words fit another passage now"}
				}
				return textDirective{}, relinkNone, linkGuess{}
			case now.Kind == linkFound && now.Passages[0] == i && now.last() == j:
				return textDirective{}, relinkKept, linkGuess{} // it finds them still
			case sure[was.Passages[0]] && sure[was.last()]:
				if d, unique := fragmentFor(after, i, j); unique {
					return d, relinkFix, linkGuess{}
				}
				if now.found() { // another passage: say so
					return textDirective{}, relinkGuess, linkGuess{Why: "its passage reads as another one now, and nothing tells them apart"}
				}
				return textDirective{}, relinkNone, linkGuess{}
			}
			return guess(i, j, "its passage changed: this one seems to be it now, by its place and its words")
		case was.Kind == linkMissing && now.found():
			return textDirective{}, relinkGuess, linkGuess{Why: "it was not found before the source changed, and finds a passage now"}
		case was.Kind == linkAmbiguous && now.found():
			for _, x := range was.Passages {
				if to[x] == now.Passages[0] && now.last() == now.Passages[0] { // (a range: not known to be the same)
					return textDirective{}, relinkNone, linkGuess{} // one of those it fitted, the others changed
				}
			}
			return textDirective{}, relinkGuess, linkGuess{Why: "it fitted several passages before the source changed, and another one now"}
		}
	}
	if now.Kind == linkNear {
		return guess(now.Passages[0], now.last(), "its passage's words changed since it was linked: it nearly fits this one")
	}
	return textDirective{}, relinkNone, linkGuess{}
}

// alignPassages is where each passage of old is in nw (-1: gone), a
// monotone alignment of the two (Joop 2026-11-23: "fix a lot also by
// position instead of rough match … an almost complete rough match of a
// full new source and half-done edit"): the same passages at the start
// and the end; between, anchors: a passage there once on each side with
// the same words (the most of them in order; one out of order is moved
// and found all the same); and between two anchors the rest paired in
// order (pairStretch), their likeness and their place in the stretch the
// score, so a passage reworded past recognising still pairs with the one
// in its place. What is left unpaired on the old side is gone; on the
// new, added.
func alignPassages(old, nw []passage) []int {
	to, _ := alignPassagesSure(old, nw)
	return to
}

// alignPassagesSure is alignPassages, and which of its pairs are
// certain: the same words (at the ends, once on each side, or moved),
// or the only passage left between two such on each side.
func alignPassagesSure(old, nw []passage) ([]int, []bool) {
	sure := make([]bool, len(old))
	to := make([]int, len(old))
	for k := range to {
		to[k] = -1
	}
	p := 0
	for p < len(old) && p < len(nw) && old[p].Plain == nw[p].Plain {
		to[p], sure[p] = p, true
		p++
	}
	q := 0
	for q < len(old)-p && q < len(nw)-p && old[len(old)-1-q].Plain == nw[len(nw)-1-q].Plain {
		to[len(old)-1-q], sure[len(old)-1-q] = len(nw)-1-q, true
		q++
	}
	o, n := old[p:len(old)-q], nw[p:len(nw)-q] // the changed stretch
	if len(o) == 0 || len(n) == 0 {
		return to, sure
	}
	// anchors: the same words, once on each side
	count := func(ps []passage) map[string]int {
		m := map[string]int{}
		for _, x := range ps {
			m[x.Plain]++
		}
		return m
	}
	co, cn := count(o), count(n)
	at := map[string]int{}
	for k, x := range n {
		at[x.Plain] = k
	}
	type pair struct{ a, b int }
	var uniq []pair // in old's order
	for k, x := range o {
		if co[x.Plain] == 1 && cn[x.Plain] == 1 {
			uniq = append(uniq, pair{k, at[x.Plain]})
		}
	}
	// the most of them in order on both sides (a passage moved is one
	// out of order: the others stay anchors), as patience diff does
	tails, prev := []int{}, make([]int, len(uniq)) // tails[l]: the uniq ending the best run of l+1
	for k, u := range uniq {
		l := sort.Search(len(tails), func(l int) bool { return uniq[tails[l]].b >= u.b })
		prev[k] = -1
		if l > 0 {
			prev[k] = tails[l-1]
		}
		if l == len(tails) {
			tails = append(tails, k)
		} else {
			tails[l] = k
		}
	}
	run := make([]pair, len(tails))
	if len(tails) > 0 {
		for k, l := tails[len(tails)-1], len(tails)-1; k >= 0; k, l = prev[k], l-1 {
			run[l] = uniq[k]
		}
	}
	placed, taken := map[int]bool{}, map[int]bool{}
	for _, u := range uniq { // moved: found by its words all the same
		to[p+u.a], sure[p+u.a] = p+u.b, true
		placed[u.a], taken[u.b] = true, true
	}
	anchors := append(append([]pair{{-1, -1}}, run...), pair{len(o), len(n)})
	var lone []int // old passages alone between two anchors
	for k := 1; k < len(anchors); k++ {
		lo, hi := anchors[k-1], anchors[k]
		if lo.a >= 0 {
			to[p+lo.a], sure[p+lo.a] = p+lo.b, true
		}
		var xs, ys []int // the stretch between, not placed already
		for x := lo.a + 1; x < hi.a; x++ {
			if !placed[x] {
				xs = append(xs, x)
			}
		}
		for y := lo.b + 1; y < hi.b; y++ {
			if !taken[y] {
				ys = append(ys, y)
			}
		}
		if len(xs) == 1 && len(ys) == 1 { // the only one between its neighbours, on both sides
			to[p+xs[0]], sure[p+xs[0]] = p+ys[0], true
			lone = append(lone, xs[0])
			continue
		}
		for i, j := range pairStretch(o, n, xs, ys) {
			if j >= 0 {
				to[p+xs[i]] = p + ys[j]
			}
		}
	}
	// one alone in its place sharing too little with what is there now,
	// or nothing much like it while
	// another passage reads like it (it moved and was edited, or a copy
	// of it stayed), or what is there now reads like another one: not
	// sure
	for _, x := range lone {
		like := alike(o[x].Words, nw[to[p+x]].Words)
		if like < minSure { // its place alone: no words to go on (a passage taken out and another put there)
			sure[p+x] = false
			continue
		}
		if like >= 0.6 {
			continue
		}
		for z := range nw {
			if z != to[p+x] && bagAlike(o[x].Words, nw[z].Words) >= 0.6 {
				sure[p+x] = false
				break
			}
		}
		for z := range old { // and what is there now reads like another passage (copied or moved in)
			if z != p+x && sure[p+x] && bagAlike(nw[to[p+x]].Words, old[z].Words) >= 0.6 {
				sure[p+x] = false
				break
			}
		}
	}
	return to, sure
}

// minSure is how alike (alike) a passage alone between two anchors must
// be to what is there now for the pairing to be certain: below it, its
// place is all there is to go on, and a passage taken out with another
// put in its place reads the same (Joop: never write from a guess).
var minSure = 0.4

// pairStretch pairs the old passages xs with the new ones ys, in order
// (a Needleman–Wunsch alignment, a gap costing nothing): a pair scores
// its likeness and its place, how near the two are to the same share of
// the way through the stretch, and is made only if that is worth it.
// Two of another kind (a heading and a paragraph) pair only by their
// words. For each of xs, the index in ys it pairs with, or -1. A long
// stretch (a whole new source) looks only near its diagonal and compares
// words as a bag.
func pairStretch(o, n []passage, xs, ys []int) []int {
	A, B := len(xs), len(ys)
	out := make([]int, A)
	for i := range out {
		out[i] = -1
	}
	if A == 0 || B == 0 {
		return out
	}
	long := A*B > 1<<14
	band := max(A, B) // how far from the diagonal a pair is looked for
	if long {
		band = max(16, max(A, B)/20)
	}
	score := func(i, j int) float64 {
		x, y := o[xs[i]], n[ys[j]]
		d := float64(i) + 0.5 - (float64(j)+0.5)*float64(A)/float64(B) // off the diagonal, in old passages
		if d < 0 {
			d = -d
		}
		pos := 1 - min(1, 2*d/float64(max(A, 1)))
		if A == 1 || B == 1 {
			pos = 1 - min(1, d) // the one passage between its neighbours: as near as can be
		}
		var like float64
		if long {
			like = bagAlike(x.Words, y.Words)
		} else {
			like = alike(x.Words, y.Words)
		}
		if x.Plain == y.Plain {
			return 2 // word for word: above any likeness, a twin claims its own place
		}
		if x.Kind != y.Kind {
			return like - 0.5
		}
		return like + 0.5*pos - 0.3
	}
	// dp[i][j]: the best pairs of xs[:i] with ys[:j]
	dp := make([][]float64, A+1)
	for i := range dp {
		dp[i] = make([]float64, B+1)
	}
	pair := make([][]bool, A+1)
	for i := range pair {
		pair[i] = make([]bool, B+1)
	}
	for i := 1; i <= A; i++ {
		center := float64(i) * float64(B) / float64(A)
		for j := 1; j <= B; j++ {
			best := max(dp[i-1][j], dp[i][j-1])
			if c := float64(j) - center; c <= float64(band) && -c <= float64(band) {
				if sc := score(i-1, j-1); sc > 0 && dp[i-1][j-1]+sc > best {
					best = dp[i-1][j-1] + sc
					pair[i][j] = true
				}
			}
			dp[i][j] = best
		}
	}
	for i, j := A, B; i > 0 && j > 0; {
		switch {
		case pair[i][j] && dp[i][j] == dp[i-1][j-1]+score(i-1, j-1):
			out[i-1] = j - 1
			i, j = i-1, j-1
		case dp[i][j] == dp[i-1][j]:
			i--
		default:
			j--
		}
	}
	return out
}

// bagAlike is alike without the order: the words a and b share, as a
// share of the longer one's (for a long stretch, where order costs).
func bagAlike(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	m := make(map[string]int, len(a))
	for _, w := range a {
		m[w]++
	}
	same := 0
	for _, w := range b {
		if m[w] > 0 {
			m[w]--
			same++
		}
	}
	return float64(same) / float64(max(len(a), len(b)))
}

// alike is how many of a's words are b's, in the same order (their
// longest common run of words, gaps allowed), as a share of the longer
// one's: an edit or two leave most; another passage of the same words
// in another order, little. Only the first 64 words of each count.
func alike(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := max(len(a), len(b))
	a, b = a[:min(len(a), 64)], b[:min(len(b), 64)]
	row := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		diag := 0
		for j := 1; j <= len(b); j++ {
			up := row[j]
			if a[i-1] == b[j-1] {
				row[j] = diag + 1
			} else {
				row[j] = max(row[j], row[j-1])
			}
			diag = up
		}
	}
	return float64(row[len(b)]) / float64(min(n, 64))
}

// applyLinkEdits makes edits (the last first) to the script as one undo
// step, the cursor kept where it was in the text.
func (w *MainWindow) applyLinkEdits(edits []linkEdit) {
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			cursor := b.Cursor()
			for _, ed := range edits { // the last first: the offsets before stay
				b.Replace(ed.Start, ed.End, ed.Text)
				switch {
				case cursor >= ed.End:
					cursor += len([]rune(ed.Text)) - (ed.End - ed.Start)
				case cursor > ed.Start:
					cursor = ed.Start
				}
			}
			b.SetCursor(cursor, false)
		})
	})
}

// relink writes the script's links again for the source's change
// (relinkFixes' edits, certain), if the script is still script, and keeps
// the guesses for the links' chips; the status line says what it did. It
// reports whether it could look (the script had not changed).
func (w *MainWindow) relink(script string, edits []linkEdit, c relinkCount, moved map[string]string, guesses map[string]linkGuess) bool {
	if w.markdown || w.textEditor.Text() != script {
		return false
	}
	if len(guesses) > 0 || len(w.linkGuesses) > 0 && len(moved) > 0 {
		w.linkGuesses = carryGuesses(w.linkGuesses, moved, guesses)
		w.guessesChanged()
	}
	if len(edits) == 0 && !c.swap && c.check == 0 {
		return true
	}
	if len(edits) > 0 {
		w.applyLinkEdits(edits)
	}
	switch {
	case c.swap:
		w.setStatus("Another source: " + c.String() + " (Undo takes the moves back)")
	case len(edits) > 0:
		w.setStatus("The source changed: " + c.String() + " (Undo takes them back)")
	default:
		w.setStatus("The source changed: " + c.String())
	}
	if w.source != nil {
		w.source.follow()
	}
	return true
}

// carryGuesses is kept with the new guesses, and a warning of a link
// written again (moved: its key before to after) going with it: a link
// that warned still warns, written for the same passages.
func carryGuesses(kept map[string]linkGuess, moved map[string]string, guesses map[string]linkGuess) map[string]linkGuess {
	if kept == nil {
		kept = map[string]linkGuess{}
	}
	for from, to := range moved {
		if g, ok := kept[from]; ok && from != to {
			delete(kept, from)
			kept[to] = g
		}
	}
	maps.Copy(kept, guesses)
	return kept
}

// guessesChanged has the chips drawn again for the guesses.
func (w *MainWindow) guessesChanged() {
	w.chips.ok, w.chips.lines = false, nil
	if w.textEditor != nil {
		w.textEditor.RedrawMarks()
	}
}

// relinkInBackground is relink for the source's change from old to nw
// (their texts; ps, nw's passages, if read already), worked out in the
// background (a novel's passages take a while); swap, another source for
// the script. done says whether it could look. Looked at, nw is what the
// links were looked at against (for ref).
func (w *MainWindow) relinkInBackground(ref sourceRef, old, nw string, ps []passage, swap bool, done func(bool)) {
	if w.markdown {
		return
	}
	script, file := w.textEditor.Text(), w.currentFile
	type found struct {
		edits   []linkEdit
		c       relinkCount
		moved   map[string]string
		guesses map[string]linkGuess
	}
	w.relinking++
	inBackground(func() found {
		if ps == nil {
			ps = passagesOf(nw)
		}
		e, c, m, g := relinkFixes(script, passagesOf(old), ps)
		c.swap = swap
		return found{e, c, m, g}
	}, func(f found) {
		w.relinking--
		if s := w.source; !w.markdown && w.textEditor.Text() == script && s != nil && s.ref == ref {
			s.linkedSrc, s.linkedRef, s.linkedScript = nw, ref, file // (before relink's follow looks again)
		}
		ok := w.relink(script, f.edits, f.c, f.moved, f.guesses)
		if done != nil {
			done(ok)
		}
	})
}

// relinkAfterEdit looks at the links after the source document d was
// edited in Accolade: from its text when they were last looked at.
func (w *MainWindow) relinkAfterEdit(d *document) {
	if d == nil || d.kind != docSource || w.markdown {
		return
	}
	text, base := d.Text(), d.linkBase
	if sameString(text, base) {
		return
	}
	w.relinkInBackground(d.ref, base, text, nil, false, func(ok bool) {
		if ok && sameString(d.linkBase, base) {
			d.linkBase = text // else looked at again from base the next time (the script changed meanwhile)
		}
	})
}

// relinkRead looks at the links when the source pane reads the source r
// (its passages ps; the pane's source before, was and wasSrc):
//
//   - its file changed (read from disk again): from the text the links
//     were last looked at against;
//   - another source for the same script (Choose Source…: a new draft, a
//     translation, a heavily edited copy): from the one before, the whole
//     of it aligned with the new (alignPassages), the status saying how
//     many links moved, how many are to check, how many are not found;
//   - another script opened: from the source as it was when the script
//     was last saved (linkBaseOf), if that was this source and it has
//     changed since.
//
// Not for a source in the script (its edits are the script's, undone
// with it) or one being edited (relinkAfterEdit does, as its cursor
// leaves a passage).
func (s *sourcePane) relinkRead(r *sourceRead, was sourceRef, wasSrc string, ps []passage) {
	w := s.w
	switch {
	case r.ref.Origin == sourceNone || r.err != nil || w.markdown:
	case r.ref != was:
		swapFrom := ""
		if was.Origin != sourceNone && s.linkedRef == was && s.linkedScript == w.currentFile && s.linkedSrc != "" {
			swapFrom = s.linkedSrc // the same script, another source
		}
		s.linkedSrc, s.linkedRef, s.linkedScript = r.src, r.ref, w.currentFile
		if r.ref.Origin == sourceInside || sameString(r.src, wasSrc) { // (kept in the script now, the same text)
			break
		}
		if swapFrom != "" {
			if !sameString(swapFrom, r.src) {
				s.linkedSrc = swapFrom
				w.relinkInBackground(r.ref, swapFrom, r.src, ps, true, nil)
			}
			break
		}
		base, ok := readLinkBase(w.currentFile, r.ref)
		switch {
		case !ok:
			writeLinkBase(w.currentFile, r.ref, r.src) // the first time: as it is
			s.snapWritten = r.src
		case base != r.src:
			s.linkedSrc = base
			s.snapWritten = base
			w.relinkInBackground(r.ref, base, r.src, ps, false, nil)
		default:
			s.snapWritten = base
		}
	case r.disk && s.linkedRef == r.ref && !sameString(s.linkedSrc, r.src):
		w.relinkInBackground(r.ref, s.linkedSrc, r.src, ps, false, nil)
	}
}

// relinkAgain is the pane following: the links not looked at against
// the source as the pane read it (the last look dropped, the script
// having changed meanwhile), and none being looked at, they are now.
// Not for a source in the script, or one being edited (relinkAfterEdit
// does, as its cursor leaves a passage).
func (s *sourcePane) relinkAgain() {
	w := s.w
	switch {
	case w.relinking > 0 || w.markdown || s.err != nil || s.linkedRef != s.ref || sameString(s.linkedSrc, s.src):
	case s.ref.Origin == sourceNone || s.ref.Origin == sourceInside:
	case w.srcDoc != nil && w.srcDoc.ref == s.ref:
	default:
		w.relinkInBackground(s.ref, s.linkedSrc, s.src, s.passages, false, nil)
	}
}

// keepLinkBase keeps the source the script's links were last looked at
// against, for the script saved as path: what its next opening measures
// the source's change from.
func (w *MainWindow) keepLinkBase(path string) {
	s := w.source
	if s == nil || path == "" || s.linkedRef != s.ref || s.ref.Origin == sourceNone || s.ref.Origin == sourceInside ||
		s.linkedSrc == "" || path == w.currentFile && sameString(s.linkedSrc, s.snapWritten) {
		return
	}
	if writeLinkBase(path, s.ref, s.linkedSrc) == nil && path == w.currentFile {
		s.snapWritten = s.linkedSrc
	}
}

// linkBaseOf is where the source a script's links were last looked at
// against is kept (keepLinkBase): Accolade's data folder, beside the
// sources it keeps, by the script's path.
func linkBaseOf(script string) string {
	sum := sha256.Sum256([]byte(script))
	return filepath.Join(filepath.Dir(keptSourcesDir()), "links", hex.EncodeToString(sum[:12])+".md")
}

// linkRefKey is a source's place, as the kept base names it.
func linkRefKey(ref sourceRef) string {
	return fmt.Sprintf("%d\t%s\t%s", ref.Origin, ref.Path, ref.Entry)
}

// readLinkBase is the source kept for the script's links, if it was
// kept for ref (another source since: none).
func readLinkBase(script string, ref sourceRef) (string, bool) {
	if script == "" {
		return "", false
	}
	b, err := os.ReadFile(linkBaseOf(script))
	if err != nil {
		return "", false
	}
	head, src, ok := strings.Cut(string(b), "\n")
	if !ok || head != linkRefKey(ref) {
		return "", false
	}
	return src, true
}

// writeLinkBase keeps src as the source of the script's links (for ref).
func writeLinkBase(script string, ref sourceRef, src string) error {
	if script == "" || ref.Origin == sourceInside {
		return nil
	}
	p := linkBaseOf(script)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(p, []byte(linkRefKey(ref)+"\n"+src))
}

// relinkOnLeave is the main cursor moving: when it leaves a passage of
// the source after the text changed, the links are looked at (not while
// a passage's words are being typed).
func (w *MainWindow) relinkOnLeave() {
	d := w.midDoc()
	if d == nil || d.kind != docSource || w.source == nil {
		return
	}
	s := w.source
	if s.loading != nil {
		return
	}
	b := d.ed.Buffer()
	line, _ := b.Position(b.Cursor())
	if line == d.atLine { // typing in its line (a paragraph is one): the same passage
		return
	}
	d.atLine = line
	p := passageAt(d.Text(), s.passages, b, &w.passageSeen)
	if p != d.atPassage {
		if d.atPassage >= 0 && b.Rev() != d.seenRev {
			d.seenRev = b.Rev()
			w.relinkAfterEdit(d)
		}
		d.atPassage = p
	}
}

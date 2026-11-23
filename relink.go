package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
// a script opened, or a source chosen; Update Source Links) every link
// is looked at:
//
//   - one that found its passages before the change, and finds other
//     ones now or none (or several), is written again for them where
//     they are now: lengthened, or re-anchored to their new words;
//   - one that only nearly finds its passages (their words edited since
//     it was written) is re-anchored to them;
//   - one that cannot be told apart from another place (two passages
//     alike, and their neighbours), or whose passages are gone, warns: its
//     chip says "↗ 2 places" or "↗ source?", and so does the Source pane.
//     It is written so it does not find another passage, now or after
//     later edits: for passages alike, the words that fit them all
//     (ambiguous); for passages gone, their whole words as they were (not
//     found, and found again if they come back).
//
// The adaptation block's entries (cut:, skipped links:) the same. The
// edits are one undo step of the script's; the status line says how many.

// relinkCount is what relinkFixes wrote again.
type relinkCount struct {
	lengthened, reanchored int // links
	warned                 int // links written so they warn (they found another passage)
	entries                int // the adaptation block's
}

// String is the status line's words for it.
func (c relinkCount) String() string {
	var parts []string
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
	if c.warned > 0 {
		parts = append(parts, plural(c.warned, "link")+" that cannot find its passage alone now (its chip warns)")
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

// relinkFixes are the edits that write the script's links again for
// the source as it is now (nw), the last first; old is the source before
// the change, nil if not known (a script opened: only near links are
// re-anchored then). moved is each link written again: its key before
// (sourceLink.Key, or an old form's words) to its key after.
func relinkFixes(script string, old, nw []passage) (edits []linkEdit, c relinkCount, moved map[string]string) {
	if len(nw) == 0 {
		return nil, c, nil
	}
	after := newLinkMatcher(nw)
	var before *linkMatcher
	var to []int
	if len(old) > 0 {
		before, to = newLinkMatcher(old), alignPassages(old, nw)
	}
	moved = map[string]string{}
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
			if d, how := relinkDir(l.Ref, now, before, after, to); how != relinkNone {
				dir, changed = d, true
				switch {
				case how == relinkWarn:
					c.warned++
				case now.Kind == linkAmbiguous:
					c.lengthened++
				default:
					c.reanchored++
				}
				key := l.Key
				if l.Ref.Old {
					key = strings.Join(l.Words, " ")
				}
				moved[key] = linkRef{Dir: d}.rawKey()
			}
			dirs = append(dirs, dir)
		}
		if changed {
			edits = append(edits, linkEdit{note[0].Start, note[0].End, fragmentNote(note[0].Ref.File, dirs)})
		}
	}
	// the adaptation block's entries: one edit of the block (a boneyard,
	// where parseLinks finds no links)
	if inside, ok := adaptationBlock(script); ok {
		nu := inside
		fix := func(e cutMark) (cutMark, bool) {
			r := e.ref()
			nowMt := after.matchRef(r)
			now := cutAt(e, nowMt)
			t := relinkAt{first: -1}
			if before != nil {
				mt := before.matchRef(r)
				if mt.found() || e.Nth > 0 && mt.Kind == linkAmbiguous {
					if was := cutAt(e, mt); was.first >= 0 {
						t = targetOf(was.first, was.last, to, now.first >= 0 && nowMt.Kind != linkNear, now.first, now.last)
					}
				}
			}
			if t.first < 0 && nowMt.Kind == linkNear && now.first >= 0 {
				t = relinkAt{how: relinkFix, i: now.first, j: now.last}
			}
			d, how := t.dir(before, after, r.Dir, now.first >= 0, len(r.words())+len(normWords(r.Dir.End)))
			if how == relinkNone {
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
	return edits, c, moved
}

// relinkHow is what is done with a link.
type relinkHow int

const (
	relinkNone relinkHow = iota // it stays as it is
	relinkFix                   // written again for its passages
	relinkWarn                  // written so it warns: it found another passage
)

// relinkAt is a link's passages after the change: how it is to be
// written; [i, j] where they are now (relinkFix), or [first, last] where
// they were (relinkWarn for passages gone; first -1 when not known).
type relinkAt struct {
	how         relinkHow
	i, j        int
	first, last int
}

// targetOf is what a link that found old passages [first, last] before
// the change is to become: still found exactly where they are now
// (found, at [ni, nj]), it stays.
func targetOf(first, last int, to []int, found bool, ni, nj int) relinkAt {
	i, j := to[first], to[last]
	switch {
	case i < 0 || j < i: // gone (or no longer in order): nothing to anchor it to
		return relinkAt{how: relinkWarn, first: first, last: last}
	case found && ni == i && nj == j: // it finds them still
		return relinkAt{how: relinkNone, first: first, last: last}
	}
	return relinkAt{how: relinkFix, i: i, j: j, first: first, last: last}
}

// dir is the directive for it, the link's now cur: for passages now,
// the shortest that finds them alone, else the words that fit them and
// the others alike (ambiguous: a few more, and the context); for
// passages gone, found again elsewhere by their words (moved, nearly
// as they were), the shortest for them there; else their words as they
// were (in before's, wholeDir): not found, and found again if they come back,
// where its few words could later fit a passage added. relinkNone: it
// stays.
func (t relinkAt) dir(before, after *linkMatcher, cur textDirective, found bool, words int) (textDirective, relinkHow) {
	var d textDirective
	how := relinkWarn
	switch {
	case t.how == relinkFix:
		var unique bool
		if d, unique = fragmentFor(after, t.i, t.j); unique {
			how = relinkFix
		}
	case t.how == relinkWarn && before != nil:
		d = wholeDir(before, t.first, t.last)
		// moved (and edited): their words, many enough, found elsewhere
		if probe := (textDirective{Start: d.Start, End: d.End}); len(normWords(probe.Start)) >= 4 {
			if mt := after.matchRef(linkRef{Dir: probe}); mt.found() {
				if nd, unique := fragmentFor(after, mt.Passages[0], mt.last()); unique {
					return nd, relinkFix
				}
			}
		}
		if !found && words >= safeWords {
			return d, relinkNone // not found, and too many words to fit a passage added by chance: as it is
		}
	default:
		return d, relinkNone
	}
	if how == relinkWarn && d == cur {
		return d, relinkNone // as it is already
	}
	return d, how
}

// wholeDir is a link to passages [i, j] of lm's in many of their words:
// the first one's first wholeWords words as textStart, the last one's
// last as textEnd (one short block: all of it, textStart), and the words
// before them (else after): a short passage's words alone may be inside
// another one.
func wholeDir(lm *linkMatcher, i, j int) textDirective {
	const wholeWords = 8
	ps := lm.ps
	bi, bj := blockFields(ps[i].shown()), blockFields(ps[j].shown())
	if len(bi) == 0 || len(bj) == 0 {
		return textDirective{Start: ps[i].Plain}
	}
	fi, fj := bi[0], bj[len(bj)-1]
	ns, ne := wordFields(fi, wholeWords, false), wordFields(fj, wholeWords, true)
	d := textDirective{Start: strings.Join(fi, " ")}
	switch {
	case i == j && len(bi) == 1 && ns+ne >= len(fi):
	case i == j && len(bi) == 1:
		d = textDirective{Start: strings.Join(fi[:ns], " "), End: strings.Join(fi[len(fi)-ne:], " ")}
	default:
		d = textDirective{Start: strings.Join(fi[:ns], " "), End: strings.Join(fj[len(fj)-ne:], " ")}
	}
	const context = 3
	if q := lm.index().prev[i]; q >= 0 {
		if b := blockFields(ps[q].shown()); len(b) > 0 {
			f := b[len(b)-1]
			d.Prefix = strings.Join(f[len(f)-min(context, len(f)):], " ")
		}
	} else if q := lm.fi.next[j]; q >= 0 {
		if b := blockFields(ps[q].shown()); len(b) > 0 {
			d.Suffix = strings.Join(b[0][:min(context, len(b[0]))], " ")
		}
	}
	if normWords(d.Prefix) == nil {
		d.Prefix = ""
	}
	if normWords(d.Suffix) == nil {
		d.Suffix = ""
	}
	return d
}

// relinkDir is the directive a link is written as after the change, and
// how (relinkNone: it stays): the passages it found before the change
// (before; nil if not known) where they are now; or, the change not
// known, those it only nearly finds now.
func relinkDir(r linkRef, now linkMatch, before, after *linkMatcher, to []int) (textDirective, relinkHow) {
	if r.Bad != "" {
		return textDirective{}, relinkNone
	}
	t := relinkAt{first: -1}
	if before != nil {
		if was := before.matchRef(r); was.found() {
			ni, nj := -1, -1
			if now.Kind == linkFound {
				ni, nj = now.Passages[0], now.last()
			}
			t = targetOf(was.Passages[0], was.last(), to, ni >= 0, ni, nj)
		}
	}
	if t.first < 0 && now.Kind == linkNear {
		t = relinkAt{how: relinkFix, i: now.Passages[0], j: now.last()}
	}
	return t.dir(before, after, r.Dir, now.found(), len(r.words())+len(normWords(r.Dir.End)))
}

// alignPassages is where each passage of old is in nw (-1: gone): the
// same passages before and after the change, at its start and its end;
// between, a passage there once on each side with the same words (most
// of them in order; one out of order moved), and the others to the most
// alike in their stretch (alike, an edited passage). Nothing is guessed
// by its place alone: a link re-anchored to the wrong passage would be
// worse than one that warns.
func alignPassages(old, nw []passage) []int {
	to := make([]int, len(old))
	for k := range to {
		to[k] = -1
	}
	p := 0
	for p < len(old) && p < len(nw) && old[p].Plain == nw[p].Plain {
		to[p] = p
		p++
	}
	q := 0
	for q < len(old)-p && q < len(nw)-p && old[len(old)-1-q].Plain == nw[len(nw)-1-q].Plain {
		to[len(old)-1-q] = len(nw) - 1 - q
		q++
	}
	o, n := old[p:len(old)-q], nw[p:len(nw)-q] // the changed stretch
	if len(o) == 0 || len(n) == 0 {
		return to
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
		to[p+u.a] = p + u.b
		placed[u.a], taken[u.b] = true, true
	}
	anchors := append(append([]pair{{-1, -1}}, run...), pair{len(o), len(n)})
	for k := 1; k < len(anchors); k++ {
		lo, hi := anchors[k-1], anchors[k]
		if lo.a >= 0 {
			to[p+lo.a] = p + lo.b
		}
		// the stretch between: each to the one most alike, if it is the
		// most alike to it as well, and no other is as alike (in order)
		ga, gb := hi.a-lo.a-1, hi.b-lo.b-1
		if ga <= 0 || gb <= 0 || ga*gb > 1<<12 {
			continue
		}
		sim := make([][]float64, ga)
		for x := range sim {
			sim[x] = make([]float64, gb)
			if placed[lo.a+1+x] {
				continue
			}
			for y := range sim[x] {
				if !taken[lo.b+1+y] {
					sim[x][y] = alike(o[lo.a+1+x].Words, n[lo.b+1+y].Words)
				}
			}
		}
		bestOf := func(n int, at func(int) float64) int { // the one most alike, -1 if none is enough or two are as much
			best, bestAt, tie := minAlike, -1, false
			for k := range n {
				switch v := at(k); {
				case v > best:
					best, bestAt, tie = v, k, false
				case v == best && bestAt >= 0:
					tie = true
				}
			}
			if tie {
				return -1
			}
			return bestAt
		}
		from := -1
		for x := range ga {
			y := bestOf(gb, func(y int) float64 { return sim[x][y] })
			if y < 0 || y <= from || bestOf(ga, func(x2 int) float64 { return sim[x2][y] }) != x {
				continue
			}
			to[p+lo.a+1+x] = p + lo.b + 1 + y
			from = y
		}
	}
	return to
}

// safeWords is how many words a link not found may have and stay as it
// is when its passages are gone: one with fewer is written in more
// (wholeDir), lest a passage added later fit them by chance.
const safeWords = 6

// minAlike is how alike an edited passage is to what it was, at least.
const minAlike = 0.6

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

// relink writes the script's links again for the source's change from
// old to nw (relinkFixes), if the script is still script; the status
// line says what it did. It reports whether it could look (the script
// had not changed).
func (w *MainWindow) relink(script string, edits []linkEdit, c relinkCount, moved map[string]string) bool {
	if w.markdown || w.textEditor.Text() != script {
		return false
	}
	if len(edits) == 0 {
		return true
	}
	if s := w.source; s != nil && len(moved) > 0 && s.linkedWords != nil {
		// what the passages read when first seen linked goes with the
		// link: still "changed since linked" (before the edit, which reads
		// the links again)
		for key, seen := range s.linkedWords {
			base, rest, _ := strings.Cut(key, "\x03")
			if nu, ok := moved[base]; ok {
				if rest != "" {
					nu += "\x03" + rest
				}
				s.linkedWords[nu] = seen
			}
		}
	}
	w.applyLinkEdits(edits)
	w.setStatus("The source changed: " + c.String() + " (Undo takes them back)")
	if w.source != nil {
		w.source.follow()
	}
	return true
}

// relinkInBackground is relink for the source's change from old to nw
// (their texts; ps, nw's passages, if read already), worked out in the
// background (a novel's passages take a while); done says whether it
// could look. Looked at, nw is what the links were looked at against
// (for ref).
func (w *MainWindow) relinkInBackground(ref sourceRef, old, nw string, ps []passage, done func(bool)) {
	if w.markdown {
		return
	}
	script := w.textEditor.Text()
	type found struct {
		edits []linkEdit
		c     relinkCount
		moved map[string]string
	}
	w.relinking++
	inBackground(func() found {
		if ps == nil {
			ps = passagesOf(nw)
		}
		e, c, m := relinkFixes(script, passagesOf(old), ps)
		return found{e, c, m}
	}, func(f found) {
		w.relinking--
		ok := w.relink(script, f.edits, f.c, f.moved)
		if s := w.source; ok && s != nil && s.ref == ref {
			s.linkedSrc, s.linkedRef = nw, ref
		}
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
	w.relinkInBackground(d.ref, base, text, nil, func(ok bool) {
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
//   - another source than before (a script opened, a source chosen):
//     from the source as it was when the script was last saved
//     (linkBaseOf), if that was this source and it has changed since.
//
// Not for a source in the script (its edits are the script's, undone
// with it) or one being edited (relinkAfterEdit does, as its cursor
// leaves a passage).
func (s *sourcePane) relinkRead(r *sourceRead, was sourceRef, wasSrc string, ps []passage) {
	w := s.w
	switch {
	case r.ref.Origin == sourceNone || r.err != nil || w.markdown:
	case r.ref != was:
		s.linkedSrc, s.linkedRef = r.src, r.ref
		if r.ref.Origin == sourceInside || sameString(r.src, wasSrc) { // (kept in the script now, the same text)
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
			w.relinkInBackground(r.ref, base, r.src, ps, nil)
		default:
			s.snapWritten = base
		}
	case r.disk && s.linkedRef == r.ref && !sameString(s.linkedSrc, r.src):
		w.relinkInBackground(r.ref, s.linkedSrc, r.src, ps, nil)
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
		w.relinkInBackground(s.ref, s.linkedSrc, s.src, s.passages, nil)
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

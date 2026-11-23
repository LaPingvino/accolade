package scriptdiff

import (
	"sort"
	"strings"

	"github.com/LaPingvino/accolade/internal/pace"
)

// Merge Changes (LOOP.md, Joop 2026-11-20): a copy of the script edited
// outside Accolade (a co-writer's Highland file, an FDX from production,
// the flattened .fountain) comes back. With the version it was made from
// (the base, a stored draft) a three-way merge tells their changes from
// ours: what only they changed is offered, each change to take or leave;
// what only we changed stays; where both changed the same element, it is
// a conflict shown both ways. Nothing is taken unless the writer takes
// it (Apply).
//
// Scenes are matched first, by heading and content, so a scene they moved
// is a move and not a cut and an addition; then the elements of each
// scene, base against ours and theirs (diff3).

// MergeOptions say how the texts are read.
type MergeOptions struct {
	// IsHeading tells a line that starts a scene (a section's too, so
	// that a section stays where it is when scenes move)
	IsHeading func(string) bool
	// Body is where the part of a text to merge ends, in bytes (nil: all
	// of it); what comes after it in ours (Accolade's blocks at the end)
	// stays as it is
	Body func(string) int
	// Key is an element as compared: without what the outside copy could
	// not carry (links, numbers, notes). "" leaves the element out: it is
	// never offered, and stays in ours where it is. nil: the text.
	Key func(string) string
	// Carry is the text for ours' element when theirs is taken: theirs,
	// with what ours had that the comparison left out (nil: theirs)
	Carry func(ours, theirs string) string
}

// ChangeKind is what a change does.
type ChangeKind int

const (
	ChangeEdit  ChangeKind = iota // elements of a scene changed, cut or added (Row says which)
	SceneCut                      // they cut a whole scene
	SceneAdded                    // they added a scene
	SceneMoved                    // they moved a scene elsewhere
)

// Change is one of their changes, to take or leave.
type Change struct {
	ID   int
	Kind ChangeKind
	// Row is an edit's: Removed (they cut the elements), Added (they
	// added them) or Changed
	Row Kind
	// Conflict: we changed it too (or cut it): Ours and Theirs differ
	// from Base and from each other
	Conflict bool
	// the texts, elements joined by a blank line ("" none): Theirs as
	// taking it writes it
	Base, Ours, Theirs string
	Words              []Word // ours → theirs word by word, a one-element change's
	Scene              int    // its scene in Merge.Scenes
}

// MergeScene is a scene with changes, in the script's order.
type MergeScene struct {
	Heading string // ours' heading; theirs' for a scene they added ("" before the first scene)
	Changes []int
	New     bool // a scene they added (or one we cut and they changed)
}

// Merge is a three-way merge: their changes against the base, onto ours.
type Merge struct {
	Changes []Change
	Scenes  []MergeScene
	Kept    int // our own changes, kept as they are

	ours    string
	body    int // ours' body end
	oe      []melem
	groups  []*mgroup // ours' groups in ours' order, then theirs-only ones
	nOurs   int
	deps    map[int][]int // group → the placed groups after it, in theirs' order
	lineAt  []int         // ours' line starts (bytes)
	nLines  int
	firstAt int // ours' first element's start
}

// melem is an element as merged: its text, key and (ours') bytes.
type melem struct {
	Element
	key        string
	start, end int // ours': from its first byte to the next element's
	textEnd    int // the end of its last line
}

// mslot is a stretch of a scene: ours' elements (the text there) or, if
// change is taken, theirs.
type mslot struct {
	ours   []int    // ours' elements (indexes in Merge.oe)
	theirs []string // the text taking it writes, one element each
	change int      // -1: always ours'
}

// mgroup is a scene as merged.
type mgroup struct {
	slots  []mslot
	whole  int // a SceneCut, SceneMoved or SceneAdded change (-1 none)
	placed bool
	moved  bool
}

type mgroupIn struct {
	elems   []melem
	heading string // its heading's key ("" none)
	keys    map[string]int
	n       int // its elements compared
}

// groups cuts elements into scenes: the elements before the first
// heading (always, maybe none), then one a heading.
func groupsOf(es []melem) []mgroupIn {
	gs := []mgroupIn{{}}
	for _, e := range es {
		if e.Heading {
			gs = append(gs, mgroupIn{heading: e.key})
		}
		g := &gs[len(gs)-1]
		g.elems = append(g.elems, e)
	}
	for i := range gs {
		g := &gs[i]
		g.keys = map[string]int{}
		for k, e := range g.elems {
			if e.key == "" {
				continue
			}
			g.n++
			if k == 0 && e.Heading {
				continue
			}
			g.keys[e.key]++
		}
	}
	return gs
}

// NewMerge merges theirs' changes since base onto ours, nothing taken.
func NewMerge(base, ours, theirs string, o MergeOptions) *Merge {
	key := o.Key
	if key == nil {
		key = func(s string) string { return s }
	}
	carry := o.Carry
	if carry == nil {
		carry = func(_, t string) string { return t }
	}
	body := func(s string) int {
		if o.Body == nil {
			return len(s)
		}
		return o.Body(s)
	}
	read := func(s string) []melem {
		es := SplitAt(s[:body(s)], o.IsHeading)
		out := make([]melem, len(es))
		for i, e := range es {
			out[i] = melem{Element: e.Element, key: key(e.Text), start: e.Start, end: e.End, textEnd: e.TextEnd}
		}
		return out
	}
	m := &Merge{ours: ours, body: body(ours), deps: map[int][]int{}}
	m.oe = read(ours)
	m.lineAt = []int{0}
	for i := 0; i < len(ours); i++ {
		if ours[i] == '\n' {
			m.lineAt = append(m.lineAt, i+1)
		}
	}
	m.nLines = len(m.lineAt)
	m.firstAt = m.body
	if len(m.oe) > 0 {
		m.firstAt = m.oe[0].start
	}
	bg, og, tg := groupsOf(read(base)), groupsOf(m.oe), groupsOf(read(theirs))
	// ours' elements by group, as indexes into m.oe
	oIdx := make([][]int, len(og))
	at := 0
	for g := range og {
		for range og[g].elems {
			oIdx[g] = append(oIdx[g], at)
			at++
		}
	}
	bo := matchGroups(bg, og) // base → ours
	bt := matchGroups(bg, tg) // base → theirs
	ob := invert(bo, len(og))
	tb := invert(bt, len(tg))
	// scenes both added: ours' and theirs' not in the base, matched
	var oNew, tNew []int
	for i := 1; i < len(og); i++ {
		if ob[i] < 0 && og[i].n > 0 {
			oNew = append(oNew, i)
		}
	}
	for i := 1; i < len(tg); i++ {
		if tb[i] < 0 && tg[i].n > 0 {
			tNew = append(tNew, i)
		}
	}
	ot := map[int]int{} // ours → theirs, for both-added
	to := map[int]int{}
	if len(oNew) > 0 && len(tNew) > 0 {
		sub := func(gs []mgroupIn, idx []int) []mgroupIn {
			out := []mgroupIn{{}}
			for _, i := range idx {
				out = append(out, gs[i])
			}
			return out
		}
		mm := matchGroups(sub(og, oNew), sub(tg, tNew))
		for i, j := range mm {
			if i > 0 && j > 0 {
				ot[oNew[i-1]], to[tNew[j-1]] = tNew[j-1], oNew[i-1]
			}
		}
	}
	for b, o := range bo {
		if o >= 0 && bt[b] >= 0 {
			ot[o], to[bt[b]] = bt[b], o
		}
	}
	// moved: out of the longest run in base's order, theirs' and ours'
	var tSeq, oSeq []int
	for j := 1; j < len(tg); j++ {
		if b := tb[j]; b > 0 && bo[b] > 0 {
			tSeq = append(tSeq, b)
		}
	}
	for j := 1; j < len(og); j++ {
		if b := ob[j]; b > 0 && bt[b] > 0 {
			oSeq = append(oSeq, b)
		}
	}
	tKeep, oKeep := lisSet(tSeq), lisSet(oSeq)
	// anchor: the ours group a theirs group follows (the nearest before
	// it in theirs that ours has; 0 the start)
	anchor := func(j int) int {
		for k := j - 1; k > 0; k-- {
			if o, ok := to[k]; ok {
				return o
			}
		}
		return 0
	}

	add := func(c Change) int {
		c.ID = len(m.Changes)
		m.Changes = append(m.Changes, c)
		return c.ID
	}
	join := func(es []melem) string {
		s := make([]string, len(es))
		for i, e := range es {
			s[i] = e.Text
		}
		return strings.Join(s, "\n\n")
	}
	texts := func(es []melem) []string {
		s := make([]string, len(es))
		for i, e := range es {
			s[i] = e.Text
		}
		return s
	}

	// ours' groups, merged
	for g := range og {
		pace.Yield()
		mg := &mgroup{whole: -1}
		m.groups = append(m.groups, mg)
		b := ob[g]
		t, hasT := ot[g]
		if g == 0 {
			b, t, hasT = 0, 0, true
		}
		switch {
		case b >= 0 && hasT: // in all three: their changes to it
			if b > 0 && !tKeep[b] && oKeep[b] {
				mg.moved = true
				mg.whole = add(Change{Kind: SceneMoved, Ours: og[g].elems[0].Text, Theirs: og[g].elems[0].Text})
				m.deps[anchor(t)] = append(m.deps[anchor(t)], g)
			}
			mg.slots = m.merge3(bg[b].elems, og[g].elems, oIdx[g], tg[t].elems, add, carry)
		case b < 0 && hasT: // both added it
			mg.slots = m.merge3(nil, og[g].elems, oIdx[g], tg[t].elems, add, carry)
		case b > 0 && !hasT && bg[b].n > 0: // they cut it
			mg.slots = []mslot{{ours: oIdx[g], change: -1}}
			c := Change{Kind: SceneCut, Row: Removed, Base: join(bg[b].elems), Ours: join(og[g].elems)}
			c.Conflict = !sameKeys(bg[b].elems, og[g].elems)
			mg.whole = add(c)
		default: // ours' own
			mg.slots = []mslot{{ours: oIdx[g], change: -1}}
		}
	}
	m.nOurs = len(m.groups)
	// theirs' scenes ours has not: added, or cut by us and changed by them
	for j := 1; j < len(tg); j++ {
		if _, ok := to[j]; ok || tg[j].n == 0 {
			continue
		}
		c := Change{Kind: SceneAdded, Row: Added, Theirs: join(tg[j].elems)}
		if b := tb[j]; b > 0 {
			if sameKeys(bg[b].elems, tg[j].elems) {
				continue // we cut it, they left it: cut
			}
			c.Conflict, c.Base = true, join(bg[b].elems)
		}
		id := add(c)
		mg := &mgroup{whole: id, placed: true, slots: []mslot{{theirs: texts(tg[j].elems), change: id}}}
		m.groups = append(m.groups, mg)
		a := anchor(j)
		m.deps[a] = append(m.deps[a], len(m.groups)-1)
	}
	// the placed after each group in theirs' order: they were added so
	m.scenes(og, tg)
	return m
}

// scenes lists the scenes with changes, in the order shown: ours', a
// scene they added after the one it follows.
func (m *Merge) scenes(og, tg []mgroupIn) {
	shown := map[int]bool{}
	var visit func(g int)
	visit = func(g int) {
		if shown[g] {
			return
		}
		shown[g] = true
		mg := m.groups[g]
		var ids []int
		if mg.whole >= 0 {
			ids = append(ids, mg.whole)
		}
		for _, s := range mg.slots {
			if s.change >= 0 && s.change != mg.whole {
				ids = append(ids, s.change)
			}
		}
		if len(ids) > 0 {
			h := ""
			if g < m.nOurs {
				if s := m.groupElems(g); len(s) > 0 && m.oe[s[0]].Heading {
					h = m.oe[s[0]].Text
				}
			} else if t := mg.slots[0].theirs; len(t) > 0 {
				h = t[0]
			}
			sc := len(m.Scenes)
			m.Scenes = append(m.Scenes, MergeScene{Heading: h, Changes: ids, New: mg.placed})
			for _, id := range ids {
				m.Changes[id].Scene = sc
			}
		}
		for _, d := range m.deps[g] {
			if m.groups[d].placed {
				visit(d)
			}
		}
	}
	for g := 0; g < m.nOurs; g++ {
		visit(g)
	}
}

// groupElems are ours' elements in group g.
func (m *Merge) groupElems(g int) []int {
	var out []int
	for _, s := range m.groups[g].slots {
		out = append(out, s.ours...)
	}
	sort.Ints(out)
	return out
}

// merge3 merges a scene's elements: base's, ours' (at their indexes
// oi) and theirs'. Elements left out of the comparison (no key) stay in
// ours where they are.
func (m *Merge) merge3(b, o []melem, oi []int, t []melem, add func(Change) int, carry func(o, t string) string) []mslot {
	in := func(es []melem) (keys []string, at []int) {
		for i, e := range es {
			if e.key != "" {
				keys, at = append(keys, e.key), append(at, i)
			}
		}
		return
	}
	bk, _ := in(b)
	ok, oat := in(o)
	tk, tat := in(t)
	mo, mt := matches(bk, ok), matches(bk, tk)
	var slots []mslot
	oc := 0 // ours' next element not yet in a slot
	flush := func(to int) { // ours' left-out elements before to
		for ; oc < to; oc++ {
			if o[oc].key == "" {
				slots = append(slots, mslot{ours: []int{oi[oc]}, change: -1})
			}
		}
	}
	oursSlot := func(js []int, change int, theirs []string) {
		var idx []int
		for _, j := range js {
			flush(oat[j])
			idx = append(idx, oi[oat[j]])
			oc = oat[j] + 1
		}
		slots = append(slots, mslot{ours: idx, theirs: theirs, change: change})
	}
	textsOf := func(es []melem, at []int, from, to int) []string {
		var s []string
		for k := from; k < to; k++ {
			s = append(s, es[at[k]].Text)
		}
		return s
	}
	span := func(from, to int) []int {
		var s []int
		for k := from; k < to; k++ {
			s = append(s, k)
		}
		return s
	}
	i, j, k := 0, 0, 0
	for i < len(bk) || j < len(ok) || k < len(tk) {
		pace.Yield()
		// the next base element in all three
		n := i
		for n < len(bk) && (mo[n] < 0 || mt[n] < 0) {
			n++
		}
		j2, k2 := len(ok), len(tk)
		if n < len(bk) {
			j2, k2 = mo[n], mt[n]
		}
		if n == i && n < len(bk) && j2 == j && k2 == k { // the same in all
			oursSlot([]int{j}, -1, nil)
			i, j, k = i+1, j+1, k+1
			continue
		}
		bs, os, ts := bk[i:n], ok[j:j2], tk[k:k2]
		switch {
		case equal(ts, bs) || equal(os, ts): // theirs left it, or both did the same
			if !equal(os, bs) && !equal(os, ts) {
				m.Kept++
			}
			if len(os) > 0 {
				oursSlot(span(j, j2), -1, nil)
			}
		case equal(os, bs): // only theirs changed it: each change on its own
			oe := make([]Element, len(os))
			for x := range os {
				oe[x] = Element{Text: os[x]}
			}
			te := make([]Element, len(ts))
			for x := range ts {
				te[x] = Element{Text: ts[x]}
			}
			for _, r := range diffIdx(oe, te) {
				switch r.kind {
				case Same:
					oursSlot([]int{j + r.i}, -1, nil)
				case Removed:
					oText := o[oat[j+r.i]].Text
					oursSlot([]int{j + r.i}, add(Change{Row: Removed, Base: oText, Ours: oText}), []string{})
				case Added:
					tText := t[tat[k+r.j]].Text
					oursSlot(nil, add(Change{Row: Added, Theirs: tText}), []string{tText})
				case Changed:
					oText := o[oat[j+r.i]].Text
					tText := carry(oText, t[tat[k+r.j]].Text)
					oursSlot([]int{j + r.i}, add(Change{Row: Changed, Base: oText, Ours: oText, Theirs: tText, Words: Words(oText, tText)}), []string{tText})
				}
			}
		case len(bs) > 1 && len(os) == len(bs) && len(ts) == len(bs): // element by element, side by side
			for x := range bs {
				oText, bText := o[oat[j+x]].Text, b[mustAt(b)[i+x]].Text
				tText := t[tat[k+x]].Text
				switch {
				case ts[x] == bs[x] || os[x] == ts[x]:
					if os[x] != bs[x] && os[x] != ts[x] {
						m.Kept++
					}
					oursSlot([]int{j + x}, -1, nil)
				case os[x] == bs[x]:
					tText = carry(oText, tText)
					oursSlot([]int{j + x}, add(Change{Row: Changed, Base: bText, Ours: oText, Theirs: tText, Words: Words(oText, tText)}), []string{tText})
				default:
					tText = carry(oText, tText)
					oursSlot([]int{j + x}, add(Change{Conflict: true, Row: Changed, Base: bText, Ours: oText, Theirs: tText, Words: Words(oText, tText)}), []string{tText})
				}
			}
		default: // both changed it, otherwise: a conflict
			ot := textsOf(o, oat, j, j2)
			tt := textsOf(t, tat, k, k2)
			if len(ot) == 1 && len(tt) == 1 {
				tt[0] = carry(ot[0], tt[0])
			}
			c := Change{Conflict: true, Row: Changed, Base: strings.Join(textsOf(b, mustAt(b), i, n), "\n\n"),
				Ours: strings.Join(ot, "\n\n"), Theirs: strings.Join(tt, "\n\n")}
			switch {
			case len(tt) == 0:
				c.Row = Removed
			case len(ot) == 0:
				c.Row = Added
			}
			if len(ot) == 1 && len(tt) == 1 {
				c.Words = Words(c.Ours, c.Theirs)
			}
			if tt == nil {
				tt = []string{}
			}
			oursSlot(span(j, j2), add(c), tt)
		}
		i, j, k = n, j2, k2
	}
	flush(len(o))
	return slots
}

// mustAt are the indexes of es's compared elements.
func mustAt(es []melem) []int {
	var at []int
	for i, e := range es {
		if e.key != "" {
			at = append(at, i)
		}
	}
	return at
}

// Apply is ours with the changes taken: take (their text), or edits (a
// text of the writer's own for a change, taken so). It gives the text,
// and for each of ours' lines the new line it is (-1 gone, or its text
// changed: Map) and where it went (Pos: a changed one too).
func (m *Merge) Apply(take map[int]bool, edits map[int]string) (text string, lineMap, pos []int) {
	taken := func(id int) bool {
		if id < 0 {
			return false
		}
		if _, ok := edits[id]; ok {
			return true
		}
		return take[id]
	}
	lineMap, pos = make([]int, m.nLines), make([]int, m.nLines)
	for i := range lineMap {
		lineMap[i], pos[i] = -1, -1
	}
	var b strings.Builder
	newLine := 0
	write := func(s string) {
		b.WriteString(s)
		newLine += strings.Count(s, "\n")
	}
	lineOf := func(off int) int { return sort.SearchInts(m.lineAt, off+1) - 1 }
	mapLines := func(from, to int, as bool) { // ours' bytes [from, to) written from newLine on
		l0, l1 := lineOf(from), lineOf(max(from, to-1))
		if to <= from {
			return
		}
		for l := l0; l <= l1; l++ {
			n := newLine + l - l0
			pos[l] = n
			if as {
				lineMap[l] = n
			}
		}
	}
	// the separation before the next element: ours' bytes run on
	lastEnd, need := -1, 0
	sep := func(start int) {
		if b.Len() == 0 || start == lastEnd {
			return
		}
		write(strings.Repeat("\n", need))
	}
	oursElem := func(i int) {
		e := m.oe[i]
		sep(e.start)
		mapLines(e.start, e.end, true)
		write(m.ours[e.start:e.end])
		lastEnd, need = e.end, max(0, 2-strings.Count(m.ours[e.textEnd:e.end], "\n"))
	}
	theirsElem := func(s string) {
		sep(-2)
		write(s + "\n\n")
		lastEnd, need = -1, 0
	}
	write(m.ours[:m.firstAt])
	lastEnd = m.firstAt
	var emit func(g int)
	emit = func(g int) {
		mg := m.groups[g]
		if !(mg.whole >= 0 && !mg.moved && !mg.placed && taken(mg.whole)) { // a cut scene taken: none of it
			for _, s := range mg.slots {
				if !taken(s.change) {
					for _, i := range s.ours {
						oursElem(i)
					}
					continue
				}
				at := newLine
				ts := s.theirs
				if e, ok := edits[s.change]; ok {
					ts = nil
					if strings.TrimSpace(e) != "" {
						ts = []string{strings.TrimRight(e, "\n ")}
					}
				}
				for _, t := range ts {
					theirsElem(t)
				}
				for _, i := range s.ours {
					e := m.oe[i]
					if len(ts) > 0 {
						for l := lineOf(e.start); l <= lineOf(max(e.start, e.end-1)); l++ {
							pos[l] = at
						}
					}
				}
			}
		}
		for _, d := range m.deps[g] {
			if dg := m.groups[d]; taken(dg.whole) {
				emit(d)
			}
		}
	}
	for g := 0; g < m.nOurs; g++ {
		if mg := m.groups[g]; mg.moved && taken(mg.whole) {
			continue // where it goes, after its anchor
		}
		emit(g)
	}
	tail := m.ours[m.body:]
	if b.Len() > 0 && lastEnd != m.body {
		out := b.String()
		trimmed := strings.TrimRight(out, "\n")
		end := m.ours[:m.body]
		want := len(end) - len(strings.TrimRight(end, "\n")) // ours' body ended with these
		if tail != "" {
			want = max(want, 2)
		}
		newLine -= len(out) - len(trimmed)
		b.Reset()
		b.WriteString(trimmed)
		write(strings.Repeat("\n", want))
	}
	mapLines(m.body, len(m.ours), true)
	write(tail)
	if strings.HasSuffix(m.ours, "\n") { // the last, empty line
		lineMap[m.nLines-1], pos[m.nLines-1] = newLine, newLine
	}
	return b.String(), lineMap, pos
}

// matchGroups matches a's scenes with b's: by heading and content, the
// best first; the first group (before any heading) with the first. -1:
// none.
func matchGroups(a, b []mgroupIn) []int {
	out := make([]int, len(a))
	for i := range out {
		out[i] = -1
	}
	if len(a) == 0 {
		return out
	}
	if len(b) > 0 {
		out[0] = 0
	}
	type pair struct {
		i, j  int
		score float64
	}
	var ps []pair
	for i := 1; i < len(a); i++ {
		if a[i].n == 0 {
			continue
		}
		pace.Yield()
		for j := 1; j < len(b); j++ {
			if b[j].n == 0 {
				continue
			}
			same := a[i].heading != "" && a[i].heading == b[j].heading
			common := 0
			for k, n := range a[i].keys {
				common += min(n, b[j].keys[k])
			}
			small := min(len(a[i].keys), len(b[j].keys))
			if !same && (common == 0 || 2*common < small) {
				continue
			}
			s := float64(common)
			if same {
				s += 2
			}
			// alike scenes: the one in the same place
			s -= 0.5 * abs(float64(i)/float64(len(a))-float64(j)/float64(len(b)))
			ps = append(ps, pair{i, j, s})
		}
	}
	sort.SliceStable(ps, func(x, y int) bool { return ps[x].score > ps[y].score })
	used := map[int]bool{}
	for _, p := range ps {
		if out[p.i] < 0 && !used[p.j] {
			out[p.i], used[p.j] = p.j, true
		}
	}
	return out
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func invert(m []int, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = -1
	}
	for i, j := range m {
		if j >= 0 {
			out[j] = i
		}
	}
	return out
}

// lisSet are the values of seq in its longest increasing run (not
// necessarily side by side): what kept its order.
func lisSet(seq []int) map[int]bool {
	n := len(seq)
	prev := make([]int, n)
	var tails []int // indexes into seq
	for i, v := range seq {
		k := sort.Search(len(tails), func(x int) bool { return seq[tails[x]] >= v })
		if k > 0 {
			prev[i] = tails[k-1]
		} else {
			prev[i] = -1
		}
		if k == len(tails) {
			tails = append(tails, i)
		} else {
			tails[k] = i
		}
	}
	out := map[int]bool{}
	if len(tails) == 0 {
		return out
	}
	for i := tails[len(tails)-1]; i >= 0; i = prev[i] {
		out[seq[i]] = true
	}
	return out
}

// matches is each of a's elements' place in b, where the edit script
// keeps it (-1: not).
func matches(a, b []string) []int {
	out := make([]int, len(a))
	i, j := 0, 0
	for _, op := range diff(a, b) {
		switch op {
		case Same:
			out[i] = j
			i, j = i+1, j+1
		case Removed:
			out[i] = -1
			i++
		case Added:
			j++
		}
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameKeys(a, b []melem) bool {
	ka, _ := keysOf(a)
	kb, _ := keysOf(b)
	return equal(ka, kb)
}

func keysOf(es []melem) (keys []string, at []int) {
	for i, e := range es {
		if e.key != "" {
			keys, at = append(keys, e.key), append(at, i)
		}
	}
	return
}

// Taken counts the changes taken.
func Taken(take map[int]bool, edits map[int]string) int {
	n := len(edits)
	for id, ok := range take {
		if _, e := edits[id]; ok && !e {
			n++
		}
	}
	return n
}

// Resemblance is how much of theirs a version has as it is: how many of
// theirs' elements (as keyed) it has too, and how many of its own theirs
// has not. The base to merge from is the version most like theirs.
func Resemblance(version, theirs string, o MergeOptions) (same, other int) {
	key := o.Key
	if key == nil {
		key = func(s string) string { return s }
	}
	count := func(s string) map[string]int {
		if o.Body != nil {
			s = s[:o.Body(s)]
		}
		m := map[string]int{}
		for _, e := range Split(s, o.IsHeading) {
			if k := key(e.Text); k != "" {
				m[k]++
			}
		}
		return m
	}
	v, t := count(version), count(theirs)
	for k, n := range v {
		c := min(n, t[k])
		same += c
		other += n - c
	}
	return
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

// Package scriptdiff compares two versions of a script (or of a story in
// Markdown) element by element, as a writer reads the change: which
// scenes changed, and in them which elements were cut, added or changed,
// a changed one word by word. It is what Collaboration > Drafts… > Compare shows,
// what production's revision colours (PRODUCTION step 4) build on, and
// File > Merge Changes…'s three-way merge (merge.go).
//
// An element is a paragraph: the lines between blank lines (a scene
// heading, an action paragraph, a character's cue with their lines).
// Long work calls pace.Yield, for the web version's page.
package scriptdiff

import (
	"strings"
	"unicode"

	"github.com/LaPingvino/accolade/internal/pace"
)

// Kind is what happened to an element (or a word).
type Kind int

const (
	Same    Kind = iota // in both, the same
	Removed             // only in the old
	Added               // only in the new
	Changed             // in both, changed: Words say how
)

// Element is a paragraph of the script.
type Element struct {
	Text    string
	Heading bool // a scene heading (or a Markdown heading)
}

// Word is a run of a changed element's text: words, spaces, punctuation.
type Word struct {
	Text string
	Kind Kind // Same, Removed or Added
}

// Row is an element as compared.
type Row struct {
	Kind     Kind
	Old, New string // the old and the new text ("" where it has none)
	Words    []Word // a Changed row's words, the removed before the added
	Heading  bool
}

// Scene is a scene's rows: Heading is its heading (the new one if it
// changed; "" before the first scene).
type Scene struct {
	Heading string
	Rows    []Row
	Changed bool // any row not Same
}

// Split cuts a script into its elements; isHeading tells a scene
// heading's line (nil: none).
func Split(text string, isHeading func(string) bool) []Element {
	ps := SplitAt(text, isHeading)
	es := make([]Element, len(ps))
	for i, p := range ps {
		es[i] = p.Element
	}
	return es
}

// Placed is an element with where it is in the text, in bytes: Start its
// first line's, TextEnd its last line's end, End the next element's
// start (the text's end for the last).
type Placed struct {
	Element
	Start, TextEnd, End int
}

// SplitAt is Split with each element's place in the text.
func SplitAt(text string, isHeading func(string) bool) []Placed {
	var es []Placed
	var cur []string
	start, textEnd := 0, 0
	flush := func() {
		if len(cur) == 0 {
			return
		}
		t := strings.ReplaceAll(strings.Join(cur, "\n"), "\r", "")
		if n := len(es); n > 0 {
			es[n-1].End = start
		}
		es = append(es, Placed{Element: Element{Text: t, Heading: isHeading != nil && len(cur) == 1 && isHeading(cur[0])},
			Start: start, TextEnd: textEnd, End: len(text)})
		cur = cur[:0]
	}
	for at := 0; at <= len(text); {
		end := strings.IndexByte(text[at:], '\n')
		next := 0
		if end < 0 {
			end, next = len(text), len(text)+1
		} else {
			end += at
			next = end + 1
		}
		line := strings.TrimRightFunc(text[at:end], unicode.IsSpace)
		if line == "" {
			flush()
		} else {
			if len(cur) == 0 {
				start = at
			}
			cur = append(cur, line)
			textEnd = end
		}
		at = next
	}
	flush()
	return es
}

// Diff compares the old elements with the new.
func Diff(old, new []Element) []Row {
	var rows []Row
	for _, r := range diffIdx(old, new) {
		switch r.kind {
		case Same:
			rows = append(rows, Row{Kind: Same, Old: old[r.i].Text, New: new[r.j].Text, Heading: new[r.j].Heading})
		case Removed:
			rows = append(rows, Row{Kind: Removed, Old: old[r.i].Text, Heading: old[r.i].Heading})
		case Added:
			rows = append(rows, Row{Kind: Added, New: new[r.j].Text, Heading: new[r.j].Heading})
		case Changed:
			o, n := old[r.i].Text, new[r.j].Text
			rows = append(rows, Row{Kind: Changed, Old: o, New: n, Words: Words(o, n), Heading: new[r.j].Heading})
		}
	}
	return rows
}

// idxRow is a row by its elements' indexes: i the old's, j the new's.
type idxRow struct {
	kind Kind
	i, j int
}

// diffIdx is Diff by the elements' indexes.
func diffIdx(old, new []Element) []idxRow {
	ops := diff(texts(old), texts(new))
	var rows []idxRow
	i, j := 0, 0
	for k := 0; k < len(ops); {
		if ops[k] == Same {
			rows = append(rows, idxRow{Same, i, j})
			i, j, k = i+1, j+1, k+1
			continue
		}
		// a run of removed and added: paired where alike
		var del, ins []int
		for ; k < len(ops) && ops[k] != Same; k++ {
			if ops[k] == Removed {
				del = append(del, i)
				i++
			} else {
				ins = append(ins, j)
				j++
			}
		}
		rows = append(rows, pairUp(old, new, del, ins)...)
	}
	return rows
}

// pairUp makes a run of removed (del) and added (ins) elements rows: an
// added element alike to a removed one, in the run's order, is that one
// changed; between those, the removed come before the added.
func pairUp(old, new []Element, del, ins []int) []idxRow {
	type pair struct{ d, i int }
	var pairs []pair
	d := 0
	for i, n := range ins {
		for k := d; k < len(del) && k < d+4; k++ {
			if old[del[k]].Heading == new[n].Heading && alike(old[del[k]].Text, new[n].Text) {
				pairs = append(pairs, pair{k, i})
				d = k + 1
				break
			}
		}
	}
	var rows []idxRow
	removed := func(from, to int) {
		for _, x := range del[from:to] {
			rows = append(rows, idxRow{Removed, x, -1})
		}
	}
	added := func(from, to int) {
		for _, x := range ins[from:to] {
			rows = append(rows, idxRow{Added, -1, x})
		}
	}
	dp, ip := 0, 0
	for _, p := range pairs {
		removed(dp, p.d)
		added(ip, p.i)
		rows = append(rows, idxRow{Changed, del[p.d], ins[p.i]})
		dp, ip = p.d+1, p.i+1
	}
	removed(dp, len(del))
	added(ip, len(ins))
	return rows
}

// alike tells whether two elements share half their words or more.
func alike(a, b string) bool {
	wa, wb := words(a), words(b)
	if len(wa) == 0 || len(wb) == 0 {
		return false
	}
	same := 0
	for _, op := range diff(wa, wb) {
		if op == Same {
			same++
		}
	}
	return 2*same*2 >= len(wa)+len(wb) // 2·same / (|a|+|b|) ≥ ½
}

// Words compares two texts word by word: runs of them in order, each
// Same, Removed or Added.
func Words(old, new string) []Word {
	a, b := tokens(old), tokens(new)
	var ws []Word
	add := func(t string, k Kind) {
		if n := len(ws); n > 0 && ws[n-1].Kind == k {
			ws[n-1].Text += t
			return
		}
		ws = append(ws, Word{t, k})
	}
	i, j := 0, 0
	ops := diff(a, b)
	// removed before added within a change
	for k := 0; k < len(ops); {
		if ops[k] == Same {
			add(a[i], Same)
			i, j, k = i+1, j+1, k+1
			continue
		}
		var ins []string
		for ; k < len(ops) && ops[k] != Same; k++ {
			if ops[k] == Removed {
				add(a[i], Removed)
				i++
			} else {
				ins = append(ins, b[j])
				j++
			}
		}
		for _, t := range ins {
			add(t, Added)
		}
	}
	return ws
}

// Group puts rows in their scenes.
func Group(rows []Row) []Scene {
	scenes := []Scene{{}}
	for _, r := range rows {
		if r.Heading {
			h := r.New
			if r.Kind == Removed {
				h = r.Old
			}
			scenes = append(scenes, Scene{Heading: h})
		}
		s := &scenes[len(scenes)-1]
		s.Rows = append(s.Rows, r)
		s.Changed = s.Changed || r.Kind != Same
	}
	if len(scenes[0].Rows) == 0 {
		scenes = scenes[1:]
	}
	return scenes
}

// Count is how many elements were removed, added and changed.
func Count(rows []Row) (removed, added, changed int) {
	for _, r := range rows {
		switch r.Kind {
		case Removed:
			removed++
		case Added:
			added++
		case Changed:
			changed++
		}
	}
	return
}

func texts(es []Element) []string {
	s := make([]string, len(es))
	for i, e := range es {
		s[i] = e.Text
	}
	return s
}

// words are a text's words without their punctuation, for alike.
func words(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '’'
	})
}

// tokens are a text's words, the spaces between them and its other
// characters (punctuation) one by one.
func tokens(s string) []string {
	var ts []string
	class := func(r rune) int {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '’':
			return 1
		case unicode.IsSpace(r):
			return 2
		}
		return 3
	}
	start, prev := 0, 0
	for i, r := range s {
		c := class(r)
		if i > start && (c != prev || c == 3) {
			ts = append(ts, s[start:i])
			start = i
		}
		prev = c
	}
	if start < len(s) {
		ts = append(ts, s[start:])
	}
	return ts
}

// diff is the edit script from a to b, Myers' O(ND) algorithm, the common
// start and end taken off first: Same, Removed (a's next) or Added (b's
// next) in order.
func diff(a, b []string) []Kind {
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	ops := make([]Kind, 0, len(a)+len(b))
	for range pre {
		ops = append(ops, Same)
	}
	ops = append(ops, myers(a[pre:len(a)-suf], b[pre:len(b)-suf])...)
	for range suf {
		ops = append(ops, Same)
	}
	return ops
}

// maxEdits is as many edits as myers looks for: past it (two drafts with
// little in common) the trace would grow too big.
const maxEdits = 2000

func myers(a, b []string) []Kind {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		ops := make([]Kind, 0, n+m)
		for range n {
			ops = append(ops, Removed)
		}
		for range m {
			ops = append(ops, Added)
		}
		return ops
	}
	max := n + m
	off := max + 1
	v := make([]int, 2*max+3)
	var trace [][]int // v as each round found it, from -d-1 to d+1
	found := false
	for d := 0; d <= max && !found; d++ {
		pace.Yield()
		if d > maxEdits { // too unlike to follow: all of a out, all of b in
			return append(myers(a, nil), myers(nil, b)...)
		}
		trace = append(trace, append([]int(nil), v[off-d-1:off+d+2]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[off+k-1] < v[off+k+1]) {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[off+k] = x
			if x >= n && y >= m {
				found = true
				break
			}
		}
	}
	// back through the trace
	var rev []Kind
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		vd := trace[d]
		at := func(k int) int { return vd[k+d+1] }
		k := x - y
		var pk int
		if k == -d || (k != d && at(k-1) < at(k+1)) {
			pk = k + 1
		} else {
			pk = k - 1
		}
		px := at(pk)
		py := px - pk
		for x > px && y > py {
			rev = append(rev, Same)
			x--
			y--
		}
		if d > 0 {
			if x == px {
				rev = append(rev, Added)
			} else {
				rev = append(rev, Removed)
			}
		}
		x, y = px, py
	}
	ops := make([]Kind, len(rev))
	for i, op := range rev {
		ops[len(rev)-1-i] = op
	}
	return ops
}

// Lines is the edit script from the old lines to the new: Same, Removed
// (the old's next) or Added (the new's next) in order, as the elements
// are compared (bounded the same way). Production's revision marks
// follow a script's lines through its issues with it.
func Lines(old, new []string) []Kind { return diff(old, new) }

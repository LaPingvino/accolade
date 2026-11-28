// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"math/rand"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The tail blocks are found with a plain scan now; it finds what the
// regexp it replaced found, on texts made to trip it: doubled and
// escaped closes, empty blocks, white space after the close, the
// Markdown marks, blocks not at the end, multi-byte text.

// oldBlocks are oldAt's regexps by their marks (compiled once: compiling
// one for each of the 20 000 texts was most of the test's time).
var oldBlocks = map[[2]string]*regexp.Regexp{}

// oldAt is notesMarks.at as it was, with its regexp.
func oldAt(m notesMarks, text string) (int, string) {
	block := oldBlocks[[2]string{m.open, m.close}]
	if block == nil {
		block = regexp.MustCompile(`(?s)^` + regexp.QuoteMeta(m.open) + `\n(.*?)\n?` + regexp.QuoteMeta(m.close) + `\s*$`)
		oldBlocks[[2]string{m.open, m.close}] = block
	}
	i := strings.LastIndex(text, m.open+"\n")
	if i < 0 || (i > 0 && text[i-1] != '\n') {
		return -1, ""
	}
	found := block.FindStringSubmatch(text[i:])
	if found == nil || strings.Contains(found[1], m.close) {
		return -1, ""
	}
	for i > 0 && text[i-1] == '\n' {
		i--
	}
	return i, found[1]
}

// oldTailBlocks is tailBlocks as it was, in runes.
func oldTailBlocks(text string) (src, adapt tailBlock) {
	r := []rune(text)
	end := len(r)
	if i, _ := oldAt(fountainNotes, text); i >= 0 {
		end = len([]rune(text[:i]))
	}
	for {
		before := string(r[:end])
		found := false
		for _, b := range []struct {
			marks *notesMarks
			to    *tailBlock
		}{{&sourceMarks, &src}, {&adaptationMarks, &adapt}} {
			if b.to.ok {
				continue
			}
			if i, inside := oldAt(*b.marks, before); i >= 0 {
				s := len([]rune(before[:i]))
				*b.to = tailBlock{start: s, end: end, ok: true,
					inside: strings.ReplaceAll(inside, b.marks.escaped, b.marks.close)}
				end, found = s, true
				break
			}
		}
		if !found {
			return src, adapt
		}
	}
}

func tailTexts(n int) []string {
	r := rand.New(rand.NewSource(7))
	pieces := []string{"INT. HOUSE - DAY\n", "Rain.\n", "é ü 中\n", "\n", "\n\n", " ", "\t", "\f", "\r",
		"/* Notes\n", "/* Source\n", "/* Adaptation\n", "/* Notes", "*/", "*/\n", "* /", "**/", "*",
		"<!-- Notes\n", "-->", "--->", "-- >", "cut:\n", "  \"The rain\"\n", "x/* Notes\n"}
	var out []string
	for range n {
		var b strings.Builder
		for range r.Intn(14) {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}
		out = append(out, b.String())
	}
	out = append(out, "", "/* Notes\n*/", "/* Notes\n\n*/\n", "a\n\n/* Notes\nn\n*/  \n", "/* Notes\na*/b*/",
		"/* Source\nmd\n*/\n\n/* Adaptation\ncut:\n*/\n\n/* Notes\nn\n*/\n", "/* Adaptation\n*/\n/* Source\n*/")
	return out
}

func TestTailBlocksAsBefore(t *testing.T) {
	for _, text := range tailTexts(20000) {
		for _, m := range []notesMarks{fountainNotes, markdownNotes, sourceMarks, adaptationMarks} {
			oi, on := oldAt(m, text)
			ni, nn := m.at(text)
			if oi != ni || on != nn {
				t.Fatalf("%q in %q: was %d %q, now %d %q", m.open, text, oi, on, ni, nn)
			}
		}
		os, oa := oldTailBlocks(text)
		ns, na := tailBlocks(text)
		for _, p := range [][2]tailBlock{{os, ns}, {oa, na}} {
			o, n := p[0], p[1]
			if o.ok != n.ok || o.inside != n.inside || (o.ok && (o.start != n.start || o.end != n.end)) {
				t.Fatalf("tail blocks of %q: was %+v, now %+v", text, o, n)
			}
		}
		if want := len([]rune(text[:fountainNotes.startByte(text)])); notesStart(text) != want {
			t.Fatalf("notesStart(%q) = %d, want %d", text, notesStart(text), want)
		}
	}
}

// oldPrefixDistance is prefixDistance as it was: the whole of b.
func oldPrefixDistance(a, b []string, limit int) int {
	row := make([]int, len(b)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(a); i++ {
		prev := row[0]
		row[0] = i
		low := row[0]
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur := min(row[j]+1, row[j-1]+1, prev+cost)
			prev, row[j] = row[j], cur
			low = min(low, cur)
		}
		if low > limit {
			return limit + 1
		}
	}
	return min(slices.Min(row), limit+1)
}

// The bounded distance and the matcher find what matchLink found.
func TestMatchingAsBefore(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	vocab := strings.Fields("a b c d e f g h")
	words := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = vocab[r.Intn(len(vocab))]
		}
		return out
	}
	var row []int
	for range 50000 {
		a, b := words(1+r.Intn(10)), words(r.Intn(30))
		limit := 1 + r.Intn(3)
		if o, n := oldPrefixDistance(a, b, limit), prefixDistanceIn(a, b, limit, &row); o != n {
			t.Fatalf("%v to %v (%d): was %d, now %d", a, b, limit, o, n)
		}
	}
	var ps []passage
	for i := range 300 {
		ps = append(ps, passage{Index: i, Words: words(r.Intn(12))})
	}
	lm := newLinkMatcher(ps)
	for range 3000 {
		w := words(r.Intn(9))
		if o, n := matchLink(w, ps), lm.match(w); o.Kind != n.Kind || !slices.Equal(o.Passages, n.Passages) {
			t.Fatalf("%v: was %+v, now %+v", w, o, n)
		}
	}
}

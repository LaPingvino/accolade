// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package cite

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// RenderWith is Render in a CSL style (nil: Accolade's own): the
// citations numbered in reading order (a footnote's where its mark is),
// the bibliography in the style's order.
func RenderWith(body string, lib *Library, nocite []string, st *Style) string {
	if st == nil {
		return Render(body, lib, nocite)
	}
	if lib == nil {
		return body
	}
	lines := strings.Split(body, "\n")
	type occ struct {
		line  int
		c     syntax.Citation
		at    [4]int
		items []*item
	}
	var occs []*occ
	EachCitation(body, func(i int, c syntax.Citation) {
		occs = append(occs, &occ{line: i, c: c})
	})
	pos := readingOrder(lines)
	for _, o := range occs {
		o.at = pos(o.line, o.c.Start)
	}
	sort.SliceStable(occs, func(a, b int) bool {
		x, y := occs[a].at, occs[b].at
		for i := range x {
			if x[i] != y[i] {
				return x[i] < y[i]
			}
		}
		return false
	})
	base := map[*Entry]*item{}
	var works []*item
	add := func(e *Entry) *item {
		if e == nil {
			return nil
		}
		if it := base[e]; it != nil {
			return it
		}
		it := &item{e: e}
		base[e] = it
		works = append(works, it)
		return it
	}
	seen := map[*Entry]bool{}
	var prev *Entry
	for _, o := range occs {
		r := []rune(lines[o.line])
		parts := splitRefs(r, o.c)
		for _, p := range parts {
			e := lib.Get(p.key)
			if e == nil {
				o.items = append(o.items, &item{key: p.key, prefix: p.prefix, after: p.suffix})
				continue
			}
			add(e)
			it := &item{e: e, prefix: p.prefix, after: p.suffix, suppress: p.suppress, first: !seen[e]}
			it.label, it.locator, it.after = cslLocator(p.loc, p.suffix)
			it.ibid = len(parts) == 1 && prev == e
			seen[e] = true
			o.items = append(o.items, it)
		}
		prev = nil
		if len(parts) == 1 {
			prev = lib.Get(parts[0].key)
		}
	}
	for _, k := range nocite {
		if k == "*" {
			for _, e := range lib.Entries {
				add(e)
			}
			continue
		}
		add(lib.Get(k))
	}
	if len(occs) == 0 && len(works) == 0 {
		return body
	}
	st.order(works)
	st.disambiguate(works)
	for _, o := range occs {
		for _, it := range o.items {
			if b := base[it.e]; it.e != nil && b != nil {
				it.num, it.suffix = b.num, b.suffix
			}
		}
	}
	// from each line's end: the offsets before stay
	sort.SliceStable(occs, func(a, b int) bool {
		if occs[a].line != occs[b].line {
			return occs[a].line < occs[b].line
		}
		return occs[a].c.Start > occs[b].c.Start
	})
	for _, o := range occs {
		r := []rune(lines[o.line])
		out := st.cluster(o.items, o.c.InText)
		lines[o.line] = string(r[:o.c.Start]) + out + string(r[o.c.End:])
	}
	if len(works) == 0 || st.bib == nil {
		return strings.Join(lines, "\n")
	}
	var refs []string
	for _, w := range works {
		s, _ := st.render(w, st.bib, false)
		if st.hanging() { // (1. Doe J without the mark, as the Text preview has it, is a list of the same numbers)
			s = EntryMark + s
		}
		refs = append(refs, s)
	}
	return placeRefs(lines, strings.Join(refs, "\n\n"))
}

var noteDef = regexp.MustCompile(`^\[\^([^\]\s]+)\]:`)

// readingOrder is where a citation (line, rune) is read: a footnote
// definition's where its first mark is, else where it is written.
func readingOrder(lines []string) func(line, col int) [4]int {
	defOf := map[int]string{}
	cur := ""
	for i, l := range lines {
		if m := noteDef.FindStringSubmatch(l); m != nil {
			cur = m[1]
		} else if cur != "" && strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "    ") && !strings.HasPrefix(l, "\t") {
			cur = ""
		}
		if cur != "" {
			defOf[i] = cur
		}
	}
	mark := map[string][2]int{}
	for i, l := range lines {
		off := 0 // l's start in lines[i], in bytes
		if _, def := defOf[i]; def {
			if m := noteDef.FindStringIndex(l); m != nil {
				off = m[1]
			}
		}
		for {
			j := strings.Index(l[off:], "[^")
			if j < 0 {
				break
			}
			j += off
			k := strings.IndexByte(l[j:], ']')
			if k < 0 {
				break
			}
			id := l[j+2 : j+k]
			if _, ok := mark[id]; !ok {
				mark[id] = [2]int{i, utf8.RuneCountInString(l[:j])}
			}
			off = j + k
		}
	}
	return func(line, col int) [4]int {
		if id, ok := defOf[line]; ok {
			if m, ok := mark[id]; ok {
				return [4]int{m[0], m[1], line, col}
			}
		}
		return [4]int{line, col, 0, 0}
	}
}

// locatorLabels are the locator labels written before a locator, as
// CSL's terms.
var locatorLabels = []struct{ pre, term string }{
	{"pp.", "page"}, {"p.", "page"}, {"pages", "page"}, {"page", "page"}, {"pg.", "page"},
	{"chaps.", "chapter"}, {"chap.", "chapter"}, {"chapter", "chapter"}, {"ch.", "chapter"},
	{"secs.", "section"}, {"sec.", "section"}, {"section", "section"}, {"§§", "section"}, {"§", "section"},
	{"paras.", "paragraph"}, {"para.", "paragraph"}, {"paragraph", "paragraph"}, {"¶¶", "paragraph"}, {"¶", "paragraph"},
	{"figs.", "figure"}, {"fig.", "figure"}, {"figure", "figure"}, {"vols.", "volume"}, {"vol.", "volume"},
	{"nn.", "note"}, {"n.", "note"}, {"note", "note"}, {"ll.", "line"}, {"l.", "line"}, {"line", "line"},
	{"vv.", "verse"}, {"v.", "verse"}, {"verse", "verse"}, {"cols.", "column"}, {"col.", "column"},
	{"bk.", "book"}, {"book", "book"}, {"no.", "number"}, {"nos.", "number"}, {"pts.", "part"}, {"pt.", "part"},
	{"s.v.", "sub-verbo"}, {"s.vv.", "sub-verbo"}, {"op.", "opus"}, {"fols.", "folio"}, {"fol.", "folio"},
	{"ts.", "timestamp"}, {"t.", "timestamp"},
}

// cslLocator reads a locator as written (p. 12, chap. 2, 12–15, ",
// emphasis added" after it): its label (page when it has none), its
// value, what follows (suffix when there is no locator).
func cslLocator(loc, suffix string) (label, value, after string) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "", "", suffix
	}
	if i := strings.Index(loc, ", "); i > 0 && i+2 < len(loc) && (loc[i+2] < '0' || loc[i+2] > '9') {
		loc, after = loc[:i], loc[i:]
	}
	label = "page"
	low := strings.ToLower(loc)
	for _, l := range locatorLabels {
		if strings.HasPrefix(low, l.pre) && (len(low) == len(l.pre) || strings.ContainsRune(" .0123456789", rune(low[len(l.pre)])) || strings.HasSuffix(l.pre, ".")) {
			label, loc = l.term, strings.TrimSpace(loc[len(l.pre):])
			break
		}
	}
	return label, dashes(loc), after
}

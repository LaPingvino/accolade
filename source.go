package main

import (
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/prose"
)

// A screenplay's source (docs/ADAPTATION.md, step 3): the story it is
// adapted from, in Markdown. Where it is kept: source_file.go.
//
// The source is read as passages: its paragraphs, headings, quotes, list
// items, each one thing to adapt. The script links a scene to a passage
// with an ordinary Fountain note,
//
//	[[from: "She opened the door to the…"]]
//
// holding the passage's first words. A link is found again by its text,
// not by a number or a line, so the draft can change around it.
//
// Statuses (step 4) attach to the same things: a passage is known by its
// words (passage.Words), and what the script does with it is the links
// that resolve to it (sourceModel.linked); "adapted" is linked, "cut on
// purpose" a mark keyed by the same first words.

// passage is a piece of the source to adapt.
type passage struct {
	Index    int
	Kind     prose.Kind // Para, Heading, Quote, Item, Code, Table, Rule
	Level    int        // a heading's
	Markdown string     // as shown
	Plain    string     // its text, without formatting
	Words    []string   // Plain's words, normalised (normWords)
}

// linkable: a break (* * *) is no passage to adapt.
func (p passage) linkable() bool { return p.Kind != prose.Rule && len(p.Words) > 0 }

// passagesOf the source's Markdown (its front matter and notes left out).
func passagesOf(src string) []passage {
	doc := prose.FromMarkdown(markdownBody(src))
	var out []passage
	add := func(kind prose.Kind, level int, shown, text []*prose.Block) {
		md := strings.TrimSpace((&prose.Doc{Blocks: shown}).Markdown())
		plain := strings.TrimSpace((&prose.Doc{Blocks: text}).Plain())
		if kind == prose.Rule {
			md, plain = "* * *", ""
		}
		out = append(out, passage{Index: len(out), Kind: kind, Level: level, Markdown: md, Plain: plain, Words: normWords(plain)})
	}
	for _, b := range doc.Blocks {
		switch b.Kind {
		case prose.List: // each item is a passage
			for i, it := range b.Children {
				one := *b
				one.Children = []*prose.Block{it}
				one.Start = max(b.Start, 1) + i
				add(prose.Item, 0, []*prose.Block{&one}, it.Children)
			}
		case prose.Quote:
			add(b.Kind, 0, []*prose.Block{b}, b.Children)
		default:
			add(b.Kind, b.Level, []*prose.Block{b}, []*prose.Block{b})
		}
	}
	return out
}

// normWords is text's words as links compare them: in lower case, with
// only letters and digits ("Don’t" is "dont", the quotes and the
// punctuation go; a dash or an ellipsis parts words).
func normWords(text string) []string {
	var words []string
	var w strings.Builder
	flush := func() {
		if w.Len() > 0 {
			words = append(words, w.String())
			w.Reset()
		}
	}
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r):
			w.WriteRune(unicode.ToLower(r))
		case strings.ContainsRune(`'’‘‛ʼ"“”„«»‹›`, r), r == '*' || r == '_' || r == '`':
			// inside a word: part of it (don't); quotes and emphasis marks
		default:
			flush()
		}
	}
	flush()
	return words
}

// sourceLink is a [[from: …]] note in the script, or one directive of
// it (a note with several text= is as many links, at the same place).
type sourceLink struct {
	Start, End int // where the note is in the script (runes)
	Line       int
	Quote      string   // the words, as the pane quotes them
	Words      []string // its first words, normalised
	Ref        linkRef  // what it names
	Key        string   // Ref's rawKey (the new form's; key() for both)
}

// linkWords is the number of words a new link quotes, at least: enough
// to find a passage again, short enough to read.
const linkWords = 8

// parseLinks finds the source links in a script: not in a boneyard
// (/* … */) or the notes block at its end.
func parseLinks(text string) []sourceLink {
	_, _, notes := tailsOf(text)
	text = text[:notes]
	var links []sourceLink
	var refs []linkRef // room for each note's
	off := 0 // in runes, of the line
	inBoneyard := false
	for i, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
		} else {
			for at := 0; ; {
				s := strings.Index(line[at:], "[[")
				if s < 0 {
					break
				}
				s += at
				e := strings.Index(line[s:], "]]")
				if e < 0 {
					break
				}
				e += s + 2
				if refs = appendLinkRefs(refs[:0], line[s+2:e-2]); len(refs) > 0 {
					start := off + utf8.RuneCountInString(line[:s])
					for _, r := range refs {
						l := sourceLink{Start: start, End: start + utf8.RuneCountInString(line[s:e]), Line: i,
							Quote: r.shown(), Words: r.words(), Ref: r}
						if !r.Old {
							l.Key = r.rawKey()
						}
						links = append(links, l)
					}
				}
				at = e
			}
		}
		off += utf8.RuneCountInString(line) + 1
	}
	return links
}

// appendLinkRefs is dst with what a note's inside links to if it is a
// link: the new form's directives (textfrag.go), or the old form's quote.
func appendLinkRefs(dst []linkRef, inside string) []linkRef {
	t := strings.TrimSpace(inside)
	if len(t) < 5 || !strings.EqualFold(t[:5], "from:") {
		return dst
	}
	if rest := t[5:]; isFragmentRef(rest) {
		return append(dst, fragmentRefs(rest)...)
	}
	if q, ok := linkQuote(inside); ok {
		return append(dst, linkRef{Old: true, Quote: q})
	}
	return dst
}

// linkQuote is the quoted words of a note's inside if it is a link,
// `from: "…"`: from its first quote to its last (a passage may open with
// dialogue, quotes of its own), with a closing ellipsis left out.
func linkQuote(inside string) (string, bool) {
	t := strings.TrimSpace(inside)
	if len(t) < 5 || !strings.EqualFold(t[:5], "from:") {
		return "", false
	}
	t = strings.TrimSpace(t[5:])
	const quotes = `"“”„«»'‘’`
	if r := []rune(t); len(r) >= 2 && strings.ContainsRune(quotes, r[0]) && strings.ContainsRune(quotes, r[len(r)-1]) {
		t = string(r[1 : len(r)-1])
	}
	t = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(t), ".…"))
	return t, t != ""
}

// matchKind is how a link was found in the source.
type matchKind int

const (
	linkFound     matchKind = iota // its words open exactly one passage
	linkNear                       // one passage opens with nearly its words
	linkAmbiguous                  // more than one passage fits as well
	linkMissing                    // none does
)

// linkMatch is a link found in the source: its passages, from
// Passages[0] to Last, or for an ambiguous link those that fit.
type linkMatch struct {
	Kind     matchKind
	Passages []int
	Last     int // found: the last passage of its range
}

// found reports whether the link has its one passage (or range).
func (m linkMatch) found() bool { return m.Kind == linkFound || m.Kind == linkNear }

// covers reports whether the link found passage p (in its range).
func (m linkMatch) covers(p int) bool { return m.found() && p >= m.Passages[0] && p <= m.Last }

// last is the last passage the link found (its only one if not a range).
func (m linkMatch) last() int { return max(m.Last, m.Passages[0]) }

// matchLink finds a link's passage. The rules:
//
//  1. The words are compared normalised (normWords: case, quotes,
//     punctuation and spacing do not count).
//  2. Exact: the passages that open with all the link's words. One is
//     the link's; more are ambiguous.
//  3. Near, only when none is exact and the link has at least 4 words:
//     the passage whose opening is fewest words away (a word added,
//     dropped or changed counts one), if that is at most a quarter of
//     the link's words (at least 1), and no other passage is as near.
//     A tie is ambiguous; nothing near enough is missing.
func matchLink(words []string, ps []passage) linkMatch {
	return matchIn(words, ps, nil, nil)
}

// matchIn is matchLink; with byFirst (the linkable passages by their
// first word) the exact ones are looked for only among those opening
// with the link's first word, and row is room for prefixDistance.
func matchIn(words []string, ps []passage, byFirst map[string][]int, row *[]int) linkMatch {
	if len(words) == 0 {
		return linkMatch{Kind: linkMissing}
	}
	var exact []int
	opens := func(p passage) bool {
		return p.linkable() && len(p.Words) >= len(words) && slices.Equal(p.Words[:len(words)], words)
	}
	if byFirst != nil {
		for _, i := range byFirst[words[0]] {
			if opens(ps[i]) {
				exact = append(exact, ps[i].Index)
			}
		}
	} else {
		for _, p := range ps {
			if opens(p) {
				exact = append(exact, p.Index)
			}
		}
	}
	switch {
	case len(exact) == 1:
		return linkMatch{Kind: linkFound, Passages: exact, Last: exact[0]}
	case len(exact) > 1:
		return linkMatch{Kind: linkAmbiguous, Passages: exact}
	case len(words) < 4:
		return linkMatch{Kind: linkMissing}
	}
	limit := max(1, len(words)/4)
	best, bestAt := limit+1, []int(nil)
	var scratch []int
	if row == nil {
		row = &scratch
	}
	for _, p := range ps { // a near one may differ in any word, the first too: all of them
		if !p.linkable() {
			continue
		}
		d := prefixDistanceIn(words, p.Words, limit, row)
		switch {
		case d > limit:
		case d < best:
			best, bestAt = d, []int{p.Index}
		case d == best:
			bestAt = append(bestAt, p.Index)
		}
	}
	switch {
	case len(bestAt) == 1:
		return linkMatch{Kind: linkNear, Passages: bestAt, Last: bestAt[0]}
	case len(bestAt) > 1:
		return linkMatch{Kind: linkAmbiguous, Passages: bestAt}
	}
	return linkMatch{Kind: linkMissing}
}

// linkMatcher matches links against one source's passages, and
// remembers what each link's words found: while the passages stay the
// same, a keystroke matches again only the link it changed (a near or
// missing link is compared with every passage, which is what took the
// time).
type linkMatcher struct {
	ps      []passage
	byFirst map[string][]int // the linkable passages, by their first word
	found   map[string]linkMatch
	row     []int
	mu      sync.Mutex // found and row
	fi      *fragIndex // the new form's (textfrag.go), when first needed
	fiOnce  sync.Once
}

// maxRemembered is how many links' matches a matcher keeps; past it, it
// starts again (typing a link makes a new one per letter).
const maxRemembered = 4096

func newLinkMatcher(ps []passage) *linkMatcher {
	lm := &linkMatcher{ps: ps, byFirst: map[string][]int{}, found: map[string]linkMatch{}}
	for i, p := range ps {
		if p.linkable() {
			lm.byFirst[p.Words[0]] = append(lm.byFirst[p.Words[0]], i)
		}
	}
	return lm
}

// match is matchLink(words, the matcher's passages). The match it gives
// is shared: not to be changed.
func (lm *linkMatcher) match(words []string) linkMatch {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	key := strings.Join(words, "\x00") // (the new form's keys have \x01 in them: textfrag.go)
	if mt, ok := lm.found[key]; ok {
		return mt
	}
	mt := matchIn(words, lm.ps, lm.byFirst, &lm.row)
	if len(lm.found) >= maxRemembered {
		clear(lm.found)
	}
	lm.found[key] = mt
	return mt
}

// prefixDistance is how many words a, edited, is from the nearest
// opening of b (b's first len(a) words, give or take): Levenshtein over
// words. Past limit it gives up and says limit+1.
func prefixDistance(a, b []string, limit int) int {
	return prefixDistanceIn(a, b, limit, nil)
}

// prefixDistanceIn is prefixDistance with row as room for its row. Only
// b's first len(a)+limit words can count: an opening longer than that
// is more than limit away.
func prefixDistanceIn(a, b []string, limit int, room *[]int) int {
	if len(b) > len(a)+limit {
		b = b[:len(a)+limit]
	}
	// row[j]: the distance from a[:i] to b[:j]
	var row []int
	if room != nil && cap(*room) >= len(b)+1 {
		row = (*room)[:len(b)+1]
	} else {
		row = make([]int, len(b)+1)
		if room != nil {
			*room = row
		}
	}
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
	// an opening of b: any length, the rest of b does not count
	return min(slices.Min(row), limit+1)
}

// curlyQuotes are a quote's straight double quotes made curly, opening
// or closing by what is before them, so the quote can go between
// straight ones.
func curlyQuotes(quote string) string {
	var b strings.Builder
	prev := ' '
	for _, r := range quote {
		if r == '"' {
			if unicode.IsSpace(prev) || strings.ContainsRune("([—–-", prev) {
				r = '“'
			} else {
				r = '”'
			}
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

// sceneSpan is a stretch of the script: a scene or section, from its
// heading to the next (lines [Line, End)); the first, from line 0, is
// what comes before any heading.
type sceneSpan struct {
	Line, End int
	Heading   string
}

// sceneSpans parts the script at its scene headings and sections.
func sceneSpans(text string, starts []string) []sceneSpan {
	lines := strings.Count(text, "\n") + 1
	spans := []sceneSpan{{Line: 0, End: lines}}
	for _, it := range outlineOf(text, starts) {
		last := &spans[len(spans)-1]
		if it.Line == last.Line { // the script opens with a heading
			last.Heading = it.Text
			continue
		}
		last.End = it.Line
		spans = append(spans, sceneSpan{Line: it.Line, End: lines, Heading: it.Text})
	}
	return spans
}

// spanAt is the span a line is in.
func spanAt(spans []sceneSpan, line int) sceneSpan {
	for _, s := range spans {
		if line >= s.Line && line < s.End {
			return s
		}
	}
	return spans[len(spans)-1]
}

// sourceModel is a script read against its source: the passages, the
// links and what each found.
type sourceModel struct {
	passages []passage
	links    []sourceLink
	matches  []linkMatch // the links'
	spans    []sceneSpan
	cuts     []cutMark       // the adaptation block's cut: entries
	cutAt    []cutSpan       // the passages each names (first -1: none)
	status   []passageStatus // the passages' (adaptation.go)
}

func newSourceModel(ps []passage, script string, starts []string) *sourceModel {
	return newSourceModelWith(newLinkMatcher(ps), script, starts)
}

// newSourceModelWith is newSourceModel with lm's passages, the links
// matched through it (what it found for earlier models is kept).
func newSourceModelWith(lm *linkMatcher, script string, starts []string) *sourceModel {
	ps := lm.ps
	m := &sourceModel{passages: ps, links: parseLinks(script), spans: sceneSpans(script, starts)}
	for _, l := range m.links {
		if l.Ref.Old { // its words read already
			m.matches = append(m.matches, lm.match(l.Words))
		} else {
			m.matches = append(m.matches, lm.matchRef(l.Ref))
		}
	}
	if inside, ok := adaptationBlock(script); ok {
		m.cuts = parseCuts(inside)
	}
	for _, c := range m.cuts {
		if r := c.ref(); r.Old {
			m.cutAt = append(m.cutAt, cutAt(c, lm.match(c.Words)))
		} else {
			m.cutAt = append(m.cutAt, cutAt(c, lm.matchRef(r)))
		}
	}
	m.status = statusesOf(ps, m.matches, m.cutAt)
	return m
}

// inSpan are the links (their indexes) in the span a line is in.
func (m *sourceModel) inSpan(line int) []int {
	s := spanAt(m.spans, line)
	var out []int
	for i, l := range m.links {
		if l.Line >= s.Line && l.Line < s.End {
			out = append(out, i)
		}
	}
	return out
}

// linked are the spans (their first lines, in order) with a link that
// found passage p.
func (m *sourceModel) linked(p int) []int {
	var lines []int
	for i, l := range m.links {
		if m.matches[i].covers(p) {
			if s := spanAt(m.spans, l.Line).Line; !slices.Contains(lines, s) {
				lines = append(lines, s)
			}
		}
	}
	slices.Sort(lines)
	return lines
}

// count is how many links matched so.
func (m *sourceModel) count(kind matchKind) int {
	n := 0
	for _, mt := range m.matches {
		if mt.Kind == kind {
			n++
		}
	}
	return n
}

// insertLink is the script with the note put in at the cursor (runes):
// on an empty line, the note is that line (a blank line kept around it);
// at the end of a line of text, after it on the same line (an inline
// note, which keeps dialogue together); after a heading, section or
// transition, as a line of its own below it. It returns the edit: where,
// what goes there, and the cursor after.
func insertLink(text string, cursor int, note string, starts []string) (at int, insert string, after int) {
	r := []rune(text)
	start, end := lineBounds(r, cursor)
	line := strings.TrimSpace(string(r[start:end]))
	blank := func(i int) bool { // is line-bounded rune offset i on a blank line (or out)?
		if i < 0 || i > len(r) {
			return true
		}
		s, e := lineBounds(r, i)
		return strings.TrimSpace(string(r[s:e])) == ""
	}
	switch {
	case line == "":
		insert = note
		if start > 0 && !blank(start-1) {
			insert = "\n" + insert
		}
		if end < len(r) && !blank(end+1) {
			insert += "\n"
		}
		return start, insert, start + len([]rune(strings.TrimRight(insert, "\n")))
	case strings.HasPrefix(line, "#") || strings.HasPrefix(line, ".") || isSceneHeadingIn(line, starts) ||
		strings.HasSuffix(line, "TO:") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "="):
		insert = "\n\n" + note
		if end < len(r) && !blank(end+1) {
			insert += "\n"
		}
		return end, insert, end + len([]rune(strings.TrimRight(insert, "\n")))
	}
	insert = " " + note
	return end, insert, end + len([]rune(insert))
}

// sourceHeading is a heading of the source.
type sourceHeading struct {
	Level int
	Text  string
}

// sourceHeadings are the source's headings, as passages.
func sourceHeadings(ps []passage) []sourceHeading {
	var out []sourceHeading
	for _, p := range ps {
		if p.Kind == prose.Heading && p.Plain != "" {
			out = append(out, sourceHeading{Level: max(p.Level, 1), Text: p.Plain})
		}
	}
	return out
}

// sectionEdit is a piece of text to put in the script, at a rune offset.
type sectionEdit struct {
	At   int
	Text string
}

// sectionsFromSource carries the source's headings over to the script as
// Fountain sections (# Part, ## Chapter 3), at the same levels, without
// those already there (the same words, at any level; each section in the
// script stands for one heading). If none is there yet they go in as an
// outline, at the line the cursor is on; otherwise each goes in its
// place among those already there: after the one before it in the
// source, before the next (the end of the script, before its notes, if
// none follows, before its notes and other blocks). The edits are in order, last first, to make one by one.
func sectionsFromSource(text string, starts []string, hs []sourceHeading, cursor int) []sectionEdit {
	var sections []outlineItem
	for _, it := range outlineOf(text, starts) {
		if !it.Scene {
			sections = append(sections, it)
		}
	}
	// where each heading is in the script (-1: not there)
	at := make([]int, len(hs))
	used := map[int]bool{} // sections taken
	for i, h := range hs {
		at[i] = -1
		key := strings.Join(normWords(h.Text), " ")
		for j, s := range sections {
			if !used[j] && strings.Join(normWords(s.Text), " ") == key {
				used[j], at[i] = true, s.Line
				break
			}
		}
	}
	r := []rune(text)
	lineStart := func(line int) int {
		off := 0
		for i := 0; i < line && off < len(r); off++ {
			if r[off] == '\n' {
				i++
			}
		}
		return off
	}
	section := func(h sourceHeading) string { return strings.Repeat("#", h.Level) + " " + h.Text }
	var edits []sectionEdit
	if !slices.ContainsFunc(at, func(l int) bool { return l >= 0 }) {
		if len(hs) == 0 {
			return nil
		}
		s, _ := lineBounds(r, min(cursor, len(r)))
		var b strings.Builder
		for _, h := range hs {
			b.WriteString(section(h) + "\n\n")
		}
		return []sectionEdit{{At: s, Text: b.String()}}
	}
	end := len([]rune(strings.TrimRight(string(r[:bodyEnd(text)]), "\n")))
	for i := 0; i < len(hs); {
		if at[i] >= 0 {
			i++
			continue
		}
		j := i // the missing run [i, j)
		for j < len(hs) && at[j] < 0 {
			j++
		}
		var b strings.Builder
		if j < len(hs) { // before the next one there
			for _, h := range hs[i:j] {
				b.WriteString(section(h) + "\n\n")
			}
			edits = append(edits, sectionEdit{At: lineStart(at[j]), Text: b.String()})
		} else { // at the end
			for _, h := range hs[i:j] {
				b.WriteString("\n\n" + section(h))
			}
			edits = append(edits, sectionEdit{At: end, Text: b.String()})
		}
		i = j
	}
	slices.SortStableFunc(edits, func(a, b sectionEdit) int { return b.At - a.At })
	return edits
}

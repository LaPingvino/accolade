// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// Source links in the browsers' Text Fragments syntax (the WICG's URL
// Fragment Text Directives, https://wicg.github.io/scroll-to-text-fragment/,
// supported by Chrome, Safari and Firefox):
//
//	[[from: #:~:text=She opened the door,stepped out]]
//	[[from: story.md#:~:text=said Anna.-,Yes.]]
//
// After "from:" an optional file, then the fragment directive: one or
// more text= directives joined by &, each
//
//	text=[prefix-,]textStart[,textEnd][,-suffix]
//
// textStart (and textEnd) is the passage range the link stands for: from
// the passage textStart is in to the one textEnd is in, the first after
// it; prefix- and -suffix are the words just before and after, there
// only to tell passages that read alike apart (the old form's "(2)").
//
// Written readable: only what the grammar needs is percent-encoded (the
// comma, a dash at a term's start or end, the ampersand, the percent
// sign; and the brackets, so the note's ]] stays the note's), spaces
// stay spaces, a dash inside a term too ("timber-yard"). Read leniently:
// encoded or plain (an old link's %2D inside a word as well), spaces
// around the parts aside, a dash inside a term taken as text (the spec
// rejects it; a term may still not begin or end with one, that is the
// syntax's). The writer never reads the encoding: the editor draws a
// link as a chip, its quote decoded (link_chips.go), and shows the
// encoded form only while the cursor is inside the link.
//
// Matching (findDir) is the spec's on the passages, which are the
// blocks: each term matched within one passage, as whole words
// (normWords: case, quotes and punctuation aside); a prefix ends right
// before textStart (in the same passage, or at the end of the one
// before), a suffix begins right after the match (or at the start of the
// next passage); textEnd is the first one after textStart (followed by
// the suffix, if there is one). Unlike a browser, which takes the first
// match, a link that fits more than one passage is ambiguous, and near
// matches are kept as for the old form (a few words edited).
//
// The old form, [[from: "first words…"]], is read as before (the words a
// passage opens with); Adaptation > Update Source Links writes the new one.

// textDirective is one text= directive, its parts decoded ("" for none;
// Start is always there).
type textDirective struct {
	Prefix, Start, End, Suffix string
}

// linkRef is what a link (or a cut: or skipped links: entry) names: an
// old form's quote, or a file and a text directive (a note with several
// directives is several links).
type linkRef struct {
	Old   bool
	Quote string // the old form's words, as written
	File  string // the new form's file ("" : the script's source)
	Dir   textDirective
	Bad   string // why the fragment could not be read ("": it could)
}

// key is the link's identity, for remembering what it found: its form
// and its words, normalised (the file aside: one source for now).
func (r linkRef) key() string {
	if r.Bad != "" {
		return "!\x00" + r.Bad
	}
	if r.Old {
		return "o\x00" + strings.Join(normWords(r.Quote), " ")
	}
	d := r.Dir
	return "t\x00" + wordsKey(d.Prefix) + "\x01" + wordsKey(d.Start) + "\x01" + wordsKey(d.End) + "\x01" + wordsKey(d.Suffix)
}

// rawKey is key without normalising: what a link's match is remembered
// by (cheaper per keystroke; the same words written otherwise are only
// matched once more).
func (r linkRef) rawKey() string {
	switch {
	case r.Bad != "":
		return "!\x00" + r.Bad
	case r.Old:
		return strings.Join(normWords(r.Quote), "\x00")
	}
	d := r.Dir
	return "\x01t\x00" + d.Prefix + "\x01" + d.Start + "\x01" + d.End + "\x01" + d.Suffix
}

// words are the link's first words, normalised: the old form's quote,
// the new's textStart.
func (r linkRef) words() []string {
	if r.Old {
		return normWords(r.Quote)
	}
	return normWords(r.Dir.Start)
}

// shown is the link's words as the pane quotes them.
func (r linkRef) shown() string {
	switch {
	case r.Old:
		return r.Quote
	case r.Bad != "":
		return "#:~:" + r.Bad
	case r.Dir.End != "":
		return r.Dir.Start + " … " + r.Dir.End
	}
	return r.Dir.Start
}

// String is the link as Accolade writes it after "from: ", readable;
// the old form as it was.
func (r linkRef) String() string {
	if r.Old {
		return `"` + curlyQuotes(strings.ReplaceAll(r.Quote, "\n", " ")) + `"`
	}
	return r.File + "#:~:" + r.Dir.String(false)
}

// String is the directive, text=…; for an address (href) fully
// percent-encoded, else readable.
func (d textDirective) String(href bool) string {
	enc := encodeTerm
	if href {
		enc = encodeTermHref
	}
	var b strings.Builder
	b.WriteString("text=")
	if d.Prefix != "" {
		b.WriteString(enc(d.Prefix) + "-,")
	}
	b.WriteString(enc(d.Start))
	if d.End != "" {
		b.WriteString("," + enc(d.End))
	}
	if d.Suffix != "" {
		b.WriteString(",-" + enc(d.Suffix))
	}
	return b.String()
}

// fragmentString is directives joined into one fragment directive, after
// the file: file#:~:text=…&text=….
func fragmentString(file string, dirs []textDirective, href bool) string {
	parts := make([]string, len(dirs))
	for i, d := range dirs {
		parts[i] = d.String(href)
	}
	return file + "#:~:" + strings.Join(parts, "&")
}

// encodeTerm is a term as the script shows it: only what the grammar
// needs encoded (, & and %; a dash only at the term's edges, where it
// would read as the prefix's or suffix's mark: "timber-yard" stays as it
// is), and [ ] so a note's brackets stay its own; a line break is a space.
func encodeTerm(s string) string {
	first, last := -1, -1 // the term's first and last letters, spaces aside
	for i, r := range s {
		if r != ' ' && r != '\n' && r != '\r' && r != '\t' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case r == '-' && i != first && i != last:
			b.WriteRune(r)
		case strings.ContainsRune("%,-&[]", r):
			b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(r), 16)))
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// encodeTermHref is a term for an address: all but the grammar's
// explicit characters percent-encoded, as UTF-8.
func encodeTermHref(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			strings.IndexByte("!$'()*+./:;=?@_~", c) >= 0:
			b.WriteByte(c)
		case c == '\n' || c == '\r' || c == '\t':
			b.WriteString("%20")
		default:
			b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(c)|0x100, 16)[1:]))
		}
	}
	return b.String()
}

// decodeTerm percent-decodes a term (as UTF-8); a % not followed by two
// hex digits stays as it is, and so does all of it if the bytes are no
// UTF-8.
func decodeTerm(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	hex := func(c byte) (byte, bool) {
		switch {
		case '0' <= c && c <= '9':
			return c - '0', true
		case 'a' <= c && c <= 'f':
			return c - 'a' + 10, true
		case 'A' <= c && c <= 'F':
			return c - 'A' + 10, true
		}
		return 0, false
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) {
			if h, ok := hex(s[i+1]); ok {
				if l, ok := hex(s[i+2]); ok {
					out = append(out, h<<4|l)
					i += 2
					continue
				}
			}
		}
		out = append(out, s[i])
	}
	if !utf8.Valid(out) {
		return s
	}
	return string(out)
}

// parseFragment reads what follows "from:" in a new link: the file
// before "#:~:", then the text= directives after it. A directive the
// grammar does not allow gives bad (and is left out); other directives
// than text= are glued to the text= before them, as an & in its text
// written plain (a browser ignores them).
func parseFragment(s string) (file string, dirs []textDirective, bad string) {
	i := strings.Index(s, "#:~:")
	if i < 0 {
		return "", nil, "no #:~: in it"
	}
	file = strings.TrimSpace(s[:i])
	var values []string
	for _, piece := range strings.Split(s[i+4:], "&") {
		t := strings.TrimLeft(piece, " ")
		switch {
		case strings.HasPrefix(t, "text="):
			values = append(values, t[5:])
		case len(values) > 0 && !otherDirective.MatchString(t):
			values[len(values)-1] += "&" + piece
		}
	}
	for _, v := range values {
		d, why := parseTextDirective(v)
		if why != "" {
			bad = why
			continue
		}
		dirs = append(dirs, d)
	}
	if len(dirs) == 0 && bad == "" {
		bad = "no text= in it"
	}
	return file, dirs, bad
}

// otherDirective is a directive other than text= (a browser ignores
// it): name=…; anything else after an & is the text's.
var otherDirective = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*=`)

// parseTextDirective is the spec's "parse a text directive", leniently
// (a dash inside a term is text; spaces around a part do not count): ""
// for why, or why not.
func parseTextDirective(v string) (d textDirective, why string) {
	tokens := strings.Split(v, ",")
	for i := range tokens {
		tokens[i] = strings.TrimSpace(tokens[i])
	}
	if len(tokens) > 4 {
		return d, "more than four parts in text=" + v
	}
	term := func(t string) bool { return t != "" && !strings.HasPrefix(t, "-") && !strings.HasSuffix(t, "-") }
	if strings.HasSuffix(tokens[0], "-") {
		d.Prefix = strings.TrimSpace(strings.TrimSuffix(tokens[0], "-"))
		tokens = tokens[1:]
		if !term(d.Prefix) || len(tokens) == 0 {
			return textDirective{}, "a prefix and no text in text=" + v
		}
	}
	if last := tokens[len(tokens)-1]; strings.HasPrefix(last, "-") {
		d.Suffix = strings.TrimSpace(strings.TrimPrefix(last, "-"))
		tokens = tokens[:len(tokens)-1]
		if !term(d.Suffix) || len(tokens) == 0 {
			return textDirective{}, "a suffix and no text in text=" + v
		}
	}
	if len(tokens) > 2 {
		return textDirective{}, "more than a start and an end in text=" + v
	}
	if !term(tokens[0]) {
		return textDirective{}, "no text to find in text=" + v
	}
	d.Start = tokens[0]
	if len(tokens) == 2 {
		if !term(tokens[1]) {
			return textDirective{}, "an empty end in text=" + v
		}
		d.End = tokens[1]
	}
	d.Prefix, d.Start, d.End, d.Suffix = decodeTerm(d.Prefix), decodeTerm(d.Start), decodeTerm(d.End), decodeTerm(d.Suffix)
	return d, ""
}

// isFragmentRef reports whether an entry or a link's inside is in the
// new form: a #:~: not inside quotes.
func isFragmentRef(s string) bool {
	s = strings.TrimSpace(s)
	if r, _ := utf8.DecodeRuneInString(s); s == "" || strings.ContainsRune(`"“”„«»'‘’`, r) {
		return false
	}
	return strings.Contains(s, "#:~:")
}

// fragmentRefs are the links a new form names, one per directive (one
// with Bad if none could be read).
func fragmentRefs(s string) []linkRef {
	file, dirs, bad := parseFragment(strings.TrimSpace(s))
	if len(dirs) == 0 {
		return []linkRef{{File: file, Bad: bad}}
	}
	out := make([]linkRef, len(dirs))
	for i, d := range dirs {
		out[i] = linkRef{File: file, Dir: d}
	}
	return out
}

// normDir is a directive's parts as matched: their words normalised.
type normDir struct{ pre, start, end, suf []string }

func normOf(d textDirective) normDir {
	return normDir{normWords(d.Prefix), normWords(d.Start), normWords(d.End), normWords(d.Suffix)}
}

// wordAt is a word's place in the source: its passage, its index there.
type wordAt struct{ p, w int32 }

// fragIndex is the source's words, for the new form: where each one is
// (in order), and each passage's neighbours with words.
type fragIndex struct {
	words      [][]string // each passage's words as its page shows them (Words, mostly)
	at         map[string][]wordAt
	prev, next []int // the passage before / after with words (-1: none)
}

// index is the matcher's word index, made the first time a new-form
// link is matched (a script with old links only does without it).
func (lm *linkMatcher) index() *fragIndex {
	lm.fiOnce.Do(lm.makeIndex)
	return lm.fi
}

func (lm *linkMatcher) makeIndex() {
	fi := &fragIndex{words: make([][]string, len(lm.ps)), at: map[string][]wordAt{}, prev: make([]int, len(lm.ps)), next: make([]int, len(lm.ps))}
	last := -1
	for i, p := range lm.ps {
		fi.prev[i] = last
		fi.words[i] = p.Words
		if p.Shown != p.Plain { // a link's target, a footnote: not on the page
			fi.words[i] = normWords(p.Shown)
		}
		if p.linkable() {
			for w, word := range fi.words[i] {
				fi.at[word] = append(fi.at[word], wordAt{int32(i), int32(w)})
			}
			last = i
		}
	}
	last = -1
	for i := len(lm.ps) - 1; i >= 0; i-- {
		fi.next[i] = last
		if lm.ps[i].linkable() {
			last = i
		}
	}
	lm.fi = fi
}

// runAt reports whether ws has run at w.
func runAt(ws []string, w int, run []string) bool {
	return w >= 0 && w+len(run) <= len(ws) && slices.Equal(ws[w:w+len(run)], run)
}

// prefixOK: pre ends right before word w of passage p (in it, or at the
// end of the passage before when w is its first).
func (lm *linkMatcher) prefixOK(p, w int, pre []string) bool {
	if len(pre) == 0 {
		return true
	}
	if w >= len(pre) {
		return runAt(lm.fi.words[p], w-len(pre), pre)
	}
	if w > 0 {
		return false
	}
	q := lm.fi.prev[p]
	return q >= 0 && runAt(lm.fi.words[q], len(lm.fi.words[q])-len(pre), pre)
}

// suffixOK: suf begins at word e of passage p (or, at its end, the next
// passage begins with it).
func (lm *linkMatcher) suffixOK(p, e int, suf []string) bool {
	if len(suf) == 0 {
		return true
	}
	ws := lm.fi.words[p]
	if e < len(ws) {
		return runAt(ws, e, suf)
	}
	q := lm.fi.next[p]
	return q >= 0 && runAt(lm.fi.words[q], 0, suf)
}

// findEnd is the passage of the first end at or after word e of passage
// p, followed by suf; -1 if none.
func (lm *linkMatcher) findEnd(p, e int, end, suf []string) int {
	list := lm.fi.at[end[0]]
	k := sort.Search(len(list), func(k int) bool {
		a := list[k]
		return int(a.p) > p || (int(a.p) == p && int(a.w) >= e)
	})
	for _, a := range list[k:] {
		if runAt(lm.fi.words[a.p], int(a.w), end) && lm.suffixOK(int(a.p), int(a.w)+len(end), suf) {
			return int(a.p)
		}
	}
	return -1
}

// nearSpan is how far after its start a near end is looked for.
const nearSpan = 40

// nearEnd is the passage from p on (nearSpan of them) whose last words
// are fewest edits from end, if that is few enough (as near starts are:
// at least 4 words, a quarter of them edited); -1 if none.
func (lm *linkMatcher) nearEnd(p int, end []string) int {
	if len(end) < 4 {
		return -1
	}
	limit := max(1, len(end)/4)
	rev := slices.Clone(end)
	slices.Reverse(rev)
	best, at := limit+1, -1
	for q := p; q < len(lm.ps) && q < p+nearSpan; q++ {
		if !lm.ps[q].linkable() {
			continue
		}
		ws := slices.Clone(lm.fi.words[q])
		slices.Reverse(ws)
		if d := prefixDistanceIn(rev, ws, limit, &lm.row); d < best {
			best, at = d, q
		}
	}
	return at
}

// span is a range of passages a directive found.
type span struct{ first, last int }

// findDir matches a directive on the passages (see the top of the file).
func (lm *linkMatcher) findDir(d normDir) linkMatch {
	if len(d.start) == 0 {
		return linkMatch{Kind: linkMissing}
	}
	lm.index()
	var exact []span
	var loose []wordAt     // textStart found, its context or end not: the first in each passage
	add := func(sp span) { // (in order: the same one comes in a row)
		if n := len(exact); n == 0 || exact[n-1] != sp {
			exact = append(exact, sp)
		}
	}
	endMissed := false // no end after a start: none after a later one either
	for _, a := range lm.fi.at[d.start[0]] {
		p, w := int(a.p), int(a.w)
		if !runAt(lm.fi.words[p], w, d.start) {
			continue
		}
		e := w + len(d.start)
		withContext := lm.prefixOK(p, w, d.pre)
		switch {
		case len(d.end) == 0:
			if withContext && lm.suffixOK(p, e, d.suf) {
				add(span{p, p})
			}
		case withContext && !endMissed:
			if q := lm.findEnd(p, e, d.end, d.suf); q >= 0 {
				add(span{p, q})
				continue
			}
			endMissed = true
		}
		if n := len(loose); n == 0 || int(loose[n-1].p) != p {
			loose = append(loose, a)
		}
	}
	switch {
	case len(exact) == 1:
		return linkMatch{Kind: linkFound, Passages: []int{exact[0].first}, Last: exact[0].last}
	case len(exact) > 1:
		return ambiguousOf(exact, linkFound)
	case len(loose) > 1:
		mt := linkMatch{Kind: linkAmbiguous}
		for _, a := range loose {
			mt.Passages = append(mt.Passages, int(a.p))
		}
		return mt
	case len(loose) == 1:
		// the context or the end changed: the end as it is, else near, else the start's passage alone
		p, last := int(loose[0].p), int(loose[0].p)
		if len(d.end) > 0 {
			q := lm.findEnd(p, int(loose[0].w)+len(d.start), d.end, nil)
			if q < 0 {
				q = lm.nearEnd(p, d.end)
			}
			last = max(q, p)
		}
		return linkMatch{Kind: linkNear, Passages: []int{p}, Last: last}
	}
	// textStart edited: near a passage's opening, as the old form's
	mt := matchIn(d.start, lm.ps, lm.byFirst, &lm.row)
	if mt.Kind != linkNear { // ambiguous or missing (an exact opening was an occurrence)
		return mt
	}
	p := mt.Passages[0]
	last := p
	if len(d.end) > 0 {
		if q := lm.findEnd(p, 0, d.end, nil); q >= 0 {
			last = q
		} else if q := lm.nearEnd(p, d.end); q >= 0 {
			last = q
		}
	}
	return linkMatch{Kind: linkNear, Passages: []int{p}, Last: last}
}

// ambiguousOf is what spans found: ambiguous if they start in more than
// one passage, else kind (one passage, ends apart: the nearer end).
func ambiguousOf(spans []span, kind matchKind) linkMatch {
	var ps []int
	for _, s := range spans {
		if !slices.Contains(ps, s.first) {
			ps = append(ps, s.first)
		}
	}
	slices.Sort(ps)
	if len(ps) == 1 {
		last := spans[0].last
		for _, s := range spans {
			last = min(last, s.last)
		}
		return linkMatch{Kind: kind, Passages: ps, Last: last}
	}
	return linkMatch{Kind: linkAmbiguous, Passages: ps}
}

// matchRef finds what a link names, remembered by its key.
func (lm *linkMatcher) matchRef(r linkRef) linkMatch {
	switch {
	case r.Bad != "":
		return linkMatch{Kind: linkMissing}
	case r.Old:
		return lm.match(r.words())
	}
	lm.mu.Lock()
	defer lm.mu.Unlock()
	key := r.rawKey()
	if mt, ok := lm.found[key]; ok {
		return mt
	}
	mt := lm.findDir(normOf(r.Dir))
	if len(lm.found) >= maxRemembered {
		clear(lm.found)
	}
	lm.found[key] = mt
	return mt
}

// matchRefIn is matchRef on passages (a matcher for them made, or the
// last one made for them).
func matchRefIn(r linkRef, ps []passage) linkMatch { return matcherFor(ps).matchRef(r) }

// lastMatcher is the matcher matcherFor made last.
var lastMatcher struct {
	sync.Mutex
	lm *linkMatcher
}

// matcherFor is a matcher for the passages: the one made for them last,
// while they are the same.
func matcherFor(ps []passage) *linkMatcher {
	lastMatcher.Lock()
	defer lastMatcher.Unlock()
	if lastMatcher.lm == nil || !samePassages(lastMatcher.lm.ps, ps) {
		lastMatcher.lm = newLinkMatcher(ps)
	}
	return lastMatcher.lm
}

// Writing (Joop 2026-11-22: "checking shortest unique for start and
// finish"): a new link is as short as it can be and still find its
// passages alone. textStart is the fewest first words of the first
// passage, textEnd the fewest last words of the last one, that make the
// range find exactly these passages (fewest in all; of those as many,
// the fewer end words); a passage no longer than that is quoted whole,
// as textStart alone. When no words of the passages themselves tell them
// from another place, the words before them (prefix-) or after them
// (-suffix) do. The links are looked at again whenever the source
// changes (relink.go): one the change made ambiguous grows then.
//
// (Reading old links, cut entries' old form, and near starts keep
// source.go's linkWords.)

const maxContext = 8 // a prefix's or suffix's words, at most

// wordFields is how many of fs, from the start (or the end), hold n
// words: fields of punctuation only (". . .") do not count.
func wordFields(fs []string, n int, fromEnd bool) int {
	got := 0
	for k := range fs {
		f := fs[k]
		if fromEnd {
			f = fs[len(fs)-1-k]
		}
		if normWords(f) != nil {
			got++
		}
		if got == n {
			return k + 1
		}
	}
	return len(fs)
}

// before reports whether a comes before word w of passage p.
func (a wordAt) before(p, w int) bool { return int(a.p) < p || int(a.p) == p && int(a.w) < w }

// shortestRange is the shortest textStart and textEnd that find
// passages [i, j] alone (with the context pre and suf): ns of fi's
// fields (the first passage's first block) and ne of fj's (the last
// one's last block), ne 0 for fi quoted whole; ns 0 if none does. It
// reads the index as findDir matches: textStart anywhere in a passage,
// textEnd the first after it.
func (lm *linkMatcher) shortestRange(i, j int, fi, fj []string, single bool, pre, suf []string) (ns, ne int) {
	fx := lm.index()
	fw := make([][]string, len(fi)) // each field's words
	for k, f := range fi {
		fw[k] = normWords(f)
	}
	wholeWords, lastField := 0, 0 // the whole's words; its last field with words, from 1
	for k, w := range fw {
		wholeWords += len(w)
		if len(w) > 0 {
			lastField = k + 1
		}
	}
	var run []string // textStart's words
	var cands []wordAt
	best, bestNs, bestNe := 0, 0, 0
	wholeOK := false
	for k := 1; k <= len(fi); k++ {
		if len(fw[k-1]) == 0 {
			continue // punctuation only: no word more
		}
		first := len(run) == 0
		run = append(run, fw[k-1]...)
		if first {
			for _, a := range fx.at[run[0]] {
				if lm.prefixOK(int(a.p), int(a.w), pre) {
					cands = append(cands, a)
				}
			}
		}
		kept := cands[:0]
		for _, a := range cands {
			if runAt(fx.words[a.p], int(a.w), run) {
				kept = append(kept, a)
			}
		}
		cands = kept
		if !slices.Contains(cands, wordAt{int32(i), 0}) {
			return 0, 0 // (the page's words are not the index's: findDir decides)
		}
		if single && k == lastField { // the whole: as textStart alone
			wholeOK = true
			for _, a := range cands {
				if int(a.p) != i && lm.suffixOK(int(a.p), int(a.w)+len(run), suf) {
					wholeOK = false
					break
				}
			}
			break
		}
		if best > 0 && len(run)+1 >= best {
			break // no shorter with an end
		}
		// another passage before the last one with these words finds a
		// range of its own; the first after it (later) must find no end
		later, bad := wordAt{-1, -1}, false
		for _, a := range cands {
			if int(a.p) == i {
				continue
			}
			if int(a.p) <= j {
				bad = true
			} else {
				later = a
			}
			break
		}
		if bad {
			continue
		}
		var end []string
		for m := 1; m <= len(fj); m++ {
			f := fj[len(fj)-m]
			w := normWords(f)
			if len(w) == 0 {
				continue
			}
			if single && k+m >= len(fi) {
				break // they would meet: the whole instead
			}
			end = append(slices.Clone(w), end...)
			if best > 0 && len(run)+len(end) >= best {
				break
			}
			if lm.endFinds(end, i, len(run), j, later, suf) {
				best, bestNs, bestNe = len(run)+len(end), k, m
				break
			}
		}
	}
	switch {
	case wholeOK && (best == 0 || wholeWords <= best):
		return len(fi), 0
	case best > 0:
		return bestNs, bestNe
	}
	return 0, 0
}

// endFinds reports whether textEnd end, after a textStart of n words
// opening passage i, is first found in passage j; and finds none after
// the textStart's first place past j (later; p -1 for none).
func (lm *linkMatcher) endFinds(end []string, i, n, j int, later wordAt, suf []string) bool {
	list := lm.fi.at[end[0]]
	k := sort.Search(len(list), func(k int) bool { return !list[k].before(i, n) })
	fits := func(a wordAt) bool {
		return runAt(lm.fi.words[a.p], int(a.w), end) && lm.suffixOK(int(a.p), int(a.w)+len(end), suf)
	}
	for ; k < len(list); k++ {
		if fits(list[k]) {
			break
		}
	}
	if k == len(list) || int(list[k].p) != j {
		return false
	}
	if later.p < 0 {
		return true
	}
	from := int(later.w) + n
	k = sort.Search(len(list), func(k int) bool { return !list[k].before(int(later.p), from) })
	for ; k < len(list); k++ {
		if fits(list[k]) {
			return false
		}
	}
	return true
}

// fragmentFor is the shortest link to passages [i, j] (see Writing,
// above), written as the source has the words (the browser compares the
// text). unique is false if nothing tells them apart; the directive then
// has a few words of each end and all the context there is.
func fragmentFor(lm *linkMatcher, i, j int) (d textDirective, unique bool) {
	ps := lm.ps
	bi, bj := blockFields(ps[i].shown()), blockFields(ps[j].shown())
	if len(bi) == 0 || len(bj) == 0 {
		return textDirective{Start: ps[i].Plain}, false
	}
	// a term is matched within one block of the page: textStart from the
	// first's first, textEnd from the last's last (a quote's paragraphs)
	fi, fj := bi[0], bj[len(bj)-1]
	single := i == j && len(bi) == 1
	join := strings.Join
	ok := func(d textDirective) bool {
		mt := lm.matchRef(linkRef{Dir: d})
		return mt.Kind == linkFound && mt.Passages[0] == i && mt.Last == j
	}
	mk := func(ns, ne int) textDirective {
		if single && ns+ne >= len(fi) || ne == 0 {
			return textDirective{Start: join(fi, " ")}
		}
		d := textDirective{Start: join(fi[:ns], " "), End: join(fj[len(fj)-ne:], " ")}
		// a range's inner edges need no punctuation (a word boundary is one): easier to read
		d.Start = strings.TrimRight(d.Start, ",;:-–—")
		d.End = strings.TrimLeft(d.End, ",;:-–—")
		if normWords(d.Start) == nil || normWords(d.End) == nil {
			return textDirective{Start: join(fi[:ns], " "), End: join(fj[len(fj)-ne:], " ")}
		}
		return d
	}
	try := func(pre, suf string) (textDirective, bool) {
		ns, ne := lm.shortestRange(i, j, fi, fj, single, normWords(pre), normWords(suf))
		if ns == 0 {
			return textDirective{}, false
		}
		d := mk(ns, ne)
		d.Prefix, d.Suffix = pre, suf
		return d, ok(d)
	}
	if d, good := try("", ""); good {
		return d, true
	}
	var before, after []string
	if q := lm.index().prev[i]; q >= 0 {
		if b := blockFields(ps[q].shown()); len(b) > 0 {
			before = b[len(b)-1]
		}
	}
	if q := lm.fi.next[j]; q >= 0 {
		if b := blockFields(ps[q].shown()); len(b) > 0 {
			after = b[0]
		}
	}
	for k := 1; k <= min(maxContext, len(before)); k++ {
		if pre := join(before[len(before)-k:], " "); normWords(pre) != nil {
			if d, good := try(pre, ""); good {
				return d, true
			}
		}
	}
	for k := 1; k <= min(maxContext, len(after)); k++ {
		if suf := join(after[:k], " "); normWords(suf) != nil {
			if d, good := try("", suf); good {
				return d, true
			}
		}
	}
	var pre, suf string
	if len(before) > 0 {
		pre = join(before[len(before)-min(maxContext, len(before)):], " ")
	}
	if len(after) > 0 {
		suf = join(after[:min(maxContext, len(after))], " ")
	}
	if normWords(pre) == nil {
		pre = ""
	}
	if normWords(suf) == nil {
		suf = ""
	}
	if d, good := try(pre, suf); good {
		return d, true
	}
	// nothing tells them apart (two passages alike, and their
	// neighbours): a few words of each end, all the context there is
	d = mk(wordFields(fi, 4, false), wordFields(fj, 3, true))
	d.Prefix, d.Suffix = pre, suf
	return d, ok(d)
}

// shown is the passage's text as its page shows it (Plain if not set).
func (p passage) shown() string {
	if p.Shown == "" {
		return p.Plain
	}
	return p.Shown
}

// listMark is a list item's or a quote's mark, as a passage's Plain has
// it for one nested in it (the page shows no such text).
var listMark = regexp.MustCompile(`^([-*+>]|\d{1,9}[.)])$`)

// blockFields are a passage's blocks' words as the page shows them (a
// quote's or a list item's paragraphs, nested items): split at blank
// lines, a nested item's mark left out.
func blockFields(plain string) [][]string {
	var out [][]string
	for _, para := range strings.Split(plain, "\n\n") {
		f := strings.Fields(para)
		for len(f) > 0 && listMark.MatchString(f[0]) {
			f = f[1:]
		}
		if len(f) > 0 {
			out = append(out, f)
		}
	}
	return out
}

// linkNoteFor is the note that links to passages [i, j], and whether it
// tells them from any other place.
func linkNoteFor(lm *linkMatcher, i, j int) (note string, unique bool) {
	d, unique := fragmentFor(lm, i, j)
	return fragmentNote("", []textDirective{d}), unique
}

// fragmentNote is a link note in the new form.
func fragmentNote(file string, dirs []textDirective) string {
	return "[[from: " + fragmentString(file, dirs, false) + "]]"
}

// runsOf parts passages (in any order) into runs of neighbours: [i, j]
// pairs, a heading or a break between counting as next to.
func runsOf(ps []passage, list []int) [][2]int {
	list = slices.Clone(list)
	slices.Sort(list)
	list = slices.Compact(list)
	var out [][2]int
	for _, p := range list {
		if n := len(out); n > 0 {
			last := out[n-1][1]
			between := true
			for q := last + 1; q < p; q++ {
				if ps[q].counted() {
					between = false
					break
				}
			}
			if between && p > last {
				out[n-1][1] = p
				continue
			}
		}
		out = append(out, [2]int{p, p})
	}
	return out
}

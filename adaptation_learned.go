// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/storyread"
)

// What Accolade learned from the writer (docs/ADAPTATION.md, step 7),
// kept in the script's adaptation block with the cuts, so it travels
// with the file and the writer can read and change it:
//
//	/* Adaptation
//	characters: ANNA = Anna, Annie, Miss Kovač
//	not characters: Monday, Lord
//	places: the house = INT. KOVAČ HOUSE
//	skipped links:
//	  INT. KITCHEN - NIGHT = "Anna opened the kitchen door."
//	cut:
//	  "The rain kept falling." (2)
//	*/
//
//   - characters: a cue, then the names the source has for that
//     character. The source's reading takes them as one speaker, and the
//     suggestions give their lines under the cue.
//   - not characters: capitalised words that are no one; the reading
//     drops them as characters and as speakers.
//   - places: a place as the source reads it (KITCHEN, the house; a
//     leading article does not count), then the heading the writer gave
//     it, without the time: a heading suggested for that place is it.
//   - skipped links: a scene's heading, then a passage it was offered as
//     a link and the writer skipped; it is not offered to that scene again.
//
// Each section takes its entries on the key's line or on the lines below
// it (indented, as a list, or not), up to the next "name:" line; several
// on the key's line are parted by ";" (by "," for not characters). The
// keys are read in any case, and a few other words for them too
// (aliases:, names:, not names:, skipped:). An entry it cannot read is
// kept in its section as written; lines outside the sections are kept as
// they are. A section written again by Accolade is written as above: one
// entry on the key's line, more one per line below it.

// aliasEntry is a characters: entry: the cue and the source's names.
type aliasEntry struct {
	Cue   string
	Names []string
}

// placeEntry is a places: entry.
type placeEntry struct {
	Place, Heading string
}

// skipEntry is a skipped links: entry: the scene's heading, the passage.
type skipEntry struct {
	Heading string
	Link    cutMark
}

// learned is what the block holds besides the cuts.
type learned struct {
	Aliases []aliasEntry
	Not     []string
	Places  []placeEntry
	Skipped []skipEntry
	extra   [sectionCount][]string // entries not read, kept as written
}

// The block's sections, in the order a new one is put in.
const (
	secCharacters = iota
	secNot
	secPlaces
	secSkipped
	secCut
	sectionCount
)

// sectionOf is the section a key opens (-1: none of these).
func sectionOf(key string) int {
	switch strings.Join(strings.Fields(key), " ") {
	case "characters", "character", "aliases", "names":
		return secCharacters
	case "not characters", "not character", "no characters", "not names", "no one":
		return secNot
	case "places", "place":
		return secPlaces
	case "skipped links", "skipped link", "skipped", "skips":
		return secSkipped
	case "cut", "cuts":
		return secCut
	}
	return -1
}

// sectionEntries are the entries of each section of the block: what
// follows the key on its line, then each line below it, list marks
// taken off.
func sectionEntries(inside string) (entries [sectionCount][]string, inline [sectionCount][]string) {
	in := -1
	for _, line := range strings.Split(inside, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if key, rest, ok := isBlockKey(line); ok {
			in = sectionOf(key)
			if in >= 0 && strings.TrimSpace(rest) != "" {
				inline[in] = append(inline[in], strings.TrimSpace(rest))
			}
			continue
		}
		if in >= 0 {
			e := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-*•"))
			if e != "" {
				entries[in] = append(entries[in], e)
			}
		}
	}
	return entries, inline
}

// splitPair parts "left = right" (or → or ->).
func splitPair(s string) (left, right string, ok bool) {
	for _, sep := range []string{"=", "→", "->"} {
		if l, r, found := strings.Cut(s, sep); found {
			l, r = strings.TrimSpace(l), strings.TrimSpace(r)
			return l, r, l != "" && r != ""
		}
	}
	return "", "", false
}

// splitSkipped parts a skipped links: entry, "heading = link": the link
// in the new form from the word with its #:~: (an = in text= is no
// separator), else as splitPair.
func splitSkipped(s string) (heading, link string, ok bool) {
	i := strings.Index(s, "#:~:")
	if i < 0 {
		return splitPair(s)
	}
	at := 0 // the file's start: after the last space or separator before it
	if k := strings.LastIndexAny(s[:i], " \t=→>"); k >= 0 {
		_, n := utf8.DecodeRuneInString(s[k:])
		at = k + n
	}
	heading = strings.TrimSpace(s[:at])
	for _, sep := range []string{"=", "→", "->"} {
		if h, found := strings.CutSuffix(heading, sep); found {
			heading = strings.TrimSpace(h)
			link = strings.TrimSpace(s[at:])
			return heading, link, heading != "" && link != ""
		}
	}
	return "", "", false
}

// splitList parts a list of names by commas and semicolons.
func splitList(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' }) {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// nameKey is a name's words, for comparing: case and punctuation aside.
func wordsKey(s string) string { return strings.Join(normWords(s), " ") }

// parseLearned reads the block's learned sections, leniently.
func parseLearned(inside string) learned {
	var l learned
	entries, inline := sectionEntries(inside)
	each := func(sec int, f func(e string) bool) {
		for _, line := range inline[sec] {
			parts := []string{line}
			if sec != secNot {
				parts = strings.Split(line, ";")
			}
			for _, e := range parts {
				if e = strings.TrimSpace(e); e != "" && !f(e) {
					l.extra[sec] = append(l.extra[sec], e)
				}
			}
		}
		for _, e := range entries[sec] {
			if !f(e) {
				l.extra[sec] = append(l.extra[sec], e)
			}
		}
	}
	each(secCharacters, func(e string) bool {
		cue, names, ok := splitPair(e)
		if !ok {
			return false
		}
		cue = strings.ToUpper(cueName(cue))
		for _, n := range splitList(names) {
			l.addAlias(cue, n)
		}
		return cue != ""
	})
	each(secNot, func(e string) bool {
		for _, n := range splitList(e) {
			l.addNot(n)
		}
		return true
	})
	each(secPlaces, func(e string) bool {
		place, heading, ok := splitPair(e)
		if ok {
			l.setPlace(place, heading)
		}
		return ok
	})
	each(secSkipped, func(e string) bool {
		heading, quote, ok := splitSkipped(e)
		if !ok {
			return false
		}
		cs := parseCutEntries(quote)
		for _, c := range cs {
			l.addSkip(heading, c)
		}
		return len(cs) > 0
	})
	return l
}

// aliasOf is the cue a name of the source is another name for ("": none).
func (l *learned) aliasOf(name string) string {
	k := wordsKey(name)
	for _, a := range l.Aliases {
		for _, n := range a.Names {
			if wordsKey(n) == k {
				return a.Cue
			}
		}
	}
	return ""
}

// addAlias makes name another name of cue (and of no other cue).
func (l *learned) addAlias(cue, name string) {
	l.dropAlias(name)
	k := wordsKey(name)
	if k == "" {
		return
	}
	for i := range l.Aliases {
		if l.Aliases[i].Cue == cue {
			l.Aliases[i].Names = append(l.Aliases[i].Names, name)
			return
		}
	}
	l.Aliases = append(l.Aliases, aliasEntry{Cue: cue, Names: []string{name}})
}

// renameCue makes the names of cue old those of cue nu; false if old is
// no cue of the characters: entries.
func (l *learned) renameCue(old, nu string) bool {
	i := slices.IndexFunc(l.Aliases, func(a aliasEntry) bool { return a.Cue == old })
	if i < 0 || old == nu {
		return false
	}
	names := l.Aliases[i].Names
	l.Aliases = slices.Delete(l.Aliases, i, i+1)
	for _, n := range names {
		l.addAlias(nu, n)
	}
	return true
}

// dropAlias takes name out of the characters: entries.
func (l *learned) dropAlias(name string) {
	k := wordsKey(name)
	for i := range l.Aliases {
		l.Aliases[i].Names = slices.DeleteFunc(l.Aliases[i].Names, func(n string) bool { return wordsKey(n) == k })
	}
	l.Aliases = slices.DeleteFunc(l.Aliases, func(a aliasEntry) bool { return len(a.Names) == 0 })
}

// isNot reports whether name is on the not characters list.
func (l *learned) isNot(name string) bool {
	k := wordsKey(name)
	return slices.ContainsFunc(l.Not, func(n string) bool { return wordsKey(n) == k })
}

func (l *learned) addNot(name string) {
	if wordsKey(name) != "" && !l.isNot(name) {
		l.Not = append(l.Not, name)
	}
}

func (l *learned) dropNot(name string) {
	k := wordsKey(name)
	l.Not = slices.DeleteFunc(l.Not, func(n string) bool { return wordsKey(n) == k })
}

// placeKey is a place's words without a leading article, as storyread
// matches them ("the house" is HOUSE).
func placeKey(place string) string {
	f := normWords(place)
	for len(f) > 1 && leadingArticles[f[0]] {
		f = f[1:]
	}
	return strings.Join(f, " ")
}

// leadingArticles are the articles of the languages the source is read
// in (internal/storyread/lang).
var leadingArticles = map[string]bool{"the": true, "a": true, "an": true, "de": true, "het": true, "een": true,
	"o": true, "os": true, "as": true, "um": true, "uma": true, "el": true, "la": true, "los": true, "las": true,
	"un": true, "una": true, "le": true, "les": true, "l": true, "une": true, "der": true, "die": true, "das": true,
	"ein": true, "eine": true}

// placeFor is the heading the writer gave place ("": none).
func (l *learned) placeFor(place string) string {
	k := placeKey(place)
	for _, p := range l.Places {
		if placeKey(p.Place) == k {
			return p.Heading
		}
	}
	return ""
}

// setPlace gives place its heading, the last one given winning.
func (l *learned) setPlace(place, heading string) {
	k := placeKey(place)
	for i := range l.Places {
		if placeKey(l.Places[i].Place) == k {
			l.Places[i].Heading = heading
			return
		}
	}
	l.Places = append(l.Places, placeEntry{Place: place, Heading: heading})
}

func (l *learned) dropPlace(place string) {
	k := placeKey(place)
	l.Places = slices.DeleteFunc(l.Places, func(p placeEntry) bool { return placeKey(p.Place) == k })
}

func (l *learned) addSkip(heading string, c cutMark) {
	hk := wordsKey(heading)
	for _, s := range l.Skipped {
		if wordsKey(s.Heading) == hk && s.Link.key() == c.key() {
			return
		}
	}
	l.Skipped = append(l.Skipped, skipEntry{Heading: heading, Link: c})
}

// empty reports whether nothing was learned.
func (l *learned) empty() bool {
	return len(l.Aliases)+len(l.Not)+len(l.Places)+len(l.Skipped) == 0
}

// sectionLines are a section as Accolade writes it: the key, its entries
// on the key's line if there is one, else one per line below.
func sectionLines(key string, entries []string, inline bool) []string {
	switch {
	case len(entries) == 0:
		return nil
	case inline:
		return []string{key + ": " + strings.Join(entries, ", ")}
	case len(entries) == 1 && !strings.HasPrefix(entries[0], "\""):
		return []string{key + ": " + entries[0]}
	}
	out := []string{key + ":"}
	for _, e := range entries {
		out = append(out, "  "+e)
	}
	return out
}

// lines are each learned section as written.
func (l *learned) lines() [sectionCount][]string {
	var out [sectionCount][]string
	var es []string
	for _, a := range l.Aliases {
		es = append(es, a.Cue+" = "+strings.Join(a.Names, ", "))
	}
	out[secCharacters] = sectionLines("characters", append(es, l.extra[secCharacters]...), false)
	out[secNot] = sectionLines("not characters", append(slices.Clone(l.Not), l.extra[secNot]...), true)
	es = nil
	for _, p := range l.Places {
		es = append(es, p.Place+" = "+p.Heading)
	}
	out[secPlaces] = sectionLines("places", append(es, l.extra[secPlaces]...), false)
	es = nil
	for _, s := range l.Skipped {
		es = append(es, s.Heading+" = "+s.Link.String())
	}
	if len(es)+len(l.extra[secSkipped]) > 0 {
		out[secSkipped] = append([]string{"skipped links:"}, indent(append(es, l.extra[secSkipped]...))...)
	}
	return out
}

func indent(ls []string) []string {
	out := make([]string, len(ls))
	for i, s := range ls {
		out[i] = "  " + s
	}
	return out
}

// withLearned is the block's inside with its learned sections these,
// the cuts and the rest kept as they are ("" if nothing is left).
func withLearned(inside string, l learned) string {
	ls := l.lines()
	for sec := secCharacters; sec <= secSkipped; sec++ {
		inside = withSection(inside, sec, ls[sec])
	}
	return inside
}

// withSection is the block's inside with section sec as lines (none: it
// goes), where it was; a new one before the first section that comes
// after it in the block's order, else at the end. Lines that are no
// section's are kept.
func withSection(inside string, sec int, section []string) string {
	var kept []string
	at := -1     // where the section was
	before := -1 // the first later section
	in := false
	for _, line := range strings.Split(inside, "\n") {
		if key, _, ok := isBlockKey(line); ok {
			s := sectionOf(key)
			in = s == sec
			if in {
				if at < 0 {
					at = len(kept)
				}
				continue
			}
			if s > sec && before < 0 {
				before = len(kept)
			}
		}
		if in {
			continue // its entries, and blank lines in it
		}
		kept = append(kept, line)
	}
	if at < 0 {
		at = before
	}
	if at < 0 {
		for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
			kept = kept[:len(kept)-1]
		}
		at = len(kept)
	}
	lines := slices.Concat(kept[:at], section, kept[at:])
	out := strings.TrimRight(strings.Join(lines, "\n"), "\n ")
	if strings.TrimSpace(out) == "" {
		return ""
	}
	return strings.TrimLeft(out, "\n")
}

// learnedCache is the last block read.
var learnedCache struct {
	sync.Mutex
	inside string
	l      *learned
	key    string // its hintsKey
}

// readLearned is what the script's adaptation block holds besides the
// cuts, read once per block, and its hints' key; not to be changed.
func readLearned(text string) (*learned, string) {
	inside, _ := adaptationBlock(text)
	learnedCache.Lock()
	defer learnedCache.Unlock()
	if learnedCache.l == nil || !sameString(learnedCache.inside, inside) {
		l := parseLearned(inside)
		learnedCache.inside, learnedCache.l, learnedCache.key = inside, &l, l.hintsKey()
	}
	return learnedCache.l, learnedCache.key
}

// learnedOf is readLearned's, a copy to change.
func learnedOf(text string) learned {
	l, _ := readLearned(text)
	return l.clone()
}

// clone is a copy to change.
func (l learned) clone() learned {
	c := learned{Not: slices.Clone(l.Not), Places: slices.Clone(l.Places), Skipped: slices.Clone(l.Skipped)}
	for _, a := range l.Aliases {
		c.Aliases = append(c.Aliases, aliasEntry{Cue: a.Cue, Names: slices.Clone(a.Names)})
	}
	for i := range l.extra {
		c.extra[i] = slices.Clone(l.extra[i])
	}
	return c
}

// hints are what the source's reading takes from it.
func (l *learned) hints() storyread.Hints {
	h := storyread.Hints{NotCharacters: l.Not}
	if len(l.Aliases) > 0 {
		h.Aliases = map[string]string{}
		for _, a := range l.Aliases {
			for _, n := range a.Names {
				h.Aliases[n] = a.Cue
			}
		}
	}
	if len(l.Places) > 0 {
		h.Places = map[string]string{}
		for _, p := range l.Places {
			h.Places[p.Place] = p.Heading
		}
	}
	return h
}

// hintsKey tells the hints apart, for what is read with them.
func (l *learned) hintsKey() string {
	ls := l.lines()
	return strings.Join(slices.Concat(ls[secCharacters], ls[secNot], ls[secPlaces]), "\n")
}

// skippedFor are the passages each scene heading (by nameKey) was
// offered as a link and skipped.
func (l *learned) skippedFor(ps []passage) map[string][]int {
	if len(l.Skipped) == 0 {
		return nil
	}
	out := map[string][]int{}
	for _, s := range l.Skipped {
		if c := resolveCut(s.Link, ps); c.first >= 0 {
			k := wordsKey(s.Heading)
			for p := c.first; p <= c.last; p++ {
				out[k] = append(out[k], p)
			}
		}
	}
	return out
}

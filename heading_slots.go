// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/rules"
)

// A scene heading's slots (rules.Slots), one source of truth for what
// each takes: New Scene's fields (new_scene.go) and the completion of a
// heading as it is typed (complete.go) offer the same words, in the same
// order — the language's inside/outside words (rules.Language.IOWords,
// its Matchup's both and the other way among them), the script's places
// (the most used first), the language's time words (the script's own
// where the language has none), the number a scene there gets.

// slotSource is what a script's slots take.
type slotSource struct {
	l      rules.Language
	places []string // the script's, the most used first
	times  []string // the script's, the most used first
	number string   // the number a new scene at the cursor gets ("": none)
}

// slotSourceOf is the source for a script's text in language l.
func slotSourceOf(text string, l rules.Language, starts []string) slotSource {
	places, times := scriptPlaces(text, l, starts)
	return slotSource{l: l, places: places, times: times}
}

// placesCache is the last script's places and times while one line is
// typed (completion asks on every key): the script without the cursor's
// line, the language and its starts are the key.
var placesCache struct {
	sync.Mutex
	key           string
	l             *fountain.Dialect
	places, times []string
}

// slotSourceTyping is slotSourceOf for a script being typed at cursor
// (in runes): the other lines' places read once while that line changes.
func slotSourceTyping(text string, cursor int, l rules.Language, starts []string) slotSource {
	rs := []rune(text)
	ls, le := lineBounds(rs, min(cursor, len(rs)))
	key := string(rs[:ls]) + "\x00" + string(rs[le:]) + "\x00" + strings.Join(starts, "\x1f")
	d := fountain.NewDialect(l)
	placesCache.Lock()
	defer placesCache.Unlock()
	if placesCache.key != key || placesCache.l != d {
		placesCache.places, placesCache.times = scriptPlaces(string(rs[:ls])+string(rs[le:]), l, starts)
		placesCache.key, placesCache.l = key, d
	}
	return slotSource{l: l, places: placesCache.places, times: placesCache.times}
}

// choices are what a slot of form takes, the commonest first; nil for
// free text.
func (s slotSource) choices(slot, form string) []string {
	switch slot {
	case rules.SlotIO:
		return s.l.IOWords(form)
	case rules.SlotTime:
		return s.timeWords()
	case rules.SlotPlace:
		return s.places
	case rules.SlotNumber:
		if s.number != "" {
			return []string{s.number}
		}
	}
	return nil
}

// timeWords are the language's times; for one with none (English: a
// heading's time is free) the script's, then Fountain's DAY and NIGHT.
func (s slotSource) timeWords() []string {
	if len(s.l.Times) > 0 {
		return s.l.Times
	}
	out := slices.Clone(s.times)
	if slices.Contains(s.l.Scenes, "INT") { // Fountain's own examples after the script's
		for _, t := range []string{"DAY", "NIGHT"} {
			if !slices.Contains(out, t) {
				out = append(out, t)
			}
		}
	}
	return out
}

// headingForms are the forms a heading in l may be typed in, the one
// New Scene writes first: then Fountain's (a language with scene words),
// then its patterns.
func headingForms(l rules.Language) []string {
	out := []string{l.HeadingForm()}
	if len(l.Scenes) > 0 && !slices.Contains(out, rules.SceneForm) {
		out = append(out, rules.SceneForm)
	}
	for _, p := range l.Headings {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// scriptPlaces are the places and times the script's headings have, the
// most used first, then in the order they first come (in capitals, as
// the reader gives them).
func scriptPlaces(text string, l rules.Language, starts []string) (places, times []string) {
	body := text[:bodyEndByte(text)]
	lines := strings.Split(body, "\n")
	d := fountain.NewDialect(l)
	count := map[string]int{}
	for _, it := range outlineOf(body, starts) {
		if !it.Scene {
			continue
		}
		parts, ok := d.HeadingParts(stripNotes(lines[it.Line])) // no [[red]]
		if !ok {
			continue
		}
		if p := parts[rules.SlotPlace]; p != "" {
			if count["p"+p] == 0 {
				places = append(places, p)
			}
			count["p"+p]++
		}
		if t := parts[rules.SlotTime]; t != "" {
			if count["t"+t] == 0 {
				times = append(times, t)
			}
			count["t"+t]++
		}
	}
	sort.SliceStable(places, func(i, j int) bool { return count["p"+places[i]] > count["p"+places[j]] })
	sort.SliceStable(times, func(i, j int) bool { return count["t"+times[i]] > count["t"+times[j]] })
	return places, times
}

// slotAt is where a heading being typed is: the slot being typed, from
// rune start of the line, what is typed of it, and the text joining it
// to the slot before that is still to come (lead, the separator typed
// in part: "INT. KITCHEN -" waits for " - " and the time).
type slotAt struct {
	form   string
	slot   string
	start  int
	prefix string
	lead   string
}

// slotsAt are the places a heading line typed so far can be at, by the
// language's forms (headingForms) in order; none for a line no form
// starts.
func (s slotSource) slotsAt(line string) []slotAt {
	var out []slotAt
	rs := []rune(line)
	var forms []string // each form, and it with slots left out (the writer skips one)
	for _, f := range headingForms(s.l) {
		for _, v := range rules.PatternVariants(f) {
			if !slices.Contains(forms, v) {
				forms = append(forms, v)
			}
		}
	}
	for _, f := range forms {
		p := &slotParse{rs: rs, m: modelOf(f), form: f, src: s}
		p.sep(0, 0, true)
		out = append(out, p.out...)
	}
	return out
}

// slotParse reads a line by one form, every way it can be read.
type slotParse struct {
	rs   []rune
	m    patternModel
	form string
	src  slotSource
	out  []slotAt
}

// words are the words slot i takes, for reading it.
func (p *slotParse) words(i int) []string { return p.src.choices(p.m.slots[i], p.form) }

// sep reads the text before slot j (j == len(slots): after the last) at
// rune p; exact: the slot before was typed in full (EXT., not EXT, which
// is still EXT. being typed), so the next may be offered with nothing
// of the separator typed.
func (p *slotParse) sep(j, at int, exact bool) {
	sep := []rune(p.m.seps[j])
	n, full := sepMatch(p.rs[at:], sep)
	switch {
	case full && j == len(p.m.slots):
		// the line read to its end
	case full:
		p.slot(j, at+n)
	case n == len(p.rs)-at && j < len(p.m.slots) && (n > 0 || exact): // the separator typed in part, or not yet
		p.out = append(p.out, slotAt{form: p.form, slot: p.m.slots[j], start: at, prefix: string(p.rs[at:]), lead: string(sep)})
	}
}

// slot reads slot i at rune at.
func (p *slotParse) slot(i, at int) {
	rest := p.rs[at:]
	s := p.m.slots[i]
	here := func() { p.out = append(p.out, slotAt{form: p.form, slot: s, start: at, prefix: string(rest)}) }
	free := s == rules.SlotTime && len(p.src.l.Times) == 0 // English's: any time
	switch {
	case s == rules.SlotPlace || s == rules.SlotAny || free:
		here()
		if i == len(p.m.slots)-1 {
			return
		}
		for k := 1; k < len(rest); k++ {
			sep := []rune(p.m.seps[i+1])
			switch n, full := sepMatch(rest[k:], sep); {
			case full && n > 0:
				p.slot(i+1, at+k+n)
			case !full && n > 0 && n == len(rest)-k: // the separator typed in part: "KITCHEN -"
				p.out = append(p.out, slotAt{form: p.form, slot: p.m.slots[i+1], start: at + k, prefix: string(rest[k:]), lead: string(sep)})
			}
		}
	case s == rules.SlotNumber:
		// letters before the digits and after them, as a locked script
		// gives them (A11, AB11, 10A, 10AA, 10aA: Lexington's numberSlot)
		k := 0
		for k < len(rest) && k < 4 && unicode.IsLetter(rest[k]) {
			k++
		}
		lead := k
		for k < len(rest) && unicode.IsDigit(rest[k]) {
			k++
		}
		for tail := 0; k > lead && k < len(rest) && tail < 4 && unicode.IsLetter(rest[k]); tail++ {
			k++
		}
		if k == lead && k < len(rest) { // letters, no digits: a letter being typed at most
			k = min(lead, 1)
		}
		if k == len(rest) {
			here()
		}
		if k > 0 {
			p.sep(i+1, at+k, true)
		}
	default: // a word of the language's
		up := strings.ToUpper(string(rest))
		partial := false
		for _, w := range p.words(i) {
			uw := strings.ToUpper(w)
			for k, form := range []string{uw, strings.TrimSuffix(uw, ".")} {
				if form != "" && strings.HasPrefix(up, form) && (k == 0 || form != uw) {
					p.sep(i+1, at+len([]rune(form)), k == 0)
				}
			}
			if strings.HasPrefix(uw, up) {
				partial = true
			}
		}
		if partial {
			here()
		}
	}
}

// sepMatch reads separator sep at the start of rs: how many runes it
// takes, and whether all of sep was there. A space is one or more
// spaces, a dash any dash; letters in any case. Part of it: rs ended
// first (n == len(rs), full false); else -1, false when it differs.
func sepMatch(rs, sep []rune) (int, bool) {
	n := 0
	for k := 0; k < len(sep); k++ {
		if n == len(rs) {
			return n, false
		}
		c := sep[k]
		switch {
		case unicode.IsSpace(c):
			if !unicode.IsSpace(rs[n]) {
				return -1, false
			}
			for n < len(rs) && unicode.IsSpace(rs[n]) {
				n++
			}
			for k+1 < len(sep) && unicode.IsSpace(sep[k+1]) {
				k++
			}
		case strings.ContainsRune("-–—", c):
			if !strings.ContainsRune("-–—", rs[n]) {
				return -1, false
			}
			n++
		default:
			if unicode.ToUpper(rs[n]) != unicode.ToUpper(c) {
				return -1, false
			}
			n++
		}
	}
	return n, true
}

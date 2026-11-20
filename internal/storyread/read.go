// Package storyread reads a story's prose for adapting it into a
// screenplay (docs/ADAPTATION.md, section 4): who is in it, what they
// say and who says it, where its scenes start and what their headings
// could be, and what is thought or felt rather than seen. Heuristics
// only, local and quick, with word lists per language (lang/*.txt): no
// model, no network, so the browser build has it too.
//
// It reads passages as the source pane shows them (paragraphs, headings,
// breaks) and keeps what it finds in their order, so that suggestions
// (step 6) can offer the next piece of a passage, and takes hints (step
// 7: names that are no characters, aliases) that the writer's choices
// will give.
package storyread

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
)

// PassageKind is what a passage is.
type PassageKind int

const (
	Para     PassageKind = iota // a paragraph
	Heading                     // a heading: a chapter's, a part's
	Break                       // a break: * * *, ---
	DashItem                    // a list item written with "-": dialogue, perhaps
)

// Passage is a piece of the source as given to Read.
type Passage struct {
	Kind  PassageKind
	Level int // a heading's
	Text  string
}

// Hints are what the writer has told the adaptation (its notes, step 7).
type Hints struct {
	NotCharacters []string          // capitalised words that are no one
	Aliases       map[string]string // a name: the character it is another name for
}

// Attribution is how a line's speaker was found.
type Attribution int

const (
	Unknown     Attribution = iota
	ByVerb                  // a speech verb with the name: "said Anna"
	ByPronoun               // a speech verb with a pronoun, who from the turns: "she said"
	ByParagraph             // another line of the paragraph has the name
	ByBeat                  // the paragraph's narration names them: "Anna looked up."
	ByIntro                 // the paragraph before introduced them: "Anna said:"
	ByTurn                  // the other of the two who were talking
)

func (a Attribution) String() string {
	return [...]string{"unknown", "speech verb", "pronoun and turns", "same paragraph", "action beside it",
		"introduced before", "turns"}[a]
}

// Narrator is the speaker of a line the first-person narrator says.
const Narrator = "Narrator"

// Line is a line of dialogue: what is said, who says it and how.
type Line struct {
	Text       string
	Speaker    string // "" unknown
	By         Attribution
	Manner     string   // the parenthetical: whispering
	Addressed  []string // names it calls out to
	Start, End int      // bytes, its first piece's start, its last's end
}

// Piece is a stretch of a passage, in order: said, or narration.
type Piece struct {
	Speech     bool
	Text       string
	Start, End int  // bytes
	Line       int  // the line it is part of (Speech), else -1
	Tag        bool // narration that only says who speaks a line: "said Tom,"
}

// Flag is a sentence of narration that a camera cannot see: a thought,
// a feeling, a memory. Show it, cut it, or make it voice-over.
type Flag struct {
	Sentence   string
	Word       string // the word that gives it away
	Start, End int
}

// Setting is where a scene is: inside or outside.
type Setting int

const (
	SettingUnknown Setting = iota
	Inside
	Outside
)

// Scene is a stretch of passages that reads as one scene, and the
// heading it suggests.
type Scene struct {
	First, Last int    // passages
	Why         string // what starts it: "chapter heading", "break", "The next morning"
	Setting     Setting
	Place       string // in capitals: KITCHEN, NETHERFIELD PARK
	Time        string // in capitals: MORNING; "" unknown
}

// Character is a name that reads as someone's.
type Character struct {
	Name      string
	Mentions  int
	Lines     int // spoken with the name by a speech verb
	Addressed int
	First     int // the passage it is first in
	Weight    float64
}

// PlaceName is a name that reads as a place's.
type PlaceName struct {
	Name     string
	Mentions int
	First    int
	Setting  Setting
}

// PassageReading is what a passage shows.
type PassageReading struct {
	Pieces     []Piece
	Lines      []Line
	Names      []string // the characters named in it
	Places     []string
	Flags      []Flag
	Scene      int    // which of the Reading's
	SceneStart string // why a scene starts here, "" if none does
}

// Reading is what a source shows.
type Reading struct {
	Lang       string // the word lists it was read with
	Passages   []PassageReading
	Characters []Character // the most weighty first
	Places     []PlaceName
	Scenes     []Scene
}

// para is a passage being read.
type para struct {
	Passage
	toks     []token
	segs     []segment
	dash     bool
	mentions []mention
	tags     map[int][2]int // a narration segment's start: the bytes of its sentence that says who speaks
}

// mention is a name in a passage: tokens [from, to).
type mention struct {
	from, to int
	name     string
	titled   bool
	inSpeech bool
	initial  bool    // at a sentence's start
	afterDet bool    // after a determiner: a noun, in German
	place    float64 // after "in", "at": how sure that makes it a place
	vocative bool
}

// reader is the state of one Read.
type reader struct {
	l      *Lang
	hints  Hints
	not    set
	ps     []*para
	lower  map[string]int // words written in lower case, counted
	stats  map[string]*nameStats
	turns  []string // the slots of the last speakers, the latest first
	labels map[string]string
	called string // the slot the last line addressed
}

type nameStats struct {
	mentions, nonInitial, titled, vocative, vocInitial, spoke, afterDet, first int
	place                                                                      float64
	placeWord                                                                  Setting
}

// Read reads the passages in language lang (its word lists; English's
// for one without).
func Read(ps []Passage, lang string, h Hints) *Reading {
	r := &reader{l: Language(lang), hints: h, not: set{}, lower: map[string]int{}, stats: map[string]*nameStats{},
		labels: map[string]string{}}
	for _, n := range h.NotCharacters {
		r.not[key(n)] = true
	}
	for _, p := range ps {
		r.ps = append(r.ps, r.prepare(p))
	}
	r.dashRuns()
	for _, p := range r.ps {
		for _, t := range p.toks {
			if !t.cap {
				r.lower[t.low]++
			}
		}
	}
	for i, p := range r.ps {
		r.findMentions(i, p)
	}
	out := &Reading{Lang: r.l.Code, Passages: make([]PassageReading, len(ps))}
	r.scenes(out)
	for i, p := range r.ps {
		if out.Passages[i].SceneStart != "" {
			r.turns, r.called = nil, ""
		}
		r.dialogue(i, p, &out.Passages[i])
		out.Passages[i].Flags = r.flags(p)
	}
	r.names(out)
	r.headings(out)
	return out
}

// prepare tokenizes a passage and parts it into segments.
func (r *reader) prepare(p Passage) *para {
	pp := &para{Passage: p}
	if p.Kind == DashItem {
		pp.Text = "— " + p.Text
	}
	pp.toks = tokenize(pp.Text)
	switch {
	case p.Kind == Heading || p.Kind == Break:
		pp.segs = []segment{{start: 0, end: len(pp.Text)}}
	default:
		if at, ok := dashStart(pp.Text); ok {
			pp.dash = true
			pp.segs = dashSegments(pp.Text, at)
		} else {
			pp.segs = quoteSegments(pp.Text)
		}
	}
	r.place(pp)
	if pp.dash {
		r.incises(pp)
	}
	return pp
}

// place puts each token in its segment and marks sentence starts.
func (r *reader) place(p *para) {
	s := 0
	for k := range p.toks {
		t := &p.toks[k]
		for s < len(p.segs)-1 && t.start >= p.segs[s].end {
			s++
		}
		t.seg = -1
		if s < len(p.segs) && t.start >= p.segs[s].start && t.start < p.segs[s].end {
			t.seg = s
		}
		switch {
		case k == 0:
			t.sent = true
		case t.seg != p.toks[k-1].seg && t.seg >= 0 && p.segs[t.seg].speech:
			t.sent = true // what is said starts anew
		default:
			prev := &p.toks[k-1]
			t.sent = fullStop(t.before) && !r.abbreviation(prev)
		}
	}
}

// abbreviation reports whether a word with a full stop after it is a
// title or an initial (Mr., D., S.), so the stop ends no sentence.
func (r *reader) abbreviation(t *token) bool {
	return strings.HasPrefix(strings.TrimLeft(t.after, " "), ".") &&
		(r.l.titles[t.low] && len([]rune(t.text)) <= 4 || t.cap && len([]rune(t.text)) == 1)
}

// incises splits a dash-style line where narration comes in after a
// comma, "!" or "?": "--Continue, disse eu accordando." It runs to the
// sentence's end; what follows it is said again.
func (r *reader) incises(p *para) {
	var out []segment
	for si, sg := range p.segs {
		if !sg.speech {
			out = append(out, sg)
			continue
		}
		cur := sg
		for k := range p.toks {
			t := &p.toks[k]
			if t.seg != si || t.start <= cur.start || t.start >= cur.end || t.cap {
				continue
			}
			if _, ok := r.l.isSpeech(t); !ok {
				continue
			}
			if !strings.ContainsAny(t.before, ",!?…") {
				// a clitic before it: "se dit l’hôte", "le dijo", "lui dit"
				if k == 0 {
					continue
				}
				c := &p.toks[k-1]
				if c.cap || len([]rune(c.text)) > 3 || !strings.ContainsAny(c.before, ",!?…") ||
					strings.TrimSpace(t.before) != "" || c.start <= cur.start {
					continue
				}
				t = c
			}
			out = append(out, segment{start: cur.start, end: t.start, speech: true})
			end := cur.end
			for j := k + 1; j < len(p.toks) && p.toks[j].start < cur.end; j++ {
				if sentenceEnd(p.toks[j].before) {
					end = p.toks[j].start
					break
				}
			}
			out = append(out, segment{start: t.start, end: end})
			cur = segment{start: end, end: cur.end, speech: true}
			if end >= sg.end {
				break
			}
		}
		if cur.start < cur.end && strings.TrimSpace(p.Text[cur.start:cur.end]) != "" {
			out = append(out, cur)
		}
	}
	p.segs = out
	r.place(p)
}

// dashRuns: list items written with "-" are dialogue only if their run
// reads so (a speech verb after a comma, or a question or exclamation);
// otherwise they are a list, read as narration.
func (r *reader) dashRuns() {
	for i := 0; i < len(r.ps); {
		if r.ps[i].Kind != DashItem {
			i++
			continue
		}
		j, talk := i, false
		for ; j < len(r.ps) && r.ps[j].Kind == DashItem; j++ {
			p := r.ps[j]
			if strings.ContainsAny(p.Text, "?!¿¡") || len(p.segs) > 1 {
				talk = true
			}
		}
		if !talk {
			for _, p := range r.ps[i:j] {
				p.Text, p.dash = p.Passage.Text, false
				p.toks = tokenize(p.Text)
				p.segs = []segment{{start: 0, end: len(p.Text)}}
				r.place(p)
			}
		}
		i = j
	}
}

func (p *para) speechAt(k int) bool {
	s := p.toks[k].seg
	return s >= 0 && p.segs[s].speech
}

// isPronounLike: words that are no name even when capitalised.
func (r *reader) noName(t *token) bool {
	l := r.l
	return l.notNames[t.low] || l.pronouns[t.low] || l.narrator[t.low] || l.determiners[t.low] ||
		l.possessives[t.low] || l.articles[t.low] || l.prepositions[t.low] || r.not[t.low] || unicode.IsDigit([]rune(t.text)[0])
}

// nameAt is the name starting at token k, if one does: an optional
// title (Mr, tio, don), then capitalised words, with particles between
// (de, van). Its end is the token after it.
func (r *reader) nameAt(p *para, k int) (m mention, ok bool) {
	toks, l := p.toks, r.l
	j := k
	titled := false
	joined := func(a, b int) bool { // tokens a and b=a+1 are one name
		if p.toks[a].seg != p.toks[b].seg {
			return false
		}
		between := strings.TrimSpace(p.toks[a].after)
		return between == "" || between == "." && r.abbreviation(&p.toks[a])
	}
	for j < len(toks) && l.titles[toks[j].low] && j+1 < len(toks) && joined(j, j+1) {
		titled = true
		j++
	}
	if j >= len(toks) || !toks[j].cap || r.noName(&toks[j]) || l.titles[toks[j].low] && !titled {
		return m, false
	}
	end := j + 1
	for end < len(toks) {
		switch {
		case toks[end].cap && joined(end-1, end) && !r.noName(&toks[end]) && !toks[end].sent:
			end++
			continue
		case l.particles[toks[end].low] && !toks[end].cap && end+1 < len(toks) && toks[end+1].cap &&
			joined(end-1, end) && joined(end, end+1) && !r.noName(&toks[end+1]):
			end += 2
			continue
		}
		break
	}
	var words []string
	for x := k; x < end; x++ {
		w := toks[x].text
		if r.abbreviation(&toks[x]) {
			w += "."
		}
		words = append(words, w)
	}
	n := len(words) - 1
	for _, poss := range []string{"’s", "'s"} {
		if strings.HasSuffix(words[n], poss) && len(words[n]) > len(poss) {
			words[n] = strings.TrimSuffix(words[n], poss)
		}
	}
	m = mention{from: k, to: end, name: strings.Join(words, " "), titled: titled || l.titles[toks[j].low],
		inSpeech: p.speechAt(j), initial: toks[k].sent}
	if k > 0 && (l.determiners[toks[k-1].low] || l.articles[toks[k-1].low]) && joined(k-1, k) {
		m.afterDet = true
	}
	if k > 0 && joined(k-1, k) {
		if w, ok := l.placeBefore[toks[k-1].low]; ok {
			m.place = w
		}
	}
	return m, true
}

// findMentions finds the names in passage i and counts them.
func (r *reader) findMentions(i int, p *para) {
	if p.Kind == Break {
		return
	}
	for k := 0; k < len(p.toks); {
		m, ok := r.nameAt(p, k)
		if !ok {
			k++
			continue
		}
		// "However Mr. Bennet": a capitalised word at the start that is
		// written in lower case elsewhere is no part of the name
		if m.initial && m.to-m.from > 1 && !m.titled && r.lower[p.toks[k].low] > 0 {
			k++
			continue
		}
		if p.speechAt(m.from) {
			m.vocative = r.vocative(p, m)
		}
		p.mentions = append(p.mentions, m)
		st := r.stat(m.name, i)
		st.mentions++
		if !m.initial {
			st.nonInitial++
		}
		if m.titled {
			st.titled++
		}
		if m.vocative && m.initial {
			st.vocInitial++
		} else if m.vocative {
			st.vocative++
		}
		if m.afterDet {
			st.afterDet++
		}
		st.place += m.place
		for x := m.from; x < m.to; x++ {
			switch {
			case r.l.interior[p.toks[x].low]:
				st.placeWord = Inside
			case r.l.exterior[p.toks[x].low]:
				st.placeWord = Outside
			}
		}
		k = m.to
	}
}

func (r *reader) stat(name string, passage int) *nameStats {
	st := r.stats[name]
	if st == nil {
		st = &nameStats{first: passage}
		r.stats[name] = st
	}
	return st
}

// vocative reports whether a name in what is said calls someone: set
// off by a comma at the start of a sentence ("Effi, eigentlich…", "My
// dear Mr. Bennet,", "Wel, Droogstoppel,") or ending it ("…, Hulda?").
func (r *reader) vocative(p *para, m mention) bool {
	first, last := &p.toks[m.from], &p.toks[m.to-1]
	after := strings.TrimSpace(last.after)
	end := m.to == len(p.toks) || p.toks[m.to].seg != first.seg
	start := m.from == 0 || p.toks[m.from-1].seg != first.seg
	if start && end {
		return false // all that is said: an answer ("Bingley.")
	}
	if strings.HasSuffix(strings.TrimSpace(first.before), ",") && (end || strings.HasPrefix(after, ".") ||
		strings.HasPrefix(after, "!") || strings.HasPrefix(after, "?")) {
		return true
	}
	if !strings.HasPrefix(after, ",") && !strings.HasPrefix(after, "!") && !strings.HasPrefix(after, "?") && !end {
		return false
	}
	// at most two words before it in its sentence, none a preposition
	// or article ("Immer am Trapez," is no call)
	for x, n := m.from, 0; !p.toks[x].sent; n++ {
		x--
		if n == 2 || x < 0 || p.toks[x].seg != first.seg {
			return false
		}
		t := &p.toks[x]
		if r.l.prepositions[t.low] || r.l.articles[t.low] || r.l.determiners[t.low] && !r.l.possessives[t.low] {
			return false
		}
	}
	return true
}

// isName reports whether a name found reads as someone's or somewhere's
// over the whole source.
func (r *reader) isName(name string) bool {
	st := r.stats[name]
	if st == nil || r.not[key(name)] {
		return false
	}
	signals := st.titled + st.vocative + st.spoke
	if st.nonInitial > 0 {
		signals += st.vocInitial // "Vielleicht, Mama": only a name if it is one elsewhere
	}
	if signals == 0 && st.nonInitial == 0 {
		return false
	}
	if !strings.Contains(name, " ") && signals == 0 && r.lower[strings.ToLower(name)] >= st.nonInitial {
		return false // a common word, capitalised here and there
	}
	if r.l.capNouns && signals == 0 {
		// every noun is capitalised: a name never comes after "der"
		return st.afterDet == 0 && (st.mentions >= 3 || st.place >= 1)
	}
	return true
}

// isPlace reports whether a name reads as a place's rather than a
// person's.
func (r *reader) isPlace(name string) bool {
	st := r.stats[name]
	person := float64(3*st.spoke + 2*st.vocative + 2*st.titled)
	place := st.place
	if st.placeWord != SettingUnknown {
		place += 2
	}
	return place > 0 && place >= person
}

// speaker is who a speech verb's words name.
type speaker struct {
	verb   int // the speech verb's token
	name   string
	slot   string // the same speaker, for the turns: "his wife" and "his lady" are one
	by     Attribution
	manner string
}

// near finds who the speech verb at token v names: the name, pronoun or
// description right after it ("said Anna", "dit-il", "said his wife")
// or right before it ("Anna said"); else the sentence's first name before
// it ("José Dias, after a while, said that…").
func (r *reader) near(p *para, v int) (speaker, bool) {
	sp, ok := r.nearVerb(p, v)
	sp.verb = v
	return sp, ok
}

func (r *reader) nearVerb(p *para, v int) (speaker, bool) {
	l := r.l
	t := &p.toks[v]
	manner, _ := l.isSpeech(t)
	sp := speaker{manner: manner}
	narr := func(k int) bool { return k >= 0 && k < len(p.toks) && !p.speechAt(k) }
	// dit-il, murmura-t-il, disse-lhe
	for _, part := range t.parts[1:] {
		switch {
		case l.pronouns[part]:
			sp.by = ByPronoun
			return r.adverb(p, v+1, sp), true
		case l.narrator[part]:
			sp.name, sp.slot, sp.by = Narrator, Narrator, ByVerb
			return r.adverb(p, v+1, sp), true
		}
	}
	if k := v + 1; narr(k) && !sentenceEnd(t.after) && strings.TrimSpace(t.after) != "-" {
		// "respondis al li la pli juna filino Marion", "said to him"
		if k+1 < len(p.toks) && l.prepositions[p.toks[k].low] && !p.toks[k+1].cap && len([]rune(p.toks[k+1].text)) <= 3 &&
			narr(k+2) && !sentenceEnd(p.toks[k].after) && !sentenceEnd(p.toks[k+1].after) {
			k += 2
		}
		if s, ok := r.who(p, k, true); ok {
			s.manner = manner
			return r.adverb(p, k+1, s), true
		}
	}
	if k := v - 1; narr(k) && !sentenceEnd(p.toks[k].after) && !strings.Contains(p.toks[k].after, ",") {
		if s, ok := r.whoBefore(p, k); ok {
			s.manner = manner
			return r.adverb(p, v+1, s), true
		}
	}
	// the sentence's subject: the nearest name before the verb ("José
	// Dias, after a while, said that…"), or a pronoun nearer ("Hy …, en
	// zeide:")
	for k := v - 1; k >= 0 && narr(k); k-- {
		for _, m := range p.mentions {
			if m.to == k+1 && !m.inSpeech && r.isName(m.name) && !r.isPlace(m.name) {
				return r.adverb(p, v+1, speaker{name: m.name, slot: r.slot(m.name), by: ByVerb, manner: manner}), true
			}
		}
		if l.pronouns[p.toks[k].low] {
			sp.by = ByPronoun
			return r.adverb(p, v+1, sp), true
		}
		if p.toks[k].sent {
			break
		}
	}
	return sp, false
}

// adverb adds the manner an adverb near the verb gives ("cried his wife,
// impatiently"): one of the next few words of the sentence.
func (r *reader) adverb(p *para, k int, sp speaker) speaker {
	if sp.manner != "" {
		return sp
	}
	for x := k; x < len(p.toks) && x < k+4 && !p.speechAt(x); x++ {
		t := &p.toks[x]
		for _, suf := range r.l.mannerSuffix {
			if strings.HasSuffix(t.low, suf) && len([]rune(t.low)) > len([]rune(suf))+3 && !r.l.notManner[t.low] && !t.cap {
				sp.manner = t.low
				return sp
			}
		}
		if sentenceEnd(t.after) {
			break
		}
	}
	return sp
}

// who is the speaker named starting at token k: a name, a pronoun, the
// narrator, or a description (his wife, la doktoro, minha mãe).
func (r *reader) who(p *para, k int, after bool) (speaker, bool) {
	l := r.l
	t := &p.toks[k]
	switch {
	case l.narrator[t.low]:
		return speaker{name: Narrator, slot: Narrator, by: ByVerb}, true
	case l.pronouns[t.low]:
		return speaker{by: ByPronoun}, true
	}
	if m, ok := r.nameAt(p, k); ok {
		return speaker{name: m.name, slot: r.slot(m.name), by: ByVerb}, true
	}
	if l.possessives[t.low] || l.articles[t.low] || len(t.parts) > 1 && l.articles[t.parts[0]] {
		end := k + 1
		if len(t.parts) > 1 && l.articles[t.parts[0]] { // l’hôte
			end = k + 1
		} else {
			for end < len(p.toks) && end < k+5 && !p.speechAt(end) && !l.prepositions[p.toks[end].low] &&
				!sentenceEnd(p.toks[end-1].after) && !strings.Contains(p.toks[end-1].after, ",") {
				if _, verb := l.isSpeech(&p.toks[end]); verb {
					break
				}
				if m, ok := r.nameAt(p, end); ok { // "lia filino Grace": Grace
					return speaker{name: m.name, slot: r.slot(m.name), by: ByVerb}, true
				}
				end++
			}
			end = min(end, k+4)
			if end == k+1 {
				return speaker{}, false
			}
		}
		if m, ok := r.nameAt(p, end-1); ok && end-1 > k { // la doktoro Jeddler
			end = m.to
		}
		name := p.Text[t.start:p.toks[end-1].end]
		slot := strings.ToLower(name)
		if l.possessives[t.low] {
			slot = "poss:" + t.low // "his lady", then "his wife": the same one
		}
		return speaker{name: name, slot: slot, by: ByVerb}, true
	}
	return speaker{}, false
}

// whoBefore is the speaker named just before a speech verb, ending at
// token k: "Anna said", "Mr. Bennet replied", "she said", "his wife said".
func (r *reader) whoBefore(p *para, k int) (speaker, bool) {
	l := r.l
	t := &p.toks[k]
	switch {
	case l.narrator[t.low]:
		return speaker{name: Narrator, slot: Narrator, by: ByVerb}, true
	case l.pronouns[t.low]:
		return speaker{by: ByPronoun}, true
	}
	for _, m := range p.mentions {
		if m.to == k+1 && !m.inSpeech && r.isName(m.name) {
			return speaker{name: m.name, slot: r.slot(m.name), by: ByVerb}, true
		}
	}
	for back := 1; back <= 3 && k-back >= 0; back++ {
		d := &p.toks[k-back]
		if d.seg != t.seg || sentenceEnd(d.after) {
			break
		}
		if l.possessives[d.low] || l.articles[d.low] {
			if s, ok := r.who(p, k-back, false); ok && s.slot != "" && strings.HasSuffix(s.name, t.text) {
				return s, true
			}
		}
	}
	return speaker{}, false
}

// slot is the speaker a name stands for in the turns, after the hints'
// aliases.
func (r *reader) slot(name string) string {
	if c, ok := r.hints.Aliases[name]; ok {
		return c
	}
	return name
}

// verbIn is the first speech verb in tokens [from, to) of narration.
func (r *reader) verbIn(p *para, from, to int) int {
	for k := max(from, 0); k < to && k < len(p.toks); k++ {
		if p.speechAt(k) {
			continue
		}
		if _, ok := r.l.isSpeech(&p.toks[k]); ok && !p.toks[k].cap || ok && p.toks[k].sent {
			return k
		}
	}
	return -1
}

// segTokens is the range of tokens [from, to) in segment s.
func (p *para) segTokens(s int) (from, to int) {
	from, to = -1, -1
	for k, t := range p.toks {
		if t.seg == s {
			if from < 0 {
				from = k
			}
			to = k + 1
		}
	}
	return from, to
}

// attribute finds who says segment s, from the narration next to it: the
// rest of the sentence after it, else the sentence before it. used is the
// narration segment it took (-1 none).
func (r *reader) attribute(p *para, s int, taken map[int]bool) (sp speaker, used int, ok bool) {
	// after: the next segment, to its sentence's end
	if s+1 < len(p.segs) && !p.segs[s+1].speech {
		from, to := p.segTokens(s + 1)
		if from >= 0 {
			end := from
			for end < to && (end == from || !sentenceEnd(p.toks[end-1].after)) {
				end++
			}
			if v := r.verbIn(p, from, end); v >= 0 {
				if sp, ok := r.near(p, v); ok {
					tagEnd := p.segs[s+1].end
					if end < to {
						tagEnd = p.toks[end].start
					}
					p.tag(s+1, p.segs[s+1].start, tagEnd)
					return sp, s + 1, true
				}
			}
		}
	}
	// before: the segment before, from its last sentence's start; not one
	// the line before took, if it ends its sentence ("…«, sagte Hulda.
	// »Nicht doch…«" is someone else)
	if s > 0 && !p.segs[s-1].speech {
		if taken[s-1] && strings.ContainsAny(strings.TrimSpace(p.Text[p.segs[s-1].start:p.segs[s-1].end]), ".!?") {
			return speaker{}, -2, false
		}
		from, to := p.segTokens(s - 1)
		if from >= 0 {
			start := to - 1
			for start > from && !p.toks[start].sent {
				start--
			}
			if v := r.verbIn(p, start, to); v >= 0 {
				if sp, ok := r.near(p, v); ok {
					p.tag(s-1, p.toks[start].start, p.segs[s-1].end)
					return sp, s - 1, true
				}
			}
		}
	}
	return speaker{}, -1, false
}

// tag notes the bytes [from, to) of narration segment s as the sentence
// that says who speaks.
func (p *para) tag(s, from, to int) {
	if p.tags == nil {
		p.tags = map[int][2]int{}
	}
	p.tags[p.segs[s].start] = [2]int{from, to}
}

// splitTags parts the narration pieces at the sentences that say who
// speaks, those marked as tags.
func splitTags(p *para, pieces []Piece) []Piece {
	if len(p.tags) == 0 {
		return pieces
	}
	var out []Piece
	for _, pc := range pieces {
		t, ok := p.tags[pc.Start]
		if pc.Speech || !ok {
			out = append(out, pc)
			continue
		}
		for _, part := range [][3]int{{pc.Start, t[0], 0}, {t[0], t[1], 1}, {t[1], pc.End, 0}} {
			if text := strings.TrimSpace(p.Text[part[0]:part[1]]); text != "" {
				out = append(out, Piece{Text: text, Start: part[0], End: part[1], Line: -1, Tag: part[2] == 1})
			}
		}
	}
	return out
}

// dialogue finds a passage's lines and who says them.
func (r *reader) dialogue(i int, p *para, out *PassageReading) {
	if p.Kind == Heading || p.Kind == Break {
		return
	}
	// pieces and lines: what is said is one line until a new speech verb
	// names someone, or a sentence of narration parts it from the last
	taken := map[int]bool{}
	type said struct {
		sp     speaker
		ok     bool
		action bool // narration of its own comes before it
	}
	line, lineVerb := -1, -1
	var attr []said // the lines'
	for s, sg := range p.segs {
		pc := Piece{Speech: sg.speech, Text: strings.TrimSpace(p.Text[sg.start:sg.end]), Start: sg.start, End: sg.end, Line: -1}
		if sg.speech {
			pc.Text = trimLine(pc.Text)
			sp, used, ok := r.attribute(p, s, taken)
			// action between: "»Nicht so wild…« Und die Mama schien… »Ich habe…«"
			action := s > 0 && !p.segs[s-1].speech && used != s-1 && !taken[s-1] &&
				strings.ContainsAny(p.Text[p.segs[s-1].start:p.segs[s-1].end], ".!?")
			if used >= 0 {
				taken[used] = true
			}
			split := line < 0 || used == -2 || action || ok && lineVerb >= 0 && sp.verb != lineVerb ||
				ok && lineVerb < 0 && attr[line].ok
			if split {
				out.Lines = append(out.Lines, Line{Start: sg.start})
				attr = append(attr, said{action: action && line >= 0})
				line, lineVerb = len(out.Lines)-1, -1
			}
			ln, a := &out.Lines[line], &attr[line]
			ln.Text = strings.TrimSpace(ln.Text + " " + pc.Text)
			ln.End = sg.end
			if ok && (!a.ok || a.sp.by == ByPronoun && sp.by == ByVerb) {
				a.sp, a.ok = sp, true
				lineVerb = sp.verb
			}
			if ln.Manner == "" {
				ln.Manner = sp.manner
			}
			for _, m := range p.mentions {
				if m.vocative && p.toks[m.from].seg == s && r.isName(m.name) && !slices.Contains(ln.Addressed, m.name) {
					ln.Addressed = append(ln.Addressed, m.name)
				}
			}
			pc.Line = line
		}
		if pc.Text != "" {
			out.Pieces = append(out.Pieces, pc)
		}
	}
	out.Pieces = splitTags(p, out.Pieces)
	if len(out.Lines) == 0 {
		r.narrationTurn(i, p)
		return
	}
	// several speakers named in the paragraph: each line's own, or the
	// turns; one: it is the paragraph's
	named := map[string]bool{}
	for _, a := range attr {
		if a.ok {
			named[a.sp.slot] = true
		}
	}
	multi := len(named) > 1 || len(named) == 1 && named[""] && len(attr) > 1
	for li := range out.Lines {
		ln, a := &out.Lines[li], attr[li]
		switch {
		case a.ok && a.sp.by == ByVerb:
			ln.Speaker, ln.By = a.sp.name, ByVerb
			r.labels[a.sp.slot] = a.sp.name
		case !multi && !a.ok:
			for _, o := range attr {
				if o.ok && o.sp.by == ByVerb {
					ln.Speaker, ln.By = o.sp.name, ByParagraph
					break
				}
			}
			if ln.By == Unknown {
				if name, ok := r.beat(p); ok {
					ln.Speaker, ln.By = name, ByBeat
				} else if sp, ok := r.intro(i); ok && li == 0 && sp.slot != "" {
					ln.Speaker, ln.By = sp.name, ByIntro
					r.labels[sp.slot] = sp.name
				}
			}
		}
		if ln.By == Unknown {
			by := ByTurn
			if a.ok && a.sp.by == ByPronoun {
				by = ByPronoun
			}
			if name, ok := r.turn(ln.Addressed); ok {
				ln.Speaker, ln.By = name, by
			}
		}
		if ln.Speaker != "" {
			r.push(r.slotOf(ln.Speaker))
			if ln.By == ByVerb && ln.Speaker != Narrator {
				r.stat(ln.Speaker, i).spoke++
			}
		}
		r.called = ""
		if len(ln.Addressed) > 0 {
			r.called = r.slot(ln.Addressed[0])
		}
	}
	// one speaker's lines side by side are one line
	for li := 1; li < len(out.Lines); li++ {
		a, b := &out.Lines[li-1], out.Lines[li]
		if a.Speaker != "" && a.Speaker == b.Speaker && !attr[li].action {
			a.Text = strings.TrimSpace(a.Text + " " + b.Text)
			a.End = b.End
			if a.Manner == "" {
				a.Manner = b.Manner
			}
			a.Addressed = append(a.Addressed, b.Addressed...)
			out.Lines = slices.Delete(out.Lines, li, li+1)
			attr = slices.Delete(attr, li, li+1)
			for pi := range out.Pieces {
				if out.Pieces[pi].Line >= li {
					out.Pieces[pi].Line--
				}
			}
			li--
		}
	}
}

// slotOf is the turn slot of a speaker's label.
func (r *reader) slotOf(label string) string {
	for s, l := range r.labels {
		if l == label {
			return s
		}
	}
	return r.slot(label)
}

// push makes slot the latest speaker.
func (r *reader) push(slot string) {
	r.turns = slices.DeleteFunc(r.turns, func(s string) bool { return s == slot })
	r.turns = append([]string{slot}, r.turns...)
	if len(r.turns) > 2 {
		r.turns = r.turns[:2]
	}
	if _, ok := r.labels[slot]; !ok {
		r.labels[slot] = slot
	}
}

// turn is who says a line no one is named for: the other of the two who
// were talking, or the one the last line called; not someone the line
// itself calls.
func (r *reader) turn(addressed []string) (string, bool) {
	cands := []string{}
	if len(r.turns) >= 2 {
		cands = append(cands, r.turns[1])
	}
	if r.called != "" && (len(r.turns) == 0 || r.called != r.turns[0]) {
		cands = append(cands, r.called)
	}
	for _, c := range cands {
		if !slices.ContainsFunc(addressed, func(a string) bool { return r.slot(a) == c }) {
			if l, ok := r.labels[c]; ok {
				return l, true
			}
			return c, true
		}
	}
	return "", false
}

// beat is the name a dialogue paragraph's narration opens a sentence
// with: "Anna looked up. “Who's there?”"
func (r *reader) beat(p *para) (string, bool) {
	for _, m := range p.mentions {
		if !m.inSpeech && m.initial && r.isName(m.name) && !r.isPlace(m.name) {
			return m.name, true
		}
	}
	return "", false
}

// intro is who the paragraph before introduced, when it ends with a colon
// after a speech verb: "les dijo:".
func (r *reader) intro(i int) (speaker, bool) {
	if i == 0 {
		return speaker{}, false
	}
	p := r.ps[i-1]
	if p.Kind != Para || p.dash || !strings.HasSuffix(strings.TrimSpace(p.Text), ":") {
		return speaker{}, false
	}
	start := len(p.toks) - 1
	for start > 0 && !p.toks[start].sent {
		start--
	}
	for k := len(p.toks) - 1; k >= start; k-- {
		if _, ok := r.l.isSpeech(&p.toks[k]); ok && !p.speechAt(k) {
			return r.near(p, k)
		}
	}
	return speaker{}, false
}

// narrationTurn: a paragraph without dialogue that tells someone spoke
// ("Mr. Bennet replied that he had not.") is their turn; a long one ends
// the talk.
func (r *reader) narrationTurn(i int, p *para) {
	if p.Kind != Para && p.Kind != DashItem {
		return
	}
	for k := range p.toks {
		if _, ok := r.l.isSpeech(&p.toks[k]); ok && !p.toks[k].cap {
			if sp, ok := r.near(p, k); ok && sp.slot != "" {
				r.labels[sp.slot] = sp.name
				r.push(sp.slot)
				return
			}
		}
	}
	if len(p.toks) > 120 {
		r.turns, r.called = nil, ""
	}
}

// flags are a passage's narration sentences with a thought or a feeling.
func (r *reader) flags(p *para) []Flag {
	var out []Flag
	if p.Kind != Para && p.Kind != DashItem {
		return nil
	}
	lastEnd := -1
	for k := range p.toks {
		t := &p.toks[k]
		if p.speechAt(k) || !r.l.thought.in(t) || t.start < lastEnd {
			continue
		}
		start := k
		for start > 0 && !p.toks[start].sent && !p.speechAt(start-1) {
			start--
		}
		end := k
		for end+1 < len(p.toks) && !p.toks[end+1].sent && !p.speechAt(end+1) {
			end++
		}
		a, b := p.toks[start].start, p.toks[end].end
		if rest := p.toks[end].after; strings.ContainsAny(rest, ".!?…") {
			b += strings.IndexAny(rest, ".!?…") + 1
		}
		out = append(out, Flag{Sentence: strings.TrimSpace(p.Text[a:b]), Word: t.text, Start: a, End: b})
		lastEnd = b
	}
	return out
}

// names makes the characters and places, and lists each passage's.
func (r *reader) names(out *Reading) {
	for name, st := range r.stats {
		if !r.isName(name) {
			continue
		}
		if r.isPlace(name) {
			out.Places = append(out.Places, PlaceName{Name: name, Mentions: st.mentions, First: st.first, Setting: st.placeWord})
			continue
		}
		w := float64(st.mentions + 3*st.spoke + 2*st.vocative + 2*st.titled)
		out.Characters = append(out.Characters, Character{Name: name, Mentions: st.mentions, Lines: st.spoke,
			Addressed: st.vocative, First: st.first, Weight: w})
	}
	sort.Slice(out.Characters, func(a, b int) bool {
		x, y := out.Characters[a], out.Characters[b]
		if x.Weight != y.Weight {
			return x.Weight > y.Weight
		}
		return x.First < y.First || x.First == y.First && x.Name < y.Name
	})
	sort.Slice(out.Places, func(a, b int) bool {
		x, y := out.Places[a], out.Places[b]
		if x.Mentions != y.Mentions {
			return x.Mentions > y.Mentions
		}
		return x.First < y.First || x.First == y.First && x.Name < y.Name
	})
	for i, p := range r.ps {
		pr := &out.Passages[i]
		for _, m := range p.mentions {
			if !r.isName(m.name) {
				continue
			}
			list := &pr.Names
			if r.isPlace(m.name) {
				list = &pr.Places
			}
			if !slices.Contains(*list, m.name) {
				*list = append(*list, m.name)
			}
		}
	}
}

var roman = regexp.MustCompile(`^(?i)[IVXLCDM]+\.?$`)
var numbered = regexp.MustCompile(`^(?i)[IVXLCDM]+\.?$|^\d+\.?$`)

// chapterLine reports whether a paragraph reads as a chapter's heading
// though written as text: "CHAPTER I.", "III.", "Capítulo 3".
func (r *reader) chapterLine(p *para) bool {
	t := strings.TrimSpace(p.Text)
	if p.Kind != Para || len(t) > 40 {
		return false
	}
	if roman.MatchString(t) {
		return true
	}
	f := strings.Fields(t)
	return len(f) >= 1 && len(f) <= 3 && r.l.chapter[key(f[0])] && (len(f) == 1 || numbered.MatchString(f[1]))
}

// shiftAt is the shift phrase a paragraph opens with, and the time it
// gives.
func (r *reader) shiftAt(p *para) (words, time string, ok bool) {
	for _, ph := range r.l.shift {
		if len(p.toks) < len(ph.words) || p.speechAt(0) {
			continue
		}
		match := true
		for x, w := range ph.words {
			if p.toks[x].low != w {
				match = false
				break
			}
		}
		if match {
			last := p.toks[len(ph.words)-1]
			return p.Text[p.toks[0].start:last.end], ph.value, true
		}
	}
	return "", "", false
}

// scenes parts the passages into scenes: at headings, breaks, chapter
// lines and shift phrases. A scene that has nothing in it yet takes the
// next start's reason rather than becoming one of its own.
func (r *reader) scenes(out *Reading) {
	content := false
	for i, p := range r.ps {
		why := ""
		switch {
		case p.Kind == Heading:
			why = "chapter heading"
		case p.Kind == Break:
			why = "break"
		case r.chapterLine(p):
			why = "chapter heading"
		default:
			if words, _, ok := r.shiftAt(p); ok {
				why = words
			}
		}
		if i == 0 && why == "" {
			why = "the start"
		}
		if why != "" {
			if content || len(out.Scenes) == 0 {
				out.Scenes = append(out.Scenes, Scene{First: i, Why: why})
				out.Passages[i].SceneStart = why
				content = false
			} else {
				out.Scenes[len(out.Scenes)-1].Why = why
				out.Passages[out.Scenes[len(out.Scenes)-1].First].SceneStart = why
			}
		}
		if p.Kind == Para && (len(p.toks) >= 8 || len(p.segs) > 1 || len(p.segs) == 1 && p.segs[0].speech) || p.Kind == DashItem {
			content = true
		}
		out.Scenes[len(out.Scenes)-1].Last = i
		out.Passages[i].Scene = len(out.Scenes) - 1
	}
}

// headings suggests each scene's heading: inside or outside from the
// first place word in its narration (room, street), the place by name if
// one is there (Netherfield Park), the time from its shift or its first
// time word.
func (r *reader) headings(out *Reading) {
	places := map[string]Setting{}
	for _, pl := range out.Places {
		places[pl.Name] = pl.Setting
	}
	for si := range out.Scenes {
		sc := &out.Scenes[si]
		word := ""
		for i := sc.First; i <= sc.Last; i++ {
			p := r.ps[i]
			if i == sc.First {
				if _, t, ok := r.shiftAt(p); ok && t != "" {
					sc.Time = t
				}
			}
			for k := range p.toks {
				t := &p.toks[k]
				if p.speechAt(k) || p.Kind != Para && p.Kind != DashItem {
					continue
				}
				if sc.Setting == SettingUnknown {
					switch {
					case r.l.interior.in(t):
						sc.Setting, word = Inside, t.text
					case r.l.exterior.in(t):
						sc.Setting, word = Outside, t.text
					}
				}
				if sc.Time == "" {
					if tm, ok := r.l.time[t.low]; ok {
						sc.Time = tm
					}
				}
			}
			if sc.Place == "" {
				for _, m := range p.mentions {
					if st, ok := places[m.name]; ok && !m.inSpeech {
						sc.Place = strings.ToUpper(m.name)
						if st != SettingUnknown {
							sc.Setting = st
						}
						break
					}
				}
			}
		}
		if sc.Place == "" && word != "" {
			sc.Place = strings.ToUpper(word)
		}
	}
}

// Heading is a scene's suggested heading with the script's words for
// inside, outside and either ("INT.", "EXT.", "INT./EXT."): PLACE where
// none was found, the time left out where none was.
func (s Scene) Heading(inside, outside, either string) string {
	h := either
	switch s.Setting {
	case Inside:
		h = inside
	case Outside:
		h = outside
	}
	place := s.Place
	if place == "" {
		place = "PLACE"
	}
	h += " " + place
	if s.Time != "" {
		h += " - " + s.Time
	}
	return h
}

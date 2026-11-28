// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/structure"
)

// The units inside a scene as text objects and motions (vim_structure.go
// has the scenes' and sections'), read by the structure engine and the
// script's language as those are:
//
//	ad id   a speech: its cue, parentheticals and dialogue (two side by
//	        side: the pair) / its dialogue lines alone
//	ac ic   a cue: with its extension, (V.O.) or (CONT'D) / its name
//	aB iB   a beat (beatsOf: a synopsis standing on its own, a section
//	        marked a beat, every section with Every Section Is a Beat):
//	        with the blank lines after it / its lines
//	an in   a note: with its [[ ]] / inside them
//	aa ia   an action paragraph: with the blank lines after it / alone
//	]d [d   the next / previous speech
//	]B [B   the next / previous beat
//	]n [n   the next / previous note
//
// helix: md, mc, mB, mn, ma (after m: i or a) select them; the motions
// select the unit they go to; M selects every speech of the cue's
// character (helix_multi.go).

// objSpan is what an object takes: lines (first to last, whole lines) or,
// char, runes [from, to).
type objSpan struct {
	first, last int
	from, to    int
	char        bool
	unit        rune // what p puts it as: 's', 'S' a scene or section, 'p' a block; 0 plain
	head        int  // a line-wise inside's heading line (ci on an empty one types after it); -1 none
}

// objectWords name the objects' letters for messages.
var objectWords = map[rune]string{'s': "scene", 'S': "section", 'd': "speech", 'c': "cue", 'B': "beat", 'n': "note", 'a': "action paragraph"}

// objectLetters are the objects, as the message for an unknown one has them.
const objectLetters = "as is aS iS ad id ac ic aB iB an in aa ia"

// beatsNow are the script's beats, the window's way (its outline and
// Every Section Is a Beat), else read plainly.
func (v *vim) beatsNow() []beatAt {
	if v.beats != nil {
		if b := v.beats(); b != nil {
			return b
		}
	}
	text := v.e.Text()
	return scriptBeats(text, outlineOf(text, nil), false)
}

// curAt is the offset the keys act from, as curLine's line.
func (v *vim) curAt() int {
	switch {
	case v.helix != nil && v.helix():
		at, _ := v.e.Buffer().Selection()
		return at
	case v.visual():
		return v.vcur
	}
	return v.e.CursorOffset()
}

// lineSpan is lines first to last as a span; a: with the blank lines after.
func lineSpan(tr *structure.Tree, first, last int, a bool, unit rune) objSpan {
	if a {
		last = tr.Trail(last)
	}
	return objSpan{first: first, last: last, unit: unit, head: -1}
}

// objectSpan is the object ia+r at the cursor, n of them (a count); ""
// when there is one, else why not.
func (v *vim) objectSpan(ia, r rune, n int) (objSpan, string) {
	tr := v.unitTree()
	cur := v.curLine()
	a := ia == 'a'
	switch r {
	case 's', 'S':
		k, _ := unitKindOf(r)
		first, last, ok := tr.SpanOf(cur, k, n)
		if !ok {
			return objSpan{}, "Not in a " + unitWord(k)
		}
		if a {
			return lineSpan(tr, first, last, true, r), ""
		}
		return v.inside(tr, first, last), ""
	case 'd':
		sp, ok := tr.SpeechAt(cur)
		if !ok {
			return objSpan{}, "Not in a speech"
		}
		if !a {
			return objSpan{first: sp.InnerFirst, last: sp.Last, head: -1}, ""
		}
		first, last := sp.Block[0], sp.Block[1]
		for m := 1; m < n; m++ { // the next speeches, right after it
			nx, ok := tr.BlockAfter(last)
			if !ok {
				break
			}
			s2, ok := tr.SpeechAt(nx)
			if !ok || s2.Block[0] != nx || tr.At(nx) != tr.At(first) {
				break
			}
			last = s2.Block[1]
		}
		return lineSpan(tr, first, last, true, 'p'), ""
	case 'a':
		first, last, ok := tr.Paragraph(cur)
		if !ok {
			return objSpan{}, "Not in an action paragraph"
		}
		for m := 1; m < n; m++ {
			nx, ok := tr.BlockAfter(last)
			if !ok {
				break
			}
			f2, l2, ok := tr.Paragraph(nx)
			if !ok || f2 != nx || tr.At(nx) != tr.At(first) {
				break
			}
			last = l2
		}
		return lineSpan(tr, first, last, a, unitIf(a, 'p')), ""
	case 'B':
		return v.beatSpan(tr, cur, a)
	case 'c':
		sp, ok := tr.SpeechAt(cur)
		if !ok {
			return objSpan{}, "Not in a speech: ac and ic take its cue"
		}
		b := v.e.Buffer()
		line := tr.S.Lines[sp.Cue]
		pre, name, _ := structure.CueParts(line)
		at := b.LineStart(sp.Cue)
		from := at + utf8.RuneCountInString(pre)
		if a {
			if strings.HasSuffix(pre, "@") {
				from--
			}
			return objSpan{from: from, to: at + utf8.RuneCountInString(strings.TrimRight(line, " \t")), char: true}, ""
		}
		return objSpan{from: from, to: from + utf8.RuneCountInString(name), char: true}, ""
	case 'n':
		c := v.curAt()
		ns := v.notes()
		l, _ := v.e.Buffer().Position(c)
		for _, s := range ns {
			if c >= s.start && c < s.end || s.start > c && v.lineOf(s.start) == l { // in it, or the next on the line
				if a {
					return objSpan{from: s.start, to: s.end, char: true}, ""
				}
				t := []rune(v.e.Text())
				from, to := s.start+2, s.end-2
				for from < to && (t[from] == ' ' || t[from] == '\t') {
					from++
				}
				for to > from && (t[to-1] == ' ' || t[to-1] == '\t') {
					to--
				}
				return objSpan{from: from, to: to, char: true}, ""
			}
		}
		return objSpan{}, "Not in a note"
	}
	return objSpan{}, "No text object " + string(ia) + string(r) + ": the screenplay's are " + objectLetters
}

func unitIf(a bool, r rune) rune {
	if a {
		return r
	}
	return 0
}

// inside is a scene's or section's lines after its heading, the blank
// lines after the heading left out.
func (v *vim) inside(tr *structure.Tree, first, last int) objSpan {
	head := first
	first++
	for first <= last && strings.TrimSpace(tr.S.Lines[first]) == "" {
		first++
	}
	return objSpan{first: first, last: last, head: head}
}

// beatSpan is the beat at line cur: a synopsis's lines; a section
// marked a beat: the section with all it holds (aB), its inside (iB).
func (v *vim) beatSpan(tr *structure.Tree, cur int, a bool) (objSpan, string) {
	best, size := -1, 0
	bs := v.beatsNow()
	for i, b := range bs {
		first, last := b.from, b.to-1
		if b.sect {
			f, l, ok := tr.Span(b.section, structure.Section)
			if !ok {
				continue
			}
			first, last = f, l
		}
		if cur >= first && cur <= last && (best < 0 || last-first < size) {
			best, size = i, last-first
		}
	}
	if best < 0 {
		return objSpan{}, "Not in a beat"
	}
	b := bs[best]
	if b.sect {
		f, l, _ := tr.Span(b.section, structure.Section)
		if a {
			return lineSpan(tr, f, l, true, 'S'), ""
		}
		return v.inside(tr, f, l), ""
	}
	return lineSpan(tr, b.from, b.to-1, a, unitIf(a, 'p')), ""
}

// noteAt is a note's runes [start, end), its [[ and ]] in.
type noteAt struct{ start, end int }

// notes are the script's notes in runes (outside the boneyards and the
// blocks at the end).
func (v *vim) notes() []noteAt {
	text := v.e.Text()
	end := fountainBodyEnd(text)
	var out []noteAt
	toRune := byteToRune(text)
	for _, s := range noteSpans(text, 0, end, boneyards(text, end)) {
		out = append(out, noteAt{toRune(s.start), toRune(s.end)})
	}
	return out
}

// byteToRune converts byte offsets of text, asked in increasing order,
// to rune offsets.
func byteToRune(text string) func(int) int {
	b, r := 0, 0
	return func(at int) int {
		if at < b {
			b, r = 0, 0
		}
		r += utf8.RuneCountInString(text[b:at])
		b = at
		return r
	}
}

// lineOf is offset at's line.
func (v *vim) lineOf(at int) int {
	l, _ := v.e.Buffer().Position(at)
	return l
}

// doObject does f with the object ia+r: d c y, m, v, or h (helix's select).
func (v *vim) doObject(ia, f, r rune) {
	n := v.times()
	s, why := v.objectSpan(ia, r, n)
	if why != "" {
		v.say(why)
		return
	}
	b := v.e.Buffer()
	t := []rune(v.e.Text())
	if !s.char && s.first > s.last { // an empty inside
		if f == 'c' && s.head >= 0 {
			v.edit(func(b *buffer.Buffer, _ []rune, _ int) {
				at := b.LineEnd(s.head)
				b.Replace(at, at, "\n\n")
				b.SetCursor(at+2, false)
			})
			v.setMode(vimInsert)
			return
		}
		v.say("This " + objectWords[r] + " has nothing inside")
		return
	}
	from, to := s.from, s.to
	if !s.char {
		from, to = b.LineStart(s.first), b.LineEnd(s.last)
	}
	if v.noticed != nil {
		v.noticed("obj:"+string(ia)+string(r), v.curLine())
	}
	switch f {
	case 'd', 'c', 'y':
		v.apply(f, t, from, to, !s.char)
		if s.unit != 0 && f != 'c' {
			v.unitKind, v.unitText = s.unit, v.reg
		}
	case 'm':
		switch {
		case s.char:
			v.say("A " + objectWords[r] + " moves with its line: mad moves the speech, maa the paragraph")
		case r == 's' || r == 'S' || r == 'B' && (s.unit == 'S' || s.head >= 0): // a scene or section: :move
			if v.exWith == nil {
				v.say("Moving needs the window's command line")
				return
			}
			v.setMode(vimNormal)
			v.exWith([2]int{s.first, s.last}, "move ")
		default: // a block: selected, for Alt+j and Alt+k, the engine's moves
			v.selectLines(s, r)
			v.say("-- VISUAL LINE -- Alt+j / Alt+k move the " + objectWords[r] + ", Alt+J / Alt+K into the next / previous scene")
		}
	case 'v':
		if s.char {
			v.anchor, v.vcur = from, max(to-1, from)
			v.visualUnit = [3]int{0, 0, 0}
			v.mode = vimVisual
			v.select_()
			v.say(v.mode.String())
			return
		}
		v.selectLines(s, r)
		v.say(v.mode.String())
	case 'h':
		end := to
		if !s.char {
			end = min(to+1, len(t))
		}
		v.e.Navigate(func(bf *buffer.Buffer) { bf.Select(from, end) })
		v.helixUnit = [3]int{0, 0, 0}
		if s.unit != 0 {
			v.helixUnit = [3]int{int(s.unit), from, end}
		}
	}
}

// selectLines selects span s line-wise in vim's visual line mode.
func (v *vim) selectLines(s objSpan, r rune) {
	b := v.e.Buffer()
	v.anchor, v.vcur = b.LineStart(s.first), b.LineStart(s.last)
	v.visualUnit = [3]int{0, 0, 0}
	if s.unit != 0 {
		v.visualUnit = [3]int{int(s.unit), v.anchor, v.vcur}
	}
	v.mode = vimVisualLine
	v.select_()
}

// stop is where a motion can go: at (a rune offset) and the unit helix
// selects there, [from, to).
type stop struct{ at, from, to int }

// stopsOf are the stops of r's motion (]d: 'd', ]B: 'B', ]n: 'n'), in
// order, and their unit's word.
func (v *vim) stopsOf(r rune) []stop {
	b := v.e.Buffer()
	t := []rune(v.e.Text())
	lines := func(first, last int) stop {
		return stop{b.LineStart(first), b.LineStart(first), min(b.LineEnd(last)+1, len(t))}
	}
	var out []stop
	switch r {
	case 'd':
		for _, s := range v.unitTree().Speeches() {
			out = append(out, lines(s.Cue, s.Last))
		}
	case 'B':
		tr := v.unitTree()
		for _, bt := range v.beatsNow() {
			first, last := bt.from, bt.to-1
			if bt.sect {
				if f, l, ok := tr.Span(bt.section, structure.Section); ok {
					first, last = f, l
				}
			}
			out = append(out, lines(first, last))
		}
	case 'n':
		for _, n := range v.notes() {
			out = append(out, stop{n.start, n.start, n.end})
		}
	}
	return out
}

// unitMotion completes ]d [d ]B [B ]n [n: back for [, n times.
func (v *vim) unitMotion(r rune, back bool, n int) {
	stops := v.stopsOf(r)
	c := v.curAt()
	byLine := r != 'n' // speeches and beats from the line, as ]]: [d in a speech goes to its cue
	if byLine {
		c = v.e.Buffer().LineStart(v.curLine())
	}
	at := -1
	for i, s := range stops {
		if back && s.at < c {
			at = i
		}
		if !back && s.at > c && at < 0 {
			at = i
		}
	}
	if at < 0 {
		where := "after this"
		if back {
			where = "before this"
		}
		v.say("No " + objectWords[r] + " " + where)
		return
	}
	if back {
		at = max(at-(n-1), 0)
	} else {
		at = min(at+(n-1), len(stops)-1)
	}
	s := stops[at]
	bf := v.e.Buffer()
	switch {
	case v.helix != nil && v.helix():
		from, end := s.from, s.to
		if v.mode == vimVisual { // select mode: the selection grows to it
			a, e := bf.Selection()
			from, end = min(from, a), max(end, e)
		}
		v.e.Navigate(func(b *buffer.Buffer) { b.Select(from, end) })
		v.helixUnit = [3]int{0, 0, 0}
	case v.visual():
		v.vcur = s.at
		v.select_()
	default:
		v.e.Navigate(func(b *buffer.Buffer) { b.SetCursor(s.at, false) })
	}
	if v.noticed != nil {
		v.noticed("motion:"+string(r), v.curLine())
	}
}

// speechSpans are every speech of the character whose speech the line
// is in, each as ad takes it (a pair's half alone), for helix's M.
func (v *vim) speechSpans(line int) ([]buffer.Span, string) {
	tr := v.unitTree()
	sp, ok := tr.SpeechAt(line)
	if !ok {
		return nil, ""
	}
	b := v.e.Buffer()
	var out []buffer.Span
	for _, s := range tr.Speeches() {
		if strings.EqualFold(s.Name, sp.Name) {
			end := min(b.LineEnd(tr.Trail(s.Last))+1, b.Len())
			out = append(out, buffer.Span{Anchor: b.LineStart(s.Cue), Cursor: end})
		}
	}
	return out, sp.Name
}

package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/LaPingvino/accolade/internal/prose"
)

// What the adaptation keeps in the script (docs/ADAPTATION.md, steps 4
// and 7): a boneyard block at its end, before the source block and the
// notes, so it travels with the file and other Fountain apps ignore it:
//
//	/* Adaptation
//	cut:
//	  "The rain kept falling." (2)
//	  "She opened the door to the garden and…"
//	*/
//
// "cut:" lists the passages cut on purpose, one per line, each by its
// first words as a link quotes them (anchorFor; matched as links are,
// matchLink); a passage whose words are in the source more than once
// carries which of them it is, "(2)" for the second. The other lines of
// the block (later: characters:, places: …) are kept as they are, and so
// are cut entries that find no passage (the source may have changed, or
// not be at hand).
//
// A passage's status comes from the script: adapted if a link resolves
// to it (an ambiguous one does not), else cut if a cut entry does, else
// to do. Headings and breaks have none: they are not counted.

// adaptationMarks are the block's, as the notes' (notes.go).
var adaptationMarks = newNotesMarks("/* Adaptation", "*/", "* /")

// tailBlock is one of the boneyard blocks at a script's end: where it is
// in runes (from the blank lines before it to the next block, the notes
// or the end) and what is inside it.
type tailBlock struct {
	start, end int
	inside     string
	ok         bool
}

// tailBlocks are the script's source and adaptation blocks, the last
// things before its notes (or its end), in either order.
func tailBlocks(text string) (src, adapt tailBlock) {
	r := []rune(text)
	end := notesStart(text)
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
			if i, inside := b.marks.at(before); i >= 0 {
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

// bodyEnd is where the script proper ends, in runes: before its source,
// adaptation and notes blocks.
func bodyEnd(text string) int {
	end := notesStart(text)
	src, adapt := tailBlocks(text)
	if src.ok {
		end = min(end, src.start)
	}
	if adapt.ok {
		end = min(end, adapt.start)
	}
	return end
}

// adaptationBlock is what is inside the script's adaptation block.
func adaptationBlock(text string) (string, bool) {
	_, a := tailBlocks(text)
	return a.inside, a.ok
}

// cutMark is a cut: entry, the passage it names.
type cutMark struct {
	Quote string   // the words as written
	Nth   int      // which of the passages opening so (1, 2 …); 0: the one
	Words []string // normalised
}

// String is the entry as the block writes it.
func (c cutMark) String() string {
	q := strings.NewReplacer("\n", " ", `"`, "”").Replace(c.Quote)
	s := `"` + q + `"`
	if c.Nth > 0 {
		s += " (" + strconv.Itoa(c.Nth) + ")"
	}
	return s
}

// parseCutEntry reads an entry: a quote (straight quotes around it, or
// curly ones), or bare words, with "(2)" after it for the second of
// passages alike; a leading "-" or "*" is a list's.
func parseCutEntry(s string) (cutMark, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(strings.TrimLeft(s, "-*•"))
	nth := 0
	if strings.HasSuffix(s, ")") {
		if i := strings.LastIndex(s, "("); i >= 0 {
			if n, err := strconv.Atoi(strings.TrimSpace(s[i+1 : len(s)-1])); err == nil && n > 0 {
				nth, s = n, strings.TrimSpace(s[:i])
			}
		}
	}
	const quotes = `"“”„«»'‘’`
	if r := []rune(s); len(r) >= 2 && strings.ContainsRune(quotes, r[0]) && strings.ContainsRune(quotes, r[len(r)-1]) {
		s = string(r[1 : len(r)-1])
	}
	s = strings.TrimSpace(s) // a closing ellipsis is kept: it says the passage goes on
	if s == "" {
		return cutMark{}, false
	}
	return cutMark{Quote: s, Nth: nth, Words: normWords(s)}, len(normWords(s)) > 0
}

// splitCutEntries parts what follows "cut:" on its line: several
// entries in straight quotes ("a", "b"), else one.
func splitCutEntries(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.Count(s, `"`) >= 4 && strings.HasPrefix(s, `"`) {
		var out []string
		for s != "" {
			s = strings.TrimSpace(strings.TrimLeft(s, ", ;"))
			if !strings.HasPrefix(s, `"`) {
				if s != "" {
					out = append(out, s)
				}
				break
			}
			e := strings.Index(s[1:], `"`)
			if e < 0 {
				out = append(out, s)
				break
			}
			e += 2                                                              // past the closing quote
			if rest := strings.TrimSpace(s[e:]); strings.HasPrefix(rest, "(") { // its (n)
				if c := strings.Index(rest, ")"); c >= 0 {
					e = len(s) - len(rest) + c + 1
				}
			}
			out = append(out, s[:e])
			s = s[e:]
		}
		return out
	}
	return []string{s}
}

// isBlockKey reports whether a line of the block opens a section of it,
// "name: …" at the line's start (not an entry of the one before).
func isBlockKey(line string) (key, rest string, ok bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' || strings.ContainsRune(`-*•"“„«'‘`, []rune(line)[0]) {
		return "", "", false
	}
	k, rest, found := strings.Cut(line, ":")
	if !found || strings.ContainsAny(k, `"“”`) || len(strings.Fields(k)) > 3 {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(k)), rest, true
}

// parseCuts reads the cut: entries of the block, leniently: "cut:" (or
// "cuts:") with entries after it on its line or on the lines below it
// (indented, as a list, or quoted), up to the next "name:" line.
func parseCuts(inside string) []cutMark {
	var out []cutMark
	in := false
	for _, line := range strings.Split(inside, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if key, rest, ok := isBlockKey(line); ok {
			in = key == "cut" || key == "cuts"
			if in {
				for _, e := range splitCutEntries(rest) {
					if c, ok := parseCutEntry(e); ok {
						out = append(out, c)
					}
				}
			}
			continue
		}
		if in {
			if c, ok := parseCutEntry(line); ok {
				out = append(out, c)
			}
		}
	}
	return out
}

// withCuts is the block's inside with its cut: entries these, the rest
// of it kept as it is ("" if nothing is left).
func withCuts(inside string, cuts []cutMark) string {
	var kept []string
	at := -1 // where the cut: section was
	in := false
	for _, line := range strings.Split(inside, "\n") {
		if key, _, ok := isBlockKey(line); ok {
			in = key == "cut" || key == "cuts"
			if in {
				if at < 0 {
					at = len(kept)
				}
				continue
			}
		}
		if in && strings.TrimSpace(line) != "" {
			continue
		}
		if in { // a blank line in the section: part of it
			continue
		}
		kept = append(kept, line)
	}
	var section []string
	if len(cuts) > 0 {
		section = append(section, "cut:")
		for _, c := range cuts {
			section = append(section, "  "+c.String())
		}
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

// withAdaptationBlock is the edit that makes the script's adaptation
// block hold inside ("" takes it out): where (runes) and what. A new
// block goes after the script, before the source and notes blocks.
func withAdaptationBlock(text, inside string) (start, end int, block string) {
	_, a := tailBlocks(text)
	if a.ok {
		start, end = a.start, a.end
	} else {
		end = bodyEnd(text)
		start = len([]rune(strings.TrimRight(string([]rune(text)[:end]), "\n")))
	}
	block = strings.TrimRight(adaptationMarks.text(inside), "\n")
	if end == len([]rune(text)) { // nothing after it: the script ends with a new line
		block += "\n"
	}
	return start, end, block
}

// resolveCut is the passage a cut entry names, or -1.
func resolveCut(c cutMark, ps []passage) int {
	mt := matchLink(c.Words, ps)
	switch {
	case c.Nth > 0 && (mt.found() || mt.Kind == linkAmbiguous) && c.Nth <= len(mt.Passages):
		return mt.Passages[c.Nth-1]
	case c.Nth <= 1 && mt.found():
		return mt.Passages[0]
	}
	return -1
}

// cutFor is the entry that names passage i: its link's words, and which
// of the passages opening so it is if more do.
func cutFor(ps []passage, i int) cutMark {
	quote, unique := anchorFor(ps, i)
	c := cutMark{Quote: strings.TrimSuffix(quote, "…"), Words: normWords(quote)}
	if !unique {
		mt := matchLink(c.Words, ps)
		c.Nth = slices.Index(mt.Passages, i) + 1
	}
	if strings.HasSuffix(quote, "…") {
		c.Quote += "…"
	}
	return c
}

// passageStatus is what the script has done with a passage.
type passageStatus int

const (
	statusNone    passageStatus = iota // a heading or a break: not counted
	statusToDo                         // not started
	statusAdapted                      // a link resolves to it
	statusCut                          // cut on purpose
)

func (st passageStatus) String() string {
	switch st {
	case statusToDo:
		return "not started"
	case statusAdapted:
		return "adapted"
	case statusCut:
		return "cut on purpose"
	}
	return ""
}

// counted reports whether a passage has a status: not a heading or a
// break.
func (p passage) counted() bool { return p.linkable() && p.Kind != prose.Heading }

// statusesOf are the passages' statuses: adapted by the links that found
// one, cut by the entries that name one.
func statusesOf(ps []passage, matches []linkMatch, cuts []int) []passageStatus {
	st := make([]passageStatus, len(ps))
	for i, p := range ps {
		if p.counted() {
			st[i] = statusToDo
		}
	}
	for _, c := range cuts {
		if c >= 0 && st[c] == statusToDo {
			st[c] = statusCut
		}
	}
	for _, mt := range matches {
		if mt.found() && st[mt.Passages[0]] != statusNone {
			st[mt.Passages[0]] = statusAdapted
		}
	}
	return st
}

// adaptCount is how far the adaptation is.
type adaptCount struct{ Adapted, Cut, ToDo int }

func countOf(st []passageStatus) adaptCount {
	var c adaptCount
	for _, s := range st {
		switch s {
		case statusAdapted:
			c.Adapted++
		case statusCut:
			c.Cut++
		case statusToDo:
			c.ToDo++
		}
	}
	return c
}

// String is the pane's line: "62% adapted · 5 cut · 12 to go". The
// percentage is of the passages not cut: with none to go, it is 100.
func (c adaptCount) String() string {
	if c.Adapted+c.Cut+c.ToDo == 0 {
		return ""
	}
	pct := 100
	if n := c.Adapted + c.ToDo; n > 0 {
		pct = c.Adapted * 100 / n
	}
	return fmt.Sprintf("%d%% adapted · %d cut · %d to go", pct, c.Cut, c.ToDo)
}

// position is where the script is in the source at a line: the last
// passage the line's scene links to; if it links none, the last one
// linked from the nearest scene before it that does; -1 for none.
func (m *sourceModel) position(line int) int {
	s := spanAt(m.spans, line)
	for k := slices.IndexFunc(m.spans, func(o sceneSpan) bool { return o.Line == s.Line }); k >= 0; k-- {
		at := -1
		for i, l := range m.links {
			if mt := m.matches[i]; mt.found() && l.Line >= m.spans[k].Line && l.Line < m.spans[k].End {
				at = max(at, mt.Passages[0])
			}
		}
		if at >= 0 {
			return at
		}
	}
	return -1
}

// nextToDo is the first passage after `after` not adapted and not cut;
// past the end, from the start again (wrapped); -1 if none is left.
func (m *sourceModel) nextToDo(after int) (next int, wrapped bool) {
	for i := after + 1; i < len(m.status); i++ {
		if m.status[i] == statusToDo {
			return i, false
		}
	}
	for i := 0; i <= after && i < len(m.status); i++ {
		if m.status[i] == statusToDo {
			return i, true
		}
	}
	return -1, false
}

// chapterOf is the source heading passage p is under (-1: none).
func chapterOf(ps []passage, p int) int {
	for i := p - 1; i >= 0; i-- {
		if ps[i].Kind == prose.Heading && ps[i].Plain != "" {
			return i
		}
	}
	return -1
}

// headingPlaceholder is a new scene's heading to write over: the script
// language's first scene start, and the words to replace.
func headingPlaceholder(starts []string) (start, words string) {
	start = "INT."
	if len(starts) > 0 {
		start = strings.TrimSpace(starts[0])
	}
	words = "…"
	if start == "INT." {
		words = "PLACE - DAY"
	}
	return start, words
}

// sceneEdit is a new scene put in the script: at At (runes) the Text
// goes in, and the placeholder [Select, SelectEnd) is selected after.
type sceneEdit struct {
	At, Replace       int // [At, Replace) is replaced (trailing blank lines at the end)
	Text              string
	Select, SelectEnd int
}

// sceneForPassage is the edit that starts a scene for passage p after
// the scene at the cursor: its chapter's section first if p opens a
// chapter the script is not in yet (another than that of passage from,
// where the script is in the source; -1 for none) and has no section of
// that name, then a scene heading to write over, and the link.
func sceneForPassage(text string, starts []string, ps []passage, p, from, cursor int) sceneEdit {
	r := []rune(text)
	body := bodyEnd(text)
	line := strings.Count(string(r[:min(cursor, len(r))]), "\n")
	span := spanAt(sceneSpans(text, starts), line)

	var parts []string
	if ch := chapterOf(ps, p); ch >= 0 && (from < 0 || chapterOf(ps, from) != ch) {
		key := strings.Join(ps[ch].Words, " ")
		there := false
		for _, it := range outlineOf(text, starts) {
			if !it.Scene && strings.Join(normWords(it.Text), " ") == key {
				there = true
				break
			}
		}
		if !there {
			parts = append(parts, strings.Repeat("#", max(ps[ch].Level, 1))+" "+ps[ch].Plain)
		}
	}
	start, words := headingPlaceholder(starts)
	heading := start + " " + words
	quote, _ := anchorFor(ps, p)
	parts = append(parts, heading, linkNote(quote))
	block := strings.Join(parts, "\n\n")

	// where: the line the next scene starts on, or the end of the script
	at := 0
	for i, l := 0, 0; i < len(r) && l < span.End; i++ {
		if r[i] == '\n' {
			l++
			at = i + 1
		}
	}
	var ed sceneEdit
	if span.End >= strings.Count(text, "\n")+1 || at >= body {
		at = len([]rune(strings.TrimRight(string(r[:body]), "\n")))
		ed = sceneEdit{At: at, Replace: body, Text: "\n\n" + block}
		if body == len(r) {
			ed.Text += "\n"
		}
	} else {
		ed = sceneEdit{At: at, Replace: at, Text: block + "\n\n"}
		if at >= 2 && !(r[at-1] == '\n' && r[at-2] == '\n') { // a blank line before it
			ed.Text = "\n" + ed.Text
		}
	}
	h := strings.LastIndex(ed.Text, heading+"\n\n"+parts[len(parts)-1])
	ed.Select = at + len([]rune(ed.Text[:h])) + len([]rune(start)) + 1
	ed.SelectEnd = ed.Select + len([]rune(words))
	return ed
}

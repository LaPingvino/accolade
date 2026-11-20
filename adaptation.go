package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

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
// or the end), the same in bytes, and what is inside it.
type tailBlock struct {
	start, end   int
	bstart, bend int
	inside       string
	md           string // the source block's: inside, ending with a new line
	ok           bool
}

// tails is what tailBlocks found last, for the text it found it in: the
// window asks for it several times per event, of the same text.
var tails struct {
	sync.Mutex
	text             string
	src, adapt, prod tailBlock
	notes            int // notesStart, in bytes
	ok               bool
}

// tailBlocks are the script's source and adaptation blocks, the last
// things before its notes (or its end), in either order.
func tailBlocks(text string) (src, adapt tailBlock) {
	src, adapt, _ = tailsOf(text)
	return src, adapt
}

// tailsOf is tailBlocks, and where the notes block starts (in bytes; the
// text's length without one); remembered for the last text asked about.
func tailsOf(text string) (src, adapt tailBlock, notes int) {
	src, adapt, _, notes = allTails(text)
	return src, adapt, notes
}

// allTails is tailsOf with the production block (production.go).
func allTails(text string) (src, adapt, prod tailBlock, notes int) {
	tails.Lock()
	defer tails.Unlock()
	if tails.ok && sameString(tails.text, text) {
		return tails.src, tails.adapt, tails.prod, tails.notes
	}
	src, adapt, prod, notes = findTails(text)
	tails.text, tails.src, tails.adapt, tails.prod, tails.notes, tails.ok = text, src, adapt, prod, notes, true
	return src, adapt, prod, notes
}

// findTails scans for the tail blocks, in bytes, then gives their places
// in runes too.
func findTails(text string) (src, adapt, prod tailBlock, notes int) {
	notes = fountainNotes.startByte(text)
	end := notes
	for {
		before := text[:end]
		found := false
		for _, b := range []struct {
			marks *notesMarks
			to    *tailBlock
		}{{&sourceMarks, &src}, {&adaptationMarks, &adapt}, {&productionMarks, &prod}} {
			if b.to.ok {
				continue
			}
			if i, inside := b.marks.at(before); i >= 0 {
				*b.to = tailBlock{bstart: i, bend: end, ok: true,
					inside: strings.ReplaceAll(inside, b.marks.escaped, b.marks.close)}
				end, found = i, true
				break
			}
		}
		if !found {
			break
		}
	}
	if src.ok {
		src.md = src.inside
		if src.md != "" && !strings.HasSuffix(src.md, "\n") { // the block's last line ends as a file's
			src.md += "\n"
		}
	}
	// the places in runes, counted once from the start: the text is long,
	// the blocks are at its end
	at, n := 0, 0
	runes := func(b int) int {
		if b < at {
			return utf8.RuneCountInString(text[:b])
		}
		n += utf8.RuneCountInString(text[at:b])
		at = b
		return n
	}
	found := []*tailBlock{}
	for _, t := range []*tailBlock{&src, &adapt, &prod} {
		if t.ok {
			found = append(found, t)
		}
	}
	slices.SortFunc(found, func(a, b *tailBlock) int { return a.bstart - b.bstart })
	for _, t := range found {
		t.start = runes(t.bstart)
		t.end = runes(t.bend)
	}
	return src, adapt, prod, notes
}

// sameString reports whether a and b are the same text, at once if they
// are the same string (a text asked for again).
func sameString(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 || unsafe.StringData(a) == unsafe.StringData(b) {
		return true
	}
	return a == b
}

// bodyEnd is where the script proper ends, in runes: before its source,
// adaptation and notes blocks.
func bodyEnd(text string) int {
	return utf8.RuneCountInString(text[:bodyEndByte(text)])
}

// bodyEndByte is bodyEnd in bytes.
func bodyEndByte(text string) int {
	src, adapt, prod, end := allTails(text)
	for _, t := range []tailBlock{src, adapt, prod} {
		if t.ok {
			end = min(end, t.bstart)
		}
	}
	return end
}

// adaptationBlock is what is inside the script's adaptation block.
func adaptationBlock(text string) (string, bool) {
	_, a := tailBlocks(text)
	return a.inside, a.ok
}

// cutMark is a cut: entry (or a skipped links: entry's link), the
// passage it names: in the new form a text directive (textfrag.go), in
// the old its first words and which of the passages opening so.
type cutMark struct {
	Quote string   // the words as written (the new form's: as shown)
	Nth   int      // old form: which of the passages opening so (1, 2 …); 0: the one
	Words []string // normalised
	Ref   linkRef
}

// String is the entry as the block writes it.
func (c cutMark) String() string {
	if !c.Ref.Old && c.Ref.Bad == "" && c.Ref.Dir.Start != "" {
		return c.Ref.String()
	}
	q := curlyQuotes(strings.ReplaceAll(c.Quote, "\n", " "))
	s := `"` + q + `"`
	if c.Nth > 0 {
		s += " (" + strconv.Itoa(c.Nth) + ")"
	}
	return s
}

// key is the entry's identity: its link's, and which one.
func (c cutMark) key() string { return c.ref().key() + "\x02" + strconv.Itoa(c.Nth) }

// ref is what the entry names (an entry made with words only: those, in
// the old form).
func (c cutMark) ref() linkRef {
	if !c.Ref.Old && c.Ref.Bad == "" && c.Ref.Dir.Start == "" {
		return linkRef{Old: true, Quote: c.Quote}
	}
	return c.Ref
}

// cutOf is the entry for a link in the new form.
func cutOf(r linkRef) cutMark { return cutMark{Quote: r.shown(), Words: r.words(), Ref: r} }

// parseCutEntries reads an entry: in the new form (#:~:text=…, a file
// before it or not) each directive of it; else as parseCutEntry.
func parseCutEntries(s string) []cutMark {
	t := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(s), "-*•"))
	if isFragmentRef(t) {
		var out []cutMark
		for _, r := range fragmentRefs(t) {
			if r.Bad == "" {
				out = append(out, cutOf(r))
			}
		}
		return out
	}
	if c, ok := parseCutEntry(s); ok {
		return []cutMark{c}
	}
	return nil
}

// parseCutEntry reads an old-form entry: a quote (straight quotes around
// it, or curly ones), or bare words, with "(2)" after it for the second
// of passages alike; a leading "-" or "*" is a list's.
func parseCutEntry(s string) (cutMark, bool) {
	if isFragmentRef(strings.TrimLeft(strings.TrimSpace(s), "-*•")) {
		cs := parseCutEntries(s)
		if len(cs) == 0 {
			return cutMark{}, false
		}
		return cs[0], true
	}
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
	return cutMark{Quote: s, Nth: nth, Words: normWords(s), Ref: linkRef{Old: true, Quote: s}}, len(normWords(s)) > 0
}

// splitCutEntries parts what follows "cut:" on its line: several
// entries in straight quotes ("a", "b"), else one.
func splitCutEntries(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if isFragmentRef(s) { // new-form entries: parted by ;
		var out []string
		for _, e := range strings.Split(s, ";") {
			if len(out) > 0 && !isFragmentRef(e) { // a ; in the text
				out[len(out)-1] += ";" + e
				continue
			}
			out = append(out, e)
		}
		for i := range out {
			out[i] = strings.TrimSpace(out[i])
		}
		return out
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
	if line == "" || line[0] == ' ' || line[0] == '\t' || strings.ContainsRune(`-*•"“„«'‘#`, []rune(line)[0]) {
		return "", "", false
	}
	k, rest, found := strings.Cut(line, ":")
	if !found || strings.ContainsAny(k, `"“”#`) || len(strings.Fields(k)) > 3 {
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
					out = append(out, parseCutEntries(e)...)
				}
			}
			continue
		}
		if in {
			out = append(out, parseCutEntries(line)...)
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

// cutSpan is the passages a cut entry names, [first, last]; first -1:
// none.
type cutSpan struct{ first, last int }

var noCut = cutSpan{-1, -1}

// has reports whether the entry names passage p.
func (c cutSpan) has(p int) bool { return c.first >= 0 && p >= c.first && p <= c.last }

// resolveCut is the passages a cut entry names.
func resolveCut(c cutMark, ps []passage) cutSpan {
	return cutAt(c, matchRefIn(c.ref(), ps))
}

// cutAt is the passages a cut entry names, given what its link found.
func cutAt(c cutMark, mt linkMatch) cutSpan {
	switch {
	case c.Nth > 0 && (mt.found() || mt.Kind == linkAmbiguous) && c.Nth <= len(mt.Passages):
		p := mt.Passages[c.Nth-1]
		return cutSpan{p, p}
	case c.Nth <= 1 && mt.found():
		return cutSpan{mt.Passages[0], mt.last()}
	}
	return noCut
}

// cutFor is the entry that names passage i: a link to it (fragmentFor),
// its prefix or suffix telling it from passages alike.
func cutFor(ps []passage, i int) cutMark {
	d, _ := fragmentFor(matcherFor(ps), i, i)
	return cutOf(linkRef{Dir: d})
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
// them, cut by the entries that name them.
func statusesOf(ps []passage, matches []linkMatch, cuts []cutSpan) []passageStatus {
	st := make([]passageStatus, len(ps))
	for i, p := range ps {
		if p.counted() {
			st[i] = statusToDo
		}
	}
	for _, c := range cuts {
		for p := max(c.first, 0); c.first >= 0 && p <= c.last && p < len(st); p++ {
			if st[p] == statusToDo {
				st[p] = statusCut
			}
		}
	}
	for _, mt := range matches {
		if !mt.found() {
			continue
		}
		for p := mt.Passages[0]; p <= mt.last() && p < len(st); p++ {
			if st[p] != statusNone {
				st[p] = statusAdapted
			}
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
				at = max(at, mt.last())
			}
		}
		if at >= 0 {
			return at
		}
	}
	return -1
}

// lastLinked is the last passage in the source a link found, and the
// line of that link (-1, -1: none).
func (m *sourceModel) lastLinked() (passage, line int) {
	passage, line = -1, -1
	for i, mt := range m.matches {
		if mt.found() && mt.last() > passage {
			passage, line = mt.last(), m.links[i].Line
		}
	}
	return passage, line
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
// that name, then a scene heading to write over, and the link. If the
// script has the chapter's section and the cursor is not in it, the
// scene goes at the end of that chapter, before the next section (sections
// made from the source's headings before any scene: each scene under its
// own; a passage under a part's heading before the part's first chapter).
func sceneForPassage(text string, starts []string, ps []passage, p, from, cursor int) sceneEdit {
	r := []rune(text)
	body := bodyEnd(text)
	line := strings.Count(string(r[:min(cursor, len(r))]), "\n")
	span := spanAt(sceneSpans(text, starts), line)
	lines := strings.Count(text, "\n") + 1
	endLine := span.End // the scene goes in at this line

	var parts []string
	if ch := chapterOf(ps, p); ch >= 0 {
		key := strings.Join(ps[ch].Words, " ")
		outline := outlineOf(text, starts)
		sec := slices.IndexFunc(outline, func(it outlineItem) bool {
			return !it.Scene && strings.Join(normWords(it.Text), " ") == key
		})
		switch {
		case sec >= 0:
			chEnd := lines // the chapter's passages end at the next section, or the end
			for _, it := range outline[sec+1:] {
				if !it.Scene {
					chEnd = it.Line
					break
				}
			}
			if line < outline[sec].Line || line >= chEnd {
				endLine = chEnd
			}
		case from < 0 || chapterOf(ps, from) != ch:
			parts = append(parts, strings.Repeat("#", max(ps[ch].Level, 1))+" "+ps[ch].Plain)
		}
	}
	start, words := headingPlaceholder(starts)
	heading := start + " " + words
	note, _ := linkNoteFor(matcherFor(ps), p, p)
	parts = append(parts, heading, note)
	block := strings.Join(parts, "\n\n")

	// where: the line the next scene starts on, or the end of the script
	at := 0
	for i, l := 0, 0; i < len(r) && l < endLine; i++ {
		if r[i] == '\n' {
			l++
			at = i + 1
		}
	}
	var ed sceneEdit
	if endLine >= lines || at >= body {
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

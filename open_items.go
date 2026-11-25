package main

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// What is open in a script, read from its text (LOOP.md, Joop
// 2026-11-24: "the whole app always also kinda works like a todo list"):
// the notes no other tool claims, the to-dos among them, the boneyard
// sections (a cut scene or a note kept out of the print), the beats no
// scene grew from yet, and the unsorted notes at the end. Nothing of it
// is kept anywhere else: an item is gone when the text no longer has it.
// The Notes pane (notes_pane.go) lists them; the status bar counts the
// open ones; the outline's and the cards' headers count their beats; the
// editor's margin marks their lines. A Revision ribbon is to step
// through the same items.

// openKind is what an item is.
type openKind int

const (
	openTodo     openKind = iota // [[todo: …]], [[TODO …]]
	openCut                      // a boneyard that reads as script
	openNote                     // a note [[…]], or a boneyard of prose
	openBeat                     // a synopsis no scene grew from yet
	openUnsorted                 // a paragraph of the notes block at the end
	openRevision                 // [[revision: …]]: a finding kept, to come back to
	openCheck                    // a check's finding: computed, never stored (revision_checks.go)
	openKinds
)

// String is the kind's name, as the pane's filters and rows say it.
func (k openKind) String() string {
	return [...]string{"To do", "Cut", "Note", "Beat", "Unsorted", "Revision", "Check"}[k]
}

// open tells whether the item is still to be dealt with: the status
// bar's count (a note or a cut is kept, not open; a done to-do is done;
// a check's finding is counted by the Revision tab, not the status bar).
func (it openItem) open() bool {
	switch it.Kind {
	case openTodo:
		return !it.Done
	case openBeat, openRevision:
		return true
	}
	return false
}

// openShape is how an item is written in the text, for writing it again.
type openShape int

const (
	shapeNote     openShape = iota // [[…]]
	shapeBoneyard                  // /* … */ (Markdown: <!-- … -->)
	shapeSynopsis                  // = … lines
	shapePara                      // a paragraph of the notes block
	shapeFinding                   // a check's finding: its line, nothing written
)

// openItem is one item: where it is in the text (runes), what it says,
// and where in the script it is.
type openItem struct {
	Kind  openKind
	Shape openShape
	// Start and End are the whole of it in the text, in runes: the note
	// with its brackets, the boneyard with its marks, the synopsis's lines
	Start, End int
	Line       int    // its first line
	Raw        string // the text from Start to End, as it was read
	Text       string // what it says: what the pane shows and edits
	// Prefix is what a to-do's note starts with ("todo: "), kept when
	// it is edited; Label is a boneyard's first line when Accolade wrote
	// it there ("Omitted 12"), not its text
	Prefix, Label string
	// Title is a cut's: its first scene heading (or first line)
	Title string
	// Ctx is where it is: the heading of the scene or section it is in
	// ("" before the first); CtxLine that heading's line (-1 none);
	// Section the line of the section it is in (-1 none: its group)
	Ctx              string
	CtxLine, Section int
	// Omitted is a cut kept under its OMITTED heading in a locked script
	// ("12"; "" for an "Omitted scene" one); Kept: it is under its
	// OMITTED heading, which starts at HeadStart (runes), and Restore
	// brings the scene back with its number (Restore Scene's).
	Omitted   string
	Kept      bool
	HeadStart int
	// Single: a boneyard on one line, /* … */
	Single bool
	// Done: a to-do written as done ([[done: …]], [[todo: [x] …]])
	Done bool
	// Check is a finding's check (checkPlace …, checkLint); its Start and
	// End are its line's, Raw the line as it was read (a prose check's:
	// what it is about); KeepAt where Keep writes it (runes; 0: at its
	// line's end), a prose finding's paragraph's end
	Check  string
	KeepAt int
}

// openMemo is the last text's items (the pane, the status bar, the
// marks all ask for the same text after a pause).
var openMemo struct {
	sync.Mutex
	text     string
	markdown bool
	starts   []string
	items    []openItem
	ok       bool
}

// openItemsOf are the text's items in script order (the unsorted notes
// last). markdown: a Markdown document's (its comments).
func openItemsOf(text string, starts []string, markdown bool) []openItem {
	openMemo.Lock()
	defer openMemo.Unlock()
	if openMemo.ok && openMemo.markdown == markdown && slices.Equal(openMemo.starts, starts) && sameString(openMemo.text, text) {
		return openMemo.items
	}
	var items []openItem
	if markdown {
		items = markdownOpenItems(text)
	} else {
		items = fountainOpenItems(text, starts)
	}
	openMemo.text, openMemo.markdown, openMemo.starts, openMemo.items, openMemo.ok = text, markdown, slices.Clone(starts), items, true
	return items
}

// runeCounter turns byte offsets, asked in order, into rune offsets.
type runeCounter struct {
	text  string
	at, n int
}

func (c *runeCounter) runes(b int) int {
	if b < c.at {
		return utf8.RuneCountInString(c.text[:b])
	}
	c.n += utf8.RuneCountInString(c.text[c.at:b])
	c.at = b
	return c.n
}

// byteSpan is a part of the text, in bytes.
type byteSpan struct{ start, end int }

// fountainBodyEnd is where the script's own text ends: before Accolade's
// blocks at its end (the notes block too), in bytes.
func fountainBodyEnd(text string) int {
	end := bodyEndByte(text)
	if f := tailSetOf(text).fix; f.ok {
		end = min(end, f.bstart)
	}
	return end
}

// boneyards are the text's boneyard sections up to end, by Fountain's
// line rule (a line starting with /* opens one, the first */ closes it),
// in bytes; one never closed runs to end.
func boneyards(text string, end int) []byteSpan {
	var out []byteSpan
	for at := 0; at < end; {
		le := strings.IndexByte(text[at:end], '\n')
		next := end
		if le >= 0 {
			next = at + le + 1
		}
		line := text[at:next]
		t := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(t, "/*") {
			s := at + len(line) - len(t)
			c := strings.Index(text[s+2:end], "*/")
			e := end
			if c >= 0 {
				e = s + 2 + c + 2
			}
			out = append(out, byteSpan{s, e})
			if nl := strings.IndexByte(text[e:end], '\n'); nl >= 0 {
				next = e + nl + 1
			} else {
				next = end
			}
		}
		at = next
	}
	return out
}

// inSpans tells whether b is in one of spans (sorted).
func inSpans(spans []byteSpan, b int) bool {
	i, _ := slices.BinarySearchFunc(spans, b, func(s byteSpan, b int) int {
		switch {
		case s.end <= b:
			return -1
		case s.start > b:
			return 1
		}
		return 0
	})
	return i < len(spans) && spans[i].start <= b && b < spans[i].end
}

// fountainOpenItems are a screenplay's items.
func fountainOpenItems(text string, starts []string) []openItem {
	end := fountainBodyEnd(text)
	lines := strings.Split(text, "\n")
	lineAt := lineIndex(text)
	tp := titlePageEnd(lines)
	tpEnd := 0 // bytes
	for _, l := range lines[:tp] {
		tpEnd += len(l) + 1
	}
	outline := outlineOf(text, starts)
	headings := map[int]bool{}
	for _, it := range outline {
		headings[it.Line] = true
	}
	var items []openItem
	bones := boneyards(text, end)
	// the boneyards
	for _, s := range bones {
		if s.start < tpEnd {
			continue
		}
		if it, ok := boneyardItem(text, s, starts, lines, lineAt); ok {
			items = append(items, it)
		}
	}
	// the notes: from a [[ to the first ]] in its paragraph, outside the
	// boneyard and the title page
	for at := tpEnd; at < end; {
		o := strings.Index(text[at:end], "[[")
		if o < 0 {
			break
		}
		o += at
		if inSpans(bones, o) {
			at = o + 2
			continue
		}
		c := strings.Index(text[o+2:end], "]]")
		if c < 0 {
			break
		}
		c += o + 2
		if p := strings.Index(text[o:c], "\n\n"); p >= 0 { // a note ends with its paragraph
			at = o + p
			continue
		}
		at = c + 2
		inside := text[o+2 : c]
		line := lineAt(o)
		it := openItem{Kind: openNote, Shape: shapeNote, Start: o, End: c + 2, Line: line, Text: strings.TrimSpace(inside)}
		if p, rest, ok := revisionNote(inside); ok { // kept on a heading too: a finding's there
			it.Kind, it.Prefix, it.Text = openRevision, p, rest
			items = append(items, it)
			continue
		}
		if headings[line] || claimedNote(inside) {
			continue
		}
		if p, rest, done, ok := todoNote(inside); ok {
			it.Kind, it.Prefix, it.Text, it.Done = openTodo, p, rest, done
		}
		items = append(items, it)
	}
	for _, b := range beatsOf(lines, outline, tp, endLine(text, end)) {
		items = append(items, beatItem(lines, b.from, b.to))
	}
	slices.SortStableFunc(items, func(a, b openItem) int { return a.Start - b.Start })
	// where each is: the heading before it
	setContext(items, outline, lines)
	items = append(items, unsortedItems(text, fountainNotes, lineAt)...)
	toRunes(text, items)
	return items
}

// beatAt is a beat no scene grew from yet: its lines [from, to), the
// line of the section it is under and of the outline item it is loose
// under (-1 none: before the first heading).
type beatAt struct{ from, to, section, under int }

// beatsOf are the script's beats no scene grew from: the synopses before
// the first heading (outside the title page, before tp), a section's
// after its own, a scene's at its end with nothing but plans after them;
// before line end (the blocks at the script's end are not its). The
// outline's and the cards' headers count them (outline.go, cards.go).
func beatsOf(lines []string, outline []outlineItem, tp, end int) []beatAt {
	var out []beatAt
	first := len(lines)
	if len(outline) > 0 {
		first = outline[0].Line
	}
	inBoneyard := false
	for i := tp; i < min(first, end); i++ {
		t := strings.TrimSpace(lines[i])
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		if isSynopsisLine(t) {
			e := synopsisEnd(lines, i)
			out = append(out, beatAt{i, e, -1, -1})
			i = e - 1
		}
	}
	section := -1
	for k, it := range outline {
		if !it.Scene {
			section = it.Line
		}
		next := len(lines)
		if k+1 < len(outline) {
			next = outline[k+1].Line
		}
		for _, l := range it.Loose {
			if l.Line >= end || it.Scene && !onlyPlans(lines[l.Line:next]) {
				continue
			}
			out = append(out, beatAt{l.Line, synopsisEnd(lines, l.Line), section, it.Line})
		}
	}
	return out
}

// beatsBySection are the beats' first lines by the section they are
// under (-1: none).
func beatsBySection(beats []beatAt) map[int][]int {
	out := map[int][]int{}
	for _, b := range beats {
		out[b.section] = append(out[b.section], b.from)
	}
	return out
}

// beatsByItem are the beats' first lines by the item they are loose
// under (-1: before the first heading): the cards' marks, each card its
// own.
func beatsByItem(beats []beatAt) map[int][]int {
	out := map[int][]int{}
	for _, b := range beats {
		out[b.under] = append(out[b.under], b.from)
	}
	return out
}

// endLine is the first line not before byte end: a line end stops
// short of (a beat on the script's last line, the blocks after it) is in.
func endLine(text string, end int) int {
	n := strings.Count(text[:end], "\n")
	if end > 0 && text[end-1] != '\n' {
		n++
	}
	return n
}

// lineIndex gives the line of a byte offset (offsets asked in any order).
func lineIndex(text string) func(b int) int {
	var starts []int
	starts = append(starts, 0)
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return func(b int) int {
		i, found := slices.BinarySearch(starts, b)
		if found {
			return i
		}
		return i - 1
	}
}

// lineStartByte is where line i starts, in bytes (for a few lines: it
// counts from the top).
func lineStartByte(lines []string, i int) int {
	n := 0
	for _, l := range lines[:i] {
		n += len(l) + 1
	}
	return n
}

// beatItem is the synopsis paragraph on lines [from, to).
func beatItem(lines []string, from, to int) openItem {
	start := lineStartByte(lines, from)
	var text []string
	end := start
	for i := from; i < to; i++ {
		text = append(text, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "=")))
		end += len(lines[i]) + 1
	}
	end-- // without the last new line
	return openItem{Kind: openBeat, Shape: shapeSynopsis, Start: start, End: end, Line: from, Text: strings.Join(text, "\n")}
}

// setContext gives each item the heading it is under, and its section.
func setContext(items []openItem, outline []outlineItem, lines []string) {
	k, section := -1, -1
	for i := range items {
		it := &items[i]
		for k+1 < len(outline) && outline[k+1].Line <= it.Line {
			k++
			if !outline[k].Scene {
				section = outline[k].Line
			}
		}
		it.CtxLine, it.Section = -1, section
		if k >= 0 {
			o := outline[k]
			it.CtxLine = o.Line
			it.Ctx = o.Text
			if !o.Scene {
				it.Ctx = strings.Repeat("#", o.Level) + " " + o.Text
			}
		}
	}
}

// claimedNote tells whether a note's inside is another tool's: a colour,
// a link to the source, a storyboard frame, the frames' folder.
func claimedNote(inside string) bool {
	t := strings.TrimSpace(inside)
	if t == "" {
		return false
	}
	if lex.NoteColor("[["+inside+"]]") != "" || fountain.IsSourceLink(inside) {
		return true
	}
	if _, ok := lex.ParseBoard(inside); ok {
		return true
	}
	return len(t) > 9 && strings.EqualFold(t[:9], "dataPath:")
}

// todoNote reads a note's inside as a to-do: "todo: …", "TODO …",
// "to do: …" (any case); its prefix as written, and the rest. done: it
// is written as done, "done: …" (the colon needed: "done with it" is a
// note) or "todo: [x] …" (its prefix then the whole of that).
func todoNote(inside string) (prefix, rest string, done, ok bool) {
	if p, r, ok := notePrefixed(inside, "to do", "todo", "to-do"); ok {
		if p2, r2, ok2 := cutCheckbox(r); ok2 {
			return p + p2, r2, true, true
		}
		return p, r, false, true
	}
	if p, r, ok := notePrefixed(inside, "done"); ok && strings.Contains(p, ":") {
		return p, r, true, true
	}
	return "", "", false, false
}

// cutCheckbox cuts a ticked box, "[x] ", from the start of a to-do's
// text: what it cut, the rest.
func cutCheckbox(r string) (cut, rest string, ok bool) {
	for _, b := range []string{"[x]", "[X]"} {
		if strings.HasPrefix(r, b) {
			rest = strings.TrimLeft(r[len(b):], " \t")
			return r[:len(r)-len(rest)], rest, true
		}
	}
	return "", r, false
}

// revisionNote reads a note's inside as a revision note, a finding kept
// ("revision: …", any case): its prefix as written, and the rest.
func revisionNote(inside string) (prefix, rest string, ok bool) {
	p, r, ok := notePrefixed(inside, "revision")
	if !ok || !strings.Contains(p, ":") { // "revision" alone, or "revisions to…": a note
		return "", "", false
	}
	return p, r, true
}

// notePrefixed reads a note's inside as starting with one of words (any
// case), then a colon or a space: its prefix as written, and the rest.
func notePrefixed(inside string, words ...string) (prefix, rest string, ok bool) {
	t := strings.TrimLeft(inside, " \t\n")
	lead := len(inside) - len(t)
	for _, w := range words {
		if len(t) >= len(w) && strings.EqualFold(t[:len(w)], w) {
			after := t[len(w):]
			r := strings.TrimLeft(after, " \t")
			switch {
			case after == "" || strings.TrimSpace(after) == "":
			case strings.HasPrefix(r, ":"):
				r = strings.TrimLeft(r[1:], " \t")
			case r == after: // "todos", "todolist": a word of its own
				continue
			}
			p := inside[:lead+len(t)-len(r)]
			return p, strings.TrimSpace(r), true
		}
	}
	return "", "", false
}

// omittedLabel reads a boneyard's first line as an OMITTED scene's
// ("Omitted 12", "Omitted scene"): its number ("" for none), ok.
func omittedLabel(first string) (string, bool) {
	f := strings.TrimSpace(first)
	n, ok := strings.CutPrefix(f, "Omitted")
	if !ok || n != "" && n[0] != ' ' {
		return "", false
	}
	n = strings.TrimSpace(n)
	if n == "scene" {
		return "", true
	}
	return n, true
}

// boneyardItem is a boneyard s (bytes) as an item: a cut when its inside
// reads as script, else a note; ok false for a language line.
func boneyardItem(text string, s byteSpan, starts []string, lines []string, lineAt func(int) int) (openItem, bool) {
	raw := text[s.start:s.end]
	inner := strings.TrimPrefix(raw, "/*")
	inner = strings.TrimSuffix(inner, "*/")
	it := openItem{Shape: shapeBoneyard, Start: s.start, End: s.end, Line: lineAt(s.start), Single: !strings.Contains(raw, "\n")}
	if it.Single {
		t := strings.TrimSpace(inner)
		if strings.HasPrefix(t, "Language:") {
			return it, false
		}
		it.Text = t
	} else {
		first, rest, _ := strings.Cut(inner, "\n")
		if n, ok := omittedLabel(first); ok {
			it.Label = strings.TrimSpace(first)
			it.Omitted = n
			first = ""
		}
		body := rest
		if strings.TrimSpace(first) != "" {
			body = strings.TrimLeft(first, " ") + "\n" + rest
		}
		it.Text = strings.ReplaceAll(strings.TrimRight(strings.TrimLeft(body, "\n"), " \t\n"), "* /", "*/")
	}
	it.Kind = openNote
	if title, script := readsAsScript(it.Text, starts); script {
		it.Kind, it.Title = openCut, title
	}
	// kept under its OMITTED heading: Restore Scene's
	if it.Label != "" {
		for i := it.Line - 1; i >= 0; i-- {
			t := strings.TrimSpace(lines[i])
			if t == "" {
				continue
			}
			if n, ok := lex.OmittedNumber(t); ok && (n == it.Omitted || it.Omitted == "") {
				it.HeadStart = lineStartByte(lines, i) + len(lines[i]) - len(strings.TrimLeft(lines[i], " \t"))
				it.Kind, it.Kept = openCut, true
				if it.Title == "" {
					it.Title, _, _ = strings.Cut(it.Text, "\n")
				}
				return it, true
			}
			break
		}
		it.Omitted = "" // no heading above it: a plain boneyard
	}
	return it, true
}

// readsAsScript tells whether text reads as a screenplay by Lexington's
// parse: a scene heading, a speech or a transition in it; with its first
// heading (else its first line) as its title.
func readsAsScript(text string, starts []string) (title string, ok bool) {
	if strings.TrimSpace(text) == "" {
		return "", false
	}
	if starts == nil {
		starts = sceneStarts("en")
	}
	// a blank line first: "Key: value" at the top is no title page here
	sp := fountain.Parse(starts, strings.NewReader("\n"+text+"\n"))
	speaker := false
	for _, l := range sp {
		switch l.Type {
		case lex.TypeScene:
			h, _ := lex.SceneNumber(strings.TrimSpace(l.Contents))
			return strings.TrimSpace(h), true
		case lex.TypeSpeaker, lex.TypeDualOpen:
			speaker = true
		case lex.TypeDialog:
			ok = ok || speaker
		case lex.TypeTrans:
			ok = true
		}
	}
	first, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(first), ok
}

// unsortedItems are the paragraphs of the notes block at the end of the
// text (marks: its kind's), each an item.
func unsortedItems(text string, m notesMarks, lineAt func(int) int) []openItem {
	i, inside := m.at(text)
	if i < 0 {
		return nil
	}
	o := strings.Index(text[i:], m.open+"\n")
	if o < 0 {
		return nil
	}
	at := i + o + len(m.open) + 1 // where the inside starts, in bytes
	var out []openItem
	for p := 0; p < len(inside); {
		for p < len(inside) && inside[p] == '\n' {
			p++
		}
		if p >= len(inside) {
			break
		}
		e := strings.Index(inside[p:], "\n\n")
		if e < 0 {
			e = len(inside)
		} else {
			e += p
		}
		para := strings.TrimRight(inside[p:e], " \t\n")
		if strings.TrimSpace(para) != "" {
			out = append(out, openItem{Kind: openUnsorted, Shape: shapePara, Start: at + p, End: at + p + len(para), Line: lineAt(at + p),
				Text: strings.ReplaceAll(para, m.escaped, m.close), CtxLine: -1, Section: -1, Ctx: "Unsorted notes"})
		}
		p = e
	}
	return out
}

// toRunes turns the items' places from bytes into runes, and keeps
// their text as it was.
func toRunes(text string, items []openItem) {
	order := make([]*openItem, len(items))
	for i := range items {
		order[i] = &items[i]
	}
	// in order of every offset: Start, End, HeadStart
	type at struct {
		b   int
		set func(int)
	}
	var ats []at
	for _, it := range order {
		it.Raw = text[it.Start:it.End]
		ats = append(ats, at{it.Start, func(r int) { it.Start = r }}, at{it.End, func(r int) { it.End = r }})
		if it.Kept {
			ats = append(ats, at{it.HeadStart, func(r int) { it.HeadStart = r }})
		}
	}
	slices.SortStableFunc(ats, func(a, b at) int { return a.b - b.b })
	c := runeCounter{text: text}
	for _, a := range ats {
		a.set(c.runes(a.b))
	}
}

// markdownOpenItems are a Markdown document's: its comments, as notes
// (to-dos when they start so), and its unsorted notes.
func markdownOpenItems(text string) []openItem {
	end := markdownEndByte(text)
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	lineAt := lineIndex(text)
	outline := markdownOutline(text)
	var items []openItem
	at := 0
	for i := 0; i < len(lines) && at < end; i++ {
		l := lines[i]
		start := at
		at += len(l) + 1
		if kinds[i] != syntax.MdComment {
			continue
		}
		t := strings.TrimLeft(l, " \t")
		if !strings.HasPrefix(t, "<!--") {
			continue
		}
		s := start + len(l) - len(t)
		c := strings.Index(text[s+4:end], "-->")
		if c < 0 {
			break
		}
		e := s + 4 + c + 3
		raw := text[s:e]
		inner := strings.TrimSuffix(strings.TrimPrefix(raw, "<!--"), "-->")
		it := openItem{Kind: openNote, Shape: shapeBoneyard, Start: s, End: e, Line: i, Single: !strings.Contains(raw, "\n"),
			Text: strings.ReplaceAll(strings.TrimSpace(inner), "-- >", "-->")}
		if p, rest, done, ok := todoNote(inner); ok {
			it.Kind, it.Prefix, it.Text, it.Done = openTodo, strings.TrimLeft(p, " \t\n"), rest, done
		} else if p, rest, ok := revisionNote(inner); ok { // a finding kept (prose_checks.go)
			it.Kind, it.Prefix, it.Text = openRevision, strings.TrimLeft(p, " \t\n"), strings.ReplaceAll(rest, "-- >", "-->")
		}
		items = append(items, it)
		for at < e && i+1 < len(lines) { // past its lines
			i++
			at += len(lines[i]) + 1
		}
	}
	setContext(items, outline, lines)
	items = append(items, unsortedItems(text, markdownNotes, lineAt)...)
	toRunes(text, items)
	return items
}

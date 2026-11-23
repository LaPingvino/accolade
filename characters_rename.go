package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/collab"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/lexington/lex"
)

// Rename Everywhere and Merge Into… (the Characters pane's actions).
//
// Each is one undo step of the script's, made of the least edits: every
// place the name stands is replaced by itself (the name's runes only: a
// cue's @, extension and ^ stay), the character list's entry and the
// learned notes' cue only where they change. Nothing between them is
// written again, so a co-writer's typing anywhere else, a line beside a
// cue too, merges as any edit does (docs/COLLAB.md, fix 5: the minimal
// edit is made here, before the Doc's own diff).
//
// Rename Everywhere renames the character wherever the name stands in
// the script as a word: its cues, the action that introduces them, the
// dialogue that calls them, a heading (INT. ANNA'S FLAT). Each keeps the
// case it was in: ANNA becomes HANNA, Anna Hanna. Merge Into… is for two
// spellings of one character (ANA and ANNA): only the cues change, the
// two entries of the list become one (the notes together, a field kept
// where the other has none), and the learned names go with them.
//
// The breakdown and the schedule are kept beside the script, outside its
// undo, and follow (as they follow the scene numbers, renumber_follow.go):
// a cast member of the old name takes the new one (merged into the other
// one's, for Merge), each tag whose words or the words around them changed
// is anchored again where its words went, a scene marked done and a strip
// take their heading's new words, and a strip's sum is taken over where
// only the name changed in its scene. Undo and Redo of the step take them
// back and forth again (followRenamed).

// renameScope is what a rename changes.
type renameScope int

const (
	renameEverywhere renameScope = iota // the name as a word anywhere in the script
	renameCues                          // its cues only (Merge)
)

// casedLike is name written in the case of was: all capitals, all small
// letters, or a capital each word; else as given.
func casedLike(was, name string) string {
	upper, lower, title := true, true, true
	start := true
	for _, r := range was {
		if !unicode.IsLetter(r) {
			start = true
			continue
		}
		if unicode.IsUpper(r) {
			lower = false
			if !start {
				title = false
			}
		} else if unicode.IsLower(r) {
			upper = false
			if start {
				title = false
			}
		}
		start = false
	}
	switch {
	case upper && !lower:
		return strings.ToUpper(name)
	case lower && !upper:
		return strings.ToLower(name)
	case title:
		rs := []rune(strings.ToLower(name))
		start = true
		for i, r := range rs {
			if !unicode.IsLetter(r) {
				start = true
				continue
			}
			if start {
				rs[i] = unicode.ToUpper(r)
			}
			start = false
		}
		return string(rs)
	}
	return name
}

// renamePlaces are the places a rename can change (edits, in runes, in
// order), by where they stand: the script's body (a Markdown document's
// text), and what Rename Everywhere offers besides, off unless asked: the
// notes block, and the source block with the source links' quotes and
// the adaptation block's entries that name its passages (so that they
// keep finding them).
type renamePlaces struct {
	body   []lineEdit
	notes  []lineEdit
	source []lineEdit // the source block's
	links  []lineEdit // the links' quotes in the body, with the source
}

// renameAlso are the blocks a rename takes in besides the body.
type renameAlso struct{ notes, source bool }

// renamePlacesIn are the places that rename the character with key to
// name in text (md: a Markdown document, which has notes but no source
// block): its cues (renameCues: Merge, the body only), or the name as a
// word wherever it stands (renameEverywhere).
//
// Everywhere, a place counts when it is written in capitals (ANNA), with a
// capital each word (Anna), or as the character is spelled in a cue or the
// list (known: McCLANE, ben); so a character called WILL leaves "I will
// go" alone. The title page (Author: Ben Smith; a Markdown document's
// front matter) is left as it is, and in the body the source links'
// quotes ([[from: …text=Anna opened…]], which must keep finding their
// passage): they are renamed with the source (links), as its passages are.
func renamePlacesIn(text string, md bool, starts []string, key, name string, scope renameScope, known []string) renamePlaces {
	var out renamePlaces
	if scope == renameCues {
		if md {
			return out
		}
		m := screenplayChars(text, starts)
		for _, c := range m.chars {
			for _, s := range c.Cues {
				if charKey(s.name) == key {
					out.body = append(out.body, lineEdit{s.start, s.end, casedLike(s.name, name)})
				}
			}
		}
		slices.SortFunc(out.body, func(a, b lineEdit) int { return a.start - b.start })
		return out
	}
	want := []rune(key)
	if len(want) == 0 {
		return out
	}
	var bodyEnd, from int
	var notes notesMarks
	if md {
		bodyEnd = markdownEndByte(text)
		notes = markdownNotes
	} else {
		bodyEnd = bodyEndByte(text)
		notes = fountainNotes
	}
	rs := []rune(text)
	body := []rune(text[:bodyEnd])
	if md {
		from = frontMatterRunes(body)
	} else {
		from = titleRunes(body)
	}
	skip := leftAlone(body)
	out.body = wordEdits(rs, from, len(body), want, name, known, skip)
	if !md {
		for _, q := range linkQuotes(body) {
			out.links = append(out.links, wordEdits(rs, q[0], q[1], want, name, known, nil)...)
		}
		if t := tailSetOf(text).src; t.ok {
			if a, b, ok := blockInside(text, t.bstart, t.bend, sourceMarks); ok {
				out.source = wordEdits(rs, a, b, want, name, known, nil)
			}
		}
	}
	if i, _ := notes.at(text); i >= 0 {
		if a, b, ok := blockInside(text, i, len(text), notes); ok {
			out.notes = wordEdits(rs, a, b, want, name, known, nil)
		}
	}
	return out
}

// edits are the places a rename changes, in order: the body, and the
// blocks asked for besides.
func (p renamePlaces) edits(also renameAlso) []lineEdit {
	out := slices.Clone(p.body)
	if also.source {
		out = append(out, p.links...)
		out = append(out, p.source...)
	}
	if also.notes {
		out = append(out, p.notes...)
	}
	slices.SortFunc(out, func(a, b lineEdit) int { return a.start - b.start })
	return out
}

// wordEdits rename the name (want: its key) where it stands as a word in
// rs[from:to], each place in its own case; skip: stretches left alone
// (leftAlone's, nil none).
func wordEdits(rs []rune, from, to int, want []rune, name string, known []string, skip map[int]int) []lineEdit {
	var out []lineEdit
	for i := from; i+len(want) <= to; i++ {
		if i > 0 && isWordRune(rs[i-1]) {
			continue
		}
		if !sameUpper(rs[i:i+len(want)], want) {
			continue
		}
		j := i + len(want)
		if j < len(rs) && isWordRune(rs[j]) {
			continue
		}
		if k, ok := skip[i]; ok { // a source link's: past it
			i = k - 1
			continue
		}
		was := string(rs[i:j])
		if !namelike(was) && !slices.Contains(known, was) {
			continue
		}
		out = append(out, lineEdit{i, j, casedLike(was, name)})
		i = j - 1
	}
	return out
}

// blockInside is where the inside of the block at text[start:end] (as
// found: the blank lines before it, its marks) is, in runes: after its
// opening line, before its close mark.
func blockInside(text string, start, end int, m notesMarks) (from, to int, ok bool) {
	j := strings.Index(text[start:end], m.open+"\n")
	if j < 0 {
		return 0, 0, false
	}
	a := start + j + len(m.open) + 1
	k := strings.LastIndex(text[a:end], m.close)
	if k < 0 {
		return 0, 0, false
	}
	from = utf8.RuneCountInString(text[:a])
	return from, from + utf8.RuneCountInString(text[a:a+k]), true
}

// linkQuotes are the source links' quotes in body (runes, [from, to)):
// in [[from: file#:~:text=…]] what follows "#:~:", in the old form
// [[from: "…"]] what follows "from:"; the file's name is the link's.
func linkQuotes(body []rune) [][2]int {
	var out [][2]int
	for _, l := range linkStretches(body) {
		link := string(body[l[0]:l[1]])
		at, skip := strings.Index(link, "#:~:"), len("#:~:")
		if at < 0 {
			at, skip = len("[[from:"), 0
		}
		end := l[1]
		if strings.HasSuffix(link, "]]") {
			end -= 2
		}
		out = append(out, [2]int{l[0] + utf8.RuneCountInString(link[:at+skip]), end})
	}
	return out
}

// namelike reports whether s is written as a name is: in capitals, or
// with a capital each word and small letters after it.
func namelike(s string) bool {
	if strings.ToUpper(s) == s && strings.ToLower(s) != s {
		return true
	}
	start := true
	for _, r := range s {
		switch {
		case !unicode.IsLetter(r):
			start = true
			continue
		case start && !unicode.IsUpper(r), !start && unicode.IsUpper(r):
			return false
		}
		start = false
	}
	return true
}

// leftAlone are the stretches of body a rename leaves as they are, the
// source links ([[from: …]]): each rune in one, the stretch's end.
func leftAlone(body []rune) map[int]int {
	out := map[int]int{}
	for _, l := range linkStretches(body) {
		for k := l[0]; k < l[1]; k++ {
			out[k] = l[1]
		}
	}
	return out
}

// linkStretches are the source links ([[from: …]]) in body, in runes:
// [start, end), in order.
func linkStretches(body []rune) [][2]int {
	var out [][2]int
	s := string(body)
	at := 0 // runes before s[b:]
	for b := 0; ; {
		i := strings.Index(s[b:], "[[from:")
		if i < 0 {
			return out
		}
		at += utf8.RuneCountInString(s[b : b+i])
		b += i
		j := strings.Index(s[b:], "]]")
		if j < 0 {
			j = len(s) - b
		} else {
			j += 2
		}
		n := utf8.RuneCountInString(s[b : b+j])
		out = append(out, [2]int{at, at + n})
		at += n
		b += j
	}
}

// titleRunes is where the script's title page ends, in runes (0: it has
// none), as the title page dialog reads it (titlePageEnd).
func titleRunes(body []rune) int {
	lines := strings.Split(string(body), "\n")
	n := 0
	for _, l := range lines[:titlePageEnd(lines)] {
		n += utf8.RuneCountInString(l) + 1
	}
	return min(n, len(body))
}

// frontMatterRunes is where a Markdown document's front matter ends, in
// runes (0: it has none), as markdownBody reads it.
func frontMatterRunes(body []rune) int {
	lines := strings.Split(string(body), "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	n := 0
	for i := 0; i < len(kinds) && kinds[i] == syntax.MdFrontMatter; i++ {
		n += utf8.RuneCountInString(lines[i]) + 1
	}
	return min(n, len(body))
}

// sameUpper reports whether rs in capitals are want (want in capitals,
// its words one space apart: rs may have several).
func sameUpper(rs, want []rune) bool {
	if len(rs) != len(want) {
		return false
	}
	for k, r := range rs {
		if unicode.ToUpper(r) != want[k] && !(unicode.IsSpace(r) && want[k] == ' ') {
			return false
		}
	}
	return true
}

// followed is the last change the extras followed (a rename, a
// structural move), for Undo and Redo to take the breakdown and the
// schedule back and forth with it: as they were kept before and after it
// (the files as JSON), put back while nothing else changed them; else
// followed again by its edits (refollow; nil: they cannot be, as a
// merge's cast members cannot be told apart again).
type followed struct {
	before, after string
	atAfter       bool // the extras follow after
	bd, sched     [2][]byte
	// refollow follows the text back to before (back) or on to after, for
	// the tags, the strips or both
	refollow func(back, tags, strips bool)
}

// extrasEdit is how the breakdown and the schedule follow an edit of
// the text: where each offset of the text before went (-1: its words
// gone or changed), which scene after each scene before is (-1: none);
// a rename's cast member's names (capitals; "" none); sums: the strips'
// words taken over where the scene's words did not change (a rename:
// the change is no change of the scene's).
type extrasEdit struct {
	off      func(off int, end bool) int
	scene    func(i int) int
	from, to string
	sums     bool
	// prepare, if set, is given the texts as read before scene is asked
	prepare func(before, after *bdText)
}

// renameEdit is how the extras follow a rename's edits.
func renameEdit(before, after string, starts []string, edits []lineEdit, from, to string) extrasEdit {
	same := len(readBdText(before, starts).scenes) == len(readBdText(after, starts).scenes)
	return extrasEdit{
		off: func(off int, end bool) int { return mapThrough(edits, off, end) },
		scene: func(i int) int {
			if !same { // a rename adds no scene and takes none away
				return -1
			}
			return i
		},
		from: from, to: to, sums: true,
	}
}

// renameCharacter renames the character with key to name (scope) as one
// undo step, the list, the learned notes and the extras following; it
// reports how many places changed. known are the character's spellings
// (renamePlacesIn); also, the blocks Rename Everywhere takes in besides the
// body (renamePlaces).
func (w *MainWindow) renameCharacter(key, name string, scope renameScope, known []string, also renameAlso) int {
	name = strings.Join(strings.Fields(name), " ")
	newKey := charKey(name)
	if w.textEditor == nil || key == "" || newKey == "" {
		return 0
	}
	e := w.textEditor
	before := e.Text()
	edits := renamePlacesIn(before, w.markdown, w.sceneStarts(), key, name, scope, known).edits(also)
	// the list: the entry renamed, or for Merge put into the other's
	k := w.charBlock()
	list := k.list(before)
	entries := slices.Clone(list.entries)
	if i := list.find(key); i >= 0 {
		if j := list.find(newKey); scope == renameCues && j >= 0 && j != i {
			entries[j] = mergedEntry(entries[j], entries[i])
			entries = slices.Delete(entries, i, i+1)
		} else {
			entries[i].Name = listSpelling(entries[i].Name, name)
		}
	}
	var blocks []lineEdit
	if inside, ok := k.inside(before); ok {
		if nu := list.inside(entries); nu != inside {
			s, t, block := k.with(before, nu)
			blocks = append(blocks, lineEdit{s, t, block})
		}
	}
	// the learned notes: the cue's names under the new one; with the
	// source, the entries that name its passages as their quotes go
	l := learnedOf(before)
	for _, a := range l.Aliases {
		if charKey(a.Cue) == key {
			l.renameCue(a.Cue, casedLike(a.Cue, name))
			break
		}
	}
	if inside, ok := adaptationBlock(before); ok && !w.markdown {
		nu := withLearned(inside, l)
		if also.source && scope == renameEverywhere {
			nu = renamedEntries(nu, key, name, known)
		}
		if nu != inside {
			s, t, block := withAdaptationBlock(before, nu)
			blocks = append(blocks, lineEdit{s, t, block})
		}
	}
	if len(edits) == 0 && len(blocks) == 0 {
		return 0
	}
	w.followNumbers(true) // the extras in line with the text before
	fwd := slices.Clone(edits)
	rs := []rune(before)
	for _, b := range blocks { // a block's lines and words that changed, not the block
		for _, d := range collab.Edits(string(rs[b.start:b.end]), b.text) {
			fwd = append(fwd, lineEdit{b.start + d.Pos, b.start + d.Pos + d.Del, d.Text})
		}
	}
	slices.SortFunc(fwd, func(a, b lineEdit) int { return a.start - b.start })
	merge := scope == renameCues
	r := &followed{before: before}
	r.bd[0], r.sched[0] = w.extrasKept()
	w.shareEditsFirst(fwd)
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			for i := len(fwd) - 1; i >= 0; i-- { // the last first: the places before stay
				replaceLeast(b, fwd[i].start, fwd[i].end, fwd[i].text)
			}
		})
	})
	r.after = e.Text()
	back := invertEdits(before, fwd)
	if !merge {
		r.refollow = func(toBefore, tags, strips bool) {
			starts := w.sceneStarts()
			if toBefore {
				w.followEdit(r.after, r.before, renameEdit(r.after, r.before, starts, back, newKey, key), tags, strips)
			} else {
				w.followEdit(r.before, r.after, renameEdit(r.before, r.after, starts, fwd, key, newKey), tags, strips)
			}
		}
	}
	w.followedLast = r
	w.followRename(before, r.after, fwd, key, newKey, merge)
	r.bd[1], r.sched[1] = w.extrasKept()
	r.atAfter = true
	w.followNumbers(true) // the headings that changed are the ones seen now
	return len(edits)
}

// renamedEntries is the adaptation block's inside with the name renamed
// in its cut: and skipped links: entries, which name the source's
// passages by their words (a rename in the source with them).
func renamedEntries(inside, key, name string, known []string) string {
	lines := strings.Split(inside, "\n")
	in := false
	want := []rune(key)
	for i, line := range lines {
		if k, _, ok := isBlockKey(line); ok {
			switch k {
			case "cut", "cuts", "skipped links", "skipped link", "skipped", "skips":
				in = true
			default:
				in = false
			}
		}
		if !in {
			continue
		}
		rs := []rune(line)
		ed := wordEdits(rs, 0, len(rs), want, name, known, nil)
		for j := len(ed) - 1; j >= 0; j-- {
			rs = slices.Concat(rs[:ed[j].start], []rune(ed[j].text), rs[ed[j].end:])
		}
		lines[i] = string(rs)
	}
	return strings.Join(lines, "\n")
}

// shareEditsFirst gives a shared script's session a change made of many
// small edits (in order, in runes) as those edits, before the editor makes
// it: the editor's change then finds the Doc as it is already. Doc.SetText
// would find them by its own diff, which gives up past 2000 lines changed
// (scriptdiff's maxEdits) and then takes the whole stretch between, a
// co-writer's typing in it lost; a lead's rename in a feature is that many.
func (w *MainWindow) shareEditsFirst(edits []lineEdit) {
	if w.shared == nil {
		return
	}
	var ops []collab.Op
	w.shared.session.Do(func(d *collab.Doc) {
		for i := len(edits) - 1; i >= 0; i-- {
			e := edits[i]
			ops = append(ops, d.Delete(e.start, e.end-e.start)...)
			ops = append(ops, d.Insert(e.start, e.text)...)
		}
	})
	w.shared.session.Local(ops)
}

// extrasKept are the breakdown and the schedule as kept now (JSON).
func (w *MainWindow) extrasKept() (bd, sched []byte) {
	if w.markdown {
		return nil, nil
	}
	bd, _ = json.Marshal(w.breakdownNow())
	sched, _ = json.Marshal(w.scheduleNow())
	return bd, sched
}

// listSpelling is name as the list writes it: in the case the entry was
// written in when that was all capitals or all small letters, else as given.
func listSpelling(was, name string) string {
	if c := casedLike(was, name); c == strings.ToUpper(name) || c == strings.ToLower(name) {
		return c
	}
	return name
}

// mergedEntry is entry into with from's note after its own, and from's
// fields where into has none.
func mergedEntry(into, from charEntry) charEntry {
	if n := strings.TrimSpace(from.Note); n != "" {
		if strings.TrimSpace(into.Note) == "" {
			into.Note = from.Note
		} else {
			into.Note = strings.TrimRight(into.Note, "\n") + "\n" + from.Note
		}
	}
	for f := range into.Fields {
		if strings.TrimSpace(into.Fields[f]) == "" {
			into.Fields[f] = from.Fields[f]
		}
	}
	return into
}

// invertEdits are the edits that take after (before with edits, in order)
// back to before.
func invertEdits(before string, edits []lineEdit) []lineEdit {
	rs := []rune(before)
	out := make([]lineEdit, len(edits))
	shift := 0
	for i, e := range edits {
		n := utf8.RuneCountInString(e.text)
		out[i] = lineEdit{e.start + shift, e.start + shift + n, string(rs[e.start:e.end])}
		shift += n - (e.end - e.start)
	}
	return out
}

// mapThrough is where offset off of the text before goes in the text
// after edits (in order); inside an edit, its start (end: its end).
func mapThrough(edits []lineEdit, off int, end bool) int {
	shift := 0
	for _, e := range edits {
		n := utf8.RuneCountInString(e.text)
		switch {
		case off < e.start || (off == e.start && !end):
			return off + shift
		case off < e.end || (off == e.end && end && e.end > e.start):
			if end {
				return e.start + shift + n
			}
			return e.start + shift
		}
		shift += n - (e.end - e.start)
	}
	return off + shift
}

// followRenamed takes the extras with an Undo or Redo of the last rename
// or move: the text back as it was before, or again as after. Each is put
// back as it was kept then, if nothing changed it since; else followed
// by its edits (a merge's cast members cannot be told apart again: left
// as they are).
func (w *MainWindow) followRenamed() {
	r := w.followedLast
	if r == nil || w.textEditor == nil {
		return
	}
	text := w.textEditor.Text()
	var from, to int // the snapshots: as now, to put back
	switch {
	case r.atAfter && len(text) == len(r.before) && text == r.before:
		from, to = 1, 0
	case !r.atAfter && len(text) == len(r.after) && text == r.after:
		from, to = 0, 1
	default:
		return
	}
	r.atAfter = to == 1
	defer w.noteNumbers() // the extras are this change's: no numbers to rename after it
	bd, sched := w.extrasKept()
	follow, tagsFollow, stripsFollow := false, false, false
	if string(bd) == string(r.bd[from]) {
		if string(r.bd[to]) != string(r.bd[from]) {
			var f breakdownFile
			if json.Unmarshal(r.bd[to], &f) == nil {
				*w.breakdownNow() = f
				w.saveBreakdown()
			}
		}
	} else {
		follow, tagsFollow = true, true
	}
	if string(sched) == string(r.sched[from]) {
		if string(r.sched[to]) != string(r.sched[from]) {
			var f scheduleFile
			if json.Unmarshal(r.sched[to], &f) == nil {
				*w.scheduleNow() = f
				w.writeSchedule()
			}
		}
	} else {
		follow, stripsFollow = true, true
	}
	if !follow {
		return
	}
	if r.refollow == nil {
		if say := keptAsIs(to == 0, tagsFollow && string(r.bd[0]) != string(r.bd[1]),
			stripsFollow && string(r.sched[0]) != string(r.sched[1])); say != "" {
			w.setStatus(say)
		}
		return
	}
	r.refollow(to == 0, tagsFollow, stripsFollow)
}

// keptAsIs says that an Undo (back) or Redo of a merge was the script's
// only: the breakdown (bd), the schedule (sched) or both changed since,
// so they were left as they are, a merge's cast members not to be told
// apart again ("": neither; nothing to say).
func keptAsIs(back, bd, sched bool) string {
	var what string
	switch {
	case bd && sched:
		what = "the breakdown and the schedule"
	case bd:
		what = "the breakdown"
	case sched:
		what = "the schedule"
	default:
		return ""
	}
	done := "redone"
	if back {
		done = "undone"
	}
	return "Merge " + done + " in the script only: " + what + " changed since, so it was left as it is (its cast as you see it there)"
}

// followRename brings the breakdown and the schedule from the text before
// to after (edits): the cast member from takes the name to (merge: into
// to's own), the tags are anchored again where their words went, the
// scenes' keys and the strips' sums follow their headings.
func (w *MainWindow) followRename(before, after string, edits []lineEdit, from, to string, merge bool) {
	w.followEdit(before, after, renameEdit(before, after, w.sceneStarts(), edits, from, to), true, true)
}

// followEdit brings the breakdown (tags), the schedule (strips), or both
// from the text before to after, as ed says the text went.
func (w *MainWindow) followEdit(before, after string, ed extrasEdit, tags, strips bool) {
	from, to := ed.from, ed.to
	if w.markdown {
		return
	}
	bf := w.breakdownNow()
	hasTags := tags && w.breakdown.err == nil && (len(bf.Tags) > 0 || len(bf.Done) > 0 || len(bf.Elements) > 0)
	s := w.scheduleNow()
	hasStrips := strips && w.schedule.err == nil && len(s.Days) > 0
	if !hasTags && !hasStrips {
		return
	}
	starts := w.sceneStarts()
	oldBt, newBt := readBdText(before, starts), readBdText(after, starts)
	if ed.prepare != nil {
		ed.prepare(oldBt, newBt)
	}
	sceneAfter := func(i int) int {
		if j := ed.scene(i); j >= 0 && j < len(newBt.scenes) {
			return j
		}
		return -1
	}
	keyAfter := func(k bdSceneKey) (bdSceneKey, bool) {
		for i, sc := range oldBt.scenes {
			if sc.key == k {
				j := sceneAfter(i)
				if j < 0 {
					return k, false
				}
				n := newBt.scenes[j].key
				if n.Heading == k.Heading && n.Nth == k.Nth {
					return k, false
				}
				nk := k
				nk.Heading, nk.Nth = n.Heading, n.Nth
				return nk, true
			}
		}
		return k, false
	}
	if hasTags {
		changed := false
		// the cast member of the old name
		castTo := ""
		for _, e := range bf.Elements {
			if c, _ := bf.category(e.Category); from != "" && c.Name == lex.CastCategory && charKey(e.Name) == to {
				castTo = e.ID
			}
		}
		retarget := map[string]string{}
		for i, e := range bf.Elements {
			if c, _ := bf.category(e.Category); from == "" || c.Name != lex.CastCategory || charKey(e.Name) != from {
				continue
			}
			if castTo != "" && castTo != e.ID {
				retarget[e.ID] = castTo // merged into the other's
				continue
			}
			bf.Elements[i].Name = to
			changed = true
		}
		for i, p := range oldBt.place(bf) {
			t := &bf.Tags[i]
			if nu, ok := retarget[t.Element]; ok {
				t.Element, changed = nu, true
			}
			if p.lost {
				if k, ok := keyAfter(t.bdSceneKey); ok {
					t.bdSceneKey, changed = k, true
				}
				continue
			}
			ns, ne := ed.off(p.start, false), ed.off(p.end, true)
			if ns < 0 || ne < 0 {
				if k, ok := keyAfter(t.bdSceneKey); ok {
					t.bdSceneKey, changed = k, true
				}
				continue
			}
			wi, wj := newBt.wordRange(ns, ne)
			if wi >= wj {
				continue
			}
			key, at := newBt.anchorOf(wi, wj)
			if key.Scene == "" {
				key.Scene = t.Scene // a number only the tag had (Lock gives it)
			}
			if key != t.bdSceneKey || at != t.At {
				t.bdSceneKey, t.At, changed = key, at, true
			}
		}
		for i := range bf.Done {
			if k, ok := keyAfter(bf.Done[i]); ok {
				bf.Done[i], changed = k, true
			}
		}
		if len(retarget) > 0 {
			bf.pruneElements()
		}
		if changed {
			w.saveBreakdown()
		}
	}
	if !hasStrips {
		return
	}
	follow := func(f *scheduleFile) bool {
		p := f.place(oldBt)
		changed := false
		for d := range f.Days {
			for k := range f.Days[d].Strips {
				st := &f.Days[d].Strips[k]
				sc := p.days[d][k]
				if sc < 0 {
					continue
				}
				j := sceneAfter(sc)
				if j < 0 {
					continue
				}
				if n := newBt.scenes[j].key; n.Heading != st.Heading || n.Nth != st.Nth {
					st.Heading, st.Nth, changed = n.Heading, n.Nth, true
				}
				if ed.sums && st.Sum != "" && st.Sum == oldBt.sceneSum(sc) {
					if sum := newBt.sceneSum(j); sum != st.Sum {
						st.Sum, changed = sum, true
					}
				}
			}
		}
		return changed
	}
	if w.schedule.undo != nil {
		follow(w.schedule.undo)
	}
	if follow(s) {
		w.writeSchedule()
	}
}

// renameStatus says what a rename did.
func renameStatus(n int, old, name string, merge bool) string {
	if merge {
		return fmt.Sprintf("%s merged into %s: %s", old, name, plural(n, "cue"))
	}
	return fmt.Sprintf("%s renamed %s in %s (one undo step)", old, name, plural(n, "place"))
}

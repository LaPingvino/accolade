package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
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

// nameEdits are the edits in the script's body that rename the character
// with key to name: its cues (renameCues), or the name as a word wherever
// it stands (renameEverywhere). In runes, in order.
func nameEdits(text string, starts []string, key, name string, scope renameScope) []lineEdit {
	var out []lineEdit
	if scope == renameCues {
		m := screenplayChars(text, starts)
		for _, c := range m.chars {
			for _, s := range c.Cues {
				if charKey(s.name) == key {
					out = append(out, lineEdit{s.start, s.end, casedLike(s.name, name)})
				}
			}
		}
		slices.SortFunc(out, func(a, b lineEdit) int { return a.start - b.start })
		return out
	}
	body := []rune(text[:bodyEndByte(text)])
	want := []rune(key)
	if len(want) == 0 {
		return nil
	}
	for i := 0; i+len(want) <= len(body); i++ {
		if i > 0 && isWordRune(body[i-1]) {
			continue
		}
		if !sameUpper(body[i:i+len(want)], want) {
			continue
		}
		j := i + len(want)
		if j < len(body) && isWordRune(body[j]) {
			continue
		}
		was := string(body[i:j])
		out = append(out, lineEdit{i, j, casedLike(was, name)})
		i = j - 1
	}
	return out
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

// renamed is the last rename made, for Undo and Redo to take the
// breakdown and the schedule back and forth with it.
type renamed struct {
	before, after string
	fwd, back     []lineEdit // the edits from before to after, and back
	from, to      string     // the cast member's names (capitals)
	merge         bool
	atAfter       bool // the extras follow after
}

// renameCharacter renames the character with key to name (scope) as one
// undo step, the list, the learned notes and the extras following; it
// reports how many places changed.
func (w *MainWindow) renameCharacter(key, name string, scope renameScope) int {
	name = strings.Join(strings.Fields(name), " ")
	newKey := charKey(name)
	if w.textEditor == nil || key == "" || newKey == "" {
		return 0
	}
	e := w.textEditor
	before := e.Text()
	edits := nameEdits(before, w.sceneStarts(), key, name, scope)
	// the list: the entry renamed, or for Merge put into the other's
	list := charListOf(before)
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
	if inside, ok := charactersBlock(before); ok {
		if nu := list.inside(entries); nu != inside {
			s, t, block := withCharactersBlock(before, nu)
			blocks = append(blocks, lineEdit{s, t, block})
		}
	}
	// the learned notes: the cue's names under the new one
	l := learnedOf(before)
	for _, a := range l.Aliases {
		if charKey(a.Cue) == key {
			l.renameCue(a.Cue, casedLike(a.Cue, name))
			break
		}
	}
	if inside, ok := adaptationBlock(before); ok {
		if nu := withLearned(inside, l); nu != inside {
			s, t, block := withAdaptationBlock(before, nu)
			blocks = append(blocks, lineEdit{s, t, block})
		}
	}
	if len(edits) == 0 && len(blocks) == 0 {
		return 0
	}
	w.followNumbers(true) // the extras in line with the text before
	fwd := append(slices.Clone(edits), blocks...)
	slices.SortFunc(fwd, func(a, b lineEdit) int { return a.start - b.start })
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			for i := len(fwd) - 1; i >= 0; i-- { // the last first: the places before stay
				replaceLeast(b, fwd[i].start, fwd[i].end, fwd[i].text)
			}
		})
	})
	after := e.Text()
	r := &renamed{before: before, after: after, fwd: fwd, from: key, to: newKey, merge: scope == renameCues}
	r.back = invertEdits(before, r.fwd)
	w.charRenamed = r
	w.followRename(before, after, r.fwd, key, newKey, r.merge)
	r.atAfter = true
	w.followNumbers(true) // the headings that changed are the ones seen now
	return len(edits)
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

// followRenamed takes the extras with an Undo or Redo of the last rename:
// the text back as it was before, or again as after.
func (w *MainWindow) followRenamed() {
	r := w.charRenamed
	if r == nil || w.textEditor == nil {
		return
	}
	text := w.textEditor.Text()
	switch {
	case r.atAfter && len(text) == len(r.before) && text == r.before:
		w.followRename(r.after, r.before, r.back, r.to, r.from, false)
		r.atAfter = false
	case !r.atAfter && len(text) == len(r.after) && text == r.after:
		w.followRename(r.before, r.after, r.fwd, r.from, r.to, r.merge)
		r.atAfter = true
	}
}

// followRename brings the breakdown and the schedule from the text before
// to after (edits): the cast member from takes the name to (merge: into
// to's own), the tags are anchored again where their words went, the
// scenes' keys and the strips' sums follow their headings.
func (w *MainWindow) followRename(before, after string, edits []lineEdit, from, to string, merge bool) {
	if w.markdown {
		return
	}
	bf := w.breakdownNow()
	hasTags := w.breakdown.err == nil && (len(bf.Tags) > 0 || len(bf.Done) > 0 || len(bf.Elements) > 0)
	s := w.scheduleNow()
	hasStrips := w.schedule.err == nil && len(s.Days) > 0
	if !hasTags && !hasStrips {
		return
	}
	starts := w.sceneStarts()
	oldBt, newBt := readBdText(before, starts), readBdText(after, starts)
	// the scenes one for one (a rename adds none and takes none away)
	sameScenes := len(oldBt.scenes) == len(newBt.scenes)
	keyAfter := func(k bdSceneKey) (bdSceneKey, bool) {
		if !sameScenes {
			return k, false
		}
		for i, sc := range oldBt.scenes {
			if sc.key == k {
				n := newBt.scenes[i].key
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
			if c, _ := bf.category(e.Category); c.Name == lex.CastCategory && charKey(e.Name) == to {
				castTo = e.ID
			}
		}
		retarget := map[string]string{}
		for i, e := range bf.Elements {
			if c, _ := bf.category(e.Category); c.Name != lex.CastCategory || charKey(e.Name) != from {
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
			ns, ne := mapThrough(edits, p.start, false), mapThrough(edits, p.end, true)
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
				if sc < 0 || !sameScenes {
					continue
				}
				if n := newBt.scenes[sc].key; n.Heading != st.Heading || n.Nth != st.Nth {
					st.Heading, st.Nth, changed = n.Heading, n.Nth, true
				}
				if st.Sum != "" && st.Sum == oldBt.sceneSum(sc) {
					if sum := newBt.sceneSum(sc); sum != st.Sum {
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

package main

import (
	"fmt"
	"slices"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Format > Update Source Links: the old form's links and block entries
// ([[from: "…"]], cut: "…" (2)) written in the new one (textfrag.go), in
// one undo step. Only what is found is written again, for the passages
// it found (a near one keeps its words, so it still says the source
// changed); an ambiguous or missing link, or one the new form cannot
// tell from another place, stays as it was.

// linkEdit is a piece of the script replaced: [Start, End) in runes.
type linkEdit struct {
	Start, End int
	Text       string
}

// updateLinksEdits are the edits that update text's links, the last
// first; how many links and entries were written anew, and how many were
// left as they were.
func updateLinksEdits(text string, lm *linkMatcher) (edits []linkEdit, updated, left int) {
	newNote := func(r linkRef, mt linkMatch) (string, bool) {
		switch mt.Kind {
		case linkFound:
			return linkNoteFor(lm, mt.Passages[0], mt.last())
		case linkNear: // its words kept: still near, still flagged
			nr := linkRef{Dir: textDirective{Start: r.Quote}}
			if again := lm.matchRef(nr); again.found() && again.Passages[0] == mt.Passages[0] {
				return fragmentNote("", []textDirective{nr.Dir}), true
			}
		}
		return "", false
	}
	for _, l := range parseLinks(text) {
		if !l.Ref.Old {
			continue
		}
		if note, ok := newNote(l.Ref, lm.matchRef(l.Ref)); ok {
			edits = append(edits, linkEdit{l.Start, l.End, note})
			updated++
		} else {
			left++
		}
	}
	// the block: cut: and skipped links:
	inside, ok := adaptationBlock(text)
	if ok {
		entry := func(c cutMark) (cutMark, bool) {
			if !c.ref().Old {
				return c, false
			}
			sp := cutAt(c, lm.matchRef(c.ref()))
			if sp.first < 0 {
				left++
				return c, false
			}
			d, unique := fragmentFor(lm, sp.first, sp.last)
			if !unique {
				left++
				return c, false
			}
			updated++
			return cutOf(linkRef{Dir: d}), true
		}
		nu := inside
		cuts := parseCuts(inside)
		changed := false
		for i, c := range cuts {
			if c2, ok := entry(c); ok {
				cuts[i], changed = c2, true
			}
		}
		if changed {
			nu = withCuts(nu, cuts)
		}
		l := parseLearned(nu)
		changed = false
		for i, s := range l.Skipped {
			if c2, ok := entry(s.Link); ok {
				l.Skipped[i].Link, changed = c2, true
			}
		}
		if changed {
			nu = withLearned(nu, l)
		}
		if nu != inside {
			start, end, block := withAdaptationBlock(text, nu)
			edits = append(edits, linkEdit{start, end, block})
		}
	}
	slices.SortStableFunc(edits, func(a, b linkEdit) int { return b.Start - a.Start })
	return edits, updated, left
}

// updateSourceLinks is Format > Update Source Links.
func (w *MainWindow) updateSourceLinks() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	s.load()
	switch {
	case s.ref.Origin == sourceNone:
		w.setStatus("No source: File > Choose Source…")
		return
	case s.loading != nil:
		w.setStatus("The source is being read: Update Source Links again in a moment")
		return
	}
	e := w.textEditor
	edits, updated, left := updateLinksEdits(e.Text(), matcherFor(s.passages))
	if len(edits) > 0 {
		e.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				cursor := b.Cursor()
				for _, ed := range edits { // the last first: the offsets before stay
					b.Replace(ed.Start, ed.End, ed.Text)
					switch {
					case cursor >= ed.End:
						cursor += len([]rune(ed.Text)) - (ed.End - ed.Start)
					case cursor > ed.Start:
						cursor = ed.Start
					}
				}
				b.SetCursor(cursor, false)
			})
		})
	}
	switch {
	case updated == 0 && left == 0:
		w.setStatus("The source links are all in the new form already")
	case left == 0:
		w.setStatus(fmt.Sprintf("Updated %s to the new form", plural(updated, "source link")))
	default:
		w.setStatus(fmt.Sprintf("Updated %s to the new form; %d left as they were (ambiguous or not found)",
			plural(updated, "source link"), left))
	}
	s.follow()
}

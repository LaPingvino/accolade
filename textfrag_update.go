package main

import (
	"fmt"
	"slices"
)

// Adaptation > Update Source Links: the old form's links and block entries
// ([[from: "…"]], cut: "…" (2)) written in the new one (textfrag.go), in
// one undo step, for the passages they find; a link of either form that
// only nearly finds its passages is re-anchored to their words now
// (relink.go). An ambiguous or missing link, or one the new form cannot
// tell from another place, stays as it was (its chip warns).

// linkEdit is a piece of the script replaced: [Start, End) in runes.
type linkEdit struct {
	Start, End int
	Text       string
}

// updateLinksEdits are the edits that update text's links, the last
// first; how many links and entries were written anew, and how many were
// left as they were.
func updateLinksEdits(text string, lm *linkMatcher) (edits []linkEdit, updated, left int) {
	links := parseLinks(text)
	for a := 0; a < len(links); {
		b := a // the note's links: one note may hold several directives
		for b < len(links) && links[b].Start == links[a].Start {
			b++
		}
		note := links[a:b]
		a = b
		var dirs []textDirective
		changed := false
		for _, l := range note {
			dir := l.Ref.Dir
			if l.Ref.Old {
				dir = textDirective{Start: l.Ref.Quote}
			}
			if mt := lm.matchRef(l.Ref); l.Ref.Bad == "" && (mt.Kind == linkNear || l.Ref.Old && mt.Kind == linkFound) {
				if d, unique := fragmentFor(lm, mt.Passages[0], mt.last()); unique {
					dir, changed = d, true
					updated++
					dirs = append(dirs, dir)
					continue
				}
			}
			if l.Ref.Old {
				left++
			}
			dirs = append(dirs, dir)
		}
		if changed {
			edits = append(edits, linkEdit{note[0].Start, note[0].End, fragmentNote(note[0].Ref.File, dirs)})
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

// updateSourceLinks is Adaptation > Update Source Links.
func (w *MainWindow) updateSourceLinks() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	s.load()
	switch {
	case s.ref.Origin == sourceNone:
		w.setStatus("No source: Adaptation > Choose Source…")
		return
	case s.loading != nil:
		w.setStatus("The source is being read: Update Source Links again in a moment")
		return
	}
	e := w.textEditor
	edits, updated, left := updateLinksEdits(e.Text(), matcherFor(s.passages))
	if len(edits) > 0 {
		w.applyLinkEdits(edits)
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

package main

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Editing a passage's opening words (LOOP.md 2026-11-21): the script's
// [[from:]] links that found that passage by its old words no longer
// find it, or only nearly. When the writer leaves a passage changed so
// (the source in the main editor, or the Source pane's Done, or Save),
// a notice above the status bar offers to write those links again for
// the passage's new words, and the adaptation block's entries quoting it
// (cut:, skipped links:) with them. Never silently: until Update Links is
// given they stay as they were. The update is one undo step of the
// script's.

// linkOfferNotice is the notice: what it says, and the passages before
// and after the change it is about.
type linkOfferNotice struct {
	box          *fyne.Container
	text         *widget.Label
	update, keep *widget.Button
	old, nw      []passage
}

func (w *MainWindow) newLinkOffer() *fyne.Container {
	n := &linkOfferNotice{text: widget.NewLabel("")}
	n.text.Wrapping = fyne.TextWrapWord
	n.update = widget.NewButtonWithIcon("Update Links", theme.ViewRefreshIcon(), w.updateOfferedLinks)
	n.update.Importance = widget.HighImportance
	n.keep = widget.NewButton("Not Now", w.hideLinkOffer)
	n.box = container.NewVBox(widget.NewSeparator(), n.text, container.NewHBox(layout.NewSpacer(), n.keep, n.update))
	n.box.Hide()
	w.linkOffer = n
	return n.box
}

// linkFixes are the edits that write the script's links again for the
// passages' new words (the last first), and how many links they write;
// entries, how many of the adaptation block's entries (cut:, skipped
// links:) they write. A link is written again when it found passages
// [i, j] in old exactly, those passages are still there in nw
// (alignPassages), and it does not find exactly them there now; one that
// cannot be told apart from another place in nw is left as it is. An
// entry the same (fixedCut).
func linkFixes(script string, old, nw []passage) (edits []linkEdit, n, entries int) {
	if len(old) == 0 || len(nw) == 0 {
		return nil, 0, 0
	}
	before, after := newLinkMatcher(old), newLinkMatcher(nw)
	to := alignPassages(old, nw)
	links := parseLinks(script)
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
			if d, ok := fixedLink(l.Ref, before, after, to); ok {
				dir, changed = d, true
				n++
			}
			dirs = append(dirs, dir)
		}
		if changed {
			edits = append(edits, linkEdit{note[0].Start, note[0].End, fragmentNote(note[0].Ref.File, dirs)})
		}
	}
	// the adaptation block's entries quoting the passages: one edit of
	// the block (a boneyard, where parseLinks finds no links)
	if inside, ok := adaptationBlock(script); ok {
		nu := inside
		cuts, changed := parseCuts(inside), false
		for i, c := range cuts {
			if c2, ok := fixedCut(c, before, after, to); ok {
				cuts[i], changed = c2, true
				entries++
			}
		}
		if changed {
			nu = withCuts(nu, cuts)
		}
		l, changed := parseLearned(nu), false
		for i, s := range l.Skipped {
			if c2, ok := fixedCut(s.Link, before, after, to); ok {
				l.Skipped[i].Link, changed = c2, true
				entries++
			}
		}
		if changed {
			nu = withLearned(nu, l)
		}
		if nu != inside {
			start, end, block := withAdaptationBlock(script, nu)
			edits = append(edits, linkEdit{start, end, block})
		}
	}
	slices.SortStableFunc(edits, func(x, y linkEdit) int { return y.Start - x.Start })
	return edits, n, entries
}

// fixedCut is an entry of the adaptation block written for its
// passages' new words, if it is to change: as fixedLink, it named its
// passages exactly before (an old-form entry's "(2)" one of passages
// alike), they changed and are still there, it does not name them now,
// and the new form tells them from any other place.
func fixedCut(c cutMark, before, after *linkMatcher, to []int) (cutMark, bool) {
	r := c.ref()
	if r.Bad != "" {
		return c, false
	}
	mt := before.matchRef(r)
	if mt.Kind != linkFound && !(c.Nth > 0 && mt.Kind == linkAmbiguous) {
		return c, false
	}
	was := cutAt(c, mt)
	if was.first < 0 {
		return c, false
	}
	i, j := to[was.first], to[was.last]
	if i < 0 || j < i {
		return c, false
	}
	if before.ps[was.first].Plain == after.ps[i].Plain && before.ps[was.last].Plain == after.ps[j].Plain {
		return c, false // its passages' words are as they were
	}
	if now := cutAt(c, after.matchRef(r)); now.first == i && now.last == j {
		return c, false // it names them still
	}
	d, unique := fragmentFor(after, i, j)
	if !unique {
		return c, false
	}
	return cutOf(linkRef{Dir: d}), true
}

// fixesSaid is how many links and entries an offer is about, in words.
func fixesSaid(links, entries int) string {
	var parts []string
	if links > 0 {
		parts = append(parts, plural(links, "link"))
	}
	if entries > 0 {
		e := "1 entry"
		if entries > 1 {
			e = fmt.Sprint(entries, " entries")
		}
		parts = append(parts, e+" of the adaptation block")
	}
	return strings.Join(parts, " and ")
}

// fixedLink is the directive a link is written as for the passages'
// new words, if it is to change.
//
// Only a link that found its passages exactly before is written again,
// and only when they themselves changed: a link already near (flagged as
// the source having changed since it was linked) keeps its words and its
// flag, as Update Source Links leaves it.
func fixedLink(r linkRef, before, after *linkMatcher, to []int) (textDirective, bool) {
	was := before.matchRef(r)
	if was.Kind != linkFound {
		return textDirective{}, false
	}
	i, j := to[was.Passages[0]], to[was.last()]
	if i < 0 || j < i {
		return textDirective{}, false
	}
	if before.ps[was.Passages[0]].Plain == after.ps[i].Plain && before.ps[was.last()].Plain == after.ps[j].Plain {
		return textDirective{}, false // its passages' words are as they were
	}
	if now := after.matchRef(r); now.Kind == linkFound && now.Passages[0] == i && now.last() == j {
		return textDirective{}, false // it finds them still
	}
	d, unique := fragmentFor(after, i, j)
	return d, unique
}

// alignPassages is where each passage of old is in nw (-1: gone): the
// same passages before and after the change, and those between them in
// order, the last of a run to the last (a passage split in two).
func alignPassages(old, nw []passage) []int {
	to := make([]int, len(old))
	p := 0
	for p < len(old) && p < len(nw) && old[p].Plain == nw[p].Plain {
		to[p] = p
		p++
	}
	q := 0
	for q < len(old)-p && q < len(nw)-p && old[len(old)-1-q].Plain == nw[len(nw)-1-q].Plain {
		to[len(old)-1-q] = len(nw) - 1 - q
		q++
	}
	mo, mn := len(old)-p-q, len(nw)-p-q // the changed runs
	for k := 0; k < mo; k++ {
		switch {
		case mn == 0:
			to[p+k] = -1
		case k == mo-1:
			to[p+k] = p + mn - 1
		default:
			to[p+k] = p + min(k, mn-1)
		}
	}
	return to
}

// offerLinkUpdate looks at what the source document d's edits did to the
// links: if links quote a passage whose opening changed, the notice
// offers to update them. The passages it measures from move on with each
// look (the notice keeps its own).
func (w *MainWindow) offerLinkUpdate(d *document) {
	if d == nil || d.kind != docSource || w.markdown || w.linkOffer == nil {
		return
	}
	text := d.Text()
	if sameString(text, d.linkBase) {
		return
	}
	base := d.linkBase
	n := w.linkOffer
	if n.box.Visible() && n.old != nil {
		base = "" // an offer not answered yet: from where it began (n.old)
	}
	d.linkBase = text
	script := w.textEditor.Text()
	type found struct {
		old, nw        []passage
		links, entries int
	}
	old := n.old
	// a novel's passages take a while: read in the background, the
	// writing going on meanwhile
	inBackground(func() found {
		o := old
		if base != "" || o == nil {
			o = passagesOf(base)
		}
		nw := passagesOf(text)
		_, links, entries := linkFixes(script, o, nw)
		return found{o, nw, links, entries}
	}, func(f found) {
		if f.links+f.entries == 0 || w.textEditor.Text() != script {
			return // nothing to ask, or the script changed meanwhile: asked again on the next look
		}
		n.old, n.nw = f.old, f.nw
		say := fmt.Sprintf("A passage of the source opens with other words now: %s in the script %s its old words. "+
			"Update them to the new ones?", fixesSaid(f.links, f.entries), map[bool]string{true: "quotes", false: "quote"}[f.links+f.entries == 1])
		if n.box.Visible() && n.text.Text == say {
			return // shown as it is: laying the whole window out again on every key was most of typing's cost
		}
		n.text.SetText(say)
		n.box.Show()
		w.fyneWindow.Content().Refresh()
	})
}

// updateOfferedLinks is the notice's Update Links: one undo step.
func (w *MainWindow) updateOfferedLinks() {
	n := w.linkOffer
	edits, links, entries := linkFixes(w.textEditor.Text(), n.old, n.nw)
	w.hideLinkOffer()
	if len(edits) == 0 {
		w.setStatus("The links find their passages already")
		return
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
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
	w.setStatus(fmt.Sprintf("Updated %s to the passages' new words (Undo takes them back)", fixesSaid(links, entries)))
	if w.source != nil {
		w.source.follow()
	}
}

// hideLinkOffer is the notice's Not Now: the links stay as they are.
func (w *MainWindow) hideLinkOffer() {
	if n := w.linkOffer; n != nil && n.box.Visible() {
		n.box.Hide()
		n.old, n.nw = nil, nil
		w.fyneWindow.Content().Refresh()
	}
}

// linksOfferOnLeave is the main cursor moving: when it leaves a passage
// of the source after the text changed, the links are looked at.
func (w *MainWindow) linksOfferOnLeave() {
	d := w.mainDoc
	if d == nil || d.kind != docSource || w.source == nil {
		return
	}
	s := w.source
	if s.loading != nil {
		return
	}
	b := d.ed.Buffer()
	line, _ := b.Position(b.Cursor())
	if line == d.atLine { // typing in its line (a paragraph is one): the same passage
		return
	}
	d.atLine = line
	p := passageAt(d.Text(), s.passages, b, &w.scriptPane().seen)
	if p != d.atPassage {
		if d.atPassage >= 0 && b.Rev() != d.seenRev {
			d.seenRev = b.Rev()
			w.offerLinkUpdate(d)
		}
		d.atPassage = p
	}
}

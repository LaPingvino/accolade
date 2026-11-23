package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
)

// Source links as chips (LOOP.md, Joop 2026-11-20): in the script's
// editor a [[from: …]] note is drawn as a small chip, "↗ source" (or
// "↗ 3 passages" for a link to a range, or a note naming several; a
// warning, "↗ source?" or "↗ 2 places", for one that cannot find its
// passage alone), its
// quote decoded on hover or a long press: the words as the source has
// them, commas and all, a range as "first words … last words". The
// Text Fragments form, percent-encoding and all (textfrag.go), is the
// file's and other apps'; the writer sees it only with the cursor inside
// the link, where it is the text as it is. Display only, as the blocks'
// folds (block_folds.go).

// chipCache is the script's chips as found last: for its text, and the
// source pane's matcher then (what tells how many passages a link has).
type chipCache struct {
	text  string
	lm    *linkMatcher
	chips []editor.Chip
	ok    bool
	// lines are the chips of each line with links, by its text (columns
	// from the line's start): a key reads its own line's links again only
	lines map[string][]editor.Chip
}

// linkChips are the script's links as chips: the editor's Chips. The
// body only (not the tail blocks, nor a boneyard: as parseLinks), the
// same slice for the same text.
func (w *MainWindow) linkChips(text string) []editor.Chip {
	if w.markdown {
		return nil
	}
	var lm *linkMatcher
	if w.source != nil {
		lm = w.source.matcher
	}
	c := &w.chips
	if c.ok && c.lm == lm && sameString(c.text, text) {
		return c.chips
	}
	if c.lm != lm {
		c.lines = nil // what links find changed
	}
	lines := map[string][]editor.Chip{}
	c.chips = findLinkChipsIn(text[:bodyEndByte(text)], lm, w.linkGuesses, func(line string, find func() []editor.Chip) []editor.Chip {
		cs, ok := c.lines[line]
		if !ok {
			cs = find()
		}
		lines[line] = cs
		return cs
	})
	c.text, c.lm, c.ok, c.lines = text, lm, true, lines
	return c.chips
}

// findLinkChips finds the links in a script's body.
func findLinkChips(body string, lm *linkMatcher) []editor.Chip {
	return findLinkChipsIn(body, lm, nil, func(_ string, find func() []editor.Chip) []editor.Chip { return find() })
}

// findLinkChipsIn is findLinkChips, a line's chips (columns from its
// start) asked of kept: what find finds, or what was found for the same
// line before.
func findLinkChipsIn(body string, lm *linkMatcher, guesses map[string]linkGuess, kept func(line string, find func() []editor.Chip) []editor.Chip) []editor.Chip {
	var out []editor.Chip
	var refs []linkRef
	off := 0 // in runes, of the line
	inBoneyard := false
	for len(body) > 0 {
		line := body
		if i := strings.IndexByte(body, '\n'); i >= 0 {
			line, body = body[:i], body[i+1:]
		} else {
			body = ""
		}
		t := strings.TrimLeft(line, " \t")
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(strings.TrimSpace(t), "/*"), "*/")
		} else if strings.Contains(line, "[[") {
			for _, c := range kept(line, func() []editor.Chip { return lineChips(line, lm, guesses, &refs) }) {
				c.Start += off
				c.End += off
				out = append(out, c)
			}
		}
		off += utf8.RuneCountInString(line) + 1
	}
	return out
}

// lineChips are the links of a line, from its start.
func lineChips(line string, lm *linkMatcher, guesses map[string]linkGuess, refs *[]linkRef) []editor.Chip {
	var out []editor.Chip
	for at := 0; ; {
		s := strings.Index(line[at:], "[[")
		if s < 0 {
			break
		}
		s += at
		e := strings.Index(line[s:], "]]")
		if e < 0 {
			break
		}
		e += s + 2
		if *refs = appendLinkRefs((*refs)[:0], line[s+2:e-2]); len(*refs) > 0 {
			start := utf8.RuneCountInString(line[:s])
			label, tip, warn := chipText(*refs, lm, guesses)
			out = append(out, editor.Chip{Start: start, End: start + utf8.RuneCountInString(line[s:e]), Label: label, Tip: tip, Warn: warn})
		}
		at = e
	}
	return out
}

// chipText is a link's chip, and its tip: the quote, decoded. A link
// that cannot find its passages alone warns (relink.go): "↗ source?" for
// one not found, "↗ 2 places" for one that fits more than one place, the
// tip saying why.
func chipText(refs []linkRef, lm *linkMatcher, guesses map[string]linkGuess) (label, tip string, warn bool) {
	n, lost, places := 0, false, 0
	var check []string // the guesses' words (relink.go)
	quotes := make([]string, 0, len(refs))
	for _, r := range refs {
		quotes = append(quotes, readableQuote(r))
		if g, ok := guesses[r.rawKey()]; ok && lm != nil {
			if g.Target && lm.matchRef(linkRef{Dir: g.Dir}).found() {
				check = append(check, "("+g.Why+": "+readableQuote(linkRef{Dir: g.Dir})+". Tap the chip to move it there.)")
				continue
			}
			if !g.Target {
				lost = true
				check = append(check, "("+g.Why+")")
				continue
			}
		}
		if lm == nil {
			n++
			continue
		}
		switch mt := lm.matchRef(r); {
		case mt.found():
			n += mt.last() - mt.Passages[0] + 1
		case mt.Kind == linkAmbiguous:
			places = max(places, len(mt.Passages))
		default:
			lost = true
		}
	}
	switch {
	case lost:
		label = "↗ source?"
	case len(check) > 0:
		label = "↗ check?"
	case places > 1:
		label = "↗ " + groupDigits(places) + " places"
	case n > 1:
		label = "↗ " + groupDigits(n) + " passages"
	default:
		label = "↗ source"
	}
	tip = strings.Join(quotes, "\n")
	switch {
	case len(check) > 0:
		tip += "\n" + strings.Join(check, "\n")
	case lost:
		tip += "\n(not in the source as it is now: its passage was taken out, or its words changed past finding)"
	case places > 1:
		tip += "\n(fits " + groupDigits(places) + " places in the source alike, and nothing around them tells them apart: " +
			"which one the scene adapts is not known)"
	case lm != nil && n > 1:
		tip += "\n(" + groupDigits(n) + " passages of the source)"
	}
	return label, tip, lost || places > 1 || len(check) > 0
}

// readableQuote is what a link names as a writer reads it: its words
// decoded, in quotes, a range as "first words … last words", the words
// around it that tell it apart said so. Never the percent-encoding.
func readableQuote(r linkRef) string {
	switch {
	case r.Old:
		return "“" + r.Quote + "”"
	case r.Bad != "":
		return "a link Accolade cannot read: " + r.Bad
	}
	d := r.Dir
	q := "“" + d.Start
	if d.End != "" {
		q += " … " + d.End
	}
	q += "”"
	switch {
	case d.Prefix != "" && d.Suffix != "":
		q += " (after “" + d.Prefix + "”, before “" + d.Suffix + "”)"
	case d.Prefix != "":
		q += " (after “" + d.Prefix + "”)"
	case d.Suffix != "":
		q += " (before “" + d.Suffix + "”)"
	}
	if r.File != "" {
		q += " in " + r.File
	}
	return q
}

// chipHover shows a chip's quote while the pointer is on it in the
// script's editor (a footnote's tip's way: noteHover).
func (w *MainWindow) chipHover(offset int, at fyne.Position, cell fyne.Size) {
	tip := ""
	if offset >= 0 {
		if c, ok := w.textEditor.ChipAt(offset); ok {
			tip = c.Tip
		}
	}
	w.showEditorTip(tip, at, cell)
}

// chipPressed shows a chip's quote on a long press (a right click).
func (w *MainWindow) chipPressed(c editor.Chip, at fyne.Position, cell fyne.Size) {
	if _, _, ok := w.chipGuess(c); ok { // a guess to check (relink.go): Move It There or Keep As It Is
		menu := fyne.NewMenu("",
			fyne.NewMenuItem("Move It There", func() { w.moveLinkToGuess(c) }),
			fyne.NewMenuItem("Keep As It Is", func() { w.keepLinkAsIs(c) }))
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(w.textEditor).Add(at).AddXY(0, cell.Height)
		widget.ShowPopUpMenuAtPosition(menu, w.fyneWindow.Canvas(), pos)
		w.showEditorTip(c.Tip, at, cell)
		return
	}
	if w.tooltips == nil {
		w.setStatus(strings.ReplaceAll(c.Tip, "\n", " "))
		return
	}
	w.showEditorTip(c.Tip, at, cell)
}

// showEditorTip shows tip under the cell at in the script's editor, or
// hides the one shown ("").
func (w *MainWindow) showEditorTip(tip string, at fyne.Position, cell fyne.Size) {
	if w.tooltips == nil || tip == w.noteTipShown {
		return
	}
	if w.noteTipShown != "" && w.tooltips.Text() == w.noteTipShown {
		w.tooltips.HideTip()
	}
	w.noteTipShown = tip
	if tip != "" {
		w.tooltips.ShowTipUnder(tip, w.textEditor, at, cell)
	}
}

// readableLinks is text (a scene the Adapt dialog will write) with its
// links shown as the writer reads them, "[[↗ “first words … last
// words”]]", and back, which puts the links' own text in again where the
// readable form is still as it was shown (written over by hand, it goes
// in as written).
func readableLinks(text string) (shown string, back func(string) string) {
	raw := map[string]string{}
	var b strings.Builder
	rs := []rune(text)
	at := 0
	for _, c := range findLinkChips(text, nil) {
		if c.Start < at || c.End > len(rs) {
			continue
		}
		r := "[[↗ " + strings.ReplaceAll(c.Tip, "\n", "; ") + "]]"
		raw[r] = string(rs[c.Start:c.End])
		b.WriteString(string(rs[at:c.Start]))
		b.WriteString(r)
		at = c.End
	}
	if len(raw) == 0 {
		return text, func(s string) string { return s }
	}
	b.WriteString(string(rs[at:]))
	return b.String(), func(s string) string {
		for r, note := range raw {
			s = strings.ReplaceAll(s, r, note)
		}
		return s
	}
}

// chipGuess is the note a chip stands for, its links, and whether one of
// them has a guess to move it to (relink.go).
func (w *MainWindow) chipGuess(c editor.Chip) (note string, refs []linkRef, ok bool) {
	rs := []rune(w.textEditor.Text())
	if c.Start < 0 || c.End > len(rs) || c.End-c.Start < 4 {
		return "", nil, false
	}
	note = string(rs[c.Start:c.End])
	refs = appendLinkRefs(nil, note[2:len(note)-2])
	for _, r := range refs {
		if g, has := w.linkGuesses[r.rawKey()]; has && g.Target {
			ok = true
		}
	}
	return note, refs, ok
}

// moveLinkToGuess writes a chip's link for the passage its guess is
// about: one undo step.
func (w *MainWindow) moveLinkToGuess(c editor.Chip) {
	note, refs, ok := w.chipGuess(c)
	if !ok {
		return
	}
	if w.source != nil && w.source.matcher != nil {
		for _, r := range refs {
			if g, has := w.linkGuesses[r.rawKey()]; has && g.Target && !w.source.matcher.matchRef(linkRef{Dir: g.Dir}).found() {
				w.setStatus("The source changed again: that passage is not there any more")
				return
			}
		}
	}
	dirs := make([]textDirective, len(refs))
	file := ""
	for k, r := range refs {
		dirs[k] = r.Dir
		if r.Old {
			dirs[k] = textDirective{Start: r.Quote}
		}
		file = r.File
		if g, has := w.linkGuesses[r.rawKey()]; has && g.Target {
			dirs[k] = g.Dir
			delete(w.linkGuesses, r.rawKey())
		}
	}
	nu := fragmentNote(file, dirs)
	if nu == note {
		return
	}
	w.applyLinkEdits([]linkEdit{{c.Start, c.End, nu}})
	w.guessesChanged()
	w.setStatus("The link moved to " + readableQuote(linkRef{Dir: dirs[0]}) + " (Undo takes it back)")
	if w.source != nil {
		w.source.follow()
	}
}

// keepLinkAsIs drops a chip's guesses: the link stays as it is, its chip
// as its words find it.
func (w *MainWindow) keepLinkAsIs(c editor.Chip) {
	_, refs, _ := w.chipGuess(c)
	for _, r := range refs {
		delete(w.linkGuesses, r.rawKey())
	}
	w.guessesChanged()
}

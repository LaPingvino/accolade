package main

import (
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor"
)

// Source links as chips (LOOP.md, Joop 2026-11-20): in the script's
// editor a [[from: …]] note is drawn as a small chip, "↗ source" (or
// "↗ 3 passages" for a link to a range, or a note naming several), its
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
	c.chips = findLinkChipsIn(text[:bodyEndByte(text)], lm, func(line string, find func() []editor.Chip) []editor.Chip {
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
	return findLinkChipsIn(body, lm, func(_ string, find func() []editor.Chip) []editor.Chip { return find() })
}

// findLinkChipsIn is findLinkChips, a line's chips (columns from its
// start) asked of kept: what find finds, or what was found for the same
// line before.
func findLinkChipsIn(body string, lm *linkMatcher, kept func(line string, find func() []editor.Chip) []editor.Chip) []editor.Chip {
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
			for _, c := range kept(line, func() []editor.Chip { return lineChips(line, lm, &refs) }) {
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
func lineChips(line string, lm *linkMatcher, refs *[]linkRef) []editor.Chip {
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
			label, tip := chipText(*refs, lm)
			out = append(out, editor.Chip{Start: start, End: start + utf8.RuneCountInString(line[s:e]), Label: label, Tip: tip})
		}
		at = e
	}
	return out
}

// chipText is a link's chip, and its tip: the quote, decoded.
func chipText(refs []linkRef, lm *linkMatcher) (label, tip string) {
	n, lost := 0, false
	quotes := make([]string, 0, len(refs))
	for _, r := range refs {
		quotes = append(quotes, readableQuote(r))
		if lm == nil {
			n++
			continue
		}
		if mt := lm.matchRef(r); mt.found() {
			n += mt.last() - mt.Passages[0] + 1
		} else {
			lost = true
		}
	}
	switch {
	case lost:
		label = "↗ source?"
	case n > 1:
		label = "↗ " + groupDigits(n) + " passages"
	default:
		label = "↗ source"
	}
	tip = strings.Join(quotes, "\n")
	switch {
	case lost:
		tip += "\n(not found in the source as it is now: Adaptation > Update Source Links)"
	case lm != nil && n > 1:
		tip += "\n(" + groupDigits(n) + " passages of the source)"
	}
	return label, tip
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

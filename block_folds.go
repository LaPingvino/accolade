// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Accolade's own blocks fold away (LOOP.md, Joop 2026-11-20): in the
// script's editor the boneyard blocks at its end (/* Characters */,
// /* Source */, /* Adaptation */, /* Production */, /* Notes */) and an
// OMITTED scene's kept boneyard ("/* Omitted 3 … */") are folded by
// default, each to one quiet line that says what it holds:
//
//	▸ Production · scene numbers locked · pages locked · Blue
//	▸ Characters · 5, 4 with notes
//	▸ Omitted scene · INT. TAVERN - DAY #3#, 3 lines
//
// Display only: the text is the text, for the file, other apps, peers
// and screen readers (which read the editor's whole text). A block opens
// while the cursor is in it (a click on its line, the arrows, a Find
// hit) and folds again when the cursor leaves; opened by hand (its ▸ in
// the gutter, View > Fold or Unfold Here, vim's zo, Unfold All) it stays
// open until folded by hand (▾, zc, za, Fold All). The blocks are found
// once per text, their summaries made only for the rows drawn.

// blockKind is which of Accolade's blocks a fold is.
type blockKind int

const (
	blockNotes blockKind = iota
	blockSource
	blockAdaptation
	blockProduction
	blockCharacters
	blockOmitted
)

// blockFold is one of the blocks in the script: its lines [first, last]
// (the "/* …" line to the "*/" one), what is inside it in bytes, and its
// key, by which it is remembered as opened by hand.
type blockFold struct {
	kind        blockKind
	key         string
	first, last int
	in, inEnd   int // the inside, in bytes of the text
}

// blockLabel is a block's summary as made last, for the inside it was
// made from.
type blockLabel struct {
	inside, label string
}

// blocksOf are the script's blocks that fold, read once a text.
func (f *folds) blocksOf() []blockFold {
	if !f.blocks || f.w.markdown {
		return nil
	}
	b := &f.bl
	if rev := f.ed.Buffer().Rev(); !b.ok || b.rev != rev {
		b.list, b.rev, b.ok = findBlockFolds(f.ed.Text()), rev, true
	}
	return b.list
}

// findBlockFolds finds the tail blocks (as tailSetOf has them) and the
// OMITTED scenes' boneyards in the script's body.
func findBlockFolds(text string) []blockFold {
	var out []blockFold
	ts := tailSetOf(text)
	lines := strings.Count(text, "\n") + 1
	lineOf := func(b int) int { return lines - 1 - strings.Count(text[b:], "\n") } // the tail is short
	add := func(kind blockKind, name string, start, end int) {
		for start < end && (text[start] == '\n' || text[start] == ' ' || text[start] == '\t' || text[start] == '\r') {
			start++
		}
		if !strings.HasPrefix(text[start:end], "/*") {
			return
		}
		close := strings.LastIndex(text[start:end], "*/")
		if close < 0 {
			return
		}
		eol := strings.IndexByte(text[start:end], '\n')
		if eol < 0 {
			eol = end - start
		}
		in := start + eol + 1
		inEnd := max(start+close, in)
		first, last := lineOf(start), lineOf(start+close)
		if last > first {
			out = append(out, blockFold{kind: kind, key: "tail:" + name, first: first, last: last, in: min(in, inEnd), inEnd: inEnd})
		}
	}
	// the body's OMITTED scenes, in order
	body := bodyEndByte(text)
	at, line := 0, 0
	for {
		i := strings.Index(text[at:body], "\n/* Omitted")
		if i < 0 {
			break
		}
		open := at + i + 1
		line += strings.Count(text[at:open], "\n")
		close := strings.Index(text[open:body], "*/")
		if close < 0 {
			break
		}
		close += open
		last := line + strings.Count(text[open:close], "\n")
		eol := strings.IndexByte(text[open:], '\n') + open
		if last > line {
			out = append(out, blockFold{kind: blockOmitted, key: "omit:" + strings.TrimSpace(text[open:eol]) + "@" + strconv.Itoa(len(out)),
				first: line, last: last, in: eol + 1, inEnd: max(close, eol+1)})
		}
		line, at = last, close
	}
	kinds := []struct {
		t    tailBlock
		kind blockKind
		name string
	}{{ts.chars, blockCharacters, "Characters"}, {ts.src, blockSource, "Source"}, {ts.adapt, blockAdaptation, "Adaptation"}, {ts.prod, blockProduction, "Production"}}
	for _, k := range kinds {
		if k.t.ok {
			add(k.kind, k.name, k.t.bstart, k.t.bend)
		}
	}
	if ts.notes < len(text) {
		add(blockNotes, "Notes", ts.notes, len(text))
	}
	return out
}

// blockAt is the block whose lines hold line (nil: none).
func (f *folds) blockAt(line int) *blockFold {
	bs := f.blocksOf()
	for i := range bs {
		if line >= bs[i].first && line <= bs[i].last {
			return &bs[i]
		}
	}
	return nil
}

// blockRanges are the blocks folded, as the editor's folds.
func (f *folds) blockRanges() []editor.Fold {
	var out []editor.Fold
	for _, b := range f.blocksOf() {
		if !f.opened[b.key] {
			out = append(out, editor.Fold{From: b.first + 1, To: b.last + 1, Block: true})
		}
	}
	return out
}

// openBlock opens a block by hand (it stays open), or with open false
// folds it again: the cursor, if it is in it, goes to the line above.
func (f *folds) openBlock(b *blockFold, open bool) {
	if open {
		if f.opened == nil {
			f.opened = map[string]bool{}
		}
		f.opened[b.key] = true
		f.w.setStatus(blockName(b.kind) + " shown: Fold or Unfold Here (vim zc) folds it again")
	} else {
		delete(f.opened, b.key)
		f.cursorOutOf(b)
	}
	f.ed.Refresh()
}

// cursorOutOf puts the cursor at the end of the line above the block, if
// it is in it: a block folds only with the cursor out of it.
func (f *folds) cursorOutOf(b *blockFold) {
	if line, _ := f.ed.Buffer().Position(f.ed.CursorOffset()); line < b.first || line > b.last {
		return
	}
	above := max(b.first-1, 0)
	f.ed.Navigate(func(buf *buffer.Buffer) { buf.SetCursor(buf.LineEnd(above), false) })
}

// allBlocks opens every block by hand, or folds them all again.
func (f *folds) allBlocks(open bool) {
	f.opened = nil
	if !open {
		for _, b := range f.blocksOf() {
			f.cursorOutOf(&b)
		}
		return
	}
	for _, b := range f.blocksOf() {
		if f.opened == nil {
			f.opened = map[string]bool{}
		}
		f.opened[b.key] = true
	}
}

// label is a folded block's line, ▸ in the gutter before it: what it is
// and what it holds.
func (f *folds) label(line int) string {
	b := f.blockAt(line)
	if b == nil {
		return "…"
	}
	text := f.ed.Text()
	inside := text[b.in:b.inEnd]
	if l, ok := f.labels[b.key]; ok && l.inside == inside {
		return l.label
	}
	label := f.w.blockSummary(b, text, inside)
	if f.labels == nil {
		f.labels = map[string]blockLabel{}
	}
	f.labels[b.key] = blockLabel{inside: inside, label: label}
	return label
}

// blockName is how a block is called on its line.
func blockName(k blockKind) string {
	return [...]string{"Notes", "Source", "Adaptation", "Production", "Characters", "Omitted scene"}[k]
}

// blockSummary is what a block's folded line says after its name.
func (w *MainWindow) blockSummary(b *blockFold, text, inside string) string {
	var parts []string
	switch b.kind {
	case blockNotes:
		parts = notesSummary(inside)
	case blockSource:
		name := w.sourceName(w.sourceRef())
		n := 0
		if s := w.source; s != nil && len(s.passages) > 0 {
			for _, p := range s.passages {
				if p.linkable() {
					n++
				}
			}
		}
		if n > 0 {
			parts = append(parts, name+", "+nCount(n, "passage"))
		} else {
			parts = append(parts, name+", "+nCount(len(strings.Fields(inside)), "word"))
		}
	case blockAdaptation:
		l := parseLearned(inside)
		names := 0
		for _, a := range l.Aliases {
			names += len(a.Names)
		}
		for _, p := range []struct {
			n    int
			what string
		}{{names, "other name"}, {len(l.Not), "not a character"}, {len(l.Places), "place"},
			{len(parseCuts(inside)), "cut"}, {len(l.Skipped), "link skipped"}} {
			if p.n > 0 {
				parts = append(parts, nCount(p.n, p.what))
			}
		}
		if len(parts) == 0 {
			parts = append(parts, "what Accolade learned")
		}
	case blockProduction:
		if sceneLockOf(text).locked {
			parts = append(parts, "scene numbers locked")
		}
		if pageLockOf(text) != nil {
			parts = append(parts, "pages locked")
		}
		if r := currentRevisionOf(text); r != "" {
			set, _, _ := strings.Cut(r, ",")
			parts = append(parts, strings.TrimSpace(set))
		}
		if len(parts) == 0 {
			parts = append(parts, nCount(len(nonEmptyLines(inside)), "line"))
		}
	case blockCharacters:
		l := parseCharList(inside)
		with := 0
		for _, e := range l.entries {
			if !e.empty() {
				with++
			}
		}
		parts = append(parts, fmt.Sprintf("%s, %d with notes", groupDigits(len(l.entries)), with))
	case blockOmitted:
		lines := nonEmptyLines(inside)
		head := ""
		if len(lines) > 0 {
			head = strings.TrimSpace(lines[0]) + ", "
		}
		parts = append(parts, head+nCount(len(lines), "line"))
	}
	return blockName(b.kind) + " · " + strings.Join(parts, " · ")
}

// notesSummary is the notes' paragraphs and their first words.
func notesSummary(inside string) []string {
	paras := 0
	for _, p := range strings.Split(strings.ReplaceAll(inside, "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(p) != "" {
			paras++
		}
	}
	if paras == 0 {
		return []string{"empty"}
	}
	first := strings.Join(strings.Fields(inside), " ")
	if r := []rune(first); len(r) > 60 {
		first = string(r[:59]) + "…"
	}
	return []string{nCount(paras, "paragraph"), first}
}

// nonEmptyLines are the lines of s with something on them.
func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// nCount is "1 passage", "2 592 passages".
func nCount(n int, what string) string {
	if n != 1 {
		switch {
		case strings.HasSuffix(what, " skipped"):
			what = strings.TrimSuffix(what, " skipped") + "s skipped"
		case strings.HasPrefix(what, "not a "):
			what = "not " + strings.TrimPrefix(what, "not a ") + "s"
		default:
			what += "s"
		}
	}
	return groupDigits(n) + " " + what
}

// groupDigits writes n with a space between thousands: 2 592.
func groupDigits(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + " " + s[i:]
	}
	return s
}

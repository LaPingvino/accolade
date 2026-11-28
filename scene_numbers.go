// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
)

// Scene numbers in the headings' words (Joop 2026-11-23: "We have some
// number based new formats, I think that should be toggleable and easily
// auto-numbered").
//
// A language's heading pattern may have the scene number in it ({n}:
// zh's 场景{n} {place} {time} {io}, he's {n}. {io} {any}, hi-Deva's {n}.
// {place}/{time}/{io}). That number is the scene's number, the same one
// Fountain writes as #12#: one number, read from either place
// (sceneHeads: #12# first where a heading has both, written equal again
// the next time it is numbered), written where the heading has it
// (Lexington's Dialect.WithHeadingNumber). So the lock, a locked script's
// letters for new scenes (12A), Renumber, Unlock's Remove, the breakdown
// and the schedule following the numbers all work on it as on #12#.
//
// Format › Number Scenes in Headings is the script's choice, kept in its
// production block ("numbers in headings: on" or "off") so that it
// travels with the file and Undo takes it back with the text. Without the
// line it is on for a language that writes its headings by a pattern
// with {n} (not English, not Hindi written as English), unless the
// script has scene headings and none has a number in its words (a
// script written without them is not numbered by itself). On, every
// heading has its number in its words (where its words cannot have one,
// as #12#), and the numbers are kept in step with the scenes' order: a
// scene put in, moved (the structure moves, Swap…/Move…, the outline's
// and the cards' drag) or deleted, and the scenes after it are numbered
// again, as part of the same undo step (a move, New Scene: at once;
// typing and deleting: once it pauses, the heading being typed left
// alone), the status saying "Scenes renumbered (3 changed)". Locked
// numbers stay as they are: a new scene gets its letter (12A, as for
// #12#). Turning it off takes the numbers out of the words (locked ones
// move to #12#); turning it on puts them in, each one undo step. Off,
// Renumber Scenes… numbers on demand, as before.
//
// Shared, a peer keeps the numbers in step after its own moves and New
// Scene only: the others see its edits, and two peers numbering the same
// headings at once would write them twice.

// numbersInHeadingsKey is the production block's line for the choice.
const numbersInHeadingsKey = "numbers in headings"

// numbersLineOf is the script's "numbers in headings:" value: "on",
// "off", or "" (no line: the language's default).
func numbersLineOf(text string) string {
	inside, ok := productionBlock(text)
	if !ok {
		return ""
	}
	return numbersLineIn(inside)
}

// numbersLineIn is numbersLineOf for the block's inside.
func numbersLineIn(inside string) string {
	for _, line := range strings.Split(inside, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), numbersInHeadingsKey) {
			continue
		}
		switch v := strings.ToLower(strings.TrimSpace(value)); {
		case strings.HasPrefix(v, "off"), strings.HasPrefix(v, "no"):
			return "off"
		case v != "":
			return "on"
		}
	}
	return ""
}

// languageNumbersHeadings tells whether a language writes its headings
// by a pattern with the scene number in it: its numbers in the headings
// by default.
func languageNumbersHeadings(l rules.Language) bool {
	if l.HeadingForm() == rules.SceneForm {
		return false
	}
	for _, p := range append([]string{l.HeadingForm()}, l.Headings...) {
		if slices.Contains(rules.PatternSlots(p), rules.SlotNumber) {
			return true
		}
	}
	return false
}

// wordsDialect is the dialect a heading's number in its words is read
// and written by (the language whose scene starts these are), nil for
// one with no heading patterns.
func wordsDialect(starts []string) *fountain.Dialect {
	d := dialectForStarts(starts)
	if d == nil || !d.HasPatterns() {
		return nil
	}
	return d
}

// numbersInHeadings tells whether the script has its scene numbers in
// its headings, kept in step: its line, else the language's default
// (not for a script whose headings have none in their words).
func (w *MainWindow) numbersInHeadings(text string) bool {
	if w.markdown {
		return false
	}
	switch numbersLineOf(text) {
	case "on":
		return true
	case "off":
		return false
	}
	return w.numbersByDefault(text)
}

// numbersByDefault is the choice without the line.
func (w *MainWindow) numbersByDefault(text string) bool {
	if !languageNumbersHeadings(w.lex().Language(w.language)) {
		return false
	}
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	if len(heads) == 0 {
		return true
	}
	for _, h := range heads {
		if h.inWords {
			return true
		}
	}
	return false
}

// mayNumberInHeadings is the cheap part of numbersInHeadings, asked on
// every key: whether it can be on (the full answer once typing pauses).
func (w *MainWindow) mayNumberInHeadings(inside string, block bool) bool {
	if block {
		switch numbersLineIn(inside) {
		case "on":
			return true
		case "off":
			return false
		}
	}
	return languageNumbersHeadings(w.lex().Language(w.language))
}

// numberWriter writes a number into a heading: in its words where it
// has one there or the script has them there (inWords) and its words can
// take one, else as Fountain's #12#. move: the number moves to where it
// goes, out of the other place (the toggle); else a heading with both
// keeps both, equal.
type numberWriter struct {
	d       *fountain.Dialect
	inWords bool
	move    bool
}

// numberWriterFor is the writer for the window's script as it is.
func (w *MainWindow) numberWriterFor(text string) numberWriter {
	return numberWriter{d: wordsDialect(w.sceneStarts()), inWords: w.numbersInHeadings(text)}
}

// write is heading h with number n ("" none).
func (nw numberWriter) write(h sceneHead, n string) string {
	if nw.d != nil && (nw.inWords || h.inWords && !nw.move) {
		text := h.text
		if nw.move && n != "" {
			text = lex.WithSceneNumber(text, "")
		}
		if out, ok := nw.d.WithHeadingNumber(text, n); ok {
			return out
		}
	}
	text := h.text
	if nw.move && h.inWords && nw.d != nil { // out of the words, to #n#
		if out, ok := nw.d.WithHeadingNumber(text, ""); ok {
			text = out
		}
	}
	return lex.WithSceneNumber(text, n)
}

// stepNumbers are the scenes' numbers kept in step: 1…n in the script's
// order; OMITTED scenes and labels out of the scheme (PROLOGUE, I-1-A)
// as they are, not counted.
func stepNumbers(heads []sceneHead) []string {
	out := make([]string, len(heads))
	var at []int
	var nums []string
	for i, h := range heads {
		if isOmitted(h) {
			out[i] = h.num
			continue
		}
		n := ""
		if h.num != "" && !lex.InNumberScheme(h.num) {
			n = h.num
		}
		at, nums = append(at, i), append(nums, n)
	}
	for k, n := range lex.NumberLabels(nums, lex.NumberOptions{}) {
		out[at[k]] = n
	}
	return out
}

// stepEdits are the edits that keep the numbers in step: each heading
// whose number is not its place's, or that has none, written anew, but
// the one on line skip (being typed; -1 none).
func stepEdits(b *buffer.Buffer, lines []string, heads []sceneHead, nw numberWriter, skip int) []lineEdit {
	nums := stepNumbers(heads)
	var out []lineEdit
	for i, h := range heads {
		if h.line == skip || nums[i] == h.num && !h.stray {
			continue
		}
		if t := nw.write(h, nums[i]); t != lines[h.line] {
			out = append(out, lineEdit{b.LineStart(h.line), b.LineEnd(h.line), t})
		}
	}
	return out
}

// renumberedNote is what the status says of scenes numbered in step.
func renumberedNote(n int) string {
	return fmt.Sprintf("Scenes renumbered (%d changed)", n)
}

// keepInStep keeps the script's numbers in step after an edit of its
// own (a move, New Scene, typing once it pauses): unlocked, 1…n;
// locked, a scene without a number gets its letter (numberNewScenes),
// as part of the last undo step, the heading on line skip left alone
// (-1 none). It does nothing where the numbers are not in the headings.
// The extras follow; it returns how many headings changed.
func (w *MainWindow) keepInStep(skip int) int {
	if w.markdown || w.textEditor == nil {
		return 0
	}
	if w.keepingInStep { // its own edit, coming back
		return 0
	}
	w.keepingInStep = true
	defer func() { w.keepingInStep = false }()
	e := w.textEditor
	text := e.Text()
	if sceneLockOf(text).locked || !w.numbersInHeadings(text) {
		return 0
	}
	lines := strings.Split(text, "\n")
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	nw := w.numberWriterFor(text)
	var edits []lineEdit
	e.Edit(func(b *buffer.Buffer) {
		edits = stepEdits(b, lines, heads, nw, skip)
		applyBackwards(edits, b.Amend)
	})

	if len(edits) > 0 {
		w.followSceneNumbers()
	}
	return len(edits)
}

// holdingStep runs a command's edit with the numbers not kept in step
// meanwhile (the pause's pass), so its keepInStepSaying does it, saying
// so.
func (w *MainWindow) holdingStep(edit func()) {
	was := w.keepingInStep
	w.keepingInStep = true
	defer func() { w.keepingInStep = was }()
	edit()
}

// keepInStepSaying is keepInStep after a command, its status said with
// the renumbering after it: the status as said.
func (w *MainWindow) keepInStepSaying(said string) string {
	if n := w.keepInStep(-1); n > 0 {
		if said != "" {
			said += " · "
		}
		said += renumberedNote(n)
		w.setStatus(said)
	}
	return said
}

// toggleNumbersInHeadings is Format › Number Scenes in Headings: the
// script's choice turned the other way, the headings following, one
// undo step.
func (w *MainWindow) toggleNumbersInHeadings() {
	if w.markdown || w.textEditor == nil || w.scriptAway("Number Scenes in Headings") {
		return
	}
	e := w.textEditor
	text := e.Text()
	on := !w.numbersInHeadings(text)
	heads := sceneHeads(text[:bodyEndByte(text)], w.sceneStarts())
	lock := sceneLockOf(text)
	nw := numberWriter{d: wordsDialect(w.sceneStarts()), inWords: on, move: true}
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads))}
	switch {
	case on && lock.locked: // the locked numbers into the words, a new scene its letter
		p.nums = sceneNumbersFor(heads, lex.NumberOptions{Locked: true, Style: lock.style})
	case on:
		p.nums = stepNumbers(heads)
	case lock.locked: // the locked numbers stay, as #12#
		for i, h := range heads {
			p.nums[i] = h.num
		}
	default: // the numbers out, a label out of the scheme (PROLOGUE) kept
		nw.move = false
		for i, h := range heads {
			if h.num != "" && (!lex.InNumberScheme(h.num) || isOmitted(h)) {
				p.nums[i] = h.num
			}
		}
	}
	inside, _ := productionBlock(text)
	value := map[bool]string{true: "on", false: "off"}[on]
	inside = withBlockLine(inside, numbersInHeadingsKey, value)
	n := w.applyNumberPlanWith(heads, p, inside, nw)
	msg := "Scene numbers in the headings: " + value
	switch {
	case on && n > 0:
		msg += ", kept in step with the scenes' order · " + renumberedNote(n)
	case on:
		msg += ", kept in step with the scenes' order"
	case lock.locked:
		msg += "; the locked numbers stay, as #12#: " + plural(n, "heading") + " changed"
	default:
		msg += "; " + plural(n, "number") + " taken out, Renumber Scenes… numbers them on demand"
	}
	w.setStatus(msg + " (Undo takes it back)")
	w.refreshMenu()
}

// applyNumberPlanWith is applyNumberPlan with the numbers written by nw:
// how many headings changed.
func (w *MainWindow) applyNumberPlanWith(heads []sceneHead, p numberPlan, inside string, nw numberWriter) int {
	e := w.textEditor
	text := e.Text()
	lines := strings.Split(text, "\n")
	_, _, pb, _ := allTails(text)
	w.followNumbers(true) // the extras in line with the text before
	start, end, block := withProductionBlock(text, inside)
	if strings.TrimSpace(inside) == "" { // the block goes
		block = ""
		if end == utf8.RuneCountInString(text) {
			block = "\n"
		}
	}
	changed := 0
	e.Edit(func(b *buffer.Buffer) {
		line, col := b.Position(b.Cursor())
		edits := planEdits(b, lines, heads, p, nw)
		changed = len(edits)
		b.Group(func() {
			if pb.ok || strings.TrimSpace(inside) != "" {
				b.Replace(start, end, block) // after the script: the headings' places hold
			}
			applyBackwards(edits, b.Replace)
			b.SetCursor(b.Offset(line, col), false)
		})
	})
	w.followSceneNumbers()
	return changed
}

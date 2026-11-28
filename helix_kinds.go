// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/structure"
)

// The screenplay's kinds in helix's s and S (Joop 2026-11-28).
//
// Design note, three passes (helix's keymap, docs.helix-editor.com/keymap:
// s selects regex matches inside the selections, S splits on them, K
// keeps the selections matching a regex and Alt-K removes them; Alt-o
// Alt-i grow and shrink to the parent or child syntax node, Alt-n Alt-p
// the next and previous sibling, Alt-a all siblings, Alt-I all
// children; mi/ma with a textobject; ]f-style motions).
//
// The structure engine is the screenplay's syntax tree: section > scene
// > block (a speech: cue, parentheticals, dialogue; an action paragraph;
// a transition; a note or beat), and Alt+O / Alt+I (Expand and Shrink
// Selection, structure.expand) already grow and shrink along it in
// every mode, so tree-sitter's grow and shrink are there.
//
//	task                           A: s @kind        B: K @kind (s regex)    C: tree (Alt-n/p, Alt-I)
//	rename a character everywhere  % s @cue s BOB c  % s BOB K @cue c        M s BOB c (one character)
//	a scene's dialogue             mas s @dialogue   mas Alt-s K @dialogue   mas Alt-I … no kind to stop at
//	every note                     % s @note         % s \[\[.*?\]\]         none
//	a section's parentheticals, d  maS s @paren d    maS Alt-s K @paren d    none
//	through the beats              ]B [B             ]B [B                   Alt-n among siblings
//	discovering it                 the s prompt      K's prompt lists them,  nothing to list: the
//	                               lists the @kinds  but Alt-s first is      tree is invisible
//	                               as the @ is typed one more thing to learn
//
// Pass 1 (A, kinds in s): every task one verb after the selection, read
// as said ("in this scene, select the dialogue"); S @kind splits at the
// elements, as S does at matches. Pass 2 (B, kinds in K): keeps s pure
// regex, but K filters selections that exist already, so every task
// needs Alt-s first, which cuts multi-line elements (a run of dialogue, a
// note over two lines) into lines: the element is lost. Pass 3 (C, the
// tree's siblings): good for walking, but selecting a kind across a
// scene or the script needs a filter anyway, and Alt-o Alt-i are there
// already; ]d ]B ]n walk the kinds that matter.
//
// Discoverability decides it too: A's names appear where the writer
// already is (the s prompt completes them with what each is), a tip
// with a selection says s @dialogue, the palette has Select All
// Character Cues and the rest in any keys, and Fun Things to Try names
// it; B's hides behind Alt-s, C's has nothing to show.
//
// Chosen: A. A kind is a match like a regex's, read by the structure
// engine and the script's language instead of by a pattern, so s and S
// keep their meaning and only grow a vocabulary; K and Alt-K stay free
// for helix's own regex filters later. Vim's grammar is untouched: s
// and S are helix's keys, and the objects (as ad ac aB an aa) are the
// same in both modes. A pattern not starting with @ is a regex as ever,
// @@ a literal @; the prompt completes the @names.
//
//	% s @cue          every cue          mas s @dialogue   a scene's dialogue
//	% s @note         every note         maS s @paren d    a section's parentheticals gone
//	S @speech         split at the speeches

// elementKind is a kind s and S take after an @.
type elementKind struct {
	name, title, desc string                                    // title: the palette's (Select All Character Cues)
	spans             func(v *vim, tr *structure.Tree) [][2]int // runes [from, to), in order
}

// elementKinds are the kinds, as the prompt completes them.
var elementKinds = []elementKind{
	{"cue", "Character Cues", "every character cue", func(v *vim, tr *structure.Tree) [][2]int {
		var out [][2]int
		for _, s := range tr.Speeches() {
			out = append(out, v.linesSpan(s.Cue, s.Cue))
		}
		return out
	}},
	{"speech", "Speeches", "every speech: cue, parentheticals and dialogue (each of a pair)", func(v *vim, tr *structure.Tree) [][2]int {
		var out [][2]int
		for _, s := range tr.Speeches() {
			out = append(out, v.linesSpan(s.Cue, s.Last))
		}
		return out
	}},
	{"dialogue", "Dialogue", "the dialogue lines (a run between parentheticals as one)", func(v *vim, tr *structure.Tree) [][2]int {
		return v.kindRuns(tr, syntax.Dialogue)
	}},
	{"paren", "Parentheticals", "the parentheticals", func(v *vim, tr *structure.Tree) [][2]int {
		return v.kindRuns(tr, syntax.Parenthetical)
	}},
	{"action", "Action Paragraphs", "the action paragraphs", func(v *vim, tr *structure.Tree) [][2]int {
		var out [][2]int
		for _, p := range tr.Paragraphs() {
			out = append(out, v.linesSpan(p[0], p[1]))
		}
		return out
	}},
	{"heading", "Scene Headings", "the scene headings", func(v *vim, tr *structure.Tree) [][2]int { return v.headSpans(tr, true) }},
	{"section", "Section Headings", "the section headings", func(v *vim, tr *structure.Tree) [][2]int { return v.headSpans(tr, false) }},
	{"transition", "Transitions", "the transitions", func(v *vim, tr *structure.Tree) [][2]int {
		return v.kindRuns(tr, syntax.Transition)
	}},
	{"note", "Notes", "the notes, [[ ]] and all", func(v *vim, _ *structure.Tree) [][2]int {
		var out [][2]int
		for _, n := range v.notes() {
			out = append(out, [2]int{n.start, n.end})
		}
		return out
	}},
	{"beat", "Beats", "the beats: synopses standing on their own, sections marked a beat", func(v *vim, _ *structure.Tree) [][2]int {
		var out [][2]int
		for _, b := range v.beatsNow() {
			out = append(out, v.linesSpan(b.from, b.to-1))
		}
		return out
	}},
}

// elementKindNames are the kinds with their @, for the prompt and tips.
func elementKindNames() []string {
	var out []string
	for _, k := range elementKinds {
		out = append(out, "@"+k.name)
	}
	return out
}

// linesSpan is lines first to last in runes, the last's line break out.
func (v *vim) linesSpan(first, last int) [2]int {
	b := v.e.Buffer()
	return [2]int{b.LineStart(first), b.LineEnd(last)}
}

// kindRuns are the runs of lines of kind k in the body (outside the
// title page, the boneyards and the blocks at the end).
func (v *vim) kindRuns(tr *structure.Tree, k syntax.Kind) [][2]int {
	var out [][2]int
	end := min(tr.S.End, len(tr.S.Kinds))
	for i := tr.Front; i < end; i++ {
		if tr.S.Kinds[i] != k || tr.InBoneyard(i) {
			continue
		}
		j := i
		for j+1 < end && tr.S.Kinds[j+1] == k && !tr.InBoneyard(j+1) {
			j++
		}
		out = append(out, v.linesSpan(i, j))
		i = j
	}
	return out
}

// headSpans are the scene headings' lines (scene) or the sections'.
func (v *vim) headSpans(tr *structure.Tree, scene bool) [][2]int {
	var out [][2]int
	for _, u := range tr.Outline() {
		if u.Scene == scene {
			out = append(out, v.linesSpan(u.Line, u.Line))
		}
	}
	return out
}

// kindSpans are the elements of the kind named (no @) in the spans, or
// with split the parts between them; ok false for no such kind.
func (v *vim) kindSpans(name string, spans []buffer.Span, split bool) ([]buffer.Span, bool) {
	i := slices.IndexFunc(elementKinds, func(k elementKind) bool { return k.name == strings.ToLower(name) })
	if i < 0 {
		return nil, false
	}
	els := elementKinds[i].spans(v, v.unitTree())
	n := v.e.Buffer().Len()
	var out []buffer.Span
	for _, s := range spans {
		from, to := s.Start(), s.End()
		if from == to {
			from, to = 0, n
		}
		prev := from
		for _, el := range els {
			a, b := max(el[0], from), min(el[1], to)
			if a >= b {
				continue
			}
			if split {
				if a > prev {
					out = append(out, buffer.Span{Anchor: prev, Cursor: a})
				}
				prev = b
			} else {
				out = append(out, buffer.Span{Anchor: a, Cursor: b})
			}
		}
		if split && prev < to {
			out = append(out, buffer.Span{Anchor: prev, Cursor: to})
		}
	}
	return out, true
}

// kindDesc is what the kind named (with its @) is, "" none.
func kindDesc(name string) string {
	for _, k := range elementKinds {
		if "@"+k.name == name {
			return k.desc
		}
	}
	return ""
}

// kindCommands are Select All Character Cues and the rest (Edit >
// Select, the palette, :select-cue): s @kind in any keys.
func kindCommands() []*command {
	var out []*command
	for _, k := range elementKinds {
		name := k.name
		out = append(out, &command{id: "select." + name, label: "Select All " + k.title, ex: "select-" + name,
			icon: theme.ListIcon, editing: true, when: screenplayOnly, keywords: []string{"@" + name, "select every " + name},
			tip: "Select " + k.desc + " in the selection (none: the whole script), to type at all of them",
			run: func(w *MainWindow) { w.selectKind(name) }})
	}
	return out
}

// kindHomes are the Select All commands' places: Edit > Clipboard's More.
func kindHomes() []home {
	var out []home
	for _, k := range elementKinds {
		out = append(out, at("select."+k.name))
	}
	return out
}

// selectKind selects the elements of a kind in the main editor's
// selections (none: the whole script).
func (w *MainWindow) selectKind(name string) {
	v := w.vim
	if v == nil || w.textEditor == nil {
		return
	}
	spans, _ := v.kindSpans(name, w.textEditor.Buffer().Spans(), false)
	if len(spans) == 0 {
		w.setStatus("None here: @" + name)
		return
	}
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetSpans(spans) })
	w.setStatus(fmt.Sprintf("%d selected (@%s): typing goes to all of them", len(spans), name))
}

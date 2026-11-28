// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LaPingvino/lexington/lex"
)

// Renumber's strategies (docs/PRODUCTION.md §15): which scenes get a new
// number, which numbers stay, what becomes of A-scenes and of OMITTED
// ones. The numbering itself is Lexington's (lex.Renumber); this file
// turns the dialog's choices into its options, and its result into a
// numberPlan that the preview shows and Apply writes: the same plan, so
// what the preview says is what the script gets.

// renumberScope is which scenes Renumber may give a new number.
type renumberScope int

const (
	scopeAll      renumberScope = iota
	scopeSelected               // the scenes the selection touches (the cursor's without one)
	scopeRange                  // from scene From to scene To
	scopeCursor                 // from the cursor's scene on
)

// renumberList is what the typed list of numbers means.
type renumberList int

const (
	listNone renumberList = iota
	listKeep              // these numbers stay; the rest are numbered around them
	listOnly              // only these are renumbered
)

// omittedChoice is what Renumber does with the OMITTED scenes in its
// scope: drop them (the gaps close), keep them, or keep them and print a
// run as one ("12-14 OMITTED": the production block's setting,
// omitted_runs.go). The text keeps one heading a scene either way.
type omittedChoice int

const (
	omitDrop omittedChoice = iota
	omitKeep
	omitPrintRuns
)

// mode is the choice as Lexington's Renumber takes it.
func (o omittedChoice) mode() lex.OmittedMode {
	if o == omitDrop {
		return lex.OmittedDrop
	}
	return lex.OmittedKeep
}

// renumberChoice is the dialog's choices, as "Last used" keeps them.
type renumberChoice struct {
	Scope       renumberScope `json:"scope"`
	From        int           `json:"from"` // scopeRange: scene indices, in order
	To          int           `json:"to"`
	List        renumberList  `json:"list"`
	Numbers     string        `json:"numbers"` // "5, 12A, 20-24"
	KeepLetters bool          `json:"keepLetters"`
	Omitted     omittedChoice `json:"omitted"`
}

// renumberLastPref is the setting that keeps the last used choice.
const renumberLastPref = "renumber-last"

// cleanChoice is the preset "Clean 1…n": every scene, A-numbers folded,
// OMITTED dropped and the gaps closed (Renumber before strategies).
func cleanChoice() renumberChoice { return renumberChoice{} }

// sentChoice is the preset "After an issue: keep sent scenes": every
// number that went out in an issue stays (and is never given to another
// scene), OMITTED ones too; only what came since is numbered, around
// them, its letters folded where a free number fits (a 1B the lock gave
// between 1A and a cut 3 becomes 2).
// The OMITTED ones are kept, printed as the script prints them now.
func sentChoice(sent string, printRuns bool) renumberChoice {
	c := renumberChoice{List: listKeep, Numbers: sent, Omitted: omitKeep}
	if printRuns {
		c.Omitted = omitPrintRuns
	}
	return c
}

// lastChoice is the last used choice, if one is kept.
func (w *MainWindow) lastChoice() (renumberChoice, bool) {
	var c renumberChoice
	if w.settings == nil {
		return c, false
	}
	s := w.settings.GetString(renumberLastPref)
	if s == "" || json.Unmarshal([]byte(s), &c) != nil {
		return renumberChoice{}, false
	}
	c.Omitted = min(max(c.Omitted, omitDrop), omitPrintRuns)
	c.Scope = min(max(c.Scope, scopeAll), scopeCursor)
	c.List = min(max(c.List, listNone), listOnly)
	return c, true
}

// rememberChoice keeps c as the last used.
func (w *MainWindow) rememberChoice(c renumberChoice) {
	if w.settings == nil {
		return
	}
	if b, err := json.Marshal(c); err == nil {
		w.settings.SetString(renumberLastPref, string(b))
	}
}

// renumberContext is what a choice is read against: the cursor's and the
// selection's scenes (indices into the headings), the numbering style,
// and the hand-typed numbers of a lock.
type renumberContext struct {
	cursor         int
	selFrom, selTo int
	style          lex.NumberStyle
	locked         bool
	hand           map[string]bool // numbered before the lock
	printRuns      bool            // the script prints OMITTED runs as one now
}

// sceneIndexOf is the index of the heading a line is in (the last at or
// before it; 0 before the first).
func sceneIndexOf(heads []sceneHead, line int) int {
	at := 0
	for i, h := range heads {
		if h.line <= line {
			at = i
		}
	}
	return at
}

// renumberContextNow is the window's: its cursor and selection now.
func (w *MainWindow) renumberContextNow(text string, heads []sceneHead) renumberContext {
	e := w.textEditor
	b := e.Buffer()
	cursorLine, _ := b.Position(e.CursorOffset())
	ctx := renumberContext{cursor: sceneIndexOf(heads, cursorLine)}
	ctx.selFrom, ctx.selTo = ctx.cursor, ctx.cursor
	if b.HasSelection() {
		s, t := b.Selection()
		l1, _ := b.Position(s)
		l2, c2 := b.Position(t)
		if c2 == 0 && l2 > l1 { // ending at a line's start: not that line's scene
			l2--
		}
		ctx.selFrom, ctx.selTo = sceneIndexOf(heads, l1), sceneIndexOf(heads, l2)
	}
	l := sceneLockOf(text)
	ctx.locked = l.locked
	ctx.printRuns = omittedRunsPrinted(text)
	ctx.style = w.sceneNumberStyle()
	if l.locked {
		ctx.style = l.style
		ctx.hand, _ = handNumbersOf(text)
	}
	return ctx
}

// sceneLabelsOf are the headings' numbers, in order ("" none).
func sceneLabelsOf(heads []sceneHead) []string {
	out := make([]string, len(heads))
	for i, h := range heads {
		out[i] = h.num
	}
	return out
}

// numbersOf reads the choice's list against the script's labels, in its
// order (lex.ParseNumberSet: a range runs from scene to scene as the
// script has them).
func numbersOf(c renumberChoice, heads []sceneHead) lex.NumberSet {
	return lex.ParseNumberSet(c.Numbers, sceneLabelsOf(heads))
}

// problems are what keeps the choice from being applied: the list's
// errors (a range whose end comes first in the script, an end no scene
// has), and a From … To whose To comes before its From. Neither is
// swapped: swapping scenes is not renumbering.
func (c renumberChoice) problems(heads []sceneHead) []string {
	var out []string
	if c.Scope == scopeRange && c.To < c.From && c.From < len(heads) {
		out = append(out, fmt.Sprintf("From %s to %s: scene %s comes before scene %s in the script",
			pickerName(c.From, heads[c.From]), pickerName(c.To, heads[c.To]), pickerName(c.To, heads[c.To]), pickerName(c.From, heads[c.From])))
	}
	if c.List != listNone {
		out = append(out, numbersOf(c, heads).Errors...)
	}
	return out
}

// pickerName is a heading as a message names it: its number, else its
// place ("scene 3").
func pickerName(i int, h sceneHead) string {
	if h.num != "" {
		return h.num
	}
	return fmt.Sprintf("(%d)", i+1)
}

// inScope tells whether heading i is in the choice's scope (a From … To
// the wrong way round has none: problems says why).
func (c renumberChoice) inScope(i int, ctx renumberContext) bool {
	switch c.Scope {
	case scopeSelected:
		return i >= ctx.selFrom && i <= ctx.selTo
	case scopeRange:
		return i >= c.From && i <= c.To
	case scopeCursor:
		return i >= ctx.cursor
	}
	return true
}

// strategyPlan is the plan a choice makes of the headings.
func strategyPlan(heads []sceneHead, c renumberChoice, ctx renumberContext) numberPlan {
	set := numbersOf(c, heads)
	scenes := make([]lex.RenumberScene, len(heads))
	free := make([]bool, len(heads))
	for i, h := range heads {
		scenes[i] = lex.RenumberScene{Number: h.num, Omitted: isOmitted(h)}
		free[i] = c.inScope(i, ctx)
		switch {
		case c.List == listKeep && !set.Empty():
			free[i] = free[i] && !set.Has(i)
		case c.List == listOnly:
			free[i] = free[i] && set.Has(i)
		}
	}
	o := lex.RenumberOptions{Free: free, KeepLetters: c.KeepLetters, Omitted: c.Omitted.mode(), Style: ctx.style}
	if c.List == listKeep {
		o.Reserved = set.Unknown // a kept number no scene has now is not given either
	}
	res := lex.Renumber(scenes, o)
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads)), free: free}
	for i, r := range res {
		p.nums[i], p.drop[i] = r.Number, r.Drop
		// a fixed label (I-1-A) Lexington keeps: pinned whatever the scope
		p.free[i] = p.free[i] && (heads[i].num == "" || lex.InNumberScheme(heads[i].num))
	}
	return p
}

// handLineAfter is the "numbered before the lock" line once p is
// written: the hand-typed numbers whose scene keeps its number; the
// renumbered ones are the lock's now.
func handLineAfter(heads []sceneHead, p numberPlan, ctx renumberContext) string {
	var nums []string
	for i, h := range heads {
		if ctx.hand[h.num] && !p.free[i] && p.nums[i] == h.num && !isOmitted(h) {
			nums = append(nums, h.num)
		}
	}
	if len(nums) == 0 {
		return "none"
	}
	return strings.Join(nums, ", ")
}

// previewRow is one heading in the preview: old → new, and how it
// prints when OMITTED runs print as one.
type previewRow struct {
	old, new, heading string
	changed           bool
}

// previewRows are the plan's rows, one a heading, and how many change;
// runs (printRuns) are the OMITTED runs as they will print.
func previewRows(heads []sceneHead, p numberPlan, runs map[int]string) (rows []previewRow, changed int) {
	for i, h := range heads {
		heading, _ := lex.SceneNumber(strings.TrimSpace(h.text))
		heading = strings.TrimPrefix(strings.TrimSpace(heading), ".")
		r := previewRow{old: h.num, heading: heading}
		if p.drop[i] {
			r.new = "gone"
		} else {
			r.new = p.nums[i]
		}
		if r.old == "" {
			r.old = "–"
		}
		r.changed = p.drop[i] || p.nums[i] != h.num
		if !r.changed {
			r.new = "stays"
		} else if r.new == "" {
			r.new = "none"
		}
		if run, ok := runs[i]; ok {
			r.heading += " · prints as " + run
		}
		if r.changed {
			changed++
		}
		rows = append(rows, r)
	}
	return rows, changed
}

// printRuns are the runs of OMITTED headings a plan leaves that print as
// one (two or more kept one after the other, only blank lines and their
// boneyard between them): for each heading in one, the run's label
// (12-14), its numbers as the plan gives them.
func printRuns(text string, heads []sceneHead, p numberPlan) map[int]string {
	lines := strings.Split(text, "\n")
	// only blank lines and boneyard between headings a and b
	adjacent := func(a, b int) bool {
		in := false
		for _, l := range lines[heads[a].line+1 : heads[b].line] {
			t := strings.TrimSpace(l)
			switch {
			case in:
				in = !strings.Contains(t, "*/")
			case strings.HasPrefix(t, "/*"):
				in = !strings.Contains(t[2:], "*/")
			case t != "":
				return false
			}
		}
		return true
	}
	out := map[int]string{}
	var kept []int // the OMITTED headings that stay, in order
	for i, h := range heads {
		if isOmitted(h) && !p.drop[i] && p.nums[i] != "" {
			kept = append(kept, i)
		}
	}
	for k := 0; k < len(kept); {
		j := k
		for j+1 < len(kept) && kept[j+1] == kept[j]+1 && adjacent(kept[j], kept[j+1]) {
			j++
		}
		if j > k {
			label := lex.OmittedRunLabel(p.nums[kept[k]], p.nums[kept[j]])
			for _, i := range kept[k : j+1] {
				out[i] = label
			}
		}
		k = j + 1
	}
	return out
}

// sentNumbers are the numbers that went out in the issues, as the keep
// list writes them: the script's runs of scenes that all went out as
// ranges in its order ("1-40"), a scene written since breaking a run (a
// 4A after the issue splits 1-9 into "1-4, 5-9", so that 4A is not
// kept); then the numbers that went out and no scene has now, one by one
// (never given again).
func sentNumbers(chain *revisionChain, starts []string, now []sceneHead) string {
	if chain == nil {
		return ""
	}
	sent := map[string]bool{}
	var order []string
	for _, is := range chain.issues {
		for _, h := range sceneHeads(strings.Join(is.lines, "\n"), starts) {
			n := h.num
			if n == "" || sent[n] {
				continue
			}
			sent[n] = true
			order = append(order, n)
		}
	}
	labels := sceneLabelsOf(now)
	has := map[string]bool{}
	for _, l := range labels {
		has[l] = true
	}
	var out []string
	for i := 0; i < len(now); {
		if !sent[now[i].num] {
			i++
			continue
		}
		j := i
		for j+1 < len(now) && sent[now[j+1].num] {
			j++
		}
		a, b := now[i].num, now[j].num
		switch {
		case i == j:
			out = append(out, a)
		case strings.Contains(a, "-") || strings.Contains(b, "-") || has[a+"-"+b]:
			out = append(out, a+" – "+b)
		default:
			out = append(out, a+"-"+b)
		}
		i = j + 1
	}
	for _, n := range order {
		if !has[n] {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

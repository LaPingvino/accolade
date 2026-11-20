package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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

// renumberChoice is the dialog's choices, as "Last used" keeps them.
type renumberChoice struct {
	Scope       renumberScope   `json:"scope"`
	From        int             `json:"from"` // scopeRange: scene indices, in order
	To          int             `json:"to"`
	List        renumberList    `json:"list"`
	Numbers     string          `json:"numbers"` // "5, 12A, 20-24"
	KeepLetters bool            `json:"keepLetters"`
	Omitted     lex.OmittedMode `json:"omitted"`
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
func sentChoice(sent string) renumberChoice {
	return renumberChoice{List: listKeep, Numbers: sent, Omitted: lex.OmittedKeep}
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
	ctx.style = w.sceneNumberStyle()
	if l.locked {
		ctx.style = l.style
		ctx.hand, _ = handNumbersOf(text)
	}
	return ctx
}

// numbersOf reads the choice's list; bad are the items that are no
// number of the script nor a range.
func numbersOf(c renumberChoice, heads []sceneHead) (set lex.NumberSet, bad []string) {
	set, odd := lex.ParseNumberSet(c.Numbers)
	for _, n := range odd { // 3-x2k9q is fine if a scene has it
		found := false
		for _, h := range heads {
			found = found || h.num == n
		}
		if !found {
			bad = append(bad, n)
		}
	}
	return set, bad
}

// inScope tells whether heading i is in the choice's scope.
func (c renumberChoice) inScope(i int, ctx renumberContext) bool {
	switch c.Scope {
	case scopeSelected:
		return i >= ctx.selFrom && i <= ctx.selTo
	case scopeRange:
		from, to := min(c.From, c.To), max(c.From, c.To)
		return i >= from && i <= to
	case scopeCursor:
		return i >= ctx.cursor
	}
	return true
}

// strategyPlan is the plan a choice makes of the headings.
func strategyPlan(heads []sceneHead, c renumberChoice, ctx renumberContext) numberPlan {
	set, _ := numbersOf(c, heads)
	scenes := make([]lex.RenumberScene, len(heads))
	free := make([]bool, len(heads))
	for i, h := range heads {
		scenes[i] = lex.RenumberScene{Number: h.num, Omitted: isOmitted(h)}
		free[i] = c.inScope(i, ctx)
		switch {
		case c.List == listKeep && !set.Empty():
			free[i] = free[i] && !set.Has(h.num)
		case c.List == listOnly:
			free[i] = free[i] && set.Has(h.num)
		}
	}
	o := lex.RenumberOptions{Free: free, KeepLetters: c.KeepLetters, Omitted: c.Omitted, Style: ctx.style}
	if c.List == listKeep {
		o.Reserved = set.Items() // a kept number no scene has now is not given either
	}
	res := lex.Renumber(scenes, o)
	p := numberPlan{nums: make([]string, len(heads)), drop: make([]bool, len(heads)), merge: make([]bool, len(heads)), free: free}
	for i, r := range res {
		p.nums[i], p.drop[i], p.merge[i] = r.Number, r.Drop, r.Merged
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

// previewRow is one heading in the preview: old → new.
type previewRow struct {
	old, new, heading string
	changed           bool
}

// previewRows are the plan's rows, one a heading, and how many change.
func previewRows(heads []sceneHead, p numberPlan) (rows []previewRow, changed int) {
	for i, h := range heads {
		heading, _ := lex.SceneNumber(strings.TrimSpace(h.text))
		heading = strings.TrimPrefix(strings.TrimSpace(heading), ".")
		r := previewRow{old: h.num, heading: heading}
		switch {
		case p.drop[i]:
			r.new = "gone"
		case p.merge[i]:
			r.new = "into " + mergedInto(p, i)
		default:
			r.new = p.nums[i]
		}
		if r.old == "" {
			r.old = "–"
		}
		r.changed = p.drop[i] || p.merge[i] || p.nums[i] != h.num
		if !r.changed {
			r.new = "stays"
		} else if r.new == "" {
			r.new = "none"
		}
		if r.changed {
			changed++
		}
		rows = append(rows, r)
	}
	return rows, changed
}

// mergedInto is the range a merged heading went into: the run's first's.
func mergedInto(p numberPlan, i int) string {
	for j := i - 1; j >= 0; j-- {
		if !p.merge[j] {
			return p.nums[j]
		}
	}
	return ""
}

// sentNumbers are the numbers that went out in the issues, as the keep
// list writes them: runs of whole numbers as ranges (1-40), where no
// scene of the script now has a number inside the run that did not go
// out (a 4A written since splits 1-9 into "1-4, 5-9", so that 4A is not
// kept).
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
	whole := map[int]bool{}
	var other []string
	for _, n := range order {
		if v, err := strconv.Atoi(n); err == nil && fmt.Sprint(v) == n {
			whole[v] = true
		} else {
			other = append(other, n)
		}
	}
	// a number of the script now that did not go out breaks a run
	breaks := func(lo, hi int) bool {
		r := fmt.Sprintf("%d-%d", lo, hi)
		set, _ := lex.ParseNumberSet(r)
		for _, h := range now {
			if h.num != "" && !sent[h.num] && set.Has(h.num) {
				return true
			}
		}
		return false
	}
	var keys []int
	for v := range whole {
		keys = append(keys, v)
	}
	slices.Sort(keys)
	var out []string
	for i := 0; i < len(keys); {
		j := i
		for j+1 < len(keys) && keys[j+1] == keys[j]+1 && !breaks(keys[i], keys[j+1]) {
			j++
		}
		if j > i {
			out = append(out, fmt.Sprintf("%d-%d", keys[i], keys[j]))
		} else {
			out = append(out, fmt.Sprint(keys[i]))
		}
		i = j + 1
	}
	return strings.Join(append(out, other...), ", ")
}

package main

import (
	"slices"
)

// The breakdown and the schedule follow the scene numbers.
//
// A breakdown tag, a scene marked done and a strip name their scene by its
// number first (bdSceneKey), and those are kept beside the script, outside
// its undo. When the numbers change and nothing else does (Lock, Unlock's
// Remove, Renumber, and Undo or Redo of any of them: the same headings in
// the same order, OMITTED ones aside), each old number is renamed to the
// new number of the scene in its place, in the breakdown, the schedule
// and the board's Undo, so a number never points at another scene. A
// strip's words (its sum, which counts the heading's number) are taken
// over where its scene was unchanged, so it is not flagged as changed.
//
// The window keeps the numbers it last saw; they are compared once typing
// pauses (updateBreakdown), whenever the tags or strips are found
// (breakdownNowFound), and right around the commands that change them.
// A number two scenes share is no one's: it is renamed only where a tag or
// strip was found, and the window does not take a text with a number
// twice as the one it saw (a number being retyped, #12# → #1# → #13#,
// passes through one), but waits for the next. An OMITTED scene that went
// (Renumber, Remove) takes its number with it: what named it is lost, not
// moved to the scene that has the number now.

// numbersSeen is the scene numbers the extras were last brought in line
// with: the script's (key), its text, its headings' words and numbers
// (OMITTED ones left out).
type numbersSeen struct {
	key, text string
	heads     []string
	nums      []string
	omitted   []string // the OMITTED scenes' numbers
}

// sceneSignature are a text's headings (their words, without number) and
// numbers, in order, OMITTED ones left out (their numbers apart).
func sceneSignature(text string, starts []string) (heads, nums, omitted []string) {
	for _, h := range sceneHeads(text[:bodyEndByte(text)], starts) {
		if isOmitted(h) {
			omitted = append(omitted, h.num)
			continue
		}
		heads = append(heads, bdHeadingKey(h.text))
		nums = append(nums, h.num)
	}
	return heads, nums, omitted
}

// counts is how often each number is in nums.
func counts(nums []string) map[string]int {
	c := map[string]int{}
	for _, n := range nums {
		if n != "" {
			c[n]++
		}
	}
	return c
}

// hasTwice tells whether a number is in nums more than once.
func hasTwice(nums ...[]string) bool {
	c := map[string]int{}
	for _, ns := range nums {
		for _, n := range ns {
			if n != "" {
				c[n]++
				if c[n] > 1 {
					return true
				}
			}
		}
	}
	return false
}

// followSceneNumbers brings the breakdown and the schedule in line with
// the script's numbers, if only those changed since it last looked.
func (w *MainWindow) followSceneNumbers() {
	w.followRenamed() // an Undo of a character's rename (characters_rename.go)
	w.followNumbers(false)
}

// noteNumbers takes the text's numbers as seen, renaming nothing: a
// change that brought the extras along itself (a structural move, its
// Undo and Redo), where numbers in another order are scenes that moved,
// not numbers that changed.
func (w *MainWindow) noteNumbers() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.noteNumbersOf(w.textEditor.Text())
}

// noteNumbersOf is noteNumbers for the text about to be (a move's, so the
// change itself renames nothing).
func (w *MainWindow) noteNumbersOf(text string) {
	heads, nums, omitted := sceneSignature(text, w.sceneStarts())
	w.numbers = numbersSeen{key: w.breakdownKey(), text: text, heads: heads, nums: nums, omitted: omitted}
}

// followNumbers is followSceneNumbers; anyway: take the text as seen even
// with a number twice (before Renumber or Remove, which end that), though
// nothing is renamed into it.
func (w *MainWindow) followNumbers(anyway bool) {
	if w.markdown || w.textEditor == nil {
		return
	}
	text, key := w.textEditor.Text(), w.breakdownKey()
	old := w.numbers
	if old.key == key && sameString(old.text, text) {
		return
	}
	starts := w.sceneStarts()
	heads, nums, omitted := sceneSignature(text, starts)
	twice := hasTwice(nums, omitted)
	if twice && !anyway && old.key == key {
		return // a number being retyped, say: wait for the next
	}
	w.numbers = numbersSeen{key: key, text: text, heads: heads, nums: nums, omitted: omitted}
	if twice || old.key != key || !slices.Equal(old.heads, heads) || (slices.Equal(old.nums, nums) && slices.Equal(old.omitted, omitted)) {
		return
	}
	// old number → new; a scene that had none, by its heading and which
	// of those it is (Undo of Remove, a Lock)
	to := map[string]string{}
	type place struct {
		heading string
		nth     int
	}
	given := map[place]string{}
	nth := map[string]int{}
	once := counts(append(append([]string(nil), old.nums...), old.omitted...))
	for _, n := range old.omitted {
		if once[n] == 1 && !slices.Contains(omitted, n) {
			to[n] = "" // the OMITTED scene went, its number with it
		}
	}
	for i, n := range old.nums {
		switch {
		case n != "" && n != nums[i]:
			if once[n] == 1 {
				to[n] = nums[i]
			}
		case n == "" && nums[i] != "":
			given[place{heads[i], nth[heads[i]]}] = nums[i]
		}
		nth[heads[i]]++
	}
	w.renameScenes(old.text, text, starts, func(k *bdSceneKey) bool {
		if k.Scene == "" {
			if n, ok := given[place{k.Heading, k.Nth}]; ok {
				k.Scene = n
				return true
			}
			return false
		}
		if n, ok := to[k.Scene]; ok {
			k.Scene = n
			return true
		}
		return false
	})
}

// renameScenes renames the scenes' numbers in the breakdown and the
// schedule: a tag or strip found in the text before (by its anchors, as
// it is shown) gets the number its scene has after, if that changed;
// one not found, or a scene marked done, by rename (old number → new,
// else heading and which of those it is). The texts before and after, for
// finding them and for the strips' words.
func (w *MainWindow) renameScenes(before, after string, starts []string, rename func(*bdSceneKey) bool) {
	bf := w.breakdownNow()
	hasTags := w.breakdown.err == nil && (len(bf.Tags) > 0 || len(bf.Done) > 0)
	s := w.scheduleNow()
	hasStrips := w.schedule.err == nil && (len(s.Days) > 0 || w.schedule.undo != nil)
	if !hasTags && !hasStrips {
		return
	}
	oldBt, newBt := readBdText(before, starts), readBdText(after, starts)
	// each old scene's place in the new text, OMITTED ones aside
	pos := make([]int, len(oldBt.scenes))
	var kept []int
	for j, sc := range newBt.scenes {
		if !sc.omitted {
			kept = append(kept, j)
		}
	}
	k := 0
	for i, sc := range oldBt.scenes {
		pos[i] = -1
		if !sc.omitted && k < len(kept) {
			pos[i] = kept[k]
			k++
		}
	}
	// renumbered sets a key found in old scene sc to the number it has now
	renumbered := func(key *bdSceneKey, sc int) bool {
		if sc < 0 || pos[sc] < 0 { // not found: only a number is renamed (a heading's place may be stale)
			return key.Scene != "" && rename(key)
		}
		n := newBt.scenes[pos[sc]].key.Scene
		if n == oldBt.scenes[sc].key.Scene || n == key.Scene {
			return false
		}
		key.Scene = n
		return true
	}
	if hasTags {
		changed := false
		for i, p := range oldBt.place(bf) {
			sc := p.scene
			if p.lost {
				sc = -1
			}
			changed = renumbered(&bf.Tags[i].bdSceneKey, sc) || changed
		}
		for i := range bf.Done {
			changed = rename(&bf.Done[i]) || changed
		}
		if changed {
			w.saveBreakdown()
		}
	}
	if !hasStrips {
		return
	}
	follow := func(f *scheduleFile) bool {
		p := f.place(oldBt)
		changed := false
		for d := range f.Days {
			for k := range f.Days[d].Strips {
				st := &f.Days[d].Strips[k]
				sc := p.days[d][k]
				changed = renumbered(&st.bdSceneKey, sc) || changed
				if sc < 0 {
					continue
				}
				if j := pos[sc]; j >= 0 && st.Sum != "" && st.Sum == oldBt.sceneSum(sc) {
					if sum := newBt.sceneSum(j); sum != st.Sum {
						st.Sum, changed = sum, true
					}
				}
			}
		}
		return changed
	}
	if w.schedule.undo != nil {
		follow(w.schedule.undo)
	}
	if follow(s) {
		w.writeSchedule()
	}
}

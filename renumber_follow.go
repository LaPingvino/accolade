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

// numbersSeen is the scene numbers the extras were last brought in line
// with: the script's (key), its text, its headings' words and numbers
// (OMITTED ones left out).
type numbersSeen struct {
	key, text string
	heads     []string
	nums      []string
}

// sceneSignature are a text's headings (their words, without number) and
// numbers, in order, OMITTED ones left out.
func sceneSignature(text string, starts []string) (heads, nums []string) {
	for _, h := range sceneHeads(text[:bodyEndByte(text)], starts) {
		if isOmitted(h) {
			continue
		}
		heads = append(heads, bdHeadingKey(h.text))
		nums = append(nums, h.num)
	}
	return heads, nums
}

// followSceneNumbers brings the breakdown and the schedule in line with
// the script's numbers, if only those changed since it last looked.
func (w *MainWindow) followSceneNumbers() {
	if w.markdown || w.textEditor == nil {
		return
	}
	text, key := w.textEditor.Text(), w.breakdownKey()
	old := w.numbers
	if old.key == key && sameString(old.text, text) {
		return
	}
	starts := w.sceneStarts()
	heads, nums := sceneSignature(text, starts)
	w.numbers = numbersSeen{key: key, text: text, heads: heads, nums: nums}
	if old.key != key || !slices.Equal(old.heads, heads) || slices.Equal(old.nums, nums) {
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
	for i, n := range old.nums {
		switch {
		case n != "" && n != nums[i]:
			to[n] = nums[i]
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

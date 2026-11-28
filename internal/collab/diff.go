// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package collab

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/scriptdiff"
)

// Edit is one replacement in a text, in runes: Del runes at Pos become
// Text.
type Edit struct {
	Pos, Del int
	Text     string
}

// Edits are the edits that make old new, small and in order (each Pos in
// old's runes): the common start and end are kept, the rest compared by
// lines and the changed lines by words. A change made all over a text at
// once (renumbering, a block rewritten) so becomes many small edits, and
// a peer's typing in between merges with it.
func Edits(old, new string) []Edit {
	// the common ends by bytes, on whole runes: a key in a novel costs
	// the scan, not the novel in runes twice
	pre := 0
	for pre < len(old) && pre < len(new) && old[pre] == new[pre] {
		pre++
	}
	for pre > 0 && (pre < len(old) && !utf8.RuneStart(old[pre]) || pre < len(new) && !utf8.RuneStart(new[pre])) {
		pre-- // back to the start of the rune they differ in
	}
	suf := 0
	for suf < len(old)-pre && suf < len(new)-pre && old[len(old)-1-suf] == new[len(new)-1-suf] {
		suf++
	}
	for suf > 0 && (!utf8.RuneStart(old[len(old)-suf]) || !utf8.RuneStart(new[len(new)-suf])) {
		suf--
	}
	o, n := []rune(old[pre:len(old)-suf]), []rune(new[pre:len(new)-suf])
	start := 0
	for start < len(o) && start < len(n) && o[start] == n[start] {
		start++
	}
	end := 0
	for end < len(o)-start && end < len(n)-start && o[len(o)-1-end] == n[len(n)-1-end] {
		end++
	}
	mo, mn := o[start:len(o)-end], n[start:len(n)-end]
	start += utf8.RuneCountInString(old[:pre])
	if len(mo) == 0 && len(mn) == 0 {
		return nil
	}
	if len(mo) == 0 || len(mn) == 0 || len(mo)+len(mn) <= 2 {
		return []Edit{{Pos: start, Del: len(mo), Text: string(mn)}}
	}
	var out []Edit
	add := func(e Edit) {
		if e.Del == 0 && e.Text == "" {
			return
		}
		if k := len(out) - 1; k >= 0 && out[k].Pos+out[k].Del == e.Pos { // touching: one
			out[k].Del += e.Del
			out[k].Text += e.Text
			return
		}
		out = append(out, e)
	}
	// by lines, then each changed stretch by words
	diffTokens(lines(mo), lines(mn), start, func(pos int, removed, added []string) {
		if len(removed) == 0 || len(added) == 0 {
			add(Edit{Pos: pos, Del: runeCount(removed), Text: strings.Join(added, "")})
			return
		}
		diffTokens(words([]rune(strings.Join(removed, ""))), words([]rune(strings.Join(added, ""))), pos,
			func(pos int, removed, added []string) {
				add(Edit{Pos: pos, Del: runeCount(removed), Text: strings.Join(added, "")})
			})
	})
	return out
}

// diffTokens calls hunk for each stretch where a and b differ, in order:
// at rune offset pos (counted from at), a's tokens removed became b's.
func diffTokens(a, b []string, at int, hunk func(pos int, removed, added []string)) {
	kinds := scriptdiff.Lines(a, b)
	pos := at
	i, j := 0, 0
	var rem, add []string
	hunkAt := pos
	flush := func() {
		if len(rem) > 0 || len(add) > 0 {
			hunk(hunkAt, rem, add)
		}
		rem, add = nil, nil
	}
	for _, k := range kinds {
		switch k {
		case scriptdiff.Same:
			flush()
			pos += len([]rune(a[i]))
			i++
			j++
			hunkAt = pos
		case scriptdiff.Removed:
			if len(rem) == 0 && len(add) == 0 {
				hunkAt = pos
			}
			rem = append(rem, a[i])
			pos += len([]rune(a[i]))
			i++
		case scriptdiff.Added:
			if len(rem) == 0 && len(add) == 0 {
				hunkAt = pos
			}
			add = append(add, b[j])
			j++
		}
	}
	flush()
}

// lines are a text's lines, each with its line break.
func lines(rs []rune) []string {
	var out []string
	from := 0
	for i, r := range rs {
		if r == '\n' {
			out = append(out, string(rs[from:i+1]))
			from = i + 1
		}
	}
	if from < len(rs) {
		out = append(out, string(rs[from:]))
	}
	return out
}

// words are a text cut into words (letters and digits) and single other
// runes.
func words(rs []rune) []string {
	var out []string
	from := 0
	word := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	for i := 1; i <= len(rs); i++ {
		if i == len(rs) || !word(rs[i]) || !word(rs[i-1]) {
			out = append(out, string(rs[from:i]))
			from = i
		}
	}
	return out
}

func runeCount(ss []string) int {
	n := 0
	for _, s := range ss {
		n += len([]rune(s))
	}
	return n
}

// SetText makes the document's text text by the edits between the two
// (Edits), and returns the ops: an editor's change becomes ops this way.
func (d *Doc) SetText(text string) []Op {
	es := Edits(d.Text(), text)
	var ops []Op
	for i := len(es) - 1; i >= 0; i-- { // from the end: the positions before stay
		e := es[i]
		ops = append(ops, d.Delete(e.Pos, e.Del)...)
		ops = append(ops, d.Insert(e.Pos, e.Text)...)
	}
	if d.size == utf8.RuneCountInString(text) {
		d.text, d.textOK = text, true
	}
	return ops
}

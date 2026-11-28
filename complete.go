// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"sort"
	"strings"

	"github.com/LaPingvino/lexington/rules"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Tab completes what is being typed on a line after a blank one: a scene
// heading slot by slot, in the order of the language's heading forms
// (heading_slots.go: after the scene word the script's places, the most
// used first, after the dash the time words; a Chinese or Hindi
// heading's slots in its pattern's order), and a character's cue (the
// source's speakers, then the scene's speakers, the most recent first
// but the last, who spoke just now, then the script's by how often they
// speak; after "NAME (" the language's extensions). The status bar says
// what Tab would give and the list under the line shows it all
// (complete_list.go); Tab again goes through them.

// completion is what Tab completes on the line with the cursor.
type completion struct {
	start      int      // where the completed text starts (after the indent)
	end        int      // the end of the line (the cursor, or before a closing bracket after it)
	prefix     string   // what was typed
	candidates []string // what it can become: the line from start
	labels     []string // each as the list shows it: the slot's word, the name
}

// completeAt finds the completions for the cursor in text, if it is at
// the end of a line that begins a character name or a scene heading.
func completeAt(text string, cursor int, starts []string) *completion {
	return completeFirst(text, cursor, starts, nil, nil)
}

// languageOfStarts is the language whose scene starts these are, English
// for none known.
func languageOfStarts(starts []string) rules.Language {
	if d := dialectForStarts(starts); d != nil && starts != nil {
		return d.Lang
	}
	return lexSettings.Language("en")
}

// completeFirst is completeAt with names and headings that go first,
// in their order, when they fit what was typed (the source's speakers,
// the heading it reads; suggest.go), the script's need not have them.
func completeFirst(text string, cursor int, starts []string, firstNames, firstHeads []string) *completion {
	return completeIn(text, cursor, completeCtx{starts: starts, l: languageOfStarts(starts), names: firstNames, heads: firstHeads})
}

// completeCtx is what a completion is made with: the language, its
// scene starts, the source's names and headings to offer first.
type completeCtx struct {
	starts       []string
	l            rules.Language
	names, heads []string
}

// completeIn is the completion at the cursor.
func completeIn(text string, cursor int, cx completeCtx) *completion {
	runes := []rune(text)
	if cursor > len(runes) {
		return nil
	}
	start, end := lineBounds(runes, cursor)
	closing := ""
	if cursor != end {
		// only an automatic closing bracket after the cursor: NAME (V|)
		if after := string(runes[cursor:end]); after == ")" || after == "）" {
			closing = after
		} else {
			return nil
		}
	}
	line := string(runes[start:cursor])
	typed := strings.TrimLeft(line, " ")
	if typed == "" {
		return nil
	}
	prev := ""
	if start > 0 {
		ps, pe := lineBounds(runes, start-1)
		prev = string(runes[ps:pe])
	}
	start += len([]rune(line)) - len([]rune(typed)) // after the indent
	// a heading or a cue only after a blank line, as Fountain wants
	if strings.TrimSpace(prev) != "" {
		return nil
	}
	c := &completion{start: start, end: end, prefix: typed + closing}
	add := func(cand, label string) {
		if !slices.Contains(c.candidates, cand) && !strings.EqualFold(cand, typed+closing) {
			c.candidates, c.labels = append(c.candidates, cand), append(c.labels, label)
		}
	}
	upper := strings.ToUpper(typed)
	if name, ext, ok := typedExtension(typed); ok && !isSceneHeadingIn(typed, cx.starts) { // NAME (V: the language's extensions (○台所（ is a heading)
		open, shut := "(", ")"
		if strings.HasSuffix(name, "（") {
			open, shut = "（", "）"
		}
		name = strings.TrimSuffix(name, open)
		for _, e := range cx.l.Extensions {
			if hasPrefixFold(e, ext) {
				add(name+open+e+shut, e)
			}
		}
		return c.orNil()
	}
	if closing != "" {
		return nil
	}
	long := len([]rune(typed)) >= 2
	if long {
		for _, h := range cx.heads {
			if strings.HasPrefix(h, upper) {
				add(h, h)
			}
		}
	}
	if typed == upper && hasLetter(typed) && !strings.ContainsAny(typed, ".!?:") {
		for _, n := range cueNames(text, cursor, cx.names) {
			if strings.HasPrefix(n, upper) {
				add(n, n)
			}
		}
	}
	// read by the language's words alone; the script's places only for a
	// line that is a heading being typed (they are read once per line)
	states := slotSource{l: cx.l}.slotsAt(typed)
	if len(states) == 0 {
		return c.orNil()
	}
	src := slotSourceTyping(text, cursor, cx.l, cx.starts)
	for _, st := range states {
		if st.start == 0 && !long {
			continue // a heading's first word from two letters on, as before
		}
		for _, w := range src.choices(st.slot, st.form) {
			if cand := st.lead + w; hasPrefixFold(cand, st.prefix) {
				add(string([]rune(typed)[:st.start])+cand, w)
			}
		}
	}
	return c.orNil()
}

// orNil is c, nil if it has nothing to give.
func (c *completion) orNil() *completion {
	if len(c.candidates) == 0 {
		return nil
	}
	return c
}

// hasPrefixFold tells whether s starts with prefix, in any case.
func hasPrefixFold(s, prefix string) bool {
	return strings.HasPrefix(strings.ToUpper(s), strings.ToUpper(prefix))
}

// typedExtension reads a cue with an extension being typed ("ANNA (V",
// "张三（画"): the name with its bracket, what is typed of the extension.
func typedExtension(typed string) (name, ext string, ok bool) {
	i := max(strings.LastIndex(typed, "("), strings.LastIndex(typed, "（"))
	if i <= 0 || strings.ContainsAny(typed[i:], ")）") {
		return "", "", false
	}
	b := typed[i:]
	open := "("
	if strings.HasPrefix(b, "（") {
		open = "（"
	}
	name = strings.TrimSpace(typed[:i])
	if name == "" || !hasLetter(name) || name != strings.ToUpper(name) {
		return "", "", false
	}
	sp := ""
	if open == "(" {
		sp = " "
	}
	return name + sp + open, strings.TrimPrefix(b, open), true
}

// cueNames are the names a cue being typed at cursor may be, in order:
// first (the source's), the scene's speakers (the most recent first, but
// the one who spoke last after the others), then the script's by how
// often they speak.
func cueNames(text string, cursor int, first []string) []string {
	out := slices.Clone(first)
	addOnce := func(n string) {
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	// the scene so far: back from the cursor to its heading
	runes := []rune(text)
	before := string(runes[:min(cursor, len(runes))])
	lines := strings.Split(before, "\n")
	var recent []string
	for i := len(lines) - 2; i >= 0; i-- {
		t := strings.TrimSpace(lines[i])
		if (i == 0 || strings.TrimSpace(lines[i-1]) == "") && isSceneHeadingIn(t, nil) {
			break
		}
		if (i > 0 && strings.TrimSpace(lines[i-1]) != "") || strings.TrimSpace(lines[i+1]) == "" {
			continue
		}
		t = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "@"), "^"))
		if !isCharacterCue(t) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "@") {
			continue
		}
		if p := strings.IndexAny(t, "(（"); p > 0 {
			t = strings.TrimSpace(t[:p])
		}
		if t = strings.ToUpper(t); t != "" && !slices.Contains(recent, t) {
			recent = append(recent, t)
		}
	}
	if len(recent) > 1 { // the one who spoke last is seldom next
		recent = append(recent[1:], recent[0])
	}
	for _, n := range recent {
		addOnce(n)
	}
	counts := characterNames(text)
	names := make([]string, 0, len(counts))
	for n := range counts {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	for _, n := range names {
		addOnce(n)
	}
	return out
}

// characterNames are the script's character names (without extensions
// like (V.O.)) and how often they speak.
func characterNames(text string) map[string]int {
	names := map[string]int{}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == "" || (i > 0 && strings.TrimSpace(lines[i-1]) != "") {
			continue
		}
		t = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "@"), "^"))
		if !isCharacterCue(t) && !strings.HasPrefix(strings.TrimSpace(l), "@") {
			continue
		}
		if p := strings.Index(t, "("); p > 0 {
			t = strings.TrimSpace(t[:p])
		}
		if t != "" {
			names[strings.ToUpper(t)]++
		}
	}
	return names
}

// sceneHeadings are the script's scene headings (without scene numbers
// or notes, such as a color) and how often they come back.
func sceneHeadings(text string, starts []string) map[string]int {
	heads := map[string]int{}
	for _, l := range strings.Split(text, "\n") {
		t, _ := lex.SceneNumber(l)
		t = strings.TrimSpace(t)
		if isSceneHeadingIn(t, starts) || (strings.HasPrefix(t, ".") && !strings.HasPrefix(t, "..")) {
			heads[strings.ToUpper(t)]++
		}
	}
	return heads
}

// completer keeps the Tab cycle: the completion being cycled through
// and which candidate is on the line.
type completer struct {
	starts func() []string // the script's scene heading starts (nil: English)
	// first are the names and headings to offer first at the cursor
	// (the source's; nil: none)
	first func(cursor int) (names, heads []string)
	// lang is the script's language (nil: the starts')
	lang func() rules.Language

	// chosen is the list's chosen candidate of c, if it shows one
	chosen func(c *completion) int

	c     *completion
	index int
}

// tab completes or goes to the next candidate; false if there is
// nothing to complete (Tab then indents as before).
func (k *completer) tab(e *editor.ScriptEditor) bool {
	text, cursor := e.Text(), e.CursorOffset()
	if k.c != nil {
		// still on the line Tab completed, unchanged since: next one
		runes := []rune(text)
		cur := k.c.candidates[k.index]
		if k.c.start+len([]rune(cur)) == cursor && cursor <= len(runes) && string(runes[k.c.start:cursor]) == cur {
			if len(k.c.candidates) < 2 {
				// nothing to go through: the line after it comes next
				// (the keyboard flow's: a cue's dialogue)
				k.c = nil
				return false
			}
			k.index = (k.index + 1) % len(k.c.candidates)
			k.replace(e, cur, k.c.candidates[k.index])
			return true
		}
	}
	var starts, names, heads []string
	if k.starts != nil {
		starts = k.starts()
	}
	if k.first != nil {
		names, heads = k.first(cursor)
	}
	cx := completeCtx{starts: starts, l: languageOfStarts(starts), names: names, heads: heads}
	if k.lang != nil {
		cx.l = k.lang()
	}
	c := completeIn(text, cursor, cx)
	if c == nil {
		k.c = nil
		return false
	}
	k.c, k.index = c, 0
	if k.chosen != nil { // the one chosen in the list (complete_list.go)
		if i := k.chosen(c); i > 0 && i < len(c.candidates) {
			k.index = i
		}
	}
	k.replace(e, c.prefix, c.candidates[0])
	return true
}

// active reports whether Tab is going through completions: the line is
// the one Tab put there, unchanged since.
func (k *completer) active(e *editor.ScriptEditor) bool {
	if k == nil || k.c == nil || len(k.c.candidates) < 2 {
		return false // one: nothing to go through, the line after it comes next
	}
	cur := k.c.candidates[k.index]
	cursor := e.CursorOffset()
	b := e.Buffer()
	end := k.c.start + len([]rune(cur))
	return end == cursor && cursor <= b.Len() && b.Slice(k.c.start, cursor) == cur && !b.HasSelection()
}

// cycleHint is what Tab gives next while going through completions.
func (k *completer) cycleHint() string {
	n := len(k.c.candidates)
	if n < 2 {
		return ""
	}
	var next []string
	for i := 1; i < n && i <= 4; i++ {
		next = append(next, k.c.labels[(k.index+i)%n])
	}
	h := "Tab: " + strings.Join(next, " · ")
	if n-1 > len(next) {
		h += " …"
	}
	return h
}

// pick puts candidate i of the completions being gone through on the
// line (the list's arrows and taps).
func (k *completer) pick(e *editor.ScriptEditor, i int) {
	if k.c == nil || i < 0 || i >= len(k.c.candidates) {
		return
	}
	cur := k.c.candidates[k.index]
	k.index = i
	k.replace(e, cur, k.c.candidates[i])
}

func (k *completer) replace(e *editor.ScriptEditor, from, to string) {
	start := k.c.start
	e.Edit(func(b *buffer.Buffer) {
		b.Replace(start, start+len([]rune(from)), to)
		b.SetCursor(start+len([]rune(to)), false)
	})
}

// hint is what the status bar shows for the cursor's completions.
func completionHint(text string, cursor int, starts []string) string {
	return completionHintWith(text, cursor, starts, nil, nil)
}

// completionHintWith is completionHint with names and headings first.
func completionHintWith(text string, cursor int, starts, names, heads []string) string {
	return hintOf(completeFirst(text, cursor, starts, names, heads))
}

// hintOf is the status bar's hint for c.
func hintOf(c *completion) string {
	if c == nil {
		return ""
	}
	shown := c.labels
	if len(shown) > 4 {
		shown = shown[:4]
	}
	h := "Tab: " + strings.Join(shown, " · ")
	if len(c.labels) > len(shown) {
		h += " …"
	}
	return h
}

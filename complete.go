package main

import (
	"sort"
	"strings"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Tab completes the character name or scene heading being typed, from
// the ones the script already has (the most used first); Tab again goes
// to the next one. While typing, the status bar shows what Tab would give.

// completion is what Tab completes on the line with the cursor.
type completion struct {
	start      int      // where the completed text starts (after the indent)
	end        int      // the cursor, at the end of the line
	prefix     string   // what was typed
	candidates []string // what it can become
}

// completeAt finds the completions for the cursor in text, if it is at
// the end of a line that begins a character name or a scene heading.
func completeAt(text string, cursor int, starts []string) *completion {
	return completeFirst(text, cursor, starts, nil, nil)
}

// completeFirst is completeAt with names and headings that go first,
// in their order, when they fit what was typed (the source's speakers,
// the heading it reads; suggest.go), the script's need not have them.
func completeFirst(text string, cursor int, starts []string, firstNames, firstHeads []string) *completion {
	runes := []rune(text)
	if cursor > len(runes) {
		return nil
	}
	start, end := lineBounds(runes, cursor)
	if cursor != end {
		return nil
	}
	line := string(runes[start:end])
	typed := strings.TrimLeft(line, " ")
	if typed == "" {
		return nil
	}
	prev := ""
	if start > 0 {
		ps, pe := lineBounds(runes, start-1)
		prev = string(runes[ps:pe])
	}
	afterBlank := strings.TrimSpace(prev) == ""
	start += len([]rune(line)) - len([]rune(typed)) // after the indent

	// scene headings for a heading's start ("ext", "INT. K"), names for an
	// upper-case start after a blank line; a heading only after a blank
	// line too, as Fountain wants
	if !afterBlank {
		return nil
	}
	pool := map[string]int{}
	upper := strings.ToUpper(typed)
	var first []string
	if len([]rune(typed)) >= 2 {
		for h, n := range sceneHeadings(text, starts) {
			pool[h] = n
		}
		first = append(first, firstHeads...)
	}
	if typed == upper && hasLetter(typed) && !strings.ContainsAny(typed, ".!?:") {
		for name, n := range characterNames(text) {
			pool[name] += n
		}
		first = append(first, firstNames...)
	}
	rank := map[string]int{}
	for i, f := range first {
		if _, ok := rank[f]; !ok {
			rank[f] = len(first) - i
			if _, ok := pool[f]; !ok {
				pool[f] = 0
			}
		}
	}
	c := &completion{start: start, end: end, prefix: typed}
	for cand := range pool {
		if strings.HasPrefix(cand, upper) && cand != upper {
			c.candidates = append(c.candidates, cand)
		}
	}
	if len(c.candidates) == 0 {
		return nil
	}
	sort.Slice(c.candidates, func(i, j int) bool {
		a, b := c.candidates[i], c.candidates[j]
		if rank[a] != rank[b] {
			return rank[a] > rank[b]
		}
		if pool[a] != pool[b] {
			return pool[a] > pool[b]
		}
		return a < b
	})
	return c
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
	c := completeFirst(text, cursor, starts, names, heads)
	if c == nil {
		k.c = nil
		return false
	}
	k.c, k.index = c, 0
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
		next = append(next, k.c.candidates[(k.index+i)%n])
	}
	h := "Tab: " + strings.Join(next, " · ")
	if n-1 > len(next) {
		h += " …"
	}
	return h
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
	c := completeFirst(text, cursor, starts, names, heads)
	if c == nil {
		return ""
	}
	shown := c.candidates
	if len(shown) > 4 {
		shown = shown[:4]
	}
	h := "Tab: " + strings.Join(shown, " · ")
	if len(c.candidates) > len(shown) {
		h += " …"
	}
	return h
}

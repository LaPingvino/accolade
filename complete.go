package main

import (
	"sort"
	"strings"

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
	if len([]rune(typed)) >= 2 {
		for h, n := range sceneHeadings(text, starts) {
			pool[h] = n
		}
	}
	if typed == upper && hasLetter(typed) && !strings.ContainsAny(typed, ".!?:") {
		for name, n := range characterNames(text) {
			pool[name] += n
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

// sceneHeadings are the script's scene headings (without scene numbers)
// and how often they come back.
func sceneHeadings(text string, starts []string) map[string]int {
	heads := map[string]int{}
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if i := strings.Index(t, " #"); i > 0 && strings.HasSuffix(t, "#") {
			t = strings.TrimSpace(t[:i])
		}
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
			k.index = (k.index + 1) % len(k.c.candidates)
			k.replace(e, cur, k.c.candidates[k.index])
			return true
		}
	}
	var starts []string
	if k.starts != nil {
		starts = k.starts()
	}
	c := completeAt(text, cursor, starts)
	if c == nil {
		k.c = nil
		return false
	}
	k.c, k.index = c, 0
	k.replace(e, c.prefix, c.candidates[0])
	return true
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
	c := completeAt(text, cursor, starts)
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

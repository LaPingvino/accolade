package main

import (
	"strings"
)

// Planning on the cards: a synopsis written on a card, and new scenes.

// synopsisLines are the lines of item i's synopsis ("= ..." under it, up
// to the next item), as outlineOf reads them.
func synopsisLines(lines []string, items []outlineItem, i int) []int {
	end := len(lines)
	if i+1 < len(items) {
		end = items[i+1].Line
	}
	var at []int
	for n := items[i].Line + 1; n < end; n++ {
		t := strings.TrimSpace(lines[n])
		switch {
		case isSynopsisLine(t):
			at = append(at, n)
		case len(at) > 0: // its own ends at the first line after it
			return at
		case t == "" || isNoteLine(t):
		default:
			return at
		}
	}
	return at
}

// setSynopsis gives item i the synopsis syn: one "= " line in place of
// its own synopsis lines (right under its heading if it had none), the
// synopses after it left as they are; none when syn is empty.
func setSynopsis(text string, starts []string, i int, syn string) string {
	items := outlineOf(text, starts)
	if i < 0 || i >= len(items) {
		return text
	}
	lines := strings.Split(text, "\n")
	old := synopsisLines(lines, items, i)
	drop := map[int]bool{}
	for _, n := range old {
		drop[n] = true
	}
	syn = strings.Join(strings.Fields(syn), " ") // one line
	at := items[i].Line                          // the new line goes after this one
	if len(old) > 0 {
		at = old[0] - 1
	}
	var out []string
	for n, l := range lines {
		if !drop[n] {
			out = append(out, l)
		}
		if n == at && syn != "" {
			out = append(out, "= "+syn)
		}
	}
	if len(old) > 0 && syn == "" && old[0] > 0 && old[len(old)-1]+1 < len(lines) &&
		strings.TrimSpace(lines[old[0]-1]) == "" && strings.TrimSpace(lines[old[len(old)-1]+1]) == "" {
		// a synopsis between blank lines gone: one blank line left
		k := old[0] - 1 - countBefore(drop, old[0]-1)
		out = append(out[:k], out[k+1:]...)
	}
	return strings.Join(out, "\n")
}

// countBefore is how many lines of drop come before line n.
func countBefore(drop map[int]bool, n int) int {
	c := 0
	for k := range drop {
		if k < n {
			c++
		}
	}
	return c
}

// addScene puts a new scene, heading with synopsis syn, before item at (at
// the end when at is len(items)), and returns the text and the heading's
// line. A heading that is not one in the script's language is forced with
// a ".".
func addScene(text string, starts []string, at int, heading, syn string) (string, int) {
	items := outlineOf(text, starts)
	heading = strings.TrimSpace(heading)
	if !isSceneHeadingIn(heading, starts) && !strings.HasPrefix(heading, ".") {
		heading = "." + heading
	}
	var lines []string
	if t := strings.TrimRight(text, "\n"); t != "" {
		lines = strings.Split(t, "\n")
	}
	line := len(lines)
	if at >= 0 && at < len(items) {
		line = items[at].Line
	}
	before := append([]string(nil), lines[:line]...)
	if len(before) > 0 && strings.TrimSpace(before[len(before)-1]) != "" {
		before = append(before, "") // a heading follows a blank line
	}
	block := []string{heading}
	if syn = strings.Join(strings.Fields(syn), " "); syn != "" {
		block = append(block, "= "+syn)
	}
	if line < len(lines) {
		block = append(block, "") // and the next item follows one too
	}
	out := append(append(before, block...), lines[line:]...)
	return strings.Join(out, "\n") + "\n", len(before)
}

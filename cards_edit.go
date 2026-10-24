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
	inBoneyard := false
	for n := items[i].Line + 1; n < end; n++ {
		t := strings.TrimSpace(lines[n])
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		if strings.HasPrefix(t, "=") && !strings.HasPrefix(t, "==") {
			at = append(at, n)
		}
	}
	return at
}

// setSynopsis gives item i the synopsis syn: one "= " line right under
// its heading, in place of the synopsis lines it had (none when syn is
// empty).
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
	var out []string
	for n, l := range lines {
		if drop[n] {
			continue
		}
		out = append(out, l)
		if n == items[i].Line && syn != "" {
			out = append(out, "= "+syn)
		}
	}
	return strings.Join(out, "\n")
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

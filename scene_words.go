package main

import (
	"slices"
	"strings"
)

// A scene's words, read for matching it to the source (scene_align.go,
// linkup.go).

// commonFrom is how many passages a source has before its commonest
// words are left out of the matching.
const commonFrom = 50

// sceneBody is a scene's words: what its people say, and the rest.
type sceneBody struct {
	Line     int
	Heading  string
	Dialogue [][]string // each speech, normalised words
	Words    map[string]bool
	Linked   bool // a link is in it already
}

// scenesOf reads the script's scenes for their words: the heading, the
// action and the dialogue, notes and boneyard left out.
func scenesOf(text string, starts []string) []sceneBody {
	body := text[:bodyEndByte(text)]
	lines := strings.Split(body, "\n")
	outline := outlineOf(body, starts)
	links := parseLinks(body)
	var out []sceneBody
	for k, it := range outline {
		if !it.Scene {
			continue
		}
		end := len(lines)
		if k+1 < len(outline) {
			end = outline[k+1].Line
		}
		sc := sceneBody{Line: it.Line, Heading: it.Text, Words: map[string]bool{}}
		for _, l := range links {
			if l.Line >= it.Line && l.Line < end {
				sc.Linked = true
			}
		}
		add := func(ws []string) {
			for _, w := range ws {
				sc.Words[w] = true
			}
		}
		add(normWords(it.Text))
		speaking, blank := false, true
		for _, raw := range lines[min(it.Line+1, end):end] {
			line := strings.TrimSpace(stripNotes(raw))
			switch {
			case line == "":
				speaking, blank = false, true
				continue
			case strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*/") || strings.HasPrefix(line, ">") ||
				strings.HasSuffix(line, "TO:") || strings.HasPrefix(line, "="):
			case blank && isCharacterCue(line):
				speaking = true
			case speaking && strings.HasPrefix(line, "("): // a parenthetical
			case speaking:
				ws := normWords(line)
				sc.Dialogue = append(sc.Dialogue, ws)
				add(ws)
			default:
				add(normWords(line))
			}
			blank = false
		}
		out = append(out, sc)
	}
	return out
}

// stripNotes is a line without its [[notes]].
func stripNotes(line string) string {
	for {
		i := strings.Index(line, "[[")
		if i < 0 {
			return line
		}
		j := strings.Index(line[i:], "]]")
		if j < 0 {
			return line[:i]
		}
		line = line[:i] + " " + line[i+j+2:]
	}
}

// sortedKeys is a set's words in order: sums over them come out the
// same each time (a map's order differs from run to run).
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for w := range set {
		out = append(out, w)
	}
	slices.Sort(out)
	return out
}

// overlap is how much of a's words b has (0–1).
func overlap(a, b []string) float64 {
	if len(a) == 0 {
		return 0
	}
	in := map[string]bool{}
	for _, w := range b {
		in[w] = true
	}
	hit := 0
	for _, w := range a {
		if in[w] {
			hit++
		}
	}
	return float64(hit) / float64(len(a))
}

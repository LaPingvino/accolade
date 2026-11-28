// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The story as the timeline and the strip over the editor show it
// (timeline.go, timeline_strip.go): the scenes in story order with their
// lengths, colours, sections, who speaks in them and their tags, read from
// the Fountain text alone (LOOP.md, TIMELINE: no state of its own). A
// scene's length is in eighths of a page, as the stripboard measures it
// once the pages are laid out (the schedule's measure); until then, and
// while typing makes the measure stale, it is counted from the lines.

// storyScene is a scene of the story.
type storyScene struct {
	item     int // its outline item
	line     int
	heading  string
	color    string
	synopsis string
	eighths  int // its length, in eighths of a page (1 at least)
	// empty: nothing written under the heading (synopsisOnly: but its
	// synopsis)
	empty, synopsisOnly bool
	sections            []int    // its sections' items, the outermost first
	chars               []string // who speaks in it, by charKey
	tags                []string // its heading's and its sections' notes (not colours)
}

// storyBand is a section over the scenes it holds.
type storyBand struct {
	item        int
	level       int // 1 for #, 2 for ##, …
	title       string
	color       string
	first, last int // its scenes (first > last: none)
}

// storyChar is a character with a lane: who speaks.
type storyChar struct {
	key, name string
	scenes    []int // the scenes they speak in, by index
}

// storyModel is the story read from a text.
type storyModel struct {
	items  []outlineItem
	scenes []storyScene
	bands  []storyBand
	chars  []storyChar // the most present first
	tags   []string    // scriptTags' order, colours left out
	total  int         // the scenes' eighths together
}

// storyOf reads the story of a screenplay's text.
func storyOf(text string, starts []string, kinds []syntax.Kind) storyModel {
	m := storyModel{items: outlineOf(text, starts)}
	lines := strings.Split(text, "\n")
	if kinds == nil || len(kinds) != len(lines) {
		kinds = syntax.ClassifyWith(lines, starts)
	}
	end := strings.Count(text[:bodyEndByte(text)], "\n")
	var open []int // the sections around, by level
	for i, it := range m.items {
		if !it.Scene {
			for len(open) > 0 && m.items[open[len(open)-1]].Level >= it.Level {
				open = open[:len(open)-1]
			}
			open = append(open, i)
			m.bands = append(m.bands, storyBand{item: i, level: it.Level, title: it.Text, color: it.Color, first: len(m.scenes), last: len(m.scenes) - 1})
			continue
		}
		next := min(end, len(lines))
		if i+1 < len(m.items) {
			next = min(next, m.items[i+1].Line)
		}
		s := storyScene{item: i, line: it.Line, heading: it.Text, color: it.Color, synopsis: it.Synopsis, sections: slices.Clone(open)}
		s.eighths, s.empty = sceneLength(lines, kinds, it.Line+1, next)
		s.synopsisOnly = s.empty && it.Synopsis != ""
		m.scenes = append(m.scenes, s)
		m.total += s.eighths
	}
	// the bands' scenes: up to the next band of their level or above
	for b := range m.bands {
		last := len(m.scenes) - 1
		for c := b + 1; c < len(m.bands); c++ {
			if m.bands[c].level <= m.bands[b].level {
				last = m.bands[c].first - 1
				break
			}
		}
		m.bands[b].last = last
	}
	// the tags: a heading's notes, and its sections'
	notes := func(line int) []string {
		var out []string
		for _, n := range lex.Notes(lines[line]) {
			if t := strings.TrimSpace(n); t != "" && lex.NoteColor("[["+n+"]]") == "" {
				out = append(out, t)
			}
		}
		return out
	}
	for k := range m.scenes {
		s := &m.scenes[k]
		for _, sec := range s.sections {
			s.tags = appendFold(s.tags, notes(m.items[sec].Line)...)
		}
		s.tags = appendFold(s.tags, notes(s.line)...)
	}
	for _, t := range scriptTags(text, starts) {
		if _, ok := noteRGB(t); !ok && lex.NoteColor("[["+t+"]]") == "" {
			m.tags = append(m.tags, t)
		}
	}
	// who speaks where (the Characters pane's reading)
	cm := screenplayChars(text, starts)
	byLine := map[int]int{}
	for k, s := range m.scenes {
		byLine[s.line] = k
	}
	for _, c := range cm.chars {
		if c.Not || c.Speeches == 0 {
			continue
		}
		sc := storyChar{key: c.Key, name: c.Name}
		for _, i := range c.Scenes {
			if k, ok := byLine[cm.scenes[i].line]; ok {
				sc.scenes = append(sc.scenes, k)
				m.scenes[k].chars = append(m.scenes[k].chars, c.Key)
			}
		}
		if len(sc.scenes) > 0 {
			m.chars = append(m.chars, sc)
		}
	}
	return m
}

// appendFold appends the tags not there yet (by their case-folded text).
func appendFold(tags []string, add ...string) []string {
	for _, t := range add {
		if !slices.ContainsFunc(tags, func(s string) bool { return strings.EqualFold(s, t) }) {
			tags = append(tags, t)
		}
	}
	return tags
}

// sceneLength counts the scene's lines from..to (the heading's next to the
// next heading) as printed lines, in eighths of a page: action wraps at
// about 60 characters, dialogue at 35, a parenthetical at 25; a blank line
// is a line, the synopsis, notes and boneyard none. empty: nothing in it
// but those.
func sceneLength(lines []string, kinds []syntax.Kind, from, to int) (eighths int, empty bool) {
	printed := 2 // the heading and the blank line after it
	empty = true
	inBone := false
	for n := from; n < to && n < len(lines); n++ {
		t := strings.TrimSpace(lines[n])
		if inBone || strings.HasPrefix(t, "/*") {
			inBone = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		k := syntax.Action
		if n < len(kinds) {
			k = kinds[n]
		}
		switch {
		case t == "":
			printed++
			continue
		case k == syntax.Synopsis || k == syntax.Note || strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]"):
			continue
		}
		empty = false
		width := actionColumns // (stats.go's page geometry)
		switch k {
		case syntax.Dialogue:
			width = dialogColumns
		case syntax.Parenthetical:
			width = parenColumns
		}
		printed += max(1, (utf8.RuneCountInString(t)+width-1)/width)
	}
	if empty {
		return 1, true
	}
	return max(1, (printed*8+linesPerPage/2)/linesPerPage), false
}

// current is the scene at line (the last whose heading is at or above
// it), -1 before the first.
func (m *storyModel) current(line int) int {
	k, _ := slices.BinarySearchFunc(m.scenes, line, func(s storyScene, l int) int {
		if s.line <= l {
			return -1
		}
		return 1
	})
	return k - 1
}

// goneFor are the characters who speak before scene k but not in the
// last few scenes up to it, the longest gone first: name and how many
// scenes since they last spoke (gap at least).
func (m *storyModel) goneFor(k, gap int) []storyGone {
	var out []storyGone
	if k < 0 {
		return nil
	}
	for _, c := range m.chars {
		last := -1
		for _, s := range c.scenes {
			if s <= k {
				last = s
			}
		}
		if last < 0 || k-last < gap {
			continue
		}
		out = append(out, storyGone{name: c.name, scenes: k - last})
	}
	slices.SortStableFunc(out, func(a, b storyGone) int { return b.scenes - a.scenes })
	return out
}

// storyGone is a character not seen for a while.
type storyGone struct {
	name   string
	scenes int
}

// withLengths takes the measured lengths (by heading line) where the
// pages were laid out: the stripboard's measure.
func (m *storyModel) withLengths(measured map[int]int) {
	if len(measured) == 0 {
		return
	}
	m.total = 0
	for k := range m.scenes {
		if e, ok := measured[m.scenes[k].line]; ok && e > 0 {
			m.scenes[k].eighths = e
		}
		m.total += m.scenes[k].eighths
	}
}

// eighthsText is a length in pages and eighths: "1 3/8", "5/8".
func eighthsText(e int) string { return lex.Eighths(e).String() }

// lexEighths is a length as Lexington's eighths (lengthText's).
func lexEighths(e int) lex.Eighths { return lex.Eighths(e) }

func countText(n int) string { return strconv.Itoa(n) }

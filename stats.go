package main

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// scriptStats is what the status bar shows about a script.
type scriptStats struct {
	Scenes int // scene headings in the script
	Scene  int // the scene the cursor is in (0 before the first)
	Words  int // words after the title page
	Pages  int // estimated pages
}

// Rough page geometry of a screenplay in Courier 12 pt: about 55 lines a
// page; action 60 columns wide, dialogue 35, parentheticals 25.
const (
	linesPerPage  = 55
	actionColumns = 60
	dialogColumns = 35
	parenColumns  = 25
)

// computeStats counts scenes, words and pages, and finds the scene the
// cursor (a rune offset) is in. The counts are made once per text (a
// cursor move only finds its scene again).
func computeStats(text string, cursor int, starts []string) scriptStats {
	if strings.TrimSpace(text) == "" {
		return scriptStats{}
	}
	c := countsOf(text, starts)
	st := c.stats
	cursorLine, n := 0, 0
	for _, r := range c.body {
		if n >= cursor {
			break
		}
		if r == '\n' {
			cursorLine++
		}
		n++
	}
	st.Scene, _ = slices.BinarySearch(c.headings, cursorLine+1) // the headings on lines up to the cursor's
	return st
}

// scriptCounts are a text's counts, without the cursor's scene.
type scriptCounts struct {
	stats    scriptStats
	body     string // the text counted: without the source kept in it
	headings []int  // the scene headings' lines
}

// counted is the last text counted, as computeStats asks of every event.
var counted struct {
	sync.Mutex
	text   string
	starts []string
	counts scriptCounts
	ok     bool
}

func countsOf(text string, starts []string) scriptCounts {
	counted.Lock()
	defer counted.Unlock()
	if counted.ok && sameString(counted.text, text) && slices.Equal(counted.starts, starts) {
		return counted.counts
	}
	c := countScript(text, starts)
	counted.text, counted.starts, counted.counts, counted.ok = text, slices.Clone(starts), c, true
	return c
}

func countScript(text string, starts []string) scriptCounts {
	var c scriptCounts
	if src, _ := tailBlocks(text); src.ok { // the source kept in it is not the script
		text = text[:src.bstart]
	}
	c.body = text
	st := &c.stats
	lines := strings.Split(text, "\n")
	rows := 0
	for i, k := range syntax.ClassifyIn(lines, starts, dialectForStarts(starts)) {
		if k == syntax.TitlePage {
			continue
		}
		if k == syntax.SceneHeading {
			st.Scenes++
			c.headings = append(c.headings, i)
		}
		line := strings.TrimSpace(lines[i])
		st.Words += countWordsIn(line)
		width := actionColumns
		switch k {
		case syntax.Dialogue:
			width = dialogColumns
		case syntax.Parenthetical:
			width = parenColumns
		}
		rows += max(1, (utf8.RuneCountInString(line)+width-1)/width)
	}
	if st.Words > 0 {
		st.Pages = max(1, (rows+linesPerPage-1)/linesPerPage)
	}
	return c
}

// String is the status bar text, e.g. "Scene 2 of 3 · 412 words · ~2 pages".
func (st scriptStats) String() string {
	var parts []string
	if st.Scenes > 0 {
		if st.Scene > 0 {
			parts = append(parts, fmt.Sprintf("Scene %d of %d", st.Scene, st.Scenes))
		} else {
			parts = append(parts, plural(st.Scenes, "scene"))
		}
	}
	parts = append(parts, plural(st.Words, "word"))
	if st.Pages > 0 {
		parts = append(parts, "~"+plural(st.Pages, "page"))
	}
	return strings.Join(parts, " · ")
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// countWordsIn counts the words in a line, leaving out punctuation on its
// own such as the dash in "INT. BARN - DAY".
func countWordsIn(line string) int {
	n := 0
	for _, f := range strings.Fields(line) {
		if strings.IndexFunc(f, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
			n++
		}
	}
	return n
}

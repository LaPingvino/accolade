package main

import (
	"fmt"
	"strings"
	"unicode"

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
// cursor (a rune offset) is in.
func computeStats(text string, cursor int) scriptStats {
	var st scriptStats
	if strings.TrimSpace(text) == "" {
		return st
	}
	runes := []rune(text)
	cursorLine := strings.Count(string(runes[:min(max(cursor, 0), len(runes))]), "\n")
	lines := strings.Split(text, "\n")
	rows := 0
	for i, k := range syntax.Classify(lines) {
		if k == syntax.TitlePage {
			continue
		}
		if k == syntax.SceneHeading {
			st.Scenes++
			if i <= cursorLine {
				st.Scene = st.Scenes
			}
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
		rows += max(1, (len([]rune(line))+width-1)/width)
	}
	if st.Words > 0 {
		st.Pages = max(1, (rows+linesPerPage-1)/linesPerPage)
	}
	return st
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

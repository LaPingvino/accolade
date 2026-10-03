package main

import (
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// screenplayEntry is the script editor: a multi-line Entry that formats
// a line once it is completed with Enter, and starts the next line at the
// indent the next element needs (dialogue after a character name).
//
// Formatting only ever touches the line just completed and goes in as
// typed edits, so it stays on the undo stack and never fights the writer
// mid-word.
type screenplayEntry struct {
	widget.Entry
	window *MainWindow
}

func newScreenplayEntry(w *MainWindow) *screenplayEntry {
	e := &screenplayEntry{window: w}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *screenplayEntry) TypedKey(key *fyne.KeyEvent) {
	if (key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter) && e.SelectedText() == "" {
		e.completeLine()
		return
	}
	e.Entry.TypedKey(key)
}

// completeLine handles Enter: format the current line if the cursor is at
// its end, then break the line and indent the new one.
func (e *screenplayEntry) completeLine() {
	text := []rune(e.Text)
	cursor := e.CursorTextOffset()
	start, end := lineBounds(text, cursor)
	line := string(text[start:end])

	element := "Action"
	if cursor == end {
		prev := ""
		if start > 0 {
			ps, _ := lineBounds(text, start-1)
			prev = string(text[ps : start-1])
		}
		formatted, kind := formatCompletedLine(prev, line)
		element = kind
		if formatted != line {
			e.batchEdit(func() {
				selectRange(&e.Entry, start, end)
				typeOverSelection(&e.Entry, formatted)
			})
		}
	}

	indent := nextLineIndent(element)
	e.batchEdit(func() {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		for i := 0; i < indent; i++ {
			e.Entry.TypedRune(' ')
		}
	})
}

// batchEdit runs several programmatic edits and reports a single change,
// instead of re-parsing the script for every typed rune.
func (e *screenplayEntry) batchEdit(edit func()) {
	onChanged := e.OnChanged
	e.OnChanged = nil
	edit()
	e.OnChanged = onChanged
	if onChanged != nil {
		onChanged(e.Text)
	}
}

// lineBounds returns the rune range [start, end) of the line containing pos.
func lineBounds(text []rune, pos int) (start, end int) {
	start, end = pos, pos
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	for end < len(text) && text[end] != '\n' {
		end++
	}
	return start, end
}

// formatCompletedLine returns how a just-completed line should look, given
// the line before it, and which element it is. Lines that are not clearly
// a screenplay element are returned unchanged.
func formatCompletedLine(prev, line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, "Empty"
	}
	upper := strings.ToUpper(trimmed)
	afterBlank := strings.TrimSpace(prev) == ""

	switch {
	case isSceneHeading(upper) && afterBlank:
		return upper, "Scene Heading"
	case isTransition(upper) && afterBlank:
		return strings.Repeat(" ", TransitionIndent) + upper, "Transition"
	case strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") && !afterBlank:
		return strings.Repeat(" ", ParentheticalIndent) + trimmed, "Parenthetical"
	case isCharacterCue(trimmed) && afterBlank:
		return strings.Repeat(" ", CharacterIndent) + trimmed, "Character"
	case strings.HasPrefix(line, strings.Repeat(" ", DialogueIndent)):
		return line, "Dialogue"
	}
	return line, "Action"
}

// nextLineIndent is how far the line after an element of the given kind
// starts: dialogue follows a character name or a parenthetical.
func nextLineIndent(element string) int {
	switch element {
	case "Character", "Parenthetical":
		return DialogueIndent
	}
	return 0
}

func isSceneHeading(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, p := range []string{"INT.", "EXT.", "EST.", "INT/EXT.", "I/E.", "INT ", "EXT "} {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	return false
}

// isTransition follows Fountain: an uppercase line ending in TO:, plus the
// usual endings. FADE IN: is deliberately not one.
func isTransition(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	if strings.HasSuffix(upper, " TO:") || upper == "CUT TO:" {
		return true
	}
	switch upper {
	case "FADE OUT.", "FADE OUT:", "FADE TO BLACK.", "THE END":
		return true
	}
	return false
}

// isCharacterCue reports whether a trimmed line reads as a character name:
// entirely uppercase with at least one letter, optionally followed by an
// extension like (V.O.) or (CONT'D).
func isCharacterCue(trimmed string) bool {
	name := trimmed
	if i := strings.Index(trimmed, "("); i > 0 && strings.HasSuffix(trimmed, ")") {
		name = strings.TrimSpace(trimmed[:i])
	}
	if len(name) < 2 || strings.HasSuffix(name, ":") || strings.ContainsAny(name, ".!?") {
		return false
	}
	return name == strings.ToUpper(name) && hasLetter(name)
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

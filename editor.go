package main

import (
	"strings"
	"unicode"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// newScriptEditor creates the screenplay editor (internal/editor) with
// Accolade's behaviour: Fountain colouring, and on Enter the completed line
// is formatted and the next one indented (dialogue after a character
// name). Formatting only touches the line just completed and is one undo
// step together with the line break.
func newScriptEditor(w *MainWindow) *editor.ScriptEditor {
	e := editor.New("")
	e.Syntax = true
	e.OnEnter = func() bool { return completeLine(w, e) }
	tab := completer{starts: w.sceneStarts}
	e.OnTab = func() bool { return tab.tab(e) }
	// vim mode (Preferences > Editor), before anything else sees a key
	vi := newVim(e)
	vi.on = func() bool { return w != nil && w.settings != nil && w.settings.GetBoolean("vim-mode") }
	vi.status = func(s string) {
		if w != nil {
			w.setStatus(s)
		}
	}
	vi.search = func() {
		if w != nil {
			w.showFind()
		}
	}
	if w != nil {
		w.vim = vi
	}
	e.OnKey = vi.key
	e.OnShortcut = vi.shortcut
	e.OnRune = func(r rune) bool {
		if vi.rune(r) {
			return true
		}
		if w == nil || w.settings == nil || !w.settings.GetBoolean("auto-close-brackets") {
			return false
		}
		return closeBracket(e, r)
	}
	return e
}

// brackets are the pairs closed as they are typed: a parenthetical, a
// note ([[...]]).
var brackets = map[rune]rune{'(': ')', '[': ']'}

// closeBracket types an opening bracket with its closing one after the
// cursor (or around the selection), and steps over a closing bracket
// that is already there. It reports false for other characters.
func closeBracket(e *editor.ScriptEditor, r rune) bool {
	if closing, ok := brackets[r]; ok {
		e.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				start, end := b.Selection()
				sel := b.Slice(start, end)
				b.Replace(start, end, string(r)+sel+string(closing))
				if sel == "" {
					b.SetCursor(start+1, false)
				} else {
					b.Select(start+1, start+1+len([]rune(sel)))
				}
			})
		})
		return true
	}
	for _, closing := range brackets {
		if r != closing {
			continue
		}
		b := e.Buffer()
		if c := b.Cursor(); !b.HasSelection() && c < b.Len() && b.Slice(c, c+1) == string(r) {
			e.Navigate(func(b *buffer.Buffer) { b.SetCursor(c+1, false) })
			return true
		}
	}
	return false
}

// completeLine handles Enter: format the current line if the cursor is at
// its end, then break the line and indent the new one. It reports false
// to leave Enter to the editor (auto-indent off, or a selection).
func completeLine(w *MainWindow, e *editor.ScriptEditor) bool {
	if w != nil && w.settings != nil && !w.settings.GetBoolean("auto-indent") {
		return false
	}
	if e.SelectedText() != "" {
		return false
	}

	text := []rune(e.Text())
	cursor := e.CursorOffset()
	start, end := lineBounds(text, cursor)
	line := string(text[start:end])

	element := "Action"
	formatted := line
	if cursor == end {
		prev := ""
		if start > 0 {
			ps, _ := lineBounds(text, start-1)
			prev = string(text[ps : start-1])
		}
		formatted, element = formatCompletedLine(prev, line, w.sceneStarts())
	}

	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if formatted != line {
				b.Replace(start, end, formatted)
			}
			b.Insert("\n" + strings.Repeat(" ", nextLineIndent(element)))
		})
	})
	return true
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
func formatCompletedLine(prev, line string, starts []string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, "Empty"
	}
	upper := strings.ToUpper(trimmed)
	afterBlank := strings.TrimSpace(prev) == ""

	switch {
	case isSceneHeadingIn(upper, starts) && afterBlank:
		return upper, "Scene Heading"
	case isTransition(upper) && afterBlank:
		// Only "...TO:" is a transition by itself in Fountain; others such
		// as FADE OUT. need the ">" marker for other Fountain apps (and
		// the PDF export) to read them as one.
		if !strings.HasSuffix(upper, "TO:") && !strings.HasPrefix(upper, ">") {
			upper = "> " + upper
		}
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

// isSceneHeading reports whether a line starts like an English scene
// heading.
func isSceneHeading(line string) bool { return isSceneHeadingIn(line, nil) }

// isTransition follows Fountain: an uppercase line ending in TO:, or one
// forced with ">", plus the usual endings like FADE OUT. that a writer types
// without the marker. FADE IN: is deliberately not one.
func isTransition(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	if strings.HasSuffix(upper, " TO:") || upper == "CUT TO:" {
		return true
	}
	if strings.HasPrefix(upper, ">") && !strings.HasSuffix(upper, "<") {
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

// Package syntax tells which Fountain element each line of a script is,
// for colouring it in Accolade's editor (docs/EDITOR_WIDGET.md, step 5).
// It follows the Fountain rules as lexington parses them, line by line.
package syntax

import (
	"strings"
	"unicode"
)

// Kind is the Fountain element a line belongs to.
type Kind int

const (
	Action Kind = iota
	Empty
	TitlePage
	SceneHeading
	Character
	Parenthetical
	Dialogue
	Transition
	Centered
	Section
	Synopsis
	Note
	PageBreak
)

// SceneStarts are the English scene heading starts: Classify's.
var SceneStarts = []string{"INT.", "EXT.", "EST.", "INT./EXT.", "INT/EXT.", "I/E.", "INT ", "EXT ", "EST ", "INT/EXT ", "I/E "}

// Classify returns the kind of each line, with English scene headings.
func Classify(lines []string) []Kind { return ClassifyWith(lines, SceneStarts) }

// ClassifyWith returns the kind of each line, with the scene headings of
// a language: what they start with ("INT.", "EN. ").
func ClassifyWith(lines []string, sceneStarts []string) []Kind {
	if sceneStarts == nil {
		sceneStarts = SceneStarts
	}
	kinds := make([]Kind, len(lines))
	trimmed := func(i int) string {
		if i < 0 || i >= len(lines) {
			return ""
		}
		return strings.TrimSpace(lines[i])
	}

	i := 0
	// title page: leading "Key: value" fields and their indented values
	if titleField(lines, 0, sceneStarts) {
		for ; i < len(lines); i++ {
			t := trimmed(i)
			switch {
			case t == "":
				kinds[i] = Empty
				if i+1 < len(lines) && !titleField(lines, i+1, sceneStarts) && !indented(lines[i+1]) {
					i++
					goto body
				}
			case titleField(lines, i, sceneStarts) || indented(lines[i]):
				kinds[i] = TitlePage
			default:
				goto body
			}
		}
	}
body:
	inDialogue := false
	for ; i < len(lines); i++ {
		t := trimmed(i)
		upper := strings.ToUpper(t)
		afterBlank := trimmed(i-1) == ""
		switch {
		case t == "":
			kinds[i] = Empty
			inDialogue = false
			continue
		case strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]"):
			kinds[i] = Note
		case strings.HasPrefix(t, "#"):
			kinds[i] = Section
		case strings.HasPrefix(t, "=") && strings.Trim(t, "=") == "" && len(t) >= 3:
			kinds[i] = PageBreak
		case strings.HasPrefix(t, "="):
			kinds[i] = Synopsis
		case strings.HasPrefix(t, ">") && strings.HasSuffix(t, "<"):
			kinds[i] = Centered
		case strings.HasPrefix(t, ">"), afterBlank && t == upper && strings.HasSuffix(t, " TO:"):
			kinds[i] = Transition
		case afterBlank && (strings.HasPrefix(t, ".") && !strings.HasPrefix(t, "..") || hasAnyPrefix(upper, sceneStarts)):
			kinds[i] = SceneHeading
		case inDialogue && strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")"):
			kinds[i] = Parenthetical
		case inDialogue:
			kinds[i] = Dialogue
		case afterBlank && trimmed(i+1) != "" && (strings.HasPrefix(t, "@") || isCharacterCue(t)):
			kinds[i] = Character
			inDialogue = true
			continue
		default:
			kinds[i] = Action
		}
		if kinds[i] != Parenthetical && kinds[i] != Dialogue {
			inDialogue = false
		}
	}
	return kinds
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// isCharacterCue: uppercase name with a letter, optional extension like
// (V.O.), optional dual dialogue caret.
func isCharacterCue(t string) bool {
	name := strings.TrimSpace(strings.TrimSuffix(t, "^"))
	if i := strings.Index(name, "("); i > 0 && strings.HasSuffix(name, ")") {
		name = strings.TrimSpace(name[:i])
	}
	if name == "" || name != strings.ToUpper(name) || strings.ContainsAny(name, ":!?") {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func indented(line string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "   ")
}

// titleField: "Key: value", or "Key:" followed by an indented value.
func titleField(lines []string, i int, sceneStarts []string) bool {
	if i >= len(lines) || indented(lines[i]) {
		return false
	}
	key, value, ok := strings.Cut(lines[i], ":")
	if !ok || strings.TrimSpace(key) == "" || hasAnyPrefix(strings.ToUpper(lines[i]), sceneStarts) {
		return false
	}
	if strings.TrimSpace(value) != "" || key != strings.ToUpper(key) {
		return true
	}
	return i+1 < len(lines) && indented(lines[i+1]) && strings.TrimSpace(lines[i+1]) != ""
}

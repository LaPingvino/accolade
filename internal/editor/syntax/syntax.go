// Package syntax tells which Fountain element each line of a script is,
// for colouring it in Accolade's editor (docs/EDITOR_WIDGET.md, step 5).
// It follows the Fountain rules as lexington parses them, line by line.
package syntax

import (
	"strings"
	"unicode"

	"github.com/LaPingvino/lexington/fountain"
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
	Lyrics // ~ lines: sung
)

// SceneStarts are the English scene heading starts: Classify's.
var SceneStarts = []string{"INT.", "EXT.", "EST.", "INT./EXT.", "INT/EXT.", "I/E.", "INT ", "EXT ", "EST ", "INT/EXT ", "I/E "}

// Classify returns the kind of each line, with English scene headings.
// spaceOrMark: a space, or a direction mark (a line's left-to-right or
// right-to-left override, see editor.RTL), which does not change what
// element a line is.
func spaceOrMark(r rune) bool { return unicode.IsSpace(r) || r == '\u200E' || r == '\u200F' }

func Classify(lines []string) []Kind { return ClassifyWith(lines, SceneStarts) }

// ClassifyWith returns the kind of each line, with the scene headings of
// a language: what they start with ("INT.", "EN. ").
func ClassifyWith(lines []string, sceneStarts []string) []Kind {
	if sceneStarts == nil {
		sceneStarts = SceneStarts
	}
	return classify(lines, sceneStarts, nil, nil)
}

// ClassifyIn returns the kind of each line as a language's rules read
// it (Lexington's fountain.Dialect: its heading words and patterns, its
// cue and transition rules), with the script's cast for the list and
// repeat rules: the names in its /* Characters */ block (lines holds the
// whole text) and every cue before. sceneStarts are its scene words as
// the editor tells them apart (for the title page).
func ClassifyIn(lines []string, sceneStarts []string, d *fountain.Dialect) []Kind {
	if d == nil {
		return ClassifyWith(lines, sceneStarts)
	}
	if sceneStarts == nil {
		sceneStarts = SceneStarts
	}
	return classify(lines, sceneStarts, d, d.CastOfLines(lines))
}

// ClassifyCached is ClassifyIn with the script's cast kept between
// edits (fountain.CastCache: an editor's, per key).
func ClassifyCached(lines []string, sceneStarts []string, c *fountain.CastCache) []Kind {
	if sceneStarts == nil {
		sceneStarts = SceneStarts
	}
	return classify(lines, sceneStarts, c.Dialect(), c.Cast(lines))
}

// ClassifyTypingCached is ClassifyTypingIn with a cast cache.
func ClassifyTypingCached(lines []string, sceneStarts []string, c *fountain.CastCache, cursorLine int) []Kind {
	if cursorLine < 1 || cursorLine >= len(lines) || strings.TrimSpace(lines[cursorLine]) != "" {
		return ClassifyCached(lines, sceneStarts, c)
	}
	typed := append([]string(nil), lines...)
	typed[cursorLine] = "x"
	kinds := ClassifyCached(typed, sceneStarts, c)
	if k := kinds[cursorLine-1]; k == Character || k == Parenthetical {
		return kinds
	}
	return ClassifyCached(lines, sceneStarts, c)
}

func classify(lines []string, sceneStarts []string, d *fountain.Dialect, cast *fountain.Cast) []Kind {
	kinds := make([]Kind, len(lines))
	trimmed := func(i int) string {
		if i < 0 || i >= len(lines) {
			return ""
		}
		return strings.TrimFunc(lines[i], spaceOrMark)
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
		// (a speech's line of two spaces is no blank line)
		afterBlank := trimmed(i-1) == "" && (i == 0 || kinds[i-1] != Dialogue)
		switch {
		case t == "" && inDialogue && kinds[i-1] != Character && twoSpaces(lines[i]):
			// two spaces in a speech: its blank line, the speech goes on
			// (Fountain; not right after the name, which then has none)
			kinds[i] = Dialogue
			continue
		case t == "":
			kinds[i] = Empty
			inDialogue = false
			continue
		case strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]"):
			kinds[i] = Note
		case strings.HasPrefix(t, "~"):
			kinds[i] = Lyrics // in dialogue or on its own
			continue
		case strings.HasPrefix(t, "#"):
			kinds[i] = Section
		case strings.HasPrefix(t, "=") && strings.Trim(t, "=") == "" && len(t) >= 3:
			kinds[i] = PageBreak
		case strings.HasPrefix(t, "="):
			kinds[i] = Synopsis
		case strings.HasPrefix(t, ">") && strings.HasSuffix(t, "<"):
			kinds[i] = Centered
		case strings.HasPrefix(t, ">"), afterBlank && trimmed(i+1) == "" &&
			(d == nil && t == upper && strings.HasSuffix(t, "TO:") && t != strings.ToLower(t) || d != nil && t == upper && d.Transition(t)):
			// a transition: forced, or in capitals with blank lines around it
			kinds[i] = Transition
		case strings.HasPrefix(t, ".") && !strings.HasPrefix(t, "..") && len(t) > 1,
			afterBlank && (hasAnyPrefix(upper, sceneStarts) || d != nil && d.HasPatterns() && d.Heading(t)):
			// a heading: forced anywhere, by its word after a blank line
			kinds[i] = SceneHeading
		case inDialogue && strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")"):
			kinds[i] = Parenthetical
		case inDialogue:
			kinds[i] = Dialogue
		case trimmed(i+1) != "" && (strings.HasPrefix(t, "@") ||
			afterBlank && (d == nil && isCharacterCue(t) || d != nil && d.Cue(t, trimmed(i+1), cast))):
			kinds[i] = Character
			inDialogue = true
			if d != nil && d.Lang.Has("list") {
				cast.Add(t) // a later bare name is a cue (Lexington's list rule)
			}
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

// ClassifyTyping is ClassifyWith for a script being typed: an empty line
// with the cursor right after a character name or a parenthetical is where
// its dialogue goes (Fountain's rules wait for the dialogue's text).
func ClassifyTyping(lines []string, sceneStarts []string, cursorLine int) []Kind {
	return ClassifyTypingIn(lines, sceneStarts, nil, cursorLine)
}

// ClassifyTypingIn is ClassifyTyping by a language's rules (ClassifyIn).
func ClassifyTypingIn(lines []string, sceneStarts []string, d *fountain.Dialect, cursorLine int) []Kind {
	if cursorLine < 1 || cursorLine >= len(lines) || strings.TrimSpace(lines[cursorLine]) != "" {
		return ClassifyIn(lines, sceneStarts, d)
	}
	typed := append([]string(nil), lines...)
	typed[cursorLine] = "x"
	kinds := ClassifyIn(typed, sceneStarts, d)
	if k := kinds[cursorLine-1]; k == Character || k == Parenthetical {
		return kinds
	}
	return ClassifyIn(lines, sceneStarts, d)
}

// twoSpaces tells whether a line is two spaces or more and nothing else.
func twoSpaces(line string) bool {
	return len(line) >= 2 && strings.Trim(line, " ") == ""
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
	// capitals only, and at least one: scripts without capitals (Hebrew,
	// Arabic, Chinese) count as small letters, so their cues take an @
	if name == "" || name != strings.ToUpper(name) || strings.IndexFunc(name, unicode.IsUpper) < 0 ||
		strings.ContainsAny(name, ":!?") {
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

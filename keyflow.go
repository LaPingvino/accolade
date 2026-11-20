package main

import (
	"strings"
	"unicode"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The Final Draft keyboard flow (LOOP.md, Joop 2026-11-20), writing only
// ordinary Fountain:
//
//   - Tab on an empty line outside a speech starts a character cue: a
//     blank line above it if there was none, the letters typed in
//     capitals (the header bar says Character);
//   - Tab or Enter after a cue goes to its dialogue line;
//   - Tab in dialogue makes a parenthetical, "()" with the cursor inside;
//     Tab or Enter in it (or after it) goes back to the dialogue;
//   - Enter on an empty dialogue line leaves the speech: that line becomes
//     the blank one and the next is action;
//   - Enter after a scene heading writes the blank line Fountain wants
//     after it and goes on in action (a second Enter straight after is
//     taken as that blank line, for those who type it themselves).
//
// Completion (complete.go) and the source's suggestions (suggest.go) see
// Tab first: while the status bar shows what Tab would give, Tab gives
// it. vim's and helix's normal modes keep Tab and Enter for themselves
// (vim.go); in insert mode the flow is the same as here.

// keyFlow is the flow's state in a window.
type keyFlow struct {
	// cue is the line Tab made a cue of, plus one (0: none): what is
	// typed on it is in capitals, Enter and Tab go to its dialogue
	cue int
	// afterHeading is where Enter after a scene heading left the cursor,
	// and the text's revision then: a second Enter there adds nothing
	afterHeading struct {
		rev    uint64
		cursor int
		on     bool
	}
}

// flowOn reports whether the flow applies: a screenplay, auto-indent on
// (Preferences > Editor), one cursor and no selection.
func (w *MainWindow) flowOn() bool {
	if w == nil || w.markdown || w.textEditor == nil {
		return false
	}
	if w.settings != nil && !w.settings.GetBoolean("auto-indent") {
		return false
	}
	b := w.textEditor.Buffer()
	return !b.HasSelection() && !b.HasOthers() && !inBoneyard(b.Slice(0, b.Cursor()))
}

// inBoneyard reports whether the end of before is inside a /* … */
// block: the boneyard, or Accolade's Notes, Source, Adaptation and
// Production blocks, which are not screenplay.
func inBoneyard(before string) bool {
	open := strings.LastIndex(before, "/*")
	return open >= 0 && !strings.Contains(before[open:], "*/")
}

// cueHere reports whether the cursor is on the line Tab made a cue of.
func (w *MainWindow) cueHere() bool {
	if w == nil || w.flow.cue == 0 || w.textEditor == nil {
		return false
	}
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	return line+1 == w.flow.cue
}

// flowCursorMoved ends the cue when the cursor leaves its line.
func (w *MainWindow) flowCursorMoved() {
	if w.flow.cue != 0 && !w.cueHere() {
		w.flow.cue = 0
	}
}

// flowLines is the text's lines, the cursor's line and column.
func (w *MainWindow) flowLines() (lines []string, line, col int) {
	b := w.textEditor.Buffer()
	line, col = b.Position(b.Cursor())
	return strings.Split(b.Text(), "\n"), line, col
}

// inSpeech reports whether line i is in a speech (dialogue or a
// parenthetical), as Fountain reads it with something typed on it.
func (w *MainWindow) inSpeech(lines []string, i int) bool {
	typed := lines
	if strings.TrimSpace(lines[i]) == "" {
		typed = append([]string(nil), lines...)
		typed[i] = "x"
	}
	k := syntax.ClassifyWith(typed, w.sceneStarts())[i]
	return k == syntax.Dialogue || k == syntax.Parenthetical
}

// isParenthetical reports whether a trimmed line is one: "(…)".
func isParenthetical(trimmed string) bool {
	return strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")")
}

// asCue is a line typed as a cue made one: in capitals, or with Fountain's
// @ for a name in a script without capitals (Hebrew, Arabic, Chinese).
func asCue(line string) string {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "@") {
		return t
	}
	up := strings.ToUpper(t)
	if strings.IndexFunc(up, unicode.IsUpper) < 0 {
		return "@" + t
	}
	return up
}

// flowTab is Tab's step in the flow, after completion and suggestions
// had theirs; false leaves Tab to the editor (four spaces).
func (w *MainWindow) flowTab() bool {
	if !w.flowOn() {
		return false
	}
	e := w.textEditor
	lines, i, col := w.flowLines()
	line := lines[i]
	trimmed := strings.TrimSpace(line)
	b := e.Buffer()
	start, end := b.LineStart(i), b.LineEnd(i)
	atEnd := b.Cursor() == end

	switch {
	case trimmed == "" && w.inSpeech(lines, i):
		// a parenthetical, the cursor inside
		e.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				b.Replace(start, end, "()")
				b.SetCursor(start+1, false)
			})
		})
		return true

	case trimmed == "":
		// a cue: Fountain's blank line before it (and after it, before
		// action that follows, which would be read as its dialogue)
		before, after := "", ""
		defer w.updateCurrentElement()
		if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			before = "\n"
		}
		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
			after = "\n"
		}
		if before+after != "" || start != end { // nothing to write: no undo step
			e.Edit(func(b *buffer.Buffer) {
				b.Group(func() {
					b.Replace(start, end, before+after)
					b.SetCursor(start+len(before), false)
				})
			})
		}
		w.flow.cue = i + len(before) + 1
		w.setStatus("Character: type the name; Enter or Tab for the dialogue")
		return true

	case atEnd && w.cueHere():
		w.toDialogue(start, end, asCue(line))
		return true

	case atEnd && (i == 0 || strings.TrimSpace(lines[i-1]) == "") && !isSceneHeadingIn(strings.ToUpper(trimmed), w.sceneStarts()) &&
		(isCharacterCue(strings.TrimSuffix(trimmed, "^")) || strings.HasPrefix(trimmed, "@")):
		w.toDialogue(start, end, line)
		return true

	case !w.inSpeech(lines, i):
		return false

	case isParenthetical(trimmed) && (atEnd || closingRest(line, col)):
		w.toDialogue(start, end, line)
		return true

	case atEnd:
		// a parenthetical on the next line
		e.Edit(func(b *buffer.Buffer) {
			b.Group(func() {
				b.SetCursor(end, false)
				b.Insert("\n()")
				b.SetCursor(end+2, false)
			})
		})
		return true
	}
	return false
}

// closingRest reports whether what follows column col on line is the
// closing bracket alone (spaces aside): the cursor inside "(beat|)".
func closingRest(line string, col int) bool {
	r := []rune(line)
	return col <= len(r) && strings.TrimSpace(string(r[col:])) == ")"
}

// toDialogue puts line (from start to end) as it should be and goes to a
// new line after it: the dialogue's.
func (w *MainWindow) toDialogue(start, end int, line string) {
	w.flow.cue = 0
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if b.Slice(start, end) != line {
				b.Replace(start, end, line)
			}
			at := start + len([]rune(line))
			b.SetCursor(at, false)
			b.Insert("\n")
		})
	})
}

// flowEnter is Enter's step in the flow; false leaves Enter to
// completeLine's formatting.
func (w *MainWindow) flowEnter() bool {
	if !w.flowOn() {
		return false
	}
	e := w.textEditor
	b := e.Buffer()
	if a := w.flow.afterHeading; a.on {
		w.flow.afterHeading.on = false
		if a.rev == b.Rev() && a.cursor == b.Cursor() {
			return true // the blank line is there already
		}
	}
	lines, i, col := w.flowLines()
	line := lines[i]
	trimmed := strings.TrimSpace(line)
	start, end := b.LineStart(i), b.LineEnd(i)

	switch {
	case w.cueHere():
		w.toDialogue(start, end, asCue(line))
		return true
	case b.Cursor() != end && isParenthetical(trimmed) && closingRest(line, col) && w.inSpeech(lines, i):
		if trimmed == "()" { // nothing in it: the dialogue's line instead
			e.Edit(func(b *buffer.Buffer) {
				b.Group(func() {
					b.Replace(start, end, "")
					b.SetCursor(start, false)
				})
			})
			return true
		}
		w.toDialogue(start, end, line)
		return true
	}
	return false
}

// flowHeadingDone notes that Enter just completed a scene heading with
// the blank line after it, the cursor on the action's line.
func (w *MainWindow) flowHeadingDone() {
	if w == nil || w.textEditor == nil {
		return
	}
	b := w.textEditor.Buffer()
	w.flow.afterHeading.rev, w.flow.afterHeading.cursor, w.flow.afterHeading.on = b.Rev(), b.Cursor(), true
}

package main

import (
	"strings"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The Format menu: emphasis around the selection, the element of the
// current line, moving lines. Each change is one undo step.

// toggleEmphasis puts Fountain's emphasis marker (** bold, * italic,
// _ underline) around the selection, or takes it away if it is there
// already; without a selection it types the pair with the cursor between.
func toggleEmphasis(e *editor.ScriptEditor, marker string) {
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			start, end := b.Selection()
			n := len([]rune(marker))
			sel := b.Slice(start, end)
			if sel == "" {
				b.Insert(marker + marker)
				b.SetCursor(start+n, false)
				return
			}
			// the marker inside the selection, or just around it
			if len([]rune(sel)) >= 2*n && strings.HasPrefix(sel, marker) && strings.HasSuffix(sel, marker) &&
				!(marker == "*" && strings.HasPrefix(sel, "**") != strings.HasPrefix(sel, "***")) {
				inner := string([]rune(sel)[n : len([]rune(sel))-n])
				b.Replace(start, end, inner)
				b.Select(start, start+len([]rune(inner)))
				return
			}
			if start >= n && end+n <= b.Len() && b.Slice(start-n, start) == marker && b.Slice(end, end+n) == marker &&
				!(marker == "*" && start > n && b.Slice(start-n-1, start-n) == "*" && !(start > n+1 && b.Slice(start-n-2, start-n-1) == "*")) {
				b.Replace(end, end+n, "")
				b.Replace(start-n, start, "")
				b.Select(start-n, end-n)
				return
			}
			b.Replace(start, end, marker+sel+marker)
			b.Select(start+n, end+n)
		})
	})
}

// Elements the Format menu can make the current line.
const (
	elementScene         = "Scene Heading"
	elementAction        = "Action"
	elementCharacter     = "Character"
	elementParenthetical = "Parenthetical"
	elementDialogue      = "Dialogue"
	elementTransition    = "Transition"
	elementCentered      = "Centered"
)

// asElement is the line written as the element, with Accolade's indents
// and the Fountain markers that make other Fountain apps read it so.
func asElement(line, element string) string {
	bare := strings.TrimSpace(line)
	for _, m := range []string{".", "!", "@", ">"} { // forcing markers
		if strings.HasPrefix(bare, m) && !strings.HasPrefix(bare, "...") {
			bare = strings.TrimSpace(bare[len(m):])
			break
		}
	}
	bare = strings.TrimSpace(strings.TrimSuffix(bare, "<"))
	if element != elementParenthetical && len(bare) > 1 && strings.HasPrefix(bare, "(") && strings.HasSuffix(bare, ")") {
		bare = strings.TrimSpace(bare[1 : len(bare)-1])
	}
	indent := func(n int, s string) string { return strings.Repeat(" ", n) + s }
	switch element {
	case elementScene:
		s := strings.ToUpper(bare)
		if !isSceneHeading(s) {
			s = "." + s
		}
		return s
	case elementCharacter:
		s := strings.ToUpper(bare)
		if !isCharacterCue(s) {
			s = "@" + s
		}
		return indent(CharacterIndent, s)
	case elementParenthetical:
		if !strings.HasPrefix(bare, "(") {
			bare = "(" + bare
		}
		if !strings.HasSuffix(bare, ")") {
			bare += ")"
		}
		return indent(ParentheticalIndent, bare)
	case elementDialogue:
		return indent(DialogueIndent, bare)
	case elementTransition:
		s := strings.ToUpper(bare)
		if !strings.HasSuffix(s, "TO:") {
			s = "> " + s
		}
		return indent(TransitionIndent, s)
	case elementCentered:
		return "> " + bare + " <"
	}
	// action: forced if it would read as something else
	if isSceneHeading(bare) || isCharacterCue(bare) || isTransition(bare) {
		return "!" + bare
	}
	return bare
}

// setLineElement makes the line with the cursor the element.
func setLineElement(e *editor.ScriptEditor, element string) {
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			line, _ := b.Position(b.Cursor())
			start, end := b.LineStart(line), b.LineEnd(line)
			to := asElement(b.Slice(start, end), element)
			b.Replace(start, end, to)
			b.SetCursor(start+len([]rune(to)), false)
		})
	})
}

// moveLines moves the line with the cursor, or the selected lines, one
// line up (-1) or down (1), keeping them selected.
func moveLines(e *editor.ScriptEditor, dir int) {
	e.Edit(func(b *buffer.Buffer) {
		start, end := b.Selection()
		first, _ := b.Position(start)
		last, _ := b.Position(end)
		if end > start && last > first && end == b.LineStart(last) {
			last-- // a selection ending at a line's start leaves that line
		}
		if (dir < 0 && first == 0) || (dir > 0 && last >= b.LineCount()-1) {
			return
		}
		lines := strings.Split(b.Text(), "\n")
		block := append([]string(nil), lines[first:last+1]...)
		var moved []string
		if dir < 0 {
			moved = append(append(append(moved, lines[:first-1]...), block...), lines[first-1])
			moved = append(moved, lines[last+1:]...)
		} else {
			moved = append(append(append(moved, lines[:first]...), lines[last+1]), block...)
			moved = append(moved, lines[last+2:]...)
		}
		_, col := b.Position(b.Cursor())
		cursorLine, _ := b.Position(b.Cursor())
		b.Group(func() {
			b.Replace(0, b.Len(), strings.Join(moved, "\n"))
			if start == end {
				b.SetCursor(b.Offset(cursorLine+dir, col), false)
			} else {
				from := b.LineStart(first + dir)
				b.Select(from, b.LineEnd(last+dir))
			}
		})
	})
}

package main

import (
	"slices"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/structure"
)

// orgMoves are the moves evil-org's Alt keys make (M-J: Alt+Shift+j).
var orgMoves = map[string]structure.Op{
	"M-j": structure.MoveDown, "M-k": structure.MoveUp, "M-h": structure.Promote, "M-l": structure.Demote,
	"M-J": structure.IntoNext, "M-K": structure.IntoPrev, "M-H": structure.Barf, "M-L": structure.Slurp,
}

// evil-org's keys in vim mode, for the outline's headings: Tab and
// Shift+Tab cycle folds, gj gk the next and previous heading, gh the
// section a heading is in, gl the first heading in it; and the
// structural moves (internal/structure): Alt+j and Alt+k move the unit at
// the cursor (a section or scene with all it holds, a speech, a
// paragraph) past the next or previous one, Alt+h and Alt+l promote and
// demote it, Alt+J and Alt+K take it into the next or previous scene,
// Alt+H and Alt+L barf and slurp.
func (w *MainWindow) org(cmd string) bool {
	e := w.textEditor
	text := e.Text()
	items := w.outlineItems(text)
	line, _ := e.Buffer().Position(e.CursorOffset())
	i := w.scriptFolds.cursorItem(items)
	goTo := func(j int) bool {
		if j < 0 || j >= len(items) {
			w.setStatus("No heading there")
			return true
		}
		l := items[j].Line
		e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(l), false) })
		return true
	}
	switch cmd {
	case "tab":
		if i >= 0 && items[i].Line == line {
			w.scriptFolds.cycleAt(line)
		}
		return true
	case "stab":
		return w.scriptFolds.cycleAll()
	case "gj":
		return goTo(slices.IndexFunc(items, func(it outlineItem) bool { return it.Line > line }))
	case "gk":
		if i >= 0 && items[i].Line == line {
			i--
		}
		return goTo(i)
	case "gh": // the section around
		for j := i - 1; i >= 0 && j >= 0; j-- {
			if !items[j].Scene && items[j].Level < items[i].Level {
				return goTo(j)
			}
		}
		return goTo(-1)
	case "gs": // the cursor's scene's guessed link: its menu (linkup.go)
		w.linkGuessHere()
		return true
	case "gl": // the first heading inside
		if i >= 0 && i+1 < len(items) && items[i+1].Level > items[i].Level {
			return goTo(i + 1)
		}
		return goTo(-1)
	}
	if op, ok := orgMoves[cmd]; ok { // the structural moves (structure_window.go)
		w.structDo(op)
		return true
	}
	return false
}

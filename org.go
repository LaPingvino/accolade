package main

import (
	"slices"
	"strings"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// evil-org's keys in vim mode, for the outline's headings: Tab and
// Shift+Tab cycle folds, gj gk the next and previous heading, gh the
// section a heading is in, gl the first heading in it; Alt+j and Alt+k
// move a scene or section (with all it holds) past the next or previous
// one, Alt+h and Alt+l promote and demote a section.
func (w *MainWindow) org(cmd string) bool {
	e := w.textEditor
	text := e.Text()
	items := outlineOf(text, w.sceneStarts())
	line, _ := e.Buffer().Position(e.CursorOffset())
	i := w.cursorItem(items)
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
			w.cycleAt(line)
		}
		return true
	case "stab":
		return w.cycleAll()
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
	case "gl": // the first heading inside
		if i >= 0 && i+1 < len(items) && items[i+1].Level > items[i].Level {
			return goTo(i + 1)
		}
		return goTo(-1)
	case "M-j", "M-k":
		if i < 0 {
			return goTo(-1)
		}
		lines := strings.Count(text, "\n") + 1
		next := func(j int) int { // the item after j and all it holds
			_, end := outlineExtent(items, j, lines)
			k := slices.IndexFunc(items, func(it outlineItem) bool { return it.Line >= end })
			if k < 0 {
				return len(items)
			}
			return k
		}
		var to int
		if cmd == "M-j" {
			sib := next(i)
			if sib >= len(items) || items[sib].Level != items[i].Level {
				w.setStatus("Nothing to move past")
				return true
			}
			to = next(sib)
		} else {
			to = -1
			for j := i - 1; j >= 0 && items[j].Level >= items[i].Level; j-- {
				if items[j].Level == items[i].Level {
					to = j
					break
				}
			}
			if to < 0 {
				w.setStatus("Nothing to move past")
				return true
			}
		}
		heading := items[i]
		w.moveOutlineItem(i, to)
		for _, it := range outlineOf(e.Text(), w.sceneStarts()) { // the cursor goes with it
			if it.Text == heading.Text && it.Level == heading.Level {
				l := it.Line
				e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(l), false) })
				break
			}
		}
		return true
	case "M-h", "M-l":
		if i < 0 || items[i].Scene {
			w.setStatus("Only a section has a level to change")
			return true
		}
		level := items[i].Level - 1
		if cmd == "M-l" {
			level = items[i].Level + 1
		}
		if level < 1 {
			w.setStatus("A section's level is at least one #")
			return true
		}
		l := items[i].Line
		all := strings.Split(text, "\n")
		old := all[l]
		indent := old[:len(old)-len(strings.TrimLeft(old, " \t"))]
		all[l] = indent + strings.Repeat("#", level) + " " + items[i].Text
		w.replaceText(text, strings.Join(all, "\n"))
		e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(l), false) })
		return true
	}
	return false
}

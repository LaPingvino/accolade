package main

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

// Scene and section colors, as Beat has them: a note on the heading line
// ("INT. BARN - DAY [[red]]", "# ACT ONE [[color blue]]", "[[#ff8800]]")
// gives the scene or section its color; lex.NoteColor reads it the way
// Beat does. The outline and the cards show it, and the color picker
// writes it as Beat writes it: "[[red]]" at the end of the line.

// beatColorNames are the colors the picker offers, in Beat's order.
var beatColorNames = []string{"red", "blue", "green", "pink", "magenta", "gray", "purple", "cyan", "teal", "yellow", "orange", "brown"}

// noteRGB is the color a lex.NoteColor color stands for: one of Beat's
// names, in Beat's own RGB (lex.ColorRGB, from Beat's BeatColors.m), or
// "#rrggbb". Anything else (an unknown name, "#zzzzzz") is no color.
func noteRGB(name string) (color.NRGBA, bool) {
	c, ok := lex.ColorRGB(name)
	return color.NRGBA{c[0], c[1], c[2], 255}, ok
}

// withColor is a heading or section line with the color c, written as
// Beat writes it: "[[c]]" at the end of the line, in place of the color
// notes it had (wherever they were, each with the white space in front of
// it); "" or "none" takes them out. Other notes stay where they are. On
// a line with a "[[" that is never closed, the color goes before that.
func withColor(line, c string) string {
	isBlank := func(r rune) bool { return r != '\n' && unicode.IsSpace(r) }
	var b strings.Builder
	rest, unclosed := line, ""
	for {
		open := strings.Index(rest, "[[")
		if open < 0 {
			break
		}
		end := strings.Index(rest[open+2:], "]]")
		if end < 0 { // never closed: no note, and the color goes before it
			rest, unclosed = rest[:open], rest[open:]
			break
		}
		end += open + 4
		if lex.NoteColor(rest[open:end]) == "" { // another kind of note
			b.WriteString(rest[:end])
		} else {
			b.WriteString(strings.TrimRightFunc(rest[:open], isBlank))
			if b.Len() == 0 { // at the start of the line: the space after it goes
				rest = strings.TrimLeftFunc(rest[end:], isBlank)
				continue
			}
		}
		rest = rest[end:]
	}
	b.WriteString(rest)
	out := b.String()
	if c = strings.ToLower(strings.TrimSpace(c)); c == "" || c == "none" {
		return out + unclosed
	}
	if unclosed != "" { // "[[c]]" after it would be read as part of it
		return strings.TrimRightFunc(out, isBlank) + " [[" + c + "]] " + unclosed
	}
	return strings.TrimRightFunc(out, isBlank) + " [[" + c + "]]"
}

// setItemColor gives item i of the script's outline the color c ("" or
// "none": none).
func setItemColor(text string, starts []string, i int, c string) string {
	items := outlineOf(text, starts)
	if i < 0 || i >= len(items) {
		return text
	}
	lines := strings.Split(text, "\n")
	lines[items[i].Line] = withColor(lines[items[i].Line], c)
	return strings.Join(lines, "\n")
}

// swatchSize is the size of a color's mark in the outline and the cards.
const swatchSize = 10

// newSwatch is a mark that shows a scene's or section's color (see
// showSwatch).
func newSwatch() *canvas.Rectangle {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(swatchSize, swatchSize))
	r.CornerRadius = swatchSize / 2
	return r
}

// showSwatch shows the color name (lex.NoteColor's) on r: nothing for no
// color, or one it does not know.
func showSwatch(r *canvas.Rectangle, name string) {
	if c, ok := noteRGB(name); ok {
		r.FillColor = c
	} else {
		r.FillColor = color.Transparent
	}
	r.Refresh()
}

// colorName is how the color name reads, for a screen reader: "" for no
// color, or one Accolade does not know.
func colorName(name string) string {
	if _, ok := noteRGB(name); !ok {
		return ""
	}
	if name == "grey" {
		return "gray"
	}
	return name // a name, or "#rrggbb"
}

// The color picker: Beat's colors and None, for a scene or section. The
// outline's and the cards' secondary tap (right click) opens it for theirs;
// Format > Color… (Ctrl+Shift+L) and the editing row's Color button for
// the one the cursor is in, or the card whose synopsis is being written.
// Picking writes the note in the script as any edit does: one undo step.

// swatchIcon is a color's mark for the picker's menu.
func swatchIcon(name string) fyne.Resource {
	c, _ := noteRGB(name)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" fill="#%02x%02x%02x"/></svg>`, c.R, c.G, c.B)
	return fyne.NewStaticResource("swatch-"+name, []byte(svg))
}

// colorMenu is the picker for item it: None and Beat's colors, its own
// checked; then (if not nil) runs after picking one.
func (w *MainWindow) colorMenu(it outlineItem, then func()) *fyne.Menu {
	pick := func(c string) func() {
		return func() {
			w.setOutlineColor(it, c)
			if then != nil {
				then()
			}
		}
	}
	none := fyne.NewMenuItem("None", pick("none"))
	none.Checked = it.Color == ""
	items := []*fyne.MenuItem{none, fyne.NewMenuItemSeparator()}
	for _, name := range beatColorNames {
		item := fyne.NewMenuItem(strings.ToUpper(name[:1])+name[1:], pick(name))
		item.Icon = swatchIcon(name)
		item.Checked = it.Color == name || (name == "gray" && it.Color == "grey")
		items = append(items, item)
	}
	return fyne.NewMenu("", items...)
}

// pickColor shows the picker for item it at pos (absolute); then as for
// colorMenu.
func (w *MainWindow) pickColor(it outlineItem, pos fyne.Position, then func()) {
	if w.markdown { // colours are Fountain notes
		w.setStatus("Colors are for a screenplay's scenes and sections")
		return
	}
	m := widget.NewPopUpMenu(w.colorMenu(it, then), w.fyneWindow.Canvas())
	fyne.SetAccessibleLabel(m, "Color of "+it.Text)
	m.ShowAtPosition(pos)
}

// pickColorHere shows the picker for the card whose synopsis is being
// written, or the scene or section the editor's cursor is in, under at
// (nil: at the cursor or the card).
func (w *MainWindow) pickColorHere(at fyne.CanvasObject) {
	if !w.cardsShown() && w.scriptAway("Scene or Section Color") { // another middle tab in front (middle.go)
		return
	}
	if w.markdown {
		w.pickColor(outlineItem{}, fyne.Position{}, nil)
		return
	}
	d := fyne.CurrentApp().Driver()
	below := func(o fyne.CanvasObject) fyne.Position {
		return d.AbsolutePositionForObject(o).Add(fyne.NewPos(0, o.Size().Height))
	}
	if s, ok := w.fyneWindow.Canvas().Focused().(*synopsisEntry); ok && w.cardsShown() && s.i < len(w.cards.items) {
		if at == nil {
			at = s
		}
		w.pickColor(w.cards.items[s.i], below(at), nil)
		return
	}
	if w.cardsShown() { // the editor is not shown: no cursor to go by
		w.setStatus("Right-click a card to color it, or write in its synopsis first")
		return
	}
	e := w.textEditor
	items := w.outlineItems(e.Text())
	i := w.scriptFolds.cursorItem(items)
	if i < 0 {
		w.setStatus("No scene or section here to color")
		return
	}
	var pos fyne.Position
	if at != nil {
		pos = below(at)
	} else {
		pos = d.AbsolutePositionForObject(e).Add(e.CursorPosition()).Add(fyne.NewPos(0, theme.TextSize()*1.5))
	}
	w.pickColor(items[i], pos, func() { w.fyneWindow.Canvas().Focus(e) }) // back to writing
}

// setOutlineColor gives item it, if it is still in the script, the color
// c ("none": none), as one undo step.
func (w *MainWindow) setOutlineColor(it outlineItem, c string) {
	text := w.textEditor.Text()
	items := w.outlineItems(text)
	i := slices.IndexFunc(items, func(o outlineItem) bool { return o.Line == it.Line && o.Text == it.Text })
	if i < 0 { // moved (an edit together, say): by its heading
		i = slices.IndexFunc(items, func(o outlineItem) bool { return o.Text == it.Text && o.Level == it.Level })
	}
	if i < 0 {
		w.setStatus("That heading is gone")
		return
	}
	if nw := setItemColor(text, w.sceneStarts(), i, c); nw != text {
		w.replaceText(text, nw)
		// the cards wait while a synopsis is written: shown again, the
		// synopsis keeps the focus
		cv := w.fyneWindow.Canvas()
		if s, ok := cv.Focused().(*synopsisEntry); ok && w.cardsShown() {
			cv.Unfocus()
			w.cards.update(nw)
			if k := w.cards.card(s.i); k != nil {
				cv.Focus(k.syn)
			}
		}
	}
	if c == "none" {
		w.setStatus("No color: " + it.Text)
	} else {
		w.setStatus("Color " + c + ": " + it.Text)
	}
}

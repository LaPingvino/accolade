package main

import (
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The card view (View > Cards, or the outline's Cards button): the outline
// laid out as index cards in the editor's place, as Highland has them. A
// card shows a scene's heading, its synopsis and how it starts, or a
// section's title; tapped, it opens the scene in the editor; dragged, it
// moves its scene or section where it is dropped.

// cardSize is an index card's size.
var cardSize = fyne.NewSize(240, 150)

type cardsView struct {
	w     *MainWindow
	items []outlineItem
	grid  *fyne.Container
	box   fyne.CanvasObject
}

func newCardsView(w *MainWindow) *cardsView {
	c := &cardsView{w: w, grid: container.NewGridWrap(cardSize)}
	back := widget.NewButtonWithIcon("Back to the Script", theme.DocumentIcon(), w.toggleCards)
	back.Importance = widget.LowImportance
	heading := widget.NewLabelWithStyle("Cards", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("Tap a card to open its scene; drag it to move it.")
	hint.Importance = widget.LowImportance
	c.box = container.NewBorder(container.NewHBox(heading, hint, back), nil, nil, nil, container.NewVScroll(c.grid))
	return c
}

// update shows the script's outline as cards.
func (c *cardsView) update(text string) {
	c.items = outlineOf(text, c.w.sceneStarts())
	lines := strings.Split(text, "\n")
	c.grid.RemoveAll()
	for i, it := range c.items {
		c.grid.Add(newCard(c, i, it, cardPreview(lines, c.items, i)))
	}
	c.grid.Refresh()
}

// cardPreview is how an item's text starts: its first lines after the
// heading, the synopsis and notes left out.
func cardPreview(lines []string, items []outlineItem, i int) string {
	end := len(lines)
	if i+1 < len(items) {
		end = items[i+1].Line
	}
	var words []string
	for _, l := range lines[items[i].Line+1 : end] {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "=") || strings.HasPrefix(t, "[[") {
			continue
		}
		words = append(words, t)
		if len(strings.Join(words, " ")) > 140 {
			break
		}
	}
	p := strings.Join(words, " ")
	if r := []rune(p); len(r) > 140 {
		p = string(r[:140]) + "…"
	}
	return p
}

// dropIndex is where card i dragged by d goes: before the card at the
// place it is dropped (len: the end), -1 if it stays.
func (c *cardsView) dropIndex(i int, d fyne.Delta) int {
	cell := cardSize.Add(fyne.NewSize(theme.Padding(), theme.Padding()))
	cols := max(1, int((c.grid.Size().Width+theme.Padding())/cell.Width))
	col := i%cols + int(math.Round(float64(d.DX/cell.Width)))
	row := i/cols + int(math.Round(float64(d.DY/cell.Height)))
	col = max(0, min(col, cols-1))
	j := max(0, min(row*cols+col, len(c.items)-1))
	switch {
	case j == i:
		return -1
	case j > i:
		return j + 1 // after the card it is dropped on
	}
	return j
}

// card is one index card.
type card struct {
	widget.BaseWidget
	c    *cardsView
	i    int
	item outlineItem
	body fyne.CanvasObject
	drag fyne.Delta
}

func newCard(c *cardsView, i int, it outlineItem, preview string) *card {
	title := widget.NewLabel(it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Wrapping = fyne.TextWrapWord
	title.Truncation = fyne.TextTruncateEllipsis
	var parts []fyne.CanvasObject
	if it.Scene {
		if it.Synopsis != "" {
			syn := widget.NewLabel(it.Synopsis)
			syn.Wrapping = fyne.TextWrapWord
			syn.TextStyle = fyne.TextStyle{Italic: true}
			parts = append(parts, syn)
		}
		if preview != "" {
			p := widget.NewLabel(preview)
			p.Wrapping = fyne.TextWrapWord
			p.SizeName = theme.SizeNameCaptionText
			parts = append(parts, p)
		}
	} else {
		title.Importance = widget.HighImportance
		title.Alignment = fyne.TextAlignCenter
		kind := widget.NewLabel(strings.Repeat("#", it.Level) + " section")
		kind.Alignment = fyne.TextAlignCenter
		kind.Importance = widget.LowImportance
		parts = append(parts, kind)
	}
	bg := canvasCard()
	k := &card{c: c, i: i, item: it,
		body: container.NewStack(bg, container.NewPadded(container.NewBorder(title, nil, nil, nil, container.NewVBox(parts...))))}
	k.ExtendBaseWidget(k)
	fyne.SetAccessibleLabel(k, it.Text)
	return k
}

func (k *card) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(k.body) }

// Tapped opens the card's scene in the editor.
func (k *card) Tapped(*fyne.PointEvent) {
	w := k.c.w
	w.toggleCards()
	if w.outline != nil {
		w.outline.goTo(k.item.Line)
	} else {
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(k.item.Line))
		w.fyneWindow.Canvas().Focus(w.textEditor)
	}
}

func (k *card) Dragged(e *fyne.DragEvent) {
	k.drag = fyne.Delta{DX: k.drag.DX + e.Dragged.DX, DY: k.drag.DY + e.Dragged.DY}
	k.Move(k.Position().Add(e.Dragged)) // it follows the pointer
	if to := k.c.dropIndex(k.i, k.drag); to >= 0 && to < len(k.c.items) {
		k.c.w.setStatus("Move before " + k.c.items[to].Text)
	} else if to >= 0 {
		k.c.w.setStatus("Move to the end")
	}
}

func (k *card) DragEnd() {
	to := k.c.dropIndex(k.i, k.drag)
	k.drag = fyne.Delta{}
	if to >= 0 {
		k.c.w.moveOutlineItem(k.i, to) // the cards follow the text
	} else {
		k.c.grid.Refresh()
	}
}

// toggleCards shows the card view in the editor's place, or the editor.
func (w *MainWindow) toggleCards() {
	w.cardsVisible = !w.cardsVisible
	if w.cardsVisible {
		if w.cards == nil {
			w.cards = newCardsView(w)
		}
		w.cards.update(w.textEditor.Text())
	}
	w.relayout()
	if !w.cardsVisible {
		w.fyneWindow.Canvas().Focus(w.textEditor)
	}
}

// canvasCard is a card's paper: the input background with a border.
func canvasCard() fyne.CanvasObject {
	r := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	r.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.StrokeWidth = 1
	r.CornerRadius = theme.InputRadiusSize()
	return r
}

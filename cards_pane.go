package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The Cards pane (View > Cards, LOOP.md 2026-11-21): the cards as a
// pane of the dock, beside the script, in one compact column: each
// scene's and section's heading with its colour, and its synopsis to
// write. A card dragged moves its scene, as the outline's rows do; a tap
// goes to it. The card of the cursor's scene is lit; with the source in
// the main editor (swap.go), the cards of the scenes linking to the
// passage there. Swapped with the main area, the pane is the full Cards
// view (cards.go).

type cardsPane struct {
	w     *MainWindow
	items []outlineItem
	lit   map[int]bool
	seen  linesSeen
	list  *widget.List
	box   fyne.CanvasObject
	row   float32 // a card's height
}

func (w *MainWindow) cardsPane() *cardsPane {
	if w.cardsDock == nil {
		w.cardsDock = newCardsPane(w)
	}
	return w.cardsDock
}

// paneSynopsis is a card's synopsis field in the pane.
type paneSynopsis struct {
	widget.Entry
	id int // the card's item
}

func newCardsPane(w *MainWindow) *cardsPane {
	c := &cardsPane{w: w, lit: map[int]bool{}}
	c.list = widget.NewList(
		func() int { return len(c.items) },
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			bg.CornerRadius = theme.InputRadiusSize()
			bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
			bg.StrokeWidth = 1
			title := widget.NewLabel("")
			title.Truncation = fyne.TextTruncateEllipsis
			syn := &paneSynopsis{}
			syn.ExtendBaseWidget(syn)
			syn.SetPlaceHolder("Synopsis…")
			syn.OnChanged = func(s string) { c.setSynopsis(syn.id, s) }
			swatch := container.NewCenter(newSwatch())
			body := container.NewBorder(nil, nil, swatch, nil, container.NewVBox(title, syn))
			return newDragRow(c, container.NewStack(bg, container.NewPadded(body)))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(c.items) {
				return
			}
			c.show(o.(*dragRow), id)
		},
	)
	fyne.SetAccessibleLabel(c.list, "Cards")
	c.list.OnSelected = func(id widget.ListItemID) {
		c.list.UnselectAll()
		c.tapRow(id)
	}
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(260, 0))
	full := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() { w.swapWithMain("cards") })
	full.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(full, "The cards in the main area")
	hint := widget.NewLabel("Drag a card to move its scene")
	hint.SizeName = theme.SizeNameCaptionText
	hint.Importance = widget.LowImportance
	hint.Truncation = fyne.TextTruncateEllipsis
	c.box = container.NewStack(width, container.NewBorder(container.NewBorder(nil, nil, nil, full, hint), nil, nil, nil, c.list))
	return c
}

// rowParts are a card row's parts.
func cardRowParts(r *dragRow) (bg *canvas.Rectangle, title *widget.Label, syn *paneSynopsis, swatch *canvas.Rectangle) {
	stack := r.content
	bg = stack.Objects[0].(*canvas.Rectangle)
	body := stack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
	box := body.Objects[0].(*fyne.Container)
	title, syn = box.Objects[0].(*widget.Label), box.Objects[1].(*paneSynopsis)
	swatch = body.Objects[1].(*fyne.Container).Objects[0].(*canvas.Rectangle)
	return
}

// show fills row r with card id.
func (c *cardsPane) show(r *dragRow, id int) {
	it := c.items[id]
	r.id = id
	bg, title, syn, swatch := cardRowParts(r)
	title.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	if !it.Scene {
		title.Importance = widget.HighImportance
	} else {
		title.Importance = widget.MediumImportance
	}
	title.Refresh()
	showSwatch(swatch, it.Color)
	typing := c.w.fyneWindow.Canvas().Focused() == fyne.Focusable(syn) && syn.id == id
	syn.id = id
	if !typing && syn.Text != it.Synopsis { // being typed: as it is (the script keeps one line)
		syn.SetText(it.Synopsis)
	}
	fyne.SetAccessibleLabel(syn, "Synopsis of "+it.Text)
	bg.FillColor = theme.Color(theme.ColorNameInputBackground)
	if c.lit[id] {
		bg.FillColor = tint(theme.ColorNamePrimary, 0x40)
	}
	bg.Refresh()
}

// update reads the cards from the script's text again.
func (c *cardsPane) update(text string) {
	c.items = c.w.outlineItems(text)
	if c.row == 0 {
		c.row = c.list.CreateItem().MinSize().Height
	}
	for id := range c.items {
		c.list.SetItemHeight(id, c.row)
	}
	c.follow()
	c.list.Refresh()
}

// follow lights the cards that go with the main cursor.
func (c *cardsPane) follow() {
	lit, _, ok := c.w.litScenes(c.items, &c.seen)
	if !ok || sameSet(lit, c.lit) {
		return
	}
	c.lit = lit
	c.list.Refresh()
	for i := range c.items {
		if lit[i] {
			c.list.ScrollTo(i)
			break
		}
	}
}

// setSynopsis writes card id's synopsis into the script.
func (c *cardsPane) setSynopsis(id int, s string) {
	w := c.w
	if id < 0 || id >= len(c.items) || s == c.items[id].Synopsis {
		return
	}
	old := w.textEditor.Text()
	if nw := setSynopsis(old, w.sceneStarts(), id, s); nw != old {
		w.replaceText(old, nw)
	}
}

// The pane as a dragList (outline.go).
func (c *cardsPane) rowItems() []outlineItem { return c.items }
func (c *cardsPane) rowHeight(int) float32   { return c.row }
func (c *cardsPane) listWindow() *MainWindow { return c.w }
func (c *cardsPane) moveRow(from, to int)    { c.w.moveOutlineItem(from, to) }
func (c *cardsPane) tapRow(id int) {
	if id >= 0 && id < len(c.items) {
		c.w.goToScript(c.items[id].Line)
	}
}

// shown reports whether the pane is in sight.
func (c *cardsPane) shown() bool { return c.w.paneInSight("cards") }

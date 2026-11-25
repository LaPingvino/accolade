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
// the middle (middle.go), the cards of the scenes linking to the
// passage there. In the middle, the cards are their full view (cards.go),
// which View > Cards opens.

type cardsPane struct {
	w        *MainWindow
	of       docKey // the script it shows (state.go)
	items    []outlineItem
	lit      map[int]bool
	seen     linesSeen
	list     *widget.List
	box      fyne.CanvasObject
	row      float32               // a card's height
	rowB     float32               // a card's with its beats' mark
	beats    map[int][]int         // each card's beats without a scene, by its line (notes_marks.go)
	unplaced *fyne.Container       // the beats before the first heading, above the cards
	dots     map[int][]color.Color // who is in which card, by index (collab_cursors.go)
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
			dots := container.NewCenter(newPeerDots()) // the co-writers in it (collab_cursors.go)
			dots.Hide()
			beats := newLinkLabel(nil) // its beats without a scene: a tap steps through them
			beats.Importance = widget.WarningImportance
			beats.Truncation = fyne.TextTruncateEllipsis
			beats.Hide()
			body := container.NewBorder(nil, nil, swatch, dots, container.NewVBox(title, syn, beats))
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
	full := widget.NewButtonWithIcon("", theme.ViewFullScreenIcon(), func() { w.openMiddle("cards") })
	full.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(full, "The cards in the middle")
	hint := widget.NewLabel("Drag a card to move its scene")
	hint.SizeName = theme.SizeNameCaptionText
	hint.Importance = widget.LowImportance
	hint.Truncation = fyne.TextTruncateEllipsis
	c.unplaced = container.NewVBox()
	c.unplaced.Hide()
	c.box = container.NewStack(width, container.NewBorder(container.NewVBox(container.NewBorder(nil, nil, nil, full, hint), c.unplaced), nil, nil, nil, c.list))
	return c
}

// rowParts are a card row's parts.
func cardRowParts(r *dragRow) (bg *canvas.Rectangle, title *widget.Label, syn *paneSynopsis, swatch *canvas.Rectangle, beats *linkLabel) {
	stack := r.content
	bg = stack.Objects[0].(*canvas.Rectangle)
	body := stack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
	box := body.Objects[0].(*fyne.Container)
	title, syn, beats = box.Objects[0].(*widget.Label), box.Objects[1].(*paneSynopsis), box.Objects[2].(*linkLabel)
	swatch = body.Objects[1].(*fyne.Container).Objects[0].(*canvas.Rectangle)
	return
}

// show fills row r with card id.
func (c *cardsPane) show(r *dragRow, id int) {
	it := c.items[id]
	r.id = id
	bg, title, syn, swatch, beats := cardRowParts(r)
	indent := strings.Repeat("  ", max(0, it.Level-1))
	title.SetText(indent + it.Text)
	if lines := c.beats[it.Line]; len(lines) > 0 {
		beats.SetText(indent + beatsMark(len(lines)))
		beats.onTap = func() { c.w.stepToBeat(lines) }
		beats.Show()
	} else {
		beats.Hide()
	}
	r.content.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*fyne.Container).Refresh() // laid out again: the mark shown or not
	title.TextStyle = fyne.TextStyle{Bold: true}
	if !it.Scene {
		title.Importance = widget.HighImportance
	} else {
		title.Importance = widget.MediumImportance
	}
	title.Refresh()
	showSwatch(swatch, it.Color)
	showRowDots(r.content.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container).Objects[2].(*fyne.Container), c.dots[id])
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
	c.of = c.w.scriptKey()
	c.items = c.w.outlineItems(text)
	c.dots = c.w.peersInScenes(c.items) // who is in which (collab_cursors.go)
	all := scriptBeats(text, c.items)
	c.beats = beatsByItem(all)
	c.showUnplaced(unplacedBeats(strings.Split(text, "\n"), all))
	if c.row == 0 {
		r := c.list.CreateItem().(*dragRow)
		c.row = r.MinSize().Height
		_, _, _, _, b := cardRowParts(r)
		b.SetText(beatsMark(1))
		b.Show()
		c.rowB = r.MinSize().Height
	}
	for id := range c.items {
		c.list.SetItemHeight(id, c.rowHeight(id))
	}
	c.follow()
	c.list.Refresh()
}

// showUnplaced shows the beats before the first heading above the cards:
// Unplaced, their mark, and each beat, a tap going to it.
func (c *cardsPane) showUnplaced(u []looseSynopsis) {
	c.unplaced.Objects = nil
	if len(u) == 0 {
		c.unplaced.Hide()
		return
	}
	title := widget.NewLabel("Unplaced")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Importance = widget.HighImportance
	c.unplaced.Add(title)
	c.unplaced.Add(beatsLink(c.w, unplacedLines(u)))
	for _, l := range u {
		line := l.Line
		b := newLinkLabel(func() { c.w.goToScript(line) })
		b.SizeName = theme.SizeNameText
		b.TextStyle = fyne.TextStyle{Italic: true}
		b.Truncation = fyne.TextTruncateEllipsis
		b.SetText("= " + l.Text)
		fyne.SetAccessibleLabel(b, "Unplaced beat: "+l.Text)
		c.unplaced.Add(b)
	}
	c.unplaced.Show()
	c.unplaced.Refresh()
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
func (c *cardsPane) colours() bool           { return true }
func (c *cardsPane) rowHeight(id int) float32 {
	if id < len(c.items) && len(c.beats[c.items[id].Line]) > 0 {
		return c.rowB
	}
	return c.row
}
func (c *cardsPane) listWindow() *MainWindow { return c.w }
func (c *cardsPane) moveRow(from, to int)    { c.w.moveOutlineItem(from, to) }
func (c *cardsPane) tapRow(id int) {
	if id >= 0 && id < len(c.items) {
		c.w.goToScript(c.items[id].Line)
	}
}

// shown reports whether the pane is in sight.
func (c *cardsPane) shown() bool { return c.w.paneInSight("cards") }

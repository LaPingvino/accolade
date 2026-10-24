package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The card view (View > Cards, or the outline's Cards button): the outline
// laid out as index cards in the editor's place, as Highland has them,
// grouped under the sections. A card shows a scene's heading, its synopsis
// (written on the card, for planning) and how it starts; tapped, it opens
// the scene in the editor; dragged, it moves its scene where it is dropped,
// as a section's header moves the section. A card at the end of each group
// adds a scene there.

// cardSize is an index card's size.
var cardSize = fyne.NewSize(240, 170)

// cardsView shows the outline's scenes as cards, grouped under their
// sections; each group ends with a card that adds a scene to it.
type cardsView struct {
	w       *MainWindow
	items   []outlineItem
	list    *fyne.Container // the groups, one under the other
	targets []dropTarget    // where a dragged card or section can go
	box     fyne.CanvasObject
}

// dropTarget is a card or header something can be dropped on: before
// item before, or after it when dropped on the right half of a card.
type dropTarget struct {
	obj    fyne.CanvasObject
	before int
	halves bool
}

func newCardsView(w *MainWindow) *cardsView {
	c := &cardsView{w: w, list: container.NewVBox()}
	back := widget.NewButtonWithIcon("Back to the Script", theme.DocumentIcon(), w.toggleCards)
	back.Importance = widget.LowImportance
	heading := widget.NewLabelWithStyle("Cards", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("Tap a card to open its scene; drag it to move it; write its synopsis on it.")
	hint.Importance = widget.LowImportance
	hint.Truncation = fyne.TextTruncateEllipsis
	c.box = container.NewBorder(container.NewBorder(nil, nil, heading, back, hint), nil, nil, nil, container.NewVScroll(c.list))
	return c
}

// update shows the script's outline as cards. While a synopsis is being
// written on a card, the cards stay (they would lose the typing).
func (c *cardsView) update(text string) {
	c.items = outlineOf(text, c.w.sceneStarts())
	if _, typing := c.w.fyneWindow.Canvas().Focused().(*synopsisEntry); typing {
		return
	}
	lines := strings.Split(text, "\n")
	c.list.RemoveAll()
	c.targets = nil
	var grid *fyne.Container
	group := func(header fyne.CanvasObject) {
		grid = container.NewGridWrap(cardSize)
		if header != nil {
			c.list.Add(header)
		}
		c.list.Add(grid)
	}
	end := func(at int) { // the group's add card: a new scene before item at
		add := newAddCard(c, at)
		grid.Add(add)
		c.targets = append(c.targets, dropTarget{obj: add, before: at})
	}
	for i, it := range c.items {
		if !it.Scene {
			if grid != nil {
				end(i)
			}
			h := newSectionHeader(c, i, it)
			group(h)
			c.targets = append(c.targets, dropTarget{obj: h, before: i + 1})
			continue
		}
		if grid == nil {
			group(nil)
		}
		k := newCard(c, i, it, cardPreview(lines, c.items, i))
		grid.Add(k)
		c.targets = append(c.targets, dropTarget{obj: k, before: i, halves: true})
	}
	if grid == nil {
		group(nil)
	}
	end(len(c.items))
	c.list.Refresh()
}

// dropAt is where what is dragged (moving) goes when dropped at pos
// (absolute): before which item (len: the end), or -1 when not on another
// card or header.
func (c *cardsView) dropAt(moving fyne.CanvasObject, pos fyne.Position) int {
	d := fyne.CurrentApp().Driver()
	for _, t := range c.targets {
		if t.obj == moving {
			continue
		}
		at, size := d.AbsolutePositionForObject(t.obj), t.obj.Size()
		if pos.X < at.X || pos.Y < at.Y || pos.X >= at.X+size.Width || pos.Y >= at.Y+size.Height {
			continue
		}
		if t.halves && pos.X >= at.X+size.Width/2 {
			return t.before + 1
		}
		return t.before
	}
	return -1
}

// dropped moves item i, dragged (as moving) to pos, there.
func (c *cardsView) dropped(i int, moving fyne.CanvasObject, pos fyne.Position) {
	to := c.dropAt(moving, pos)
	old := c.w.textEditor.Text()
	if to >= 0 {
		c.w.moveOutlineItem(i, to) // the cards follow the text
	}
	if c.w.textEditor.Text() == old {
		c.update(old) // nothing moved: back in its place
	}
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
		if len(strings.Join(words, " ")) > 100 {
			break
		}
	}
	p := strings.Join(words, " ")
	if r := []rune(p); len(r) > 100 {
		p = string(r[:100]) + "…"
	}
	return p
}

// dragged is what cards and section headers share: they follow the
// pointer, and are dropped where its middle is.
type dragged struct {
	c    *cardsView
	i    int
	self fyne.CanvasObject
}

func (d *dragged) Dragged(e *fyne.DragEvent) {
	d.self.Move(d.self.Position().Add(e.Dragged))
	if to := d.c.dropAt(d.self, d.middle()); to >= 0 && to < len(d.c.items) {
		d.c.w.setStatus("Move before " + d.c.items[to].Text)
	} else if to >= 0 {
		d.c.w.setStatus("Move to the end")
	}
}

func (d *dragged) DragEnd() { d.c.dropped(d.i, d.self, d.middle()) }

func (d *dragged) middle() fyne.Position {
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(d.self)
	return at.Add(fyne.NewPos(d.self.Size().Width/2, d.self.Size().Height/2))
}

// card is a scene's index card: its heading, its synopsis to write, and
// how it starts.
type card struct {
	widget.BaseWidget
	dragged
	body fyne.CanvasObject
}

// synopsisEntry is where a card's synopsis is written.
type synopsisEntry struct{ widget.Entry }

func newCard(c *cardsView, i int, it outlineItem, preview string) *card {
	title := widget.NewLabel(it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Truncation = fyne.TextTruncateEllipsis
	syn := &synopsisEntry{}
	syn.MultiLine = true
	syn.Wrapping = fyne.TextWrapWord
	syn.ExtendBaseWidget(syn)
	syn.SetPlaceHolder("Synopsis…")
	syn.SetText(it.Synopsis)
	syn.OnChanged = func(s string) {
		w := c.w
		old := w.textEditor.Text()
		if nw := setSynopsis(old, w.sceneStarts(), i, s); nw != old {
			w.replaceText(old, nw)
		}
	}
	fyne.SetAccessibleLabel(syn, "Synopsis of "+it.Text)
	var bottom fyne.CanvasObject
	if preview != "" {
		p := widget.NewLabel(preview)
		p.Wrapping = fyne.TextWrapWord
		p.Truncation = fyne.TextTruncateEllipsis
		p.SizeName = theme.SizeNameCaptionText
		p.Importance = widget.LowImportance
		bottom = p
	}
	k := &card{body: container.NewStack(canvasCard(), container.NewPadded(container.NewBorder(title, bottom, nil, nil, syn)))}
	k.dragged = dragged{c: c, i: i, self: k}
	k.ExtendBaseWidget(k)
	fyne.SetAccessibleLabel(k, it.Text)
	return k
}

func (k *card) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(k.body) }

// Tapped opens the card's scene in the editor.
func (k *card) Tapped(*fyne.PointEvent) { k.c.open(k.i) }

// open shows item i in the editor.
func (c *cardsView) open(i int) {
	w := c.w
	line := c.items[i].Line
	w.toggleCards()
	if w.outline != nil {
		w.outline.goTo(line)
	} else {
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(line))
		w.fyneWindow.Canvas().Focus(w.textEditor)
	}
}

// sectionHeader heads a section's cards; tapped, it opens the section,
// dragged, it moves the section with all it holds.
type sectionHeader struct {
	widget.BaseWidget
	dragged
	body fyne.CanvasObject
}

func newSectionHeader(c *cardsView, i int, it outlineItem) *sectionHeader {
	title := widget.NewLabel(it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Importance = widget.HighImportance
	title.Truncation = fyne.TextTruncateEllipsis
	rows := []fyne.CanvasObject{widget.NewSeparator(), title}
	if it.Synopsis != "" {
		syn := widget.NewLabel(it.Synopsis)
		syn.TextStyle = fyne.TextStyle{Italic: true}
		syn.Importance = widget.LowImportance
		syn.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, syn)
	}
	var indent fyne.CanvasObject // nested sections a little in
	if it.Level > 1 {
		indent = container.NewGridWrap(fyne.NewSize(float32(it.Level-1)*4*theme.Padding(), 1))
	}
	h := &sectionHeader{body: container.NewBorder(nil, nil, indent, nil, container.NewVBox(rows...))}
	h.dragged = dragged{c: c, i: i, self: h}
	h.ExtendBaseWidget(h)
	fyne.SetAccessibleLabel(h, "Section "+it.Text)
	return h
}

func (h *sectionHeader) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(h.body) }

func (h *sectionHeader) Tapped(*fyne.PointEvent) { h.c.open(h.i) }

// addCard ends a group of cards: tapped, it asks for a new scene's heading
// and synopsis and puts the scene there.
type addCard struct {
	widget.BaseWidget
	c    *cardsView
	at   int
	body fyne.CanvasObject
}

func newAddCard(c *cardsView, at int) *addCard {
	label := widget.NewLabelWithStyle("New scene", fyne.TextAlignCenter, fyne.TextStyle{})
	label.Importance = widget.LowImportance
	paper := canvas.NewRectangle(color.Transparent)
	paper.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	paper.StrokeWidth = 1
	paper.CornerRadius = theme.InputRadiusSize()
	a := &addCard{c: c, at: at, body: container.NewStack(paper, container.NewCenter(container.NewVBox(widget.NewIcon(theme.ContentAddIcon()), label)))}
	a.ExtendBaseWidget(a)
	fyne.SetAccessibleLabel(a, "New scene")
	return a
}

func (a *addCard) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(a.body) }

func (a *addCard) Tapped(*fyne.PointEvent) {
	w := a.c.w
	heading := widget.NewEntry()
	heading.SetPlaceHolder(newHeadingHint(w))
	syn := widget.NewMultiLineEntry()
	syn.Wrapping = fyne.TextWrapWord
	syn.SetPlaceHolder("What happens")
	form := dialog.NewForm("New Scene", "Add", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Heading", heading),
		widget.NewFormItem("Synopsis", syn),
	}, func(ok bool) {
		if !ok || strings.TrimSpace(heading.Text) == "" {
			return
		}
		old := w.textEditor.Text()
		nw, _ := addScene(old, w.sceneStarts(), a.at, heading.Text, syn.Text)
		w.replaceText(old, nw)
	}, w.fyneWindow)
	form.Resize(fyne.NewSize(420, 260))
	form.Show()
	w.fyneWindow.Canvas().Focus(heading)
}

// newHeadingHint is an example heading in the script's language.
func newHeadingHint(w *MainWindow) string {
	if in, _, _, ok := placeWords(sceneHeaders(w.language)); ok {
		return strings.TrimSpace(in) + " PLACE - DAY"
	}
	return "INT. PLACE - DAY"
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

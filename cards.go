package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The card view (View > Cards, or the outline's Cards button): the outline
// laid out as index cards in the editor's place, as Highland has them,
// grouped under the sections. A card shows a scene's heading, its synopsis
// (written on the card, for planning) and how it starts; tapped, it opens
// the scene in the editor; dragged, it moves its scene where it is dropped,
// as a section's header moves the section. A card at the end of each group
// adds a scene there.

// cardSize is an index card's size; foldedCardSize its size with the
// synopses folded (Fold Synopses: its heading and how it starts).
var (
	cardSize       = fyne.NewSize(240, 170)
	foldedCardSize = fyne.NewSize(240, 92)
)

// cardsView shows the outline's scenes as cards, grouped under their
// sections (or by colour, character or tag: the Cards tab's Group), each
// section's group ending with a card that adds a scene to it; or, as its
// other arrangement, the timeline (timeline.go).
type cardsView struct {
	w       *MainWindow
	of      docKey // the script it shows (state.go)
	items   []outlineItem
	list    *fyne.Container // the groups, one under the other
	targets []dropTarget    // where a dragged card or section can go
	box     fyne.CanvasObject
	// body holds the grid's scroll or the timeline; tools are the
	// header's buttons, which the Ribbon look's Cards tab has instead
	body  *fyne.Container
	grid  *container.Scroll
	tl    *timelineView
	tools *fyne.Container
	hint  *widget.Label
	// shown is the look as last laid out; grouped: by other than section
	// (a view: the cards do not move by dragging then); lit is the
	// current card's item
	shown   cardsLook
	grouped bool
	lit     int
}

// dropTarget is a card or header something can be dropped on: before
// item before, or after it when dropped on the right half of a card.
type dropTarget struct {
	obj    fyne.CanvasObject
	before int
	halves bool
}

func newCardsView(w *MainWindow) *cardsView {
	c := &cardsView{w: w, list: container.NewVBox(), lit: -1}
	back := widget.NewButtonWithIcon("Back to the Script", theme.DocumentIcon(), func() { w.showMiddle("") })
	back.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(back, "Back to the Script")
	heading := widget.NewLabelWithStyle("Cards", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	c.hint = widget.NewLabel("")
	c.hint.Importance = widget.LowImportance
	c.hint.Truncation = fyne.TextTruncateEllipsis
	// Swap… and Move… (structure_picker.go): a phone has no right-click
	swap := widget.NewButton("Swap…", func() { w.showStructPicker(false, c.at()) })
	move := widget.NewButton("Move…", func() { w.showStructPicker(true, c.at()) })
	swap.Importance, move.Importance = widget.LowImportance, widget.LowImportance
	arrange := widget.NewButtonWithIcon("", timelineIcon, nil)
	arrange.Importance = widget.LowImportance
	arrange.OnTapped = func() {
		l := w.cardsLook()
		l.timeline = !l.timeline
		w.setCardsLook(l)
	}
	fyne.SetAccessibleLabel(arrange, "Grid or Timeline")
	c.tools = container.NewHBox(arrange, swap, move, back)
	c.grid = container.NewVScroll(c.list)
	c.tl = newTimelineView(w)
	c.body = container.NewStack(c.grid)
	c.box = container.NewBorder(container.NewBorder(nil, nil, heading, c.tools, c.hint), nil, nil, nil, c.body)
	c.followLook()
	return c
}

// followLook shows the header's buttons in the Menus look, not in the
// Ribbon look (its Cards tab has them).
func (c *cardsView) followLook() {
	setVisible(c.tools, !c.w.ribbonShown())
	if b, ok := c.tools.Objects[3].(*widget.Button); ok { // a phone's: short
		label := "Back to the Script"
		if c.w.narrow {
			label = "Script"
		}
		if b.Text != label {
			b.SetText(label)
		}
	}
	if b, ok := c.tools.Objects[0].(*widget.Button); ok { // the other arrangement's icon
		ic := timelineIcon
		if c.w.cardsLook().timeline {
			ic = theme.GridIcon()
		}
		if b.Icon != ic {
			b.SetIcon(ic)
		}
	}
}

// timeline is the timeline view.
func (c *cardsView) timeline() *timelineView { return c.tl }

// timelineFocused reports whether the timeline has the keys.
func (c *cardsView) timelineFocused() bool {
	return c.w.fyneWindow.Canvas().Focused() == fyne.Focusable(c.tl)
}

// relook shows the cards in the script's look (a Cards tab choice).
func (c *cardsView) relook() {
	c.followLook()
	if c.w.textEditor != nil {
		c.update(c.w.textEditor.Text())
	}
	if c.shown.timeline && c.w.cardsShown() {
		c.w.fyneWindow.Canvas().Focus(c.tl) // the arrows go along it
	}
}

// at is the heading line of the current card (the one whose synopsis
// is being written, else the cursor's), for Swap… and Move….
func (c *cardsView) at() int {
	if s, ok := c.w.fyneWindow.Canvas().Focused().(*synopsisEntry); ok && s.i < len(c.items) {
		return c.items[s.i].Line
	}
	return -1
}

// update shows the script's outline as cards (or its timeline). While a
// synopsis is being written on a card, the cards stay (they would lose
// the typing).
func (c *cardsView) update(text string) {
	c.of = c.w.scriptKey()
	c.items = c.w.outlineItems(text)
	if _, typing := c.w.fyneWindow.Canvas().Focused().(*synopsisEntry); typing {
		return
	}
	l := c.w.cardsLook()
	if l.timeline != c.shown.timeline || len(c.body.Objects) == 0 {
		if l.timeline {
			c.body.Objects = []fyne.CanvasObject{c.tl}
		} else {
			c.body.Objects = []fyne.CanvasObject{c.grid}
		}
		c.body.Refresh()
	}
	c.shown = l
	c.followLook()
	if l.timeline {
		c.hint.SetText(timelineHint(c.w.narrow))
		c.tl.update()
		return
	}
	c.grouped = l.group != ""
	switch {
	case c.grouped:
		c.hint.SetText("Grouped " + strings.ToLower(groupLabel(l.group)) + ": a view; Make It the Order puts the script so")
	default:
		c.hint.SetText("Tap a card to open its scene; drag it to move it; write its synopsis on it.")
	}
	lines := strings.Split(text, "\n")
	c.list.RemoveAll()
	c.targets = nil
	size := cardSize
	folded := c.w.settings != nil && c.w.settings.GetBoolean(foldSynopsesPref)
	if folded {
		size = foldedCardSize
	}
	var grid *fyne.Container
	group := func(header fyne.CanvasObject) {
		grid = container.NewGridWrap(size)
		if header != nil {
			c.list.Add(header)
		}
		c.list.Add(grid)
	}
	addCard := func(i int) {
		k := newCard(c, i, c.items[i], cardPreview(lines, c.items, i), folded)
		grid.Add(k)
		c.targets = append(c.targets, dropTarget{obj: k, before: i, halves: true})
	}
	addLoose := func(i int) { // its loose synopses, a card each after it
		if c.grouped {
			return
		}
		for _, l := range c.items[i].Loose {
			grid.Add(newSynopsisCard(c, l))
		}
	}
	if c.grouped {
		for _, g := range cardGroups(c.w.storyNow(), l.group) {
			group(newGroupHeader(g))
			for _, i := range g.items {
				addCard(i)
			}
		}
		c.targets = nil // a view: nothing to drop on
		c.follow()
		c.showDots()
		c.list.Refresh()
		return
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
			addLoose(i) // its own, before its scenes
			continue
		}
		if grid == nil {
			group(nil)
		}
		addCard(i)
		addLoose(i)
	}
	if grid == nil {
		group(nil)
	}
	end(len(c.items))
	c.lit = -1
	c.follow()
	c.showDots()
	c.list.Refresh()
}

// groupLabel is a Group choice's label: "By Color".
func groupLabel(value string) string {
	for _, g := range cardsGroups {
		if g[0] == value {
			return g[1]
		}
	}
	return ""
}

// follow lights the current card (the script's cursor's scene), and the
// timeline's.
func (c *cardsView) follow() {
	if c.shown.timeline {
		c.tl.follow()
		return
	}
	at := c.w.cardsCurrent(c.items)
	for _, g := range c.list.Objects {
		grid, ok := g.(*fyne.Container)
		if !ok {
			continue
		}
		for _, o := range grid.Objects {
			if k, ok := o.(*card); ok {
				k.light(k.i == at)
			}
		}
	}
	c.lit = at
}

// showDots puts a dot on each card whose scene a co-writer is in.
func (c *cardsView) showDots() {
	if c.shown.timeline {
		return
	}
	dots := c.w.peersInScenes(c.items)
	for _, g := range c.list.Objects {
		if grid, ok := g.(*fyne.Container); ok {
			for _, o := range grid.Objects {
				if k, ok := o.(*card); ok {
					showRowDots(k.dots, dots[k.i])
				}
			}
		}
	}
}

// newGroupHeader heads a group of the grouped cards: its name and colour.
func newGroupHeader(g cardGroup) fyne.CanvasObject {
	title := widget.NewLabel(g.title)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Importance = widget.HighImportance
	title.Truncation = fyne.TextTruncateEllipsis
	return container.NewVBox(widget.NewSeparator(), withSwatch(title, g.color))
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
	if c.grouped {
		c.w.setStatus("Grouped, the cards are a view: Make It the Order, or group by section to drag them")
		c.update(c.w.textEditor.Text())
		return
	}
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

// dropBoard is where dragged things are dropped: the cards, the
// stripboard.
type dropBoard interface {
	// dragging says where what is dragged (item i, as self) would go,
	// its middle at pos (absolute).
	dragging(i int, self fyne.CanvasObject, pos fyne.Position)
	// dropped puts it there.
	dropped(i int, self fyne.CanvasObject, pos fyne.Position)
}

// dragged is what cards, section headers and strips share: they follow
// the pointer, and are dropped where its middle is.
type dragged struct {
	board dropBoard
	i     int
	self  fyne.CanvasObject
}

func (d *dragged) Dragged(e *fyne.DragEvent) {
	d.self.Move(d.self.Position().Add(e.Dragged))
	d.board.dragging(d.i, d.self, d.middle())
}

func (d *dragged) DragEnd() { d.board.dropped(d.i, d.self, d.middle()) }

// dragging says where a card or section would go.
func (c *cardsView) dragging(_ int, self fyne.CanvasObject, pos fyne.Position) {
	if to := c.dropAt(self, pos); to >= 0 && to < len(c.items) {
		c.w.setStatus("Move before " + c.items[to].Text)
	} else if to >= 0 {
		c.w.setStatus("Move to the end")
	}
}

func (d *dragged) middle() fyne.Position {
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(d.self)
	return at.Add(fyne.NewPos(d.self.Size().Width/2, d.self.Size().Height/2))
}

// card is a scene's index card: its heading, its synopsis to write, and
// how it starts.
type card struct {
	widget.BaseWidget
	dragged
	c     *cardsView
	body  fyne.CanvasObject
	syn   *synopsisEntry
	paper *canvas.Rectangle
	lit   bool
	dots  *fyne.Container // the co-writers in its scene (collab_cursors.go)
}

// synopsisEntry is where a card's synopsis is written.
type synopsisEntry struct {
	widget.Entry
	i int // the card's item
}

func newCard(c *cardsView, i int, it outlineItem, preview string, folded bool) *card {
	title := widget.NewLabel(it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Truncation = fyne.TextTruncateEllipsis
	syn := &synopsisEntry{i: i}
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
	paper := canvasCard()
	dots := container.NewCenter(newPeerDots())
	dots.Hide()
	head := container.NewBorder(nil, nil, nil, dots, withSwatch(title, it.Color))
	var inside fyne.CanvasObject = container.NewBorder(head, bottom, nil, nil, syn)
	if folded { // its heading and how it starts
		if bottom == nil {
			bottom = layout.NewSpacer()
		}
		inside = container.NewBorder(head, nil, nil, nil, bottom)
	}
	k := &card{syn: syn, paper: paper, dots: dots, body: container.NewStack(paper, container.NewPadded(inside))}
	k.c = c
	k.dragged = dragged{board: c, i: i, self: k}
	k.ExtendBaseWidget(k)
	fyne.SetAccessibleLabel(k, withColorName(it.Text, it.Color))
	return k
}

func (k *card) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(k.body) }

// light shows whether the card is the current one (the cursor's scene).
func (k *card) light(on bool) {
	if k.lit == on {
		return
	}
	k.lit = on
	k.paper.FillColor = theme.Color(theme.ColorNameInputBackground)
	k.paper.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	k.paper.StrokeWidth = 1
	if on {
		k.paper.FillColor = tint(theme.ColorNamePrimary, 0x30)
		k.paper.StrokeColor = theme.Color(theme.ColorNamePrimary)
		k.paper.StrokeWidth = 2
	}
	k.paper.Refresh()
}

// Tapped opens the card's scene in the editor.
func (k *card) Tapped(*fyne.PointEvent) { k.c.open(k.i) }

// TappedSecondary shows the scene's menu: its color, Swap…, Move….
func (k *card) TappedSecondary(e *fyne.PointEvent) {
	k.c.w.itemMenu(k.c.items[k.i], e.AbsolutePosition)
}

// card is item i's card, nil if it has none.
func (c *cardsView) card(i int) *card {
	for _, g := range c.list.Objects {
		if grid, ok := g.(*fyne.Container); ok {
			for _, o := range grid.Objects {
				if k, ok := o.(*card); ok && k.i == i {
					return k
				}
			}
		}
	}
	return nil
}

// open shows item i in the editor.
func (c *cardsView) open(i int) {
	w := c.w
	line := c.items[i].Line
	w.goToScript(line) // the script's tab in front
}

// synopsisCard is a loose synopsis's card (a beat, a moment planned: a
// "= …" paragraph on its own): its text, in italics; tapped, the script
// at its line, where it is written, moved (Alt+Up and Alt+Down) and grown
// into a scene (New Scene on it).
type synopsisCard struct {
	widget.BaseWidget
	c    *cardsView
	line int
	body fyne.CanvasObject
}

func newSynopsisCard(c *cardsView, l looseSynopsis) *synopsisCard {
	text := widget.NewLabel(l.Text)
	text.TextStyle = fyne.TextStyle{Italic: true}
	text.Wrapping = fyne.TextWrapWord
	text.Truncation = fyne.TextTruncateEllipsis
	mark := widget.NewLabel("=")
	mark.Importance = widget.LowImportance
	paper := canvas.NewRectangle(color.Transparent) // not a scene yet: no paper, an outline
	paper.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	paper.StrokeWidth = 1
	paper.CornerRadius = theme.InputRadiusSize()
	k := &synopsisCard{c: c, line: l.Line, body: container.NewStack(paper, container.NewPadded(container.NewBorder(mark, nil, nil, nil, text)))}
	k.ExtendBaseWidget(k)
	fyne.SetAccessibleLabel(k, "Synopsis: "+l.Text)
	return k
}

func (k *synopsisCard) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(k.body) }

func (k *synopsisCard) Tapped(*fyne.PointEvent) { k.c.w.goToScript(k.line) }

// sectionHeader heads a section's cards; tapped, it opens the section,
// dragged, it moves the section with all it holds.
type sectionHeader struct {
	widget.BaseWidget
	dragged
	c    *cardsView
	body fyne.CanvasObject
}

func newSectionHeader(c *cardsView, i int, it outlineItem) *sectionHeader {
	title := widget.NewLabel(it.Text)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Importance = widget.HighImportance
	title.Truncation = fyne.TextTruncateEllipsis
	rows := []fyne.CanvasObject{widget.NewSeparator(), withSwatch(title, it.Color)}
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
	h.c = c
	h.dragged = dragged{board: c, i: i, self: h}
	h.ExtendBaseWidget(h)
	fyne.SetAccessibleLabel(h, withColorName("Section "+it.Text, it.Color))
	return h
}

func (h *sectionHeader) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(h.body) }

func (h *sectionHeader) Tapped(*fyne.PointEvent) { h.c.open(h.i) }

// TappedSecondary shows the section's menu: its color, Swap…, Move….
func (h *sectionHeader) TappedSecondary(e *fyne.PointEvent) {
	h.c.w.itemMenu(h.c.items[h.i], e.AbsolutePosition)
}

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

func (a *addCard) Tapped(*fyne.PointEvent) { a.c.w.askNewScene(a.at) }

// askNewScene asks for a new scene's heading and synopsis and puts the
// scene before item at (len: at the end), as one undo step.
func (w *MainWindow) askNewScene(at int) {
	// New Scene's form (new_scene.go), with the card's synopsis
	var dlg dialog.Dialog
	content, first := w.newSceneContentWith(func() { dlg.Hide() }, func(heading, syn string) {
		old := w.textEditor.Text()
		nw, line := addScene(old, w.sceneStarts(), at, heading, syn)
		w.holdingStep(func() {
			w.replaceText(old, nw)
			w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) }) // the current card now
		})
		w.keepInStepSaying("New scene: " + heading) // the scenes after it, in the same undo step
		if w.cards != nil && w.cardsShown() {
			w.cards.follow()
		}
	})
	dlg = dialog.NewCustomWithoutButtons("New Scene", content, w.fyneWindow)
	dlg.Resize(fitDialog(w.fyneWindow, 520, 680))
	dlg.Show()
	if first != nil {
		w.fyneWindow.Canvas().Focus(first)
	}
}

// toggleCards is the outline's Cards button: the cards' middle tab in
// front (opened if it is not, out of the dock if it was there), or the
// script's if it is in front.
func (w *MainWindow) toggleCards() {
	if w.markdown {
		w.setStatus("Cards are a screenplay's scenes")
		return
	}
	if w.cardsShown() {
		w.showMiddle("")
		return
	}
	w.openMiddle("cards")
}

// canvasCard is a card's paper: the input background with a border.
func canvasCard() *canvas.Rectangle {
	r := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	r.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.StrokeWidth = 1
	r.CornerRadius = theme.InputRadiusSize()
	return r
}

// withSwatch is a card's or section header's title with its color's mark
// before it, if it has a color.
func withSwatch(title fyne.CanvasObject, name string) fyne.CanvasObject {
	if _, ok := noteRGB(name); !ok {
		return title
	}
	r := newSwatch()
	showSwatch(r, name)
	return container.NewBorder(nil, nil, container.NewCenter(r), nil, title)
}

// withColorName is a screen reader's name for something with a color:
// "INT. BARN - DAY, red".
func withColorName(label, name string) string {
	if n := colorName(name); n != "" {
		return label + ", " + n
	}
	return label
}

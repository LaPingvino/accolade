package main

import (
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The timeline (LOOP.md, TIMELINE): the Cards tab's second arrangement.
// The scenes left to right in story order (on a phone top to bottom) as
// blocks as long as the scene is (in eighths of a page, story_model.go),
// in the scene's colour, the acts and sections as bands above them; lanes
// by character or tag below, a mark where they are in a scene and a thin
// line where they are gone. It answers the writing's questions: where am I
// (the cursor's scene is lit), what is the rhythm (lengths side by side),
// who has been gone (the lanes), which scenes are still empty or only a
// synopsis (an outline, no fill). Tap opens the scene in the script; the
// arrows go from scene to scene (the script's cursor goes with them);
// Enter opens; Alt+Left/Right (Alt+Up/Down on a phone) and dragging move
// the scene through the structure engine, one undo step; Ctrl+wheel, + and
// - zoom. Nothing is kept here but the zoom: every change is the text's.

// timeline rows' sizes.
const (
	tlBand     = 22  // a band's row (a section level)
	tlScene    = 76  // the scenes' row
	tlLane     = 20  // a lane's row
	tlLabels   = 120 // the lanes' names' column
	tlPad      = 8
	tlMinBlock = 4 // the narrowest a block is drawn
)

// timelineView is the timeline, a widget with the keys.
type timelineView struct {
	widget.BaseWidget
	w      *MainWindow
	m      storyModel
	lands  map[int]beatPlace // where the sections land, by outline item (beat_place.go)
	zoom   float32           // 1: the story fits the width (a phone: an eighth is 10 px)
	scroll *zoomScroll
	area   *fyne.Container // the drawing
	labels *fyne.Container // the lanes' names (left to right)
	focus  *canvas.Rectangle
	box    *fyne.Container
	blocks []*timelineBlock
	lit    int // the current scene (-1: none)
	// as last drawn: the width, the layout's direction, the lanes
	drawnW    float32
	vertical  bool
	lanes     string
	scale     float32 // pixels an eighth
	focused   bool
	dragStart fyne.Position
}

func newTimelineView(w *MainWindow) *timelineView {
	t := &timelineView{w: w, zoom: 1, lit: -1}
	t.area = container.NewWithoutLayout()
	t.labels = container.NewWithoutLayout()
	t.scroll = newZoomScroll(t.area, func(dy float32) {
		if dy > 0 {
			t.zoomBy(1)
		} else if dy < 0 {
			t.zoomBy(-1)
		}
	})
	t.focus = canvas.NewRectangle(color.Transparent)
	t.focus.StrokeWidth = 2
	t.focus.CornerRadius = theme.InputRadiusSize()
	t.box = container.NewStack(container.NewBorder(nil, nil, t.labels, nil, t.scroll), t.focus)
	t.ExtendBaseWidget(t)
	fyne.SetAccessibleLabel(t, "Timeline: the scenes in story order; Left and Right go from scene to scene, Enter opens one")
	return t
}

func (t *timelineView) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.box) }

// Resize draws the timeline again for a new width (fitted to it).
func (t *timelineView) Resize(s fyne.Size) {
	t.BaseWidget.Resize(s)
	if s.Width > 0 && math.Abs(float64(s.Width-t.drawnW)) > 1 && len(t.m.scenes) > 0 {
		t.draw()
	}
}

// update reads the story again and draws it.
func (t *timelineView) update() {
	t.m = t.w.storyNow()
	t.lands = map[int]beatPlace{} // by band item
	if l := t.w.landings(t.m.items, true); l != nil {
		for i, it := range t.m.items {
			if p, ok := l[it.Line]; ok {
				t.lands[i] = p
			}
		}
	}
	t.w.measureStoryLater()
	t.draw()
}

// zoomBy zooms in (dir 1) or out (-1).
func (t *timelineView) zoomBy(dir int) {
	z := t.zoom * float32(math.Pow(1.25, float64(dir)))
	z = min(max(z, 0.5), 16)
	if z == t.zoom {
		return
	}
	t.zoom = z
	t.draw()
	t.showLit()
}

// draw lays the timeline out for the story and the room it has.
func (t *timelineView) draw() {
	w := t.w
	t.vertical = w.narrow
	t.lanes = w.cardsLook().lanes
	size := t.Size()
	t.drawnW = size.Width
	if size.Width <= 0 {
		size = fyne.NewSize(900, 500)
	}
	t.area.RemoveAll()
	t.labels.RemoveAll()
	t.blocks = nil
	m := &t.m
	lanes := t.laneList()
	levels := 0
	for _, b := range m.bands {
		levels = max(levels, b.level)
	}
	total := float32(max(1, m.total))
	text := theme.Color(theme.ColorNameForeground)
	faint := theme.Color(theme.ColorNameDisabled)
	labelW := float32(0)
	if len(lanes) > 0 && !t.vertical {
		labelW = tlLabels
	}
	if !t.vertical {
		room := max(200, size.Width-labelW-2*tlPad-2*theme.Padding()) // (the border's gap beside the names)
		t.scale = room / total * t.zoom
		pos := make([]float32, len(m.scenes)+1)
		for k, s := range m.scenes {
			pos[k+1] = pos[k] + float32(s.eighths)*t.scale
		}
		x := func(k int) float32 { return tlPad + pos[k] }
		y := float32(tlPad)
		for lv := 1; lv <= levels; lv++ {
			for _, b := range m.bands {
				if b.level != lv || b.first > b.last {
					continue
				}
				t.addBand(b, x(b.first), y, x(b.last+1)-x(b.first), tlBand-3)
			}
			y += tlBand
		}
		for k := range m.scenes {
			bw := max(tlMinBlock, x(k+1)-x(k)-2)
			t.addBlock(k, fyne.NewPos(x(k), y), fyne.NewSize(bw, tlScene))
		}
		y += tlScene + 6
		for _, ln := range lanes {
			name := canvas.NewText(ln.name, text)
			name.TextSize = theme.CaptionTextSize()
			name.Move(fyne.NewPos(tlPad, y+3))
			t.labels.Add(name)
			t.addLane(ln, func(k int) (fyne.Position, fyne.Size) {
				return fyne.NewPos(x(k), y+4), fyne.NewSize(max(tlMinBlock, x(k+1)-x(k)-2), tlLane-8)
			}, faint)
			y += tlLane
		}
		t.labels.Objects = append([]fyne.CanvasObject{minBox(labelW, y)}, t.labels.Objects...)
		t.area.Add(minBox(x(len(m.scenes))+tlPad, y+tlPad))
	} else {
		// a phone: down the screen, a band's title a row over its scenes,
		// the lanes as narrow columns at the right
		t.scale = 10 * t.zoom
		laneW := float32(18)
		width := max(160, size.Width-2*tlPad)
		blockW := width - float32(len(lanes))*laneW - 4
		ys := make([]float32, len(m.scenes)+1)
		y := float32(tlPad)
		if len(lanes) > 0 { // the lanes' names, a letter or three
			for j, ln := range lanes {
				name := canvas.NewText(shortName(ln.name), text)
				name.TextSize = theme.CaptionTextSize() * 0.85
				name.Move(fyne.NewPos(tlPad+blockW+4+float32(j)*laneW, y))
				t.area.Add(name)
			}
			y += tlBand
		}
		bandAt := map[int][]storyBand{}
		for _, b := range m.bands {
			if b.first <= b.last {
				bandAt[b.first] = append(bandAt[b.first], b)
			}
		}
		for k, s := range m.scenes {
			for _, b := range bandAt[k] {
				indent := float32(b.level-1) * 10
				t.addBand(b, tlPad+indent, y, blockW-indent, tlBand-3)
				y += tlBand
			}
			ys[k] = y
			h := max(float32(14), float32(s.eighths)*t.scale)
			t.addBlock(k, fyne.NewPos(tlPad, y), fyne.NewSize(blockW, h-2))
			y += h
		}
		ys[len(m.scenes)] = y
		for j, ln := range lanes {
			lx := tlPad + blockW + 4 + float32(j)*laneW
			t.addLane(ln, func(k int) (fyne.Position, fyne.Size) {
				return fyne.NewPos(lx+3, ys[k]), fyne.NewSize(laneW-6, max(tlMinBlock, ys[k+1]-ys[k]-2))
			}, faint)
		}
		t.area.Add(minBox(width+tlPad, y+tlPad))
	}
	if len(m.scenes) == 0 {
		none := canvas.NewText("No scenes yet: a heading (INT. PLACE - DAY) starts one", faint)
		none.Move(fyne.NewPos(tlPad, tlPad))
		t.area.Add(none)
	}
	t.labels.Refresh()
	t.area.Refresh()
	t.scroll.Direction = container.ScrollHorizontalOnly
	if t.vertical {
		t.scroll.Direction = container.ScrollVerticalOnly
	}
	t.scroll.Refresh()
	t.lit = -1
	t.follow()
}

// minBox is an empty object of the size a drawing needs (its min size).
func minBox(w, h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, h))
	r.Resize(fyne.NewSize(w, h))
	return r
}

// shortName is a lane's name in a narrow column: its first letters.
func shortName(s string) string {
	r := []rune(s)
	if len(r) > 3 {
		return string(r[:3])
	}
	return s
}

// addBand draws a section's band.
func (t *timelineView) addBand(b storyBand, x, y, w, h float32) {
	bg := canvas.NewRectangle(tint(theme.ColorNamePrimary, uint8(max(0x18, 0x40-0x10*b.level))))
	if c, ok := noteRGB(b.color); ok {
		c.A = 0x60
		bg.FillColor = c
	}
	bg.CornerRadius = 3
	bg.Move(fyne.NewPos(x, y))
	bg.Resize(fyne.NewSize(max(w-2, 2), h))
	t.area.Add(bg)
	room := w - 8
	if p, ok := t.lands[b.item]; ok { // where it lands, at the band's end; a tap goes there and says what to do
		word := "✓"
		col := theme.Color(theme.ColorNameDisabled)
		if p.off() {
			word, col = "⚑ "+p.landWord(), theme.Color(theme.ColorNameWarning)
		}
		mark := canvas.NewText(word+" · "+p.rng().kind(), col) // measured on films or the convention (beat_films.go), where it fits
		mark.TextSize = theme.CaptionTextSize()
		mark.TextStyle = fyne.TextStyle{Bold: p.off()}
		mw := mark.MinSize().Width
		if mw+24 >= room {
			mark.Text = word
			mw = mark.MinSize().Width
		}
		if mw+24 < room { // a band too narrow for it keeps its title
			mark.Move(fyne.NewPos(x+w-6-mw, y+2))
			t.area.Add(mark)
			room -= mw + 6
		}
		zone := newTapZone(func() { t.w.goToLanding(p) })
		zone.Move(fyne.NewPos(x, y))
		zone.Resize(fyne.NewSize(max(w-2, 2), h))
		defer t.area.Add(zone)
	}
	title := canvas.NewText(fitText(b.title, room, theme.CaptionTextSize(), true), theme.Color(theme.ColorNameForeground))
	title.TextSize = theme.CaptionTextSize()
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Move(fyne.NewPos(x+4, y+2))
	t.area.Add(title)
}

// timelineLane is a lane: a character's or a tag's scenes.
type timelineLane struct {
	name string
	in   []bool // by scene
}

// laneList are the lanes the look asks for.
func (t *timelineView) laneList() []timelineLane {
	m := &t.m
	var out []timelineLane
	switch t.lanes {
	case "characters":
		for _, c := range m.chars {
			ln := timelineLane{name: c.name, in: make([]bool, len(m.scenes))}
			for _, k := range c.scenes {
				ln.in[k] = true
			}
			out = append(out, ln)
		}
	case "tags":
		for _, tag := range m.tags {
			ln := timelineLane{name: tag, in: make([]bool, len(m.scenes))}
			for k, s := range m.scenes {
				for _, x := range s.tags {
					if strings.EqualFold(x, tag) {
						ln.in[k] = true
					}
				}
			}
			out = append(out, ln)
		}
	}
	return out
}

// addLane draws a lane: a mark in each scene it is in, a thin line
// between where it is gone (from its first scene to its last).
func (t *timelineView) addLane(ln timelineLane, at func(k int) (fyne.Position, fyne.Size), faint color.Color) {
	first, last := -1, -1
	for k, in := range ln.in {
		if in {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	if first < 0 {
		return
	}
	p0, _ := at(first)
	p1, s1 := at(last)
	gone := canvas.NewRectangle(faint)
	if t.vertical {
		gone.Move(fyne.NewPos(p0.X+s1.Width/2-1, p0.Y))
		gone.Resize(fyne.NewSize(2, p1.Y+s1.Height-p0.Y))
	} else {
		gone.Move(fyne.NewPos(p0.X, p0.Y+s1.Height/2-1))
		gone.Resize(fyne.NewSize(p1.X+s1.Width-p0.X, 2))
	}
	t.area.Add(gone)
	for k, in := range ln.in {
		if !in {
			continue
		}
		p, s := at(k)
		r := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
		if c, ok := noteRGB(t.m.scenes[k].color); ok {
			r.FillColor = c
		}
		r.CornerRadius = 2
		r.Move(p)
		r.Resize(s)
		t.area.Add(r)
	}
}

// addBlock draws scene k's block.
func (t *timelineView) addBlock(k int, pos fyne.Position, size fyne.Size) {
	b := newTimelineBlock(t, k)
	b.Move(pos)
	b.Resize(size)
	t.blocks = append(t.blocks, b)
	t.area.Add(b)
}

// follow lights the cursor's scene, and keeps it in sight.
func (t *timelineView) follow() {
	k := -1
	if e := t.w.textEditor; e != nil {
		line, _ := e.Buffer().Position(e.CursorOffset())
		k = t.m.current(line)
	}
	if k == t.lit {
		return
	}
	if t.lit >= 0 && t.lit < len(t.blocks) {
		t.blocks[t.lit].light(false)
	}
	t.lit = k
	if k >= 0 && k < len(t.blocks) {
		t.blocks[k].light(true)
		t.showLit()
	}
}

// showLit scrolls the lit block into sight.
func (t *timelineView) showLit() {
	if t.lit < 0 || t.lit >= len(t.blocks) {
		return
	}
	b := t.blocks[t.lit]
	s := t.scroll
	view := s.Size()
	if view.Width <= 0 {
		return
	}
	p, sz := b.Position(), b.Size()
	if t.vertical {
		if p.Y < s.Offset.Y || p.Y+sz.Height > s.Offset.Y+view.Height {
			s.Offset.Y = max(0, p.Y-view.Height/3)
		}
	} else if p.X < s.Offset.X || p.X+sz.Width > s.Offset.X+view.Width {
		s.Offset.X = max(0, p.X-view.Width/3)
	}
	s.Refresh()
}

// toScene makes scene k the current one: the script's cursor to its
// heading (the script stays behind the cards).
func (t *timelineView) toScene(k int) {
	if k < 0 || k >= len(t.m.scenes) {
		return
	}
	line := t.m.scenes[k].line
	t.w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	t.follow()
	s := t.m.scenes[k]
	t.w.setStatus(sceneSaid(k, len(t.m.scenes), s))
}

// sceneSaid is a scene as the status says it.
func sceneSaid(k, n int, s storyScene) string {
	out := "Scene " + countText(k+1) + " of " + countText(n) + ": " + s.heading + " · " + lengthText(lexEighths(s.eighths))
	switch {
	case s.synopsisOnly:
		out += " · only its synopsis so far"
	case s.empty:
		out += " · nothing written yet"
	}
	return out
}

// FocusGained shows the focus around the timeline.
func (t *timelineView) FocusGained() {
	t.focused = true
	t.focus.StrokeColor = theme.Color(theme.ColorNameFocus)
	t.focus.Refresh()
	if t.lit < 0 && len(t.m.scenes) > 0 {
		t.follow()
	}
}

func (t *timelineView) FocusLost() {
	t.focused = false
	t.focus.StrokeColor = color.Transparent
	t.focus.Refresh()
}

func (t *timelineView) TypedRune(r rune) {
	switch r {
	case '+', '=':
		t.zoomBy(1)
	case '-':
		t.zoomBy(-1)
	}
}

// TypedKey: the arrows go from scene to scene, Enter opens one.
func (t *timelineView) TypedKey(e *fyne.KeyEvent) {
	n := len(t.m.scenes)
	if n == 0 {
		return
	}
	at := max(t.lit, 0)
	switch e.Name {
	case fyne.KeyLeft, fyne.KeyUp:
		if t.lit < 0 {
			t.toScene(0)
		} else {
			t.toScene(max(0, at-1))
		}
	case fyne.KeyRight, fyne.KeyDown:
		if t.lit < 0 {
			t.toScene(0)
		} else {
			t.toScene(min(n-1, at+1))
		}
	case fyne.KeyHome:
		t.toScene(0)
	case fyne.KeyEnd:
		t.toScene(n - 1)
	case fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace:
		if t.lit >= 0 {
			t.w.goToScript(t.m.scenes[t.lit].line)
		}
	}
}

// Tapped takes the keys.
func (t *timelineView) Tapped(*fyne.PointEvent) { t.w.fyneWindow.Canvas().Focus(t) }

// The timeline as a dropBoard: a block dragged along it moves its scene.
func (t *timelineView) dropIndex(self fyne.CanvasObject, pos fyne.Position) (item int, ok bool) {
	d := fyne.CurrentApp().Driver()
	for k, b := range t.blocks {
		if b == self {
			continue
		}
		at, size := d.AbsolutePositionForObject(b), b.Size()
		lo, hi, p := at.X, at.X+size.Width, pos.X
		if t.vertical {
			lo, hi, p = at.Y, at.Y+size.Height, pos.Y
		}
		if p < lo || p >= hi {
			continue
		}
		if p >= (lo+hi)/2 { // after it: before what follows it
			if k+1 < len(t.m.scenes) {
				return t.m.scenes[k+1].item, true
			}
			return len(t.m.items), true
		}
		return t.m.scenes[k].item, true
	}
	return -1, false
}

func (t *timelineView) dragging(k int, self fyne.CanvasObject, pos fyne.Position) {
	if to, ok := t.dropIndex(self, pos); ok {
		if to < len(t.m.items) {
			t.w.setStatus("Move before " + t.m.items[to].Text)
		} else {
			t.w.setStatus("Move to the end")
		}
	}
}

func (t *timelineView) dropped(k int, self fyne.CanvasObject, pos fyne.Position) {
	to, ok := t.dropIndex(self, pos)
	w := t.w
	old := w.textEditor.Text()
	if ok && k < len(t.m.scenes) {
		line := t.m.scenes[k].line
		w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) }) // it goes, the current one
		w.moveOutlineItem(t.m.scenes[k].item, to)
	}
	if w.textEditor.Text() == old {
		t.draw() // nothing moved: back in its place
	}
}

// timelineHint is the header's line over the timeline.
func timelineHint(phone bool) string {
	if phone {
		return "Tap a scene to open it; drag it to move it; Alt+Up/Down moves the current one"
	}
	return "Tap a scene to open it; drag it, or Alt+Left/Right, to move it; Ctrl+wheel zooms"
}

// timelineBlock is a scene's block on the timeline.
type timelineBlock struct {
	widget.BaseWidget
	dragged
	t     *timelineView
	k     int
	bg    *canvas.Rectangle
	title *canvas.Text
	sub   *canvas.Text
	lit   bool
}

func newTimelineBlock(t *timelineView, k int) *timelineBlock {
	s := t.m.scenes[k]
	b := &timelineBlock{t: t, k: k}
	b.bg = canvas.NewRectangle(color.Transparent)
	b.bg.CornerRadius = 4
	b.title = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	b.title.TextSize = theme.CaptionTextSize()
	b.title.TextStyle = fyne.TextStyle{Bold: true}
	b.sub = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	b.sub.TextSize = theme.CaptionTextSize() * 0.9
	b.dragged = dragged{board: t, i: k, self: b}
	b.ExtendBaseWidget(b)
	b.style()
	name := s.heading + ", " + lengthText(lexEighths(s.eighths))
	if s.synopsisOnly {
		name += ", only a synopsis"
	} else if s.empty {
		name += ", empty"
	}
	fyne.SetAccessibleLabel(b, withColorName(name, s.color))
	return b
}

// style colours the block: its scene's colour (a neutral one without),
// an outline alone while the scene is empty, the primary colour's ring
// while it is the current one.
func (b *timelineBlock) style() {
	s := b.t.m.scenes[b.k]
	fill := tint(theme.ColorNameForeground, 0x28)
	if c, ok := noteRGB(s.color); ok {
		c.A = 0xc0
		fill = c
	}
	b.bg.FillColor = fill
	b.bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	b.bg.StrokeWidth = 1
	b.title.TextStyle.Italic = false
	if s.empty {
		b.bg.FillColor = color.Transparent
		b.bg.StrokeColor = fill
		b.bg.StrokeWidth = 2
		b.title.TextStyle.Italic = true
	}
	if b.lit {
		b.bg.StrokeColor = theme.Color(theme.ColorNamePrimary)
		b.bg.StrokeWidth = 3
	}
}

// light shows whether it is the current scene.
func (b *timelineBlock) light(on bool) {
	b.lit = on
	b.style()
	b.Refresh()
}

func (b *timelineBlock) CreateRenderer() fyne.WidgetRenderer {
	return &timelineBlockRenderer{b: b}
}

type timelineBlockRenderer struct{ b *timelineBlock }

func (r *timelineBlockRenderer) Layout(size fyne.Size) {
	b := r.b
	s := b.t.m.scenes[b.k]
	b.bg.Resize(size)
	ts := b.title.TextSize
	b.title.Text = fitText(s.heading, size.Width-8, ts, true)
	b.title.Move(fyne.NewPos(4, 3))
	sub := lengthText(lexEighths(s.eighths))
	switch {
	case s.synopsisOnly:
		sub = s.synopsis
	case s.empty:
		sub = "Empty"
	}
	b.sub.Text = fitText(sub, size.Width-8, b.sub.TextSize, false)
	b.sub.TextStyle = fyne.TextStyle{Italic: s.empty}
	b.sub.Move(fyne.NewPos(4, 5+ts*1.3))
	b.title.Hidden = size.Height < ts+4
	b.sub.Hidden = size.Height < ts*2.6+6
}

func (r *timelineBlockRenderer) MinSize() fyne.Size { return fyne.NewSize(tlMinBlock, 10) }
func (r *timelineBlockRenderer) Refresh() {
	r.b.title.Color = theme.Color(theme.ColorNameForeground)
	r.b.sub.Color = theme.Color(theme.ColorNameForeground)
	r.Layout(r.b.Size())
	canvas.Refresh(r.b.bg)
	canvas.Refresh(r.b.title)
	canvas.Refresh(r.b.sub)
}
func (r *timelineBlockRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.b.bg, r.b.title, r.b.sub}
}
func (r *timelineBlockRenderer) Destroy() {}

// Tapped opens the scene in the script.
func (b *timelineBlock) Tapped(*fyne.PointEvent) {
	if b.k < len(b.t.m.scenes) {
		b.t.w.goToScript(b.t.m.scenes[b.k].line)
	}
}

// TappedSecondary: its colour, Swap…, Move….
func (b *timelineBlock) TappedSecondary(e *fyne.PointEvent) {
	t := b.t
	if b.k < len(t.m.scenes) {
		t.toScene(b.k)
		t.w.itemMenu(t.m.items[t.m.scenes[b.k].item], e.AbsolutePosition)
	}
}

// Dragged moves the block along the timeline only.
func (b *timelineBlock) Dragged(e *fyne.DragEvent) {
	d := e.Dragged
	if b.t.vertical {
		d.DX = 0
	} else {
		d.DY = 0
	}
	b.Move(b.Position().Add(d))
	b.t.dragging(b.k, b, b.middle())
}

// Cursor is a pointer: the block can be tapped and dragged.
func (b *timelineBlock) Cursor() desktop.Cursor { return desktop.PointerCursor }

// fitText is s cut to fit width at size, with an ellipsis.
func fitText(s string, width, size float32, bold bool) string {
	style := fyne.TextStyle{Bold: bold}
	if width <= 0 {
		return ""
	}
	if fyne.MeasureText(s, size, style).Width <= width {
		return s
	}
	r := []rune(s)
	lo, hi := 0, len(r)
	for lo < hi { // the most runes that fit with the ellipsis
		mid := (lo + hi + 1) / 2
		if fyne.MeasureText(string(r[:mid])+"…", size, style).Width <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		return ""
	}
	return string(r[:lo]) + "…"
}

// zoomScroll is a scroll whose Ctrl+wheel (Cmd on a Mac) zooms.
type zoomScroll struct {
	container.Scroll
	zoom func(dy float32)
}

func newZoomScroll(content fyne.CanvasObject, zoom func(dy float32)) *zoomScroll {
	s := &zoomScroll{zoom: zoom}
	s.Content = content
	s.Direction = container.ScrollHorizontalOnly
	s.ExtendBaseWidget(s)
	return s
}

func (s *zoomScroll) Scrolled(e *fyne.ScrollEvent) {
	if heldModifiers()&(fyne.KeyModifierControl|fyne.KeyModifierSuper) != 0 {
		s.zoom(e.Scrolled.DY)
		return
	}
	s.Scroll.Scrolled(e)
}

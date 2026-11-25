package main

import (
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The timeline strip (LOOP.md, TIMELINE, Joop: "Accolade is very much a
// Fountain/writing first program"): a thin strip over the script's editor
// (View > Timeline Strip, off at first) that follows the cursor while
// writing: the whole story as blocks as long as the scenes, in their
// colours, the acts' starts marked, the scene being written lit; under
// it, where that is (the scene, its length, its act) and who has been
// gone longest. An empty scene is an outline, one with only its synopsis
// an outline with a dot. A tap goes to a scene; the arrow at its start
// folds it to a line of colour. It keeps nothing of its own: it is read
// from the text again once typing pauses, and only lights another block
// as the cursor moves (the per-key cost is a look-up).

// The strip's settings.
const (
	stripPref          = "timeline-strip"
	stripCollapsedPref = "timeline-strip-collapsed"
)

// stripPause is how long typing pauses before the strip reads the story
// again; the pages are measured (the lengths as printed) after a longer
// one.
const (
	stripPause   = 400 * time.Millisecond
	measurePause = 3 * time.Second
)

// stripShown reports whether the strip is over the editor now.
func (w *MainWindow) stripShown() bool {
	return w.settings != nil && w.settings.GetBoolean(stripPref) && !w.markdown && w.scriptInMain()
}

// storyStrip is the strip.
type storyStrip struct {
	widget.BaseWidget
	w        *MainWindow
	of       docKey
	m        storyModel
	lit      int
	fold     *tooltip.Button
	bar      *fyne.Container // the blocks
	caption  *canvas.Text
	box      *fyne.Container
	drawnW   float32
	blocks   []*canvas.Rectangle
	litMark  *canvas.Rectangle
	collapse bool
}

func (w *MainWindow) storyStrip() *storyStrip {
	if w.strip == nil {
		w.strip = newStoryStrip(w)
	}
	return w.strip
}

func newStoryStrip(w *MainWindow) *storyStrip {
	s := &storyStrip{w: w, lit: -1}
	s.bar = container.NewWithoutLayout()
	s.caption = canvas.NewText("", theme.Color(theme.ColorNameForeground))
	s.caption.TextSize = theme.CaptionTextSize()
	s.fold = tooltip.NewButton(theme.MenuDropUpIcon(), "Fold the timeline strip to a line", func() {
		w.changeSetting(stripCollapsedPref, func(st *Settings) { st.SetBoolean(stripCollapsedPref, !st.GetBoolean(stripCollapsedPref)) })
		s.relook()
	}, w.tooltips)
	s.fold.Importance = widget.LowImportance
	s.litMark = canvas.NewRectangle(color.Transparent)
	s.box = container.New(&stripLayout{s: s}, s.fold, s.bar, s.caption)
	s.ExtendBaseWidget(s)
	fyne.SetAccessibleLabel(s, "Timeline strip: the story's scenes, the one you are in lit")
	s.relook()
	return s
}

func (s *storyStrip) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(s.box) }

// relook follows the folding setting.
func (s *storyStrip) relook() {
	s.collapse = s.w.settings != nil && s.w.settings.GetBoolean(stripCollapsedPref)
	if s.collapse {
		s.fold.Icon, s.fold.Tip = theme.MenuDropDownIcon(), "Open the timeline strip"
	} else {
		s.fold.Icon, s.fold.Tip = theme.MenuDropUpIcon(), "Fold the timeline strip to a line"
	}
	setVisible(s.caption, !s.collapse)
	s.fold.Refresh()
	s.drawnW = -1
	s.box.Refresh()
	s.Refresh()
}

// The strip's heights: its blocks, and its line under them.
const (
	stripBlocks = 16
	stripThin   = 6
)

// stripLayout puts the fold button at the start, the blocks beside it and
// the caption under them.
type stripLayout struct{ s *storyStrip }

func (l *stripLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	if l.s.collapse {
		return fyne.NewSize(60, max(stripThin+4, 20))
	}
	return fyne.NewSize(60, stripBlocks+l.s.caption.MinSize().Height+8)
}

func (l *stripLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	s := l.s
	fw := float32(20)
	s.fold.Move(fyne.NewPos(0, 0))
	s.fold.Resize(fyne.NewSize(fw, min(size.Height, 20)))
	s.bar.Move(fyne.NewPos(fw+2, 3))
	h := float32(stripBlocks)
	if s.collapse {
		h = stripThin
		s.bar.Move(fyne.NewPos(fw+2, (min(size.Height, 20)-h)/2))
	}
	s.bar.Resize(fyne.NewSize(size.Width-fw-8, h))
	s.caption.Move(fyne.NewPos(fw+4, h+5))
	if bw := size.Width - fw - 8; bw != s.drawnW {
		s.draw(bw, h)
	}
}

// update reads the story again (once typing pauses).
func (s *storyStrip) update() {
	s.of = s.w.scriptKey()
	s.m = s.w.storyNow()
	s.w.measureStoryLater()
	s.drawnW = -1
	s.box.Refresh()
}

// draw lays the blocks out across width.
func (s *storyStrip) draw(width, h float32) {
	s.drawnW = width
	s.bar.RemoveAll()
	s.blocks = nil
	m := &s.m
	total := float32(max(1, m.total))
	x := float32(0)
	starts := map[int]int{} // a band's first scene: its level (the top one)
	off := map[int]bool{}   // a band's first scene: a section there lands off its range (beat_place.go), its tick flagged
	lands := s.w.landings(m.items, false)
	for _, b := range m.bands {
		if lv, ok := starts[b.first]; !ok || b.level < lv {
			starts[b.first] = b.level
		}
		if b.item < len(m.items) {
			if p, ok := lands[m.items[b.item].Line]; ok && p.off() {
				off[b.first] = true
			}
		}
	}
	neutral := tint(theme.ColorNameForeground, 0x30)
	for k, sc := range m.scenes {
		bw := float32(sc.eighths) / total * width
		if lv, ok := starts[k]; ok && k > 0 { // an act's start: a tick
			tick := canvas.NewRectangle(theme.Color(theme.ColorNameForeground))
			if off[k] {
				tick.FillColor = theme.Color(theme.ColorNameWarning)
			}
			th := h
			if lv > 1 {
				th = h * 0.6
			}
			tick.Move(fyne.NewPos(x-1, h-th))
			tick.Resize(fyne.NewSize(2, th))
			s.bar.Add(tick)
		}
		r := canvas.NewRectangle(neutral)
		if c, ok := noteRGB(sc.color); ok {
			r.FillColor = c
		}
		if sc.empty {
			r.StrokeColor, r.StrokeWidth = r.FillColor, 1
			r.FillColor = color.Transparent
		}
		r.Move(fyne.NewPos(x+1, 0))
		r.Resize(fyne.NewSize(max(1, bw-2), h))
		s.bar.Add(r)
		if sc.synopsisOnly && bw >= 4 && !s.collapse {
			dot := canvas.NewCircle(theme.Color(theme.ColorNameForeground))
			dot.Move(fyne.NewPos(x+bw/2-1.5, h/2-1.5))
			dot.Resize(fyne.NewSize(3, 3))
			s.bar.Add(dot)
		}
		s.blocks = append(s.blocks, r)
		x += bw
	}
	s.bar.Add(s.litMark)
	s.lit = -1
	s.follow()
	s.bar.Refresh()
}

// follow lights the cursor's scene and says where it is: cheap, it runs
// on each cursor move.
func (s *storyStrip) follow() {
	e := s.w.textEditor
	if e == nil {
		return
	}
	line, _ := e.Buffer().Position(e.CursorOffset())
	k := s.m.current(line)
	if k == s.lit {
		return
	}
	s.lit = k
	if k < 0 || k >= len(s.blocks) {
		s.litMark.Hide()
		s.setCaption(s.startCaption())
		return
	}
	b := s.blocks[k]
	s.litMark.StrokeColor = theme.Color(theme.ColorNamePrimary)
	s.litMark.StrokeWidth = 2
	s.litMark.FillColor = tint(theme.ColorNamePrimary, 0x50)
	s.litMark.Move(b.Position().Subtract(fyne.NewPos(1, 2)))
	s.litMark.Resize(b.Size().Add(fyne.NewSize(2, 4)))
	s.litMark.Show()
	s.litMark.Refresh()
	s.setCaption(s.where(k))
}

func (s *storyStrip) setCaption(t string) {
	if s.caption.Text != t {
		s.caption.Text = t
		s.caption.Color = theme.Color(theme.ColorNameForeground)
		s.caption.Refresh()
	}
}

// startCaption is the caption before the first scene.
func (s *storyStrip) startCaption() string {
	if len(s.m.scenes) == 0 {
		return "No scenes yet"
	}
	return countText(len(s.m.scenes)) + " scenes · " + lengthText(lexEighths(s.m.total))
}

// where says where scene k is: its place, length and act, what it still
// needs, and who has been gone longest.
func (s *storyStrip) where(k int) string {
	sc := s.m.scenes[k]
	parts := []string{"Scene " + countText(k+1) + " of " + countText(len(s.m.scenes)), lengthText(lexEighths(sc.eighths))}
	if len(sc.sections) > 0 {
		parts = append(parts, s.m.items[sc.sections[0]].Text)
	}
	switch {
	case sc.synopsisOnly:
		parts = append(parts, "only its synopsis so far")
	case sc.empty:
		parts = append(parts, "nothing written yet")
	}
	if gone := s.m.goneFor(k, 4); len(gone) > 0 {
		var g []string
		for _, x := range gone[:min(3, len(gone))] {
			g = append(g, x.name+" "+countText(x.scenes))
		}
		parts = append(parts, "gone longest (scenes): "+strings.Join(g, ", "))
	}
	return strings.Join(parts, " · ")
}

// Tapped goes to the scene tapped, in the editor.
func (s *storyStrip) Tapped(e *fyne.PointEvent) {
	x := e.Position.X - s.bar.Position().X
	for k, b := range s.blocks {
		if x >= b.Position().X-0.5 && x < b.Position().X+b.Size().Width+0.5 {
			line := s.m.scenes[k].line
			ed := s.w.textEditor
			ed.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(bf.LineStart(line), false) })
			s.w.fyneWindow.Canvas().Focus(ed)
			s.follow()
			return
		}
	}
}

// scriptView is the script's editor as the middle shows it: with the
// strip over it when it is on.
func (w *MainWindow) scriptView() fyne.CanvasObject {
	if !w.stripShown() {
		return w.editorView
	}
	s := w.storyStrip()
	if s.of != w.scriptKey() {
		s.update()
	}
	return container.NewBorder(container.NewVBox(s, widget.NewSeparator()), nil, nil, nil, w.editorView)
}

// toggleStrip is View > Timeline Strip.
func (w *MainWindow) toggleStrip() {
	w.changeSetting(stripPref, func(s *Settings) { s.SetBoolean(stripPref, !s.GetBoolean(stripPref)) })
	if w.stripShown() {
		w.storyStrip().update()
	}
	w.relayout()
	w.focusMain()
}

// stripEdited reads the story again once typing pauses (the per-key
// cost: a timer).
func (w *MainWindow) stripEdited() {
	if w.strip == nil || !w.stripShown() {
		return
	}
	w.debounce(&w.stripTimer, stripPause, func() {
		if w.stripShown() {
			w.strip.update()
		}
	})
}

// stripCursor lights the cursor's scene on the strip.
func (w *MainWindow) stripCursor() {
	if w.strip != nil && w.stripShown() {
		w.strip.follow()
	}
}

// storyCache is the story as last read, for the text it was read from.
type storyCache struct {
	text  string
	lang  string
	lens  int // the measure it took the lengths from (scheduleState.run)
	m     storyModel
	valid bool
}

// storyNow is the script's story (story_model.go), read once a text.
func (w *MainWindow) storyNow() storyModel {
	text := w.textEditor.Text()
	c := &w.story
	if c.valid && sameString(c.text, text) && c.lang == w.language && c.lens == w.schedule.lensRun {
		return c.m
	}
	starts := w.sceneStarts()
	lines := strings.Split(text, "\n")
	m := storyOf(text, starts, syntax.ClassifyIn(lines, starts, w.dialect()))
	m.withLengths(w.measuredLengths(m, lines))
	*c = storyCache{text: text, lang: w.language, lens: w.schedule.lensRun, m: m, valid: true}
	return m
}

// headingKey keys a scene by its heading's text and how many before it
// read the same, so a length measured stays with its scene while the
// text around it changes.
func headingKeys(lines []string, heads []int) []string {
	seen := map[string]int{}
	out := make([]string, len(heads))
	for i, l := range heads {
		t := ""
		if l >= 0 && l < len(lines) {
			t = strings.TrimSpace(lines[l])
		}
		seen[t]++
		out[i] = t + "\x00" + countText(seen[t])
	}
	return out
}

// measuredLengths are the scenes' lengths as last measured (the
// schedule's measure: the pages laid out), by heading line.
func (w *MainWindow) measuredLengths(m storyModel, lines []string) map[int]int {
	byKey := w.schedule.lens
	if len(byKey) == 0 {
		return nil
	}
	heads := make([]int, len(m.scenes))
	for k, s := range m.scenes {
		heads[k] = s.line
	}
	out := map[int]int{}
	for k, key := range headingKeys(lines, heads) {
		if e, ok := byKey[key]; ok {
			out[heads[k]] = e
		}
	}
	return out
}

// measureStoryLater measures the pages once typing has paused a while,
// for the timeline's and the strip's lengths (as the stripboard has them).
func (w *MainWindow) measureStoryLater() {
	if w.textEditor == nil || w.markdown {
		return
	}
	w.debounce(&w.measureTimer, measurePause, func() {
		if !w.markdown && (w.stripShown() || w.cardsShown() && w.cardsLook().timeline) {
			w.measureSchedule()
		}
	})
}

// storyMeasured shows the lengths measured on the timeline and strip.
func (w *MainWindow) storyMeasured() {
	if w.strip != nil && w.stripShown() {
		w.strip.update()
	}
	if w.cards != nil && w.cardsShown() && w.cardsLook().timeline {
		w.cards.tl.update()
	}
}

package main

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The source pane's side of the statuses (adaptation.go): a mark beside
// each passage, the count, Mark as Cut, Next and Previous (adapt_plan.go
// for where they go on from), Adapt… (adapt_dialog.go).

// statusMarkWidth is how wide a passage's status mark is.
const statusMarkWidth = 7

// statusMark is a passage's status as a bar beside it, told apart by its
// shape as well as its colour (for colour blindness, and in grayscale):
// adapted a full solid bar (green), cut a dashed one (red), not started
// a thin line (gray); a heading has none.
type statusMark struct {
	widget.BaseWidget
	status  passageStatus
	changed bool // adapted, but the passage changed since: in the warning colour
}

func newStatusMark() *statusMark {
	m := &statusMark{}
	m.ExtendBaseWidget(m)
	return m
}

func (m *statusMark) setStatus(st passageStatus, changed bool) {
	if st != m.status || changed != m.changed {
		m.status, m.changed = st, changed
		m.Refresh()
	}
}

func (m *statusMark) MinSize() fyne.Size { return fyne.NewSize(statusMarkWidth, 0) }

func (m *statusMark) CreateRenderer() fyne.WidgetRenderer {
	return &statusMarkRenderer{m: m}
}

type statusMarkRenderer struct {
	m       *statusMark
	objects []fyne.CanvasObject
	size    fyne.Size
}

func (r *statusMarkRenderer) Layout(size fyne.Size) {
	r.size = size
	r.Refresh()
}

func (r *statusMarkRenderer) MinSize() fyne.Size { return r.m.MinSize() }

func (r *statusMarkRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *statusMarkRenderer) Destroy() {}

// rect is the renderer's i-th rectangle, made if need be.
func (r *statusMarkRenderer) rect(i int) *canvas.Rectangle {
	for len(r.objects) <= i {
		r.objects = append(r.objects, canvas.NewRectangle(color.Transparent))
	}
	return r.objects[i].(*canvas.Rectangle)
}

func (r *statusMarkRenderer) Refresh() {
	size := r.size
	const inset = 3 // from the row's top and bottom
	h := size.Height - 2*inset
	n := 0
	if h > 0 {
		switch r.m.status {
		case statusAdapted:
			b := r.rect(0)
			b.FillColor, b.StrokeWidth, b.CornerRadius = theme.Color(theme.ColorNameSuccess), 0, 1
			if r.m.changed {
				b.FillColor = theme.Color(theme.ColorNameWarning)
			}
			b.Move(fyne.NewPos(0, inset))
			b.Resize(fyne.NewSize(size.Width, h))
			n = 1
		case statusToDo:
			b := r.rect(0)
			b.FillColor, b.StrokeWidth, b.CornerRadius = theme.Color(theme.ColorNamePlaceHolder), 0, 0
			b.Move(fyne.NewPos((size.Width-2)/2, inset))
			b.Resize(fyne.NewSize(2, h))
			n = 1
		case statusCut:
			const dash, gap = 4, 3
			for y := float32(0); y+dash <= h || n == 0; y += dash + gap {
				b := r.rect(n)
				b.FillColor, b.StrokeWidth, b.CornerRadius = theme.Color(theme.ColorNameError), 0, 0
				b.Move(fyne.NewPos(0, inset+y))
				b.Resize(fyne.NewSize(size.Width, min(dash, h)))
				n++
			}
		}
	}
	for i, o := range r.objects {
		if i < n {
			o.Show()
		} else {
			o.Hide()
		}
		o.Refresh()
	}
}

// statusPrompt is the pane's question: go round after Next found none
// left, a suggested link, starting from the source. Not a dialog:
// writing goes on, and the next question replaces it.
type statusPrompt struct {
	label         *widget.Label
	yes, skip, no *widget.Button
	box           *fyne.Container
	yesDo, skipDo func()
	passage       int // what it asks about (-1: no passage)
	wrap          int // asks to go round, after Next (1) or Previous (-1); 0 not
}

func newStatusPrompt(s *sourcePane) *statusPrompt {
	p := &statusPrompt{passage: -1}
	p.label = widget.NewLabel("")
	p.label.Wrapping = fyne.TextWrapWord
	p.yes = widget.NewButton("", func() {
		if do := p.yesDo; do != nil {
			p.hide()
			do()
		}
	})
	p.yes.Importance = widget.HighImportance
	p.skip = widget.NewButton("", func() {
		if do := p.skipDo; do != nil {
			do()
		}
	})
	p.skip.Hide()
	p.no = widget.NewButton("Not Now", func() {
		p.hide()
		s.w.fyneWindow.Canvas().Focus(s.w.textEditor)
	})
	fyne.SetAccessibleLabel(p.no, "Not now: close the question")
	bg := canvas.NewRectangle(tint(theme.ColorNamePrimary, 0x20))
	bg.CornerRadius = 4
	p.box = container.NewStack(bg, container.NewPadded(container.NewVBox(p.label, container.NewHBox(p.yes, p.skip, p.no))))
	p.box.Hide()
	return p
}

// ask shows the question, with the action for its first button.
func (p *statusPrompt) ask(passage int, text, yes string, do func()) {
	p.passage, p.wrap, p.yesDo, p.skipDo = passage, 0, do, nil
	p.skip.Hide()
	p.label.SetText(text)
	p.yes.SetText(yes)
	fyne.SetAccessibleLabel(p.yes, yes)
	if bg, ok := p.box.Objects[0].(*canvas.Rectangle); ok {
		bg.FillColor = tint(theme.ColorNamePrimary, 0x20)
	}
	p.box.Show()
	p.box.Refresh()
}

// askSkip shows a question of a series: its action, a Skip to the next
// one, and Not Now.
func (p *statusPrompt) askSkip(text, yes string, do func(), skip string, skipDo func()) {
	p.ask(-1, text, yes, do)
	p.skipDo = skipDo
	p.skip.SetText(skip)
	fyne.SetAccessibleLabel(p.skip, skip)
	p.skip.Show()
}

func (p *statusPrompt) hide() {
	p.passage, p.wrap, p.yesDo, p.skipDo = -1, 0, nil, nil
	p.box.Hide()
}

// The pane.

// statusOf is passage id's status as the pane shows it.
func (s *sourcePane) statusOf(id int) passageStatus {
	if id < 0 || id >= len(s.statuses) {
		return statusNone
	}
	return s.statuses[id]
}

// showStatuses takes the statuses and the count from the model: the
// rows again if a status changed, the count line, the Cut button.
func (s *sourcePane) showStatuses(m *sourceModel) {
	// changed since linked: linked nearly (the quoted words changed), or
	// the passage's words other than when this session first saw it linked
	changed := map[int]bool{}
	if s.linkedWords == nil {
		s.linkedWords = map[string][]string{}
	}
	for i, mt := range m.matches {
		if !mt.found() {
			continue
		}
		for p := mt.Passages[0]; p <= mt.last(); p++ {
			key := m.links[i].Key
			if m.links[i].Ref.Old {
				key = strings.Join(m.links[i].Words, " ")
			}
			if p > mt.Passages[0] {
				key += "\x03" + strconv.Itoa(p-mt.Passages[0])
			}
			if seen, ok := s.linkedWords[key]; !ok {
				s.linkedWords[key] = s.passages[p].Words
			} else if !slices.Equal(seen, s.passages[p].Words) {
				changed[p] = true
			}
			if mt.Kind == linkNear {
				changed[p] = true
			}
		}
	}
	linkedBy := map[int][]int{}
	for i, mt := range m.matches {
		if !mt.found() {
			continue
		}
		at := spanAt(m.spans, m.links[i].Line).Line
		for p := mt.Passages[0]; p <= mt.last(); p++ {
			if !slices.Contains(linkedBy[p], at) {
				linkedBy[p] = append(linkedBy[p], at)
				slices.Sort(linkedBy[p])
			}
		}
	}
	if !slices.Equal(m.status, s.statuses) || !sameSet(changed, s.changed) || !sameLinks(linkedBy, s.linkedBy) {
		s.statuses = slices.Clone(m.status)
		s.changed = changed
		s.linkedBy = linkedBy
		s.syncActs()
		s.list.Refresh()
	}
	if s.prompt.passage >= 0 && s.statusOf(s.prompt.passage) != statusToDo {
		s.prompt.hide() // linked or cut meanwhile: nothing to ask
	}
	count := countOf(s.statuses).String()
	if n := len(s.changed); n > 0 {
		count += fmt.Sprintf(" · %d changed since linked", n)
	}
	s.count.SetText(count)
}

// sameLinks reports whether a and b link the same passages to the same
// spans.
func sameLinks(a, b map[int][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !slices.Equal(v, b[k]) {
			return false
		}
	}
	return true
}

// toggleCut marks the picked passage as cut, or unmarks it: one edit to
// the script's adaptation block, undone as one.
func (s *sourcePane) toggleCut() {
	s.load()
	if s.selected < 0 || s.selected >= len(s.passages) {
		s.w.setStatus("Pick the passage in the source first")
		return
	}
	s.setCut(s.selected, s.statusOf(s.selected) != statusCut)
}

// setCut marks passage p as cut (on) or not.
func (s *sourcePane) setCut(p int, on bool) {
	m := s.current()
	switch m.status[p] {
	case statusNone:
		s.w.setStatus("A heading is not cut or adapted, only its passages")
		return
	case statusAdapted:
		s.w.setStatus("This passage is adapted: a scene links to it")
		return
	}
	if on == (m.status[p] == statusCut) {
		return
	}
	cuts := s.cutsWithout(m, p)
	if on {
		cuts = append(cuts, cutFor(s.passages, p))
	}
	e := s.w.textEditor
	inside, _ := adaptationBlock(e.Text())
	start, end, block := withAdaptationBlock(e.Text(), withCuts(inside, cuts))
	e.Edit(func(b *buffer.Buffer) {
		cursor := b.Cursor()
		b.Replace(start, end, block)
		if cursor > start {
			cursor = min(cursor, start)
		}
		b.SetCursor(cursor, false)
	})
	if on {
		s.w.setStatus("Cut on purpose: “" + shortQuote(s.passages[p].Plain) + "”")
	} else {
		s.w.setStatus("Not cut any more: “" + shortQuote(s.passages[p].Plain) + "”")
	}
	s.follow()
}

// cutsWithout are the model's cut entries but those naming passage p.
func (s *sourcePane) cutsWithout(m *sourceModel, p int) []cutMark {
	var cuts []cutMark
	for i, c := range m.cuts {
		if !m.cutAt[i].has(p) {
			cuts = append(cuts, c)
		}
	}
	return cuts
}

// stepPassage is Next (dir 1) and Previous (dir -1): the passage still
// to do after (before) the anchor, in the source's order, picked and
// scrolled to; nothing is written (Adapt… writes it). At the end it says
// so and offers to go round; pressed again then, it goes round.
func (s *sourcePane) stepPassage(dir int) {
	w := s.w
	if !s.shown() {
		w.showPane("source", false)
	}
	s.load()
	if s.ref.Origin == sourceNone {
		w.setStatus("No source: Adaptation > Choose Source…")
		return
	}
	m := s.current()
	if s.prompt.wrap == dir && s.prompt.box.Visible() { // pressed again on the offer: round
		s.prompt.hide()
		s.wrapPassage(dir)
		return
	}
	a := s.anchorNow(m)
	next := m.step(a, dir)
	if next < 0 {
		round := m.wrapTo(dir)
		switch {
		case round < 0:
			s.prompt.hide()
			w.setStatus("Every passage is adapted or cut; what is added to the source later, Next finds")
		case round == a.passage && a.kind == anchorPicked:
			s.prompt.hide()
			w.setStatus("No other passage is left to adapt")
		default:
			end, button := "after", "From the Start"
			if dir < 0 {
				end, button = "before", "From the End"
			}
			text := "No passage to adapt " + end + " " + s.anchorText(a) + ". Go round to “" + shortQuote(s.passages[round].Plain) + "”?"
			d := dir
			s.prompt.ask(-1, text, button, func() { s.wrapPassage(d) })
			s.prompt.wrap = dir
			w.setStatus("None left that way: " + button + " in the source pane (or " + map[int]string{1: "Next", -1: "Previous"}[dir] + " again) goes round")
		}
		s.follow()
		return
	}
	s.prompt.hide()
	s.offer(next)
	w.setStatus("To adapt: “" + shortQuote(s.passages[next].Plain) + "”. Adapt… writes it into the script")
	s.follow()
}

// wrapPassage goes round: the first passage still to do (dir 1), or the
// last.
func (s *sourcePane) wrapPassage(dir int) {
	p := s.current().wrapTo(dir)
	if p < 0 {
		return
	}
	s.offer(p)
	from := "start"
	if dir < 0 {
		from = "end"
	}
	s.w.setStatus("From the " + from + " of the source: “" + shortQuote(s.passages[p].Plain) + "”")
	s.follow()
}

// anchorNow is the anchor as the writer is now: the passage picked, if
// picked (or gone to with Next) since the cursor last moved; the
// cursor's scene; the top of the pane's list.
func (s *sourcePane) anchorNow(m *sourceModel) anchor {
	e := s.w.textEditor
	picked := -1
	if s.pickFresh && e.CursorOffset() == s.pickCursor && s.selected >= 0 && s.selected < len(s.passages) {
		picked = s.selected
	}
	top := -1
	if s.shown() && s.list.Size().Height > 0 && len(s.heights) == len(s.passages) && len(s.passages) > 0 {
		top = s.firstInView()
	}
	line, _ := e.Buffer().Position(e.CursorOffset())
	return m.anchorAt(line, picked, top)
}

// anchorText is the anchor in words: “…”, the passage.
func (s *sourcePane) anchorText(a anchor) string {
	if a.passage < 0 || a.passage >= len(s.passages) {
		return "the start of the source"
	}
	return "“" + shortQuote(s.passages[a.passage].Plain) + "”"
}

// showAnchor is the caption line saying where Next and Previous go on
// from.
func (s *sourcePane) showAnchor(m *sourceModel) {
	a := s.anchorNow(m)
	var t string
	switch a.kind {
	case anchorStart:
		t = "Next goes from the start of the source"
	case anchorView:
		t = "from: " + s.anchorText(a) + " · the top of the list"
	case anchorPicked:
		t = "after: " + s.anchorText(a) + " · picked"
	case anchorScene:
		t = "after: " + s.anchorText(a) + " · the cursor's scene"
	case anchorBefore:
		label := spanLabelsFor(s)[a.span]
		t = "after: " + s.anchorText(a) + " · " + label + ", above the cursor"
	}
	if s.anchorLine.Text != t {
		s.anchorLine.SetText(t)
	}
}

// offer picks passage p as the one Next or Previous went to, and scrolls
// to it.
func (s *sourcePane) offer(p int) {
	s.offered, s.offeredWords = p, nil
	if p < 0 {
		return
	}
	s.offeredWords = s.passages[p].Words
	s.setPicked(p)
	s.list.ScrollTo(p)
}

// offeredPassage is the passage Next showed last, found again by its
// words if the source changed (-1: none).
func (s *sourcePane) offeredPassage() int {
	if s.offered < 0 || s.offered >= len(s.passages) || !slices.Equal(s.passages[s.offered].Words, s.offeredWords) {
		s.offered = slices.IndexFunc(s.passages, func(p passage) bool {
			return s.offeredWords != nil && slices.Equal(p.Words, s.offeredWords)
		})
	}
	return s.offered
}

// The window's side.

// nextSourcePassage is Adaptation > Next Source Passage.
func (w *MainWindow) nextSourcePassage() { w.stepSourcePassage(1, "Next Source Passage") }

// previousSourcePassage is Adaptation > Previous Source Passage.
func (w *MainWindow) previousSourcePassage() { w.stepSourcePassage(-1, "Previous Source Passage") }

func (w *MainWindow) stepSourcePassage(dir int, what string) {
	if w.scriptAway(what) { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown {
		return
	}
	w.sourcePane().stepPassage(dir)
}

// adaptSourcePassage is Adaptation > Adapt Source Passage…: Adapt… for
// the passage picked in the source pane.
func (w *MainWindow) adaptSourcePassage() {
	if w.scriptAway("Adapt Source Passage") {
		return
	}
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.showPane("source", false)
	}
	s.load()
	s.adaptPassage(s.selected)
}

// goToPassageScene is Adaptation > Go to the Passage's Scene: the scene
// linking to the passage picked in the source pane, the next of them on
// each go if several do.
func (w *MainWindow) goToPassageScene() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	s.load()
	if s.selected < 0 || s.selected >= len(s.passages) {
		w.setStatus("Pick the passage in the source first")
		return
	}
	scenes := s.current().linked(s.selected)
	if len(scenes) == 0 {
		w.setStatus("No scene links to “" + shortQuote(s.passages[s.selected].Plain) + "” yet: Adapt… writes it into the script")
		return
	}
	if s.cycleFor != s.selected {
		s.cycleFor, s.cycle = s.selected, 0
	} else {
		s.cycle++
	}
	s.jumpTo(s.selected, s.cycle%len(scenes))
}

// flashTime is how long a scene gone to from the source pane is lit.
const flashTime = 900 * time.Millisecond

// flashScene lights the scene (or section) at line for a moment, in the
// script's editor; with the script swapped out the Script tab lights it
// already.
func (w *MainWindow) flashScene(line int) {
	if !w.scriptInMain() {
		return
	}
	e := w.textEditor
	text := e.Text()
	span := spanAt(sceneSpans(text, w.sceneStarts()), line)
	b := e.Buffer()
	n, _ := b.Position(b.Len())
	start, end := b.LineStart(span.Line), b.Len()
	if span.End <= n {
		end = b.LineStart(span.End)
	}
	end = min(end, bodyEnd(text))
	e.SetFlash(start, end)
	cancel(&w.flashTimer)
	w.flashTimer = w.after(flashTime, func() {
		w.flashTimer = nil
		e.SetFlash(0, 0)
	})
}

// toggleCutHere is Adaptation > Mark Source Passage as Cut: the passage
// picked in the source pane cut on purpose, or not any more.
func (w *MainWindow) toggleCutHere() {
	if w.scriptAway("Mark Source Passage as Cut") { // the script is in the side panes (swap.go)
		return
	}
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.showPane("source", true)
		w.setStatus("Pick the passage in the source, then Mark Source Passage as Cut again")
		return
	}
	s.toggleCut()
}

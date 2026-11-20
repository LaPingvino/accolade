package main

import (
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The source pane's side of the statuses (adaptation.go): a mark beside
// each passage, the count, Mark as Cut, and Next passage with what it
// asks.

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

// statusPrompt is the pane's question after Next: start a scene for the
// passage, or mark the one skipped as cut. Not a dialog: writing goes
// on, and the next Next replaces it.
type statusPrompt struct {
	label         *widget.Label
	yes, skip, no *widget.Button
	box           *fyne.Container
	yesDo, skipDo func()
	passage       int // what it asks about
	skipping      bool
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
func (p *statusPrompt) ask(passage int, skipping bool, text, yes string, do func()) {
	p.passage, p.skipping, p.yesDo, p.skipDo = passage, skipping, do, nil
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
	p.ask(-1, false, text, yes, do)
	p.skipDo = skipDo
	p.skip.SetText(skip)
	fyne.SetAccessibleLabel(p.skip, skip)
	p.skip.Show()
}

func (p *statusPrompt) hide() {
	p.passage, p.yesDo, p.skipDo = -1, nil, nil
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
	if !slices.Equal(m.status, s.statuses) || !sameSet(changed, s.changed) {
		s.statuses = slices.Clone(m.status)
		s.changed = changed
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
	s.showCutButton()
}

// showCutButton is Mark as Cut, or Unmark for a cut passage; there only
// while a passage that can be cut is picked.
func (s *sourcePane) showCutButton() {
	st := s.statusOf(s.selected)
	label, name := "Mark as Cut", "Mark the picked passage as cut on purpose"
	if st == statusCut {
		label, name = "Unmark Cut", "Unmark the picked passage: not cut"
	}
	if s.cut.Text != label {
		s.cut.SetText(label)
	}
	fyne.SetAccessibleLabel(s.cut, name)
	if st == statusToDo || st == statusCut {
		s.cut.Enable()
	} else {
		s.cut.Disable()
	}
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

// nextPassage is Next passage: the first passage after where the script
// is in the source (the last one the cursor's scene links to, or the
// one Next showed last) that is neither adapted nor cut. It is picked
// and scrolled to; a scene is started for it (or offered, as the
// preference says). Pressed again without linking the one it showed,
// it asks whether that one is cut.
func (s *sourcePane) nextPassage() {
	w := s.w
	if !s.shown() {
		w.notes.tabs.Select(w.notes.sourceTab)
		if !w.notesVisible {
			w.toggleNotes()
		}
	}
	s.load()
	if s.ref.Origin == sourceNone {
		w.setStatus("No source: File > Choose Source…")
		return
	}
	m := s.current()
	e := w.textEditor
	line, _ := e.Buffer().Position(e.CursorOffset())
	pos := m.position(line)
	s.nextAt = -1 // the scene goes after the cursor's
	if last, at := m.lastLinked(); pos < 0 && last >= 0 {
		// the cursor before every link (a script just opened): on from the
		// last passage adapted, the scene after the scene linking to it
		pos, s.nextAt = last, e.Buffer().LineStart(at)
	}
	skipped := -1
	if o := s.offeredPassage(); o >= 0 && m.status[o] == statusToDo {
		skipped = o
		pos = max(pos, o)
	}
	next, wrapped := m.nextToDo(pos)
	if skipped >= 0 && next == skipped {
		next = -1 // the only one left
	}
	s.offer(next)
	switch {
	case skipped >= 0:
		sk := skipped
		text := "Skipped “" + shortQuote(s.passages[sk].Plain) + "”. Cut it on purpose?"
		s.prompt.ask(sk, true, text, "Mark as Cut", func() { s.setCut(sk, true) })
		if next >= 0 && w.nextInserts() {
			s.startScene(next)
		}
	case next < 0:
		s.prompt.hide()
	case w.nextInserts():
		s.prompt.hide()
		s.startScene(next)
	default:
		n := next
		s.prompt.ask(n, false, "Next: “"+shortQuote(s.passages[n].Plain)+"”", "Start a Scene", func() { s.startScene(n) })
	}
	switch {
	case next < 0 && skipped >= 0:
		w.setStatus("No other passage is left to adapt")
	case next < 0:
		w.setStatus("Every passage is adapted or cut; what is added to the source later, Next finds")
	case wrapped:
		w.setStatus("Next passage, from the start of the source: none is left after this scene's")
	case !w.nextInserts():
		w.setStatus("Next passage: Start a Scene in the source pane, or link it where you like")
	}
	s.follow()
}

// offer picks passage p as the one Next shows, and scrolls to it.
func (s *sourcePane) offer(p int) {
	s.offered, s.offeredWords = p, nil
	if p < 0 {
		return
	}
	s.offeredWords = s.passages[p].Words
	s.selected, s.cycle = p, 0
	s.list.Refresh()
	s.list.ScrollTo(p)
	s.showCutButton()
	s.showReading(true)
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

// startScene puts a scene for passage p in the script after the
// cursor's scene (or after the one Next went on from, nextAt;
// sceneForPassage), one undo step, its heading's placeholder selected to
// write over.
func (s *sourcePane) startScene(p int) {
	w := s.w
	if p < 0 || p >= len(s.passages) {
		return
	}
	e := w.textEditor
	at := e.CursorOffset()
	if s.nextAt >= 0 && s.nextAt <= len([]rune(e.Text())) {
		at = s.nextAt
	}
	s.nextAt = -1
	line, _ := e.Buffer().Position(at)
	ed := sceneForPassage(e.Text(), w.sceneStarts(), s.passages, p, s.current().position(line), at)
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			b.Replace(ed.At, ed.Replace, ed.Text)
			b.Select(ed.Select, ed.SelectEnd)
		})
	})
	w.fyneWindow.Canvas().Focus(e)
	w.setStatus("A scene for “" + shortQuote(s.passages[p].Plain) + "”: write its heading over the placeholder")
	s.follow()
}

// The window's side.

// nextInserts reports whether Next passage starts a scene for the
// passage itself (the preference "next-passage-inserts"), or only shows
// it with the offer.
func (w *MainWindow) nextInserts() bool {
	return w.settings == nil || w.settings.GetBoolean("next-passage-inserts")
}

// nextSourcePassage is Format > Next Source Passage.
func (w *MainWindow) nextSourcePassage() {
	if w.markdown {
		return
	}
	w.sourcePane().nextPassage()
}

// toggleCutHere is Format > Mark Source Passage as Cut: the passage
// picked in the source pane cut on purpose, or not any more.
func (w *MainWindow) toggleCutHere() {
	if w.markdown {
		return
	}
	s := w.sourcePane()
	if !s.shown() {
		w.toggleSource()
		w.setStatus("Pick the passage in the source, then Mark Source Passage as Cut again")
		return
	}
	s.toggleCut()
}

package main

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Adapt… (Adaptation > Adapt Source Passage…, or Adapt… under the passage
// picked in the source pane): the step between choosing a passage and
// writing it into the script. How (a new scene, a drafted one, the
// current scene, cut, skip) and where (in the story's order, at the
// cursor, after the cursor's scene, at the end), the likely one of each
// chosen with its reason, and exactly what goes in, to edit. A dialog on
// a desktop, a sheet at the bottom on a phone. Enter takes the choices;
// what goes in is one undo step, an insertion only.

type adaptDialog struct {
	s    *sourcePane
	p    int
	ctx  adaptCtx
	plan adaptPlan

	how, where *choiceList
	heading    *widget.Entry
	headings   *choiceList // the headings the source suggests, when more than one
	headRow    *fyne.Container
	whereBox   *fyne.Container
	preview    *widget.Entry
	ok         *widget.Button
	pop        *widget.PopUp
	box        *fyne.Container
	body       *fyne.Container // in the box's scroll
	wide       bool            // two columns (a desktop with the room)

	headingNow string // the heading as the preview has it
	// linksBack puts the links' own text back for their readable form in
	// the preview (link_chips.go's readableLinks)
	linksBack func(string) string
	edited    bool // the preview written in by hand
	setting   bool // the dialog is setting the preview
	waits     int  // looks again for the story's reading (a long one, read in the background)
}

// adaptCtx is what Adapt… plans passage p from: the script and cursor as
// they are now, what the source suggests for it.
func (s *sourcePane) adaptCtx(p int) adaptCtx {
	w := s.w
	e := w.textEditor
	text := e.Text()
	c := adaptCtx{text: text, starts: w.sceneStarts(), ps: s.passages, m: s.current(), cursor: e.CursorOffset(),
		labels: spanLabels(text, w.sceneStarts())}
	sg := w.suggester()
	if o := sg.offersFor(s); o != nil && p < len(o.pieces) {
		c.headings = sg.headingsFor(o.places[p])
		c.draft = draftOf(sg, o.pieces[p])
		if key, _ := s.readingKey(); s.reading != nil && s.readingFor == key && p < len(s.reading.Passages) {
			c.opens = s.reading.Passages[p].SceneStart
		}
	}
	return c
}

// draftOf is a passage's pieces as the script's: a line under its cue
// (with how it is said), the narration as action.
func draftOf(sg *suggester, pieces []offerPiece) string {
	var parts []string
	for _, pc := range pieces {
		switch {
		case pc.Speech && pc.Speaker == "":
			parts = append(parts, castPlaceholder+"\n"+paren(pc.Manner)+pc.Text)
		case pc.Speech:
			parts = append(parts, sg.cueFor(pc.Speaker)+"\n"+paren(pc.Manner)+pc.Text)
		default:
			parts = append(parts, pc.Text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n")
}

// adaptPassage opens Adapt… for passage p.
func (s *sourcePane) adaptPassage(p int) {
	w := s.w
	if p < 0 || p >= len(s.passages) || !s.passages[p].counted() {
		w.setStatus("Pick a passage to adapt in the source first (Next finds the next one)")
		return
	}
	if s.current().status[p] == statusAdapted {
		w.setStatus("“" + shortQuote(s.passages[p].Plain) + "” is adapted already: the → under it goes to its scene")
		return
	}
	if s.adapting != nil {
		s.adapting.close()
	}
	d := &adaptDialog{s: s, p: p}
	d.ctx = s.adaptCtx(p)
	d.plan = planAdapt(d.ctx, p)
	d.build()
	s.adapting = d
	d.show()
	if d.ctx.draft == "" && len(d.ctx.headings) == 0 {
		d.waitForReading()
	}
}

func (d *adaptDialog) build() {
	pl := d.plan
	title := widget.NewLabelWithStyle("Adapt “"+shortQuote(d.s.passages[d.p].Plain)+"”", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	title.Wrapping = fyne.TextWrapWord
	d.how = newChoiceList("How", d.howItems(), pl.How)
	d.how.OnChange = func(int) { d.update(true) }
	d.how.OnEnter = d.accept
	d.where = newChoiceList("Where", d.whereItems(), pl.Where)
	d.where.OnChange = func(int) { d.update(true) }
	d.where.OnEnter = d.accept
	d.heading = widget.NewEntry()
	d.heading.SetText(pl.Headings[0])
	d.headings = newChoiceList("Suggested headings", headingItems(pl.Headings), 0)
	d.headings.OnChange = func(i int) { d.heading.SetText(d.plan.Headings[i]) }
	d.headings.OnEnter = d.accept
	setVisible(d.headings, len(pl.Headings) > 1)
	d.heading.OnChanged = func(string) { d.headingChanged() }
	d.heading.OnSubmitted = func(string) { d.accept() }
	fyne.SetAccessibleLabel(d.heading, "Scene heading")
	d.preview = widget.NewMultiLineEntry()
	d.preview.TextStyle = fyne.TextStyle{Monospace: true}
	d.preview.Wrapping = fyne.TextWrapWord
	d.preview.SetMinRowsVisible(5)
	d.preview.OnChanged = func(string) {
		if !d.setting {
			d.edited = true
		}
	}
	fyne.SetAccessibleLabel(d.preview, "What goes in the script")
	cancel := widget.NewButton("Cancel", d.close)
	d.ok = widget.NewButtonWithIcon("Adapt", theme.ConfirmIcon(), d.accept)
	d.ok.Importance = widget.HighImportance
	caption := func(t string) *widget.Label {
		l := widget.NewLabelWithStyle(t, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		l.SizeName = theme.SizeNameCaptionText
		return l
	}
	hint := widget.NewLabel("Enter takes the choices · ↑↓ choose · Tab goes on · Esc cancels")
	hint.SizeName = theme.SizeNameCaptionText
	hint.Importance = widget.LowImportance
	hint.Wrapping = fyne.TextWrapWord
	d.headRow = container.NewVBox(container.NewBorder(nil, nil, caption("Heading"), nil, d.heading), d.headings)
	d.whereBox = container.NewVBox(caption("Where"), d.where)
	d.wide = !d.s.w.narrow && d.s.w.fyneWindow.Canvas().Size().Width >= 940
	if d.wide { // side by side: how and the heading, where and what goes in, all in sight
		d.preview.SetMinRowsVisible(12)
		d.body = container.NewGridWithColumns(2, container.NewVBox(caption("How"), d.how, d.headRow),
			container.NewVBox(d.whereBox, caption("What goes in"), d.preview))
	} else {
		d.body = container.NewVBox(caption("How"), d.how, d.headRow, d.whereBox, caption("What goes in"), d.preview)
	}
	d.box = container.NewBorder(title, container.NewVBox(hint, container.NewHBox(layout.NewSpacer(), cancel, d.ok)), nil, nil,
		container.NewVScroll(d.body))
	d.update(false)
}

// headingItems are the suggested headings as choices, the first the
// likely.
func headingItems(hs []string) []choiceItem {
	var out []choiceItem
	for i, h := range hs {
		it := choiceItem{Label: h}
		if i == 0 {
			it.Note = "likely: the place and time the story reads"
		}
		out = append(out, it)
	}
	return out
}

// howItems are the ways, the likely one with its reason.
func (d *adaptDialog) howItems() []choiceItem {
	var out []choiceItem
	for _, h := range d.plan.Hows {
		note := h.Note
		if h.Reason != "" {
			note = "likely: " + h.Reason
		}
		out = append(out, choiceItem{Label: h.Label, Note: note, Off: h.Off})
	}
	return out
}

// whereItems are the places, the likely one with its reason.
func (d *adaptDialog) whereItems() []choiceItem {
	var out []choiceItem
	for i, o := range d.plan.Wheres {
		note := o.Note
		if i == d.plan.Where && o.Reason != "" {
			note += "\nlikely: " + o.Reason
		}
		out = append(out, choiceItem{Label: o.Label, Note: note})
	}
	return out
}

// howNow is the way chosen.
func (d *adaptDialog) howNow() adaptHow { return d.plan.Hows[d.how.Chosen()].How }

// update shows what the choices make: the heading and the places for a
// new scene, the preview made again (anew: what was written in it by
// hand goes), the action's name.
func (d *adaptDialog) update(anew bool) {
	how := d.howNow()
	scene := how == howNewScene || how == howDraft
	setVisible(d.headRow, scene)
	setVisible(d.whereBox, scene)
	labels := map[adaptHow]string{howNewScene: "Start the Scene", howDraft: "Put the Draft In", howAddHere: "Link Here",
		howCut: "Mark as Cut", howSkip: "Skip"}
	d.ok.SetText(labels[how])
	if anew || !d.edited {
		d.setPreview(d.block(d.ctx))
	}
	if scene {
		d.preview.Enable()
	} else {
		d.preview.Disable()
	}
	d.box.Refresh()
}

func (d *adaptDialog) setPreview(text string) {
	d.setting = true
	text, d.linksBack = readableLinks(text)
	d.preview.SetText(text)
	d.setting = false
	d.edited = false
	d.headingNow = strings.TrimSpace(d.heading.Text)
}

// headingChanged puts the heading in the preview: made again, or the
// old heading's line written over if the preview was written in.
func (d *adaptDialog) headingChanged() {
	h := strings.TrimSpace(d.heading.Text)
	if !d.edited {
		d.setPreview(d.block(d.ctx))
		return
	}
	lines := strings.Split(d.preview.Text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == d.headingNow && d.headingNow != "" {
			lines[i] = h
			d.setting = true
			d.preview.SetText(strings.Join(lines, "\n"))
			d.setting = false
			break
		}
	}
	d.headingNow = h
}

// placeNow is the place chosen, in plan pl (made again when the dialog
// is taken: the script may have changed meanwhile, a co-writer's edits).
func (d *adaptDialog) placeNow(pl adaptPlan) placeOption {
	want := d.plan.Wheres[d.where.Chosen()]
	for _, o := range pl.Wheres {
		if o.Where == want.Where {
			return o
		}
	}
	return pl.Wheres[0]
}

// block is what goes in for the choices, against ctx.
func (d *adaptDialog) block(c adaptCtx) string {
	switch d.howNow() {
	case howNewScene, howDraft:
		body := ""
		if d.howNow() == howDraft {
			body = c.draft
		}
		return sceneBlock(c, d.p, d.placeNow(planAdapt(c, d.p)).Line, d.heading.Text, body)
	case howAddHere:
		note, _ := linkNoteFor(matcherFor(c.ps), d.p, d.p)
		return note
	case howCut:
		return "(nothing in the scenes: the passage goes under cut: in the script's Adaptation block)"
	}
	return "(nothing: the passage stays to do, and the next one is picked)"
}

// waitForReading looks again for what the source suggests while a long
// one is read in the background (on the UI goroutine: a timer of the
// window's), and puts it in if nothing was chosen or written yet.
func (d *adaptDialog) waitForReading() {
	if backgroundAtOnce { // the tests read at once: nothing to wait for
		return
	}
	d.waits++
	if d.waits > 40 {
		return
	}
	d.s.w.after(250*time.Millisecond, func() {
		if d.s.adapting != d {
			return
		}
		c := d.s.adaptCtx(d.p)
		if c.draft == "" && len(c.headings) == 0 {
			d.waitForReading()
			return
		}
		if d.edited || d.how.Chosen() != d.plan.How || d.where.Chosen() != d.plan.Where {
			return // the writer is choosing: left alone
		}
		d.ctx, d.plan = c, planAdapt(c, d.p)
		d.how.SetItems(d.howItems(), d.plan.How)
		d.headings.SetItems(headingItems(d.plan.Headings), 0)
		setVisible(d.headings, len(d.plan.Headings) > 1)
		d.heading.SetText(d.plan.Headings[0])
		d.update(true)
	})
}

// show shows the dialog: in the middle on a desktop, a sheet at the
// bottom on a phone; the keyboard on How.
func (d *adaptDialog) show() {
	w := d.s.w
	c := w.fyneWindow.Canvas()
	d.pop = widget.NewModalPopUp(d.box, c)
	d.pop.OnEscape = func() bool {
		d.close()
		return true
	}
	size := c.Size()
	minH := d.box.MinSize().Height + d.body.MinSize().Height + 4*theme.Padding() // the scroll's content, all of it if it fits
	width, height := min(size.Width-2*theme.Padding(), 600), min(size.Height*0.85, max(minH, 420))
	if d.wide {
		width = min(size.Width-2*theme.Padding(), 900)
	}
	pos := fyne.NewPos((size.Width-width)/2, max((size.Height-height)/2, 0))
	if w.narrow {
		width, height = size.Width, min(size.Height*0.8, max(minH, 360))
		pos = fyne.NewPos(0, size.Height-height)
	}
	d.pop.Resize(fyne.NewSize(width, height))
	d.pop.Show()
	d.pop.Move(pos)
	c.Focus(d.how)
}

// close closes it, the keyboard back in the script.
func (d *adaptDialog) close() {
	if d.pop != nil {
		d.pop.Hide()
	}
	if d.s.adapting == d {
		d.s.adapting = nil
	}
	d.s.w.focusMain()
}

// accept does what the choices say.
func (d *adaptDialog) accept() {
	s, p := d.s, d.p
	how := d.howNow()
	heading := strings.TrimSpace(d.heading.Text)
	block, edited := d.preview.Text, d.edited
	if d.linksBack != nil {
		block = d.linksBack(block)
	}
	where := d.plan.Wheres[d.where.Chosen()].Where
	d.close()
	switch how {
	case howCut:
		s.setCut(p, true)
	case howSkip:
		s.skipPassage(p)
	case howAddHere:
		s.linkPassage(p)
	default:
		c := s.adaptCtx(p) // the script now
		pl := planAdapt(c, p)
		place := pl.Wheres[0]
		for _, o := range pl.Wheres {
			if o.Where == where {
				place = o
			}
		}
		if !edited { // made again for the script as it is now
			d.ctx = c
			block = d.block(c)
		}
		start, words := headingPlaceholder(s.w.sceneStarts())
		s.putScene(p, place, strings.TrimSpace(block), heading, heading == start+" "+words)
	}
}

// putScene puts a new scene for passage p in the script before the
// place's line: one undo step, an insertion only (and a cut entry for it
// taken away); with selectHeading the heading's words after INT. (or
// the placeholder's) selected to write over, else the cursor after what
// went in.
func (s *sourcePane) putScene(p int, place placeOption, block, heading string, selectHeading bool) {
	w := s.w
	e := w.textEditor
	m := s.current()
	ed := insertBlock(e.Text(), place.Line, block)
	uncut := m.status[p] == statusCut
	e.Edit(func(b *buffer.Buffer) {
		b.Group(func() {
			if uncut { // adapted now, so not cut: its entry goes (after the scene's place: done first)
				inside, _ := adaptationBlock(b.Text())
				start, end, blk := withAdaptationBlock(b.Text(), withCuts(inside, s.cutsWithout(m, p)))
				if start >= ed.At {
					b.Replace(start, end, blk)
				}
			}
			b.Replace(ed.At, ed.Replace, ed.Text)
			lead := len([]rune(ed.Text[:strings.Index(ed.Text, block)]))
			at := ed.At + lead
			h := strings.Index(block, heading)
			if sp := strings.IndexByte(heading, ' '); selectHeading && h >= 0 && sp > 0 {
				sel := at + len([]rune(block[:h+sp+1]))
				b.Select(sel, sel+len([]rune(heading[sp+1:])))
			} else {
				b.SetCursor(at+len([]rune(block)), false)
			}
		})
	})
	w.fyneWindow.Canvas().Focus(e)
	where := strings.TrimPrefix(place.Note, "at the end: no scene adapts a passage near it")
	if where == "" {
		where = strings.ToLower(place.Label)
	}
	w.setStatus("A scene for “" + shortQuote(s.passages[p].Plain) + "”, " + where + " (Undo takes it back)")
	s.follow()
	line, _ := e.Buffer().Position(e.CursorOffset())
	w.flashScene(line)
}

// skipPassage is Skip for now: the passage stays to do, the next one
// after it is picked.
func (s *sourcePane) skipPassage(p int) {
	m := s.current()
	next := m.step(anchor{passage: p, kind: anchorPicked}, 1)
	if next < 0 {
		next = m.wrapTo(1)
	}
	if next < 0 || next == p {
		s.w.setStatus("Skipped for now; no other passage is left to adapt")
		return
	}
	s.offer(next)
	s.w.setStatus("Skipped for now (still to do). Next: “" + shortQuote(s.passages[next].Plain) + "”")
	s.follow()
}

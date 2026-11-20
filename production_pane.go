package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
)

// The Production pane (docs/PRODUCTION.md §6, step 6): the Notes pane's
// third tab, the guided way through a shooting script's stages, as the
// source pane is through an adaptation. A strip of the stages (Lock,
// Revise, Break down, Schedule; Day pack to come), the current one
// highlighted, and for the stage shown its count line, its list (each
// row goes to its place in the script) and its buttons, which are the
// Production menu's actions: the menu stays, the pane is the way to them.
//
//   - Lock: the check before the lock (preLockChecks: headings without
//     INT./EXT. or a time, duplicate or missing numbers, a boneyard or
//     note not closed, one character under two cues, very long scenes, no
//     title page); Lock Scene Numbers, Lock Pages, Issue White….
//   - Revise: "Blue (open) · 6 pages changed · scenes 4, 12A, 17" and the
//     changed scenes; Issue Blue…, Revised Pages PDF…, Compare with the
//     last issue.
//
// The stage is Lock until the script has an issue (the drafts store's,
// not the production block's line, which Undo can take away), then
// Revise; either can be looked at again. The pane follows the text when
// the revision marks are made again (400 ms after typing pauses: never
// per key), and the changed pages are counted in the background, by the
// PDF's own pagination, only while Revise is shown.

// The stages.
const (
	stageLock = iota
	stageRevise
	stageBreakdown
	stageSchedule
)

type productionPane struct {
	w *MainWindow

	stages  [4]*widget.Button
	later   *widget.Label
	count   *widget.Label // the stage's count line
	rows    *fyne.Container
	scroll  *container.Scroll
	buttons *fyne.Container
	inList  *fyne.Container // the list and, on a phone, the buttons under it
	bottom  *fyne.Container // the buttons under the list, beside the script
	narrow  bool            // the buttons are in the list (a phone: the list has the room)

	lockNumbers, lockPages, issueWhite *widget.Button // Lock's
	issueNext, revisedPDF, compare     *widget.Button // Revise's
	bd                                 breakdownStage // Break down's
	sched                              scheduleStage  // Schedule's

	box *fyne.Container

	stage  int // the script's stage
	picked int // the stage looked at (-1: the script's)

	checks    []prodCheck // the last check's
	checksFor string      // the text they are of
	changed   []changedScene
	set       string // the current revision's set
	pages     changedPages
	pagesRun  int
	shownKey  string // what the list shows (the rows made again only when it changes)
	listView  int    // the stage the list was scrolled to the top for
}

// changedPages are the revised pages of a text, as the PDF prints them.
type changedPages struct {
	text, set string
	n         int
	err       error
	ok        bool
}

func newProductionPane(w *MainWindow) *productionPane {
	p := &productionPane{w: w, picked: -1, listView: -1}
	for i, name := range []string{"1 Lock", "2 Revise", "3 Break down", "4 Schedule"} {
		p.stages[i] = widget.NewButton(name, func() { p.pick(i) })
	}
	p.later = widget.NewLabel("5 Day pack: coming")
	p.later.Wrapping = fyne.TextWrapWord
	p.later.SizeName = theme.SizeNameCaptionText
	p.later.Importance = widget.LowImportance
	p.count = widget.NewLabel("")
	p.count.SizeName = theme.SizeNameCaptionText
	p.count.TextStyle.Bold = true
	p.count.Wrapping = fyne.TextWrapWord
	p.rows = container.NewVBox()
	p.inList = container.NewVBox(p.rows)
	p.scroll = container.NewVScroll(p.inList)

	button := func(label string, icon fyne.Resource, f func()) *widget.Button {
		b := widget.NewButtonWithIcon(label, icon, f)
		b.Alignment = widget.ButtonAlignLeading
		return b
	}
	p.lockNumbers = button("Lock Scene Numbers", theme.ListIcon(), w.lockSceneNumbers)
	p.lockPages = button("Lock Pages", theme.FileIcon(), w.lockPages)
	p.issueWhite = button("Issue White…", theme.ConfirmIcon(), w.showIssueDraft)
	p.issueWhite.Importance = widget.HighImportance
	p.issueNext = button("Issue Blue…", theme.ConfirmIcon(), w.showIssueDraft)
	p.issueNext.Importance = widget.HighImportance
	p.revisedPDF = button("Revised Pages PDF…", theme.DownloadIcon(), func() { w.showRevisedPagesExport(p.set) })
	p.compare = button("Compare with the Last Issue", theme.ViewRestoreIcon(), w.compareWithLastIssue)
	p.buttons = container.NewVBox()
	p.bottom = container.NewStack(p.buttons)
	p.bd.make(p)
	p.sched.make(p)

	strip := container.NewVBox(container.NewHBox(p.stages[0], p.stages[1], p.stages[2]),
		container.NewBorder(nil, nil, p.stages[3], nil, p.later))
	p.box = container.NewBorder(container.NewVBox(strip, widget.NewSeparator(), p.count), p.bottom, nil, nil, p.scroll)
	return p
}

// place puts the buttons under the list beside the script, and at the
// list's end on a phone, where the pane is half a screen: fixed under
// the list, they left it one row.
func (p *productionPane) place() {
	if p.w.narrow == p.narrow {
		return
	}
	p.narrow = p.w.narrow
	if p.narrow {
		p.bottom.Objects = nil
		p.inList.Objects = []fyne.CanvasObject{p.rows, p.buttons}
	} else {
		p.inList.Objects = []fyne.CanvasObject{p.rows}
		p.bottom.Objects = []fyne.CanvasObject{p.buttons}
	}
	p.inList.Refresh()
	p.bottom.Refresh()
}

// shown reports whether the pane is in sight.
func (p *productionPane) shown() bool {
	n := p.w.notes
	return n != nil && p.w.notesVisible && !p.w.markdown && n.tabs.Selected() == n.productionTab
}

// pick shows a stage (the script's own again: back to following it).
func (p *productionPane) pick(stage int) {
	p.picked = stage
	if stage == p.stage {
		p.picked = -1
	}
	p.shownKey = ""
	p.refresh()
}

// view is the stage shown.
func (p *productionPane) view() int {
	if p.picked >= 0 {
		return p.picked
	}
	return p.stage
}

// productionFollow has the pane follow the script, if it is shown: the
// revision marks were made again (after typing paused, or an issue).
func (w *MainWindow) productionFollow() {
	if w.notes != nil && w.notes.production.shown() {
		w.notes.production.refresh()
	}
}

// refresh shows the script's stage and the stage looked at.
func (p *productionPane) refresh() {
	if !p.shown() {
		return
	}
	w := p.w
	text := w.textEditor.Text()
	chain, err := w.revisionChainNow()
	issued := err == nil && chain != nil && len(chain.issues) > 0
	stage := stageLock
	if issued {
		stage = stageRevise
		if f := w.breakdownNow(); len(f.Tags) > 0 || len(f.Done) > 0 {
			stage = stageBreakdown // the breakdown begun
		}
	}
	if len(w.scheduleNow().Days) > 0 {
		stage = stageSchedule // a shoot day made
	}
	if stage != p.stage { // issued (or a new script): follow it
		p.stage, p.picked, p.shownKey = stage, -1, ""
	}
	p.place()
	p.showStrip(issued)
	switch p.view() {
	case stageLock:
		p.showLock(text, chain, issued)
	case stageRevise:
		p.showRevise(text, chain, issued)
	case stageBreakdown:
		p.showBreakdown(text)
	default:
		p.showSchedule(text)
	}
	// laid out again: a container's or a label's new height does not
	// reach the pane's layout by itself (the buttons went under the window)
	p.box.Refresh()
}

// showStrip shows the stages: the one looked at stands out, a done one
// has a tick.
func (p *productionPane) showStrip(issued bool) {
	names := []string{"Lock", "Revise", "Break down", "Schedule"}
	for i, b := range p.stages {
		imp := widget.LowImportance
		switch {
		case i == p.view():
			imp = widget.HighImportance
		case i == p.stage:
			imp = widget.MediumImportance
		}
		var icon fyne.Resource
		if i == stageLock && issued {
			icon = theme.ConfirmIcon()
		}
		what := names[i] + " stage"
		if i == p.stage {
			what += " (the script's)"
		}
		if b.Importance != imp || b.Icon != icon {
			b.Importance, b.Icon = imp, icon
			b.Refresh()
		}
		fyne.SetAccessibleLabel(b, what)
	}
}

// setButtons puts the stage's buttons under the list.
func (p *productionPane) setButtons(bs ...fyne.CanvasObject) {
	if len(p.buttons.Objects) == len(bs) {
		same := true
		for i := range bs {
			same = same && p.buttons.Objects[i] == bs[i]
		}
		if same {
			return
		}
	}
	p.buttons.Objects = bs
	p.buttons.Refresh()
}

// setRows puts the list's rows in, if what they show (key) changed.
func (p *productionPane) setRows(key string, make func() []fyne.CanvasObject) {
	if key == p.shownKey {
		return
	}
	p.shownKey = key
	p.rows.Objects = make()
	p.scroll.Refresh()                  // its content laid out again: rows of other heights
	if v := p.view(); v != p.listView { // another stage's list; an edit keeps the place
		p.listView = v
		p.scroll.ScrollToTop()
	}
}

// showLock is the Lock stage: the check and the lock's buttons.
func (p *productionPane) showLock(text string, chain *revisionChain, issued bool) {
	w := p.w
	starts := w.sceneStarts()
	if p.checks == nil || !sameString(p.checksFor, text) {
		p.checks, p.checksFor = preLockChecks(text, starts), text
		if p.checks == nil {
			p.checks = []prodCheck{}
		}
	}
	scenes := len(sceneHeads(text[:bodyEndByte(text)], starts))
	var parts []string
	if issued {
		first := chain.issues[0]
		parts = append(parts, "Locked "+first.set.Date.Format("2 Jan 2006"))
		if l := pageLockOf(text); l != nil {
			parts = append(parts, plural(len(l.lock.Pages), "page"))
		} else {
			parts = append(parts, "~"+plural(computeStats(text, 0, starts).Pages, "page"))
		}
		parts = append(parts, plural(scenes, "scene"))
	} else {
		parts = append(parts, "Not locked yet", plural(scenes, "scene"))
	}
	if n := problems(p.checks); n > 0 {
		parts = append(parts, fmt.Sprintf("%d to check", n))
	} else {
		parts = append(parts, "nothing to check")
	}
	p.count.SetText(strings.Join(parts, " · "))

	var key strings.Builder
	key.WriteString("lock\x00")
	for _, c := range p.checks {
		fmt.Fprintf(&key, "%d\x00%s\x00", c.line, c.text)
	}
	p.setRows(key.String(), func() []fyne.CanvasObject {
		out := []fyne.CanvasObject{caption("Before the lock")}
		if len(p.checks) == 0 {
			out = append(out, note("Nothing to check: every scene has INT./EXT. and a time, the numbers are unique, "+
				"every /* and [[ is closed, and no character has two cues."))
		}
		for _, c := range p.checks {
			c := c
			jump := func() { w.goToLine(c.line) }
			if c.kind == checkTitle {
				jump = w.showTitlePageDialog
			}
			out = append(out, p.newRow(c.text, nil, jump))
		}
		return out
	})

	locked := sceneLockOf(text).locked
	if locked {
		p.lockNumbers.SetText("Scene Numbers Locked")
		p.lockNumbers.Disable()
	} else {
		p.lockNumbers.SetText("Lock Scene Numbers")
		p.lockNumbers.Enable()
	}
	if pageLockOf(text) != nil {
		p.lockPages.SetText("Lock Pages Again")
	} else {
		p.lockPages.SetText("Lock Pages")
	}
	if issued {
		p.issueWhite.SetText("White Issued")
		p.issueWhite.Disable()
	} else {
		p.issueWhite.SetText("Issue White…")
		p.issueWhite.Enable()
	}
	p.setButtons(p.lockNumbers, p.lockPages, p.issueWhite)
}

// showRevise is the Revise stage: the current revision, its changed
// pages and scenes, and its buttons.
func (p *productionPane) showRevise(text string, chain *revisionChain, issued bool) {
	w := p.w
	if !issued {
		p.count.SetText("Nothing to revise yet: issuing White (the Lock stage) starts the revisions")
		p.set, p.changed = "", nil
		p.setRows("revise-none", func() []fyne.CanvasObject {
			return []fyne.CanvasObject{note("Once White is issued, writing goes on as usual: each change is marked in the margin " +
				"in the next colour (Blue), and listed here by scene until it is issued too.")}
		})
		p.setButtons()
		return
	}
	if !sameString(w.revisions.text, text) {
		if w.revisionTimer != nil {
			return // typing: the marks' timer brings them, after the pause
		}
		w.updateRevisionMarks() // follows again when they are made
		if !sameString(w.revisions.text, text) {
			return
		}
	}
	last := chain.issues[len(chain.issues)-1]
	open := !sameLines(bodyLines(text), last.lines)
	set := w.revisions.current.Name
	if set == "" {
		set = last.set.Name
	}
	p.set = set
	p.changed = changedScenes(text, w.sceneStarts(), w.revisions.marks, set)

	var line string
	switch {
	case open:
		line = set + " (open)"
	default:
		line = last.set.Name + " issued " + last.set.Date.Format("2 Jan 2006")
	}
	switch {
	case len(p.changed) == 0:
		line += " · no changes since"
		if len(chain.issues) >= 2 && !open {
			line = last.set.Name + " issued " + last.set.Date.Format("2 Jan 2006") + " · no changes in it"
		}
	default:
		counted := func() bool { return p.pages.ok && sameString(p.pages.text, text) && p.pages.set == set }
		if !counted() {
			p.countPages(text, set) // the tests' is done at once
		}
		pages := "counting pages…"
		if counted() {
			pages = plural(p.pages.n, "page") + " changed"
			if p.pages.err != nil {
				pages = "pages not counted"
			}
		}
		line += " · " + pages + " · " + sceneList(p.changed)
	}
	p.count.SetText(line)

	ink := revisionInk(set)
	var key strings.Builder
	key.WriteString("revise\x00" + set + "\x00")
	for _, c := range p.changed {
		fmt.Fprintf(&key, "%d\x00%s\x00", c.line, c.label)
	}
	p.setRows(key.String(), func() []fyne.CanvasObject {
		out := []fyne.CanvasObject{caption("Changed in " + set)}
		if len(p.changed) == 0 {
			out = append(out, note("No changes since "+last.set.Name+". Writing goes on as usual: each change is marked "+
				chain.nextSet(time.Now()).Name+" in the margin, and its scene listed here."))
		}
		for _, c := range p.changed {
			c := c
			label := c.label
			if c.heading != "" && c.heading != c.label {
				label += "  " + c.heading
			}
			out = append(out, p.newRow(label, ink, func() { w.goToLine(c.line) }))
		}
		return out
	})

	next := chain.nextSet(time.Now()).Name
	p.issueNext.SetText("Issue " + next + "…")
	if open {
		p.issueNext.Enable()
	} else {
		p.issueNext.Disable()
	}
	if len(p.changed) > 0 {
		p.revisedPDF.Enable()
	} else {
		p.revisedPDF.Disable()
	}
	p.compare.SetText("Compare with " + last.set.Name)
	p.setButtons(p.issueNext, p.revisedPDF, p.compare)
}

// countPages counts the revised pages of text in the background, as the
// PDF of revised pages only prints them, then shows them.
func (p *productionPane) countPages(text, set string) {
	p.pagesRun++
	run := p.pagesRun
	job := p.w.revisedPagesJob(text)
	chain, _ := p.w.revisionChainNow()
	inBackground(func() changedPages {
		if chain != nil && len(chain.issues) > 0 { // the current revision's marks (a diff: here, not on the UI goroutine)
			marks, sets, _ := chain.marksOf(text, false, time.Now())
			r := *job.revisions
			r.Sets = sets
			job.revisions, job.marks = &r, marks
		}
		return revisedPageCount(text, set, job)
	}, func(r changedPages) {
		if run != p.pagesRun {
			return // a later text's are coming
		}
		p.pages = r
		p.refresh()
	})
}

// revisedPageCount counts the pages of text that job prints as revised.
func revisedPageCount(text, set string, job exportJob) (r changedPages) {
	r = changedPages{text: text, set: set, ok: true}
	defer func() { // Lexington failing is a count not made, not the app's end
		if e := recover(); e != nil {
			r.err = fmt.Errorf("the pages could not be laid out: %v", e)
		}
	}()
	job = withLockedPages(text, "pdf", job)
	pages, err := cli.Pages(job.script(text), job.options("pdf", ""))
	if err != nil {
		r.err = err
		return r
	}
	for _, pg := range pages {
		if pg.Revision != "" {
			r.n++
		}
	}
	return r
}

// revisedPagesJob is the PDF of the text's current revision, in the
// pages it prints against: its marks (the current revision's, whatever
// the margin shows) are put in by countPages, in the background.
// Revised Pages PDF… prints the same with the export's setting.
func (w *MainWindow) revisedPagesJob(text string) exportJob {
	j := w.savedExportJob()
	j.revisions = &lex.Revisions{}
	if w.settings != nil {
		j.revisions.Header = w.settings.GetString(revisionHeaderPref)
		j.revisions.DateLayout = w.settings.GetString(revisionDatePref)
	}
	return w.withIssuedPages(j, text)
}

// goToLine puts the cursor at the start of a line (the last if the text
// has fewer now) and the focus in the editor.
func (w *MainWindow) goToLine(line int) {
	e := w.textEditor
	e.Navigate(func(b *buffer.Buffer) {
		n, _ := b.Position(b.Len())
		b.SetCursor(b.LineStart(max(0, min(line, n))), false)
	})
	w.fyneWindow.Canvas().Focus(e)
}

// showRevisedPagesExport is the Export dialog set up for the current
// revision's pages: PDF, revision marks, revised pages only.
func (w *MainWindow) showRevisedPagesExport(set string) *ExportDialog {
	if w.markdown || w.textEditor == nil {
		return nil
	}
	ed := NewExportDialog(w)
	ed.formatSelect.SetSelected(exportFormatLabel("pdf"))
	ed.revisionsCheck.SetChecked(true)
	ed.revisedOnlyCheck.Enable()
	ed.revisedOnlyCheck.SetChecked(true)
	if set != "" {
		name := strings.TrimSuffix(ed.filenameEntry.Text, ".pdf")
		ed.filenameEntry.SetText(name + "-" + strings.ToLower(strings.ReplaceAll(set, " ", "-")) + "-pages.pdf")
	}
	ed.Show()
	return ed
}

// compareWithLastIssue is Drafts' Compare, of the last issue with the
// script now.
func (w *MainWindow) compareWithLastIssue() {
	if w.markdown || w.textEditor == nil {
		return
	}
	w.afterBundleWrites(func() {
		ds, err := readDrafts(w.extrasHome())
		if err != nil {
			dialog.ShowError(err, w.fyneWindow)
			return
		}
		last := ""
		for _, d := range ds {
			if d.Kind == kindIssue {
				last = d.ID
			}
		}
		if last == "" {
			w.setStatus("Nothing issued yet: Issue White first")
			return
		}
		showCompare(w, ds, last, nowID)
	})
}

// toggleProduction is View > Production: the Notes pane with the
// production stages in it, or (if that is shown) neither.
func (w *MainWindow) toggleProduction() {
	if w.markdown {
		return
	}
	if w.notes == nil {
		w.notes = newNotesPanel(w)
	}
	p := w.notes.production
	if p.shown() {
		w.toggleNotes()
		return
	}
	w.notes.tabs.Select(w.notes.productionTab)
	if !w.notesVisible {
		w.toggleNotes()
	}
	w.fyneWindow.Canvas().Focus(p.focusTarget())
	p.refresh()
}

// focusTarget is what takes the keyboard in the pane: the stage shown.
func (p *productionPane) focusTarget() fyne.Focusable { return p.stages[p.view()] }

// caption is a small heading over the list.
func caption(s string) *widget.Label {
	l := widget.NewLabelWithStyle(s, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	l.SizeName = theme.SizeNameCaptionText
	return l
}

// note is a line of explanation in the list.
func note(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	l.SizeName = theme.SizeNameCaptionText
	l.Importance = widget.LowImportance
	return l
}

// newRow is a row of the pane's list: a colour (nil none), its text,
// wrapped, which a tap takes to its place, and a button that does the
// same for the keyboard (scrolled into sight when Tab reaches it).
func (p *productionPane) newRow(text string, ink color.Color, tap func()) fyne.CanvasObject {
	label := newTapLabel(text, tap)
	var row *fyne.Container
	b := &goButton{}
	b.Icon, b.OnTapped, b.Importance = theme.NavigateNextIcon(), tap, widget.LowImportance
	b.ExtendBaseWidget(b)
	b.onFocus = func() { p.scrollTo(row) }
	fyne.SetAccessibleLabel(b, "Go to: "+text)
	var left fyne.CanvasObject
	if ink != nil {
		bar := canvas.NewRectangle(ink)
		bar.SetMinSize(fyne.NewSize(4, 0))
		left = bar
	}
	row = container.NewBorder(nil, nil, left, b, label)
	return row
}

// scrollTo scrolls the list so that the row is in sight.
func (p *productionPane) scrollTo(row fyne.CanvasObject) {
	y, h, view := row.Position().Y, row.Size().Height, p.scroll.Size().Height
	off := p.scroll.Offset.Y
	switch {
	case y < off:
		off = y
	case y+h > off+view:
		off = y + h - view
	default:
		return
	}
	p.scroll.Offset = fyne.NewPos(0, off)
	p.scroll.Refresh()
}

// goButton is a row's button: it says when Tab reaches it.
type goButton struct {
	widget.Button
	onFocus func()
}

func (b *goButton) FocusGained() {
	b.Button.FocusGained()
	if b.onFocus != nil {
		b.onFocus()
	}
}

// tapLabel is a wrapping label a tap acts on.
type tapLabel struct {
	widget.Label
	onTap func()
}

func newTapLabel(text string, tap func()) *tapLabel {
	l := &tapLabel{onTap: tap}
	l.Text = text
	l.Wrapping = fyne.TextWrapWord
	l.SizeName = theme.SizeNameCaptionText
	l.ExtendBaseWidget(l)
	return l
}

func (l *tapLabel) Tapped(*fyne.PointEvent) { l.onTap() }

// Cursor is a pointer: the row can be tapped.
func (l *tapLabel) Cursor() desktop.Cursor { return desktop.PointerCursor }

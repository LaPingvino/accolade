package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Swap with Main (LOOP.md, Joop 2026-11-21: "make it all equivalent? you
// have move to top and move to bottom … a swap with main added to
// that"): a dock pane's document or view goes into the main area, and
// the script takes the pane's place in the dock, as its Script tab.
// The source becomes a Markdown document in the main editor (its
// commands, Live Preview, vim), the notes too; the cards their full
// view; any other pane its box. Swapping again (the Script tab's menu,
// the pane's View command, Adaptation > Swap Source and Script for the
// common case) puts them back. One pane at a time: swapping another one
// puts the first back first. docs/SWAP.md has what changed for it.

// swapWithMain swaps the pane with id and the main area, or back if it
// is swapped already.
func (w *MainWindow) swapWithMain(id string) {
	if w.swapped == id {
		w.swapBack()
		return
	}
	if why := w.swapBlocked(id); why != "" {
		w.setStatus(why)
		return
	}
	if w.swapped != "" {
		w.swapBack()
	}
	var doc *document
	switch id {
	case "source":
		doc = w.sourceDoc()
		if s := w.source; s != nil && s.editing { // the pane's Edit: the same document, here now
			s.editing = false
			s.edit.SetText("Edit")
			s.edit.SetIcon(theme.DocumentCreateIcon())
			s.showBody() // its editor out of the pane: one place at a time
		}
		if s := w.source; s != nil {
			s.measuring++ // its rows out of sight: measured again when it is back
		}
	case "notes":
		doc = w.notesDocument()
	case "cards":
		if w.cards == nil {
			w.cards = newCardsView(w)
		}
		w.cards.update(w.textEditor.Text())
	}
	if w.stripsVisible { // one view in the main area
		w.stripsVisible = false
	}
	if w.searchBar != nil {
		w.textEditor.SetHighlights(nil)
	}
	d := w.dock
	d.on[id] = true // the Script tab is in its place
	d.setFront(id)
	w.swapped, w.mainDoc = id, doc
	w.textEditor.SetOffstage(true) // out of sight: laid out again when back (internal/editor)
	w.cardsVisible = id == "cards"
	w.relayout()
	w.editorKindChanged()
	w.scriptPane().refresh()
	w.focusMain()
	w.setStatus(w.swapStatus())
}

// swapSourceAndScript is Adaptation > Swap Source and Script: the Source
// pane brought in if it is not there, and swapped with the main area (or
// back).
func (w *MainWindow) swapSourceAndScript() {
	if w.swapped != "source" && !w.paneShown("source") {
		w.showPane("source", false)
	}
	w.swapWithMain("source")
}

// swapBack puts the script back in the main area and the pane in its
// place: a document written to its home and let go.
func (w *MainWindow) swapBack() {
	id, doc := w.swapped, w.mainDoc
	if id == "" {
		return
	}
	if doc != nil {
		w.offerLinkUpdate(doc) // a passage's opening changed: the links quoting it
		if err := w.closeDoc(doc); err != nil {
			dialog.ShowError(err, w.fyneWindow)
		}
	}
	w.swapped, w.mainDoc = "", nil
	w.textEditor.SetOffstage(false)
	w.cardsVisible = false
	if !w.markdown {
		w.previewVisible = false // a screenplay's preview is a PDF
	}
	if w.searchBar != nil && doc != nil {
		doc.ed.SetHighlights(nil)
	}
	w.relayout()
	w.editorKindChanged()
	if p := w.dock.pane(id); p != nil && p.edited != nil && w.dock.shown(id) && doc != nil {
		p.edited(w, w.textEditor.Text()) // the script's changes while it was away
	}
	if p := w.dock.pane(id); p != nil && p.shown != nil && w.dock.inSight(id) {
		p.shown(w)
	}
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

// swapBlocked is why the pane with id cannot be swapped with the main
// area now ("" it can).
func (w *MainWindow) swapBlocked(id string) string {
	p := w.dock.pane(id)
	if p == nil || !w.dock.applies(p) {
		return "That pane is not there for this document"
	}
	switch id {
	case "source":
		if w.sourceRef().Origin == sourceNone {
			return "No source yet: Adaptation > Choose Source… first"
		}
	case "cards":
		if w.markdown {
			return "Cards are a screenplay's scenes"
		}
	}
	return ""
}

// swapStatus is what the status line says after a swap.
func (w *MainWindow) swapStatus() string {
	what := w.dock.pane(w.swapped).cmd.label
	msg := what + " in the main area, the script in its place in the side panes: Swap with Main again puts them back"
	if d := w.mainDoc; d != nil {
		msg = what + " in the main editor (" + d.where() + "); the script is the Script tab: Swap with Main again puts them back"
		if w.shared != nil && d.sharedAs() == "" {
			msg += ". Editing together: the source is yours only, the others do not see its changes (it is not shared yet)"
		}
	}
	return msg
}

// swappedCenter is what the main area shows while a pane is swapped in.
func (w *MainWindow) swappedCenter() fyne.CanvasObject {
	if d := w.mainDoc; d != nil {
		if w.previewVisible && w.previewPane != nil {
			split := container.NewHSplit(d.view, w.previewPane.box)
			split.Offset = 0.55
			return split
		}
		return d.view
	}
	if w.swapped == "cards" && w.cards != nil {
		return w.cards.box
	}
	if p := w.dock.pane(w.swapped); p != nil {
		return p.box(w)
	}
	return w.editorView
}

// swappedIsView reports whether the pane swapped in shows its own box
// in the main area (the outline, help…), not a document or the cards'
// full view.
func (w *MainWindow) swappedIsView() bool {
	return w.swapped != "" && w.mainDoc == nil && w.swapped != "cards"
}

// editorKindChanged follows the main editor's kind: the menu (Markdown's
// commands or the screenplay's), the editing row, the ribbon, the status
// bar.
func (w *MainWindow) editorKindChanged() {
	if w.headerBar != nil {
		w.headerBar.showEditingRow(w.editorMarkdown(), w.mainDoc != nil && !w.markdown)
	}
	w.refreshMenu()
	w.ribbonRefresh()
	if w.help != nil {
		w.help.fillPicker() // the cheat sheets in the main editor's order
	}
	w.updateTitle()
	w.updateCurrentElement()
	w.updateStatusBar()
	if w.previewVisible && w.previewPane != nil {
		w.previewPane.update(w.writeEditor().Text())
	}
}

// mainChanged is the text in the main editor changing, when it is not
// the script (whose changes onTextChanged follows).
func (w *MainWindow) mainChanged() {
	text := w.mainEditor().Text()
	if w.previewVisible && w.previewPane != nil {
		w.previewPane.update(text)
	}
	if w.searchBar != nil && w.searchBar.IsVisible() {
		w.searchBar.updateSearch()
	}
	w.updateCurrentElement()
	w.updateStatusBar()
	if w.swapped == "source" { // the Script tab, once typing pauses
		w.debounce(&w.followTimer, followPause, func() { w.scriptPane().follow() })
	}
	if d := w.mainDoc; d != nil && !d.inScript() {
		w.scheduleAutoSave()
	}
}

// mainCursorMoved is the main editor's cursor moving (not the script's).
func (w *MainWindow) mainCursorMoved() {
	w.updateCurrentElement()
	w.updateStatusBar()
	if w.followTimer == nil { // the text as followed: at once
		w.scriptPane().follow()
		if w.cardsDock != nil && w.paneInSight("cards") {
			w.cardsDock.follow() // the passage's scenes' cards
		}
	}
	w.linksOfferOnLeave()
}

// goToScript puts the script's cursor at the start of a line: the
// keyboard in the script if it is in the main editor, else the Script tab
// lights its scene (the outline, the cards, the changes, Production).
func (w *MainWindow) goToScript(line int) {
	e := w.textEditor
	e.Navigate(func(b *buffer.Buffer) {
		n, _ := b.Position(b.Len())
		b.SetCursor(b.LineStart(max(0, min(line, n))), false)
	})
	if w.scriptInMain() {
		w.fyneWindow.Canvas().Focus(e)
		return
	}
	w.scriptPane().follow()
}

// scriptAway says why a command that works at the script's cursor does
// nothing while the script is not in the main editor; false if it is.
func (w *MainWindow) scriptAway(what string) bool {
	if w.scriptInMain() {
		return false
	}
	w.setStatus(what + " works in the script: Swap with Main puts it back in the main editor first")
	return true
}

// The Script tab: the script in the dock while a pane has its place,
// one row a scene or section. With the source in the main editor it
// follows that cursor the other way round: the scenes linking to the
// passage there lit, the passage's status above them, a tap going to a
// scene's passage (and the script's cursor to the scene). Otherwise it
// lights the script cursor's scene.

type scriptPane struct {
	w     *MainWindow
	items []outlineItem
	lit   map[int]bool // the items lit
	seen  linesSeen
	info  *widget.Label
	list  *widget.List
	box   fyne.CanvasObject
}

func (w *MainWindow) scriptPane() *scriptPane {
	if w.scriptTab == nil {
		w.scriptTab = newScriptPane(w)
	}
	return w.scriptTab
}

func newScriptPane(w *MainWindow) *scriptPane {
	sp := &scriptPane{w: w, lit: map[int]bool{}}
	sp.info = widget.NewLabel("")
	sp.info.Wrapping = fyne.TextWrapWord
	sp.info.SizeName = theme.SizeNameCaptionText
	sp.list = widget.NewList(
		func() int { return len(sp.items) },
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			bg.CornerRadius = 4
			title := widget.NewLabel("")
			title.Truncation = fyne.TextTruncateEllipsis
			return container.NewStack(bg, container.NewBorder(nil, nil, nil, container.NewCenter(newSwatch()), title))
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(sp.items) {
				return
			}
			it := sp.items[id]
			row := o.(*fyne.Container)
			bg := row.Objects[0].(*canvas.Rectangle)
			inner := row.Objects[1].(*fyne.Container)
			title := inner.Objects[0].(*widget.Label)
			showSwatch(inner.Objects[1].(*fyne.Container).Objects[0].(*canvas.Rectangle), it.Color)
			title.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Text)
			title.TextStyle = fyne.TextStyle{Bold: !it.Scene || sp.lit[id]}
			title.Refresh()
			bg.FillColor = color.Transparent
			if sp.lit[id] {
				bg.FillColor = tint(theme.ColorNamePrimary, 0x40)
			}
			bg.Refresh()
		},
	)
	fyne.SetAccessibleLabel(sp.list, "Script scenes")
	sp.list.OnSelected = func(id widget.ListItemID) {
		sp.list.UnselectAll()
		sp.tap(id)
	}
	width := canvas.NewRectangle(color.Transparent)
	width.SetMinSize(fyne.NewSize(240, 0))
	sp.box = container.NewStack(width, container.NewBorder(sp.info, nil, nil, nil, sp.list))
	return sp
}

// refresh reads the script's scenes again and follows.
func (sp *scriptPane) refresh() {
	sp.items = sp.w.outlineItems(sp.w.textEditor.Text())
	sp.follow()
}

// shown reports whether the Script tab is in sight.
func (sp *scriptPane) shown() bool {
	w := sp.w
	return w.swapped != "" && w.dock.inSight(w.swapped)
}

// follow lights what goes with the main cursor.
func (sp *scriptPane) follow() {
	w := sp.w
	cancel(&w.followTimer)
	if w.swapped == "" {
		return
	}
	lit, info, ok := w.litScenes(sp.items, &sp.seen)
	if !ok {
		sp.setInfo(info)
		return
	}
	sp.setInfo(info)
	if !sameSet(lit, sp.lit) {
		sp.lit = lit
		sp.list.Refresh()
		for i := range sp.items {
			if lit[i] {
				sp.list.ScrollTo(i)
				break
			}
		}
	}
}

// litScenes are the items to light for the main cursor: with the source
// in the main editor, the scenes linking to the passage there (and what
// is said of it); else the script cursor's scene. ok is false while the
// source is being read.
func (w *MainWindow) litScenes(items []outlineItem, seen *linesSeen) (lit map[int]bool, info string, ok bool) {
	lit = map[int]bool{}
	if d := w.mainDoc; d != nil && d.kind == docSource {
		s := w.sourcePane()
		s.loadSource(true)
		if s.loading != nil {
			return nil, "Reading the source…", false
		}
		m := s.current()
		p := passageAt(d.Text(), m.passages, d.ed.Buffer(), seen)
		if p < 0 {
			info = "Before the source's first passage"
		} else {
			lines := m.linked(p)
			for i, it := range items {
				for _, l := range lines {
					if it.Line == l {
						lit[i] = true
					}
				}
			}
			q := "“" + shortQuote(m.passages[p].Plain) + "”"
			switch st := statusOf(m, p); {
			case len(lines) > 0:
				info = q + ": " + plural(len(lines), "scene") + " from it"
			case st == statusCut:
				info = q + ": cut on purpose"
			case st == statusNone:
				info = q
			default:
				info = q + ": not adapted yet"
			}
		}
	} else {
		line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
		at := -1
		for i, it := range items {
			if it.Line <= line {
				at = i
			}
		}
		if at >= 0 {
			lit[at] = true
			info = "The cursor's scene: " + items[at].Text
		}
	}
	return lit, info, true
}

func (sp *scriptPane) setInfo(text string) {
	if sp.info.Text != text {
		sp.info.SetText(text)
		if text == "" {
			sp.info.Hide()
		} else {
			sp.info.Show()
		}
	}
}

// tap goes to a scene: the script's cursor there, and with the source in
// the main editor, its cursor to the passage the scene links to first.
func (sp *scriptPane) tap(id int) {
	if id < 0 || id >= len(sp.items) {
		return
	}
	w := sp.w
	line := sp.items[id].Line
	w.goToScript(line)
	d := w.mainDoc
	if d == nil || d.kind != docSource {
		return
	}
	m := w.sourcePane().current()
	first := -1
	for _, i := range m.inSpan(line) {
		if mt := m.matches[i]; mt.found() && (first < 0 || mt.Passages[0] < first) {
			first = mt.Passages[0]
		}
	}
	if first < 0 {
		w.setStatus(sp.items[id].Text + ": no passage linked")
		return
	}
	starts := passageLines(d.Text(), m.passages)
	if l := starts[first]; l >= 0 {
		d.ed.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(l), false) })
		w.fyneWindow.Canvas().Focus(d.ed)
	}
	w.setStatus(fmt.Sprintf("%s: its passage", sp.items[id].Text))
}

// statusOf is passage p's status in the model.
func statusOf(m *sourceModel, p int) passageStatus {
	if p >= 0 && p < len(m.status) {
		return m.status[p]
	}
	return statusNone
}

// passageLines are the lines of a source's Markdown each passage starts
// on (-1: not found). passagesOf keeps no places, so each passage is
// looked for by its first words, after the one before it: on the first
// line from there whose words hold them (a list's mark or number, a
// heading's #, a quote's > before them).
func passageLines(src string, ps []passage) []int {
	lines := strings.Split(src, "\n")
	body := markdownBody(src)
	from := len(lines) - len(strings.Split(body, "\n")) // the front matter's lines
	end := strings.Count(src[:markdownNotes.startByte(src)], "\n") + 1
	out := make([]int, len(ps))
	at := max(0, from)
	for i, p := range ps {
		out[i] = -1
		if len(p.Words) == 0 {
			continue
		}
		want := p.Words[:min(3, len(p.Words))]
		for l := at; l < min(end, len(lines), at+passageReach); l++ {
			if opensWith(normWords(lines[l]), want) {
				out[i], at = l, l+1
				break
			}
		}
	}
	return out
}

// passageReach is how many lines on a passage is looked for: one not
// found there (its words changed) is left out, the next looked for from
// where the one before it was.
const passageReach = 400

// opensWith reports whether want comes in ws, at its start or after a
// mark or two (a list's number).
func opensWith(ws, want []string) bool {
	for skip := 0; skip <= 2 && skip+len(want) <= len(ws); skip++ {
		ok := true
		for j, x := range want {
			if ws[skip+j] != x {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// linesSeen are the passages' lines found last (the cursor moves often,
// the text not).
type linesSeen struct {
	rev   uint64
	buf   *buffer.Buffer
	ps    []passage
	lines []int
}

// passageAt is the passage the cursor of a source's editor is in: the
// last one starting on its line or before (-1: before the first).
func passageAt(src string, ps []passage, b *buffer.Buffer, c *linesSeen) int {
	line, _ := b.Position(b.Cursor())
	at := -1
	if c.rev != b.Rev() || c.buf != b || !samePassages(c.ps, ps) {
		c.rev, c.buf, c.ps, c.lines = b.Rev(), b, ps, passageLines(src, ps)
	}
	for i, l := range c.lines {
		if l >= 0 && l <= line {
			at = i
		}
	}
	return at
}

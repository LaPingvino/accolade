// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The middle tabs (LOOP.md, Joop 2026-11-22: "make the cards and script
// tab like the sidebar when in the middle? And maybe we can use a similar
// approach for the source, enable them to be active in the middle at the
// same time with this tabbing"): the middle of the window is a row of
// tabs like the dock's. The script is always its first tab. Cards (View >
// Cards) opens as a tab there, and any dock pane goes there with its tab's
// Open in the Middle: the source and the notes as Markdown documents in an
// editor of their own, the cards as their full view, any other pane as its
// box. Several are open at once, one shown; a pane is in the dock or in the
// middle, never both. A middle tab's menu has Back to the Side and Close.
// The row is there only while a tab beside the script's is open.
//
// Every change of what the middle shows goes through show, which alone
// brings the views derived from it along: the preview (the script's pages,
// a document's Live Preview), the editor kept off stage, the find bar's
// highlights, the menus, the ribbon's contextual tabs, the status, the
// focus. docs/MIDDLE.md has the model and what it replaced (Swap with
// Main).

// middle is the window's middle tabs.
type middle struct {
	w *MainWindow
	// tabs are the panes open in the middle, in order, after the script
	tabs []string
	// shown is the tab in front: "" the script, else a pane's id
	shown string
	// docs are the documents the tabs edit (the source, the notes)
	docs map[string]*document
	// keepPreview: a preview was shown when a tab without one (the cards,
	// the outline) came to the front; the next editor tab gets its own
	keepPreview bool
	// as last built: what the tab in front shows, the tabs (by id, "" the
	// script), the row
	body fyne.CanvasObject
	tabW map[string]*dockTab
	row  *overflowRow
	shut *tooltip.Button
	// showing: show is under way (hidePages, called from it, leaves the
	// tab in front to it)
	showing bool
	// pagesBack: the script's pages being made again for the script's tab
	// coming back (previewFollowsMain): on a phone their tab comes in
	// behind the script's, not in front of it
	pagesBack bool
}

// midView is a middle tab that is no dock pane, with no place in the side
// panes: the stripboard (too wide for them; the Production pane's
// Schedule stage is its side), and on a phone the script's pages. Its
// tab's menu has Close alone.
type midView struct {
	title   string
	applies func(w *MainWindow) bool // there for the document and role now
	why     string                   // the status when it is not
	box     func(w *MainWindow) fyne.CanvasObject
	focus   func(w *MainWindow) fyne.Focusable // nil: its tab
	shown   func(w *MainWindow)                // coming to the front
	closed  func(w *MainWindow)                // its tab closed
}

// middleViews are the middle tabs that are no dock pane, by id (made in
// init: they call into the window's code that calls back here).
var middleViews map[string]*midView

func init() { middleViews = makeMiddleViews() }

// middleView is the middle tab id that is no dock pane, nil if it is none.
func middleView(id string) *midView { return middleViews[id] }

func makeMiddleViews() map[string]*midView {
	return map[string]*midView{
		// the stripboard (stripboard.go): Production > Stripboard, View >
		// Stripboard, the Schedule stage's Open Stripboard and its rows
		"strips": {title: "Stripboard", why: "The stripboard is a screenplay's scenes, in the Production role",
			applies: func(w *MainWindow) bool { return !w.markdown && !w.writerView() },
			box:     func(w *MainWindow) fyne.CanvasObject { return w.stripsBoard().box },
			shown: func(w *MainWindow) {
				sb := w.stripsBoard()
				sb.shownKey = ""
				sb.update()
				w.productionFollow()
			},
			closed: func(w *MainWindow) { w.productionFollow() }},
		// Compare (drafts_view.go): two drafts, or one and the script now;
		// the ribbon's Compare tab while in front (compare_tab.go)
		"compare": {title: "Compare", applies: func(w *MainWindow) bool { return w.compare != nil },
			why: "Nothing to compare: Collaboration > Compare…",
			box: func(w *MainWindow) fyne.CanvasObject {
				if w.compare == nil {
					return layout.NewSpacer()
				}
				w.compare.follow()
				return w.compare.box
			},
			closed: func(w *MainWindow) { w.compare = nil }},
		// on a phone, the page view (pageview_window.go): beside the script's
		// tab, so the way back to the script is a tap
		// the Table Read, in the browser and on a phone (tableread_view.go)
		"tableread": tableReadMidView(),
		"pages": {title: "Pages", applies: func(*MainWindow) bool { return true },
			box: func(w *MainWindow) fyne.CanvasObject {
				if w.pages == nil {
					return layout.NewSpacer()
				}
				return w.pages
			},
			focus: func(w *MainWindow) fyne.Focusable {
				if w.pages == nil {
					return nil
				}
				return w.pages.area
			},
			closed: func(w *MainWindow) { w.hidePages() }},
	}
}

// has reports whether the pane with id is a middle tab.
func (m *middle) has(id string) bool { return id != "" && slices.Contains(m.tabs, id) }

// boxShown reports whether pane p is in the middle as its own box (the
// outline, help…), not a document or full view.
func (m *middle) boxShown(p *dockPane) bool {
	return m.has(p.id) && m.docs[p.id] == nil && p.full == nil
}

// ids are the tabs' ids in order, "" the script's first.
func (m *middle) ids() []string { return append([]string{""}, m.tabs...) }

// title is tab id's title.
func (m *middle) title(id string) string {
	if p := m.w.dock.pane(id); p != nil {
		return p.cmd.label
	}
	if v := middleView(id); v != nil {
		return v.title
	}
	return "Script"
}

// main is the document in the main editor: the script, or the source or
// the notes in front in the middle. A view in front (the cards, the
// outline…) is no document: nil.
func (w *MainWindow) main() *document {
	if w.mid.shown == "" {
		return w.scriptDoc()
	}
	return w.mid.docs[w.mid.shown]
}

// midDoc is the document in front in the middle other than the script
// (nil: the script, or a view).
func (w *MainWindow) midDoc() *document { return w.mid.docs[w.mid.shown] }

// scriptInMain reports whether the script is the middle tab in front.
func (w *MainWindow) scriptInMain() bool { return w.mid.shown == "" }

// cardsShown reports whether the cards' full view is in front.
func (w *MainWindow) cardsShown() bool { return w.mid.shown == "cards" }

// cardsFull is the cards' full view, made at the first need.
func (w *MainWindow) cardsFull() *cardsView {
	if w.cards == nil {
		w.cards = newCardsView(w)
	}
	return w.cards
}

// middleBlocked is why the pane with id cannot go in the middle now (""
// it can).
func (w *MainWindow) middleBlocked(id string) string {
	if v := middleView(id); v != nil {
		if !v.applies(w) {
			return v.why
		}
		return ""
	}
	p := w.dock.pane(id)
	if p == nil || !w.dock.applies(p) {
		return "That pane is not there for this document"
	}
	if id == "source" && w.sourceRef().Origin == sourceNone {
		return "No source yet: Adaptation > Choose Source… first"
	}
	return ""
}

// openMiddle opens the pane with id as a middle tab (out of the dock if it
// is there) and shows it; already open, it shows it.
func (w *MainWindow) openMiddle(id string) {
	m := &w.mid
	if m.has(id) {
		w.showMiddle(id)
		return
	}
	if why := w.middleBlocked(id); why != "" {
		w.setStatus(why)
		return
	}
	if middleView(id) != nil {
		m.tabs = append(m.tabs, id)
		w.showMiddle(id)
		w.setStatus(m.status())
		return
	}
	p := w.dock.pane(id)
	slot := w.dock.slotNow(id)
	wasDocked := w.dock.shown(id)
	if p.doc != nil {
		d := p.doc(w)
		if d == nil {
			w.setStatus(p.cmd.label + " has no document to open")
			return
		}
		if m.docs == nil {
			m.docs = map[string]*document{}
		}
		m.docs[id] = d
	}
	w.dock.on[id] = false // one place at a time: a Fyne object has one parent
	if p.toMiddle != nil {
		p.toMiddle(w)
	}
	m.tabs = append(m.tabs, id)
	w.showMiddle(id)
	if wasDocked {
		w.dock.refreshFront(slot) // the tab now in front where it was
	}
	w.setStatus(m.status())
}

// status is what the status line says of the tab in front.
func (m *middle) status() string {
	w := m.w
	what := m.title(m.shown)
	if middleView(m.shown) != nil {
		return what + " in the middle: the × on its tab row closes it"
	}
	if d := m.docs[m.shown]; d != nil {
		msg := what + " in the middle (" + d.where() + "): its tab's menu puts it back to the side"
		if w.shared != nil && d.sharedAs() == "" {
			msg += ". Editing together: the source is yours only, the others do not see its changes (it is not shared)"
		}
		return msg
	}
	return what + " in the middle: its tab's menu puts it back to the side"
}

// showMiddle puts middle tab id in front ("" the script).
func (w *MainWindow) showMiddle(id string) { w.mid.show(id) }

// show is the one transition of what the middle shows: tab id to the
// front, and with it every view that follows the main area.
func (m *middle) show(id string) {
	w := m.w
	if id != "" && !m.has(id) {
		return
	}
	if id == m.shown {
		w.relayout()
		w.focusMain()
		return
	}
	m.showing = true
	defer func() { m.showing = false }()
	// a phone's Pages tab stays among the script and the views: the pages
	// are a tab of their own there, not the preview beside the script
	// (pagesTabFollow); a document coming in has its own preview
	keepPages := m.has("pages") && m.docs[m.shown] == nil && m.docs[id] == nil
	// a preview shown is of the document in front: it goes with it, and
	// the one coming in gets its own (Joop: the preview "gets stuck when
	// clicking swap")
	hadPreview := w.pagesVisible || w.previewVisible || m.keepPreview
	if d := m.docs[m.shown]; d != nil { // leaving a document
		w.relinkAfterEdit(d) // the source edited: its links looked at again (relink.go)
	}
	if w.searchBar != nil {
		w.searchBar.of = docKey{} // its matches are found again in the tab coming in (state.go)
	}
	if !keepPages {
		w.previewGen++ // a script's PDF still being made is not shown over another tab
		if id != "pages" {
			w.hidePages()
		}
		w.previewVisible = false
	}
	w.sheetFollow(m.shown, id) // a phone's panes folded down under the board
	m.shown = id
	w.textEditor.SetOffstage(id != "") // out of sight: laid out again when back (internal/editor)
	w.docsOnStage()                    // the documents' editors too (document.go)
	p := w.dock.pane(id)
	switch v := middleView(id); {
	case id == "cards":
		w.cardsFull().update(w.textEditor.Text())
	case p != nil && m.boxShown(p) && p.shown != nil:
		p.shown(w)
	case v != nil && v.shown != nil:
		v.shown(w)
	}
	editorTab := w.main() != nil
	w.relayout()
	if keepPages {
		// the pages stay as they are, in their tab
	} else if editorTab {
		m.keepPreview = false
		w.previewFollowsMain(hadPreview)
	} else {
		m.keepPreview = hadPreview
	}
	w.editorKindChanged()
	w.focusMain()
	if id == "" {
		w.setStatus("")
	} else {
		w.setStatus(m.status())
	}
	w.mainDocChanged() // the views of the document in front (state.go)
}

// previewFollowsMain shows the preview of the document now in the main
// editor if one was shown before: a Markdown document's Live Preview (its
// Pages if those were chosen), a screenplay's pages.
func (w *MainWindow) previewFollowsMain(had bool) {
	if !had {
		return
	}
	w.mid.pagesBack = w.narrow && !w.editorMarkdown() // the script's tab in front, its pages' behind (pagesTabFollow)
	if w.editorMarkdown() {
		if w.previewPane == nil {
			w.previewPane = newPreviewPanel()
			w.previewPane.withPageSetup(w)
		}
		w.previewVisible = true
		w.updateLivePreview()
		w.relayout()
		w.refreshMenu()
		w.ribbonRefresh()
		return
	}
	w.previewPDF()
}

// closeMiddle closes middle tab id: the tab before it to the front if it
// was there, its document written to its home and let go. The script's
// tab stays.
func (w *MainWindow) closeMiddle(id string) {
	m := &w.mid
	if !m.has(id) {
		return
	}
	if m.shown == id {
		i := slices.Index(m.tabs, id)
		before := ""
		if i > 0 {
			before = m.tabs[i-1]
		}
		m.show(before)
	}
	m.tabs = slices.DeleteFunc(m.tabs, func(s string) bool { return s == id })
	if d := m.docs[id]; d != nil {
		delete(m.docs, id)
		if w.searchBar != nil {
			d.ed.SetHighlights(nil)
		}
		if err := w.closeDoc(d); err != nil && w.fyneWindow != nil {
			dialog.ShowError(err, w.fyneWindow)
		}
	}
	if p := w.dock.pane(id); p != nil && p.toSide != nil {
		p.toSide(w)
	}
	if v := middleView(id); v != nil && v.closed != nil {
		v.closed(w)
	}
	w.relayout()
	w.focusMain()
	w.refreshMenu()
	w.ribbonRefresh()
}

// backToSide closes middle tab id and shows its pane in the dock.
func (w *MainWindow) backToSide(id string) {
	if !w.mid.has(id) {
		return
	}
	if v := middleView(id); v != nil {
		w.setStatus(v.title + " has no place in the side panes: the × on the tab row closes it")
		return
	}
	w.closeMiddle(id)
	w.showPane(id, true)
	if p := w.dock.pane(id); p != nil {
		w.setStatus(p.cmd.label + " back in the side panes")
	}
}

// closeMiddleDocs closes the middle tabs that edit a document and puts
// the script's in front (leaveDocs: the window about to take another
// script); a view's tab stays.
func (w *MainWindow) closeMiddleDocs() {
	for _, id := range slices.Clone(w.mid.tabs) {
		if w.mid.docs[id] != nil {
			w.closeMiddle(id)
		}
	}
	if !w.scriptInMain() {
		w.showMiddle("")
	}
}

// middlePrune closes the middle tabs whose panes are not there for the
// document now (Cards in a Markdown document, Production for a Writer).
func (w *MainWindow) middlePrune() {
	for _, id := range slices.Clone(w.mid.tabs) {
		if v := middleView(id); v != nil {
			if !v.applies(w) {
				w.closeMiddle(id)
			}
			continue
		}
		if p := w.dock.pane(id); p == nil || !w.dock.applies(p) {
			w.closeMiddle(id)
		}
	}
}

// docTab is the middle tab editing d ("" none).
func (m *middle) docTab(d *document) string {
	for id, o := range m.docs {
		if o == d {
			return id
		}
	}
	return ""
}

// step shows the middle tab step on from the one in front (Ctrl+PgDn,
// Ctrl+PgUp), round the row.
func (m *middle) step(step int) {
	ids := m.ids()
	if len(ids) < 2 {
		m.w.setStatus("The script is the only middle tab: View > Cards, or a pane's Open in the Middle, adds one")
		return
	}
	i := slices.Index(ids, m.shown)
	m.show(ids[((i+step)%len(ids)+len(ids))%len(ids)])
}

// center is what the middle shows: the tab in front (the script with its
// preview beside it, as relayout puts it), under the row of tabs when a
// tab beside the script's is open.
func (m *middle) center(script fyne.CanvasObject) fyne.CanvasObject {
	w := m.w
	body := script
	if d := m.docs[m.shown]; d != nil {
		body = d.view
		if w.previewVisible && w.previewPane != nil {
			split := container.NewHSplit(d.view, w.previewPane.box)
			split.Offset = 0.55
			body = split
		}
	} else if p := w.dock.pane(m.shown); p != nil {
		if p.full != nil {
			body = p.full(w)
		} else {
			body = p.box(w)
		}
	} else if v := middleView(m.shown); v != nil {
		body = v.box(w)
	}
	m.body = body
	m.tabW = map[string]*dockTab{}
	m.row, m.shut = nil, nil
	if len(m.tabs) == 0 {
		return body
	}
	var items []overflowItem
	for _, id := range m.ids() {
		t := newDockTab(m, w, id, m.title(id))
		t.selected = id == m.shown
		m.tabW[id] = t
		pick := id
		items = append(items, overflowItem{obj: t, keep: t.selected, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
			return []*fyne.MenuItem{fyne.NewMenuItem(m.title(pick), func() { m.show(pick) })}
		}})
	}
	// a phone's row is one line: the tabs that fit, the one in front
	// among them, the others in the More menu at its end
	m.row = newOverflowRow(w.tooltips, items, nil, 0)
	m.row.more.Tip = "More middle tabs"
	fyne.SetAccessibleLabel(m.row.more, "More middle tabs")
	m.shut = tooltip.NewButton(theme.CancelIcon(), "", func() { w.closeMiddle(m.shown) }, w.tooltips)
	m.shut.Importance = widget.LowImportance
	if m.shown == "" {
		m.shut.Disable() // the script stays: it is the document
		m.shut.Tip = "The script's tab stays"
	} else {
		m.shut.Tip = "Close " + m.title(m.shown)
	}
	fyne.SetAccessibleLabel(m.shut, m.shut.Tip)
	row := container.NewBorder(nil, nil, nil, m.shut, m.row.Container)
	// the row first, then the tab: the order Tab and screen readers take
	top := container.NewVBox(row, widget.NewSeparator())
	return container.New(layout.NewBorderLayout(top, nil, nil, nil), top, body)
}

// The middle as a tabHost (dock.go's tabs).

func (m *middle) pick(id string) {
	if id == "" && m.shown == "" && !m.w.titleInSight() { // the script's name has no room on the top row: its editor (toprow.go)
		m.w.renameScript()
		return
	}
	m.show(id)
}

func (m *middle) menu(id string) *fyne.Menu {
	w := m.w
	if id == "" { // the script stays in the middle; its name's editor
		return fyne.NewMenu("Script", w.renameItem())
	}
	title := m.title(id)
	if middleView(id) != nil { // no place in the side panes
		return fyne.NewMenu(title, fyne.NewMenuItem("Close "+title, func() { w.closeMiddle(id) }))
	}
	return fyne.NewMenu(title,
		fyne.NewMenuItem("Back to the Side", func() { w.backToSide(id) }),
		fyne.NewMenuItem("Close "+title, func() { w.closeMiddle(id) }))
}

func (m *middle) tabsIn(*dockTab) []*dockTab {
	var out []*dockTab
	for _, id := range m.ids() {
		if t := m.tabW[id]; t != nil {
			out = append(out, t)
		}
	}
	return out
}

func (m *middle) tabOf(id string) *dockTab { return m.tabW[id] }

func (m *middle) enter(id string) {
	m.show(id)
	m.w.focusMain()
}

// focusMain gives the keyboard to the middle: its editor, else the view
// there, else its tab.
func (w *MainWindow) focusMain() {
	c := w.fyneWindow.Canvas()
	if e := w.mainEditor(); e != nil {
		c.Focus(e)
		return
	}
	if f := w.midViewFocus(); f != nil {
		c.Focus(f)
		return
	}
	if t := w.mid.tabW[w.mid.shown]; t != nil {
		c.Focus(t)
	}
}

// midViewFocus is what takes the keyboard in the view in front in the
// middle (the outline's list, the pages), nil none.
func (w *MainWindow) midViewFocus() fyne.Focusable {
	if p := w.dock.pane(w.mid.shown); p != nil && w.mid.boxShown(p) && p.focus != nil {
		return p.focus(w)
	}
	if v := middleView(w.mid.shown); v != nil && v.focus != nil {
		return v.focus(w)
	}
	return nil
}

// mainFocus is what F6 goes to first: the middle's editor, view or tab.
func (w *MainWindow) mainFocus() fyne.Focusable {
	if e := w.mainEditor(); e != nil {
		return e
	}
	if f := w.midViewFocus(); f != nil {
		return f
	}
	if t := w.mid.tabW[w.mid.shown]; t != nil {
		return t
	}
	return w.textEditor
}

// sourceInMiddle is Adaptation > Source in the Middle: the source's
// middle tab to the front (opened if it is not), or the script's if it is
// in front.
func (w *MainWindow) sourceInMiddle() {
	if w.mid.shown == "source" {
		w.showMiddle("")
		return
	}
	w.openMiddle("source")
}

// editorKindChanged follows the main editor's kind: the menu (Markdown's
// commands or the screenplay's), the editing row, the ribbon, the status
// bar.
func (w *MainWindow) editorKindChanged() {
	w.commandsOf = w.frontKey()
	if w.headerBar != nil {
		w.headerBar.showEditingRow(w.editorMarkdown(), w.midDoc() != nil && !w.markdown)
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
		w.updateLivePreview()
	}
}

// mainChanged is the text in the main editor changing, when it is not
// the script (whose changes onTextChanged follows).
func (w *MainWindow) mainChanged() {
	if w.outline != nil && w.outlineInSight() {
		w.outline.edited() // its structure (outline.go)
	}
	if w.previewVisible && w.previewPane != nil {
		w.updateLivePreview()
	}
	if w.searchBar != nil && w.searchBar.IsVisible() {
		w.searchBar.updateSearch()
	}
	w.updateCurrentElement()
	w.updateStatusBar()
	if d := w.midDoc(); d != nil && !d.inScript() {
		w.scheduleAutoSave()
	}
}

// mainCursorMoved is the main editor's cursor moving (not the script's).
func (w *MainWindow) mainCursorMoved() {
	w.updateCurrentElement()
	w.updateStatusBar()
	if w.cardsDock != nil && w.paneInSight("cards") {
		w.cardsDock.follow() // the passage's scenes' cards
	}
	w.relinkOnLeave()
}

// goToScript puts the script's cursor at the start of a line, the
// script's tab in front with the keyboard (the outline, the cards, the
// changes, Production).
func (w *MainWindow) goToScript(line int) {
	e := w.textEditor
	e.Navigate(func(b *buffer.Buffer) {
		n, _ := b.Position(b.Len())
		b.SetCursor(b.LineStart(max(0, min(line, n))), false)
	})
	if !w.scriptInMain() {
		w.showMiddle("")
	}
	w.fyneWindow.Canvas().Focus(e)
}

// scriptAway says why a command that works at the script's cursor does
// nothing while the script is not the middle tab in front; false if it is.
func (w *MainWindow) scriptAway(what string) bool {
	if w.scriptInMain() {
		return false
	}
	w.setStatus(what + " works in the script: its middle tab first (tap Script)")
	return true
}

// updateLivePreview shows the main editor's document in the Live Preview
// (the script's while a view is in front), and records which (state.go).
func (w *MainWindow) updateLivePreview() {
	w.previewPane.of = w.frontKey()
	w.previewPane.update(w.writeEditor().Text())
}

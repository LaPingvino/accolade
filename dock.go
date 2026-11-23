package main

import (
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

// The side dock (LOOP.md, Joop 2026-11-20: "Outline can probably merge
// with the others, and they could all toggle in and out with the toolbar
// button"): one dock beside the script with two slots, top and bottom.
// Every pane is a tab in one of them; a slot alone takes the dock's full
// height, and the dock is gone when no pane is shown. Each pane's View
// command (toolbar, ribbon, menu, palette) brings it in, to the front of
// its slot if it is behind another tab, and takes it out when it is in
// sight. A tab moves to the other slot from its menu (right-click, a long
// press, Shift+F10) or View > Move Pane to Top/Bottom; the arrangement,
// the dock's width and the split between the slots are remembered. On a
// phone the dock is the sheet under the script, both slots one tab row.
//
// Adding a pane is one row in dockPanes: its command (made a View
// toggle here, in the View menu and the ribbon's Panes group with the
// others), its slot at first, how its box is made and what it does when
// it comes into sight.

const (
	dockTop = iota
	dockBottom
	dockSheet // a phone's: both slots as one row under the script
)

// The settings: the arrangement (each pane's slot, in tab order, as
// "id:top"), the dock's share of a wide window (side-pane-offset before
// the dock), the top slot's share of the dock.
const (
	dockPanesPref = "dock-panes"
	dockWidthPref = "dock-width"
	dockSplitPref = "dock-split"
	oldWidthPref  = "side-pane-offset"
)

// dockPane is one pane of the dock.
type dockPane struct {
	id string
	// cmd is its View toggle ("view."+id): label (the tab's title), keys,
	// icon, tip, and when and role (where the pane applies at all); run
	// and checked are the dock's
	cmd  *command
	slot int // where it is at first: dockTop or dockBottom
	// width is the least the pane needs beside the script
	width float32
	// focusOnShow: shown by its command, it takes the keyboard
	focusOnShow bool
	box         func(w *MainWindow) fyne.CanvasObject // made at first need
	focus       func(w *MainWindow) fyne.Focusable
	// shown brings it up to date as it comes into sight (nil: nothing)
	shown func(w *MainWindow)
	// edited brings it up to date with the script's text, while it is in
	// the dock (nil: nothing, or it follows by itself)
	edited func(w *MainWindow, text string)
	// In the middle (middle.go): doc is the document its middle tab
	// edits (the source, the notes), opened; full is what the tab shows
	// instead of its box (the cards' full view); nil, its box. midFirst:
	// its View command opens it in the middle, not in the dock (Cards).
	// toMiddle and toSide are what it does going there and coming back.
	doc              func(w *MainWindow) *document
	full             func(w *MainWindow) fyne.CanvasObject
	midFirst         bool
	toMiddle, toSide func(w *MainWindow)
}

// dockPanes are the panes in their first order.
func dockPanes() []*dockPane {
	return []*dockPane{
		{id: "outline", slot: dockTop, width: 240,
			cmd:    &command{label: "Outline", keys: []fyne.Shortcut{shortcut(fyne.KeyO, fyne.KeyModifierShift)}, icon: theme.ListIcon},
			box:    func(w *MainWindow) fyne.CanvasObject { return w.outlinePane().box },
			focus:  func(w *MainWindow) fyne.Focusable { return w.outlinePane().list },
			shown:  func(w *MainWindow) { w.outlinePane().refresh() },
			edited: func(w *MainWindow, _ string) { w.outlinePane().edited() }},
		{id: "notes", slot: dockBottom, width: 280, focusOnShow: true,
			cmd:    &command{label: "Notes", keys: []fyne.Shortcut{shortcut(fyne.KeyK, fyne.KeyModifierShift)}, icon: theme.FileTextIcon},
			box:    func(w *MainWindow) fyne.CanvasObject { return w.notesPane().entry },
			doc:    func(w *MainWindow) *document { return w.notesDocument() },
			focus:  func(w *MainWindow) fyne.Focusable { return w.notesPane().entry },
			shown:  func(w *MainWindow) { w.notesPane().update(w.textEditor.Text()) },
			edited: func(w *MainWindow, text string) { w.notesPane().update(text) }},
		// who speaks, their notes, rename and merge (characters_pane.go)
		{id: "characters", slot: dockBottom, width: 260, focusOnShow: true,
			cmd:    &command{label: "Characters", tip: "Characters: who speaks and where, their notes, rename and merge", icon: theme.AccountIcon},
			box:    func(w *MainWindow) fyne.CanvasObject { return w.charactersPane().box },
			focus:  func(w *MainWindow) fyne.Focusable { return w.charactersPane().focusTarget() },
			shown:  func(w *MainWindow) { w.charactersPane().read() },
			edited: func(w *MainWindow, text string) { w.charactersPane().update(text) }},
		// the cards: in the middle at first, their full view (cards.go);
		// beside the script, one column (cards_pane.go)
		{id: "cards", slot: dockTop, width: 260, midFirst: true,
			full:   func(w *MainWindow) fyne.CanvasObject { return w.cardsFull().box },
			cmd:    &command{label: "Cards", keys: []fyne.Shortcut{shortcut(fyne.KeyD, fyne.KeyModifierShift)}, icon: theme.GridIcon, when: screenplayOnly},
			box:    func(w *MainWindow) fyne.CanvasObject { return w.cardsPane().box },
			focus:  func(w *MainWindow) fyne.Focusable { return w.cardsPane().list },
			shown:  func(w *MainWindow) { w.cardsPane().update(w.textEditor.Text()) },
			edited: func(w *MainWindow, text string) { w.cardsPane().update(text) }},
		{id: "source", slot: dockBottom, width: sourceWidth, focusOnShow: true,
			cmd:   &command{label: "Source", tip: "Source: the story the script adapts", icon: func() fyne.Resource { return sourceIcon }, when: screenplayOnly},
			box:   func(w *MainWindow) fyne.CanvasObject { return w.sourcePane().box },
			focus: func(w *MainWindow) fyne.Focusable { return w.sourcePane().focusTarget() },
			shown: func(w *MainWindow) { w.sourcePane().follow() },
			doc: func(w *MainWindow) *document {
				if s := w.source; s != nil && s.editing { // the pane's Edit: the same document, in the middle now
					s.editing = false
					s.edit.SetText("Edit")
					s.edit.SetIcon(theme.DocumentCreateIcon())
					s.showBody()
				}
				return w.sourceDoc()
			},
			toMiddle: func(w *MainWindow) { w.sourcePane().measuring++ }}, // its rows out of sight: measured again when back
		{id: "production", slot: dockBottom, width: sourceWidth, focusOnShow: true,
			cmd: &command{label: "Production", tip: "Production: lock, revise, and the stages to the shoot", icon: func() fyne.Resource { return productionIcon },
				when: screenplayOnly, role: productionRole},
			box:   func(w *MainWindow) fyne.CanvasObject { return w.productionPane().box },
			focus: func(w *MainWindow) fyne.Focusable { return w.productionPane().focusTarget() },
			shown: func(w *MainWindow) { w.productionPane().refresh() }},
		{id: "help", slot: dockBottom, width: 280, focusOnShow: true,
			cmd:   &command{label: "Help", tip: "Help: the pages, cheat sheets and a search", icon: theme.HelpIcon},
			box:   func(w *MainWindow) fyne.CanvasObject { return w.helpPane().box },
			focus: func(w *MainWindow) fyne.Focusable { return w.helpPane().focusTarget() },
			shown: func(w *MainWindow) {
				if h := w.helpPane(); h.page.Slug == "" {
					h.open("index", "")
				}
			}},
		{id: "changes", slot: dockBottom, width: 240,
			cmd:   &command{label: "Changes", keys: []fyne.Shortcut{shortcut(fyne.KeyH, fyne.KeyModifierShift)}, icon: res(changesIcon)},
			box:   func(w *MainWindow) fyne.CanvasObject { return w.changesPane().box },
			focus: func(w *MainWindow) fyne.Focusable { return w.changesPane().list },
			shown: func(w *MainWindow) { w.changesPane().refresh() }},
	}
}

// dockCommands are the panes' View toggles, and the dock's own: the whole
// dock in and out, and the pane picked last to the other slot.
func dockCommands() []*command {
	var cmds []*command
	for _, p := range dockPanes() {
		c, id := p.cmd, p.id
		c.id = "view." + id
		c.run = func(w *MainWindow) { w.togglePane(id) }
		c.checked = func(w *MainWindow) bool { return w.paneOn(id) }
		cmds = append(cmds, c)
	}
	return append(cmds,
		&command{id: "view.dock", label: "Side Panes", tip: "The side panes: all out of sight, or back as they were", icon: res(dockIcon),
			run: func(w *MainWindow) { w.dock.toggleAll() }, checked: func(w *MainWindow) bool { return len(w.dock.visible(dockSheet)) > 0 }},
		&command{id: "view.paneTop", label: "Move Pane to Top", tip: "The pane picked last to the dock's top", icon: theme.MoveUpIcon,
			run: func(w *MainWindow) { w.dock.moveCurrent(dockTop) }},
		&command{id: "view.paneBottom", label: "Move Pane to Bottom", tip: "The pane picked last to the dock's bottom", icon: theme.MoveDownIcon,
			run: func(w *MainWindow) { w.dock.moveCurrent(dockBottom) }},
		&command{id: "view.paneMiddle", label: "Open Pane in the Middle", tip: "The pane picked last as a tab in the middle, beside the script (middle.go)", icon: res(swapIcon),
			run: func(w *MainWindow) { w.dock.middleCurrent() }},
		&command{id: "view.paneSide", label: "Middle Tab Back to the Side", tip: "The middle tab in front back in the side panes", icon: res(dockIcon),
			run: func(w *MainWindow) { w.backToSide(w.mid.shown) }, enabled: func(e cmdEnv) bool { return e.away }},
		&command{id: "view.midNext", label: "Next Middle Tab", tip: "The middle tab after the one in front: the script, the cards, a pane put there",
			keys: middleKeys(fyne.KeyPageDown), icon: theme.NavigateNextIcon,
			run: func(w *MainWindow) { w.mid.step(1) }, enabled: func(e cmdEnv) bool { return e.midTabs }},
		&command{id: "view.midPrev", label: "Previous Middle Tab", tip: "The middle tab before the one in front",
			keys: middleKeys(fyne.KeyPageUp), icon: theme.NavigateBackIcon,
			run: func(w *MainWindow) { w.mid.step(-1) }, enabled: func(e cmdEnv) bool { return e.midTabs }},
	)
}

// middleKeys are Ctrl+PgDn and Ctrl+PgUp, the middle tabs' keys; not in
// the browser, which keeps them for its own tabs (the menu, the palette
// and the arrow keys on the row there).
func middleKeys(k fyne.KeyName) []fyne.Shortcut {
	if inBrowser {
		return nil
	}
	return []fyne.Shortcut{shortcut(k, 0)}
}

// dockHomes are the panes' toggles and the dock's commands, where a menu
// or a ribbon group lists them.
func dockHomes() []home {
	var homes []home
	for _, p := range dockPanes() {
		homes = append(homes, at("view."+p.id))
	}
	return append(homes, at("view.dock"), at("view.paneTop"), at("view.paneBottom"), at("view.paneMiddle"), at("view.paneSide"),
		at("view.midNext"), at("view.midPrev"))
}

// dockRibbonHomes are the ribbon's Panes group: the first two panes
// large, the other panes and Side Panes small (moving a pane: its tab's
// menu, the View menu, the palette). Changes is Collaboration's (with
// Edit Together: what the others changed), so the View tab stays within
// a desktop's width.
func dockRibbonHomes() []home {
	var homes []home
	for i, p := range dockPanes() {
		if p.id == "changes" {
			continue
		}
		if i < 2 {
			homes = append(homes, big("view."+p.id))
			continue
		}
		homes = append(homes, at("view."+p.id))
	}
	return append(homes, at("view.dock"))
}

// dock is a window's side dock.
type dock struct {
	w      *MainWindow
	panes  []*dockPane // in tab order: the arrangement's
	on     map[string]bool
	slotOf map[string]int
	front  [3]string // each slot's tab in front; [dockSheet] a phone's
	last   string    // the pane picked last: what Move to Top/Bottom moves
	stash  []string  // the panes Side Panes took away, to bring back
	split  *container.Split
	tabs   map[string]*dockTab // as last built
	// sheetRow is a phone's tab row, as last built (nil on a wide window)
	sheetRow *overflowRow
	bodies   [3]*fyne.Container // each slot's pane in front, as last built
	// hideTips name the pane in front on each slot's hide button
	hideTips [3]func()
}

func newDock(w *MainWindow) *dock {
	d := &dock{w: w, on: map[string]bool{}, slotOf: map[string]int{}}
	d.panes = dockPanes()
	for _, p := range d.panes {
		d.slotOf[p.id] = p.slot
	}
	d.load()
	return d
}

// load reads the arrangement, and the width kept before the dock.
func (d *dock) load() {
	s := d.w.settings
	if s == nil {
		return
	}
	if !s.HasKey(dockWidthPref) && s.HasKey(oldWidthPref) {
		s.SetFloat(dockWidthPref, s.GetFloat(oldWidthPref))
	}
	byID := map[string]*dockPane{}
	for _, p := range d.panes {
		byID[p.id] = p
	}
	var order []*dockPane
	for _, e := range s.GetStringSlice(dockPanesPref) {
		id, where, _ := strings.Cut(e, ":")
		p := byID[id]
		if p == nil {
			continue // a pane this version has not
		}
		delete(byID, id)
		order = append(order, p)
		if where == "top" {
			d.slotOf[id] = dockTop
		} else {
			d.slotOf[id] = dockBottom
		}
	}
	for _, p := range d.panes { // panes new since, where they go at first
		if byID[p.id] != nil {
			order = append(order, p)
		}
	}
	d.panes = order
}

// save keeps the arrangement.
func (d *dock) save() {
	var out []string
	for _, p := range d.panes {
		where := "bottom"
		if d.slotOf[p.id] == dockTop {
			where = "top"
		}
		out = append(out, p.id+":"+where)
	}
	d.w.settings.SetStringSlice(dockPanesPref, out)
}

// pane is the pane with id (nil if none).
func (d *dock) pane(id string) *dockPane {
	for _, p := range d.panes {
		if p.id == id {
			return p
		}
	}
	return nil
}

// applies reports whether p is there for the document's kind and the
// role (Production's put away in Writer, a screenplay's in Markdown).
func (d *dock) applies(p *dockPane) bool {
	return p.cmd.applies(cmdEnv{markdown: d.w.markdown, writer: d.w.writerView()})
}

// shown reports whether the pane is in the dock (in front or not).
func (d *dock) shown(id string) bool {
	p := d.pane(id)
	return p != nil && d.on[id] && d.applies(p) && !d.w.mid.has(id) // in the middle, not also here
}

// slotNow is where the pane is shown now: its slot, or the phone's sheet.
func (d *dock) slotNow(id string) int {
	if d.w.narrow {
		return dockSheet
	}
	return d.slotOf[id]
}

// visible are the panes shown in slot, in order (dockSheet: all of them,
// the top's first).
func (d *dock) visible(slot int) []*dockPane {
	var out []*dockPane
	for _, s := range []int{dockTop, dockBottom} {
		for _, p := range d.panes {
			if d.slotOf[p.id] == s && (slot == dockSheet || slot == s) && d.shown(p.id) {
				out = append(out, p)
			}
		}
	}
	return out
}

// inFront is slot's tab in front: the one picked there, else its first.
func (d *dock) inFront(slot int) string {
	vis := d.visible(slot)
	for _, p := range vis {
		if p.id == d.front[slot] {
			return p.id
		}
	}
	if len(vis) > 0 {
		return vis[0].id
	}
	return ""
}

// inSight reports whether the pane is shown and in front of its slot.
func (d *dock) inSight(id string) bool {
	return d.shown(id) && d.inFront(d.slotNow(id)) == id
}

// setFront puts the pane in front of its slot (and of the phone's sheet).
func (d *dock) setFront(id string) {
	d.front[d.slotOf[id]] = id
	d.front[dockSheet] = id
	d.last = id
}

// edited brings the panes up to date with the script's text: those in
// the dock, and one in the middle showing its own box (the outline).
func (d *dock) edited(text string) {
	for _, p := range d.panes {
		if p.edited != nil && (d.shown(p.id) || d.w.mid.boxShown(p)) {
			p.edited(d.w, text)
		}
	}
}

// focusOf is what takes the keyboard in the pane (nil: nothing).
func (d *dock) focusOf(p *dockPane) fyne.Focusable {
	if p.focus == nil {
		return nil
	}
	return p.focus(d.w)
}

// shownNow brings the pane up to date as it comes into sight.
func (d *dock) shownNow(p *dockPane) {
	if p.shown != nil {
		p.shown(d.w)
	}
}

// refreshFront brings slot's tab in front up to date: one that came to
// the front by itself, another tab having gone.
func (d *dock) refreshFront(slot int) {
	if p := d.pane(d.inFront(slot)); p != nil {
		d.shownNow(p)
	}
}

// putAway takes the panes the role has not out of the dock (Production's
// as a Writer), so that one asked for again comes in, not out.
func (d *dock) putAway() {
	env := cmdEnv{markdown: d.w.markdown, writer: d.w.writerView()}
	for _, p := range d.panes {
		if !p.cmd.role.in(env) {
			d.on[p.id] = false
		}
	}
}

// paneShown reports whether the pane with id is in the dock.
func (w *MainWindow) paneShown(id string) bool { return w.dock != nil && w.dock.shown(id) }

// paneInSight reports whether the pane with id is in sight: in the dock,
// in front of its slot; in the middle, its box in front there (the
// outline, help…), not its document or full view (middle.go).
func (w *MainWindow) paneInSight(id string) bool {
	if w.dock == nil {
		return false
	}
	if p := w.dock.pane(id); p != nil && w.mid.has(id) {
		return w.mid.shown == id && w.mid.boxShown(p)
	}
	return w.dock.inSight(id)
}

// paneOn reports whether the pane is there, in the dock or in the middle
// (its View command's check).
func (w *MainWindow) paneOn(id string) bool { return w.paneShown(id) || w.mid.has(id) }

// togglePane is a pane's View command: out of the dock, in it (and to
// the front of its slot, taking the keyboard if it does); in sight, out.
// A pane in the middle: to the front there, or in front, its tab closed.
// Cards come in as a middle tab (midFirst).
func (w *MainWindow) togglePane(id string) {
	d := w.dock
	p := d.pane(id)
	if p == nil || !d.applies(p) {
		return
	}
	switch {
	case w.mid.has(id) && w.mid.shown == id:
		w.closeMiddle(id)
	case w.mid.has(id):
		w.showMiddle(id)
	case d.inSight(id):
		w.hidePane(id)
	case p.midFirst && !d.shown(id):
		w.openMiddle(id)
	default:
		w.showPane(id, p.focusOnShow)
	}
}

// showPane brings the pane into the dock and in front of its slot,
// focused if focus; brought up to date.
func (w *MainWindow) showPane(id string, focus bool) {
	d := w.dock
	p := d.pane(id)
	if p == nil || !d.applies(p) {
		return
	}
	if w.mid.has(id) { // a middle tab: to the front there, not into the dock too (one parent)
		w.showMiddle(id)
		return
	}
	was := d.shown(id)
	d.on[id] = true
	d.stash = nil
	d.setFront(id)
	if was && d.bodies[d.slotNow(id)] != nil {
		d.showFront(d.slotNow(id)) // already there: only the tab in front
	} else {
		w.relayout()
	}
	d.shownNow(p)
	if f := d.focusOf(p); focus && f != nil {
		w.fyneWindow.Canvas().Focus(f)
	}
	w.refreshMenu()
}

// hidePane takes the pane out of the dock.
func (w *MainWindow) hidePane(id string) {
	d := w.dock
	if !d.on[id] {
		return
	}
	slot := d.slotNow(id)
	d.on[id] = false
	w.relayout()
	d.refreshFront(slot)
	w.refreshMenu()
}

// toggleAll is View > Side Panes: every pane out of sight, or those it
// took away back again (none: the outline).
func (d *dock) toggleAll() {
	if vis := d.visible(dockSheet); len(vis) > 0 {
		d.stash = nil
		for _, p := range vis {
			d.stash = append(d.stash, p.id)
			d.on[p.id] = false
		}
		stash := d.stash
		d.w.relayout()
		d.w.refreshMenu()
		d.stash = stash
		return
	}
	back := d.stash
	if len(back) == 0 {
		back = []string{"outline"}
	}
	for _, id := range back {
		d.on[id] = true
	}
	d.stash = nil
	d.w.relayout()
	for _, id := range back {
		if p := d.pane(id); p != nil && d.inSight(id) {
			d.shownNow(p)
		}
	}
	d.w.refreshMenu()
}

// moveCurrent is View > Move Pane to Top/Bottom: the pane with the
// keyboard (its tab or inside it), else the one picked last.
func (d *dock) moveCurrent(slot int) {
	id := d.focusedPane()
	if id == "" {
		id = d.last
	}
	if id == "" || !d.shown(id) {
		d.w.setStatus("No pane to move: show one first (View > Outline, Notes…)")
		return
	}
	d.move(id, slot)
}

// focusedPane is the pane whose tab or contents have the keyboard.
func (d *dock) focusedPane() string {
	f := d.w.fyneWindow.Canvas().Focused()
	if f == nil {
		return ""
	}
	if t, ok := f.(*dockTab); ok {
		return t.id
	}
	for _, p := range d.panes {
		if d.shown(p.id) && d.focusOf(p) == f {
			return p.id
		}
	}
	return ""
}

// middleCurrent is View > Open Pane in the Middle: the pane with the
// keyboard, else the one picked last.
func (d *dock) middleCurrent() {
	id := d.focusedPane()
	if id == "" {
		id = d.last
	}
	if id == "" || !d.shown(id) {
		d.w.setStatus("No pane to put in the middle: show one first (View > Outline, Notes…)")
		return
	}
	d.w.openMiddle(id)
}

// move puts the pane in slot (last there), in front, the keyboard on its
// tab if the keyboard was in the dock.
func (d *dock) move(id string, slot int) {
	p := d.pane(id)
	if p == nil {
		return
	}
	where := map[int]string{dockTop: "top", dockBottom: "bottom"}[slot]
	title := p.cmd.label
	if d.slotOf[id] == slot {
		d.w.setStatus(title + " is at the " + where + " already")
		return
	}
	keyboard := d.focusedPane() != ""
	from := d.slotOf[id]
	d.slotOf[id] = slot
	var order []*dockPane // last in its new slot
	for _, q := range d.panes {
		if q != p {
			order = append(order, q)
		}
	}
	d.panes = append(order, p)
	d.save()
	d.setFront(id)
	d.w.relayout()
	if d.shown(id) {
		d.shownNow(p)
	}
	if !d.w.narrow {
		d.refreshFront(from) // the tab now in front where it was
	}
	if t := d.tabs[id]; keyboard && t != nil {
		d.w.fyneWindow.Canvas().Focus(t)
	}
	if d.w.narrow {
		d.w.setStatus(title + " moved to the " + where + " of the side panes (on a wider window)")
	} else {
		d.w.setStatus(title + " moved to the " + where)
	}
}

// keepSizes writes the dock's width and the split between its slots to
// the settings, as they are now.
func (d *dock) keepSizes() {
	s := d.w.settings
	if d.w.sideSplit != nil {
		s.SetFloat(dockWidthPref, d.w.sideSplit.Offset)
	}
	if d.split != nil {
		s.SetFloat(dockSplitPref, d.split.Offset)
	}
}

// object is the dock as it is to be shown, nil with no pane shown: a
// phone's sheet, or the slots above each other.
func (d *dock) object() fyne.CanvasObject {
	d.keepSizes()
	d.tabs = map[string]*dockTab{}
	d.sheetRow = nil
	d.bodies = [3]*fyne.Container{}
	d.hideTips = [3]func(){}
	d.split = nil
	if d.w.narrow {
		return d.slot(dockSheet)
	}
	top, bottom := d.slot(dockTop), d.slot(dockBottom)
	switch {
	case top != nil && bottom != nil:
		offset := d.w.settings.GetFloat(dockSplitPref)
		if offset <= 0.05 || offset >= 0.95 {
			offset = 0.5
		}
		d.split = container.NewVSplit(top, bottom)
		d.split.Offset = offset
		return d.split
	case top != nil:
		return top
	}
	return bottom // or nil
}

// slot is one slot: its tab row (and the button hiding the pane in
// front; on a phone the one making the sheet taller), the pane in front
// under it. nil if it shows no pane.
func (d *dock) slot(slot int) fyne.CanvasObject {
	vis := d.visible(slot)
	if len(vis) == 0 {
		return nil
	}
	var tabs []fyne.CanvasObject
	width := float32(0)
	for _, p := range vis {
		t := newDockTab(d, d.w, p.id, p.cmd.label)
		d.tabs[p.id] = t
		tabs = append(tabs, t)
		width = max(width, p.width)
	}
	hide := tooltip.NewButton(theme.CancelIcon(), "", func() { d.w.hidePane(d.inFront(slot)) }, d.w.tooltips)
	hide.Importance = widget.LowImportance
	buttons := []fyne.CanvasObject{hide}
	if slot == dockSheet {
		// a finger cannot drag the split in the browser (a drag scrolls),
		// so a button makes the sheet taller
		d.w.paneTaller = tooltip.NewButton(theme.MoveUpIcon(), "", nil, d.w.tooltips)
		d.w.paneTaller.Importance = widget.LowImportance
		d.w.paneTaller.OnTapped = func() {
			if d.w.paneSplit.Offset > paneTall+0.05 {
				d.w.paneSplit.SetOffset(paneTall)
			} else {
				d.w.paneSplit.SetOffset(paneHalf)
			}
			d.w.showPaneSize()
		}
		buttons = append([]fyne.CanvasObject{d.w.paneTaller}, buttons...)
	}
	// a phone's row is one line: the tabs that fit, the one in front among
	// them, the others in the menu of a More button at its end (eight panes
	// would wrap to three rows of the sheet); a slot beside the script wraps
	tabRow := fyne.CanvasObject(container.New(layout.NewRowWrapLayout(), tabs...))
	if slot == dockSheet {
		var items []overflowItem
		for _, p := range vis {
			id, title := p.id, p.cmd.label
			items = append(items, overflowItem{obj: d.tabs[id], menu: func(fyne.CanvasObject) []*fyne.MenuItem {
				return []*fyne.MenuItem{fyne.NewMenuItem(title, func() { d.pick(id) })}
			}})
		}
		d.sheetRow = newOverflowRow(d.w.tooltips, items, nil, 0)
		d.sheetRow.more.Tip = "More panes"
		fyne.SetAccessibleLabel(d.sheetRow.more, "More panes")
		tabRow = d.sheetRow.Container
	}
	row := container.NewBorder(nil, nil, nil, container.NewHBox(buttons...), tabRow)
	wide := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	wide.SetMinSize(fyne.NewSize(width, 0))
	body := container.NewStack()
	d.bodies[slot] = body
	hideTip := func() {
		if p := d.pane(d.inFront(slot)); p != nil {
			hide.Tip = "Hide " + p.cmd.label
			fyne.SetAccessibleLabel(hide, hide.Tip)
		}
	}
	d.showFront(slot)
	hideTip()
	d.hideTips[slot] = hideTip
	// the row first, then the pane: the order Tab and screen readers take
	// (NewBorder would put the pane first)
	top := container.NewVBox(row, widget.NewSeparator())
	return container.NewStack(wide, container.New(layout.NewBorderLayout(top, nil, nil, nil), top, body))
}

// showFront shows slot's tab in front, as picked: its pane under the
// row, its tab marked.
func (d *dock) showFront(slot int) {
	body := d.bodies[slot]
	if body == nil {
		return
	}
	front := d.inFront(slot)
	for _, p := range d.visible(slot) {
		if t := d.tabs[p.id]; t != nil {
			t.setSelected(p.id == front)
		}
		if p.id == front {
			body.Objects = []fyne.CanvasObject{p.box(d.w)}
		}
	}
	if r := d.sheetRow; slot == dockSheet && r != nil { // the tab in front stays on the phone's row
		for i := range r.items {
			r.items[i].keep = r.items[i].obj == fyne.CanvasObject(d.tabs[front])
		}
		r.Container.Refresh()
	}
	body.Refresh()
	if f := d.hideTips[slot]; f != nil {
		f()
	}
}

// pick puts a tab's pane in front (a tap on it, an arrow key).
func (d *dock) pick(id string) {
	d.setFront(id)
	slot := d.slotNow(id)
	d.showFront(slot)
	if p := d.pane(id); p != nil {
		d.shownNow(p)
	}
	d.w.ribbonFollow()
}

// focusTargets are what F6 goes to in the dock: each slot's pane in
// front, the top's first.
func (d *dock) focusTargets() []fyne.Focusable {
	var out []fyne.Focusable
	slots := []int{dockTop, dockBottom}
	if d.w.narrow {
		slots = []int{dockSheet}
	}
	for _, s := range slots {
		if p := d.pane(d.inFront(s)); p != nil {
			if f := d.focusOf(p); f != nil {
				out = append(out, f)
			}
		}
	}
	return out
}

// menu is a tab's menu: to the other slot, into the middle, out of the
// dock.
func (d *dock) menu(id string) *fyne.Menu {
	p := d.pane(id)
	title := p.cmd.label
	var items []*fyne.MenuItem
	if d.slotOf[id] == dockTop {
		items = append(items, fyne.NewMenuItem("Move to Bottom", func() { d.move(id, dockBottom) }))
	} else {
		items = append(items, fyne.NewMenuItem("Move to Top", func() { d.move(id, dockTop) }))
	}
	mid := fyne.NewMenuItem("Open in the Middle", func() { d.w.openMiddle(id) })
	mid.Disabled = d.w.middleBlocked(id) != ""
	items = append(items, mid, fyne.NewMenuItem("Hide "+title, func() { d.w.hidePane(id) }))
	return fyne.NewMenu(title, items...)
}

func (d *dock) tabOf(id string) *dockTab { return d.tabs[id] }

// enter is Down or Enter on a pane's tab: the keyboard into the pane.
func (d *dock) enter(id string) {
	if p := d.pane(id); p != nil {
		if f := d.focusOf(p); f != nil {
			d.w.fyneWindow.Canvas().Focus(f)
		}
	}
}

// tabsIn are the tabs of the slot tab t is in, in order.
func (d *dock) tabsIn(t *dockTab) []*dockTab {
	var out []*dockTab
	for _, p := range d.visible(d.slotNow(t.id)) {
		if o := d.tabs[p.id]; o != nil {
			out = append(out, o)
		}
	}
	return out
}

// tabHost is a row of tabs: the dock's slots, or the middle's (middle.go).
type tabHost interface {
	pick(id string)               // tab id in front
	menu(id string) *fyne.Menu    // its menu (nil: none)
	tabsIn(t *dockTab) []*dockTab // the tabs of t's row, in order
	enter(id string)              // Down or Enter on it: into what it shows
	tabOf(id string) *dockTab     // tab id as built now (a pick may build the row again)
}

// dockTab is a pane's tab (or a middle tab): a tap (or Enter, Space) puts
// it in front, the arrow keys go along the row, Down into the pane, its
// menu on a right-click, a long press or Shift+F10. The row is one stop
// for Tab: only the tab in front takes it.
type dockTab struct {
	widget.BaseWidget
	h                          tabHost
	w                          *MainWindow
	id, title                  string
	selected, hovered, focused bool
	press                      *time.Timer // a long press under way (a finger in the browser)
	pressed                    bool        // it was long: the tap after it is not one
}

func newDockTab(h tabHost, w *MainWindow, id, title string) *dockTab {
	t := &dockTab{h: h, w: w, id: id, title: title}
	t.ExtendBaseWidget(t)
	return t
}

func (t *dockTab) setSelected(on bool) {
	if t.selected != on {
		t.selected = on
		t.Refresh()
	}
}

func (t *dockTab) Tapped(*fyne.PointEvent) {
	if t.pressed {
		t.pressed = false
		return
	}
	t.h.pick(t.id)
}

func (t *dockTab) TappedSecondary(e *fyne.PointEvent) { t.showMenu(e.AbsolutePosition) }

// showMenu shows the tab's menu at pos (or under the tab).
func (t *dockTab) showMenu(pos fyne.Position) {
	c := fyne.CurrentApp().Driver().CanvasForObject(t)
	if c == nil {
		return
	}
	if pos == (fyne.Position{}) {
		pos = fyne.CurrentApp().Driver().AbsolutePositionForObject(t).AddXY(0, t.Size().Height)
	}
	m := t.h.menu(t.id)
	if m == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(m, c, pos)
}

// longPressAfter runs f, UI code, after d in the UI's turn (uiDo); a test
// replaces it and runs f itself, in its own turn.
var longPressAfter = func(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(d, func() { uiDo(f) })
}

// MouseDown and MouseUp: a press held half a second is a long one, the
// menu (the browser on a phone sends a finger as the mouse).
func (t *dockTab) MouseDown(e *desktop.MouseEvent) {
	t.pressed = false // a long press's release may have gone to its menu
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	pos := e.AbsolutePosition
	t.press = longPressAfter(500*time.Millisecond, func() {
		t.press = nil
		t.pressed = true
		t.showMenu(pos)
	})
}

func (t *dockTab) MouseUp(*desktop.MouseEvent) {
	if t.press != nil {
		t.press.Stop()
		t.press = nil
	}
}

func (t *dockTab) MouseIn(*desktop.MouseEvent) {
	t.hovered = true
	t.Refresh()
}
func (t *dockTab) MouseMoved(*desktop.MouseEvent) {}
func (t *dockTab) MouseOut() {
	t.hovered = false
	t.Refresh()
	if t.press != nil { // dragged away: no long press
		t.press.Stop()
		t.press = nil
	}
}

func (t *dockTab) FocusGained() {
	t.focused = true
	t.Refresh()
}
func (t *dockTab) FocusLost() {
	t.focused = false
	t.Refresh()
}

func (t *dockTab) TypedRune(ch rune) {
	if ch == ' ' {
		t.h.pick(t.id)
	}
}

func (t *dockTab) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyLeft:
		t.step(-1, false)
	case fyne.KeyRight:
		t.step(1, false)
	case fyne.KeyHome:
		t.step(1, true)
	case fyne.KeyEnd:
		t.step(-1, true)
	case fyne.KeyReturn, fyne.KeyEnter, fyne.KeyDown:
		t.h.enter(t.id)
	case fyne.KeyEscape:
		t.w.focusMain()
	}
}

// step picks the tab step along (from the end with fromEnd: Home, End),
// the keyboard with it.
func (t *dockTab) step(step int, fromEnd bool) {
	tabs := t.h.tabsIn(t)
	n := len(tabs)
	if n == 0 {
		return
	}
	i := 0
	for j, o := range tabs {
		if o == t {
			i = j
		}
	}
	if fromEnd {
		i = -1
		if step < 0 {
			i = n
		}
	}
	next := tabs[((i+step)%n+n)%n]
	t.h.pick(next.id)
	if n := t.h.tabOf(next.id); n != nil {
		t.w.fyneWindow.Canvas().Focus(n)
	}
}

// SkipsFocusChain: Tab reaches only the tab in front of a row.
func (t *dockTab) SkipsFocusChain() bool { return !t.selected }

func (t *dockTab) AccessibilityLabel() string             { return t.title + " tab" }
func (t *dockTab) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleTab }
func (t *dockTab) AccessibilityStates() []fyne.AccessibleState {
	if t.selected {
		return []fyne.AccessibleState{fyne.AccessibleStateSelected}
	}
	return nil
}
func (t *dockTab) AccessibilityActions() []fyne.AccessibleAction {
	return []fyne.AccessibleAction{fyne.AccessibleActionSelect}
}
func (t *dockTab) AccessibilityPerformAction(fyne.AccessibleAction) bool {
	t.h.pick(t.id)
	return true
}

func (t *dockTab) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = theme.InputRadiusSize()
	line := canvas.NewRectangle(color.Transparent)
	text := canvas.NewText(t.title, theme.Color(theme.ColorNameForeground))
	text.TextSize = theme.TextSize()
	r := &ribbonRenderer{objects: []fyne.CanvasObject{bg, line, text}}
	r.layout = func(size fyne.Size) {
		bg.Resize(size)
		line.Move(fyne.NewPos(0, size.Height-3))
		line.Resize(fyne.NewSize(size.Width, 3))
		ts := text.MinSize()
		text.Move(fyne.NewPos((size.Width-ts.Width)/2, (size.Height-ts.Height)/2))
	}
	r.min = func() fyne.Size {
		ts := fyne.MeasureText(t.title, theme.TextSize(), fyne.TextStyle{Bold: true})
		h := ts.Height + 2*theme.InnerPadding()
		if t.w.narrow {
			h = max(h, 44) // a finger's
		}
		return fyne.NewSize(ts.Width+2*theme.InnerPadding(), h)
	}
	r.refresh = func() {
		text.Text = t.title
		text.TextStyle.Bold = t.selected
		text.Color = theme.Color(theme.ColorNameForeground)
		switch {
		case t.focused:
			bg.FillColor = theme.Color(theme.ColorNameFocus)
		case t.hovered:
			bg.FillColor = theme.Color(theme.ColorNameHover)
		default:
			bg.FillColor = color.Transparent
		}
		line.FillColor = color.Transparent
		if t.selected {
			line.FillColor = theme.Color(theme.ColorNamePrimary)
		}
		bg.Refresh()
		line.Refresh()
		text.Refresh()
	}
	r.refresh()
	return r
}

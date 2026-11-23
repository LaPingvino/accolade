package main

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The top row (Joop, 2026-11-23: "I think Word has some options always
// visible, too, on their ribbon?" — Word's Quick Access Toolbar and its
// "Tell me" box; and "on Mobile the clickable name part drops, and
// that's still very useful"). The top row is the row always in sight:
// the Ribbon look's tab row (the script's name, the search field, the
// lock, ☰ at its end) and the Menus look's toolbar (header_bar.go).
//
//   - Any ribbon button, top row icon or palette entry (right-click, or
//     press and hold) has Add to the Top Row, or Remove from the Top Row;
//     View > Add to the Top Row… picks any command (a menu's item: the
//     palette has them all), View > Reset the Top Row goes back to the
//     standard row. ☰ stays (the menus are there). Kept per phone and
//     desktop, as the ribbon's lock is: the changes from the standard
//     row, so a later Accolade's new standard icons come in.
//   - The search field (the palette's button, wide) opens the palette as
//     you type, the letters typed going on in it; narrow, it is the 🔍
//     button.
//   - On a phone the script's name keeps its place before the icons: the
//     less used go into the … menu first (ranked by the Quick tab's
//     score, quick.go), the name only when nothing else is left to go; the
//     … menu then has Rename… with the name (the same editor as a tap on
//     the name), as the Script middle tab's label has (middle.go).

// topRowPref is the setting of the top row's changes on a phone or the
// desktop: "+id" added (in that order), "-id" taken off.
func topRowPref(phone bool) string {
	if phone {
		return "top-row-phone"
	}
	return "top-row-desktop"
}

// topRowFixed are the top row's commands that stay: ☰ has the menus.
func topRowFixed(id string) bool { return id == "view.menu" }

// topTitleRoom is the room the script's name keeps on the top row.
const topTitleRoom = 110

// topEdits are the window's top row changes: added in order, removed.
type topEdits struct {
	added   []string
	removed map[string]bool
}

// topRowEdits are the changes as set for the window's kind (phone or
// desktop).
func (w *MainWindow) topRowEdits() topEdits {
	e := topEdits{removed: map[string]bool{}}
	if w.settings == nil {
		return e
	}
	for _, s := range w.settings.GetStringSlice(topRowPref(w.narrow)) {
		id := s[min(1, len(s)):]
		if commands().byID[id] == nil {
			continue // a command gone since
		}
		switch {
		case strings.HasPrefix(s, "+") && !slices.Contains(e.added, id):
			e.added = append(e.added, id)
		case strings.HasPrefix(s, "-"):
			e.removed[id] = true
		}
	}
	return e
}

// setTopRowEdits keeps e as the window's kind's changes.
func (w *MainWindow) setTopRowEdits(e topEdits) {
	if w.settings == nil {
		return
	}
	var out []string
	for _, id := range e.added {
		out = append(out, "+"+id)
	}
	var removed []string
	for id := range e.removed {
		removed = append(removed, "-"+id)
	}
	slices.Sort(removed)
	w.settings.SetStringSlice(topRowPref(w.narrow), append(out, removed...))
	_ = w.settings.Save()
}

// topRowable reports whether c can go on the top row: a command with an
// icon or a mark that does something when tapped.
func topRowable(c *command) bool {
	return c != nil && (c.icon != nil || c.text != "") && (c.run != nil || c.pick != nil || c.tap != nil) && c.submenu == nil
}

// topHomes are a top row's homes: base without the removed, the added
// put at (an index of base: the toolbar's end, before the ribbon's search
// field).
func topHomes(base []home, e topEdits, at int) []home {
	var out []home
	put := func() {
		for _, id := range e.added {
			if !slices.ContainsFunc(base, func(h home) bool { return h.id == id }) {
				out = append(out, at0(id))
			}
		}
	}
	for i, h := range base {
		if i == at {
			put()
		}
		if h.id != "" && e.removed[h.id] && !topRowFixed(h.id) {
			continue
		}
		out = append(out, h)
	}
	if at >= len(base) {
		if len(e.added) > 0 && len(out) > 0 && !out[len(out)-1].sep {
			out = append(out, sep())
		}
		put()
	}
	// no gap doubled or at an end
	var clean []home
	for _, h := range out {
		if h.sep && (len(clean) == 0 || clean[len(clean)-1].sep) {
			continue
		}
		clean = append(clean, h)
	}
	for len(clean) > 0 && clean[len(clean)-1].sep {
		clean = clean[:len(clean)-1]
	}
	return clean
}

func at0(id string) home { return home{id: id} }

// toolbarHomes are the Menus look's top row; ribbonEndHomes the Ribbon
// look's tab row's end (after the name).
func (w *MainWindow) toolbarHomes() []home {
	return topHomes(commands().topRow, w.topRowEdits(), len(commands().topRow))
}

func (w *MainWindow) ribbonEndHomes() []home {
	return topHomes(commands().ribbonEnd, w.topRowEdits(), 0)
}

// onTopRow reports whether id is on the window's top row, in its look.
func (w *MainWindow) onTopRow(id string) bool {
	hs := w.toolbarHomes()
	if w.ribbonOn() {
		hs = w.ribbonEndHomes()
	}
	return slices.ContainsFunc(hs, func(h home) bool { return h.id == id })
}

// addToTopRow puts id on the top row (both looks', the window's kind's).
func (w *MainWindow) addToTopRow(id string) {
	c := commands().byID[id]
	if !topRowable(c) {
		w.setStatus("That one cannot go on the top row")
		return
	}
	e := w.topRowEdits()
	if e.removed[id] {
		delete(e.removed, id) // a standard one back in its place
	} else if !slices.Contains(e.added, id) {
		e.added = append(e.added, id)
	}
	w.setTopRowEdits(e)
	w.setStatus(c.labelIn(w.cmdEnv()) + " added to the top row (right-click it, or press and hold, to remove it)")
	w.topRowChanged()
}

// removeFromTopRow takes id off the top row.
func (w *MainWindow) removeFromTopRow(id string) {
	if topRowFixed(id) {
		w.setStatus("☰ stays on the top row: the menus are there")
		return
	}
	e := w.topRowEdits()
	if i := slices.Index(e.added, id); i >= 0 {
		e.added = slices.Delete(e.added, i, i+1)
	} else {
		e.removed[id] = true
	}
	w.setTopRowEdits(e)
	label := id
	if c := commands().byID[id]; c != nil {
		label = c.labelIn(w.cmdEnv())
	}
	w.setStatus(label + " taken off the top row (View > Add to the Top Row… puts it back)")
	w.topRowChanged()
}

// resetTopRow goes back to the standard top row.
func (w *MainWindow) resetTopRow() {
	w.setTopRowEdits(topEdits{})
	w.setStatus("The top row is the standard one again")
	w.topRowChanged()
}

// topRowEdited: the top row is not the standard one.
func (w *MainWindow) topRowEdited() bool {
	e := w.topRowEdits()
	return len(e.added) > 0 || len(e.removed) > 0
}

// topRowChanged makes every window's top row again (the same kind of
// window shares the changes).
func (w *MainWindow) topRowChanged() {
	ws := []*MainWindow{w}
	if w.app != nil {
		for _, x := range w.app.windows {
			if x != w {
				ws = append(ws, x)
			}
		}
	}
	for _, x := range ws {
		x.rebuildTopRow()
	}
}

// rebuildTopRow makes the window's top row again, in both looks.
func (w *MainWindow) rebuildTopRow() {
	if w.headerBar != nil {
		w.headerBar.rebuildTop()
	}
	if w.ribbon != nil {
		w.ribbon.buildEnd()
		w.ribbonRefresh()
	}
	w.refreshMenu() // Reset the Top Row's state
}

// topRowItem is the menu item putting id on the top row or taking it
// off (nil where it can be neither).
func (w *MainWindow) topRowItem(id string) *fyne.MenuItem {
	c := commands().byID[id]
	switch {
	case c == nil || topRowFixed(id):
		return nil
	case w.onTopRow(id):
		return fyne.NewMenuItem("Remove from the Top Row", func() { w.removeFromTopRow(id) })
	case topRowable(c):
		return fyne.NewMenuItem("Add to the Top Row", func() { w.addToTopRow(id) })
	}
	return nil
}

// showTopRowMenu shows id's top row menu at pos.
func (w *MainWindow) showTopRowMenu(id string, obj fyne.CanvasObject, pos fyne.Position) {
	item := w.topRowItem(id)
	if item == nil {
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(obj)
	if c == nil {
		c = w.fyneWindow.Canvas()
	}
	if pos == (fyne.Position{}) {
		pos = fyne.CurrentApp().Driver().AbsolutePositionForObject(obj).AddXY(0, obj.Size().Height)
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", item), c, pos)
}

// topRank is how much id stays on the top row when it is too narrow:
// the Quick tab's score (its priority and its use), ☰ the last to go, a
// command added by hand before the standard ones.
func (w *MainWindow) topRank(id string, env cmdEnv) float64 {
	switch {
	case id == "":
		return -1 // a gap: the first to go
	case topRowFixed(id):
		return 1e9
	}
	s := w.quick()
	n := s.score(id, quickPriority(env))
	if slices.Contains(w.topRowEdits().added, id) {
		n += 500
	}
	return n
}

// The search field.

// topSearch is the top row's search field: typed in, the palette opens
// with what was typed and takes the rest; narrow (a phone), the palette's
// 🔍 button in its place.
type topSearch struct {
	*fyne.Container
	entry   *widget.Entry
	button  fyne.CanvasObject
	compact bool
}

// topSearchWidth is the field's width on the desktop.
const topSearchWidth = 190

func (w *MainWindow) newTopSearch(button fyne.CanvasObject, compact bool) *topSearch {
	s := &topSearch{button: button, compact: compact}
	e := widget.NewEntry()
	e.SetPlaceHolder("Search commands")
	e.ActionItem = widget.NewIcon(theme.SearchIcon())
	fyne.SetAccessibleLabel(e, "Search commands: type to find a command")
	e.OnChanged = func(text string) {
		if text == "" {
			return
		}
		e.SetText("")
		w.showPaletteWith(text)
	}
	e.OnSubmitted = func(string) { w.showPaletteWith("") }
	s.entry = e
	s.Container = container.New(s, e, button)
	return s
}

func (s *topSearch) MinSize([]fyne.CanvasObject) fyne.Size {
	if s.compact {
		return s.button.MinSize()
	}
	return fyne.NewSize(topSearchWidth, s.entry.MinSize().Height)
}

func (s *topSearch) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	showIf(s.entry, !s.compact)
	showIf(s.button, s.compact)
	var o fyne.CanvasObject = s.entry
	if s.compact {
		o = s.button
	}
	h := min(o.MinSize().Height, size.Height)
	o.Move(fyne.NewPos(0, (size.Height-h)/2))
	o.Resize(fyne.NewSize(size.Width, h))
}

// showPaletteWith opens the palette with text typed in it.
func (w *MainWindow) showPaletteWith(text string) {
	w.showPalette()
	if p := w.palette; p != nil && text != "" {
		p.entry.SetText(p.entry.Text + text)
		p.entry.CursorColumn = len([]rune(p.entry.Text))
		p.entry.Refresh()
	}
}

// pickForTopRow opens the palette to choose a command for the top row.
func (w *MainWindow) pickForTopRow() {
	w.showPalette()
	p := w.palette
	if p == nil {
		return
	}
	p.pick = func(c *command) { w.addToTopRow(c.id) }
	p.hint.SetText("Enter adds it to the top row · ↑↓ choose · Esc closes")
	p.entry.SetPlaceHolder("A command for the top row")
	p.all = slices.DeleteFunc(p.all, func(e paletteEntry) bool { return e.action != nil || !topRowable(e.c) || w.onTopRow(e.c.id) })
	p.filter()
}

// topRowCommands are the top row's own: Add to the Top Row…, Reset.
func topRowCommands() []*command {
	return []*command{
		{id: "view.topRowAdd", label: "Add to the Top Row…", ribbonLabel: "Add to Top Row", icon: res(theme.ContentAddIcon()),
			tip: "A command always in sight on the top row (or right-click any ribbon button)",
			run: (*MainWindow).pickForTopRow},
		{id: "view.topRowReset", label: "Reset the Top Row", ribbonLabel: "Reset Top Row", icon: res(theme.ViewRestoreIcon()),
			tip:     "The standard top row again, without what was added or taken off",
			enabled: func(e cmdEnv) bool { return e.topEdited },
			run:     (*MainWindow).resetTopRow},
	}
}

// renameItem is the … menu's way to the name's editor, the name shown
// (the name itself has no room on the row).
func (w *MainWindow) renameItem() *fyne.MenuItem {
	name := w.scriptName()
	if name == "" {
		name = "Untitled"
	}
	item := fyne.NewMenuItem("Rename "+name+"…", w.renameScript)
	item.Icon = theme.DocumentCreateIcon()
	return item
}

// titleInSight reports whether the script's name is on the top row (a
// phone's may have no room for it).
func (w *MainWindow) titleInSight() bool {
	if w.ribbonShown() {
		return w.ribbon.title.Visible()
	}
	return w.headerBar == nil || w.headerBar.titleLabel.Visible()
}

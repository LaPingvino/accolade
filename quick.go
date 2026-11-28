// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// The Quick tab and the ribbon's order (Joop, 2026-11-22: "for mobile …
// auto reorder the ribbon by usage/priority/utility, and maybe have a
// quick access ribbon that puts the most prio from all together to avoid
// having to scroll to the side all the time").
//
// The ribbon has two orders, switched by the lock on its tab row (beside
// ☰), per window, one tap, live:
//
//   - Smart (a phone's at first): the Quick tab first and chosen, holding
//     the 8–12 most useful commands from every tab; the other tabs' groups
//     ordered by how much they are used.
//   - Fixed (the desktop's at first): the standard order, no Quick tab.
//
// A command's score is a built-in priority for the kind of document and
// the role (quickPriority) plus its use: each time it is given (a button,
// a menu, a key, the palette) its count goes up by one, and the counts
// halve every two weeks. The counts are kept on this device only
// (ribbon-usage.json beside settings.json), never in a script, never
// shared or synced.
//
// Nothing moves under the finger: the counts are read once, when the app
// starts, and the order made from them holds until it starts again.
// Within a session the Quick tab's order is kept (per kind of document)
// and what comes in later (a source chosen: Next Passage, a pin) goes at
// its end; flipping Smart and Fixed shows the same two layouts all
// session.
//
// A long press (or a right-click) on a ribbon button pins it to the Quick
// tab or unpins it; one on a button the Quick tab chose itself takes it
// off. Pins come first, in the order pinned, and are kept with the counts.
//
// The desktop can also have the Quick commands as a row under the ribbon
// (View > Quick Row, Preferences, or a right-click on any ribbon button;
// off at first), like Office's Quick Access toolbar, while the order is Fixed.

const (
	quickTab      = "Quick"
	quickHome     = "quick" // the Quick tab's group's home: its buttons (groupContent)
	quickFileName = "ribbon-usage.json"
	quickRowPref  = "ribbon-quick-row" // the desktop's Quick row (off at first)
	quickSize     = 10                 // what the Quick tab is filled to
	quickMax      = 12                 // the most it holds (pins and what came in later)
	quickHalfLife = 14 * 24 * time.Hour
	quickUseScale = 20.0 // a use's weight against the priorities (log1p of the count: 20 uses ≈ 60)
)

// quickNow is the clock of the usage counts (a test's own).
var quickNow = time.Now

// quickUse is a command's decayed count as of T (Unix seconds).
type quickUse struct {
	N float64 `json:"n"`
	T int64   `json:"t"`
}

// at is the count decayed to now.
func (u quickUse) at(now time.Time) float64 {
	age := now.Sub(time.Unix(u.T, 0))
	if age <= 0 {
		return u.N
	}
	return u.N * math.Exp2(-float64(age)/float64(quickHalfLife))
}

// quickData is the usage file.
type quickData struct {
	Uses   map[string]quickUse `json:"uses"`
	Pins   []string            `json:"pins,omitempty"`   // pinned to the Quick tab, in that order
	Hidden []string            `json:"hidden,omitempty"` // taken off the Quick tab
}

// quickStore is the app's usage counts and its session's Quick order.
type quickStore struct {
	path string
	data quickData
	// frozen are the counts as the app started (decayed to then): every
	// order of the session is made from these
	frozen map[string]float64
	// order is the Quick tab's order this session, per kind of document
	// (true: Markdown); computed: the score's picks made per env
	order    map[bool][]string
	computed map[string]bool
	gen      int // up by one at each pin, unpin, taking off
}

// quick is the app's store, read the first time it is asked for.
func (w *MainWindow) quick() *quickStore {
	if w.app == nil {
		if w.ownQuick == nil {
			w.ownQuick = loadQuickStore(filepath.Join(getConfigDir(), quickFileName))
		}
		return w.ownQuick
	}
	if w.app.quick == nil {
		w.app.quick = loadQuickStore(filepath.Join(getConfigDir(), quickFileName))
	}
	return w.app.quick
}

func loadQuickStore(path string) *quickStore {
	s := &quickStore{path: path, frozen: map[string]float64{}, order: map[bool][]string{}, computed: map[string]bool{}}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s.data)
	}
	if s.data.Uses == nil {
		s.data.Uses = map[string]quickUse{}
	}
	now := quickNow()
	for id, u := range s.data.Uses {
		s.frozen[id] = u.at(now)
	}
	return s
}

// save writes the usage file (whole or not at all).
func (s *quickStore) save() {
	if s.path == "" {
		return
	}
	b, err := json.MarshalIndent(s.data, "", " ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	_ = writeFileAtomic(s.path, append(b, '\n'))
}

// used counts one use of id (the frozen counts stay as they are).
func (s *quickStore) used(id string) {
	now := quickNow()
	u := s.data.Uses[id]
	s.data.Uses[id] = quickUse{N: u.at(now) + 1, T: now.Unix()}
}

// noteUse counts a use of c, saved once things are quiet.
func (w *MainWindow) noteUse(c *command) {
	if c == nil || c.id == "" || !quickCounts(c.id) {
		return
	}
	s := w.quick()
	s.used(c.id)
	w.debounce(&w.quickTimer, 5*time.Second, s.save)
}

// quickCounts reports whether id's use is counted: not the ribbon's own
// (the palette's button, ☰, the lock), not the Preview tab's (it is there
// only while a preview is), not the panes' keys.
func quickCounts(id string) bool {
	switch id {
	case "app.palette", "view.menu", "view.ribbonLock", "view.keyTips", "view.nextPane", "view.previousPane", "set.quickRow":
		return false
	}
	return !strings.HasPrefix(id, "preview.") && !strings.HasPrefix(id, "fixup.") && !strings.HasPrefix(id, "titlepage.")
}

// quickPriority is the built-in priority of the commands for the Quick
// tab, in env: a screenplay's elements, Undo, Save, Find, Preview, the
// source's Next and Adapt while adapting, production's tagging; a
// Markdown document's headings, emphasis, lists and links.
func quickPriority(env cmdEnv) map[string]float64 {
	if env.markdown {
		return map[string]float64{
			"md.headings": 95, "edit.undo": 92, "file.save": 88, "format.bold": 85, "format.italic": 84,
			"md.bullets": 76, "md.link": 74, "edit.find": 72, "view.preview": 70, "md.footnote": 66,
			"md.quote": 55, "md.numbers": 54, "edit.redo": 50, "file.startScreenplay": 45,
		}
	}
	p := map[string]float64{
		elementCommandID(elementScene): 95, "edit.undo": 92, elementCommandID(elementAction): 90,
		elementCommandID(elementCharacter): 88, elementCommandID(elementDialogue): 86, "file.save": 84,
		"edit.find": 75, "view.preview": 72, elementCommandID(elementParenthetical): 70, "edit.redo": 55,
		"write.character": 50, elementCommandID(elementTransition): 45, "file.export": 42, "format.bold": 40,
	}
	if env.source { // adapting: the next passage, and its step
		p["source.next"], p["source.adapt"] = 80, 78
	}
	if !env.writer { // production's (the role puts them away in Writer)
		p["production.tag"], p["view.production"] = 62, 60
	}
	return p
}

// quickCandidates are the commands the Quick tab may hold: those on the
// regular tabs (the elements one by one), not the Preview tab's, in the
// registry's order (the tie's).
func quickCandidates() []string {
	var out []string
	seen := map[string]bool{}
	add := func(id string) {
		if !seen[id] && quickCounts(id) {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, t := range commands().ribbon {
		if t.title == quickTab || t.contextual {
			continue
		}
		for _, g := range t.groups {
			for _, h := range g.homes {
				switch {
				case h.sep:
				case h.id == "elements":
					for _, el := range toolbarElements {
						add(elementCommandID(el))
					}
				case commands().byID[h.id] != nil:
					add(h.id)
				}
			}
		}
	}
	return out
}

// score is id's score in env: its priority and its use as the app
// started.
func (s *quickStore) score(id string, prio map[string]float64) float64 {
	return prio[id] + quickUseScale*math.Log1p(s.frozen[id])
}

// picks are the commands the score chooses in env, pins first: filled to
// quickSize.
func (s *quickStore) picks(env cmdEnv) []string {
	var out []string
	for _, id := range s.data.Pins {
		if c := commands().byID[id]; c != nil && c.applies(env) {
			out = append(out, id)
		}
	}
	prio := quickPriority(env)
	type cand struct {
		id    string
		score float64
	}
	var cands []cand
	for _, id := range quickCandidates() {
		if slices.Contains(out, id) || slices.Contains(s.data.Hidden, id) || !cmd(id).applies(env) {
			continue
		}
		if sc := s.score(id, prio); sc > 0 {
			cands = append(cands, cand{id, sc})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	for _, c := range cands {
		if len(out) >= quickSize {
			break
		}
		out = append(out, c.id)
	}
	return out
}

// quickEnvKey is what the picks depend on, besides the frozen counts.
func quickEnvKey(env cmdEnv) string {
	return strings.Join([]string{boolKey(env.markdown), boolKey(env.source), boolKey(env.writer), boolKey(env.away), boolKey(env.midDoc), boolKey(env.browser)}, "")
}

// list is the Quick tab's commands in env: the session's order, those
// that apply, with the picks not in it yet put at its end (to quickMax).
func (s *quickStore) list(env cmdEnv) []string {
	order := s.order[env.markdown]
	if key := quickEnvKey(env); !s.computed[key] {
		s.computed[key] = true
		for _, id := range s.picks(env) {
			if !slices.Contains(order, id) {
				order = append(order, id)
			}
		}
	}
	var out, kept []string
	for _, id := range order {
		c := commands().byID[id]
		if c == nil || slices.Contains(s.data.Hidden, id) {
			continue
		}
		if c.applies(env) {
			if len(out) >= quickMax && !slices.Contains(s.data.Pins, id) {
				continue // past the most: not kept (a pin always is)
			}
			out = append(out, id)
		}
		kept = append(kept, id)
	}
	s.order[env.markdown] = kept
	return out
}

// pinned reports whether id is pinned to the Quick tab.
func (s *quickStore) pinned(id string) bool { return slices.Contains(s.data.Pins, id) }

// pin pins id to the Quick tab, at its end.
func (s *quickStore) pin(id string) {
	if s.pinned(id) {
		return
	}
	s.data.Pins = append(s.data.Pins, id)
	s.data.Hidden = slices.DeleteFunc(s.data.Hidden, func(x string) bool { return x == id })
	for _, md := range []bool{false, true} {
		if !slices.Contains(s.order[md], id) {
			s.order[md] = append(s.order[md], id)
		}
	}
	s.gen++
	s.save()
}

// unpin takes id off the Quick tab: unpinned if pinned, else kept off it
// (hidden) until pinned again.
func (s *quickStore) unpin(id string) {
	if s.pinned(id) {
		s.data.Pins = slices.DeleteFunc(s.data.Pins, func(x string) bool { return x == id })
	} else if !slices.Contains(s.data.Hidden, id) {
		s.data.Hidden = append(s.data.Hidden, id)
	}
	for md := range s.order {
		s.order[md] = slices.DeleteFunc(s.order[md], func(x string) bool { return x == id })
	}
	s.gen++
	s.save()
}

// groupUse is how much a group was used as the app started.
func (s *quickStore) groupUse(g ribbonGroup) float64 {
	var n float64
	for _, h := range g.homes {
		if h.id == "elements" {
			for _, el := range toolbarElements {
				n += s.frozen[elementCommandID(el)]
			}
			continue
		}
		n += s.frozen[h.id]
	}
	return n
}

// groupsOf are t's groups in the ribbon's order: the standard one, or
// in Smart the most used first (a Production stage's group stays first;
// unused groups keep their standard order after the used ones).
func (r *ribbonBar) groupsOf(t ribbonTab) []ribbonGroup {
	if !r.env.smart || t.title == quickTab || t.contextual {
		return t.groups
	}
	s := r.w.quick()
	gs := slices.Clone(t.groups)
	sort.SliceStable(gs, func(i, j int) bool {
		si, sj := gs[i].stage != 0, gs[j].stage != 0
		if si != sj {
			return si
		}
		return s.groupUse(gs[i]) > s.groupUse(gs[j])
	})
	return gs
}

// The order: Smart or Fixed.

// ribbonOrderPref is the order the lock was last set to on a phone or
// the desktop (a new window's, and after a restart): "smart", "fixed", or
// "" (Smart on a phone, Fixed on the desktop).
func ribbonOrderPref(phone bool) string {
	if phone {
		return "ribbon-order-phone"
	}
	return "ribbon-order-desktop"
}

// ribbonSmart reports whether the window's ribbon is in the Smart order:
// as the lock was set in this window, else as it was last set on a
// phone or the desktop, else Smart on a phone and Fixed on the desktop.
func (w *MainWindow) ribbonSmart() bool {
	order := w.ribbonOrder
	if order == "" && w.settings != nil {
		order = w.settings.GetString(ribbonOrderPref(w.narrow))
	}
	switch order {
	case "smart":
		return true
	case "fixed":
		return false
	}
	return w.narrow
}

// toggleRibbonOrder is the lock's tap: Smart to Fixed or back, in this
// window, the tab in front kept if both have it.
func (w *MainWindow) toggleRibbonOrder() {
	if w.app != nil { // the others keep theirs, whatever this one saves
		for _, x := range w.app.windows {
			if x != w && x.ribbonOrder == "" {
				x.ribbonOrder = map[bool]string{true: "smart", false: "fixed"}[x.ribbonSmart()]
			}
		}
	}
	defer func() { // remembered for a new window, and the next start
		if w.settings != nil {
			w.settings.SetString(ribbonOrderPref(w.narrow), w.ribbonOrder)
			_ = w.settings.Save()
		}
	}()
	if w.ribbonSmart() {
		w.ribbonOrder = "fixed"
		w.setStatus("Ribbon order: Fixed — the standard order (the lock again for Smart)")
	} else {
		w.ribbonOrder = "smart"
		w.setStatus("Ribbon order: Smart — the Quick tab first, groups by use (the lock again for Fixed)")
	}
	w.ribbonRefresh()
	w.refreshMenu() // the item's check
}

// quickLabels are the Quick tab's labels where a tab's short one needs
// its group ("Next" is Adaptation > Passages' next source passage).
var quickLabels = map[string]string{
	"source.next": "Next Passage", "source.previous": "Previous Passage", "source.adapt": "Adapt",
	"edit.findNext": "Find Next", "edit.findPrevious": "Find Previous", "source.update": "Update Links",
	"source.linkUp": "Link Up", "source.link": "Link", "production.nextScene": "Next Scene",
	"view.production": "Production", "file.newEmpty": "New Empty",
}

// quickPhoneLabels are elements' labels on a phone's Quick tab (the
// name and tooltip stay the full one).
var quickPhoneLabels = map[string]string{elementScene: "Scene", elementParenthetical: "Paren."}

// quickLabel is c's label on the Quick tab: one that stands alone.
func quickLabel(c *command, env cmdEnv) string {
	if l, ok := quickLabels[c.id]; ok {
		return l
	}
	return c.ribbonLabelIn(env)
}

// quickButtons are the Quick commands' buttons, as on the Quick tab (an
// element's as the Write tab has it).
func (r *ribbonBar) quickButtons() []*ribbonButton {
	var out []*ribbonButton
	for _, id := range r.w.quick().list(r.env) {
		c := cmd(id)
		b := newRibbonButton(r, c, false)
		if el, ok := strings.CutPrefix(id, "format.element."); ok {
			b.label, b.mark = el, elementMarks[el]
			if short, ok := quickPhoneLabels[el]; ok && r.phone {
				b.label = short // one more in a phone's width
			}
			b.keys = c.keysShort()
		} else {
			b.label = quickLabel(c, r.env) // its name stays the full label
		}
		b.quick = true
		out = append(out, b)
	}
	return out
}

// quickRows lays the phone's Quick buttons out in two rows, in their
// order, row by row: the first row the larger half (5+5, 6+5, 6+6), so a
// phone shows them all without a swipe.
func quickRows(bs []*ribbonButton) fyne.CanvasObject {
	n := (len(bs) + 1) / 2
	row := func(bs []*ribbonButton) fyne.CanvasObject {
		objs := make([]fyne.CanvasObject, len(bs))
		for i, b := range bs {
			objs[i] = b
		}
		return container.New(layout.NewCustomPaddedHBoxLayout(2), objs...)
	}
	return container.New(layout.NewCustomPaddedVBoxLayout(2), row(bs[:n]), row(bs[n:]))
}

// quickRowShown reports whether the desktop's Quick row shows: set, in
// the Fixed order (Smart has the tab), not on a phone.
func (r *ribbonBar) quickRowShown() bool {
	w := r.w
	return w.settings != nil && w.settings.GetBoolean(quickRowPref) && !r.env.smart && !w.narrow
}

// buildQuickRow makes the desktop's Quick row again (or hides it).
func (r *ribbonBar) buildQuickRow() {
	for _, b := range r.rowButtons {
		b.hover.Out(r.w.tooltips, b.hoverTip())
	}
	r.rowButtons = nil
	r.quickRow.Objects = nil
	if r.quickRowShown() {
		for _, b := range r.quickButtons() {
			b.row = true
			b.keys = "" // the tooltip has them: the row stays narrow
			r.rowButtons = append(r.rowButtons, b)
			r.quickRow.Add(b)
		}
	}
	shown := len(r.rowButtons) > 0
	setVisible(r.quickRowBox, shown)
	r.quickRow.Refresh()
}

// quickMenu is a ribbon button's menu (a long press, a right-click): pin
// it to the Quick tab, or take it off.
func (r *ribbonBar) quickMenu(b *ribbonButton) *fyne.Menu {
	if b.c == nil || !quickCounts(b.c.id) || b.onTabRow() {
		return nil
	}
	id := b.c.id
	s := r.w.quick()
	on := slices.Contains(s.list(r.env), id)
	var item *fyne.MenuItem
	switch {
	case s.pinned(id):
		item = fyne.NewMenuItem("Unpin from Quick", func() { r.w.unpinQuick(id) })
	case on:
		item = fyne.NewMenuItem("Remove from Quick", func() { r.w.unpinQuick(id) })
	default:
		item = fyne.NewMenuItem("Pin to Quick", func() { r.w.pinQuick(id) })
	}
	items := []*fyne.MenuItem{item}
	if row := cmd("set.quickRow"); row.applies(r.env) { // the desktop's, as Office's right-click has it
		it := fyne.NewMenuItem("Show the Quick Row", func() { row.run(r.w) })
		if r.quickRowShown() {
			it.Label = "Hide the Quick Row"
		}
		items = append(items, fyne.NewMenuItemSeparator(), it)
	}
	return fyne.NewMenu("", items...)
}

// pinQuick pins id to the Quick tab, at its end.
func (w *MainWindow) pinQuick(id string) {
	s := w.quick()
	if len(s.list(w.cmdEnv())) >= quickMax {
		w.setStatus("The Quick tab is full: unpin or remove one first (press and hold it)")
		return
	}
	s.pin(id)
	w.setStatus(cmd(id).labelIn(w.cmdEnv()) + " pinned to the Quick tab")
	w.quickChanged()
}

// unpinQuick takes id off the Quick tab.
func (w *MainWindow) unpinQuick(id string) {
	pinned := w.quick().pinned(id)
	w.quick().unpin(id)
	if pinned {
		w.setStatus(cmd(id).labelIn(w.cmdEnv()) + " unpinned from the Quick tab")
	} else {
		w.setStatus(cmd(id).labelIn(w.cmdEnv()) + " taken off the Quick tab (pin it to have it back)")
	}
	w.quickChanged()
}

// quickChanged shows a pin's change in every window's ribbon.
func (w *MainWindow) quickChanged() {
	ws := []*MainWindow{w}
	if w.app != nil {
		for _, x := range w.app.windows {
			if x != w {
				ws = append(ws, x)
			}
		}
	}
	for _, x := range ws {
		x.ribbonRefresh()
	}
}

// showQuickMenu shows b's menu at pos (or under it).
func (r *ribbonBar) showQuickMenu(b *ribbonButton, pos fyne.Position) bool {
	m := r.buttonMenu(b)
	if m == nil {
		return false
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(b)
	if c == nil {
		return false
	}
	if pos == (fyne.Position{}) {
		pos = fyne.CurrentApp().Driver().AbsolutePositionForObject(b).AddXY(0, b.Size().Height)
	}
	b.hover.Out(r.w.tooltips, b.hoverTip())
	widget.ShowPopUpMenuAtPosition(m, c, pos)
	return true
}

// The commands: the lock and the desktop's Quick row.
func quickCommands() []*command {
	lock := &command{id: "view.ribbonLock", label: "Lock Ribbon Order", ribbonLabel: "Lock", icon: res(unlockIcon),
		when: func(e cmdEnv) bool { return e.ribbon },
		iconFor: func(w *MainWindow) fyne.Resource {
			if w.ribbonSmart() {
				return unlockIcon
			}
			return lockIcon
		},
		nameFor: func(w *MainWindow) string {
			if w.ribbonSmart() {
				return "Ribbon order: Smart, the Quick tab first and groups by use (tap to lock to the standard order)"
			}
			return "Ribbon order: Fixed, the standard order (tap for Smart: the Quick tab first, groups by use)"
		},
		checked: func(w *MainWindow) bool { return !w.ribbonSmart() },
		run:     (*MainWindow).toggleRibbonOrder}
	row := toggleSetting("set.quickRow", "Quick Row", quickRowPref, quickIcon, nil)
	row.tip = "The Quick commands in a row under the ribbon, while the order is Fixed"
	row.when = func(e cmdEnv) bool { return e.ribbon && !e.smart && !e.phone } // Smart and a phone have the tab
	return []*command{lock, row}
}

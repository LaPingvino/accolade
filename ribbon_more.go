package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// A group's More tab: its launcher (the ⋯ in the group's corner,
// ribbon.go: groupLauncher) opens a contextual tab of the group's less
// common commands and its settings, "More Output", and brings it to the
// front. On it:
//
//   - Close (✕) first, in sight however far the groups scroll; the tab
//     has its ✕ too, and Esc on the ribbon closes it. Either goes back
//     to the tab the launcher was on, the focus on that launcher.
//     Choosing another tab closes it as well.
//   - The group's commands, as its own buttons are.
//   - Each submenu (New from Structure, Theme, Language for New Scripts)
//     a group of its own, its items buttons, a chosen one ✓; a level
//     deeper (More variants) a button with its menu, whatever it holds.
//   - Settings: the group's settings that are on or off, ✓ when on, and
//     its other settings (Smaller, Larger).
//
// On a phone it is in the switcher's menu with the other contextual tabs.

// moreTabTitle is the More tab's title in the registry; the tab shows
// the group's name ("More Output": moreOpen.name).
const moreTabTitle = "More"

// moreTab is the More tab: its groups are made from the group whose
// launcher opened it (moreGroups).
func moreTab() ribbonTab {
	return ribbonTab{title: moreTabTitle, contextual: true, forward: true, when: func(e cmdEnv) bool { return e.moreOpen }}
}

// moreOpen is the More tab open: the tab and group it is of, the group
// (as it was laid out), its name; gen counts the changes made on it (an
// item's choice), so its buttons are made again.
type moreOpen struct {
	tab, caption string
	g            ribbonGroup
	name         string
	gen          int
}

// moreName is the More tab's name for a group: "More Output", "More
// Production Options" for a tab's Options group.
func moreName(tab, caption string) string {
	switch caption {
	case "Options", "Settings":
		return "More " + tab + " " + caption
	}
	return "More " + caption
}

// tabName is the name a tab shows: the More tab's its group's.
func (r *ribbonBar) tabName(title string) string {
	if title == moreTabTitle && r.more != nil {
		return r.more.name
	}
	return title
}

// moreKey is what the More tab's groups were made for, for refresh.
func (r *ribbonBar) moreKey() string {
	if r.more == nil {
		return ""
	}
	return r.more.tab + "\x00" + r.more.caption + "\x00" + string(rune('0'+r.more.gen%10))
}

// openMore opens g's More tab (of tab, captioned caption) and brings it
// to the front, the focus on its first button.
func (r *ribbonBar) openMore(tab, caption string, g ribbonGroup) {
	from := r.selected
	if r.more != nil {
		from = r.more.tab
	}
	if t, ok := tabOf(from); !ok || !t.contextual {
		r.beforeContext = from
	}
	r.more = &moreOpen{tab: from, caption: caption, g: g, name: moreName(tab, caption)}
	r.ctxShown[moreTabTitle] = true // came by hand: followContext has nothing to follow
	r.selected, r.picked = moreTabTitle, true
	r.leaveKeyTips(false)
	for _, tb := range r.tabs {
		if tb.title == moreTabTitle {
			tb.Refresh() // its name
		}
	}
	r.refresh()
	for _, b := range r.buttons { // the first after Close
		if !b.Disabled() && b.label != "Close" {
			r.w.fyneWindow.Canvas().Focus(b)
			return
		}
	}
	r.down()
}

// closeMore closes the More tab, if open: back to the tab it was opened
// on, the focus (back) on its group's launcher.
func (r *ribbonBar) closeMore() bool {
	m := r.more
	if m == nil {
		return false
	}
	r.dropMore()
	r.selected = m.tab
	r.refresh()
	for _, b := range r.buttons {
		if b.launcher && b.moreOf == m.caption {
			r.w.fyneWindow.Canvas().Focus(b)
			return true
		}
	}
	r.up()
	return true
}

// dropMore forgets the More tab (another tab chosen, the ribbon gone).
func (r *ribbonBar) dropMore() {
	r.more = nil
	r.ctxShown[moreTabTitle] = false
	for _, tb := range r.tabs {
		if tb.title == moreTabTitle {
			tb.Refresh()
		}
	}
}

// moreEscape is Esc on the ribbon with the More tab chosen: it closes.
func (r *ribbonBar) moreEscape() bool {
	return r.more != nil && r.selected == moreTabTitle && r.closeMore()
}

// moreGroup is a group of the More tab: its caption and buttons.
type moreGroup struct {
	caption string
	content fyne.CanvasObject
}

// moreGroups are the More tab's groups: Close, the group's commands,
// each submenu's, its settings.
func (r *ribbonBar) moreGroups() []moreGroup {
	m := r.more
	if m == nil {
		return nil
	}
	w, env := r.w, r.env
	closeB := newRibbonButton(r, nil, !r.phone)
	closeB.label, closeB.name = "Close", "Close "+m.name+" (Esc)"
	closeB.icon = theme.CancelIcon()
	closeB.onTap = func() { r.closeMore() }
	r.add(closeB)
	out := []moreGroup{{"Close", closeB}}
	var plain, toggles []home
	var subs []*command
	isSetting := func(c *command) bool { return strings.HasPrefix(c.id, "set.") }
	seen := map[string]bool{}
	for _, h := range append(append([]home{}, m.g.more...), m.g.homes...) {
		if h.id == "" || seen[h.id] {
			continue
		}
		c, ok := commands().byID[h.id]
		if !ok || !c.applies(env) {
			continue
		}
		fromHomes := !containsHome(m.g.more, h.id)
		switch {
		case fromHomes && !isSetting(c): // the group's own buttons stay on its tab
			continue
		case c.submenu != nil:
			subs = append(subs, c)
		case isSetting(c):
			toggles = append(toggles, at(h.id))
		default:
			plain = append(plain, at(h.id))
		}
		seen[h.id] = true
	}
	if content := r.groupContent(ribbonGroup{homes: plain}); content != nil {
		out = append(out, moreGroup{m.caption, content})
	}
	for _, c := range subs { // the commands' submenus, then the settings' (Theme, Font)
		if !isSetting(c) {
			out = append(out, r.menuGroups(c.labelIn(env), c.submenu(w).Items)...)
		}
	}
	var settings []fyne.CanvasObject
	for _, h := range toggles {
		c := cmd(h.id)
		b := newRibbonButton(r, c, false)
		b.label = c.labelIn(env) // the whole name: no group around it says the rest
		b.toggle = c.checked != nil
		r.add(b)
		b.follow(env)
		settings = append(settings, b)
	}
	if len(settings) > 0 {
		out = append(out, moreGroup{"Settings", r.moreColumns([][]fyne.CanvasObject{settings})})
	}
	for _, c := range subs {
		if isSetting(c) {
			out = append(out, r.menuGroups(c.labelIn(env), c.submenu(w).Items)...)
		}
	}
	return out
}

// containsHome reports whether homes has id.
func containsHome(homes []home, id string) bool {
	for _, h := range homes {
		if h.id == id {
			return true
		}
	}
	return false
}

// menuGroups is a submenu as the More tab's group captioned caption:
// each item a button (✓ if chosen), a separator a new column, an item
// with a menu of its own a button opening it (its depth whole).
func (r *ribbonBar) menuGroups(caption string, items []*fyne.MenuItem) []moreGroup {
	var runs [][]fyne.CanvasObject
	var run []fyne.CanvasObject
	for _, it := range items {
		if it.IsSeparator {
			if len(run) > 0 {
				runs, run = append(runs, run), nil
			}
			continue
		}
		run = append(run, r.itemButton(it))
	}
	if len(run) > 0 {
		runs = append(runs, run)
	}
	if len(runs) == 0 {
		return nil
	}
	return []moreGroup{{caption, r.moreColumns(runs)}}
}

// itemButton is a submenu item's button on the More tab.
func (r *ribbonBar) itemButton(it *fyne.MenuItem) *ribbonButton {
	b := newRibbonButton(r, nil, false)
	b.label, b.name = it.Label, it.Label
	b.icon = it.Icon
	b.disabled = it.Disabled
	if sub := it.ChildMenu; sub != nil {
		b.label = strings.TrimSuffix(it.Label, "…") + " ▾" // a menu, not a dialog
		b.name = it.Label + ": a menu"
		b.onTap = func() { showMenuBy(r.w, b, sub) }
	} else {
		b.toggle, b.checked = true, it.Checked
		action := it.Action
		b.onTap = func() {
			if action == nil {
				return
			}
			at := -1 // the focus kept on it, made again
			for i, x := range r.buttons {
				if x == b && r.w.fyneWindow.Canvas().Focused() == fyne.Focusable(b) {
					at = i
				}
			}
			action()
			if r.more == nil || r.selected != moreTabTitle {
				return
			}
			r.more.gen++
			r.refresh()
			if at >= 0 && at < len(r.buttons) {
				r.w.fyneWindow.Canvas().Focus(r.buttons[at])
			}
		}
	}
	r.add(b)
	return b
}

// moreColumns lays runs of small buttons out as the ribbon's groups
// have them: three to a column, a run a column of its own (all one row
// on a phone).
func (r *ribbonBar) moreColumns(runs [][]fyne.CanvasObject) fyne.CanvasObject {
	var cols []fyne.CanvasObject
	for _, run := range runs {
		if r.phone {
			cols = append(cols, run...)
			continue
		}
		for len(run) > 0 {
			n := min(3, len(run))
			cols = append(cols, container.New(ribbonRows{}, run[:n]...))
			run = run[n:]
		}
	}
	return container.New(layout.NewCustomPaddedHBoxLayout(theme.Padding()), cols...)
}

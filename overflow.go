// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/tooltip"
)

// overflowItem is one thing on an overflow row, and what it is in the
// row's menu when it does not fit (menu builds the items when the menu
// opens, with what a picker can open next to; nil for a gap).
type overflowItem struct {
	obj  fyne.CanvasObject
	menu func(anchor fyne.CanvasObject) []*fyne.MenuItem
	keep bool // stays on the row however narrow (a pane's toggle)
	off  bool // not there at all, on the row or in the menu (a role's: role.go)
}

// hidden reports whether item i is in the menu (or off), not on the row.
func (r *overflowRow) hidden(i int) bool {
	if r.items[i].off {
		return true
	}
	if r.out != nil {
		return r.out[i]
	}
	return i >= r.first && !r.items[i].keep
}

// setOff puts the item showing o away (or back).
func (r *overflowRow) setOff(o fyne.CanvasObject, off bool) {
	for i := range r.items {
		if r.items[i].obj == o && r.items[i].off != off {
			r.items[i].off = off
			r.Container.Refresh()
		}
	}
}

// overflowRow puts its items left to right, as many as fit, and the rest
// in the menu of a "more" button at its end, so a narrow window (a phone)
// shows a row of buttons without scrolling it sideways; an item kept stays
// on the row, the others before it going first. A fill object,
// if any, takes the room left over (the document's name) and is the
// first to go — unless the row keeps room for it (keepFill: the name
// before the less-used icons, toprow.go), when it goes only once no
// item is left to put in the menu, and fillMenu is then in the menu
// (the name's editor). With rank set, the items with the lowest rank
// go first, not the last ones.
type overflowRow struct {
	*fyne.Container
	items []overflowItem
	fill  fyne.CanvasObject
	at    int // where fill goes among the items
	more  *tooltip.Button
	first int // the first item in the menu
	// out: which items are in the menu, when ranked (nil: from first on)
	out      []bool
	rank     func(i int) float64
	keepFill float32 // the room kept for fill (0: none)
	fillMenu func() []*fyne.MenuItem
}

func newOverflowRow(tips *tooltip.Layer, items []overflowItem, fill fyne.CanvasObject, at int) *overflowRow {
	r := &overflowRow{items: items, fill: fill, at: at, first: len(items)}
	r.more = tooltip.NewButton(theme.MoreHorizontalIcon(), "More", nil, tips)
	r.more.Importance = widget.LowImportance
	r.more.OnTapped = func() {
		menu := r.menuItems(r.more)
		c := fyne.CurrentApp().Driver().CanvasForObject(r.more)
		if c == nil || len(menu) == 0 {
			return
		}
		at := fyne.CurrentApp().Driver().AbsolutePositionForObject(r.more)
		widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", menu...), c, at.AddXY(0, r.more.Size().Height))
	}
	objects := make([]fyne.CanvasObject, 0, len(items)+2)
	for _, it := range items {
		objects = append(objects, it.obj)
	}
	if fill != nil {
		objects = append(objects, fill)
	}
	r.Container = container.New(r, append(objects, r.more)...)
	return r
}

// menuItems are the "more" menu's: the fill's (hidden), then the items
// not on the row.
func (r *overflowRow) menuItems(anchor fyne.CanvasObject) []*fyne.MenuItem {
	var menu []*fyne.MenuItem
	if r.fill != nil && r.fillMenu != nil && !r.fill.Visible() {
		menu = append(menu, r.fillMenu()...)
	}
	for i, it := range r.items {
		if r.hidden(i) && !it.off && it.menu != nil {
			menu = append(menu, it.menu(anchor)...)
		}
	}
	return menu
}

// MinSize is the "more" button's: everything else can go in its menu.
func (r *overflowRow) MinSize(objects []fyne.CanvasObject) fyne.Size {
	h := r.more.MinSize().Height
	for _, o := range objects {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(r.more.MinSize().Width, h)
}

// natural is the row's width with every item on it, and the fill at its
// least (or the room kept for it).
func (r *overflowRow) natural() float32 {
	pad := theme.Padding()
	all := float32(0)
	for _, it := range r.items {
		if !it.off {
			all += it.obj.MinSize().Width + pad
		}
	}
	if r.fill != nil {
		all += max(r.fill.MinSize().Width, r.keepFill) + pad
	}
	return all
}

func (r *overflowRow) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	pad := theme.Padding()
	width := func(i int) float32 {
		if r.items[i].off {
			return 0
		}
		return r.items[i].obj.MinSize().Width + pad
	}
	onRow := func() float32 {
		n := float32(0)
		for i := range r.items {
			if !r.hidden(i) {
				n += width(i)
			}
		}
		return n
	}
	reserve := float32(0) // the fill's room, kept before the items' (keepFill)
	if r.fill != nil && r.keepFill > 0 {
		reserve = r.keepFill + pad
	}
	room := size.Width
	r.first, r.out = len(r.items), nil
	all := onRow()
	overflowed := all+reserve > room+pad
	if overflowed { // some go in the menu
		room -= width2(r.more, pad)
		r.layoutOut(room, reserve, width)
		all = onRow()
		if reserve > 0 && all+reserve > room+pad { // no room for the fill even so: it goes, the items take its room
			reserve = 0
			r.layoutOut(room, 0, width)
			all = onRow()
		}
	}
	inMenu := false
	for i := range r.items {
		if r.hidden(i) && !r.items[i].off {
			inMenu = true
		}
	}
	leftover := size.Width - all
	if overflowed {
		leftover -= width2(r.more, pad)
	}
	fillShown := false
	if r.fill != nil {
		if r.keepFill > 0 {
			fillShown = reserve > 0 || !overflowed
		} else {
			fillShown = !inMenu && leftover >= r.fill.MinSize().Width
		}
		showIf(r.fill, fillShown)
	}
	showIf(r.more, inMenu || r.fill != nil && !fillShown && r.fillMenu != nil && overflowed)
	x := float32(0)
	place := func() {
		if r.fill != nil && r.fill.Visible() {
			fh := min(r.fill.MinSize().Height, size.Height)
			r.fill.Move(fyne.NewPos(x, (size.Height-fh)/2))
			r.fill.Resize(fyne.NewSize(max(leftover-pad, 0), fh))
			x += leftover
		}
	}
	for i, it := range r.items {
		if i == r.at {
			place()
		}
		showIf(it.obj, !r.hidden(i))
		if r.hidden(i) {
			continue
		}
		w := it.obj.MinSize().Width
		it.obj.Move(fyne.NewPos(x, (size.Height-it.obj.MinSize().Height)/2))
		it.obj.Resize(fyne.NewSize(w, it.obj.MinSize().Height))
		x += w + pad
	}
	if r.at >= len(r.items) {
		place()
	}
	if r.more.Visible() {
		w := r.more.MinSize().Width
		r.more.Move(fyne.NewPos(size.Width-w, (size.Height-r.more.MinSize().Height)/2))
		r.more.Resize(fyne.NewSize(w, r.more.MinSize().Height))
	}
}

// width2 is o's width on a row.
func width2(o fyne.CanvasObject, pad float32) float32 { return o.MinSize().Width + pad }

// layoutOut chooses the items for the menu with room left for them and
// reserve kept: from the end (first), or the lowest ranked (out); kept
// items stay.
func (r *overflowRow) layoutOut(room, reserve float32, width func(int) float32) {
	pad := theme.Padding()
	if r.rank == nil {
		kept := float32(0) // the kept items' room
		for i, it := range r.items {
			if it.keep {
				kept += width(i)
			}
		}
		used := reserve
		r.first = len(r.items)
		for i, it := range r.items {
			if it.keep {
				kept -= width(i)
			}
			if used+width(i)+kept > room+pad {
				r.first = i
				break
			}
			used += width(i)
		}
		return
	}
	r.out = make([]bool, len(r.items))
	used := reserve
	for i := range r.items {
		used += width(i)
	}
	order := make([]int, 0, len(r.items))
	for i, it := range r.items {
		if !it.keep && !it.off && it.menu != nil {
			order = append(order, i)
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return r.rank(order[a]) < r.rank(order[b]) })
	for _, i := range order {
		if used <= room+pad {
			break
		}
		r.out[i] = true
		used -= width(i)
	}
	r.first = len(r.items)
	for i := range r.items {
		if r.out[i] {
			r.first = i
			break
		}
	}
}

// showIf shows or hides o, if it is not already (not again: that would
// lay the row out again).
func showIf(o fyne.CanvasObject, show bool) {
	if show && !o.Visible() {
		o.Show()
	} else if !show && o.Visible() {
		o.Hide()
	}
}

// buttonItem is a tooltip button on an overflow row: in the menu, its tip
// without the shortcut, and its icon.
func buttonItem(b *tooltip.Button) overflowItem {
	return overflowItem{obj: b, menu: func(fyne.CanvasObject) []*fyne.MenuItem {
		label, _, _ := strings.Cut(b.Tip, " (")
		item := fyne.NewMenuItem(label, b.OnTapped)
		item.Icon = b.Icon
		return []*fyne.MenuItem{item}
	}}
}

// gapItem is a little space between groups, nothing in the menu.
func gapItem() overflowItem { return overflowItem{obj: gap()} }

// keptItem is a tooltip button that stays on the row however narrow.
func keptItem(b *tooltip.Button) overflowItem {
	it := buttonItem(b)
	it.keep = true
	return it
}

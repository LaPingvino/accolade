package main

import (
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
}

// overflowRow puts its items left to right, as many as fit, and the rest
// in the menu of a "more" button at its end, so a narrow window (a phone)
// shows a row of buttons without scrolling it sideways. A fill object,
// if any, takes the room left over (the document's name) and is the
// first to go.
type overflowRow struct {
	*fyne.Container
	items []overflowItem
	fill  fyne.CanvasObject
	at    int // where fill goes among the items
	more  *tooltip.Button
	first int // the first item in the menu
}

func newOverflowRow(tips *tooltip.Layer, items []overflowItem, fill fyne.CanvasObject, at int) *overflowRow {
	r := &overflowRow{items: items, fill: fill, at: at, first: len(items)}
	r.more = tooltip.NewButton(theme.MoreHorizontalIcon(), "More", nil, tips)
	r.more.Importance = widget.LowImportance
	r.more.OnTapped = func() {
		var menu []*fyne.MenuItem
		for _, it := range r.items[r.first:] {
			if it.menu != nil {
				menu = append(menu, it.menu(r.more)...)
			}
		}
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

// MinSize is the "more" button's: everything else can go in its menu.
func (r *overflowRow) MinSize(objects []fyne.CanvasObject) fyne.Size {
	h := r.more.MinSize().Height
	for _, o := range objects {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(r.more.MinSize().Width, h)
}

func (r *overflowRow) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	pad := theme.Padding()
	width := func(o fyne.CanvasObject) float32 { return o.MinSize().Width + pad }
	all := float32(0)
	for _, it := range r.items {
		all += width(it.obj)
	}
	room := size.Width
	r.first = len(r.items)
	if all > room+pad { // some go in the menu
		room -= width(r.more)
		used := float32(0)
		for i, it := range r.items {
			if used+width(it.obj) > room+pad {
				r.first = i
				break
			}
			used += width(it.obj)
		}
		all = used
	}
	showIf(r.more, r.first < len(r.items))
	leftover := size.Width - all
	if r.first < len(r.items) {
		leftover = 0
	}
	if r.fill != nil {
		showIf(r.fill, leftover >= r.fill.MinSize().Width)
	}
	x := float32(0)
	for i, it := range r.items {
		if i == r.at && r.fill != nil && r.fill.Visible() {
			r.fill.Move(fyne.NewPos(x, 0))
			r.fill.Resize(fyne.NewSize(leftover-pad, size.Height))
			x += leftover
		}
		showIf(it.obj, i < r.first)
		if i >= r.first {
			continue
		}
		w := it.obj.MinSize().Width
		it.obj.Move(fyne.NewPos(x, (size.Height-it.obj.MinSize().Height)/2))
		it.obj.Resize(fyne.NewSize(w, it.obj.MinSize().Height))
		x += w + pad
	}
	if r.more.Visible() {
		w := r.more.MinSize().Width
		r.more.Move(fyne.NewPos(size.Width-w, (size.Height-r.more.MinSize().Height)/2))
		r.more.Resize(fyne.NewSize(w, r.more.MinSize().Height))
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

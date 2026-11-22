package main

import (
	"image/color"
	"net/url"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"github.com/LaPingvino/lexington/pdf"
)

// The page view's links (a PDF's, as Lexington recorded them): the
// pointer a hand over one, the link lit and where it goes in a tip; a
// click or a tap follows it — an address opens in the browser (the web's
// and mail's only: http, https, mailto), a place in the document (a
// heading, a note and back) scrolls to it. Tab and Shift+Tab go through
// the links from the page in sight (out of the pages after the last),
// Enter follows the one with the focus, ringed.

// openLink opens an address in the browser (tests set it).
var openLink = func(u *url.URL) { fyne.CurrentApp().OpenURL(u) }

// drawnLink is a link of the drawing: its page's index and the link.
type drawnLink struct {
	page int
	pdf.Link
}

// linkMarks are the view's lit link (the pointer over it) and ringed one
// (the keyboard's), in the column over the pages.
type linkMarks struct {
	lit, ring *canvas.Rectangle
}

func newLinkMarks() linkMarks {
	m := linkMarks{lit: canvas.NewRectangle(color.NRGBA{0x1a, 0x0d, 0xab, 0x22}), ring: canvas.NewRectangle(color.Transparent)}
	m.ring.StrokeWidth = 2
	m.ring.CornerRadius = 2
	m.lit.CornerRadius = 2
	m.lit.Hide()
	m.ring.Hide()
	return m
}

// links are the drawing's links in their order: page by page, as drawn.
func drawingLinks(d *pdf.Drawing) []drawnLink {
	var out []drawnLink
	if d == nil {
		return nil
	}
	for i, p := range d.Pages {
		for _, l := range p.Links {
			if l.W > 0 && l.H > 0 {
				out = append(out, drawnLink{i, l})
			}
		}
	}
	return out
}

// pageX is where page i's left edge is in the column.
func (v *pageView) pageX(i int) float32 {
	pw := float32(v.d.Pages[i].Width) * v.scale
	return max((v.column.Size().Width-pw)/2, pageGap)
}

// linkRect is a link's place and size in the column.
func (v *pageView) linkRect(l drawnLink) (fyne.Position, fyne.Size) {
	tops, _ := v.pageTops()
	return fyne.NewPos(v.pageX(l.page)+float32(l.X)*v.scale, tops[l.page]+float32(l.Y)*v.scale),
		fyne.NewSize(float32(l.W)*v.scale, float32(l.H)*v.scale)
}

// linkAt is the link at a point of the pages' area (its index in
// v.links, -1 if none).
func (v *pageView) linkAt(at fyne.Position) int {
	if v.d == nil {
		return -1
	}
	at = at.Add(v.scroll.Offset)
	for i, l := range v.links {
		pos, size := v.linkRect(l)
		if at.X >= pos.X && at.X <= pos.X+size.Width && at.Y >= pos.Y && at.Y <= pos.Y+size.Height {
			return i
		}
	}
	return -1
}

// linkTip says where a link goes.
func (v *pageView) linkTip(l drawnLink) string {
	if l.URL != "" {
		return l.URL
	}
	if l.Page > 0 {
		return "Page " + strconv.Itoa(l.Page) + " of " + strconv.Itoa(len(v.d.Pages))
	}
	return ""
}

// followLink follows link i: an address opened, a place scrolled to.
func (v *pageView) followLink(i int) {
	if i < 0 || i >= len(v.links) {
		return
	}
	l := v.links[i]
	if l.URL != "" {
		u, err := url.Parse(l.URL)
		if err != nil || !mailOrWeb(u) {
			v.status("Not opened (only web and mail addresses open): " + l.URL)
			return
		}
		v.status("Opening " + l.URL)
		openLink(u)
		return
	}
	if l.Page < 1 || l.Page > len(v.d.Pages) {
		return
	}
	tops, _ := v.pageTops()
	y := tops[l.Page-1] + float32(l.DestY)*v.scale - 2*pageGap
	v.scroll.Offset.Y = max(0, min(y, v.column.MinSize().Height-v.scroll.Size().Height))
	v.scroll.Refresh()
	v.showInSight()
	v.markLinks()
}

// status says something on the window's status line, if there is one.
func (v *pageView) status(s string) {
	if v.w != nil {
		v.w.setStatus(s)
	}
}

// markLinks puts the lit link and the ringed one where they are (hidden
// when there is none).
func (v *pageView) markLinks() {
	put := func(r *canvas.Rectangle, i int) {
		if i < 0 || i >= len(v.links) {
			setVisible(r, false)
			return
		}
		pos, size := v.linkRect(v.links[i])
		r.Move(pos.SubtractXY(2, 1))
		r.Resize(size.AddWidthHeight(4, 2))
		setVisible(r, true)
		r.Refresh()
	}
	v.marks.ring.StrokeColor = theme.Color(theme.ColorNameFocus)
	put(v.marks.lit, v.hoverLink)
	ring := v.focusLink
	if !v.area.focused {
		ring = -1
	}
	put(v.marks.ring, ring)
}

// hover lights link i (-1: none) and shows where it goes.
func (v *pageView) hover(i int) {
	if i == v.hoverLink {
		return
	}
	v.hoverLink = i
	v.markLinks()
	v.linkTipShown(i)
}

// linkTipShown shows link i's tip under it (-1: hides the one shown).
func (v *pageView) linkTipShown(i int) {
	if v.w == nil || v.w.tooltips == nil {
		return
	}
	if v.tip != "" && v.w.tooltips.Text() == v.tip {
		v.w.tooltips.HideTip()
	}
	v.tip = ""
	if i < 0 {
		return
	}
	pos, size := v.linkRect(v.links[i])
	v.tip = v.linkTip(v.links[i])
	if v.tip != "" {
		v.w.tooltips.ShowTipUnder(v.tip, v.area, pos.Subtract(v.scroll.Offset), size)
	}
}

// nextLink is the link Tab (or Shift+Tab, back) goes to: after the one
// with the focus, or the first on the page in sight; -1 past the last.
func (v *pageView) nextLink(back bool) int {
	if len(v.links) == 0 {
		return -1
	}
	if v.focusLink >= 0 {
		i := v.focusLink + 1
		if back {
			i = v.focusLink - 1
		}
		if i < 0 || i >= len(v.links) {
			return -1
		}
		return i
	}
	page, _ := v.anchor()
	if back {
		for i := len(v.links) - 1; i >= 0; i-- {
			if v.links[i].page <= page+1 {
				return i
			}
		}
		return -1
	}
	for i, l := range v.links {
		if l.page >= page {
			return i
		}
	}
	return -1
}

// focusOn gives link i the keyboard's ring, in sight.
func (v *pageView) focusOn(i int) {
	v.focusLink = i
	if i >= 0 {
		pos, size := v.linkRect(v.links[i])
		h := v.scroll.Size().Height
		if pos.Y < v.scroll.Offset.Y || pos.Y+size.Height > v.scroll.Offset.Y+h {
			v.scroll.Offset.Y = max(0, min(pos.Y-h/3, v.column.MinSize().Height-h))
			v.scroll.Refresh()
			v.showInSight()
		}
		v.status(v.linkTip(v.links[i]))
	}
	v.markLinks()
}

// AcceptsTab: Tab goes through the links while there is one to go to.
func (a *pageArea) AcceptsTab() bool { return a.v.nextLink(a.shift) >= 0 }

func (a *pageArea) KeyDown(e *fyne.KeyEvent) {
	if e.Name == desktop.KeyShiftLeft || e.Name == desktop.KeyShiftRight {
		a.shift = true
	}
}

func (a *pageArea) KeyUp(e *fyne.KeyEvent) {
	if e.Name == desktop.KeyShiftLeft || e.Name == desktop.KeyShiftRight {
		a.shift = false
	}
}

func (a *pageArea) MouseIn(e *desktop.MouseEvent) { a.MouseMoved(e) }
func (a *pageArea) MouseMoved(e *desktop.MouseEvent) {
	a.v.hover(a.v.linkAt(e.Position))
}
func (a *pageArea) MouseOut() { a.v.hover(-1) }

// Cursor is a hand over a link.
func (a *pageArea) Cursor() desktop.Cursor {
	if a.v.hoverLink >= 0 {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// mailOrWeb reports whether an address opens from the pages: a web
// page's or mail's (not a file, nor a program's).
func mailOrWeb(u *url.URL) bool {
	return u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto"
}

var (
	_ fyne.Tabbable      = (*pageArea)(nil)
	_ desktop.Hoverable  = (*pageArea)(nil)
	_ desktop.Cursorable = (*pageArea)(nil)
	_ desktop.Keyable    = (*pageArea)(nil)
)

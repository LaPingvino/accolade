// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The completion list (Joop, 2026-11-23: "proper character and scene
// header autocomplete"): what Tab completes (complete.go), shown under
// the cursor's line as the ':' line's list is (ex_window.go): ↑↓ choose,
// Tab takes the chosen one (the first if none), Enter takes one chosen
// with the arrows, Esc closes it until the line changes, typing filters
// it, a tap takes a row (a phone's way). It is a layer of the window,
// never focused: the editor keeps the keyboard, and with nothing to
// complete there is no list and no key is taken. While Tab goes through
// the completions the list shows them with the one on the line chosen.
// Tab's order: the list first while it is open, then the source's
// suggestion (suggest.go), then the keyboard flow (keyflow.go).

// complList is the list.
type complList struct {
	w      *MainWindow
	layer  *fyne.Container
	panel  *fyne.Container
	rows   *fyne.Container
	hint   *widget.Label
	bg     *canvas.Rectangle
	c      *completion
	chosen int // -1: none
	cycle  bool
	at     fyne.Position // under the cursor's line, in the window
	above  float32       // the line's top, for a list that does not fit under it
	closed struct {
		rev  uint64
		line int
		on   bool
	}
}

const complRows = 8

// complPop is the window's list, made the first time it is asked for.
func (w *MainWindow) complPop() *complList {
	if w.compl == nil {
		p := &complList{w: w, chosen: -1}
		p.rows = container.NewVBox()
		p.hint = widget.NewLabel("")
		p.hint.SizeName = theme.SizeNameCaptionText
		p.hint.Truncation = fyne.TextTruncateEllipsis
		p.bg = canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
		p.bg.StrokeWidth = 1
		p.bg.CornerRadius = theme.InputRadiusSize()
		p.panel = container.NewStack(p.bg, container.NewBorder(nil, p.hint, nil, nil, p.rows))
		p.layer = container.New(&complLayout{p: p}, p.panel)
		p.layer.Hide()
		w.compl = p
	}
	return w.compl
}

// open reports whether the list is shown.
func (p *complList) open() bool { return p != nil && p.layer.Visible() && p.c != nil }

// completionAt is the completion at the cursor in the script, in its
// language, the source's names and headings first.
func (w *MainWindow) completionAt(cursor int) *completion {
	var names, heads []string
	if w.suggest != nil || w.textEditor != nil {
		names, heads = w.suggester().firsts(cursor)
	}
	return completeIn(w.textEditor.Text(), cursor, completeCtx{starts: w.sceneStarts(), l: w.lex().Language(w.language), names: names, heads: heads})
}

// listKeys reports whether the list may take keys now: a screenplay's
// editor typing (vim's and helix's insert mode, or standard keys).
func (w *MainWindow) listKeys() bool {
	if w.markdown || w.textEditor == nil || !w.scriptInMain() {
		return false
	}
	return w.vim == nil || !w.vim.active() || w.vim.mode == vimInsert
}

// follow shows the list for the cursor, or hides it: after every edit
// and move (showSuggestionHint).
func (p *complList) follow() {
	w := p.w
	e := w.textEditor
	if !w.listKeys() || w.fyneWindow.Canvas().Focused() != e || e.Buffer().HasSelection() || e.Buffer().HasOthers() {
		p.hide()
		return
	}
	b := e.Buffer()
	line, _ := b.Position(b.Cursor())
	if p.closed.on {
		if p.closed.rev == b.Rev() && p.closed.line == line {
			p.hide()
			return
		}
		p.closed.on = false
	}
	if k := w.complete; k != nil && k.active(e) {
		p.show(k.c, k.index, true)
		return
	}
	c := w.completionAt(b.Cursor())
	if c == nil {
		p.hide()
		return
	}
	chosen := -1
	if p.c != nil && !p.cycle && p.chosen >= 0 && p.chosen < len(p.c.labels) { // the one chosen stays chosen as typing filters
		for i, l := range c.labels {
			if l == p.c.labels[p.chosen] {
				chosen = i
			}
		}
	}
	p.show(c, chosen, false)
}

// show lists c, chosen marked.
func (p *complList) show(c *completion, chosen int, cycle bool) {
	p.c, p.chosen, p.cycle = c, chosen, cycle
	e := p.w.textEditor
	d := fyne.CurrentApp().Driver()
	cur := d.AbsolutePositionForObject(e).Add(e.CursorPosition())
	p.at = cur.AddXY(0, e.LineHeight())
	p.above = cur.Y
	first := 0
	if chosen >= complRows {
		first = chosen - complRows + 1
	}
	last := min(first+complRows, len(c.labels))
	objs := make([]fyne.CanvasObject, 0, last-first)
	for k := first; k < last; k++ {
		i := k
		objs = append(objs, newComplRow(c.labels[k], k == chosen, func() { p.take(i) }))
	}
	p.rows.Objects = objs
	hint := fmt.Sprintf("%d to choose from", len(c.labels))
	if chosen >= 0 {
		hint = fmt.Sprintf("%d of %d", chosen+1, len(c.labels))
	}
	if p.w.narrow {
		hint += " · tap one"
	} else if cycle {
		hint += " · Tab: the next · ↑↓ choose · Esc closes"
	} else {
		hint += " · Tab takes · ↑↓ choose · Esc closes"
	}
	p.hint.SetText(hint)
	p.bg.FillColor = theme.Color(theme.ColorNameMenuBackground)
	p.bg.StrokeColor = theme.Color(theme.ColorNameSeparator)
	p.bg.Refresh()
	p.rows.Refresh()
	p.layer.Show()
	p.layer.Refresh()
}

func (p *complList) hide() {
	p.c = nil
	if p.layer.Visible() {
		p.layer.Hide()
	}
}

// close hides the list until the text or the line changes (Esc).
func (p *complList) close() {
	b := p.w.textEditor.Buffer()
	line, _ := b.Position(b.Cursor())
	p.closed.rev, p.closed.line, p.closed.on = b.Rev(), line, true
	p.hide()
}

// move chooses the next (+1) or previous (-1) row; going through the
// completions with Tab, the line takes it at once.
func (p *complList) move(by int) {
	n := len(p.c.labels)
	i := p.chosen + by
	if p.chosen < 0 && by < 0 {
		i = n - 1
	}
	i = (i + n) % n
	if p.cycle {
		p.w.complete.pick(p.w.textEditor, i)
		p.w.showSuggestionHint()
		return
	}
	p.show(p.c, i, false)
}

// take puts row i on the line, one undo step.
func (p *complList) take(i int) {
	e := p.w.textEditor
	if p.cycle {
		p.w.complete.pick(e, i)
	} else { // as Tab takes it: Tab again goes on to the next
		k := p.w.complete
		k.c, k.index = p.c, i
		k.replace(e, p.c.prefix, p.c.candidates[i])
	}
	p.chosen = -1
	p.w.fyneWindow.Canvas().Focus(e)
	p.w.showSuggestionHint()
}

// key takes the list's keys while it is open: ↑↓, Enter for a row
// chosen with them, Esc; false leaves the key to the editor.
func (p *complList) key(k *fyne.KeyEvent) bool {
	if !p.open() || !p.w.listKeys() {
		return false
	}
	switch k.Name {
	case fyne.KeyDown:
		p.move(1)
	case fyne.KeyUp:
		p.move(-1)
	case fyne.KeyEscape:
		p.close()
	case fyne.KeyReturn, fyne.KeyEnter:
		if p.chosen < 0 || p.cycle {
			return false
		}
		p.take(p.chosen)
	default:
		return false
	}
	return true
}

// tab is Tab while the list is open: the chosen row (the first if none),
// or the next while going through them; false if it is not open.
func (p *complList) tab() bool {
	if !p.open() {
		return false
	}
	if p.cycle {
		return false // the completer goes on (editor.go)
	}
	p.take(max(p.chosen, 0))
	return true
}

// complLayout puts the list under the cursor's line, over it if there is
// no room below; on a phone the window's width.
type complLayout struct{ p *complList }

func (l *complLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (l *complLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	p, pad := l.p, theme.Padding()
	width := float32(360)
	for _, o := range p.rows.Objects {
		width = max(width, o.MinSize().Width+4*pad)
	}
	width = min(width, size.Width-2*pad)
	x := min(max(p.at.X, pad), size.Width-width-pad)
	if p.w.narrow {
		width, x = size.Width-2*pad, pad
	}
	h := p.panel.MinSize().Height
	y := p.at.Y
	if y+h > size.Height-pad && p.above-h > pad {
		y = p.above - h
	}
	p.panel.Resize(fyne.NewSize(width, h))
	p.panel.Move(fyne.NewPos(x, y))
}

// complRow is a row of the list: tapped, it is taken.
type complRow struct {
	widget.BaseWidget
	bg    *canvas.Rectangle
	label *widget.Label
	tap   func()
}

func newComplRow(text string, chosen bool, tap func()) *complRow {
	r := &complRow{bg: canvas.NewRectangle(theme.Color(theme.ColorNameSelection)), tap: tap}
	r.label = widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true, Bold: chosen})
	if !chosen {
		r.bg.Hide()
	}
	fyne.SetAccessibleLabel(r, text)
	r.ExtendBaseWidget(r)
	return r
}

func (r *complRow) Tapped(*fyne.PointEvent) { r.tap() }

func (r *complRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(r.bg, r.label))
}

// chosenNow is the row chosen, -1 for none or no list.
func (p *complList) chosenNow() int {
	if !p.open() || p.cycle {
		return -1
	}
	return p.chosen
}

// Package tooltip adds hover tooltips to Fyne buttons, which Fyne 2.8
// lacks.
//
// The tooltip is drawn on a Layer stacked over the window's content. The
// layer holds only a rectangle and text, which take no input, so clicks
// and hovers go through to the widgets underneath; a Fyne pop-up would
// instead swallow the next click anywhere in the window.
package tooltip

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Delay is how long the pointer rests on a button before its tip shows.
var Delay = 500 * time.Millisecond

// Run runs a tip's delayed showing on the UI goroutine: fyne.Do, or the
// app's own hand-over (Accolade's uiDo, which takes the UI's turn).
var Run = func(f func()) { fyne.Do(f) }

// afterFunc runs f, UI code, after d (through Run); a test replaces it
// and fires f itself.
var afterFunc = func(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(d, func() { Run(f) })
}

// Layer is where tooltips are drawn. Stack it over the window content:
//
//	container.NewStack(content, layer)
type Layer struct {
	widget.BaseWidget
	bg   *canvas.Rectangle
	text *canvas.Text

	mu    sync.Mutex // guards the fields below
	shown bool
	tip   string
	pos   fyne.Position
}

// NewLayer creates an empty tooltip layer.
func NewLayer() *Layer {
	l := &Layer{bg: canvas.NewRectangle(color.Transparent), text: canvas.NewText("", color.Transparent)}
	l.bg.Hide()
	l.text.Hide()
	l.ExtendBaseWidget(l)
	return l
}

// CreateRenderer draws the tooltip, if any, where ShowTip put it.
func (l *Layer) CreateRenderer() fyne.WidgetRenderer {
	return &layerRenderer{l: l, objects: []fyne.CanvasObject{l.bg, l.text}}
}

// Text is the tip being shown, or "" when none is.
func (l *Layer) Text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.shown {
		return ""
	}
	return l.tip
}

// TipPosition is where the shown tip's top left corner is.
func (l *Layer) TipPosition() fyne.Position {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.pos
}

// ShowTip shows tip just below obj (a widget inside the layer's window),
// kept within the layer's width.
func (l *Layer) ShowTip(tip string, obj fyne.CanvasObject) {
	l.ShowTipUnder(tip, obj, fyne.NewPos(0, 0), obj.Size())
}

// ShowTipUnder shows tip just below a part of obj: the area at (in obj)
// size big (a link on a page).
func (l *Layer) ShowTipUnder(tip string, obj fyne.CanvasObject, in fyne.Position, area fyne.Size) {
	l.mu.Lock()
	defer l.mu.Unlock()
	d := fyne.CurrentApp().Driver()
	at := d.AbsolutePositionForObject(obj).Subtract(d.AbsolutePositionForObject(l)).Add(in)

	th := l.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	pad := th.Size(theme.SizeNameInnerPadding) / 2
	l.text.Text = tip
	l.text.TextSize = th.Size(theme.SizeNameCaptionText)
	// dark on light themes and light on dark ones, like most desktops' tips
	l.text.Color = th.Color(theme.ColorNameBackground, v)
	l.bg.FillColor = th.Color(theme.ColorNameForeground, v)
	l.bg.CornerRadius = th.Size(theme.SizeNameInputRadius)

	size := l.text.MinSize().Add(fyne.NewSize(2*pad, 2*pad))
	pos := fyne.NewPos(at.X+(area.Width-size.Width)/2, at.Y+area.Height+2)
	pos.X = max(0, min(pos.X, l.Size().Width-size.Width))
	l.bg.Move(pos)
	l.bg.Resize(size)
	l.text.Move(pos.Add(fyne.NewPos(pad, pad)))
	l.text.Resize(l.text.MinSize())
	l.shown, l.tip, l.pos = true, tip, pos
	l.bg.Show()
	l.text.Show()
	l.Refresh()
}

// HideTip hides the tooltip.
func (l *Layer) HideTip() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shown = false
	l.bg.Hide()
	l.text.Hide()
	l.Refresh()
}

type layerRenderer struct {
	l       *Layer
	objects []fyne.CanvasObject
}

func (r *layerRenderer) Layout(fyne.Size)             {}
func (r *layerRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, 0) }
func (r *layerRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *layerRenderer) Destroy()                     {}
func (r *layerRenderer) Refresh() {
	r.l.bg.Refresh()
	r.l.text.Refresh()
}

// Hover shows a widget's tip on a Layer: after Delay once the pointer
// rests on it (In), at once when it gets the keyboard focus (Now), and
// takes it away again (Out). Any widget can keep one.
type Hover struct {
	mu    sync.Mutex // guards timer and gen
	timer *time.Timer
	gen   int // which hover a pending timer belongs to
}

// In shows tip(), as it is then, below obj after Delay, unless Out or
// another In comes first.
func (h *Hover) In(layer *Layer, obj fyne.CanvasObject, tip func() string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopLocked()
	gen := h.gen
	h.timer = afterFunc(Delay, func() {
		h.mu.Lock()
		current := h.gen == gen && h.timer != nil
		h.mu.Unlock()
		if t := tip(); current && t != "" && layer != nil {
			layer.ShowTip(t, obj)
		}
	})
}

// Now shows tip below obj at once (a pending one cancelled).
func (h *Hover) Now(layer *Layer, obj fyne.CanvasObject, tip string) {
	h.mu.Lock()
	h.stopLocked()
	h.mu.Unlock()
	if tip != "" && layer != nil {
		layer.ShowTip(tip, obj)
	}
}

// Out cancels a pending tip and hides tip if it is the one shown.
func (h *Hover) Out(layer *Layer, tip string) {
	h.mu.Lock()
	h.stopLocked()
	h.mu.Unlock()
	if layer != nil && tip != "" && layer.Text() == tip {
		layer.HideTip()
	}
}

// stopLocked cancels a pending tip; h.mu must be held.
func (h *Hover) stopLocked() {
	h.gen++
	if h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
}

// Button is a button that shows a tip when the pointer rests on it.
type Button struct {
	widget.Button
	Tip   string
	layer *Layer
	hover Hover
	// Secondary, if set, is a right-click's, or a press held for
	// LongPress (a finger in a browser comes as the mouse): its menu, at
	// pos
	Secondary func(pos fyne.Position)
	press     *time.Timer
	pressed   bool // the press was a long one: the tap after it is not a tap
}

// LongPress is how long a press is held to be a long one.
var LongPress = 500 * time.Millisecond

// NewButton creates an icon button with a tip shown on layer.
func NewButton(icon fyne.Resource, tip string, tapped func(), layer *Layer) *Button {
	b := &Button{Tip: tip, layer: layer}
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// MouseIn starts the tip's delay.
func (b *Button) MouseIn(e *desktop.MouseEvent) {
	b.Button.MouseIn(e)
	b.hover.In(b.layer, b, func() string { return b.Tip })
}

// MouseOut hides the tip (and a press dragged away is no long one).
func (b *Button) MouseOut() {
	b.stopPress()
	b.Button.MouseOut()
	b.hover.Out(b.layer, b.Tip)
}

// FocusGained shows the tip at once: a keyboard user reaching an icon
// button learns what it does, as a mouse user hovering it does.
func (b *Button) FocusGained() {
	b.Button.FocusGained()
	b.hover.Now(b.layer, b, b.Tip)
}

// FocusLost hides the tip.
func (b *Button) FocusLost() {
	b.Button.FocusLost()
	b.hover.Out(b.layer, b.Tip)
}

// Tapped hides the tip and taps the button (not after a long press:
// its menu showed).
func (b *Button) Tapped(e *fyne.PointEvent) {
	b.hover.Out(b.layer, b.Tip)
	if b.pressed {
		b.pressed = false
		return
	}
	b.Button.Tapped(e)
}

// TappedSecondary is a right-click: Secondary, if set.
func (b *Button) TappedSecondary(e *fyne.PointEvent) {
	if b.Secondary != nil {
		b.hover.Out(b.layer, b.Tip)
		b.Secondary(e.AbsolutePosition)
	}
}

// MouseDown starts a long press, if the button has a Secondary.
func (b *Button) MouseDown(e *desktop.MouseEvent) {
	b.pressed = false
	b.stopPress()
	if b.Secondary == nil || e.Button != desktop.MouseButtonPrimary || b.Disabled() {
		return
	}
	pos := e.AbsolutePosition
	b.press = afterFunc(LongPress, func() {
		if b.press == nil {
			return
		}
		b.press, b.pressed = nil, true
		b.hover.Out(b.layer, b.Tip)
		b.Secondary(pos)
	})
}

// MouseUp ends a press: short, it is a tap.
func (b *Button) MouseUp(*desktop.MouseEvent) { b.stopPress() }

func (b *Button) stopPress() {
	if b.press != nil {
		b.press.Stop()
		b.press = nil
	}
}

// AccessibilityLabel is the tip: the button shows only an icon, and a
// screen reader says what it does.
func (b *Button) AccessibilityLabel() string {
	if b.Text != "" {
		return b.Text
	}
	return b.Tip
}

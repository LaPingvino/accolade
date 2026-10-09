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
	l.mu.Lock()
	defer l.mu.Unlock()
	d := fyne.CurrentApp().Driver()
	at := d.AbsolutePositionForObject(obj).Subtract(d.AbsolutePositionForObject(l))

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
	pos := fyne.NewPos(at.X+(obj.Size().Width-size.Width)/2, at.Y+obj.Size().Height+2)
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

// Button is a button that shows a tip when the pointer rests on it.
type Button struct {
	widget.Button
	Tip   string
	layer *Layer

	mu    sync.Mutex // guards timer and gen
	timer *time.Timer
	gen   int // which hover a pending timer belongs to
}

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
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopLocked()
	gen := b.gen
	b.timer = time.AfterFunc(Delay, func() {
		fyne.Do(func() {
			b.mu.Lock()
			current := b.gen == gen && b.timer != nil
			b.mu.Unlock()
			if current && b.Tip != "" && b.layer != nil {
				b.layer.ShowTip(b.Tip, b)
			}
		})
	})
}

// MouseOut hides the tip.
func (b *Button) MouseOut() {
	b.Button.MouseOut()
	b.hide()
}

// FocusGained shows the tip at once: a keyboard user reaching an icon
// button learns what it does, as a mouse user hovering it does.
func (b *Button) FocusGained() {
	b.Button.FocusGained()
	b.mu.Lock()
	b.stopLocked()
	b.mu.Unlock()
	if b.Tip != "" && b.layer != nil {
		b.layer.ShowTip(b.Tip, b)
	}
}

// FocusLost hides the tip.
func (b *Button) FocusLost() {
	b.Button.FocusLost()
	b.hide()
}

// Tapped hides the tip and taps the button.
func (b *Button) Tapped(e *fyne.PointEvent) {
	b.hide()
	b.Button.Tapped(e)
}

// stopLocked cancels a pending tip; b.mu must be held.
func (b *Button) stopLocked() {
	b.gen++
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
}

func (b *Button) hide() {
	b.mu.Lock()
	b.stopLocked()
	b.mu.Unlock()
	if b.layer != nil && b.layer.Text() == b.Tip {
		b.layer.HideTip()
	}
}

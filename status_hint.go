package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The status bar's "Tab: …" hint (a completion, a suggestion from the
// source) starts with a small Tab chip: a tap on it does what Tab does,
// for a phone, whose keyboard has none. It is shown only with a hint.

// hintPrefix starts a status message that says what Tab would do.
const hintPrefix = "Tab: "

// tabChip is the chip: "Tab" on a tinted, rounded ground.
type tabChip struct {
	widget.BaseWidget
	onTap func()
	label *widget.Label
	bg    *canvas.Rectangle
}

func newTabChip(onTap func()) *tabChip {
	c := &tabChip{onTap: onTap}
	c.label = widget.NewLabel("Tab ⇥")
	c.label.SizeName = theme.SizeNameCaptionText
	c.label.TextStyle.Bold = true
	c.bg = canvas.NewRectangle(tint(theme.ColorNamePrimary, 0x40))
	c.bg.CornerRadius = theme.InputRadiusSize()
	c.ExtendBaseWidget(c)
	fyne.SetAccessibleLabel(c, "Take the suggestion")
	return c
}

func (c *tabChip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, c.label))
}

// Tapped takes the hint's suggestion, as Tab.
func (c *tabChip) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}

func (c *tabChip) Cursor() desktop.Cursor { return desktop.PointerCursor }

// setStatus shows a message; a "Tab: …" hint with the chip before it,
// its own "Tab: " left out.
func (w *MainWindow) setStatus(text string) {
	if w.statusLabel == nil {
		return
	}
	if rest, ok := strings.CutPrefix(text, hintPrefix); ok && w.tabChip != nil {
		w.statusHint = text
		w.tabChip.Show()
		w.statusLabel.SetText(rest)
		return
	}
	w.statusHint = ""
	if w.tabChip != nil {
		w.tabChip.Hide()
	}
	w.statusLabel.SetText(text)
}

// statusText is what the status bar says, a hint with its "Tab: ".
func (w *MainWindow) statusText() string {
	if w.statusHint != "" {
		return w.statusHint
	}
	if w.statusLabel == nil {
		return ""
	}
	return w.statusLabel.Text
}

// takeHint is a tap on the chip: Tab, as the editor has it (the
// suggestion taken or the next one, a completion), the editor focused.
func (w *MainWindow) takeHint() {
	e := w.textEditor
	if w.statusHint == "" || e == nil || e.OnTab == nil {
		return
	}
	e.OnTab()
	w.fyneWindow.Canvas().Focus(e)
}

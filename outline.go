package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// The outline (View > Outline) lists the script's sections and scenes,
// with their synopses, beside the editor; clicking one goes there.

// outlineItem is a section or scene of the script.
type outlineItem struct {
	Line     int    // where it is
	Text     string // the heading, without markers
	Synopsis string // Fountain's "= ..." lines under it
	Level    int    // a section's level (# 1, ## 2); scenes are one deeper
	Scene    bool
}

// outlineOf the script's text.
func outlineOf(text string, starts []string) []outlineItem {
	var items []outlineItem
	section := 0
	lines := strings.Split(text, "\n")
	inBoneyard := false // notes and other /* ... */ are not the script
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		prevBlank := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		switch {
		case strings.HasPrefix(t, "#"):
			level := len(t) - len(strings.TrimLeft(t, "#"))
			section = level
			items = append(items, outlineItem{Line: i, Text: strings.TrimSpace(t[level:]), Level: level})
		case prevBlank && (isSceneHeadingIn(t, starts) || (strings.HasPrefix(t, ".") && !strings.HasPrefix(t, ".."))):
			heading := strings.TrimPrefix(t, ".")
			if j := strings.Index(heading, " #"); j > 0 && strings.HasSuffix(heading, "#") {
				heading = strings.TrimSpace(heading[:j]) // without its number
			}
			items = append(items, outlineItem{Line: i, Text: strings.ToUpper(heading), Level: section + 1, Scene: true})
		case strings.HasPrefix(t, "=") && !strings.HasPrefix(t, "==") && len(items) > 0:
			it := &items[len(items)-1]
			it.Synopsis = strings.TrimSpace(it.Synopsis + " " + strings.TrimSpace(t[1:]))
		}
	}
	return items
}

// outlinePanel is the outline beside the editor.
type outlinePanel struct {
	w     *MainWindow
	items []outlineItem
	list  *widget.List
	box   fyne.CanvasObject

	oneLine, twoLines float32 // row heights without and with a synopsis
}

func newOutlinePanel(w *MainWindow) *outlinePanel {
	o := &outlinePanel{w: w}
	o.list = widget.NewList(
		func() int { return len(o.items) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Truncation = fyne.TextTruncateEllipsis
			syn := widget.NewLabel("")
			syn.SizeName = theme.SizeNameCaptionText
			syn.Importance = widget.LowImportance
			syn.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, syn)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(o.items) {
				return
			}
			it := o.items[id]
			box := obj.(*fyne.Container)
			title, syn := box.Objects[0].(*widget.Label), box.Objects[1].(*widget.Label)
			title.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Text)
			title.TextStyle = fyne.TextStyle{Bold: !it.Scene}
			title.Refresh()
			syn.SetText(strings.Repeat("  ", max(0, it.Level-1)) + it.Synopsis)
			if it.Synopsis == "" {
				syn.Hide()
			} else {
				syn.Show()
			}
		},
	)
	o.list.OnSelected = func(id widget.ListItemID) {
		if id < len(o.items) {
			o.goTo(o.items[id].Line)
		}
		o.list.UnselectAll()
	}
	width := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	width.SetMinSize(fyne.NewSize(240, 0))
	heading := widget.NewLabelWithStyle("Outline", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	fyne.SetAccessibleLabel(o.list, "Outline")
	o.box = container.NewStack(width, container.NewBorder(heading, nil, nil, nil, o.list))
	return o
}

// update reads the outline from the script again.
func (o *outlinePanel) update(text string) {
	o.items = outlineOf(text, o.w.sceneStarts())
	// rows without a synopsis are one line high
	if o.oneLine == 0 {
		o.oneLine = widget.NewLabel("X").MinSize().Height
		o.twoLines = o.list.CreateItem().MinSize().Height
	}
	for id, it := range o.items {
		if it.Synopsis == "" {
			o.list.SetItemHeight(id, o.oneLine)
		} else {
			o.list.SetItemHeight(id, o.twoLines)
		}
	}
	o.list.Refresh()
}

// goTo puts the cursor at the start of a line and the focus in the editor.
func (o *outlinePanel) goTo(line int) {
	e := o.w.textEditor
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	o.w.fyneWindow.Canvas().Focus(e)
}

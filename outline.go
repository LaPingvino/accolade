package main

import (
	"image/color"
	"slices"
	"strconv"
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

// filterOutline is the items whose heading or synopsis has query in it
// (in any case), each with the sections it is in, for where it is.
func filterOutline(items []outlineItem, query string) []outlineItem {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return items
	}
	var out []outlineItem
	var sections []int // the open sections, outermost first (indexes)
	shown := map[int]bool{}
	for i, it := range items {
		if !it.Scene {
			for len(sections) > 0 && items[sections[len(sections)-1]].Level >= it.Level {
				sections = sections[:len(sections)-1]
			}
		}
		if strings.Contains(strings.ToLower(it.Text+"\n"+it.Synopsis), query) {
			for _, s := range sections {
				if !shown[s] {
					shown[s] = true
					out = append(out, items[s])
				}
			}
			shown[i] = true
			out = append(out, it)
		}
		if !it.Scene {
			sections = append(sections, i)
		}
	}
	return out
}

// foldKey names a section across edits (its line moves).
func foldKey(it outlineItem) string { return strconv.Itoa(it.Level) + " " + it.Text }

// hasInside reports whether the i-th item is a section with something in it.
func hasInside(items []outlineItem, i int) bool {
	return !items[i].Scene && i+1 < len(items) && (items[i+1].Scene || items[i+1].Level > items[i].Level)
}

// unfolded is the items without those inside a folded section.
func unfolded(items []outlineItem, folded map[string]bool) []outlineItem {
	var out []outlineItem
	hiding := 0 // inside a folded section of this level (0: none)
	for _, it := range items {
		if hiding > 0 && (it.Scene || it.Level > hiding) {
			continue
		}
		hiding = 0
		out = append(out, it)
		if !it.Scene && folded[foldKey(it)] {
			hiding = it.Level
		}
	}
	return out
}

// outlinePanel is the outline beside the editor.
type outlinePanel struct {
	w      *MainWindow
	all    []outlineItem // the script's
	items  []outlineItem // shown: those the filter lets through
	filter *widget.Entry
	folded map[string]bool // sections shown without what is in them
	list   *widget.List
	box    fyne.CanvasObject

	oneLine, twoLines float32 // row heights without and with a synopsis
}

func newOutlinePanel(w *MainWindow) *outlinePanel {
	o := &outlinePanel{w: w, folded: map[string]bool{}}
	o.list = widget.NewList(
		func() int { return len(o.items) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.Truncation = fyne.TextTruncateEllipsis
			syn := widget.NewLabel("")
			syn.SizeName = theme.SizeNameCaptionText
			syn.Importance = widget.LowImportance
			syn.Truncation = fyne.TextTruncateEllipsis
			fold := widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), nil)
			fold.Importance = widget.LowImportance
			return container.NewBorder(nil, nil, fold, nil, container.NewVBox(title, syn))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(o.items) {
				return
			}
			it := o.items[id]
			row := obj.(*fyne.Container)
			box, fold := row.Objects[0].(*fyne.Container), row.Objects[1].(*widget.Button)
			title, syn := box.Objects[0].(*widget.Label), box.Objects[1].(*widget.Label)
			o.showFold(fold, id)
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
	o.filter = widget.NewEntry()
	o.filter.SetPlaceHolder("Filter scenes and sections")
	fyne.SetAccessibleLabel(o.filter, "Filter the outline")
	o.filter.OnChanged = func(string) { o.show() }
	o.filter.OnSubmitted = func(string) { // Enter: to the first one shown
		if len(o.items) > 0 {
			o.goTo(o.items[0].Line)
		}
	}
	o.box = container.NewStack(width, container.NewBorder(container.NewVBox(heading, o.filter), nil, nil, nil, o.list))
	return o
}

// update reads the outline from the script again.
func (o *outlinePanel) update(text string) {
	o.all = outlineOf(text, o.w.sceneStarts())
	o.show()
}

// show shows the items the filter lets through.
func (o *outlinePanel) show() {
	if o.filter.Text != "" {
		o.items = filterOutline(o.all, o.filter.Text)
	} else {
		o.items = unfolded(o.all, o.folded)
	}
	// rows without a synopsis are one line high
	if o.oneLine == 0 {
		row := o.list.CreateItem().(*fyne.Container)
		o.twoLines = row.MinSize().Height
		row.Objects[0].(*fyne.Container).Objects[1].Hide()
		o.oneLine = row.MinSize().Height
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

// showFold makes the item's fold button: shown for a section with
// something in it (not while filtering, which shows all it finds).
func (o *outlinePanel) showFold(b *widget.Button, id int) {
	it := o.items[id]
	i := slices.IndexFunc(o.all, func(a outlineItem) bool { return a.Line == it.Line })
	if o.filter.Text != "" || i < 0 || !hasInside(o.all, i) {
		b.Hide()
		return
	}
	b.Show()
	key := foldKey(it)
	if o.folded[key] {
		b.SetIcon(theme.MenuExpandIcon())
		fyne.SetAccessibleLabel(b, "Unfold "+it.Text)
	} else {
		b.SetIcon(theme.MenuDropDownIcon())
		fyne.SetAccessibleLabel(b, "Fold "+it.Text)
	}
	b.OnTapped = func() {
		o.folded[key] = !o.folded[key]
		o.show()
	}
}

// goTo puts the cursor at the start of a line and the focus in the editor.
func (o *outlinePanel) goTo(line int) {
	e := o.w.textEditor
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(line), false) })
	o.w.fyneWindow.Canvas().Focus(e)
}

package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/cite"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The References pane (View > References, a Markdown document's): what
// the document cites, in the order it first does, each with its entry
// from the library (author, year, title), then what the front matter's
// nocite: lists without citing; a key the library lacks says so. A tap
// goes to the first citation of it. Above, where the library is.

// refItem is one row: a key, its entry (nil: not in the library), the
// line it is first cited on (-1: nocite).
type refItem struct {
	key   string
	entry *cite.Entry
	line  int
}

type referencesPanel struct {
	w     *MainWindow
	items []refItem
	list  *widget.List
	note  *widget.Label
	box   fyne.CanvasObject
	timer *uiTimer
}

func (w *MainWindow) referencesPane() *referencesPanel {
	if w.references == nil {
		w.references = newReferencesPanel(w)
		w.citations().changed = append(w.citations().changed, func() {
			if w.paneShown("references") {
				w.references.refresh()
			}
		})
	}
	return w.references
}

func newReferencesPanel(w *MainWindow) *referencesPanel {
	p := &referencesPanel{w: w}
	p.note = widget.NewLabel("")
	p.note.Wrapping = fyne.TextWrapWord
	p.note.Importance = widget.LowImportance
	p.note.SizeName = theme.SizeNameCaptionText
	p.list = widget.NewList(
		func() int { return len(p.items) },
		func() fyne.CanvasObject {
			key := widget.NewLabel("")
			key.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
			key.SizeName = theme.SizeNameCaptionText
			key.Truncation = fyne.TextTruncateEllipsis
			what := widget.NewLabel("")
			what.Wrapping = fyne.TextWrapWord
			return container.NewVBox(key, what)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id >= len(p.items) {
				return
			}
			it := p.items[id]
			box := o.(*fyne.Container)
			key, what := box.Objects[0].(*widget.Label), box.Objects[1].(*widget.Label)
			k := "@" + it.key
			if it.line < 0 {
				k += " (nocite)"
			}
			key.SetText(k)
			what.Importance = widget.MediumImportance
			switch {
			case it.key == "*":
				what.SetText("Every entry of the library.")
			case it.entry != nil:
				what.SetText(it.entry.Line())
			default:
				what.Importance = widget.WarningImportance
				what.SetText("Not in the library.")
			}
			p.list.SetItemHeight(id, box.MinSize().Height)
		},
	)
	p.list.OnSelected = func(id widget.ListItemID) {
		if id < len(p.items) && p.items[id].line >= 0 {
			w.goToScript(p.items[id].line) // middle.go
		}
		p.list.UnselectAll()
	}
	width := canvas.NewRectangle(color.Transparent) // as wide as the outline and notes
	width.SetMinSize(fyne.NewSize(240, 0))
	p.box = container.NewStack(width, container.NewBorder(p.note, nil, nil, nil, p.list))
	return p
}

// refItems are the rows for text, its keys looked up in lib.
func refItems(text string, lib *cite.Library) []refItem {
	cited, nocite := citedKeys(text)
	first := map[string]int{}
	cite.EachCitation(text, func(i int, c syntax.Citation) {
		for _, k := range c.Keys {
			if _, ok := first[k.Key]; !ok {
				first[k.Key] = i
			}
		}
	})
	var out []refItem
	for _, k := range cited {
		out = append(out, refItem{key: k, entry: lib.Get(k), line: first[k]})
	}
	for _, k := range nocite {
		out = append(out, refItem{key: k, entry: lib.Get(k), line: -1})
	}
	return out
}

// refresh lists what the document cites now.
func (p *referencesPanel) refresh() {
	lib := p.w.library()
	p.items = refItems(p.w.textEditor.Text(), lib)
	note := p.w.libraryNote()
	if len(p.items) == 0 {
		note = "Nothing cited yet: [@key] or @key, @ lists the library's entries. " + note
	}
	p.note.SetText(note)
	p.list.Refresh()
}

// edited follows the text once typing pauses.
func (p *referencesPanel) edited() {
	p.w.debounce(&p.timer, 300*time.Millisecond, p.refresh)
}

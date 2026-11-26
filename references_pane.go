package main

import (
	"image/color"
	"strings"
	"time"
	"unicode/utf8"

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
	style *widget.Select // the document's citation style: its front matter's csl:
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
	p.style = widget.NewSelect(citeStyleNames(""), func(name string) {
		if name == "" || citeStyleValue(name) == cite.FrontOf(w.textEditor.Text()).CSL {
			return
		}
		if e, ok := cslEdit(w.textEditor.Text(), citeStyleValue(name)); ok {
			cur := w.textEditor.CursorOffset() // where it was, the edit before it counted
			if cur >= e.End {
				cur += utf8.RuneCountInString(e.Text) - (e.End - e.Start)
			}
			w.applyTextEdit(e, max(cur, 0))
		}
	})
	p.style.PlaceHolder = "Citation style"
	p.box = container.NewStack(width, container.NewBorder(container.NewVBox(p.style, p.note), nil, nil, nil, p.list))
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
	csl := cite.FrontOf(p.w.textEditor.Text()).CSL
	p.style.Options = citeStyleNames(csl)
	p.style.SetSelected(citeStyleLabel(csl))
}

// citeStyles are the styles the References pane offers, by their
// labels: Accolade's own (no csl:) and the CSL ones it has.
var citeStyles = []struct{ label, csl string }{
	{"Author-date (Accolade's)", ""},
	{"APA", "apa"},
	{"Chicago author-date", "chicago-author-date"},
	{"IEEE (numbered)", "ieee"},
	{"Vancouver (numbered)", "vancouver"},
}

// citeStyleLabel is the label of a csl: value (a file: its name).
func citeStyleLabel(csl string) string {
	for _, s := range citeStyles {
		if s.csl == csl {
			return s.label
		}
	}
	return csl
}

// citeStyleValue is the csl: value of a label (a file's name as is).
func citeStyleValue(label string) string {
	for _, s := range citeStyles {
		if s.label == label {
			return s.csl
		}
	}
	return label
}

// citeStyleNames are the labels offered, csl (a file of the document's
// own) among them.
func citeStyleNames(csl string) []string {
	var out []string
	for _, s := range citeStyles {
		out = append(out, s.label)
	}
	if citeStyleLabel(csl) == csl && csl != "" {
		out = append(out, csl)
	}
	return out
}

// cslEdit sets text's front matter csl: to value ("": the line taken
// out), the front matter made when there is none; false: nothing to do.
func cslEdit(text, value string) (textEdit, bool) {
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	pos := 0 // in runes, the line's start
	end := -1
	for i := 0; i < len(kinds) && kinds[i] == syntax.MdFrontMatter; i++ {
		n := utf8.RuneCountInString(lines[i])
		if k, _, ok := strings.Cut(lines[i], ":"); ok && i > 0 && strings.EqualFold(strings.TrimSpace(k), "csl") {
			if value == "" {
				return textEdit{Start: pos, End: pos + n + 1}, true
			}
			return textEdit{Start: pos, End: pos + n, Text: "csl: " + value}, true
		}
		if i > 0 && (strings.TrimSpace(lines[i]) == "---" || strings.TrimSpace(lines[i]) == "...") {
			end = pos
		}
		pos += n + 1
	}
	switch {
	case value == "":
		return textEdit{}, false
	case end >= 0: // before the front matter's closing line
		return textEdit{Start: end, End: end, Text: "csl: " + value + "\n"}, true
	}
	return textEdit{Text: "---\ncsl: " + value + "\n---\n\n"}, true
}

// edited follows the text once typing pauses.
func (p *referencesPanel) edited() {
	p.w.debounce(&p.timer, 300*time.Millisecond, p.refresh)
}

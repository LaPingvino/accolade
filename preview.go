package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/layout"
	"github.com/LaPingvino/lexington/rules"
)

// The preview shows the screenplay as Lexington prints it (its layout
// package: the columns, styles and page breaks of the PDF), in a monospace
// font, one paragraph per printed line.

// previewSegments are the rich text segments of a screenplay's text.
func previewSegments(text string, sceneHeaders []string, elements rules.Set) []widget.RichTextSegment {
	lines := layout.Lay(fountain.Parse(sceneHeaders, strings.NewReader(text)), elements)
	var segs []widget.RichTextSegment
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if l.PageBreak {
			segs = append(segs, &widget.SeparatorSegment{})
			continue
		}
		if l.Column != 0 { // dual dialogue: both columns side by side
			j := i
			for j < len(lines) && lines[j].Column != 0 {
				j++
			}
			segs = append(segs, dualSegments(lines[i:j])...)
			i = j - 1
			continue
		}
		segs = append(segs, lineSegments(l.Padded()[:len(l.Padded())-len(l.Text())], l.Spans)...)
	}
	return segs
}

// lineSegments is a printed line: its padding, then its spans; the last
// segment ends the paragraph.
func lineSegments(pad string, spans []layout.Span) []widget.RichTextSegment {
	if len(spans) == 0 {
		spans = []layout.Span{{}}
	}
	var segs []widget.RichTextSegment
	if pad != "" { // unstyled: an underline does not run under the indent
		segs = append(segs, &widget.TextSegment{Text: pad, Style: widget.RichTextStyle{
			Inline: true, SizeName: theme.SizeNameCaptionText, TextStyle: fyne.TextStyle{Monospace: true},
		}})
	}
	for i, s := range spans {
		t := s.Text
		if t == "" && pad == "" {
			t = " " // an empty line keeps its height
		}
		segs = append(segs, &widget.TextSegment{Text: t, Style: widget.RichTextStyle{
			Inline:    i < len(spans)-1,
			SizeName:  theme.SizeNameCaptionText, // a whole 60-character line fits the pane
			TextStyle: fyne.TextStyle{Monospace: true, Bold: s.Bold, Italic: s.Italic, Underline: s.Underline},
		}})
	}
	return segs
}

// dualSegments puts dual dialogue's left and right column next to each
// other, line by line.
func dualSegments(lines []layout.Line) []widget.RichTextSegment {
	var left, right []layout.Line
	for _, l := range lines {
		if l.Column == 1 {
			left = append(left, l)
		} else {
			right = append(right, l)
		}
	}
	var segs []widget.RichTextSegment
	for k := 0; k < max(len(left), len(right)); k++ {
		var spans []layout.Span
		pad := ""
		used := 0
		if k < len(left) {
			p := left[k].Padded()
			pad = p[:len(p)-len(left[k].Text())]
			spans = append(spans, left[k].Spans...)
			used = len([]rune(p))
		}
		if k < len(right) {
			p := right[k].Padded()
			gap := strings.Repeat(" ", max(1, len([]rune(p))-len([]rune(right[k].Text()))-used))
			if len(spans) == 0 {
				pad = gap
			} else {
				spans = append(spans, layout.Span{Text: gap})
			}
			spans = append(spans, right[k].Spans...)
		}
		segs = append(segs, lineSegments(pad, spans)...)
	}
	return segs
}

// previewPane shows the preview's printed lines in a list, which makes
// only the rows in view: a RichText of a whole script (thousands of
// segments, all laid out and drawn) took seconds to show and froze the
// window on a long script.
type previewPane struct {
	Segments []widget.RichTextSegment // all of them, in order
	rows     [][]widget.RichTextSegment
	list     *widget.List
	content  fyne.CanvasObject
}

func newPreviewPane() *previewPane {
	p := &previewPane{}
	p.list = widget.NewList(
		func() int { return len(p.rows) },
		func() fyne.CanvasObject {
			rt := widget.NewRichText()
			rt.Wrapping = fyne.TextWrapOff // lines are wrapped to the screenplay's columns already
			return rt
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			rt := o.(*widget.RichText)
			if id < len(p.rows) {
				rt.Segments = p.rows[id]
				rt.Refresh()
			}
		})
	p.list.HideSeparators = true
	fyne.SetAccessibleLabel(p.list, "Preview")
	p.list.OnSelected = func(id widget.ListItemID) { p.list.Unselect(id) } // a page, not a choice
	p.content = container.NewThemeOverride(p.list, previewTheme{})
	return p
}

// SetSegments shows segs: one row per printed line (a paragraph's last
// segment ends it), a page break a row of its own.
func (p *previewPane) SetSegments(segs []widget.RichTextSegment) {
	p.Segments = segs
	p.rows = p.rows[:0]
	var row []widget.RichTextSegment
	for _, s := range segs {
		row = append(row, s)
		if t, ok := s.(*widget.TextSegment); ok && t.Style.Inline {
			continue
		}
		p.rows = append(p.rows, row)
		row = nil
	}
	if len(row) > 0 {
		p.rows = append(p.rows, row)
	}
	p.list.Refresh()
}

// previewTheme packs the rows as closely as the lines of a page.
type previewTheme struct{}

func (previewTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return fyne.CurrentApp().Settings().Theme().Color(n, v)
}
func (previewTheme) Font(s fyne.TextStyle) fyne.Resource {
	if f := fyne.CurrentApp().Settings().Theme().Font(s); f != nil {
		return f
	}
	return theme.DefaultTheme().Font(s) // a theme without bold monospace
}
func (previewTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fyne.CurrentApp().Settings().Theme().Icon(n)
}
func (previewTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNamePadding, theme.SizeNameInnerPadding, theme.SizeNameLineSpacing:
		return 0
	}
	return fyne.CurrentApp().Settings().Theme().Size(n)
}

package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/layout"
)

// The preview shows the screenplay as Lexington prints it (its layout
// package: the columns, styles and page breaks of the PDF), in a monospace
// font, one paragraph per printed line.

// previewSegments are the rich text segments of a screenplay's text.
func previewSegments(text string, sceneHeaders []string) []widget.RichTextSegment {
	lines := layout.Lay(fountain.Parse(sceneHeaders, strings.NewReader(text)), nil)
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
	for i, s := range spans {
		t := s.Text
		if i == 0 {
			t = pad + t
		}
		if t == "" {
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

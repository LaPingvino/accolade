package main

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/images"
)

// What a passage's row (and the Markdown preview) shows of Markdown.
//
// Fyne shows a Markdown table in a horizontal scroller of its own. A
// finger dragged over it (the browser makes a drag a wheel scroll, a
// vertical one turned sideways when only the width overflows) scrolls the
// table, not the list, and a long table stops the pane from scrolling on
// a phone. So no row holds a scroller: a table is shown a row of it to a
// line, each cell after its column's name, and a long passage is cut to a
// few screens' worth, with Show All under it.

// rowBudget is about how much of a passage a row shows before Show All:
// characters, a line break costing lineCost of them.
const (
	rowBudget = 900
	lineCost  = 40
)

// tableRowsShort is how many of a table's rows a row shows before Show All.
const tableRowsShort = 3

// markdownSegments is md as rich text, a table as lines (all its rows
// with maxRows < 0); hidden reports what was left out for maxRows.
// Pictures are read by pics (nil: none, each a placeholder), never
// fetched.
func markdownSegments(md string, maxRows int, pics *images.Loader) (segs []widget.RichTextSegment, hidden bool) {
	segs = widget.NewRichTextFromMarkdown(md).Segments
	if imgs := markdownImages(md); len(imgs) > 0 || hasImageSegment(segs) {
		segs = withPictures(segs, imgs, pics)
	}
	return flatTables(segs, maxRows)
}

// hasImageSegment tells whether Fyne made a picture of something.
func hasImageSegment(segs []widget.RichTextSegment) bool {
	for _, s := range segs {
		switch s := s.(type) {
		case *widget.ImageSegment:
			return true
		case *widget.ParagraphSegment:
			if hasImageSegment(s.Texts) {
				return true
			}
		case *widget.ListSegment:
			if hasImageSegment(s.Items) {
				return true
			}
		}
	}
	return false
}

// rowSegments is what passage md's row shows: all of it, or (short) cut to
// rowBudget; cut reports whether there is more.
func rowSegments(md string, short bool) (segs []widget.RichTextSegment, cut bool) {
	if !short {
		segs, _ = markdownSegments(md, -1, nil)
		return segs, false
	}
	segs, cut = markdownSegments(md, tableRowsShort, nil)
	budget := rowBudget
	segs, more := capSegments(segs, &budget)
	return segs, cut || more
}

// flatTables replaces each table in segs by a line saying its size and a
// paragraph per row: "Column: cell · Column: cell"; at most maxRows rows
// (all with maxRows < 0).
func flatTables(segs []widget.RichTextSegment, maxRows int) ([]widget.RichTextSegment, bool) {
	var out []widget.RichTextSegment
	hidden := false
	for _, seg := range segs {
		switch s := seg.(type) {
		case *widget.TableSegment:
			out = append(out, tableLines(s, maxRows)...)
			if maxRows >= 0 && len(s.Rows) > maxRows {
				hidden = true
			}
		case *widget.ParagraphSegment:
			texts, h := flatTables(s.Texts, maxRows)
			out = append(out, &widget.ParagraphSegment{Texts: texts})
			hidden = hidden || h
		case *widget.ListSegment:
			items, h := flatTables(s.Items, maxRows)
			l := *s
			l.Items = items
			out = append(out, &l)
			hidden = hidden || h
		default:
			out = append(out, seg)
		}
	}
	return out, hidden
}

// tableLines is a table as lines.
func tableLines(t *widget.TableSegment, maxRows int) []widget.RichTextSegment {
	cols := len(t.Headers)
	for _, r := range t.Rows {
		cols = max(cols, len(r))
	}
	summary := fmt.Sprintf("Table, %s × %s", plural(cols, "column"), plural(len(t.Rows), "row"))
	if maxRows >= 0 && len(t.Rows) > maxRows {
		summary += fmt.Sprintf(" (the first %d here; Edit shows it all as written)", maxRows)
	}
	note := widget.RichTextStyleParagraph
	note.TextStyle.Italic = true
	note.ColorName = theme.ColorNamePlaceHolder
	note.SizeName = theme.SizeNameCaptionText
	out := []widget.RichTextSegment{&widget.TextSegment{Style: note, Text: summary}}
	names := make([]string, len(t.Headers))
	for i, h := range t.Headers {
		var b strings.Builder
		for _, s := range h {
			b.WriteString(s.Textual())
		}
		names[i] = strings.TrimSpace(b.String())
	}
	for n, r := range t.Rows {
		if maxRows >= 0 && n >= maxRows {
			break
		}
		var texts []widget.RichTextSegment
		for c, cell := range r {
			if c > 0 {
				texts = append(texts, &widget.TextSegment{Style: widget.RichTextStyleInline, Text: "  ·  "})
			}
			if c < len(names) && names[c] != "" {
				name := widget.RichTextStyleStrong
				texts = append(texts, &widget.TextSegment{Style: name, Text: names[c] + ": "})
			}
			for _, s := range cell {
				if ts, ok := s.(*widget.TextSegment); ok {
					inline := *ts
					inline.Style.Inline = true // a cell's text runs on in the line
					s = &inline
				}
				texts = append(texts, s)
			}
		}
		// the row ends its line: its last text is a paragraph's
		var last *widget.TextSegment
		if len(texts) > 0 {
			last, _ = texts[len(texts)-1].(*widget.TextSegment)
		}
		if last != nil {
			last.Style.Inline = false
		} else {
			texts = append(texts, &widget.TextSegment{Style: widget.RichTextStyleParagraph})
		}
		out = append(out, texts...)
	}
	return out
}

// capSegments is segs cut when budget runs out; cut reports whether it did.
func capSegments(segs []widget.RichTextSegment, budget *int) (out []widget.RichTextSegment, cut bool) {
	for _, seg := range segs {
		if *budget <= 0 {
			return out, true
		}
		switch s := seg.(type) {
		case *widget.TextSegment:
			text, more := cutText(s.Text, budget)
			c := *s
			c.Text = text
			out = append(out, &c)
			if more {
				return out, true
			}
		case *widget.CodeBlockSegment:
			text, more := cutText(s.Text, budget)
			out = append(out, &widget.CodeBlockSegment{Text: text})
			if more {
				return out, true
			}
		case *widget.ParagraphSegment:
			texts, more := capSegments(s.Texts, budget)
			out = append(out, &widget.ParagraphSegment{Texts: texts})
			if more {
				return out, true
			}
		case *widget.ListSegment:
			items, more := capSegments(s.Items, budget)
			l := *s
			l.Items = items
			out = append(out, &l)
			if more {
				return out, true
			}
		default:
			*budget -= lineCost
			out = append(out, seg)
		}
	}
	return out, false
}

// cutText is text cut (at a space, with an ellipsis) where budget runs out.
func cutText(text string, budget *int) (string, bool) {
	used := 0
	for i, r := range text {
		cost := 1
		if r == '\n' {
			cost = lineCost
		}
		if used+cost > *budget {
			head := text[:i]
			if sp := strings.LastIndexAny(head, " \n"); sp > len(head)/2 {
				head = head[:sp]
			}
			*budget = 0
			return strings.TrimRight(head, " \n") + " …", true
		}
		used += cost
	}
	*budget -= used
	return text, false
}

// shortRunes is how many characters of a passage a short row shows, for
// estimating its height.
func shortRunes(p passage) int {
	return min(utf8.RuneCountInString(p.Plain), rowBudget)
}

// moreButton is a row's Show All / Show Less.
func newMoreButton() *widget.Button {
	b := widget.NewButton("Show All", nil)
	b.Importance = widget.LowImportance
	b.Alignment = widget.ButtonAlignLeading
	return b
}

package main

import (
	"fmt"
	"regexp"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Helix's several selections (helix keys):
//
//	s       select what a regex matches in the selections
//	S       split the selections where a regex matches
//	Alt-s   split the selections into lines
//	C       a selection like the last one on the next line
//	,       keep the primary selection only;  Alt-,  drop it
//	d c y i a   act at every selection (y: the primary's text)
//
// Typing, Backspace and Delete then go to all of them; moving keeps the
// primary only.

// multiKey handles a helix key about several selections; it reports
// whether it did.
func (v *vim) multiKey(r rune) bool {
	b := v.e.Buffer()
	switch r {
	case 's', 'S':
		if v.prompt == nil {
			return true
		}
		label := "select:"
		if r == 'S' {
			label = "split:"
		}
		v.prompt(label, func(pattern string) {
			re, err := regexp.Compile(pattern)
			if err != nil {
				v.status(err.Error())
				return
			}
			v.setSpans(regexSpans(v.e.Text(), b.Spans(), re, r == 'S'))
		})
		return true
	case 'C':
		v.setSpans(append(b.Spans(), copyDown([]rune(v.e.Text()), b.Spans())...))
		return true
	case ',':
		v.setSpans(b.Spans()[:1])
		return true
	}
	if !b.HasOthers() {
		return false
	}
	switch r {
	case 'd', 'c':
		s := b.Spans()[0]
		v.reg, v.regLine = string([]rune(v.e.Text())[s.Start():s.End()]), false
		v.e.Edit(func(b *buffer.Buffer) {
			b.EditSpans(func(s buffer.Span) (int, int, string) { return s.Start(), s.End(), "" })
		})
		if r == 'c' {
			v.mode = vimInsert
			v.status(vimInsert.String())
		}
		return true
	case 'i', 'a':
		var carets []buffer.Span
		for _, s := range b.Spans() {
			at := s.Start()
			if r == 'a' {
				at = s.End()
			}
			carets = append(carets, buffer.Span{Anchor: at, Cursor: at})
		}
		v.e.Navigate(func(b *buffer.Buffer) { b.SetSpans(carets) })
		v.mode = vimInsert
		v.status(fmt.Sprintf("%s (%d cursors)", vimInsert, len(carets)))
		return true
	case 'y':
		s := b.Spans()[0]
		v.reg, v.regLine = string([]rune(v.e.Text())[s.Start():s.End()]), false
		return true
	}
	return false
}

// multiShortcut handles Alt-s and Alt-, in helix keys.
func (v *vim) multiShortcut(s fyne.Shortcut) bool {
	c, ok := s.(*desktop.CustomShortcut)
	if !ok || c.Modifier != fyne.KeyModifierAlt {
		return false
	}
	b := v.e.Buffer()
	switch c.KeyName {
	case fyne.KeyS:
		v.setSpans(lineSpans([]rune(v.e.Text()), b.Spans()))
	case fyne.KeyComma:
		if spans := b.Spans(); len(spans) > 1 {
			v.setSpans(spans[1:])
		}
	default:
		return false
	}
	return true
}

// setSpans selects spans and says how many there are.
func (v *vim) setSpans(spans []buffer.Span) {
	if len(spans) == 0 {
		v.status("nothing selected")
		return
	}
	v.e.Navigate(func(b *buffer.Buffer) { b.SetSpans(spans) })
	n := len(v.e.Buffer().Spans())
	if n == 1 {
		v.status("-- HELIX NORMAL --")
	} else {
		v.status(fmt.Sprintf("-- HELIX NORMAL -- %d selections", n))
	}
}

// regexSpans are re's matches in the spans (the whole text where a span
// selects nothing), or with split the parts between them.
func regexSpans(text string, spans []buffer.Span, re *regexp.Regexp, split bool) []buffer.Span {
	t := []rune(text)
	var out []buffer.Span
	for _, s := range spans {
		from, to := s.Start(), s.End()
		if from == to {
			from, to = 0, len(t)
		}
		part := string(t[from:to])
		at := func(byteOff int) int { return from + len([]rune(part[:byteOff])) }
		prev := 0
		for _, m := range re.FindAllStringIndex(part, -1) {
			if split {
				if m[0] > prev {
					out = append(out, buffer.Span{Anchor: at(prev), Cursor: at(m[0])})
				}
				prev = m[1]
			} else if m[1] > m[0] {
				out = append(out, buffer.Span{Anchor: at(m[0]), Cursor: at(m[1])})
			}
		}
		if split && prev < len(part) {
			out = append(out, buffer.Span{Anchor: at(prev), Cursor: at(len(part))})
		}
	}
	return out
}

// lineSpans are the spans' lines, each a span (without its line break).
func lineSpans(t []rune, spans []buffer.Span) []buffer.Span {
	var out []buffer.Span
	for _, s := range spans {
		start := s.Start()
		for i := s.Start(); i <= s.End() && i <= len(t); i++ {
			if i == len(t) || i == s.End() || t[i] == '\n' {
				if i > start {
					out = append(out, buffer.Span{Anchor: start, Cursor: i})
				}
				start = i + 1
			}
		}
	}
	return out
}

// copyDown is the last span again on the next line, at the same columns.
func copyDown(t []rune, spans []buffer.Span) []buffer.Span {
	last := spans[0]
	for _, s := range spans {
		if s.Start() > last.Start() {
			last = s
		}
	}
	col := func(off int) (lineStart, c int) {
		ls, _ := lineBounds(t, off)
		return ls, off - ls
	}
	_, ca := col(last.Anchor)
	ls, cc := col(last.Cursor)
	_, le := lineBounds(t, ls)
	if le >= len(t) {
		return nil // no next line
	}
	next := le + 1
	_, ne := lineBounds(t, next)
	clampTo := func(c int) int { return next + min(c, ne-next) }
	return []buffer.Span{{Anchor: clampTo(ca), Cursor: clampTo(cc)}}
}

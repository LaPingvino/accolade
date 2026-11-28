// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// A Markdown document's Format menu (commands.go, Markdown only): each
// command writes plain Markdown on the selection or the cursor's lines,
// takes it away again where that makes sense (a heading of the same
// level, a list or quote all its lines have, a code block's fences), and
// is one undo step. Bold and italic are the screenplay's (** and *).

// mdLines are the lines the selection touches (the cursor's line without
// one): the first and the last, as buffer lines.
func mdLines(b *buffer.Buffer) (first, last int) {
	start, end := b.Selection()
	first, _ = b.Position(start)
	last, _ = b.Position(end)
	if end > start && last > first && end == b.LineStart(last) {
		last-- // a selection ending at a line's start leaves that line
	}
	return first, last
}

// mdReplaceLines puts lines in place of buffer lines first..last, keeping
// the cursor on its line (at the line's end when it was there) or the
// lines selected.
func mdReplaceLines(b *buffer.Buffer, first, last int, lines []string) {
	start, end := b.Selection()
	cursorLine, col := b.Position(b.Cursor())
	atEnd := b.Cursor() == b.LineEnd(cursorLine)
	b.Group(func() {
		b.Replace(b.LineStart(first), b.LineEnd(last), strings.Join(lines, "\n"))
		if start != end {
			b.Select(b.LineStart(first), b.LineEnd(first+len(lines)-1))
			return
		}
		line := cursorLine
		if line > first+len(lines)-1 {
			line = first + len(lines) - 1
		}
		off := b.LineStart(line) + col
		if atEnd || off > b.LineEnd(line) {
			off = b.LineEnd(line)
		}
		b.SetCursor(off, false)
	})
}

var (
	mdHeadingRe = regexp.MustCompile(`^(#{1,6})[ \t]+`)
	// a line's block mark: a quote, a task, a bullet or a number
	mdQuoteRe  = regexp.MustCompile(`^>[ \t]?`)
	mdTaskRe   = regexp.MustCompile(`^[-*+][ \t]+\[[ xX]\][ \t]+`)
	mdBulletRe = regexp.MustCompile(`^[-*+][ \t]+`)
	mdNumberRe = regexp.MustCompile(`^\d+[.)][ \t]+`)
)

// mdHeadingLevel is a line's heading level (0: none) and its text.
func mdHeadingLevel(line string) (int, string) {
	if m := mdHeadingRe.FindStringSubmatch(line); m != nil {
		return len(m[1]), line[len(m[0]):]
	}
	if strings.TrimSpace(line) != "" && strings.Trim(line, "#") == "" && len(line) <= 6 {
		return len(line), "" // "##" alone
	}
	return 0, line
}

// setMarkdownHeading makes the lines headings of level (0: normal text);
// lines all of that level already go back to normal text.
func setMarkdownHeading(b *buffer.Buffer, level int) {
	first, last := mdLines(b)
	all := level > 0
	var texts []string
	for i := first; i <= last; i++ {
		l, text := mdHeadingLevel(b.Line(i))
		all = all && l == level
		texts = append(texts, text)
	}
	out := make([]string, len(texts))
	for i, text := range texts {
		out[i] = text
		if level > 0 && !all {
			out[i] = strings.Repeat("#", level) + " " + text
		}
	}
	mdReplaceLines(b, first, last, out)
}

// mdBlock is a line mark a list or quote command puts on lines.
type mdBlock int

const (
	mdBullets mdBlock = iota
	mdNumbers
	mdTasks
	mdQuote
)

// mdStripList is a line without its list mark (a task's, a bullet's or
// a number's), and whether it had kind's.
func mdStripList(line string, kind mdBlock) (string, bool) {
	for _, x := range []struct {
		re   *regexp.Regexp
		kind mdBlock
	}{{mdTaskRe, mdTasks}, {mdBulletRe, mdBullets}, {mdNumberRe, mdNumbers}} {
		if m := x.re.FindString(line); m != "" {
			return line[len(m):], x.kind == kind
		}
	}
	return line, false
}

// toggleMarkdownBlock puts kind's mark on the lines (a list's replacing
// another list's; numbers counting 1, 2, 3); lines all with it lose it.
// Blank lines are left as they are.
func toggleMarkdownBlock(b *buffer.Buffer, kind mdBlock) {
	first, last := mdLines(b)
	var lines []string
	all, any := true, false
	for i := first; i <= last; i++ {
		line := b.Line(i)
		lines = append(lines, line)
		if strings.TrimSpace(line) == "" && first != last {
			continue
		}
		any = true
		has := false
		if kind == mdQuote {
			has = mdQuoteRe.MatchString(line)
		} else {
			_, has = mdStripList(line, kind)
		}
		all = all && has
	}
	all = all && any
	n := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "" && first != last {
			continue
		}
		if kind == mdQuote {
			if all {
				lines[i] = line[len(mdQuoteRe.FindString(line)):]
			} else {
				lines[i] = "> " + line
			}
			continue
		}
		text, _ := mdStripList(line, kind)
		if all {
			lines[i] = text
			continue
		}
		n++
		switch kind {
		case mdBullets:
			lines[i] = "- " + text
		case mdTasks:
			lines[i] = "- [ ] " + text
		default:
			lines[i] = strconv.Itoa(n) + ". " + text
		}
	}
	mdReplaceLines(b, first, last, lines)
}

// toggleCodeBlock fences the lines in ``` (an empty line: a block to
// write in), or takes the fences away when the lines are fenced already.
func toggleCodeBlock(b *buffer.Buffer) {
	first, last := mdLines(b)
	fence := func(i int) bool {
		return i >= 0 && i < b.LineCount() && strings.HasPrefix(strings.TrimSpace(b.Line(i)), "```")
	}
	b.Group(func() {
		switch {
		case fence(first) && fence(last) && last > first: // the fences selected with it
			b.Replace(b.LineEnd(last-1), b.LineEnd(last), "")
			b.Replace(b.LineStart(first), b.LineStart(first+1), "")
			b.Select(b.LineStart(first), b.LineEnd(last-2))
		case fence(first-1) && fence(last+1):
			b.Replace(b.LineEnd(last), b.LineEnd(last+1), "")
			b.Replace(b.LineStart(first-1), b.LineStart(first), "")
			b.Select(b.LineStart(first-1), b.LineEnd(last-1))
		case first == last && strings.TrimSpace(b.Line(first)) == "":
			start := b.LineStart(first)
			b.Replace(start, b.LineEnd(first), "```\n\n```")
			b.SetCursor(start+4, false)
		default:
			start, end := b.LineStart(first), b.LineEnd(last)
			body := b.Slice(start, end)
			b.Replace(start, end, "```\n"+body+"\n```")
			b.Select(start+4, start+4+len([]rune(body)))
		}
	})
}

// insertBlockLine puts block (a rule, a table) on lines of its own after
// the cursor's paragraph line, with a blank line around it; the cursor
// goes to sel's offset in it (-1: after it).
func insertBlockLine(b *buffer.Buffer, block string, sel int) {
	line, _ := b.Position(b.Cursor())
	b.Group(func() {
		at := b.LineEnd(line)
		prefix, suffix := "\n\n", "\n"
		if strings.TrimSpace(b.Line(line)) == "" {
			at, prefix = b.LineStart(line), ""
			if line > 0 && strings.TrimSpace(b.Line(line-1)) != "" {
				prefix = "\n"
			}
			if at == b.LineEnd(line) && line+1 < b.LineCount() {
				suffix = "\n"
			} else if line+1 >= b.LineCount() {
				suffix = ""
			}
		}
		b.Replace(at, at, prefix+block+suffix)
		start := at + len([]rune(prefix))
		if sel < 0 {
			b.SetCursor(start+len([]rune(block)), false)
		} else {
			b.SetCursor(start+sel, false)
		}
	})
}

// insertRule is a horizontal rule (---) after the cursor's line.
func insertRule(b *buffer.Buffer) { insertBlockLine(b, "---", -1) }

// tableSkeleton is a table of rows (under the header) and cols.
func tableSkeleton(rows, cols int) string {
	var head, line, row []string
	for c := 1; c <= cols; c++ {
		head = append(head, fmt.Sprintf("Column %d", c))
		line = append(line, "---")
		row = append(row, "   ")
	}
	out := []string{"| " + strings.Join(head, " | ") + " |", "| " + strings.Join(line, " | ") + " |"}
	for r := 0; r < rows; r++ {
		out = append(out, "|"+strings.Join(row, "|")+"|")
	}
	return strings.Join(out, "\n")
}

// insertTable is a table skeleton after the cursor's line, the cursor
// in its first header cell.
func insertTable(b *buffer.Buffer, rows, cols int) { insertBlockLine(b, tableSkeleton(rows, cols), 2) }

// mdLinkRe is a link or image, whole.
var mdLinkRe = regexp.MustCompile(`^(!?)\[([^\]]*)\]\(([^)]*)\)$`)

// insertMarkdownLink makes the selection a link to url (an image's alt text if
// image), or, a link selected already, its text again; without a
// selection the text is the url's, selected to write over.
func insertMarkdownLink(b *buffer.Buffer, url string, image bool) {
	b.Group(func() {
		start, end := b.Selection()
		sel := b.Slice(start, end)
		if m := mdLinkRe.FindStringSubmatch(sel); m != nil && (m[1] == "!") == image {
			b.Replace(start, end, m[2])
			b.Select(start, start+len([]rune(m[2])))
			return
		}
		text := sel
		if text == "" {
			text = "text"
			if image {
				text = "description"
			}
		}
		bang := ""
		if image {
			bang = "!"
		}
		b.Replace(start, end, bang+"["+text+"]("+url+")")
		at := start + len([]rune(bang)) + 1
		b.Select(at, at+len([]rune(text)))
	})
}

var mdFootnoteRe = regexp.MustCompile(`\[\^(\d+)\]`)

// insertFootnote puts the next footnote's mark at the cursor ([^1]) and
// its note at the document's end, the cursor there to write it.
func insertFootnote(b *buffer.Buffer) {
	n := 1
	for _, m := range mdFootnoteRe.FindAllStringSubmatch(b.Text(), -1) {
		if k, _ := strconv.Atoi(m[1]); k >= n {
			n = k + 1
		}
	}
	mark := "[^" + strconv.Itoa(n) + "]"
	b.Group(func() {
		_, end := b.Selection()
		b.Replace(end, end, mark)
		text := b.Text()
		trimmed := strings.TrimRight(text, "\n")
		want := 2 // a blank line before the notes
		last := trimmed[strings.LastIndex(trimmed, "\n")+1:]
		switch {
		case trimmed == "":
			want = 0
		case strings.HasPrefix(last, "[^") && strings.Contains(last, "]:"):
			want = 1 // under the notes before it
		}
		add := max(0, want-(len(text)-len(trimmed)))
		b.Replace(b.Len(), b.Len(), strings.Repeat("\n", add)+mark+": ")
		b.SetCursor(b.Len(), false)
	})
}

// The commands' actions on the window's editor.
func (w *MainWindow) mdEdit(f func(b *buffer.Buffer)) {
	w.writeEditor().Edit(f) // the main editor's: the source or notes in the middle
}

// askLink asks for a link's (an image's) address, then writes it.
func (w *MainWindow) askLink(image bool) {
	url := widget.NewEntry()
	url.SetPlaceHolder("https://")
	title, field := "Link", "Address"
	if image {
		title, field = "Image", "Image address or file"
	}
	fyne.SetAccessibleLabel(url, field)
	focus := w.fyneWindow.Canvas().Focused()
	items := []*widget.FormItem{widget.NewFormItem(field, url)}
	if image { // a file of one's own: kept with the document (pictures.go)
		choose := widget.NewButtonWithIcon("Choose a Picture…", theme.FolderOpenIcon(), func() {
			chooseFileIn(w.fyneWindow, pictureExtensions, func(picked string) {
				w.keepPicture(picked, w.pictureFolder(""), func(rel string, err error) {
					if err != nil {
						dialog.ShowError(err, w.fyneWindow)
						return
					}
					url.SetText(rel)
				})
			})
		})
		note := hint("Copied into the bundle's assets/, or a folder beside the document; an address on the web is linked, never fetched.")
		items = append(items, widget.NewFormItem("", container.NewVBox(choose, note)))
	}
	d := dialog.NewForm(title, "Insert", "Cancel", items, func(ok bool) {
		if ok && strings.TrimSpace(url.Text) != "" {
			w.mdEdit(func(b *buffer.Buffer) { insertMarkdownLink(b, strings.TrimSpace(url.Text), image) })
		}
		if focus != nil {
			w.fyneWindow.Canvas().Focus(focus)
		} else {
			w.focusMain()
		}
	}, w.fyneWindow)
	if image {
		d.Resize(fyne.NewSize(460, 260))
	} else {
		d.Resize(fyne.NewSize(420, 160))
	}
	d.Show()
	w.fyneWindow.Canvas().Focus(url)
}

// askTable asks for a table's rows and columns, then writes it.
func (w *MainWindow) askTable() {
	rows, cols := widget.NewEntry(), widget.NewEntry()
	rows.SetText("3")
	cols.SetText("3")
	fyne.SetAccessibleLabel(rows, "Rows")
	fyne.SetAccessibleLabel(cols, "Columns")
	d := dialog.NewForm("Table", "Insert", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Rows", rows), widget.NewFormItem("Columns", cols),
	}, func(ok bool) {
		r, err1 := strconv.Atoi(strings.TrimSpace(rows.Text))
		c, err2 := strconv.Atoi(strings.TrimSpace(cols.Text))
		if ok && err1 == nil && err2 == nil && r >= 0 && c >= 1 && r <= 100 && c <= 20 {
			w.mdEdit(func(b *buffer.Buffer) { insertTable(b, r, c) })
		}
		w.focusMain()
	}, w.fyneWindow)
	d.Show()
	w.fyneWindow.Canvas().Focus(rows)
}

// pickHeading is the Markdown row's heading list, as a menu by the button.
func (w *MainWindow) pickHeading(anchor fyne.CanvasObject) {
	var items []*fyne.MenuItem
	env := cmdEnv{markdown: true, browser: inBrowser}
	for _, id := range mdHeadingIDs() {
		c := cmd(id)
		item := fyne.NewMenuItem(c.labelIn(env), w.tapped(c, nil))
		if len(c.keys) > 0 {
			item.Shortcut = c.keys[0]
		}
		items = append(items, item)
	}
	showMenuBy(w, anchor, fyne.NewMenu("", items...))
}

// mdHeadingIDs are the heading commands, Normal Text first.
func mdHeadingIDs() []string {
	ids := []string{"md.heading0"}
	for i := 1; i <= 6; i++ {
		ids = append(ids, "md.heading"+strconv.Itoa(i))
	}
	return ids
}

// showMenuBy pops a menu up under anchor (the window's middle without one).
func showMenuBy(w *MainWindow, anchor fyne.CanvasObject, m *fyne.Menu) {
	c := w.fyneWindow.Canvas()
	pos := fyne.NewPos(c.Size().Width/3, c.Size().Height/4)
	if anchor != nil {
		if ac := fyne.CurrentApp().Driver().CanvasForObject(anchor); ac != nil {
			c = ac
			pos = fyne.CurrentApp().Driver().AbsolutePositionForObject(anchor).AddXY(0, anchor.Size().Height)
		}
	}
	widget.ShowPopUpMenuAtPosition(m, c, pos)
}

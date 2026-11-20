package main

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// wideStory is a source with what an HTML import brings: a table of 12
// columns and 80 rows, code with a long line, a long URL, between
// paragraphs.
func wideStory() string {
	var b strings.Builder
	b.WriteString("# The ledger\n\nAnna opened the ledger.\n\n")
	cols := 12
	for c := 0; c < cols; c++ {
		fmt.Fprintf(&b, "| Column %d ", c+1)
	}
	b.WriteString("|\n")
	for c := 0; c < cols; c++ {
		b.WriteString("|---")
	}
	b.WriteString("|\n")
	for r := 0; r < 80; r++ {
		for c := 0; c < cols; c++ {
			fmt.Fprintf(&b, "| cell %d.%d ", r+1, c+1)
		}
		b.WriteString("|\n")
	}
	b.WriteString("\nShe closed it again.\n\n")
	b.WriteString("```\n" + strings.Repeat("x", 300) + "\n```\n\n")
	b.WriteString("See https://example.com/" + strings.Repeat("a", 300) + " for more.\n\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&b, "Paragraph %d after the table, so the list has somewhere to scroll to.\n\n", i+1)
	}
	return b.String()
}

// A wide, long table (an HTML import's) stopped the source pane from
// scrolling on a phone: Fyne put it in a scroller of its own, which took
// the finger's drag. No row holds a scroller now, none is wider than the
// pane, a long one is cut with Show All, and a drag over it scrolls the
// list.
func TestSourceWideTable(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, wideStory())
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	s := w.sourcePane()
	s.follow()
	table := -1
	for i, p := range s.passages {
		if strings.HasPrefix(p.Markdown, "|") {
			table = i
		}
	}
	if table < 0 {
		t.Fatal("no table passage")
	}
	width := s.list.Size().Width
	if width <= 0 || width > 380 {
		t.Fatalf("list width %v", width)
	}
	check := func(id int) *fyne.Container {
		t.Helper()
		row := newPassageRow()
		s.show(row, id)
		row.Resize(fyne.NewSize(width, row.MinSize().Height))
		if mw := row.MinSize().Width; mw > width {
			t.Errorf("row %d (%.20q) is %v wide, the pane %v", id, s.passages[id].Markdown, mw, width)
		}
		walk(row, func(o fyne.CanvasObject) {
			if _, ok := o.(fyne.Scrollable); ok {
				t.Errorf("row %d (%.20q) has a scroller %T", id, s.passages[id].Markdown, o)
			}
		})
		return row
	}
	for id := range s.passages { // the table, the code, the long URL, the rest
		check(id)
	}
	// the table: short, with Show All; in full, every row, still no scroller
	row := check(table)
	r := rowParts(row)
	if !r.more.Visible() || r.more.Text != "Show All" || !strings.Contains(r.text.String(), "Table, 12 columns × 80 rows") {
		t.Errorf("short table: more %v %q, text %.80q", r.more.Visible(), r.more.Text, r.text.String())
	}
	short := row.MinSize().Height
	s.toggleFull(table)
	row = check(table)
	r = rowParts(row)
	if r.more.Text != "Show Less" || !strings.Contains(r.text.String(), "cell 80.12") || row.MinSize().Height <= short*3 {
		t.Errorf("full table: %q, %v high (short %v)", r.more.Text, row.MinSize().Height, short)
	}
	if s.heights[table] != row.MinSize().Height {
		t.Errorf("the list keeps %v for the full row, it is %v", s.heights[table], row.MinSize().Height)
	}
	s.toggleFull(table)

	// a drag over the table's row scrolls the list: the browser makes it a
	// wheel scroll at the finger, a phone's driver a drag
	y := float32(0)
	for i := range table {
		y += s.heights[i] + theme.Padding()
	}
	s.list.ScrollToOffset(y)
	over := absPos(s.list).Add(fyne.NewPos(width/2, 30))
	off := s.list.GetScrollOffset()
	test.Scroll(w.fyneWindow.Canvas(), over, 0, -100)
	if got := s.list.GetScrollOffset(); got <= off {
		t.Errorf("a scroll over the table left the list at %v (was %v)", got, off)
	}
	off = s.list.GetScrollOffset()
	test.Drag(w.fyneWindow.Canvas(), over, 0, -100)
	if _, mobile := any(container.NewVScroll(widget.NewLabel(""))).(fyne.Draggable); mobile && s.list.GetScrollOffset() <= off {
		t.Errorf("a drag over the table left the list at %v (was %v)", s.list.GetScrollOffset(), off)
	}
}

// The Markdown preview shows a table as lines too, in its one scroller.
func TestPreviewTableNoScroller(t *testing.T) {
	test.NewTempApp(t)
	p := newPreviewPanel()
	p.update(wideStory())
	w := test.NewTempWindow(t, p.box)
	w.Resize(fyne.NewSize(380, 600))
	n := 0
	walk(p.box, func(o fyne.CanvasObject) {
		if _, ok := o.(fyne.Scrollable); ok {
			n++
		}
	})
	if n != 1 || !strings.Contains(p.text.String(), "Column 12: cell 80.12") {
		t.Errorf("%d scrollers; text %.60q", n, p.text.String())
	}
	if mw := p.text.MinSize().Width; mw > 380 {
		t.Errorf("preview text %v wide", mw)
	}
}

// On a phone's width the panes go under the script, which stays in
// sight, and the source's button stays on the toolbar.
func TestNarrowPanes(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	s := w.sourcePane()
	if !w.narrow || w.paneSplit == nil || w.paneSplit.Horizontal {
		t.Fatalf("narrow %v, split %v", w.narrow, w.paneSplit)
	}
	ed, pane := w.editorView.Size(), s.box.Size()
	if ed.Height < 200 || ed.Width < 340 || pane.Width > 380 || pane.Height < 200 {
		t.Errorf("script %v, pane %v", ed, pane)
	}
	if !w.headerBar.button("view.source").Visible() {
		t.Error("the source button went into the menu")
	}
	// a finger cannot drag the split in the browser: a button makes the
	// panes taller, and half again
	half := s.box.Size().Height
	w.paneTaller.OnTapped()
	if s.box.Size().Height <= half+100 || w.editorView.Size().Height < 50 {
		t.Errorf("taller: pane %v (was %v), script %v", s.box.Size().Height, half, w.editorView.Size())
	}
	w.paneTaller.OnTapped()
	if w.paneSplit.Offset != paneHalf {
		t.Errorf("half again: %v", w.paneSplit.Offset)
	}
	// turning wide and narrow again keeps the editor's focus
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.fyneWindow.Resize(fyne.NewSize(1100, 800))
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	if w.fyneWindow.Canvas().Focused() != w.textEditor || !w.narrow {
		t.Errorf("after wide and narrow: focus %T, narrow %v", w.fyneWindow.Canvas().Focused(), w.narrow)
	}
	w.headerBar.button("view.source").OnTapped() // closed: the script alone
	if s.shown() || w.editorView.Size().Height < 600 {
		t.Errorf("closed: shown %v, script %v", s.shown(), w.editorView.Size())
	}
	w.headerBar.button("view.source").OnTapped()
	if !s.shown() {
		t.Error("the button did not open the source again")
	}
	w.fyneWindow.Resize(fyne.NewSize(1100, 650)) // wide again: beside the script
	if w.narrow || s.box.Size().Width < sourceWidth-10 || w.editorView.Size().Height < 400 {
		t.Errorf("wide: narrow %v, pane %v, script %v", w.narrow, s.box.Size(), w.editorView.Size())
	}
}

// walk visits o and everything under it, widgets' renderers too.
func walk(o fyne.CanvasObject, f func(fyne.CanvasObject)) {
	f(o)
	switch c := o.(type) {
	case *fyne.Container:
		for _, k := range c.Objects {
			walk(k, f)
		}
	case fyne.Widget:
		for _, k := range test.WidgetRenderer(c).Objects() {
			walk(k, f)
		}
	}
}

func absPos(o fyne.CanvasObject) fyne.Position {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
}

// Package editor is Accolade's own script editor: a text model (buffer)
// wrapped into rows (wrap) and drawn in a Fyne TextGrid, so selection,
// highlights and single-step undo are under Accolade's control. See
// docs/EDITOR_WIDGET.md; this is step 3 (display and keyboard).
package editor

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// ScriptEditor edits a screenplay in a monospace grid.
type ScriptEditor struct {
	widget.BaseWidget

	// OnChanged is called with the text after every edit.
	OnChanged func(text string)
	// OnEnter, when set, handles Enter instead of inserting a line break
	// (Accolade formats the completed line); it reports whether it did.
	OnEnter func() bool
	// OnCursorChanged is called when the cursor or selection moves.
	OnCursorChanged func()

	buf     *buffer.Buffer
	layout  wrap.Layout
	grid    *widget.TextGrid
	columns int
	focused bool
	shift   bool
	// preferred column for moving up and down
	goalCol int
	// dragging: selecting from dragFrom
	dragging bool
	dragFrom int

	// highlights are ranges shown with a highlight background (search
	// matches); [start, end) rune offsets
	highlights [][2]int
	// Syntax colours the text by Fountain element
	Syntax bool
	// lineNumbers shows each line's number in a gutter (SetLineNumbers)
	lineNumbers bool

	// onCursorMoved is told where the cursor is (see NewScroll)
	onCursorMoved func(cursor fyne.Position, height float32)
}

// New creates an editor holding text.
func New(text string) *ScriptEditor {
	e := &ScriptEditor{buf: buffer.New(text), grid: widget.NewTextGrid(), columns: 80, goalCol: -1}
	e.grid.Scroll = fyne.ScrollNone
	e.ExtendBaseWidget(e)
	e.relayout()
	return e
}

// Buffer gives access to the text model (selection, Replace, Group, undo).
func (e *ScriptEditor) Buffer() *buffer.Buffer { return e.buf }

// Text is the edited text.
func (e *ScriptEditor) Text() string { return e.buf.Text() }

// SetText replaces the text and forgets the undo history; OnChanged is
// called, as Fyne's Entry does.
func (e *ScriptEditor) SetText(text string) {
	e.buf.SetText(text)
	e.changed(true)
}

// SetLineNumbers shows or hides line numbers in a gutter on the left.
func (e *ScriptEditor) SetLineNumbers(on bool) {
	e.lineNumbers = on
	e.relayout()
}

// gutter is the number of columns the line numbers take (0 when hidden):
// the digits of the highest line number and a space.
func (e *ScriptEditor) gutter() int {
	if !e.lineNumbers {
		return 0
	}
	return len(strconv.Itoa(e.buf.LineCount())) + 1
}

// SetHighlights marks ranges of the text, such as all search matches, with
// a highlight background; nil clears them.
func (e *ScriptEditor) SetHighlights(ranges [][2]int) {
	e.highlights = ranges
	e.relayout()
}

// Edit changes the text through the buffer and reports the change: use
// Buffer().Group inside for one undo step.
func (e *ScriptEditor) Edit(f func(b *buffer.Buffer)) {
	f(e.buf)
	e.changed(true)
}

// Navigate changes the cursor or selection through the buffer.
func (e *ScriptEditor) Navigate(f func(b *buffer.Buffer)) {
	f(e.buf)
	e.changed(false)
}

// Select selects [start, end) (rune offsets), cursor at end.
func (e *ScriptEditor) Select(start, end int) {
	e.Navigate(func(b *buffer.Buffer) { b.Select(start, end) })
}

// Replace replaces [start, end) with text as one undo step.
func (e *ScriptEditor) Replace(start, end int, text string) {
	e.Edit(func(b *buffer.Buffer) { b.Replace(start, end, text) })
}

// GridRow is what the editor shows in a visual row (for tests and
// inspection).
func (e *ScriptEditor) GridRow(row int) widget.TextGridRow {
	if row < 0 || row >= len(e.grid.Rows) {
		return widget.TextGridRow{}
	}
	return e.grid.Rows[row]
}

// SelectedText is the selected text.
func (e *ScriptEditor) SelectedText() string { return e.buf.SelectedText() }

// CursorOffset is the cursor's rune offset.
func (e *ScriptEditor) CursorOffset() int { return e.buf.Cursor() }

// SetCursorOffset moves the cursor, dropping the selection.
func (e *ScriptEditor) SetCursorOffset(off int) {
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(off, false) })
}

// Undo undoes the last edit step.
func (e *ScriptEditor) Undo() {
	if e.buf.Undo() {
		e.changed(true)
	}
}

// Redo redoes the last undone step.
func (e *ScriptEditor) Redo() {
	if e.buf.Redo() {
		e.changed(true)
	}
}

// Columns is the number of columns the text is wrapped to.
func (e *ScriptEditor) Columns() int { return e.columns }

// CreateRenderer draws the grid.
func (e *ScriptEditor) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(e.grid)
}

// Resize re-wraps the text to the new width.
func (e *ScriptEditor) Resize(size fyne.Size) {
	e.BaseWidget.Resize(size)
	if cols := int(size.Width / e.cellWidth()); cols > 0 && cols != e.columns {
		e.columns = cols
		e.relayout()
	}
}

// MinSize is a few columns wide and as tall as the wrapped text.
func (e *ScriptEditor) MinSize() fyne.Size {
	return fyne.NewSize(e.cellWidth()*10, e.cellHeight()*float32(len(e.layout.Rows)))
}

func (e *ScriptEditor) cellSize() fyne.Size {
	size := fyne.MeasureText("M", e.Theme().Size(theme.SizeNameText), fyne.TextStyle{Monospace: true})
	return fyne.NewSize(float32(math.Round(float64(size.Width))), float32(math.Round(float64(size.Height))))
}

func (e *ScriptEditor) cellWidth() float32  { return max(e.cellSize().Width, 1) }
func (e *ScriptEditor) cellHeight() float32 { return max(e.cellSize().Height, 1) }

// relayout re-wraps the text and redraws the grid.
func (e *ScriptEditor) relayout() {
	text := []rune(e.buf.Text())
	g := e.gutter()
	e.layout = wrap.Wrap(text, max(e.columns-g, 1))

	th := e.Theme()
	v := theme.VariantLight
	if app := fyne.CurrentApp(); app != nil {
		v = app.Settings().ThemeVariant()
	}
	selected := &widget.CustomTextGridStyle{BGColor: th.Color(theme.ColorNameSelection, v)}
	cursor := &widget.CustomTextGridStyle{
		FGColor: th.Color(theme.ColorNameBackground, v),
		BGColor: th.Color(theme.ColorNameForeground, v),
	}
	highlight := &widget.CustomTextGridStyle{BGColor: th.Color(theme.ColorNameHover, v)}
	if c := th.Color(theme.ColorNameWarning, v); c != nil {
		r, g, b, _ := c.RGBA()
		highlight.BGColor = color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0x60}
	}
	selStart, selEnd := e.buf.Selection()
	curRow, curCol := e.layout.RowCol(e.buf.Cursor())
	highlighted := func(off int) bool {
		for _, h := range e.highlights {
			if off >= h[0] && off < h[1] {
				return true
			}
		}
		return false
	}
	var kinds []syntax.Kind
	if e.Syntax {
		kinds = syntax.Classify(strings.Split(string(text), "\n"))
	}

	number := &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNamePlaceHolder, v)}
	rows := make([]widget.TextGridRow, len(e.layout.Rows))
	for i, r := range e.layout.Rows {
		cells := make([]widget.TextGridCell, 0, g+r.Indent+r.End-r.Start+1)
		if g > 0 {
			label := ""
			if i == 0 || e.layout.Rows[i-1].Line != r.Line {
				label = strconv.Itoa(r.Line + 1)
			}
			label = fmt.Sprintf("%*s ", g-1, label)
			for _, ch := range label {
				cells = append(cells, widget.TextGridCell{Rune: ch, Style: number})
			}
		}
		for c := 0; c < r.Indent; c++ {
			cells = append(cells, widget.TextGridCell{Rune: ' '})
		}
		var lineStyle widget.TextGridStyle
		if r.Line < len(kinds) {
			lineStyle = kindStyle(kinds[r.Line], th, v)
		}
		for off := r.Start; off < r.End; off++ {
			cell := widget.TextGridCell{Rune: text[off], Style: lineStyle}
			switch {
			case off >= selStart && off < selEnd:
				cell.Style = withBackground(lineStyle, selected.BGColor)
			case highlighted(off):
				cell.Style = withBackground(lineStyle, highlight.BGColor)
			}
			cells = append(cells, cell)
		}
		if e.focused && i == curRow {
			for len(cells) <= g+curCol {
				cells = append(cells, widget.TextGridCell{Rune: ' '})
			}
			cells[g+curCol].Style = cursor
		}
		rows[i] = widget.TextGridRow{Cells: cells}
	}
	e.grid.Rows = rows
	e.grid.Refresh()
	e.BaseWidget.Refresh()
}

// Refresh redraws the editor with the current theme's colours: Fyne
// refreshes every widget when the theme changes, and the syntax colours
// are baked into the grid's cell styles.
func (e *ScriptEditor) Refresh() { e.relayout() }

// kindStyle is how a Fountain element is shown (nil for plain action).
func kindStyle(k syntax.Kind, th fyne.Theme, v fyne.ThemeVariant) widget.TextGridStyle {
	fg := th.Color(theme.ColorNameForeground, v)
	switch k {
	case syntax.SceneHeading:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: fg}
	case syntax.Character:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Bold: true}, FGColor: th.Color(theme.ColorNamePrimary, v)}
	case syntax.Parenthetical:
		return &widget.CustomTextGridStyle{TextStyle: fyne.TextStyle{Italic: true}, FGColor: fg}
	case syntax.Transition, syntax.Centered:
		return &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNamePrimary, v)}
	case syntax.Note, syntax.Section, syntax.Synopsis, syntax.PageBreak, syntax.TitlePage:
		return &widget.CustomTextGridStyle{FGColor: th.Color(theme.ColorNamePlaceHolder, v)}
	}
	return nil
}

// withBackground is a line style with a background colour added.
func withBackground(s widget.TextGridStyle, bg color.Color) widget.TextGridStyle {
	out := &widget.CustomTextGridStyle{BGColor: bg}
	if s != nil {
		out.TextStyle, out.FGColor = s.Style(), s.TextColor()
	}
	return out
}

// changed re-lays out after an edit or cursor move.
func (e *ScriptEditor) changed(edited bool) {
	e.relayout()
	if e.onCursorMoved != nil {
		e.onCursorMoved(e.CursorPosition(), e.cellHeight())
	}
	if e.OnCursorChanged != nil {
		e.OnCursorChanged()
	}
	if edited && e.OnChanged != nil {
		e.OnChanged(e.buf.Text())
	}
}

// Focus handling.

func (e *ScriptEditor) FocusGained() { e.focused = true; e.relayout() }
func (e *ScriptEditor) FocusLost()   { e.focused = false; e.shift = false; e.relayout() }

// Tapped focuses the editor and places the cursor at the click (Shift
// extends the selection).
func (e *ScriptEditor) Tapped(ev *fyne.PointEvent) {
	e.focus()
	e.buf.SetCursor(e.offsetAt(ev.Position), e.shift)
	e.goalCol = -1
	e.changed(false)
}

// DoubleTapped selects the word under the pointer.
func (e *ScriptEditor) DoubleTapped(ev *fyne.PointEvent) {
	e.focus()
	start, end := e.wordAt(e.offsetAt(ev.Position))
	e.buf.Select(start, end)
	e.changed(false)
}

// Dragged selects from where the drag started to the pointer.
func (e *ScriptEditor) Dragged(ev *fyne.DragEvent) {
	if !e.dragging {
		e.focus()
		e.dragging = true
		e.dragFrom = e.offsetAt(ev.Position.Subtract(ev.Dragged))
	}
	e.buf.Select(e.dragFrom, e.offsetAt(ev.Position))
	e.changed(false)
}

// DragEnd ends a drag selection.
func (e *ScriptEditor) DragEnd() { e.dragging = false }

// Cursor shows the text cursor over the editor.
func (e *ScriptEditor) Cursor() desktop.Cursor { return desktop.TextCursor }

func (e *ScriptEditor) focus() {
	if app := fyne.CurrentApp(); app != nil {
		if c := app.Driver().CanvasForObject(e); c != nil {
			c.Focus(e)
		}
	}
}

// offsetAt is the text offset nearest to a position in the editor (clicks
// land between characters).
func (e *ScriptEditor) offsetAt(p fyne.Position) int {
	if len(e.layout.Rows) == 0 {
		return 0
	}
	row := int(p.Y / e.cellHeight())
	col := int(math.Round(float64(p.X/e.cellWidth()))) - e.gutter()
	if row >= len(e.layout.Rows) {
		return e.buf.Len()
	}
	return e.layout.Offset(max(row, 0), max(col, 0))
}

// wordAt is the word (letters and digits) around an offset, or the single
// character there if it is not part of a word.
func (e *ScriptEditor) wordAt(off int) (start, end int) {
	text := []rune(e.buf.Text())
	isWord := func(i int) bool {
		return i >= 0 && i < len(text) && (unicode.IsLetter(text[i]) || unicode.IsDigit(text[i]))
	}
	if !isWord(off) && isWord(off-1) {
		off--
	}
	if !isWord(off) {
		return off, min(off+1, len(text))
	}
	start, end = off, off
	for isWord(start - 1) {
		start--
	}
	for isWord(end) {
		end++
	}
	return start, end
}

// CursorPosition is where the cursor cell is drawn, relative to the editor.
func (e *ScriptEditor) CursorPosition() fyne.Position {
	row, col := e.layout.RowCol(e.buf.Cursor())
	return fyne.NewPos(float32(e.gutter()+col)*e.cellWidth(), float32(row)*e.cellHeight())
}

// KeyDown and KeyUp track Shift for selecting with the arrow keys.
func (e *ScriptEditor) KeyDown(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		e.shift = true
	}
}

func (e *ScriptEditor) KeyUp(k *fyne.KeyEvent) {
	if k.Name == desktop.KeyShiftLeft || k.Name == desktop.KeyShiftRight {
		e.shift = false
	}
}

// TypedRune inserts a typed character.
func (e *ScriptEditor) TypedRune(r rune) {
	e.buf.Insert(string(r))
	e.goalCol = -1
	e.changed(true)
}

// TypedKey handles editing and cursor keys.
func (e *ScriptEditor) TypedKey(k *fyne.KeyEvent) {
	cur := e.buf.Cursor()
	row, col := e.layout.RowCol(cur)
	move := func(off int) {
		e.buf.SetCursor(off, e.shift)
		e.changed(false)
	}
	vertical := func(dRow int) {
		if e.goalCol < 0 {
			e.goalCol = col
		}
		target := row + dRow
		switch {
		case target < 0:
			move(0)
		case target >= len(e.layout.Rows):
			move(e.buf.Len())
		default:
			move(e.layout.Offset(target, e.goalCol))
		}
	}

	switch k.Name {
	case fyne.KeyUp:
		vertical(-1)
		return
	case fyne.KeyDown:
		vertical(1)
		return
	case fyne.KeyPageUp:
		vertical(-e.pageRows())
		return
	case fyne.KeyPageDown:
		vertical(e.pageRows())
		return
	}
	e.goalCol = -1

	switch k.Name {
	case fyne.KeyLeft:
		if s, _ := e.buf.Selection(); e.buf.HasSelection() && !e.shift {
			move(s)
		} else {
			move(cur - 1)
		}
	case fyne.KeyRight:
		if _, end := e.buf.Selection(); e.buf.HasSelection() && !e.shift {
			move(end)
		} else {
			move(cur + 1)
		}
	case fyne.KeyHome:
		move(e.layout.Offset(row, 0))
	case fyne.KeyEnd:
		move(e.layout.Offset(row, math.MaxInt32))
	case fyne.KeyBackspace:
		e.buf.DeleteBackward()
		e.changed(true)
	case fyne.KeyDelete:
		e.buf.DeleteForward()
		e.changed(true)
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.OnEnter != nil && e.OnEnter() {
			return
		}
		e.buf.Insert("\n")
		e.changed(true)
	case fyne.KeyTab:
		e.buf.Insert("    ")
		e.changed(true)
	}
}

func (e *ScriptEditor) pageRows() int {
	return max(int(e.Size().Height/e.cellHeight())-1, 1)
}

// TypedShortcut handles clipboard, select all, undo and redo.
func (e *ScriptEditor) TypedShortcut(s fyne.Shortcut) {
	switch sc := s.(type) {
	case *fyne.ShortcutCopy:
		if e.buf.HasSelection() {
			sc.Clipboard.SetContent(e.buf.SelectedText())
		}
	case *fyne.ShortcutCut:
		if e.buf.HasSelection() {
			sc.Clipboard.SetContent(e.buf.SelectedText())
			e.buf.DeleteBackward()
			e.changed(true)
		}
	case *fyne.ShortcutPaste:
		if text := sc.Clipboard.Content(); text != "" {
			s, end := e.buf.Selection()
			e.buf.Replace(s, end, text)
			e.changed(true)
		}
	case *fyne.ShortcutSelectAll:
		e.buf.SelectAll()
		e.changed(false)
	case *fyne.ShortcutUndo:
		if e.buf.Undo() {
			e.changed(true)
		}
	case *fyne.ShortcutRedo:
		if e.buf.Redo() {
			e.changed(true)
		}
	}
}

// NewScroll puts the editor in a vertical scroll container that keeps the
// cursor in view as it moves.
func NewScroll(e *ScriptEditor) *container.Scroll {
	return newScroll(e, e)
}

// NewPageScroll is NewScroll with the text in a centred column at most
// columns characters wide (plus the line number gutter), like a page;
// the margins scroll too.
func NewPageScroll(e *ScriptEditor, columns int) *container.Scroll {
	return newScroll(e, newPage(e, columns))
}

// page holds the editor's column; a click in the margins beside it goes
// to the editor at the nearest column (left of a line: its start).
type page struct {
	widget.BaseWidget
	e       *ScriptEditor
	content *fyne.Container
}

func newPage(e *ScriptEditor, columns int) *page {
	p := &page{e: e, content: container.New(&pageLayout{e: e, columns: columns}, e)}
	p.ExtendBaseWidget(p)
	return p
}

func (p *page) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(p.content) }

// Tapped handles clicks in the margins (clicks on the text reach the
// editor directly).
func (p *page) Tapped(ev *fyne.PointEvent) {
	at := ev.Position.Subtract(p.e.Position())
	at.X = min(max(at.X, 0), p.e.Size().Width-1)
	at.Y = min(max(at.Y, 0), p.e.Size().Height-1)
	p.e.Tapped(&fyne.PointEvent{Position: at, AbsolutePosition: ev.AbsolutePosition})
}

func newScroll(e *ScriptEditor, content fyne.CanvasObject) *container.Scroll {
	s := container.NewVScroll(content)
	e.onCursorMoved = func(p fyne.Position, h float32) {
		view := s.Size().Height
		y := s.Offset.Y
		switch {
		case p.Y < y:
			y = p.Y
		case p.Y+h > y+view:
			y = p.Y + h - view
		default:
			return
		}
		s.ScrollToOffset(fyne.NewPos(s.Offset.X, max(y, 0)))
	}
	return s
}

// pageLayout centres the editor in a column of a given number of
// characters, or the full width when the window is narrower.
type pageLayout struct {
	e       *ScriptEditor
	columns int
}

func (l *pageLayout) width(avail float32) float32 {
	w := l.e.cellWidth()*float32(l.columns+l.e.gutter()) + 1
	return min(w, avail)
}

// top is the space above the first line.
func (l *pageLayout) top() float32 { return 3 * theme.Padding() }

func (l *pageLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	w := l.width(size.Width)
	l.e.Resize(fyne.NewSize(w, max(size.Height-l.top(), l.e.MinSize().Height)))
	l.e.Move(fyne.NewPos((size.Width-w)/2, l.top()))
}

func (l *pageLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return l.e.MinSize().Add(fyne.NewSize(0, l.top()))
}

var (
	_ fyne.Focusable      = (*ScriptEditor)(nil)
	_ fyne.Draggable      = (*ScriptEditor)(nil)
	_ fyne.DoubleTappable = (*ScriptEditor)(nil)
	_ desktop.Cursorable  = (*ScriptEditor)(nil)
	_ fyne.Shortcutable   = (*ScriptEditor)(nil)
	_ fyne.Tappable       = (*ScriptEditor)(nil)
	_ desktop.Keyable     = (*ScriptEditor)(nil)
)

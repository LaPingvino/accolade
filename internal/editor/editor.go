// Package editor is Accolade's own script editor: a text model (buffer)
// wrapped into rows (wrap) and drawn in a Fyne TextGrid, so selection,
// highlights and single-step undo are under Accolade's control. See
// docs/EDITOR_WIDGET.md; this is step 3 (display and keyboard).
package editor

import (
	"math"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// ScriptEditor edits a screenplay in a monospace grid.
type ScriptEditor struct {
	widget.BaseWidget

	// OnChanged is called with the text after every edit.
	OnChanged func(text string)

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

// SetText replaces the text and forgets the undo history.
func (e *ScriptEditor) SetText(text string) {
	e.buf.SetText(text)
	e.changed(false)
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
	e.layout = wrap.Wrap(text, e.columns)

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
	selStart, selEnd := e.buf.Selection()
	curRow, curCol := e.layout.RowCol(e.buf.Cursor())

	rows := make([]widget.TextGridRow, len(e.layout.Rows))
	for i, r := range e.layout.Rows {
		cells := make([]widget.TextGridCell, 0, r.Indent+r.End-r.Start+1)
		for c := 0; c < r.Indent; c++ {
			cells = append(cells, widget.TextGridCell{Rune: ' '})
		}
		for off := r.Start; off < r.End; off++ {
			cell := widget.TextGridCell{Rune: text[off]}
			if off >= selStart && off < selEnd {
				cell.Style = selected
			}
			cells = append(cells, cell)
		}
		if e.focused && i == curRow {
			for len(cells) <= curCol {
				cells = append(cells, widget.TextGridCell{Rune: ' '})
			}
			cells[curCol].Style = cursor
		}
		rows[i] = widget.TextGridRow{Cells: cells}
	}
	e.grid.Rows = rows
	e.grid.Refresh()
	e.Refresh()
}

// changed re-lays out after an edit or cursor move.
func (e *ScriptEditor) changed(edited bool) {
	e.relayout()
	if e.onCursorMoved != nil {
		e.onCursorMoved(e.CursorPosition(), e.cellHeight())
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
	col := int(math.Round(float64(p.X / e.cellWidth())))
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
	return fyne.NewPos(float32(col)*e.cellWidth(), float32(row)*e.cellHeight())
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
	s := container.NewVScroll(e)
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

var (
	_ fyne.Focusable      = (*ScriptEditor)(nil)
	_ fyne.Draggable      = (*ScriptEditor)(nil)
	_ fyne.DoubleTappable = (*ScriptEditor)(nil)
	_ desktop.Cursorable  = (*ScriptEditor)(nil)
	_ fyne.Shortcutable   = (*ScriptEditor)(nil)
	_ fyne.Tappable       = (*ScriptEditor)(nil)
	_ desktop.Keyable     = (*ScriptEditor)(nil)
)

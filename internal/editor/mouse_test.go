package editor

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// at is the position of a cell (column, row) plus a fraction of a cell.
func at(e *ScriptEditor, col, row float32) fyne.Position {
	return fyne.NewPos(col*e.cellWidth(), (row+0.5)*e.cellHeight())
}

func TestTapPlacesCursor(t *testing.T) {
	e := newEditor(t, "Rain hammers the roof.", 10) // "Rain " | "hammers " | "the roof."
	e.Tapped(&fyne.PointEvent{Position: at(e, 3.4, 0)})
	if e.Buffer().Cursor() != 3 {
		t.Errorf("tap near column 3: cursor %d", e.Buffer().Cursor())
	}
	e.Tapped(&fyne.PointEvent{Position: at(e, 2.6, 1)}) // between "ha" and "m" on row 1
	if e.Buffer().Cursor() != 8 {
		t.Errorf("tap on the wrapped row: cursor %d, want 8", e.Buffer().Cursor())
	}
	e.Tapped(&fyne.PointEvent{Position: at(e, 2, 9)}) // below the text
	if e.Buffer().Cursor() != e.Buffer().Len() {
		t.Errorf("tap below the text: %d", e.Buffer().Cursor())
	}
	shift(e, true)
	e.Tapped(&fyne.PointEvent{Position: at(e, 0, 1)})
	shift(e, false)
	if s, end := e.Buffer().Selection(); s != 5 || end != e.Buffer().Len() {
		t.Errorf("shift+tap selection %d-%d", s, end)
	}
}

func TestDragSelects(t *testing.T) {
	e := newEditor(t, "one two three", 40)
	start := at(e, 4, 0)
	e.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: at(e, 6, 0)}, Dragged: fyne.Delta{DX: 2 * e.cellWidth()}})
	e.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: at(e, 7, 0)}, Dragged: fyne.Delta{DX: e.cellWidth()}})
	e.DragEnd()
	if got := e.Buffer().SelectedText(); got != "two" {
		t.Errorf("drag from %v selected %q", start, got)
	}
	// a new drag starts a new selection
	e.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: at(e, 3, 0)}, Dragged: fyne.Delta{DX: 3 * e.cellWidth()}})
	e.DragEnd()
	if got := e.Buffer().SelectedText(); got != "one" {
		t.Errorf("second drag selected %q", got)
	}
}

func TestDoubleTapSelectsWord(t *testing.T) {
	e := newEditor(t, "INT. BARN - DAY", 40)
	e.DoubleTapped(&fyne.PointEvent{Position: at(e, 6.2, 0)})
	if got := e.Buffer().SelectedText(); got != "BARN" {
		t.Errorf("double tap in BARN selected %q", got)
	}
	e.DoubleTapped(&fyne.PointEvent{Position: at(e, 10.2, 0)})
	if got := e.Buffer().SelectedText(); got != "-" {
		t.Errorf("double tap on punctuation selected %q", got)
	}
}

func TestScrollFollowsCursor(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	e := New(strings.Repeat("line\n", 100))
	s := NewScroll(e)
	w := test.NewWindow(s)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(400, 10*e.cellHeight()))
	s.Resize(fyne.NewSize(400, 10*e.cellHeight()))

	e.Buffer().SetCursor(e.Buffer().Len(), false)
	e.changed(false)
	p := e.CursorPosition()
	if s.Offset.Y <= 0 || p.Y+e.cellHeight() > s.Offset.Y+s.Size().Height+0.5 || p.Y < s.Offset.Y {
		t.Errorf("cursor at %v not in view (offset %v, height %v)", p, s.Offset, s.Size().Height)
	}
	e.Buffer().SetCursor(0, false)
	e.changed(false)
	if s.Offset.Y != 0 {
		t.Errorf("back at the top: offset %v", s.Offset)
	}
}

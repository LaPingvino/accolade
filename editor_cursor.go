package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Fyne's Entry has no public API to place the cursor at a text offset or
// to select a range. These helpers get there through the public cursor
// fields and the same key events a user would send.

// rowStart is the rune offset where the given (possibly wrapped) row
// begins. Rows past the end report 0, like Entry.CursorTextOffset does.
func rowStart(e *widget.Entry, row int) int {
	e.CursorRow, e.CursorColumn = row, 0
	return e.CursorTextOffset()
}

// setCursorOffset moves the cursor to the given rune offset.
func setCursorOffset(e *widget.Entry, offset int) {
	// Rows begin at strictly increasing offsets and every row after the
	// first begins past 0, so "row starts at or before offset" holds for
	// rows 0..k and fails after: binary search for k.
	lo, hi := 0, 1
	for rowStart(e, hi) > 0 && rowStart(e, hi) <= offset {
		lo, hi = hi, hi*2
	}
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if start := rowStart(e, mid); start > 0 && start <= offset {
			lo = mid
		} else {
			hi = mid
		}
	}
	start := rowStart(e, lo)
	e.CursorRow, e.CursorColumn = lo, offset-start
	e.Refresh()
}

// clearSelection drops any selection, leaving the text untouched.
func clearSelection(e *widget.Entry) {
	if e.SelectedText() == "" {
		return
	}
	row, col := e.CursorRow, e.CursorColumn
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	e.CursorRow, e.CursorColumn = row, col
	e.Refresh()
}

// selectRange selects the runes in [start, end), leaving the cursor at end.
func selectRange(e *widget.Entry, start, end int) {
	clearSelection(e)
	setCursorOffset(e, start)
	shift := &fyne.KeyEvent{Name: desktop.KeyShiftLeft}
	e.KeyDown(shift)
	for i := start; i < end; i++ {
		e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	}
	e.KeyUp(shift)
}

// typeOverSelection replaces the selection with text as if it were typed,
// so the change lands on the Entry's undo stack.
func typeOverSelection(e *widget.Entry, text string) {
	if text == "" {
		e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
		return
	}
	for _, r := range text {
		if r == '\n' {
			e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		} else {
			e.TypedRune(r)
		}
	}
}

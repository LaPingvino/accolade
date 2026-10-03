package editor

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func newEditor(t *testing.T, text string, columns int) *ScriptEditor {
	t.Helper()
	a := test.NewApp()
	t.Cleanup(a.Quit)
	e := New(text)
	w := test.NewWindow(e)
	t.Cleanup(w.Close)
	e.Resize(fyne.NewSize(e.cellWidth()*float32(columns)+1, 300))
	w.Canvas().Focus(e)
	return e
}

func key(e *ScriptEditor, name fyne.KeyName) { e.TypedKey(&fyne.KeyEvent{Name: name}) }

func shift(e *ScriptEditor, down bool) {
	if down {
		e.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	} else {
		e.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	}
}

// gridText is what the grid shows, one string per row.
func gridText(e *ScriptEditor) []string {
	var out []string
	for i := range e.grid.Rows {
		out = append(out, strings.TrimRight(e.grid.RowText(i), " "))
	}
	return out
}

func TestTypingAndEditingKeys(t *testing.T) {
	e := newEditor(t, "", 40)
	var changes int
	e.OnChanged = func(string) { changes++ }
	test.Type(e, "INT. BARN")
	key(e, fyne.KeyReturn)
	test.Type(e, "Rainn")
	key(e, fyne.KeyBackspace)
	key(e, fyne.KeyLeft)
	key(e, fyne.KeyLeft)
	key(e, fyne.KeyDelete) // removes the "i"
	if got := e.Text(); got != "INT. BARN\nRan" {
		t.Errorf("text %q", got)
	}
	if changes != len("INT. BARN")+1+len("Rainn")+2 {
		t.Errorf("OnChanged called %d times", changes)
	}
}

func TestWrappingAndResize(t *testing.T) {
	e := newEditor(t, "Rain hammers the roof.", 10)
	if got := gridText(e); strings.Join(got, "|") != "Rain|hammers|the roof." {
		t.Errorf("wrapped rows %q", got)
	}
	e.Resize(fyne.NewSize(e.cellWidth()*40+1, 300))
	if e.Columns() != 40 || len(e.grid.Rows) != 1 {
		t.Errorf("after widening: %d columns, %d rows", e.Columns(), len(e.grid.Rows))
	}
}

func TestVerticalMovesKeepColumn(t *testing.T) {
	e := newEditor(t, "abcdef\nab\nabcdef", 40)
	e.Buffer().SetCursor(4, false) // "abcd|ef"
	key(e, fyne.KeyDown)           // short line: end of "ab"
	if e.Buffer().Cursor() != 9 {
		t.Fatalf("down to the short line: %d", e.Buffer().Cursor())
	}
	key(e, fyne.KeyDown) // back to column 4
	if e.Buffer().Cursor() != 14 {
		t.Errorf("down again: %d, want 14 (column kept)", e.Buffer().Cursor())
	}
	key(e, fyne.KeyUp)
	key(e, fyne.KeyUp)
	key(e, fyne.KeyUp) // past the top: start of text
	if e.Buffer().Cursor() != 0 {
		t.Errorf("up past the top: %d", e.Buffer().Cursor())
	}
}

func TestHomeEndOnWrappedRow(t *testing.T) {
	e := newEditor(t, "Rain hammers the roof.", 10) // "Rain " | "hammers " | "the roof."
	e.Buffer().SetCursor(8, false)                  // in "hammers"
	key(e, fyne.KeyHome)
	if e.Buffer().Cursor() != 5 {
		t.Errorf("Home: %d, want 5", e.Buffer().Cursor())
	}
	key(e, fyne.KeyEnd)
	if e.Buffer().Cursor() != 12 {
		t.Errorf("End: %d, want 12 (before the wrap)", e.Buffer().Cursor())
	}
}

func TestShiftSelectionAndClipboard(t *testing.T) {
	e := newEditor(t, "one two three", 40)
	clip := test.NewClipboard()
	e.Buffer().SetCursor(4, false)
	shift(e, true)
	for i := 0; i < 3; i++ {
		key(e, fyne.KeyRight)
	}
	shift(e, false)
	if e.Buffer().SelectedText() != "two" {
		t.Fatalf("selected %q", e.Buffer().SelectedText())
	}
	e.TypedShortcut(&fyne.ShortcutCut{Clipboard: clip})
	if e.Text() != "one  three" || clip.Content() != "two" {
		t.Errorf("cut: %q, clipboard %q", e.Text(), clip.Content())
	}
	e.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clip})
	e.TypedShortcut(&fyne.ShortcutPaste{Clipboard: clip})
	if e.Text() != "one twotwo three" {
		t.Errorf("paste: %q", e.Text())
	}
	e.TypedShortcut(&fyne.ShortcutUndo{})
	if e.Text() != "one two three" {
		t.Errorf("undo of a paste is one step: %q", e.Text())
	}
	e.TypedShortcut(&fyne.ShortcutRedo{})
	e.TypedShortcut(&fyne.ShortcutSelectAll{})
	e.TypedShortcut(&fyne.ShortcutCopy{Clipboard: clip})
	if clip.Content() != e.Text() {
		t.Errorf("select all + copy: %q", clip.Content())
	}
	key(e, fyne.KeyLeft) // collapses the selection to its start
	if e.Buffer().HasSelection() || e.Buffer().Cursor() != 0 {
		t.Errorf("left with a selection: cursor %d", e.Buffer().Cursor())
	}
}

func TestSelectionAndCursorStyles(t *testing.T) {
	e := newEditor(t, "abc", 40)
	e.Buffer().Select(1, 2)
	e.changed(false)
	row := e.grid.Rows[0]
	if row.Cells[1].Style == nil || row.Cells[0].Style != nil {
		t.Errorf("selection style on the wrong cells")
	}
	if row.Cells[2].Style == nil {
		t.Errorf("cursor cell (offset 2) not drawn")
	}
	e.FocusLost()
	if e.grid.Rows[0].Cells[2].Style != nil {
		t.Errorf("cursor drawn without focus")
	}
}

func TestPageColumnIsCentred(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	e := New(strings.Repeat("word ", 60))
	s := NewPageScroll(e, 40)
	w := test.NewWindow(s)
	t.Cleanup(w.Close)
	col := e.cellWidth() * 40

	w.Resize(fyne.NewSize(col*2, 400))
	if e.Columns() != 40 {
		t.Errorf("wide window: %d columns, want 40", e.Columns())
	}
	left, right := e.Position().X, s.Size().Width-e.Position().X-e.Size().Width
	if left < col/3 || abs(left-right) > 2 {
		t.Errorf("column not centred: %v left, %v right", left, right)
	}

	w.Resize(fyne.NewSize(col/2, 400))
	if e.Columns() >= 40 || e.Position().X > 1 {
		t.Errorf("narrow window: %d columns at x=%v, want the full width", e.Columns(), e.Position().X)
	}
}

func TestClickInMarginPlacesCursor(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	e := New("first line\nsecond line")
	s := NewPageScroll(e, 40)
	w := test.NewWindow(s)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(e.cellWidth()*80, 300))

	row2 := e.Position().Y + e.cellHeight()*1.5
	test.TapCanvas(w.Canvas(), fyne.NewPos(e.Position().X/2, row2))
	if got := e.CursorOffset(); got != len("first line\n") {
		t.Errorf("click left of the second line: cursor at %d, want its start", got)
	}
	test.TapCanvas(w.Canvas(), fyne.NewPos(s.Size().Width-2, e.Position().Y+e.cellHeight()/2))
	if got := e.CursorOffset(); got != len("first line") {
		t.Errorf("click right of the first line: cursor at %d, want its end", got)
	}
}

func abs(f float32) float32 { return max(f, -f) }

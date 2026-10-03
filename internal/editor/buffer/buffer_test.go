package buffer

import "testing"

func typeText(b *Buffer, s string) {
	for _, r := range s {
		b.Insert(string(r))
	}
}

func TestTypingAndUndoByWord(t *testing.T) {
	b := New("")
	typeText(b, "INT. BARN")
	if b.Text() != "INT. BARN" || b.Cursor() != 9 {
		t.Fatalf("typed %q, cursor %d", b.Text(), b.Cursor())
	}
	// steps: "INT", ".", " ", "BARN"
	for _, want := range []string{"INT. ", "INT.", "INT", ""} {
		b.Undo()
		if b.Text() != want {
			t.Errorf("after undo: %q, want %q", b.Text(), want)
		}
	}
	if b.Undo() {
		t.Error("undo with nothing left")
	}
	for _, want := range []string{"INT", "INT.", "INT. ", "INT. BARN"} {
		b.Redo()
		if b.Text() != want {
			t.Errorf("after redo: %q, want %q", b.Text(), want)
		}
	}
	if b.Cursor() != 9 {
		t.Errorf("cursor after redo %d", b.Cursor())
	}
}

func TestReplaceIsOneUndoStep(t *testing.T) {
	b := New("Hello world, hello moon")
	b.Replace(13, 18, "bye")
	if b.Text() != "Hello world, bye moon" || b.Cursor() != 16 {
		t.Fatalf("replace: %q cursor %d", b.Text(), b.Cursor())
	}
	b.Undo()
	if b.Text() != "Hello world, hello moon" {
		t.Errorf("one undo should restore: %q", b.Text())
	}
}

func TestGroupIsOneUndoStep(t *testing.T) {
	b := New("int. barn\nrain")
	b.Group(func() {
		b.Replace(0, 9, "INT. BARN")
		b.SetCursor(9, false)
		b.Insert("\n")
		b.Insert("  ")
	})
	if b.Text() != "INT. BARN\n  \nrain" {
		t.Fatalf("group: %q", b.Text())
	}
	b.Undo()
	if b.Text() != "int. barn\nrain" {
		t.Errorf("group undo: %q", b.Text())
	}
	b.Redo()
	if b.Text() != "INT. BARN\n  \nrain" || b.Cursor() != 12 {
		t.Errorf("group redo: %q cursor %d", b.Text(), b.Cursor())
	}
}

func TestSelectionTypingAndDeleting(t *testing.T) {
	b := New("one two three")
	b.Select(4, 7)
	if b.SelectedText() != "two" {
		t.Fatalf("selected %q", b.SelectedText())
	}
	b.Insert("2")
	if b.Text() != "one 2 three" || b.HasSelection() {
		t.Errorf("typing over selection: %q", b.Text())
	}
	b.SetCursor(5, false)
	b.DeleteBackward()
	b.DeleteForward()
	if b.Text() != "one three" {
		t.Errorf("deletes: %q", b.Text())
	}
	b.SetCursor(0, false)
	b.DeleteBackward() // nothing before the start
	b.SetCursor(b.Len(), false)
	b.DeleteForward() // nothing after the end
	if b.Text() != "one three" {
		t.Errorf("deleting past the ends: %q", b.Text())
	}
	b.SetCursor(0, false)
	b.SetCursor(3, true)
	if s, e := b.Selection(); s != 0 || e != 3 {
		t.Errorf("extend selection %d-%d", s, e)
	}
	b.SelectAll()
	b.DeleteBackward()
	if b.Text() != "" {
		t.Errorf("select all + delete: %q", b.Text())
	}
}

func TestTypingAfterMovingStartsNewStep(t *testing.T) {
	b := New("")
	typeText(b, "ab")
	b.SetCursor(0, false)
	typeText(b, "x")
	b.Undo()
	if b.Text() != "ab" {
		t.Errorf("undo should only remove x: %q", b.Text())
	}
}

func TestNewEditClearsRedo(t *testing.T) {
	b := New("")
	typeText(b, "abc")
	b.Undo()
	typeText(b, "x")
	if b.Redo() || b.Text() != "x" {
		t.Errorf("redo after a new edit: %q", b.Text())
	}
}

func TestUnicode(t *testing.T) {
	b := New("être")
	b.SetCursor(1, false)
	b.DeleteBackward()
	if b.Text() != "tre" || b.Len() != 3 {
		t.Errorf("rune offsets: %q", b.Text())
	}
}

func TestLines(t *testing.T) {
	b := New("FADE IN:\n\nINT. BARN\nRain.")
	if b.LineCount() != 4 || b.Line(2) != "INT. BARN" || b.Line(1) != "" {
		t.Errorf("lines: %d %q %q", b.LineCount(), b.Line(2), b.Line(1))
	}
	if b.LineStart(2) != 10 || b.LineEnd(2) != 19 || b.LineStart(9) != b.Len() {
		t.Errorf("line bounds %d %d", b.LineStart(2), b.LineEnd(2))
	}
	if l, c := b.Position(13); l != 2 || c != 3 {
		t.Errorf("Position(13) = %d,%d", l, c)
	}
	if b.Offset(2, 3) != 13 || b.Offset(0, 99) != 8 || b.Offset(99, 0) != b.Len() {
		t.Errorf("Offset: %d %d", b.Offset(2, 3), b.Offset(0, 99))
	}
	for off := 0; off <= b.Len(); off++ {
		if l, c := b.Position(off); b.Offset(l, c) != off {
			t.Errorf("round trip %d -> %d,%d", off, l, c)
		}
	}
	if New("").LineCount() != 1 {
		t.Error("empty text has one line")
	}
}

func TestSetTextForgetsHistory(t *testing.T) {
	b := New("")
	typeText(b, "draft")
	b.SetText("loaded")
	if b.CanUndo() || b.Text() != "loaded" || b.Cursor() != 0 {
		t.Errorf("SetText: undo %v %q", b.CanUndo(), b.Text())
	}
}

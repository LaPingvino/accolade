package buffer

import "testing"

// Typing at three cursors at once, one undo step; the others move with
// edits elsewhere, and moving keeps the primary.
func TestSpans(t *testing.T) {
	b := New("ANNA\nBRAM\nCEES\n")
	b.SetSpans([]Span{{0, 0}, {5, 5}, {10, 10}})
	b.EditSpans(func(s Span) (int, int, string) { return s.Start(), s.End(), "@" })
	if b.Text() != "@ANNA\n@BRAM\n@CEES\n" {
		t.Fatalf("typed: %q", b.Text())
	}
	if got := b.Spans(); len(got) != 3 || got[0].Cursor != 1 || got[1].Cursor != 7 || got[2].Cursor != 13 {
		t.Errorf("cursors %v", got)
	}
	b.EditSpans(func(s Span) (int, int, string) { return s.Start() - 1, s.End(), "" }) // backspace
	if b.Text() != "ANNA\nBRAM\nCEES\n" {
		t.Errorf("deleted: %q", b.Text())
	}
	b.Undo()
	if b.Text() != "@ANNA\n@BRAM\n@CEES\n" || b.HasOthers() {
		t.Errorf("undo: %q", b.Text())
	}

	b = New("rain rain rain")
	b.SetSpans([]Span{{0, 4}, {5, 9}, {10, 14}}) // the words selected
	b.EditSpans(func(s Span) (int, int, string) { return s.Start(), s.End(), "snow" })
	if b.Text() != "snow snow snow" {
		t.Errorf("replaced: %q", b.Text())
	}
	b.SetSpans([]Span{{0, 0}, {5, 5}})
	b.Select(0, 0) // a move: the primary only
	if b.HasOthers() {
		t.Error("moving kept the others")
	}
	b.SetSpans([]Span{{0, 4}, {2, 6}, {0, 4}}) // overlapping: left out
	if len(b.Spans()) != 1 {
		t.Errorf("overlaps kept: %v", b.Spans())
	}
}

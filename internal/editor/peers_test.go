package editor

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// A co-writer's bar is drawn in the gutter on their line, in their
// colour, moves with the text as this side types above it, and goes to a
// fold's heading while their line is folded away; their selection is
// tinted; a tap on the bar tells whose it is without moving the cursor;
// typing is untouched by any of it.
func TestPeers(t *testing.T) {
	e := newEditor(t, "INT. BARN - DAY\n\nRain falls.\n\nINT. SHED - DAY\n\nTools.\n", 40)
	e.SetLineNumbers(true)
	red := color.NRGBA{R: 0xe0, G: 0x31, B: 0x31, A: 0xff}
	rain := len([]rune("INT. BARN - DAY\n\n"))
	e.SetPeers([]Peer{{Name: "Bob", Color: red, Cursor: rain + 4, From: rain}})
	if got := e.PeerLines(); got[2] != "Bob" || len(got) != 1 {
		t.Fatalf("bars on %v", got)
	}
	g := e.gutter()
	bar := e.grid.Rows[2].Cells[g-1]
	if bar.Rune != markBar || bar.Style == nil || bar.Style.TextColor() != red {
		t.Errorf("no red bar on line 2: %q %+v", bar.Rune, bar.Style)
	}
	if c := e.grid.Rows[2].Cells[g]; c.Style == nil || c.Style.BackgroundColor() == nil {
		t.Error("Bob's selection is not tinted")
	}
	if c := e.grid.Rows[1].Cells[g-1]; c.Rune == markBar {
		t.Error("a bar on a line nobody is on")
	}
	e.SetCursorOffset(0)
	test.Type(e, "FADE IN:\n\n")
	if got := e.PeerLines(); got[4] != "Bob" {
		t.Errorf("after typing above, bars on %v", got)
	}
	if p := e.Peers()[0]; p.Cursor != rain+14 || p.From != rain+10 {
		t.Errorf("Bob at %d from %d", p.Cursor, p.From)
	}
	cursor := e.CursorOffset()
	g = e.gutter() // ten lines now
	var told string
	e.OnPeer = func(names string, _ fyne.Position, _ fyne.Size) { told = names }
	e.Tapped(&fyne.PointEvent{Position: fyne.NewPos((float32(g)-0.5)*e.cellWidth(), 4.5*e.cellHeight())})
	if told != "Bob" || e.CursorOffset() != cursor {
		t.Errorf("tapped the bar: told %q, cursor %d (was %d)", told, e.CursorOffset(), cursor)
	}
	e.Folds = func() []Fold { return []Fold{{From: 3, To: 6}} }
	e.relayout()
	if got := e.PeerLines(); got[2] != "Bob" {
		t.Errorf("folded away: bars on %v", got)
	}
	e.SetPeers(nil)
	if len(e.PeerLines()) != 0 {
		t.Error("bars left after the peers went")
	}
}

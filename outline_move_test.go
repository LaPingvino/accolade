package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// Scenes and sections move with what they hold, before another or to the end.
func TestMoveOutline(t *testing.T) {
	const script = "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n"
	starts := []string{"INT.", "EXT.", "INT ", "EXT "}
	for _, c := range []struct {
		from, to int
		want     string
	}{
		// the field before the barn
		{2, 1, "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n"},
		// act two first, with its scene
		{3, 0, "# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n\n# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n"},
		// the barn to the end
		{1, 5, "# ACT ONE\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n\nINT. HOUSE - NIGHT\n\nFire.\n\nINT. BARN - DAY\n\nRain.\n"},
		// the house into act one
		{4, 2, "# ACT ONE\n\nINT. BARN - DAY\n\nRain.\n\nINT. HOUSE - NIGHT\n\nFire.\n\nEXT. FIELD - DAY\n\nWind.\n\n# ACT TWO\n"},
	} {
		got, ok := moveOutline(script, starts, c.from, c.to)
		if !ok || got != c.want {
			t.Errorf("%d before %d:\n%q\nwant\n%q", c.from, c.to, got, c.want)
		}
	}
	if _, ok := moveOutline(script, starts, 0, 1); ok {
		t.Error("an act into itself")
	}
}

// Dragging an outline row up one row moves its scene before that one.
func TestOutlineDrag(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Cleanup(func() { w.settings.SetBoolean("outline-visible", false) })
	w.textEditor.SetText("INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n")
	if !w.outlineVisible {
		w.toggleOutline()
	}
	o := w.outline
	o.update(w.textEditor.Text())
	if o.dropIndex(1, -o.oneLine) != 0 || o.dropIndex(0, o.oneLine*1.5) != 2 || o.dropIndex(0, 2) != -1 {
		t.Fatalf("drop places: %d %d %d", o.dropIndex(1, -o.oneLine), o.dropIndex(0, o.oneLine*1.5), o.dropIndex(0, 2))
	}
	r := newDragRow(o, container.NewStack())
	r.id = 1
	r.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DY: -o.oneLine}})
	r.DragEnd()
	if got := w.textEditor.Text(); got != "EXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n" {
		t.Errorf("dragged: %q", got)
	}
	w.textEditor.Undo()
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n" {
		t.Errorf("one undo: %q", got)
	}
}

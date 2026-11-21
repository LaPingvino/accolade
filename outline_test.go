package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"strings"
	"testing"
)

const outlineScript = `Title: T

# ACT ONE

= Anna wants the biscuit.

INT. KITCHEN - DAY #1#

= She finds the tin empty.

Anna looks.

## The garden

.THE GARDEN

EXT. ROAD - NIGHT
Action right after the heading.
`

func TestOutlineOf(t *testing.T) {
	var got []string
	for _, it := range outlineOf(outlineScript, nil) {
		got = append(got, strings.Join([]string{strings.Repeat(">", it.Level), it.Text, it.Synopsis}, "|"))
	}
	want := []string{
		">|ACT ONE|Anna wants the biscuit.",
		">>|INT. KITCHEN - DAY|She finds the tin empty.",
		">>|The garden|",
		">>>|THE GARDEN|",
		">>>|EXT. ROAD - NIGHT|",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("outline:\n%s", strings.Join(got, "\n"))
	}
}

func TestOutlinePanelGoesToTheScene(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	if !w.paneShown("outline") || len(w.outline.items) != 5 {
		t.Fatalf("outline: %v, %d items", w.paneShown("outline"), len(w.outline.items))
	}
	w.outline.list.Select(1) // INT. KITCHEN
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 6 {
		t.Errorf("cursor on line %d, want 6", line)
	}
	w.textEditor.SetText(outlineScript + "\nINT. HALL - DAY\n")
	if len(w.outline.items) != 6 {
		t.Errorf("outline not updated: %d items", len(w.outline.items))
	}
	capture(t, w.fyneWindow.Canvas(), "outline.png")
}

func TestOutlineAndNotesTogether(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	w.togglePane("notes")
	if !w.paneShown("outline") || !w.paneShown("notes") {
		t.Fatalf("outline %v, notes %v", w.paneShown("outline"), w.paneShown("notes"))
	}
	for name, box := range map[string]fyne.CanvasObject{"outline": w.outline.box, "notes": w.notesPane().entry} {
		if !box.Visible() || box.Size().Height < 50 {
			t.Errorf("%s: shown %v, %v", name, box.Visible(), box.Size())
		}
	}
	if fyne.CurrentApp().Driver().AbsolutePositionForObject(w.outline.box).Y >= fyne.CurrentApp().Driver().AbsolutePositionForObject(w.notesPane().entry).Y {
		t.Error("the notes are not below the outline")
	}
	capture(t, w.fyneWindow.Canvas(), "outline-notes.png")
	w.togglePane("outline")
	if w.paneShown("outline") || !w.paneShown("notes") {
		t.Errorf("closing the outline: outline %v, notes %v", w.paneShown("outline"), w.paneShown("notes"))
	}
}

// Full screen hides the menu bar, toolbar and status bar, keeps the
// outline and notes open, and has a button to leave; the menu's shortcuts
// still work.
func TestFullscreenWithoutDistractions(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(outlineScript)
	w.togglePane("outline")
	w.toggleFullscreen()
	if w.headerArea.Visible() || w.statusBar.Visible() || w.fyneWindow.MainMenu() != nil {
		t.Errorf("full screen shows toolbar %v, status %v, menu %v",
			w.headerArea.Visible(), w.statusBar.Visible(), w.fyneWindow.MainMenu() != nil)
	}
	if !w.paneShown("outline") {
		t.Error("the outline closed")
	}
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.TypedShortcut(shortcut(fyne.KeyK, fyne.KeyModifierShift)) // Notes
	if !w.paneShown("notes") {
		t.Error("the Notes shortcut did nothing in full screen")
	}
	leave := findButton(w.fyneWindow.Content(), "Leave full screen")
	if leave == nil {
		t.Fatal("no button to leave full screen")
	}
	test.Tap(leave)
	if w.isFullscreen || !w.headerArea.Visible() || !w.statusBar.Visible() || w.fyneWindow.MainMenu() == nil {
		t.Errorf("after full screen: %v, toolbar %v, status %v", w.isFullscreen, w.headerArea.Visible(), w.statusBar.Visible())
	}
	if findButton(w.fyneWindow.Content(), "Leave full screen") != nil {
		t.Error("the button stayed")
	}
}

// In full screen with Cards or the Stripboard, "Leave full screen" and the
// view's "Back to the Script" are both in sight and do not overlap (the
// leave button used to float over the view's top-right corner).
func TestFullscreenLeaveDoesNotCoverBack(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.toggleFullscreen()
	t.Cleanup(func() {
		if w.isFullscreen {
			w.toggleFullscreen()
		}
	})
	for _, view := range []struct {
		name   string
		toggle func()
	}{{"cards", w.toggleCards}, {"stripboard", w.toggleStripboard}} {
		view.toggle()
		c := w.fyneWindow.Content()
		leave, back := findButton(c, "Leave full screen"), findButton(c, "Back to the Script")
		if leave == nil || back == nil {
			t.Fatalf("%s: leave %v, back %v", view.name, leave != nil, back != nil)
		}
		d := fyne.CurrentApp().Driver()
		lp, bp := d.AbsolutePositionForObject(leave), d.AbsolutePositionForObject(back)
		ls, bs := leave.Size(), back.Size()
		if lp.X < bp.X+bs.Width && bp.X < lp.X+ls.Width && lp.Y < bp.Y+bs.Height && bp.Y < lp.Y+ls.Height {
			t.Errorf("%s: Leave full screen at %v %v covers Back to the Script at %v %v", view.name, lp, ls, bp, bs)
		}
		view.toggle()
	}
}

// findButton is the button with text in o.
func findButton(o fyne.CanvasObject, text string) *widget.Button {
	switch x := o.(type) {
	case *widget.Button:
		if x.Text == text {
			return x
		}
	case *fyne.Container:
		for _, c := range x.Objects {
			if b := findButton(c, text); b != nil {
				return b
			}
		}
	}
	return nil
}

// The filter keeps what matches, with the sections it is in.
func TestFilterOutline(t *testing.T) {
	items := outlineOf("# Act One\n\n## Morning\n\nINT. BARN - DAY\n\n= Rain.\n\n## Night\n\nEXT. FIELD - NIGHT\n\n# Act Two\n\nINT. BARN - NIGHT\n", []string{"INT", "EXT"})
	var got []string
	for _, it := range filterOutline(items, "rain") {
		got = append(got, it.Text)
	}
	if strings.Join(got, "|") != "Act One|Morning|INT. BARN - DAY" {
		t.Errorf("rain: %q", got)
	}
	got = nil
	for _, it := range filterOutline(items, "barn") {
		got = append(got, it.Text)
	}
	if strings.Join(got, "|") != "Act One|Morning|INT. BARN - DAY|Act Two|INT. BARN - NIGHT" {
		t.Errorf("barn: %q", got)
	}
	if len(filterOutline(items, " ")) != len(items) {
		t.Error("an empty filter hides")
	}
}

// A folded section shows without what is in it, up to the next of its
// level or higher.
func TestUnfolded(t *testing.T) {
	items := outlineOf("# Act One\n\n## Morning\n\nINT. BARN - DAY\n\n## Night\n\nEXT. FIELD - NIGHT\n\n# Act Two\n\nINT. BARN - NIGHT\n", []string{"INT", "EXT"})
	text := func(folded ...string) string {
		f := map[string]bool{}
		for _, k := range folded {
			f[k] = true
		}
		var got []string
		for _, it := range unfolded(items, f) {
			got = append(got, it.Text)
		}
		return strings.Join(got, "|")
	}
	if got := text("2 Morning"); got != "Act One|Morning|Night|EXT. FIELD - NIGHT|Act Two|INT. BARN - NIGHT" {
		t.Errorf("Morning folded: %s", got)
	}
	if got := text("1 Act One"); got != "Act One|Act Two|INT. BARN - NIGHT" {
		t.Errorf("Act One folded: %s", got)
	}
	if !hasInside(items, 0) || hasInside(items, 2) {
		t.Error("hasInside")
	}
}

// Colored headings: the outline and the cards show the color, not the
// note; a color Accolade does not know shows nothing.
func TestOutlineColors(t *testing.T) {
	script := "# ACT ONE [[color BLUE]]\n\nINT. BARN - DAY #12# [[red]] [[a note]]\n\nRain.\n\nEXT. FIELD - DAY [[#zzzzzz]]\n\nWind.\n\n.MONTAGE [[#00ff00]]\n"
	items := outlineOf(script, nil)
	var got []string
	for _, it := range items {
		got = append(got, it.Text+"|"+it.Color)
	}
	if want := "ACT ONE|blue\nINT. BARN - DAY|red\nEXT. FIELD - DAY|#zzzzzz\nMONTAGE|#00ff00"; strings.Join(got, "\n") != want {
		t.Errorf("outline:\n%s", strings.Join(got, "\n"))
	}

	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(script)
	w.togglePane("outline")
	o := w.outline
	for id, want := range []color.Color{color.NRGBA{0, 129, 239, 255}, color.NRGBA{239, 0, 73, 255}, color.Transparent, color.NRGBA{0, 255, 0, 255}} {
		row := o.list.CreateItem()
		o.list.UpdateItem(id, row)
		r := row.(*dragRow).content.Objects[2].(*fyne.Container).Objects[0].(*canvas.Rectangle)
		if r.FillColor != want {
			t.Errorf("row %d: swatch %v, want %v", id, r.FillColor, want)
		}
	}

	w.toggleCards()
	if h := w.cards.list.Objects[0]; fyne.AccessibleLabel(h) != "Section ACT ONE, blue" {
		t.Errorf("section header: %q", fyne.AccessibleLabel(h))
	}
	grid := w.cards.list.Objects[1].(*fyne.Container)
	for i, want := range []string{"INT. BARN - DAY, red", "EXT. FIELD - DAY", "MONTAGE, #00ff00"} {
		if got := fyne.AccessibleLabel(grid.Objects[i]); got != want {
			t.Errorf("card %d: %q, want %q", i, got, want)
		}
	}
}

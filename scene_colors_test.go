// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

func TestWithColor(t *testing.T) {
	for _, tc := range []struct{ line, color, want string }{
		{"INT. BARN - DAY", "red", "INT. BARN - DAY [[red]]"},
		{"INT. BARN - DAY  ", "Blue", "INT. BARN - DAY [[blue]]"},
		{"INT. BARN - DAY #12#", "red", "INT. BARN - DAY #12# [[red]]"},
		{"INT. BARN - DAY #12# [[red]]", "green", "INT. BARN - DAY #12# [[green]]"},
		{"INT. BARN - DAY [[COLOR BLUE]] #12#", "teal", "INT. BARN - DAY #12# [[teal]]"},
		{"INT. BARN - DAY [[#ff8800]] [[storyline A]] #3A#", "pink", "INT. BARN - DAY [[storyline A]] #3A# [[pink]]"},
		{"INT. BARN - DAY [[a note]] [[red]]", "none", "INT. BARN - DAY [[a note]]"},
		{"INT. BARN - DAY #12# [[red]]", "", "INT. BARN - DAY #12#"},
		{"INT. BARN - DAY [[red]] [[blue]]", "gray", "INT. BARN - DAY [[gray]]"},
		{"INT. BARN - DAY [[color]]", "red", "INT. BARN - DAY [[color]] [[red]]"}, // "color" alone is no color
		{"[[red]] .FLASHBACK", "none", ".FLASHBACK"},
		{"# ACT ONE [[red]]", "brown", "# ACT ONE [[brown]]"},
		{"## The garden [[note]]", "yellow", "## The garden [[note]] [[yellow]]"},
		{"INT. BARN - DAY [[open", "red", "INT. BARN - DAY [[red]] [[open"},
		{"INT. BARN - DAY [[blue]] [[open", "none", "INT. BARN - DAY [[open"},
		{"INT. BARN - DAY", "none", "INT. BARN - DAY"},
	} {
		got := withColor(tc.line, tc.color)
		if got != tc.want {
			t.Errorf("withColor(%q, %q) = %q, want %q", tc.line, tc.color, got, tc.want)
		}
		if want := tc.color; want != "" && want != "none" && !lexColorIs(got, want) {
			t.Errorf("%q: lex.NoteColor is %q", got, lex.NoteColor(got))
		}
	}
}

func lexColorIs(line, c string) bool { return lex.NoteColor(line) == strings.ToLower(c) }

func TestSetItemColor(t *testing.T) {
	text := "# ACT ONE\n\nINT. BARN - DAY #1# [[red]]\n= Anna finds the tin.\n\nRain.\n"
	got := setItemColor(text, nil, 1, "blue")
	if want := "# ACT ONE\n\nINT. BARN - DAY #1# [[blue]]\n= Anna finds the tin.\n\nRain.\n"; got != want {
		t.Errorf("scene: %q", got)
	}
	got = setItemColor(got, nil, 0, "orange")
	if want := "# ACT ONE [[orange]]\n\nINT. BARN - DAY #1# [[blue]]\n= Anna finds the tin.\n\nRain.\n"; got != want {
		t.Errorf("section: %q", got)
	}
	if got := setItemColor(text, nil, 5, "red"); got != text {
		t.Errorf("no such item: %q", got)
	}
}

func TestNoteRGB(t *testing.T) {
	for name, ok := range map[string]bool{
		"red": true, "grey": true, "brown": true, "#ff8800": true, "#FF8800": true,
		"#zzzzzz": false, "#ff88": false, "chartreuse": false, "goldenrod": true, "": false, "#+12345": false,
	} {
		if _, got := noteRGB(name); got != ok {
			t.Errorf("noteRGB(%q): %v", name, got)
		}
	}
	if c, _ := noteRGB("#ff8800"); c.R != 0xff || c.G != 0x88 || c.B != 0 {
		t.Errorf("#ff8800 is %v", c)
	}
	for _, n := range beatColorNames {
		if _, ok := noteRGB(n); !ok {
			t.Errorf("no RGB for the picker's %q", n)
		}
	}
}

// shownColorMenu is the color picker shown over the window (it has the
// focus, for the keyboard), nil if none.
func shownColorMenu(w *MainWindow) *widget.PopUpMenu {
	m, _ := w.fyneWindow.Canvas().Focused().(*widget.PopUpMenu)
	return m
}

// pickShown picks the color with the label in the picker shown.
func pickShown(t *testing.T, w *MainWindow, label string) {
	t.Helper()
	m := shownColorMenu(w)
	if m == nil {
		t.Fatal("no color picker shown")
	}
	for _, o := range m.Items {
		if tap, ok := o.(fyne.Tappable); ok && hasText(o, label) {
			tap.Tapped(&fyne.PointEvent{})
			return
		}
	}
	t.Fatalf("no %q in the picker", label)
}

// hasText reports whether o shows the text label.
func hasText(o fyne.CanvasObject, label string) bool {
	switch o := o.(type) {
	case *canvas.Text:
		return o.Text == label
	case fyne.Widget:
		for _, k := range test.WidgetRenderer(o).Objects() {
			if hasText(k, label) {
				return true
			}
		}
	case *fyne.Container:
		for _, k := range o.Objects {
			if hasText(k, label) {
				return true
			}
		}
	}
	return false
}

// The picker from the editor (Format > Scene or Section Color…), the
// outline and the cards writes the note as one undo step.
func TestColorPicker(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	script := "# ACT ONE\n\nINT. BARN - DAY #1# [[a note]]\n\nRain.\n\nEXT. FIELD - DAY [[blue]]\n\nWind.\n"
	w.textEditor.SetText(script)

	item := menuItemByLabel(t, w, "Scene or Section Color…")
	if item.Shortcut == nil || item.Shortcut.ShortcutName() != shortcut(fyne.KeyL, fyne.KeyModifierShift).ShortcutName() {
		t.Errorf("shortcut %v, want Ctrl+Shift+L", item.Shortcut)
	}
	w.fyneWindow.Canvas().Focus(w.textEditor)
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(4)) // in the barn
	item.Action()
	pickShown(t, w, "Red")
	want := strings.Replace(script, "[[a note]]", "[[a note]] [[red]]", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("picked red: %q", got)
	}
	if w.fyneWindow.Canvas().Focused() != fyne.Focusable(w.textEditor) {
		t.Error("the editor lost the focus")
	}
	w.undo()
	if got := w.textEditor.Text(); got != script {
		t.Fatalf("undone: %q", got)
	}
	w.redo()
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("redone: %q", got)
	}

	// the outline's right click; None takes the field's blue out
	w.togglePane("outline")
	row := w.outline.list.CreateItem().(*dragRow)
	w.outline.list.UpdateItem(2, row)
	row.Tapped(&fyne.PointEvent{}) // a tap still goes there
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 6 {
		t.Errorf("tapped the field's row: cursor on line %d", line)
	}
	row.TappedSecondary(&fyne.PointEvent{})
	capture(t, w.fyneWindow.Canvas(), "color-picker.png")
	m := w.colorMenu(w.outline.items[2], nil)
	if m.Items[0].Checked || !menuChecked(m, "Blue") {
		t.Errorf("the field's blue is not checked")
	}
	pickShown(t, w, "None")
	want = strings.Replace(want, " [[blue]]", "", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("outline, none: %q", got)
	}

	// the cards: right click on the section, and Ctrl+Shift+L while
	// writing a card's synopsis is for that card
	w.toggleCards()
	w.cards.list.Objects[0].(*sectionHeader).TappedSecondary(&fyne.PointEvent{})
	pickShown(t, w, "Teal")
	want = strings.Replace(want, "# ACT ONE", "# ACT ONE [[teal]]", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("section: %q", got)
	}
	capture(t, w.fyneWindow.Canvas(), "color-cards.png")
	field := w.cards.list.Objects[1].(*fyne.Container).Objects[1].(*card)
	w.fyneWindow.Canvas().Focus(synopsisOf(t, field))
	item.Action()
	pickShown(t, w, "Brown")
	want = strings.Replace(want, "EXT. FIELD - DAY", "EXT. FIELD - DAY [[brown]]", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("card: %q", got)
	}
	if got := fyne.AccessibleLabel(w.cards.list.Objects[1].(*fyne.Container).Objects[1]); got != "EXT. FIELD - DAY, brown" {
		t.Errorf("the card after picking: %q", got)
	}
}

func menuChecked(m *fyne.Menu, label string) bool {
	for _, it := range m.Items {
		if it.Label == label {
			return it.Checked
		}
	}
	return false
}

// Before the first heading there is nothing to color, nor in the cards
// without a card to go by.
func TestColorPickerNothingHere(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("Title: X\n\nINT. BARN - DAY\n")
	w.textEditor.SetCursorOffset(0)
	w.pickColorHere(nil)
	if shownColorMenu(w) != nil {
		t.Error("a picker without a scene")
	}
	if got := w.statusLabel.Text; !strings.Contains(got, "No scene or section") {
		t.Errorf("status %q", got)
	}
	// the cards, no synopsis being written: not the hidden editor's scene
	w.textEditor.SetCursorOffset(len("Title: X\n\nINT"))
	w.toggleCards()
	w.fyneWindow.Canvas().Unfocus()
	w.pickColorHere(nil)
	if shownColorMenu(w) != nil {
		t.Error("a picker for the hidden editor's scene")
	}
}

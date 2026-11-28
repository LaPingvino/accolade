// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/layout"
)

func TestNotesBlock(t *testing.T) {
	script := "INT. A - DAY\n\nHi.\n"
	with := strings.TrimRight(script, "\n") + notesText("Call Bram.\n# not a section")
	if notesOf(with) != "Call Bram.\n# not a section" {
		t.Errorf("notes %q", notesOf(with))
	}
	if notesStart(with) != len([]rune("INT. A - DAY\n\nHi.")) {
		t.Errorf("start %d", notesStart(with))
	}
	if notesText("a */ b") != "\n\n/* Notes\na * / b\n*/\n" {
		t.Errorf("*/ in notes: %q", notesText("a */ b"))
	}
	if notesText("  \n") != "" || notesOf(script) != "" {
		t.Error("empty notes")
	}
	if items := outlineOf(with, nil); len(items) != 1 {
		t.Errorf("outline reads the notes: %+v", items)
	}
}

// The Notes pane: the script's items in its order; one edited there is
// edited in the text, one undo step; a new unsorted note goes into the
// notes block at the end; Restore uncomments a cut.
func TestNotesPanel(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. A - DAY\n\nHi. [[todo: call Bram]]\n\n/*\nINT. B - NIGHT\n\nGone.\n*/\n")
	w.togglePane("notes")
	p := w.notesPane()
	if len(p.all) != 2 || p.all[0].Kind != openTodo || p.all[1].Kind != openCut {
		t.Fatalf("items: %+v", p.all)
	}
	if len(p.rows) != 3 || p.rows[0].head != "INT. A - DAY" {
		t.Errorf("rows: %+v", p.rows)
	}
	p.list.Select(1) // the to-do: the cursor goes there, its text to edit
	if w.textEditor.CursorOffset() != p.all[0].Start || !p.detail.Visible() || p.entry.Text != "call Bram" {
		t.Fatalf("picked: cursor %d, editing %q", w.textEditor.CursorOffset(), p.entry.Text)
	}
	before := w.textEditor.Text()
	p.entry.SetText("call Bram and Anna")
	p.commit()
	if got := w.textEditor.Text(); !strings.Contains(got, "Hi. [[todo: call Bram and Anna]]") {
		t.Fatalf("edited: %q", got)
	}
	w.undo()
	if w.textEditor.Text() != before {
		t.Errorf("one undo step: %q", w.textEditor.Text())
	}
	// Restore on the cut
	p.read()
	p.pick(1, true)
	if !p.restore.Visible() {
		t.Fatal("no Restore on a cut")
	}
	p.restoreItem()
	if got := w.textEditor.Text(); !strings.Contains(got, "\n\nINT. B - NIGHT\n\nGone.\n") || strings.Contains(got, "/*") {
		t.Errorf("restored: %q", got)
	}
	// a new unsorted note, at the end
	p.newUnsorted()
	p.entry.SetText("Call the lake's owner.")
	p.commit()
	if got := w.textEditor.Text(); !strings.HasSuffix(got, "\n\n/* Notes\nCall the lake's owner.\n*/\n") {
		t.Errorf("unsorted: %q", got)
	}
	if last := p.all[len(p.all)-1]; last.Kind != openUnsorted || last.Text != "Call the lake's owner." {
		t.Errorf("unsorted listed: %+v", last)
	}
	// the filters
	p.toggle(0)
	for _, r := range p.rows {
		if r.item >= 0 && p.all[r.item].Kind == openTodo {
			t.Error("a to-do shown with To do off")
		}
	}
	// the exports do not print the unsorted notes
	for _, l := range layout.Lay(exportJob{}.script(w.textEditor.Text()), nil) {
		if strings.Contains(l.Text(), "lake's owner") {
			t.Error("notes printed")
		}
	}
}

// Add Note writes an empty note at the cursor, the cursor in it.
func TestAddNoteAtCursor(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. A - DAY\n\nHi.\n")
	w.textEditor.SetCursorOffset(len("INT. A - DAY\n\nHi."))
	w.addNoteAtCursor()
	if got := w.textEditor.Text(); got != "INT. A - DAY\n\nHi. [[]]\n" || w.textEditor.CursorOffset() != len("INT. A - DAY\n\nHi. [[") {
		t.Errorf("note added: %q, cursor %d", got, w.textEditor.CursorOffset())
	}
}

// A boneyard called Notes in the middle of the script is the script's:
// the notes are the block at the very end.
func TestNotesOnlyAtTheEnd(t *testing.T) {
	script := "INT. A - DAY\n\n/* Notes\nold idea\n*/\n\nINT. B - DAY\n\nAction.\n\n/* Notes\nCall Bram.\n*/\n"
	if got := notesOf(script); got != "Call Bram." {
		t.Errorf("notes %q", got)
	}
	if start := notesStart(script); !strings.HasSuffix(string([]rune(script)[:start]), "Action.") {
		t.Errorf("notes start at %d", start)
	}
	only := "INT. A - DAY\n\n/* Notes\nold idea\n*/\n\nINT. B - DAY\n"
	if notesOf(only) != "" || notesStart(only) != len([]rune(only)) {
		t.Error("a boneyard mid-script taken for the notes")
	}
}

// F6 / Shift+F6 go round the panes that are shown.
func TestCyclePanes(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	c := w.fyneWindow.Canvas()
	c.Focus(w.textEditor)
	w.cyclePane(1)
	if c.Focused() != w.textEditor {
		t.Errorf("one pane: focus on %T", c.Focused())
	}
	w.togglePane("notes")
	w.showFind()
	t.Cleanup(w.hideFindReplace)
	c.Focus(w.textEditor)
	w.cyclePane(1)
	if c.Focused() != w.notesPane().focusTarget() {
		t.Errorf("F6 from the editor: %T", c.Focused())
	}
	w.cyclePane(1)
	if c.Focused() != w.searchBar.searchEntry {
		t.Errorf("F6 again: %T", c.Focused())
	}
	w.cyclePane(1)
	if c.Focused() != w.textEditor {
		t.Errorf("F6 round: %T", c.Focused())
	}
	w.cyclePane(-1)
	if c.Focused() != w.searchBar.searchEntry {
		t.Errorf("Shift+F6: %T", c.Focused())
	}
}

// Tab goes through the window in the order it is seen: the toolbar, the
// side panel, then the editor (which keeps Tab).
func TestTabOrderFollowsTheScreen(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. A - DAY\n\nHi. [[a note]]\n")
	w.togglePane("notes")
	c := w.fyneWindow.Canvas()
	c.Unfocus()
	var order []string
	for i := 0; i < 48; i++ { // the toolbar's buttons, the panes' tabs, then the notes
		c.FocusNext()
		switch f := c.Focused(); f {
		case w.textEditor:
			order = append(order, "editor")
		case w.notesPane().list:
			order = append(order, "notes")
		default:
			order = append(order, "toolbar")
		}
	}
	got := strings.Join(order, " ")
	if rest := strings.TrimLeft(strings.ReplaceAll(got, "toolbar ", "t"), "t"); !strings.HasPrefix(got, "toolbar") || !strings.HasPrefix(rest, "notes editor") {
		t.Errorf("Tab order: %s", got)
	}
}

// The calls to action: the status bar counts the to-dos and the beats
// without a scene and opens the pane; the margin marks the note; the
// outline's section says how many beats it has without a scene, and a
// tap steps through them.
func TestOpenCallsToAction(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	text := "# Act One\n\n= Its own.\n\n= Beat one.\n\n= Beat two.\n\nINT. A - DAY\n\nHi. [[todo: call Bram]]\n"
	w.textEditor.SetText(text)
	w.updateOpen()
	if !w.openLabel.Visible() || w.openLabel.Text != "3 open" {
		t.Fatalf("the count: %q shown %v", w.openLabel.Text, w.openLabel.Visible())
	}
	if c, ok := w.marksShown()[10]; !ok || c != marginColor(openTodo) {
		t.Errorf("the margin: %v", w.marksShown())
	}
	w.openLabel.Tapped(nil)
	if !w.paneShown("notes") {
		t.Error("the count did not open the Notes pane")
	}
	w.togglePane("outline")
	if got := w.outline.beats[0]; len(got) != 2 || got[0] != 4 || got[1] != 6 {
		t.Fatalf("the outline's beats: %v", w.outline.beats)
	}
	w.textEditor.SetCursorOffset(0)
	w.stepToBeat(w.outline.beats[0])
	w.stepToBeat(w.outline.beats[0])
	if line := strings.Count(string([]rune(w.textEditor.Text())[:w.textEditor.CursorOffset()]), "\n"); line != 6 {
		t.Errorf("stepped to line %d", line)
	}
	w.textEditor.SetText("INT. A - DAY\n\nHi.\n")
	w.updateOpen()
	if w.openLabel.Visible() || len(w.marksShown()) != 0 {
		t.Error("nothing open, and still a count or a mark")
	}
}

// TestNotesPaneScreenshots renders the Notes pane (SHOT_DIR as in
// screenshot_test.go): beside the script with an item picked, the
// outline's and the cards' beats marks, and on a phone.
func TestNotesPaneScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := newDialogTestWindow(t)
	keepDockSettings(t, w)
	w.textEditor.SetText(`Title: The Long Rain

# Act One

= Anna waits for a letter that never comes.

= The postman skips her house.

= Bram brings the letter himself.

INT. KITCHEN - DAY

Anna stands at the window. [[todo: why does she not go out?]]

[[Is the rain too much here?]]

/* INT. GARDEN - NIGHT

ANNA
Goodbye, then.
*/

= She burns the envelope.

# Act Two

INT. ROAD - DAY

Bram drives through the rain.

/* maybe the car breaks down here */

/* Notes
Ask Tom about the ending.

The lake scene could open the film.
*/
`)
	snap := func(name string) {
		t.Helper()
		w.fyneWindow.Content().Refresh()
		capture(t, w.fyneWindow.Canvas(), "notes-"+name+".png")
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	w.updateOpen()
	w.togglePane("notes")
	snap("1-pane")
	p := w.notesPane()
	p.pick(3, true) // the to-do
	snap("2-picked")
	w.togglePane("notes")
	w.togglePane("outline")
	snap("3-outline-marks")
	w.togglePane("notes")
	w.toggleCards()
	snap("4-cards-marks")
	w.showMiddle("")
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.togglePane("outline")
	w.showPane("notes", false)
	snap("5-phone")
	p.cancel()
	w.showPane("notes", false)
	snap("6-phone-list")
}

// Two edits in a row of one of two notes alike: the second goes to the
// same note, not its twin, and the script's cursor stays.
func TestNotesPaneEditTwice(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText("INT. A - DAY\n\nOne. [[todo: fix]]\n\nTwo. [[todo: fix]]\n")
	w.togglePane("notes")
	p := w.notesPane()
	p.pick(1, false)
	w.textEditor.SetCursorOffset(len("INT. A - DAY\n\nOn"))
	p.entry.SetText("fix two")
	p.commit()
	p.entry.SetText("fix it twice")
	p.commit()
	if got := w.textEditor.Text(); got != "INT. A - DAY\n\nOne. [[todo: fix]]\n\nTwo. [[todo: fix it twice]]\n" {
		t.Errorf("edited twice: %q", got)
	}
	if c := w.textEditor.CursorOffset(); c != len("INT. A - DAY\n\nOn") {
		t.Errorf("the cursor moved to %d", c)
	}
}

// A beat on the script's last line, or right before the notes block, is
// in the pane and on its section's mark alike.
func TestBeatAtTheEnd(t *testing.T) {
	for _, text := range []string{"# One\n\n= Own.\n\n= Last beat", "# One\n\n= Own.\n\n= Last beat\n\n/* Notes\nx\n*/\n"} {
		n := 0
		for _, it := range openItemsOf(text, nil, false, false) {
			if it.Kind == openBeat {
				n++
			}
		}
		marks := sectionBeats(text, nil, outlineOf(text, nil), false)
		if n != 1 || len(marks[0]) != 1 {
			t.Errorf("%q: %d beats in the pane, %v on the mark", text, n, marks)
		}
	}
}

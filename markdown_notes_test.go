// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

const footnotesText = "A claim.[^1] Another.[^x] In `code [^9]` no mark.\n\n```\n[^8]\n```\n\nThe first again.[^1]\n\n" +
	"[^x]: The named note,\n    on two lines.\n[^1]: The *first* note.\n"

// The marks are the text's, not code's; a note's start is at a line's
// start.
func TestNoteMarks(t *testing.T) {
	var got []string
	for _, m := range mdNoteMarks(footnotesText) {
		s := string([]rune(footnotesText)[m.start:m.end])
		if m.def {
			s += " (note)"
		}
		got = append(got, s)
	}
	if strings.Join(got, ", ") != "[^1], [^x], [^1], [^x]: (note), [^1]: (note)" {
		t.Errorf("marks %v", got)
	}
	if s := noteText(footnotesText, "x"); s != "The named note, on two lines." {
		t.Errorf("x's text %q", s)
	}
	if s := noteText(footnotesText, "1"); s != "The first note." {
		t.Errorf("1's text %q", s)
	}
	at := strings.Index(footnotesText, "[^1]") + 2
	if tip := noteTip(footnotesText, at); tip != "1. The first note." {
		t.Errorf("the tip %q", tip)
	}
	if tip := noteTip(footnotesText, strings.Index(footnotesText, "claim")); tip != "" {
		t.Errorf("a tip off the marks %q", tip)
	}
}

// Go to Note or Reference: from a mark to its note's text, from the
// note back to the first mark; without a mark, it says so.
func TestNoteJump(t *testing.T) {
	w := mdWindow(t, footnotesText, strings.Index(footnotesText, ".[^x]"), strings.Index(footnotesText, ".[^x]"))
	runMD(w, "md.noteJump") // the line's nearest mark: [^x]
	if off := w.textEditor.CursorOffset(); !strings.HasPrefix(string([]rune(footnotesText)[off:]), "The named note") {
		t.Errorf("at %q", string([]rune(footnotesText)[off:])[:10])
	}
	runMD(w, "md.noteJump")
	if off := w.textEditor.CursorOffset(); off != strings.Index(footnotesText, "[^x]")+4 {
		t.Errorf("back at %d", off)
	}
	w.textEditor.SetCursorOffset(strings.Index(footnotesText, "The first again"))
	runMD(w, "md.noteJump")
	if off := w.textEditor.CursorOffset(); !strings.HasPrefix(string([]rune(footnotesText)[off:]), "The *first*") {
		t.Errorf("at %q", string([]rune(footnotesText)[off:]))
	}
	runMD(w, "md.noteJump") // back to the first mark, not the nearest
	if off := w.textEditor.CursorOffset(); off != strings.Index(footnotesText, "[^1]")+4 {
		t.Errorf("back at %d", off)
	}
	w.textEditor.SetCursorOffset(strings.Index(footnotesText, "```"))
	runMD(w, "md.noteJump")
	if !strings.Contains(w.statusText(), "No note mark here") {
		t.Errorf("status %q", w.statusText())
	}
	if w.textEditor.Text() != footnotesText {
		t.Error("a jump changed the text")
	}
}

// Renumber Notes: in the order the text refers to them, the notes put
// in that order; one undo step back.
func TestRenumberNotes(t *testing.T) {
	src := "One.[^3] Two.[^b] Three.[^3]\n\n`[^3]` stays.\n\n[^b]: Bee.\n\n[^3]: Three's,\n    long.\n\n[^z]: Nobody's.\n"
	b := buffer.New(src)
	said := renumberNotesIn(b)
	want := "One.[^1] Two.[^2] Three.[^1]\n\n`[^3]` stays.\n\n[^1]: Three's,\n    long.\n\n[^2]: Bee.\n\n[^3]: Nobody's.\n"
	if b.Text() != want || said != "Notes renumbered 1 to 3." {
		t.Errorf("%q\n%s\nwant\n%s", said, b.Text(), want)
	}
	if renumberNotesIn(b) != "The notes are in order." || b.Text() != want {
		t.Error("again: changed")
	}
	b.Undo()
	if b.Text() != src {
		t.Errorf("undone:\n%s", b.Text())
	}
	// notes with text between them: numbered, left where they are
	src = "A.[^2] B.[^1]\n\n[^1]: One.\n\nA paragraph.\n\n[^2]: Two.\n"
	b = buffer.New(src)
	renumberNotesIn(b)
	if want := "A.[^1] B.[^2]\n\n[^2]: One.\n\nA paragraph.\n\n[^1]: Two.\n"; b.Text() != want {
		t.Errorf("%s\nwant\n%s", b.Text(), want)
	}
	w := mdWindow(t, "X.[^5]\n\n[^5]: Five.\n", 0, 0)
	runMD(w, "md.renumberNotes")
	if w.textEditor.Text() != "X.[^1]\n\n[^1]: Five.\n" || !strings.Contains(w.statusText(), "renumbered") {
		t.Errorf("the command: %q, %q", w.textEditor.Text(), w.statusText())
	}
}

// The pointer over a mark in the editor shows its note's text, off it
// the tip goes.
func TestNoteHoverTip(t *testing.T) {
	w := mdWindow(t, footnotesText, 0, 0)
	w.fyneWindow.Resize(fyne.NewSize(1200, 800))
	e := w.textEditor
	off := strings.Index(footnotesText, "[^1]") + 1
	e.SetCursorOffset(off)
	at := e.CursorPosition().AddXY(2, 2) // in the cell of the mark's ^
	e.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at}})
	if tip := w.tooltips.Text(); tip != "1. The first note." {
		t.Errorf("the tip %q", tip)
	}
	capture(t, w.fyneWindow.Canvas(), "notes-hover-desktop.png")
	e.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(at.X, at.Y+60)}})
	if tip := w.tooltips.Text(); tip != "" {
		t.Errorf("off the mark, the tip %q", tip)
	}
	e.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at}})
	e.MouseOut()
	if w.tooltips.Text() != "" {
		t.Error("the tip stayed when the pointer left")
	}
}

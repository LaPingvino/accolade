package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func newTestWindow(t *testing.T, text string) *MainWindow {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)

	w := NewMainWindow(&Application{fyneApp: a})
	w.textEditor.SetText(text)
	w.hasChanges = false
	setCursorOffset(&w.textEditor.Entry, 0)
	return w
}

func search(w *MainWindow, text string) {
	w.searchBar.Show()
	w.searchBar.SetSearchText(text)
}

func TestSetCursorOffsetRoundTrips(t *testing.T) {
	w := newTestWindow(t, "first line\n\nthird ñandú line\nlast")
	for _, off := range []int{0, 3, 10, 11, 12, 20, 29, 33} {
		setCursorOffset(&w.textEditor.Entry, off)
		if got := w.textEditor.CursorTextOffset(); got != off {
			t.Errorf("setCursorOffset(%d): cursor at %d", off, got)
		}
	}
}

func TestFindNextSelectsAndWraps(t *testing.T) {
	w := newTestWindow(t, "JOHN enters.\nMARY waves at john.\nJohn leaves.")
	search(w, "john")

	if w.searchBar.totalMatches != 3 {
		t.Fatalf("totalMatches = %d, want 3", w.searchBar.totalMatches)
	}
	want := []string{"JOHN", "john", "John", "JOHN"} // last one wraps
	for i, sel := range want {
		w.findNext()
		if got := w.textEditor.SelectedText(); got != sel {
			t.Fatalf("findNext #%d selected %q, want %q", i+1, got, sel)
		}
	}
	if got := w.searchBar.statusLabel.Text; got != "1 of 3" {
		t.Errorf("status = %q, want %q", got, "1 of 3")
	}
}

func TestFindPreviousWrapsToEnd(t *testing.T) {
	w := newTestWindow(t, "one two one two one")
	search(w, "one")
	w.findPrevious()
	if got := w.searchBar.statusLabel.Text; got != "3 of 3" {
		t.Fatalf("status = %q, want 3 of 3", got)
	}
	w.findPrevious()
	if got := w.searchBar.statusLabel.Text; got != "2 of 3" {
		t.Errorf("status = %q, want 2 of 3", got)
	}
}

func TestSearchOptions(t *testing.T) {
	w := newTestWindow(t, "cat Cat concat cat_x cat.")
	search(w, "cat")
	if n := w.searchBar.totalMatches; n != 5 {
		t.Errorf("plain: %d matches, want 5", n)
	}
	w.searchBar.SetCaseSensitive(true)
	if n := w.searchBar.totalMatches; n != 4 {
		t.Errorf("case sensitive: %d matches, want 4", n)
	}
	w.searchBar.SetWholeWords(true)
	if n := w.searchBar.totalMatches; n != 2 {
		t.Errorf("case sensitive + whole words: %d matches, want 2", n)
	}
	w.searchBar.SetCaseSensitive(false)
	w.searchBar.SetWholeWords(false)
	w.searchBar.SetRegularExpression(true)
	w.searchBar.SetSearchText(`c[a-z]+t`)
	if n := w.searchBar.totalMatches; n != 5 {
		t.Errorf("regex: %d matches, want 5", n)
	}
	w.searchBar.SetSearchText(`(`)
	if got := w.searchBar.statusLabel.Text; got != "Invalid regular expression" {
		t.Errorf("bad regex status = %q", got)
	}
}

func TestReplaceOneKeepsUndo(t *testing.T) {
	w := newTestWindow(t, "Hello world, hello moon")
	search(w, "hello")
	w.searchBar.SetReplaceText("Bye")

	w.findNext()
	w.replaceOne()
	if got := w.textEditor.Text; got != "Bye world, hello moon" {
		t.Fatalf("after replaceOne: %q", got)
	}
	if got := w.textEditor.SelectedText(); got != "hello" {
		t.Errorf("replaceOne should move on to the next match, selected %q", got)
	}
	if !w.hasChanges {
		t.Error("replaceOne did not mark the document changed")
	}

	// Fyne records typing over a selection as two steps (erase, insert),
	// the same as when a user does it by hand.
	w.textEditor.Undo()
	w.textEditor.Undo()
	if got := w.textEditor.Text; got != "Hello world, hello moon" {
		t.Errorf("after Undo: %q", got)
	}
}

func TestReplaceOneWithoutSelectionOnlyFinds(t *testing.T) {
	w := newTestWindow(t, "aaa bbb aaa")
	search(w, "aaa")
	w.searchBar.SetReplaceText("x")
	w.replaceOne()
	if got := w.textEditor.Text; got != "aaa bbb aaa" {
		t.Errorf("text changed without a selected match: %q", got)
	}
	if got := w.textEditor.SelectedText(); got != "aaa" {
		t.Errorf("selected %q, want first match", got)
	}
}

func TestReplaceAll(t *testing.T) {
	w := newTestWindow(t, "INT. HOUSE - DAY\nint. barn - night\nEXT. FIELD")
	search(w, "int.")
	w.searchBar.SetReplaceText("EXT.")
	w.replaceAll()
	if got, want := w.textEditor.Text, "EXT. HOUSE - DAY\nEXT. barn - night\nEXT. FIELD"; got != want {
		t.Errorf("replaceAll = %q, want %q", got, want)
	}
	if got := w.searchBar.statusLabel.Text; got != "Replaced 2" {
		t.Errorf("status = %q", got)
	}
}

func TestReplaceAllRegexGroups(t *testing.T) {
	w := newTestWindow(t, "Smith, John; Doe, Jane")
	w.searchBar.SetRegularExpression(true)
	search(w, `(\w+), (\w+)`)
	w.searchBar.SetReplaceText("$2 $1")
	w.replaceAll()
	if got, want := w.textEditor.Text, "John Smith; Jane Doe"; got != want {
		t.Errorf("replaceAll = %q, want %q", got, want)
	}
}

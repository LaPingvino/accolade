// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// shownRows are the editor's rows as drawn, as text.
func shownRows(w *MainWindow) []string {
	var out []string
	for i := 0; ; i++ {
		r := w.textEditor.GridRow(i)
		if len(r.Cells) == 0 && i > 0 && len(w.textEditor.GridRow(i+1).Cells) == 0 {
			return out
		}
		var b strings.Builder
		for _, c := range r.Cells {
			if c.Rune != 0 {
				b.WriteRune(c.Rune)
			}
		}
		out = append(out, b.String())
	}
}

// rowWith is the first drawn row holding s ("" none).
func rowWith(w *MainWindow, s string) string {
	for _, r := range shownRows(w) {
		if strings.Contains(r, s) {
			return r
		}
	}
	return ""
}

// The sample's blocks fold by default to a line each, open while the
// cursor is in them, stay open when opened by hand; the text never
// changes, and a screen reader reads all of it.
func TestBlockFoldsSample(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	e := w.textEditor
	text := e.Text()
	f := w.scriptFolds
	e.SetCursorOffset(0)
	e.Refresh()
	keys := []string{}
	for _, b := range f.blocksOf() {
		keys = append(keys, b.key)
	}
	if got := strings.Join(keys, " "); got != "omit:/* Omitted 3@0 tail:Characters tail:Adaptation tail:Production tail:Notes" {
		t.Fatalf("blocks %q", got)
	}
	if n := len(f.ranges()); n != 5 {
		t.Fatalf("%d folded, not 5", n)
	}
	for _, want := range []string{
		"▸ Omitted scene · INT. TAVERN - DAY #3#, 2 lines",
		"▸ Characters · 5, 5 with notes",
		"▸ Adaptation · 3 other names · 2 not characters · 1 place", // then 2 cuts, cut at the width
		"▸ Production · scene numbers locked · pages locked · Blue",
		"▸ Notes · 3 paragraphs · This is Accolade's sample script",
	} {
		if rowWith(w, want) == "" {
			t.Errorf("no row %q in\n%s", want, strings.Join(shownRows(w), "\n"))
		}
	}
	if rowWith(w, "scene numbers: locked") != "" || rowWith(w, "OTCHUMYELOV: The police") != "" {
		t.Error("a folded block's lines are drawn")
	}
	// the cursor in: open; out again: folded
	prod := strings.Index(text, "scene numbers: locked")
	e.SetCursorOffset(prod)
	if rowWith(w, "scene numbers: locked") == "" || rowWith(w, "▸ Production") != "" {
		t.Error("the cursor in the production block did not open it")
	}
	if rowWith(w, "▸ Notes") == "" {
		t.Error("the notes opened too")
	}
	e.SetCursorOffset(0)
	if rowWith(w, "▸ Production") == "" {
		t.Error("the production block did not fold again when the cursor left")
	}
	// by hand: its mark in the gutter; it stays open with the cursor away
	pb := f.blockAt(strings.Count(text[:prod], "\n"))
	f.foldLine(pb.first)
	if rowWith(w, "scene numbers: locked") == "" {
		t.Error("the mark did not open it")
	}
	e.SetCursorOffset(10)
	if rowWith(w, "scene numbers: locked") == "" {
		t.Error("opened by hand, it folded when the cursor moved")
	}
	// Fold or Unfold Here from inside it: folded, the cursor above it
	e.SetCursorOffset(prod)
	cmd("view.foldHere").run(w)
	if rowWith(w, "▸ Production") == "" {
		t.Error("Fold Here inside did not fold it")
	}
	if line, _ := e.Buffer().Position(e.CursorOffset()); line != pb.first-1 {
		t.Errorf("the cursor on line %d, not above the block (%d)", line, pb.first-1)
	}
	// Unfold All and Fold All take the blocks along
	w.foldAll(true)
	if n := len(f.ranges()); n != 0 {
		t.Errorf("Unfold All left %d", n)
	}
	w.foldAll(false)
	if n := len(f.blockRanges()); n != 5 {
		t.Errorf("Fold All folded %d blocks", n)
	}
	// Find opens what it finds in a folded block
	w.showFind()
	w.searchBar.SetSearchText("Gutenberg")
	w.findNext()
	if rowWith(w, "Gutenberg") == "" {
		t.Error("Find did not open the notes it found the word in")
	}
	w.hideFindReplace()
	if e.Text() != text || e.AccessibilityText() != text {
		t.Error("folding changed the text, or what a screen reader reads")
	}
}

// Shots of the sample's end: the blocks folded (SHOT_DIR as in
// screenshot_test.go).
func TestBlockFoldScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w := sampleWindow(t, sampleAccolade)
	e := w.textEditor
	e.SetCursorOffset(strings.Index(e.Text(), "Yes. [[from"))
	toBottom := func() {
		if s, ok := w.editorView.Content.(*container.Scroll); ok {
			s.ScrollToBottom()
			if s.OnScrolled != nil {
				s.OnScrolled(s.Offset)
			}
		}
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w.fyneWindow.Resize(size.size)
		w.fyneWindow.Content().Refresh()
		toBottom()
		shotTo(t, w, "folds-end-"+size.name)
		e.SetCursorOffset(strings.Index(e.Text(), "HRYUKIN\nDon't let go!"))
		shotTo(t, w, "folds-omitted-"+size.name)
		e.SetCursorOffset(strings.Index(e.Text(), "Noon heat"))
		shotTo(t, w, "chips-top-"+size.name)
		e.SetCursorOffset(strings.Index(e.Text(), "#:~:text=THE,near") + 10)
		shotTo(t, w, "chips-raw-"+size.name)
		e.SetCursorOffset(strings.Index(e.Text(), "Yes. [[from"))
	}
}

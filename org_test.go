package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

const orgScript = "# ONE\n\nINT. BARN - DAY\n\nRain.\n\nEXT. FIELD - DAY\n\nWind.\n\n# TWO\n\nINT. HALL - NIGHT\n\nDark.\n"

// lineOf is the line text starts on in the editor.
func lineOf(t *testing.T, w *MainWindow, text string) int {
	for i, l := range strings.Split(w.textEditor.Text(), "\n") {
		if l == text {
			return i
		}
	}
	t.Fatalf("no line %q", text)
	return -1
}

func cursorLine(w *MainWindow) int {
	line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	return line
}

// Tab at a heading's start cycles it as org-mode: folded, its headings,
// all; Shift+Tab cycles the script; a tap on a heading's mark folds it.
func TestOrgCycling(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	e := w.textEditor
	e.SetText(orgScript)
	folded := func(text string) bool {
		return w.editorFolds[foldKey(outlineOf(e.Text(), w.sceneStarts())[w.itemAt(lineOf(t, w, text))])]
	}

	e.SetCursorOffset(0) // # ONE
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if !folded("# ONE") {
		t.Fatal("Tab did not fold the section")
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if folded("# ONE") || !folded("INT. BARN - DAY") || !folded("EXT. FIELD - DAY") {
		t.Fatalf("second Tab: not its headings: %v", w.editorFolds)
	}
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if len(w.editorFolds) != 0 {
		t.Fatalf("third Tab: not all shown: %v", w.editorFolds)
	}
	if e.Text() != orgScript {
		t.Fatalf("Tab changed the text: %q", e.Text())
	}

	w.cycleAll()
	if !folded("# ONE") || !folded("# TWO") || folded("INT. HALL - NIGHT") {
		t.Fatalf("overview: %v", w.editorFolds)
	}
	w.cycleAll()
	if folded("# ONE") || !folded("INT. BARN - DAY") || !folded("INT. HALL - NIGHT") {
		t.Fatalf("contents: %v", w.editorFolds)
	}
	w.cycleAll()
	if len(w.editorFolds) != 0 {
		t.Fatalf("show all: %v", w.editorFolds)
	}

	// the gutter's fold marks
	w.textEditor.Refresh()
	if !w.headingLines()[lineOf(t, w, "INT. HALL - NIGHT")] {
		t.Fatal("the hall's heading has no mark")
	}
	w.foldLine(lineOf(t, w, "INT. HALL - NIGHT"))
	if !folded("INT. HALL - NIGHT") {
		t.Error("tapping the mark did not fold the hall")
	}
}

// itemAt is the index of the outline item on line.
func (w *MainWindow) itemAt(line int) int {
	for i, it := range outlineOf(w.textEditor.Text(), w.sceneStarts()) {
		if it.Line == line {
			return i
		}
	}
	return -1
}

// evil-org's keys in vim mode: gj gk gh gl move between headings, Alt+j and
// Alt+k move a scene past its neighbour, Alt+l and Alt+h change a section's
// level.
func TestEvilOrg(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	e := w.textEditor
	e.SetText(orgScript)
	e.SetCursorOffset(0)
	e.TypedKey(escape())
	keys := func(s string) {
		for _, r := range s {
			e.TypedRune(r)
		}
	}
	keys("gj")
	if cursorLine(w) != lineOf(t, w, "INT. BARN - DAY") {
		t.Fatalf("gj: line %d", cursorLine(w))
	}
	keys("gj")
	keys("gh")
	if cursorLine(w) != 0 {
		t.Errorf("gh: line %d, not the section", cursorLine(w))
	}
	keys("gl")
	if cursorLine(w) != lineOf(t, w, "INT. BARN - DAY") {
		t.Errorf("gl: line %d", cursorLine(w))
	}
	alt := func(k fyne.KeyName) {
		e.TypedShortcut(&desktop.CustomShortcut{KeyName: k, Modifier: fyne.KeyModifierAlt})
	}
	alt(fyne.KeyJ) // the barn past the field
	if want := "# ONE\n\nEXT. FIELD - DAY\n\nWind.\n\nINT. BARN - DAY\n\nRain.\n\n# TWO\n\nINT. HALL - NIGHT\n\nDark.\n"; e.Text() != want {
		t.Fatalf("Alt+j: %q", e.Text())
	}
	if cursorLine(w) != lineOf(t, w, "INT. BARN - DAY") {
		t.Errorf("Alt+j: the cursor stayed behind, line %d", cursorLine(w))
	}
	alt(fyne.KeyK)
	if e.Text() != orgScript {
		t.Fatalf("Alt+k: %q", e.Text())
	}
	keys("gk") // # ONE
	alt(fyne.KeyL)
	if !strings.HasPrefix(e.Text(), "## ONE\n") {
		t.Fatalf("Alt+l: %q", e.Text()[:10])
	}
	alt(fyne.KeyH)
	if e.Text() != orgScript {
		t.Fatalf("Alt+h: %q", e.Text()[:10])
	}
}

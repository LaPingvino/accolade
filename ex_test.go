package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Ex commands on a script's lines, as vim runs them.
func TestEx(t *testing.T) {
	script := []string{
		"INT. BARN - DAY", // 1
		"",
		"Rain falls on the barn.", // 3
		"",
		"ANNA", // 5
		"The barn leaks.",
		"",
		"EXT. FIELD - NIGHT", // 8
		"",
		"Wind.", // 10
	}
	for _, c := range []struct {
		cmd    string
		cur    int
		visual [2]int
		want   string // the lines joined with |, or the error
		line   int
	}{
		{"s/barn/stable/", 2, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the stable.||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.", 2},
		{"%s/barn/stable/gi", 0, [2]int{-1, -1}, "INT. stable - DAY||Rain falls on the stable.||ANNA|The stable leaks.||EXT. FIELD - NIGHT||Wind.", 5},
		{`%s/\(INT\|EXT\)\. \(\w\+\)/\2 (\1)/`, 0, [2]int{-1, -1}, "BARN (INT) - DAY||Rain falls on the barn.||ANNA|The barn leaks.||FIELD (EXT) - NIGHT||Wind.", 7},
		{`s/\<the\>/a/`, 2, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on a barn.||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.", 2},
		{`%s/\vbarn|wind/&!/g`, 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn!.||ANNA|The barn! leaks.||EXT. FIELD - NIGHT||Wind.", 5},
		{"g/^$/d", 0, [2]int{-1, -1}, "INT. BARN - DAY|Rain falls on the barn.|ANNA|The barn leaks.|EXT. FIELD - NIGHT|Wind.", 1},
		{"v/barn/d", 0, [2]int{-1, -1}, "Rain falls on the barn.|The barn leaks.", 0},
		{"g/^INT\\|^EXT/s/$/ #1#/", 0, [2]int{-1, -1}, "INT. BARN - DAY #1#||Rain falls on the barn.||ANNA|The barn leaks.||EXT. FIELD - NIGHT #1#||Wind.", 0},
		{"5,6d", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.|||EXT. FIELD - NIGHT||Wind.", 4},
		{"'<,'>d", 0, [2]int{2, 3}, "INT. BARN - DAY||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.", 2},
		{"/EXT/,$d", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.||ANNA|The barn leaks.|", 6},
		{"8,10m0", 0, [2]int{-1, -1}, "EXT. FIELD - NIGHT||Wind.|INT. BARN - DAY||Rain falls on the barn.||ANNA|The barn leaks.|", 2},
		{"1t$", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.|INT. BARN - DAY", 10},
		{"5,6j", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.||ANNA The barn leaks.||EXT. FIELD - NIGHT||Wind.", 4},
		{".+2", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.", 2},
		{"8", 0, [2]int{-1, -1}, "INT. BARN - DAY||Rain falls on the barn.||ANNA|The barn leaks.||EXT. FIELD - NIGHT||Wind.", 7},
		{"s/snow/rain/", 0, [2]int{-1, -1}, "pattern not found: snow", 0},
		{"sort", 0, [2]int{-1, -1}, "not an editor command: sort", 0},
		{"'<,'>d", 0, [2]int{-1, -1}, "no visual selection", 0},
	} {
		var st exState
		res, err := st.run(c.cmd, script, c.cur, c.visual)
		got := strings.Join(res.lines, "|")
		if err != nil {
			got = err.Error()
		}
		if got != c.want || (err == nil && res.line != c.line) {
			t.Errorf(":%s\n got %q (line %d)\nwant %q (line %d)", c.cmd, got, res.line, c.want, c.line)
		}
	}
}

// The last pattern is used again (:s//x/), and the write and quit commands
// say what to do.
func TestExState(t *testing.T) {
	var st exState
	lines := []string{"rain rain", "rain"}
	if _, err := st.run("/rain/", lines, 0, [2]int{-1, -1}); err != nil {
		t.Fatal(err)
	}
	res, err := st.run("%s//snow/g", lines, 0, [2]int{-1, -1})
	if err != nil || strings.Join(res.lines, "|") != "snow snow|snow" {
		t.Errorf("%q %v", res.lines, err)
	}
	for cmd, want := range map[string]exResult{
		"w": {write: true}, "w x.fountain": {write: true, file: "x.fountain"},
		"wq": {write: true, quit: true}, "x": {write: true, quit: true}, "q": {quit: true}, "q!": {quit: true, keep: true},
	} {
		res, err := st.run(cmd, lines, 0, [2]int{-1, -1})
		if err != nil || res.write != want.write || res.file != want.file || res.quit != want.quit || res.keep != want.keep {
			t.Errorf(":%s: %+v %v", cmd, res, err)
		}
	}
}

// In a window with vim keys: ":" opens the command line, a command changes
// the script as one undo step; from visual mode the range is '<,'>.
func TestExInWindow(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("editor-keys", "vim")
	t.Cleanup(func() { w.settings.SetString("editor-keys", "standard") })
	w.applyEditorSettings()
	w.textEditor.SetText("INT. BARN - DAY\n\nRain on the barn.\n")
	c := w.fyneWindow.Canvas()
	c.Focus(w.textEditor)
	w.textEditor.TypedRune(':')
	if c.Focused() != w.exEntry {
		t.Fatalf("':' focused %T", c.Focused())
	}
	test.Type(w.exEntry, "%s/barn/stable/gi")
	w.exEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if got := w.textEditor.Text(); got != "INT. stable - DAY\n\nRain on the stable.\n" {
		t.Fatalf("after :s %q", got)
	}
	if c.Focused() != w.textEditor || w.exBar.Visible() {
		t.Error("the command line did not close")
	}
	w.textEditor.TypedRune('u')
	if got := w.textEditor.Text(); got != "INT. BARN - DAY\n\nRain on the barn.\n" {
		t.Errorf("one undo: %q", got)
	}
	w.textEditor.SetCursorOffset(0)
	w.textEditor.TypedRune('V')
	w.textEditor.TypedRune('j')
	w.textEditor.TypedRune(':')
	if w.exEntry.Text != "'<,'>" {
		t.Errorf("from visual: %q", w.exEntry.Text)
	}
	test.Type(w.exEntry, "d")
	w.exEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if got := w.textEditor.Text(); got != "Rain on the barn.\n" {
		t.Errorf("after '<,'>d %q", got)
	}
}

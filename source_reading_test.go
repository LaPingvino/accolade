package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

const readingStory = `# Chapter 1

Anna stood in the kitchen and waited. She thought of the letter again.

“Who’s there?” Anna whispered.

“Only me,” said Tom. “Open the door.”

“Not tonight.”

“Please, Anna.”

* * *

The next morning the street was white with frost.
`

// The source pane's view of a picked passage: who says what, the scene's
// heading, the flags, read through the same passages the pane shows.
func TestWhatTheSourceShows(t *testing.T) {
	w, _ := sourceWindow(t, "Title: The Door\n\nINT. KITCHEN - NIGHT\n\nAnna waits.\n", readingStory)
	s := w.sourcePane()
	if s.view.box.Visible() {
		t.Error("the view is shown before a passage is picked")
	}
	view := func(opening string) string {
		t.Helper()
		s.pick(passageNamed(t, s.passages, opening))
		if !s.view.box.Visible() {
			t.Fatal("the view is not shown")
		}
		return readingMarkdown(s.currentReading(), s.selected, "INT.", "EXT.", "INT./EXT.")
	}
	for _, c := range []struct{ opening, want string }{
		{"Anna stood", "INT. KITCHEN"},
		{"Anna stood", "thought"},
		{"“Who’s there?”", "ANNA"},
		{"“Who’s there?”", "whispering"},
		{"“Only me,”", "TOM"},
		{"“Not tonight.”", "ANNA"}, // the turns
		{"“Please, Anna.”", "TOM"},
		{"The next morning", "EXT. STREET - MORNING"},
	} {
		if got := view(c.opening); !strings.Contains(got, c.want) {
			t.Errorf("%q: no %q in\n%s", c.opening, c.want, got)
		}
	}
	s.view.close.Tapped(&fyne.PointEvent{})
	if s.view.box.Visible() {
		t.Error("closed, it is still shown")
	}
	if unnamedIn := unnamed(w.fyneWindow.Canvas()); len(unnamedIn) > 0 {
		t.Errorf("no name for %v", unnamedIn)
	}
}

// A list item written with a dash is a line of dialogue when its run
// reads as talk; a plain list stays narration.
func TestDashItemsAsDialogue(t *testing.T) {
	ps := passagesOf("- Bonjour, dit Pierre.\n- Bonsoir, répondit Marie.\n\nText.\n\n- one thing\n- another\n")
	r := (&sourcePane{passages: ps, src: "x", w: &MainWindow{language: "fr"}}).currentReading()
	if l := r.Passages[0].Lines; len(l) != 1 || l[0].Speaker != "Pierre" {
		t.Errorf("dialogue item: %+v", l)
	}
	if l := r.Passages[3].Lines; len(l) != 0 {
		t.Errorf("list item read as dialogue: %+v", l)
	}
}

// TestReadingScreenshot renders the view on Austen's first chapter, a
// line of dialogue picked (SHOT_DIR as the other screenshots).
func TestReadingScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	b, err := os.ReadFile("internal/storyread/testdata/en-austen-a.md")
	if err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, "Title: Pride and Prejudice\n\nINT. LONGBOURN - DAY\n\nMrs. Bennet hurries in.\n", string(b))
	w.fyneWindow.Resize(fyne.NewSize(1200, 760))
	s := w.sourcePane()
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		id := passageNamed(t, s.passages, "“My dear Mr. Bennet,” said his lady")
		s.pick(id)
		s.list.ScrollTo(id)
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "accolade-source-reading-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
}

package main

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// The sample's links are chips: "↗ source" in place of the note, its
// quote decoded in the tip, the note's own text while the cursor is in
// it; Find opens the link it finds in; the text never changes.
func TestLinkChipsSample(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	e := w.textEditor
	text := e.Text()
	e.SetCursorOffset(0)
	chips := w.linkChips(text)
	links := 0
	seen := map[int]bool{}
	for _, l := range parseLinks(text) {
		if !seen[l.Start] {
			seen[l.Start] = true
			links++
		}
	}
	if len(chips) != links || links < 20 {
		t.Fatalf("%d chips for %d links", len(chips), links)
	}
	for _, c := range chips {
		if c.Label != "↗ source" {
			t.Errorf("label %q", c.Label)
		}
		if strings.Contains(c.Tip, "%2C") || strings.Contains(c.Tip, "#:~:") || !strings.HasPrefix(c.Tip, "“") {
			t.Errorf("a tip not decoded: %q", c.Tip)
		}
	}
	for _, r := range shownRows(w) {
		if strings.Contains(r, "#:~:text") {
			t.Errorf("a link drawn as its text with the cursor away: %q", r)
		}
	}
	if rowWith(w, "Hold him! ↗ source") == "" {
		t.Error("no chip after Hold him!")
	}
	// the tips: commas, a range, the words before
	tips := map[string]bool{}
	for _, c := range chips {
		tips[c.Tip] = true
	}
	for _, want := range []string{
		"“THE police superintendent Otchumyelov … beggar near them.”",
		"“\"H'm. Very good,\" says … it, I ask?\"”",
		"“\"Yes.\"” (after “visit?”)",
	} {
		if !tips[want] {
			t.Errorf("no tip %q", want)
		}
	}
	// the cursor in a link: its text; out: a chip again
	e.SetCursorOffset(runeIndex(t, text, "superintendent Otchumyelov,"))
	if rowWith(w, "[[from: #:~:text=THE police") == "" {
		t.Errorf("the link the cursor is in is not its text:\n%s", strings.Join(shownRows(w)[:14], "\n"))
	}
	if rowWith(w, "Hold him! ↗ source") == "" {
		t.Error("another link opened too")
	}
	e.SetCursorOffset(0)
	if rowWith(w, "[[from: #:~:text=THE police") != "" {
		t.Error("the link stayed open with the cursor out")
	}
	// a hover's tip is the chip's
	at := runeIndex(t, text, "[[from: #:~:text=\"So you bite")
	if c, ok := e.ChipAt(at); !ok || !strings.HasPrefix(c.Tip, "“\"So you bite, you") {
		t.Errorf("ChipAt %+v %v", c, ok)
	}
	// Find, a word in a link: the link opens
	w.showFind()
	w.searchBar.SetSearchText("aforementioned")
	w.findNext()
	if rowWith(w, "aforementioned") == "" {
		t.Error("Find did not open the link it found the word in")
	}
	w.hideFindReplace()
	if e.Text() != text || e.AccessibilityText() != text {
		t.Error("the chips changed the text, or what a screen reader reads")
	}
}

// With the source at hand, a link to a range says how many passages; a
// link the source no longer has says so; a tap on a chip opens it.
func TestLinkChipsPassages(t *testing.T) {
	script := "INT. GARDEN - DAY\n\n[[from: #:~:text=She opened the door,nobody answered]]\n\nRain.\n\n[[from: #:~:text=Not in the story]]\n"
	w, _ := sourceWindow(t, script, sourceStory)
	w.sourcePane().current()
	e := w.textEditor
	e.SetCursorOffset(0)
	chips := w.linkChips(e.Text())
	if len(chips) != 2 || chips[0].Label != "↗ 2 passages" || chips[1].Label != "↗ source?" {
		t.Fatalf("chips %+v", chips)
	}
	if !strings.Contains(chips[0].Tip, "“She opened the door … nobody answered”") || !strings.Contains(chips[0].Tip, "2 passages") {
		t.Errorf("tip %q", chips[0].Tip)
	}
	if rowWith(w, "↗ 2 passages") == "" {
		t.Errorf("rows:\n%s", strings.Join(shownRows(w), "\n"))
	}
	// a tap on the chip: the cursor in it, its text shown
	row := -1
	for i, r := range shownRows(w) {
		if strings.Contains(r, "↗ 2 passages") {
			row = i
		}
	}
	cw := e.Size().Width / float32(e.Columns())
	col := strings.Index(shownRows(w)[row], "↗")
	h := e.MinSize().Height / float32(len(shownRows(w)))
	e.Tapped(&fyne.PointEvent{Position: fyne.NewPos((float32(col)+0.5)*cw, (float32(row)+0.5)*h)})
	if rowWith(w, "[[from: #:~:text=She opened") == "" {
		t.Errorf("the tap did not open the chip; cursor at %d", e.CursorOffset())
	}
}

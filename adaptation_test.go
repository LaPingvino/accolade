package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// The cut: entries are read leniently: on the key's line or below it,
// quoted or bare, as a list, with (n); other sections of the block stay
// as they are when the entries are written again.
func TestCutEntries(t *testing.T) {
	inside := strings.Join([]string{
		"characters: ANNA = Anna, Annie",
		`cut: "The rain kept falling." (2), "She opened the door"`,
		`  - “A list item of its own”`,
		"  and another",
		"",
		"places: the house = INT. HOUSE",
	}, "\n")
	cuts := parseCuts(inside)
	var got []string
	for _, c := range cuts {
		got = append(got, c.String())
	}
	want := []string{`"The rain kept falling." (2)`, `"She opened the door"`, `"A list item of its own"`, `"and another"`}
	if !slices.Equal(got, want) {
		t.Fatalf("cuts %q\nwant %q", got, want)
	}
	// written again: one per line, where the section was, the rest kept
	out := withCuts(inside, cuts[:2])
	wantOut := "characters: ANNA = Anna, Annie\ncut:\n  \"The rain kept falling.\" (2)\n  \"She opened the door\"\nplaces: the house = INT. HOUSE"
	if out != wantOut {
		t.Errorf("written:\n%s\nwant\n%s", out, wantOut)
	}
	if again := parseCuts(out); len(again) != 2 || again[0].Nth != 2 || again[1].Quote != "She opened the door" {
		t.Errorf("read again: %+v", again)
	}
	// no cuts and nothing else: no block
	if out := withCuts("cut:\n  \"x y\"\n", nil); out != "" {
		t.Errorf("empty: %q", out)
	}
	// a block without cut: gets the section at its end
	if out := withCuts("characters: ANNA = Anna", cuts[1:2]); out != "characters: ANNA = Anna\ncut:\n  \"She opened the door\"" {
		t.Errorf("added: %q", out)
	}
}

// The block goes after the script, before the source and notes blocks;
// each block is found whatever the order, and links are not looked for
// in it.
func TestAdaptationBlockPlace(t *testing.T) {
	script := "INT. HOUSE - NIGHT\n\nRain. [[from: \"The rain kept falling.\"]]\n\n/* Source\n# Story\n\nThe rain kept falling.\n*/\n\n/* Notes\nkeep\n*/\n"
	inside := "cut:\n  \"[[from: \\\"x\\\"]] y\""
	start, end, block := withAdaptationBlock(script, inside)
	r := []rune(script)
	with := string(r[:start]) + block + string(r[end:])
	want := "INT. HOUSE - NIGHT\n\nRain. [[from: \"The rain kept falling.\"]]\n\n/* Adaptation\n" + inside + "\n*/\n\n/* Source\n# Story\n\nThe rain kept falling.\n*/\n\n/* Notes\nkeep\n*/\n"
	if with != want {
		t.Fatalf("with the block:\n%s\nwant\n%s", with, want)
	}
	if got, ok := adaptationBlock(with); !ok || got != inside {
		t.Errorf("block %q %v", got, ok)
	}
	if _, _, md, ok := sourceBlock(with); !ok || md != "# Story\n\nThe rain kept falling.\n" {
		t.Errorf("source beside it: %q %v", md, ok)
	}
	if n := notesOf(with); n != "keep" {
		t.Errorf("notes: %q", n)
	}
	if l := parseLinks(with); len(l) != 1 {
		t.Errorf("links: %+v", l)
	}
	if b := bodyEnd(with); string([]rune(with)[:b]) != "INT. HOUSE - NIGHT\n\nRain. [[from: \"The rain kept falling.\"]]" {
		t.Errorf("body: %q", string([]rune(with)[:b]))
	}
	// the other order: the source first
	other := "Rain.\n\n/* Source\nThe rain.\n*/\n\n/* Adaptation\ncut:\n  \"The rain\"\n*/\n"
	if a, ok := adaptationBlock(other); !ok || a != "cut:\n  \"The rain\"" {
		t.Errorf("after the source: %q", a)
	}
	if _, _, md, ok := sourceBlock(other); !ok || md != "The rain.\n" {
		t.Errorf("source before it: %q", md)
	}
	// taken out again: the script as it was
	s2, e2, b2 := withAdaptationBlock(with, "")
	if back := string([]rune(with)[:s2]) + b2 + string([]rune(with)[e2:]); back != script {
		t.Errorf("taken out:\n%q", back)
	}
	// at the very end, without other blocks
	s3, e3, b3 := withAdaptationBlock("Rain.\n", "cut:\n  \"x\"")
	if got := "Rain.\n"[:s3] + b3 + "Rain.\n"[e3:]; got != "Rain.\n\n/* Adaptation\ncut:\n  \"x\"\n*/\n" {
		t.Errorf("at the end: %q", got)
	}
}

// Statuses come from the links (an ambiguous one does not count) and the
// cut entries; headings have none; the count leaves cut passages out of
// what is to go.
func TestStatuses(t *testing.T) {
	ps := passagesOf(sourceStory)
	m := newSourceModel(ps, sourceScript, nil)
	door, who := passageNamed(t, ps, "She opened"), passageNamed(t, ps, `"Who's`)
	morning, item := passageNamed(t, ps, "The next morning"), passageNamed(t, ps, "a list item")
	for i, p := range ps {
		want := statusToDo
		switch {
		case p.Kind == 1 || !p.counted(): // headings
			want = statusNone
		case i == door || i == who || i == morning:
			want = statusAdapted
		}
		if !p.counted() {
			want = statusNone
		}
		if m.status[i] != want {
			t.Errorf("%q: %v, want %v", p.Plain, m.status[i], want)
		}
	}
	// "The rain kept falling." is there twice: its link is ambiguous
	if c := countOf(m.status); c != (adaptCount{Adapted: 3, ToDo: 4}) || c.String() != "42% adapted · 0 cut · 4 to go" {
		t.Errorf("count %+v %q", c, c.String())
	}
	// cut: the list item, and the second rain by its number
	rain2 := 9
	if ps[rain2].Plain != "The rain kept falling." {
		t.Fatalf("passage 9 is %q", ps[rain2].Plain)
	}
	c2 := cutFor(ps, rain2)
	if !strings.HasPrefix(c2.String(), "#:~:text=") || !strings.Contains(c2.String(), "-,The,falling.") ||
		resolveCut(c2, ps) != (cutSpan{rain2, rain2}) {
		t.Errorf("cut entry for the second rain: %q", c2.String())
	}
	if old, ok := parseCutEntry(`"The rain kept falling." (2)`); !ok || resolveCut(old, ps) != (cutSpan{rain2, rain2}) {
		t.Errorf("the old form's (2): %+v", old)
	}
	cuts := []cutMark{cutFor(ps, item), c2, {Quote: "Nothing like it", Words: normWords("Nothing like it")}}
	script := sourceScript + "\n/* Adaptation\n" + withCuts("", cuts) + "\n*/\n"
	m = newSourceModel(ps, script, nil)
	if m.status[item] != statusCut || m.status[rain2] != statusCut || m.status[6] != statusToDo || m.cutAt[2] != noCut {
		t.Errorf("cut: %v %v %v %v", m.status[item], m.status[rain2], m.status[6], m.cutAt)
	}
	if c := countOf(m.status); c.String() != "60% adapted · 2 cut · 2 to go" {
		t.Errorf("with cuts: %q", c.String())
	}
	// a cut passage linked is adapted, once
	m = newSourceModel(ps, script+"\nINT. X - DAY\n\n[[from: \"a list item of its own\"]]\n", nil)
	if m.status[item] != statusAdapted || countOf(m.status).Adapted != 4 {
		t.Errorf("linked and cut: %v", m.status[item])
	}
	if (adaptCount{Cut: 2}).String() != "100% adapted · 2 cut · 0 to go" || (adaptCount{}).String() != "" {
		t.Error("count edge cases")
	}
}

// Mark as Cut in the pane: an entry in the script's block, one undo
// step; Unmark takes it out; the rows and the count follow, undo too.
// The notes and the source kept in the script stay as they are.
func TestMarkAsCut(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	w.writeNotes("my notes")
	s := w.sourcePane()
	s.follow()
	item := passageNamed(t, s.passages, "a list item")
	before := w.textEditor.Text()
	if rowOf(s, item).cut.Visible() {
		t.Error("Mark as Cut with nothing picked")
	}
	s.list.Select(item)
	if c := rowOf(s, item).cut; !c.Visible() || c.Text != "Mark as Cut" || !rowOf(s, item).adapt.Visible() {
		t.Fatalf("cut button: %q shown %v", c.Text, c.Visible())
	}
	rowOf(s, item).cut.OnTapped()
	text := w.textEditor.Text()
	inside, ok := adaptationBlock(text)
	if !ok || inside != "cut:\n  #:~:text=a,own" {
		t.Fatalf("block %q %v in\n%s", inside, ok, text)
	}
	if notesOf(text) != "my notes" || w.sourceRef().Origin != sourceInside {
		t.Errorf("notes %q, source %v", notesOf(text), w.sourceRef())
	}
	if _, _, md, _ := sourceBlock(text); md != sourceStory {
		t.Errorf("source block changed: %q", md)
	}
	if c := rowOf(s, item).cut; s.statusOf(item) != statusCut || c.Text != "Unmark Cut" || !strings.Contains(s.count.Text, "1 cut") {
		t.Errorf("after cut: %v %q %q", s.statusOf(item), c.Text, s.count.Text)
	}
	if mark := rowMark(s, item); mark.status != statusCut {
		t.Errorf("row mark %v", mark.status)
	}
	// undo: one step, and the pane follows
	w.textEditor.Undo()
	if w.textEditor.Text() != before {
		t.Fatalf("undone:\n%s", w.textEditor.Text())
	}
	if s.statusOf(item) != statusToDo || !strings.Contains(s.count.Text, "0 cut") {
		t.Errorf("after undo: %v %q", s.statusOf(item), s.count.Text)
	}
	// both copies of a passage there twice, each its own
	s.list.Select(6)
	s.toggleCut()
	s.list.Select(9)
	s.toggleCut()
	inside, _ = adaptationBlock(w.textEditor.Text())
	if inside != "cut:\n  #:~:text=another-,The,falling.\n  #:~:text=open.-,The,falling." {
		t.Errorf("twice there: %q", inside)
	}
	s.list.Select(6)
	s.toggleCut() // unmark the first only
	inside, _ = adaptationBlock(w.textEditor.Text())
	if inside != "cut:\n  #:~:text=open.-,The,falling." || s.statusOf(6) != statusToDo || s.statusOf(9) != statusCut {
		t.Errorf("unmarked one: %q", inside)
	}
	// an adapted passage or a heading is not cut
	opened := passageNamed(t, s.passages, "She opened")
	s.list.Select(opened)
	if r := rowOf(s, opened); r.cut.Visible() || r.adapt.Visible() || !r.scenes.Visible() {
		t.Error("cut or Adapt… for an adapted passage, or no → to its scene")
	}
	// linking a cut passage takes its entry out, in the same step (the
	// second of two alike too: its prefix tells it apart)
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "ANNA")) - 1)
	s.list.Select(9)
	w.linkToSource()
	if inside, _ = adaptationBlock(w.textEditor.Text()); s.statusOf(9) != statusAdapted || inside != "" {
		t.Errorf("linking the second rain: %v, block %q", s.statusOf(9), inside)
	}
	w.textEditor.Undo()
	s.list.Select(item)
	s.toggleCut()
	s.list.Select(9)
	s.toggleCut() // only the list item cut now
	pre := w.textEditor.Text()
	s.list.Select(item)
	w.linkToSource()
	if _, ok := adaptationBlock(w.textEditor.Text()); ok || s.statusOf(item) != statusAdapted {
		t.Errorf("linked a cut passage: block still there, %v", s.statusOf(item))
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != pre {
		t.Error("link and uncut are not one step")
	}
}

// rowOf is passage id's row as the list shows it.
func rowOf(s *sourcePane, id int) passageRow {
	row := newPassageRow()
	s.show(row, id)
	return rowParts(row)
}

// rowMark is the status mark of passage id's row as the list shows it.
func rowMark(s *sourcePane, id int) *statusMark {
	row := newPassageRow()
	s.show(row, id)
	return rowParts(row).mark
}

const nextScript = "Title: The Garden\n\nINT. HOUSE - NIGHT\n\nAnna at the door. [[from: \"She opened the door to the garden and…\"]]\n\nINT. LATER - DAY\n\nStuff.\n\n/* Notes\nkeep\n*/\n"

// Every passage done: Next says so; Markdown has none of it.
func TestNextAllDone(t *testing.T) {
	w, _ := sourceWindow(t, "INT. A - DAY\n\nx\n", "# T\n\nOne thing.\n\nAnother.\n")
	s := w.sourcePane()
	s.setCut(1, true)
	s.setCut(2, true)
	w.nextSourcePassage()
	if !strings.Contains(w.statusLabel.Text, "Every passage") || s.count.Text != "100% adapted · 2 cut · 0 to go" {
		t.Errorf("all done: %q %q", w.statusLabel.Text, s.count.Text)
	}
	w.setDocumentType(true)
	before := w.textEditor.Text()
	w.nextSourcePassage()
	if w.textEditor.Text() != before || slices.Contains(menuLabels(w, "Adaptation"), "Next Source Passage") ||
		slices.Contains(menuLabels(w, "Adaptation"), "Mark Source Passage as Cut") {
		t.Error("Markdown with Next")
	}
}

// TestStatusScreenshots renders the pane with statuses (adapted, cut,
// to do, a heading) and the count, light and dark, and the skip
// question (SHOT_DIR as in screenshot_test.go).
func TestStatusScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	s := w.sourcePane()
	s.setCut(passageNamed(t, s.passages, "a list item"), true)
	s.selected = passageNamed(t, s.passages, "and another")
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "ANNA")))
	shot := func(name string) {
		w.fyneWindow.Content().Refresh()
		s.list.Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		shot("accolade-statuses-" + scheme + ".png")
	}
	s.stepPassage(1)
	shot("accolade-statuses-next-dark.png")
}

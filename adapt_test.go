package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
)

// The passages of sourceStory: 0 Part One, 1 Chapter 1 (headings), 2
// She opened, 3 Who's there, 4 a list item, 5 and another, 6 the rain,
// 7 Chapter 2 (a heading), 8 The next morning, 9 the rain again.

// gardenModel is sourceStory's passages read against script.
func gardenModel(script string) *sourceModel {
	return newSourceModel(passagesOf(sourceStory), script, nil)
}

// lineIn is the line of script holding s.
func lineIn(t *testing.T, script, s string) int {
	t.Helper()
	for i, l := range strings.Split(script, "\n") {
		if strings.Contains(l, s) {
			return i
		}
	}
	t.Fatalf("no %q in\n%s", s, script)
	return -1
}

const (
	linkOpened  = "[[from: #:~:text=She opened the door to the garden and stepped out into the rain.]]"
	linkWho     = `[[from: #:~:text="Who's there?" Anna called%2C but nobody answered.]]`
	linkMorning = "[[from: #:~:text=The next morning the,door stood open.]]"
	linkRain1   = "[[from: #:~:text=another-,The rain kept falling.]]"
)

// Where Next and Previous go on from, the item's cases: a middle chapter
// adapted first, scenes out of the story's order, a script partly
// linked, the cursor in a scene with no link, a passage picked, a
// passage two scenes link, the view's top; and at the end, none.
func TestAnchorRules(t *testing.T) {
	type c struct {
		name, script, at string // the cursor on the line holding at
		picked, top      int
		kind             anchorKind
		next, prev       int
	}
	middle := "Title: G\n\nEXT. GARDEN - MORNING\n\n" + linkMorning + "\n\nFrost.\n"
	outOfOrder := "INT. A - DAY\n\n" + linkRain1 + "\n\nINT. B - DAY\n\n" + linkOpened + "\n\nINT. C - DAY\n\nNothing yet.\n"
	partly := "Title: G\n\nINT. HOUSE - NIGHT\n\n" + linkOpened + "\n\nINT. LATER - DAY\n\nStuff.\n"
	twice := "INT. A - DAY\n\n" + linkOpened + "\n\nINT. B - DAY\n\n" + linkWho + "\n\nINT. C - DAY\n\n" + linkOpened + "\n" + linkRain1 + "\n"
	for _, k := range []c{
		{"middle chapter first", middle, "Frost", -1, -1, anchorScene, 9, 6},
		{"out of order: B adapts the opening", outOfOrder, linkOpened, -1, -1, anchorScene, 3, -1},
		{"out of order: A adapts the rain", outOfOrder, linkRain1, -1, -1, anchorScene, 8, 5},
		{"unlinked scene: the scene above", outOfOrder, "Nothing yet", -1, -1, anchorBefore, 3, -1},
		{"partly linked, the cursor in the title page: the view's top", partly, "Title", -1, 5, anchorView, 5, 5},
		{"partly linked, nothing to go by: the start", partly, "Title", -1, -1, anchorStart, 3, -1},
		{"a passage picked", partly, "Stuff", 8, -1, anchorPicked, 9, 6},
		{"two scenes link the opening: from A (B adapts the next)", twice, "INT. A", -1, -1, anchorScene, 4, -1},
		{"two scenes link the opening: from C, the rain", twice, "INT. C", -1, -1, anchorScene, 8, 5},
		{"the end", partly + "\nINT. END - DAY\n\n" + linkMorning + "\n[[from: #:~:text=open.-,The rain kept falling.]]\n", "INT. END", -1, -1, anchorScene, -1, 6},
	} {
		t.Run(k.name, func(t *testing.T) {
			m := gardenModel(k.script)
			a := m.anchorAt(lineIn(t, k.script, k.at), k.picked, k.top)
			if a.kind != k.kind {
				t.Errorf("anchor %+v, want kind %v", a, k.kind)
			}
			if n := m.step(a, 1); n != k.next {
				t.Errorf("next %d, want %d", n, k.next)
			}
			if p := m.step(a, -1); p != k.prev {
				t.Errorf("previous %d, want %d", p, k.prev)
			}
		})
	}
	m := gardenModel(partly)
	if m.wrapTo(1) != 3 || m.wrapTo(-1) != 9 {
		t.Errorf("round: %d %d", m.wrapTo(1), m.wrapTo(-1))
	}
}

// adaptCtxFor is the plan's context for script with the cursor on the
// line holding at, without a source's reading.
func adaptCtxFor(t *testing.T, script, at string) adaptCtx {
	ps := passagesOf(sourceStory)
	cursor := 0
	if at != "" {
		cursor = strings.Index(script, at)
		if cursor < 0 {
			t.Fatalf("no %q", at)
		}
		cursor = len([]rune(script[:cursor]))
	}
	return adaptCtx{text: script, ps: ps, m: newSourceModel(ps, script, nil), cursor: cursor, labels: spanLabels(script, nil)}
}

// Adapt…'s likely choices and why: the cursor's scene when the story's
// scene goes on from what it adapts, or it adapts nothing yet; a scene
// of its own otherwise (the reason the story's own when it starts one);
// the place in the story's order, by the nearest passage adapted before
// it (else after it), within its chapter's section.
func TestAdaptDefaults(t *testing.T) {
	partly := "Title: G\n\nINT. HOUSE - NIGHT\n\n" + linkOpened + "\n\nINT. LATER - DAY\n\nStuff.\n"
	c := adaptCtxFor(t, partly, "Stuff") // the cursor in a scene with no link
	pl := planAdapt(c, 3)
	if h := pl.Hows[pl.How]; h.How != howAddHere || !strings.Contains(h.Reason, "Scene 2 adapts no passage yet") {
		t.Errorf("an unlinked scene: %+v", h)
	}
	if w := pl.Wheres[pl.Where]; w.Where != whereStory || w.Line != lineIn(t, partly, "INT. LATER") ||
		!strings.Contains(w.Note, "after Scene 1 · INT. HOUSE - NIGHT") || !strings.Contains(w.Reason, "She opened") {
		t.Errorf("where: %+v", w)
	}
	c = adaptCtxFor(t, partly, "She opened") // in HOUSE: the story goes on from its passage
	if pl = planAdapt(c, 3); pl.Hows[pl.How].How != howAddHere || !strings.Contains(pl.Hows[pl.How].Reason, "goes on from “She opened") {
		t.Errorf("going on: %+v", pl.Hows[pl.How])
	}
	c.opens = "chapter heading" // the story starts a scene there
	if pl = planAdapt(c, 3); pl.Hows[pl.How].How != howNewScene || pl.Hows[pl.How].Reason != "a scene starts here in the story: chapter heading" {
		t.Errorf("a new scene in the story: %+v", pl.Hows[pl.How])
	}
	c = adaptCtxFor(t, partly, "Title") // not in a scene
	pl = planAdapt(c, 4)
	if pl.Hows[pl.How].How != howNewScene || !pl.Hows[2].Off {
		t.Errorf("outside a scene: %+v", pl.Hows)
	}
	// before the scene adapting a later passage, none before it adapted
	later := "Title: G\n\nINT. MORNING - DAY\n\n" + linkMorning + "\n"
	c = adaptCtxFor(t, later, "Title")
	if w := planAdapt(c, 3).Wheres[0]; w.Line != lineIn(t, later, "INT. MORNING") || !strings.Contains(w.Note, "before Scene 1") {
		t.Errorf("before a later one: %+v", w)
	}
	// in its chapter's section: a scene for Chapter 2 goes under it, not
	// after Chapter 1's last scene
	sections := "Title: G\n\n# Part One\n\n## Chapter 1\n\nINT. HOUSE - NIGHT\n\n" + linkWho + "\n\n## Chapter 2\n\nINT. LATE - DAY\n\n[[from: #:~:text=open.-,The rain kept falling.]]\n"
	c = adaptCtxFor(t, sections, "Title")
	w := planAdapt(c, 8).Wheres[0]
	if w.Line != lineIn(t, sections, "INT. LATE") || !strings.Contains(w.Note, "in ## Chapter 2") {
		t.Errorf("in its section: %+v", w)
	}
	if b := sceneBlock(c, 8, w.Line, "INT. GARDEN - MORNING", ""); b != "INT. GARDEN - MORNING\n\n"+linkMorning {
		t.Errorf("block %q (a section the script has is not written again)", b)
	}
	// the other places, one each
	c = adaptCtxFor(t, partly, "Stuff")
	var kinds []adaptWhere
	for _, o := range planAdapt(c, 3).Wheres {
		kinds = append(kinds, o.Where)
	}
	if len(kinds) != 2 || kinds[1] != whereCursor { // after the cursor's scene, and the end, are after its paragraph
		t.Errorf("places %v", kinds)
	}
}

// An insertion only, a blank line around it, before the blocks at the
// end.
func TestInsertBlock(t *testing.T) {
	for _, k := range []struct{ text, want string }{
		{"A\n\nB\n", "A\n\nX\n\nB\n"},
		{"A\nB\n", "A\n\nX\n\nB\n"},
		{"A\n\n\nB\n", "A\n\n\nX\n\nB\n"},
	} {
		ed := insertBlock(k.text, lineIn(t, k.text, "B"), "X")
		r := []rune(k.text)
		if got := string(r[:ed.At]) + ed.Text + string(r[ed.Replace:]); got != k.want || ed.Replace != ed.At {
			t.Errorf("%q: %q, want %q", k.text, got, k.want)
		}
	}
	for _, k := range []struct{ text, want string }{
		{"A\n", "A\n\nX\n"},
		{"A", "A\n\nX\n"},
		{"", "X\n"},
		{"A\n\n/* Notes\nn\n*/\n", "A\n\nX\n\n/* Notes\nn\n*/\n"},
		{"A\n/* Notes\nn\n*/\n", "A\n\nX\n\n/* Notes\nn\n*/\n"},
	} {
		ed := insertBlock(k.text, 1000, "X")
		r := []rune(k.text)
		if got := string(r[:ed.At]) + ed.Text + string(r[ed.Replace:]); got != k.want || ed.Replace != ed.At {
			t.Errorf("end of %q: %q, want %q", k.text, got, k.want)
		}
	}
}

// Next and Previous only move: the passage picked and in sight, the
// script as it was; the caption says from where; at the end it offers
// to go round, and Next again goes round.
func TestNextOnlyMoves(t *testing.T) {
	w, p := sourceWindow(t, nextScript, sourceStory)
	s := w.sourcePane()
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "Anna at the door")))
	s.follow()
	if got := s.anchorLine.Text; !strings.Contains(got, "after: “She opened") || !strings.Contains(got, "the cursor's scene") {
		t.Errorf("caption %q", got)
	}
	w.nextSourcePassage()
	who := passageNamed(t, s.passages, `"Who's`)
	if w.textEditor.Text() != nextScript || s.selected != who || s.prompt.box.Visible() {
		t.Fatalf("Next wrote, or picked %d, or asked %q", s.selected, s.prompt.label.Text)
	}
	if !strings.Contains(s.anchorLine.Text, "picked") {
		t.Errorf("caption after Next %q", s.anchorLine.Text)
	}
	// on from the one picked, skipping nothing written
	for _, want := range []int{4, 5, 6, 8, 9} {
		w.nextSourcePassage()
		if s.selected != want {
			t.Fatalf("next: %d, want %d", s.selected, want)
		}
	}
	w.previousSourcePassage()
	if s.selected != 8 {
		t.Errorf("previous: %d", s.selected)
	}
	w.nextSourcePassage()
	w.nextSourcePassage() // none left after the last
	if s.selected != 9 || !s.prompt.box.Visible() || !strings.Contains(s.prompt.label.Text, "No passage to adapt after") ||
		s.prompt.yes.Text != "From the Start" {
		t.Fatalf("at the end: %d %q", s.selected, s.prompt.label.Text)
	}
	w.nextSourcePassage() // again: round
	if s.selected != who || s.prompt.box.Visible() {
		t.Errorf("round: %d", s.selected)
	}
	// the cursor moved: the anchor is the cursor's scene again
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "Stuff")))
	s.follow()
	if !strings.Contains(s.anchorLine.Text, "above the cursor") {
		t.Errorf("caption in an unlinked scene %q", s.anchorLine.Text)
	}
	if w.textEditor.Text() != nextScript {
		t.Error("the script changed")
	}
	// a picked passage found again when the source changes before it
	s.pick(8)
	story := strings.Replace(sourceStory, "## Chapter 1\n", "## Chapter 1\n\nA new opening line.\n", 1)
	md := filepath.Join(filepath.Dir(p), "story.md")
	os.WriteFile(md, []byte(story), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(md, later, later)
	s.follow()
	if s.selected != 9 || !strings.HasPrefix(s.passages[s.selected].Plain, "The next morning") {
		t.Errorf("found again: %d", s.selected)
	}
	w.nextSourcePassage()
	if s.selected != 10 {
		t.Errorf("after the source changed: %d", s.selected)
	}
}

// Adapt…: a new scene where the story's order puts it, its heading
// written over the placeholder, one undo step, the preview exactly what
// went in; a new chapter's section first; at the end, before the blocks;
// a passage there twice told apart by the words before it.
func TestAdaptNewScene(t *testing.T) {
	w, _ := sourceWindow(t, nextScript, sourceStory)
	s := w.sourcePane()
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "Anna at the door")))
	w.nextSourcePassage()
	who := s.selected
	w.adaptSourcePassage()
	d := s.adapting
	if d == nil || d.pop == nil || !d.pop.Visible() {
		t.Fatal("no Adapt… dialog")
	}
	if d.howNow() != howAddHere {
		t.Errorf("likely %v: the story's scene goes on in HOUSE", d.howNow())
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %q", u)
	}
	d.how.choose(0) // Start a new scene
	d.heading.SetText("INT. PLACE - DAY")
	preview := d.preview.Text
	d.accept()
	want := strings.Replace(nextScript, "INT. LATER", "INT. PLACE - DAY\n\n"+linkWho+"\n\nINT. LATER", 1)
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("inserted:\n%s\nwant\n%s", got, want)
	}
	// the preview shows the link as the writer reads it, never encoded
	if preview != "INT. PLACE - DAY\n\n[[↗ “\"Who's there?\" Anna called, but nobody answered.”]]" {
		t.Errorf("preview %q", preview)
	}
	if s.statusOf(who) != statusAdapted || selected(w) != "PLACE - DAY" || s.adapting != nil {
		t.Errorf("%v, selected %q", s.statusOf(who), selected(w))
	}
	if a, b := w.textEditor.Flash(); a >= b {
		t.Error("the new scene does not flash")
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != nextScript {
		t.Fatalf("not one undo step:\n%s", w.textEditor.Text())
	}
	w.textEditor.Redo()
	// the cursor in the new scene: Next is the list item; adapted at the
	// cursor (the likely, Enter)
	w.nextSourcePassage()
	w.adaptSourcePassage()
	s.adapting.how.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if !strings.Contains(w.textEditor.Text(), linkWho+" [[from: #:~:text=a list item of its own]]") &&
		!strings.Contains(w.textEditor.Text(), "[[from: #:~:text=a list item of its own]]") {
		t.Errorf("Enter did not take the likely:\n%s", w.textEditor.Text())
	}
	// the rain, there twice: its link tells it by the words before it
	s.pick(6)
	w.adaptSourcePassage()
	s.adapting.how.choose(0)
	s.adapting.accept()
	if !strings.Contains(w.textEditor.Text(), linkRain1) || s.statusOf(6) != statusAdapted {
		t.Errorf("the first rain: %v\n%s", s.statusOf(6), w.textEditor.Text())
	}
	// Chapter 2: its section first; at the end, before the notes
	s.pick(8)
	w.adaptSourcePassage()
	d = s.adapting
	d.how.choose(0)
	d.where.choose(len(d.plan.Wheres) - 1) // at the end
	d.heading.SetText("EXT. GARDEN - MORNING")
	d.accept()
	text := w.textEditor.Text()
	if !strings.Contains(text, "## Chapter 2\n\nEXT. GARDEN - MORNING\n\n"+linkMorning+"\n\n/* Notes\nkeep\n*/\n") {
		t.Errorf("chapter 2 at the end:\n%s", text)
	}
	if strings.Contains(text, "## Chapter 1") || notesOf(text) != "keep" {
		t.Errorf("a section for the chapter the script is in, or the notes:\n%s", text)
	}
	if strings.Contains(w.textEditor.SelectedText(), "GARDEN") {
		t.Error("a heading written in the dialog is selected")
	}
}

// Draft, Mark as cut, Skip for now; Esc cancels; Adapt… on a cut passage
// takes its cut away in the same step.
func TestAdaptOtherWays(t *testing.T) {
	w, _ := sourceWindow(t, nextScript, sourceStory)
	s := w.sourcePane()
	toEnd(w)
	s.pick(3)
	w.adaptSourcePassage()
	d := s.adapting
	d.pop.OnEscape()
	if s.adapting != nil || d.pop.Visible() || w.textEditor.Text() != nextScript {
		t.Fatal("Esc did not cancel")
	}
	// draft: Anna's line under her cue, the rest as action
	w.adaptSourcePassage()
	d = s.adapting
	d.how.choose(1)
	if !strings.Contains(d.preview.Text, "\nWho's there?\n\nAnna called, but nobody answered.") { // the line under a cue, the rest as action
		t.Errorf("draft preview %q", d.preview.Text)
	}
	d.preview.SetText(d.preview.Text + "\n\nShe waits.") // written in by hand: kept
	d.accept()
	if !strings.Contains(w.textEditor.Text(), "She waits.") || s.statusOf(3) != statusAdapted {
		t.Errorf("draft:\n%s", w.textEditor.Text())
	}
	// cut, then adapted from the dialog: one step, the cut gone
	s.pick(4)
	w.adaptSourcePassage()
	s.adapting.how.choose(3)
	s.adapting.accept()
	if s.statusOf(4) != statusCut {
		t.Fatalf("cut: %v", s.statusOf(4))
	}
	before := w.textEditor.Text()
	s.pick(4)
	w.adaptSourcePassage()
	s.adapting.how.choose(0)
	s.adapting.accept()
	if _, ok := adaptationBlock(w.textEditor.Text()); ok || s.statusOf(4) != statusAdapted {
		t.Errorf("uncut: %v\n%s", s.statusOf(4), w.textEditor.Text())
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != before {
		t.Error("adapt and uncut are not one step")
	}
	// skip: still to do, the next picked
	s.pick(5)
	w.adaptSourcePassage()
	s.adapting.how.choose(4)
	s.adapting.accept()
	if s.statusOf(5) != statusToDo || s.selected != 6 || w.textEditor.Text() != before {
		t.Errorf("skip: %v, picked %d", s.statusOf(5), s.selected)
	}
	// an adapted passage: nothing to adapt
	s.pick(3)
	w.adaptSourcePassage()
	if s.adapting != nil {
		t.Error("Adapt… on an adapted passage")
	}
}

// The → under a linked passage: its scene's label, going there with the
// scene flashing; two scenes, a menu; a tap on the text only picks.
func TestPassageSceneButton(t *testing.T) {
	script := "Title: G\n\nINT. HOUSE - NIGHT\n\n" + linkOpened + "\n\nINT. GARDEN - NIGHT\n\n" + linkWho + "\n\nINT. LATE - DAY\n\n" + linkOpened + "\n"
	w, _ := sourceWindow(t, script, sourceStory)
	s := w.sourcePane()
	toEnd(w)
	s.follow()
	before := w.textEditor.CursorOffset()
	s.list.Select(3) // a tap: picked, the cursor stays
	if w.textEditor.CursorOffset() != before || s.selected != 3 {
		t.Errorf("a tap moved the cursor (%d, was %d)", w.textEditor.CursorOffset(), before)
	}
	r := rowOf(s, 3)
	if !r.scenes.Visible() || r.scenes.Text != "Scene 2" || fyne.AccessibleLabel(r.scenes) == "" {
		t.Fatalf("→ %q shown %v", r.scenes.Text, r.scenes.Visible())
	}
	r.scenes.OnTapped()
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != lineIn(t, script, "INT. GARDEN") {
		t.Errorf("went to line %d", line)
	}
	if a, b := w.textEditor.Flash(); a != w.textEditor.Buffer().LineStart(lineIn(t, script, "INT. GARDEN")) || b <= a {
		t.Errorf("flash %d–%d", a, b)
	}
	if r := rowOf(s, 2); r.scenes.Text != "2 scenes" {
		t.Errorf("two scenes: %q", r.scenes.Text)
	}
	// Go to the Passage's Scene: the next of them each time
	s.pick(2)
	w.goToPassageScene()
	first, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	w.goToPassageScene()
	second, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset())
	if first != lineIn(t, script, "INT. HOUSE") || second != lineIn(t, script, "INT. LATE") {
		t.Errorf("cycling: %d, %d", first, second)
	}
	// the passage stays the anchor after a jump
	if !strings.Contains(s.anchorLine.Text, "picked") {
		t.Errorf("caption after a jump %q", s.anchorLine.Text)
	}
}

// The setting Next Source Passage Starts a Scene went: a settings file
// with it is read without it, and written again.
func TestRetiredNextSetting(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	os.MkdirAll(getConfigDir(), 0o755)
	os.WriteFile(getConfigDir()+"/settings.json", []byte(`{"next-passage-inserts": false, "font-size": 15}`), 0o644)
	s := &Settings{data: map[string]interface{}{}}
	s.load()
	if _, ok := s.data["next-passage-inserts"]; ok || !s.dirty {
		t.Errorf("kept: %v, dirty %v", s.data["next-passage-inserts"], s.dirty)
	}
	if s.GetInt("font-size") != 15 {
		t.Error("the rest not read")
	}
}

// adaptNew adapts passage p as a scene of its own in the story's order
// (Adapt… with those chosen).
func adaptNew(s *sourcePane, p int) {
	s.pick(p)
	s.adaptPassage(p)
	d := s.adapting
	d.how.choose(0)
	d.where.choose(0)
	d.accept()
}

// Everything Tab reaches in Adapt… shows its focus: How, the heading,
// Where, what goes in, the buttons.
func TestAdaptFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race")
	}
	w, _ := sourceWindow(t, nextScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(1100, 1600)) // all of it in sight
	s := w.sourcePane()
	toEnd(w)
	s.pick(3)
	w.adaptSourcePassage()
	s.adapting.how.choose(0) // the heading and the places shown
	if u := invisibleFocus(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("Adapt…: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
}

// TestAdaptScreenshots: the sample's source pane with the → under its
// passages and the caption, Next's pick with Adapt… under it, the Adapt…
// dialog and the phone's sheet (SHOT_DIR as in screenshot_test.go).
func TestAdaptScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 700)}, {"phone", fyne.NewSize(380, 800)}} {
		w := sampleWindow(t, sampleAccolade)
		w.fyneWindow.Resize(size.size)
		s := sampleSource(w)
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(3))
		s.follow()
		shotTo(t, w, "adapt-chips-"+size.name)
		w.nextSourcePassage()
		s.follow()
		shotTo(t, w, "adapt-next-"+size.name)
		w.adaptSourcePassage()
		shotTo(t, w, "adapt-dialog-"+size.name)
		s.adapting.how.choose(1) // a draft
		shotTo(t, w, "adapt-draft-"+size.name)
		s.adapting.close()
	}
}

// shotTo writes the window's picture as SHOT_DIR/name.png.
func shotTo(t *testing.T, w *MainWindow, name string) {
	t.Helper()
	w.fyneWindow.Content().Refresh()
	f, err := os.Create(filepath.Join(os.Getenv("SHOT_DIR"), name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

// With another middle tab in front (the outline), the → goes to the scene
// in the script: the script's tab comes to the front and the scene
// flashes; with the outline in front again, Adapt… says to go to the
// script's tab instead of writing where no one sees.
func TestAdaptScriptAway(t *testing.T) {
	script := "Title: G\n\nINT. HOUSE - NIGHT\n\n" + linkOpened + "\n\nINT. LATE - DAY\n\nStuff.\n"
	w, _ := sourceWindow(t, script, sourceStory)
	s := w.sourcePane()
	s.follow()
	w.openMiddle("outline")
	if w.scriptInMain() {
		t.Fatal("the outline is not in front in the middle")
	}
	rowOf(s, 2).scenes.OnTapped()
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != lineIn(t, script, "INT. HOUSE") {
		t.Errorf("the script's cursor on line %d", line)
	}
	if a, b := w.textEditor.Flash(); !w.scriptInMain() || a >= b {
		t.Error("the script's tab not in front, or no flash on the scene")
	}
	w.showMiddle("outline")
	s.pick(3)
	w.adaptSourcePassage()
	w.nextSourcePassage()
	if s.adapting != nil || w.textEditor.Text() != script || !strings.Contains(w.statusLabel.Text, "its middle tab first") {
		t.Errorf("worked away from the script: %q", w.statusLabel.Text)
	}
}

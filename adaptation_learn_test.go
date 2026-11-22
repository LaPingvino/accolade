package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Learning from what the writer does (adaptation_learn.go), and the
// block's learned sections (adaptation_learned.go).

// A hand-written block is read leniently: keys in any case and other
// words for them, entries on the key's line or below it (indented, as a
// list, or not), several on a line; what is not read is kept, and so are
// the cuts and lines of no section.
func TestLearnedBlockLenient(t *testing.T) {
	inside := strings.Join([]string{
		"A note of my own.",
		"Characters: ANNA = Anna, Annie; TOM = Thomas",
		"  - MISS K → Miss Kovač",
		"  just a word",
		"not characters: Monday, Lord",
		"  Sunday",
		"Places:",
		"the house = INT. KOVAČ HOUSE",
		"* street -> EXT. ROAD",
		"cut:",
		`  "The rain kept falling." (2)`,
		"skipped:",
		`  INT. KITCHEN - NIGHT = "Anna opened the kitchen door."`,
	}, "\n")
	l := parseLearned(inside)
	if got := l.aliasOf("annie"); got != "ANNA" {
		t.Errorf("Annie: %q", got)
	}
	if l.aliasOf("Thomas") != "TOM" || l.aliasOf("Miss Kovač") != "MISS K" {
		t.Errorf("aliases %+v", l.Aliases)
	}
	if !l.isNot("monday") || !l.isNot("Lord") || !l.isNot("Sunday") || l.isNot("Anna") {
		t.Errorf("not %q", l.Not)
	}
	if l.placeFor("HOUSE") != "INT. KOVAČ HOUSE" || l.placeFor("the street") != "EXT. ROAD" {
		t.Errorf("places %+v", l.Places)
	}
	if len(l.Skipped) != 1 || l.Skipped[0].Heading != "INT. KITCHEN - NIGHT" || l.Skipped[0].Link.Quote != "Anna opened the kitchen door." {
		t.Errorf("skipped %+v", l.Skipped)
	}
	// written again: Accolade's form, the rest kept
	out := withLearned(inside, l)
	want := strings.Join([]string{
		"A note of my own.",
		"characters:",
		"  ANNA = Anna, Annie",
		"  TOM = Thomas",
		"  MISS K = Miss Kovač",
		"  just a word",
		"not characters: Monday, Lord, Sunday",
		"places:",
		"  the house = INT. KOVAČ HOUSE",
		"  street = EXT. ROAD",
		"cut:",
		`  "The rain kept falling." (2)`,
		"skipped links:",
		`  INT. KITCHEN - NIGHT = "Anna opened the kitchen door."`,
	}, "\n")
	if out != want {
		t.Errorf("written:\n%s\nwant:\n%s", out, want)
	}
	// and read the same
	if again := parseLearned(out); !slices.EqualFunc(again.Aliases, l.Aliases, func(a, b aliasEntry) bool {
		return a.Cue == b.Cue && slices.Equal(a.Names, b.Names)
	}) || !slices.Equal(again.Not, l.Not) || !slices.Equal(again.Places, l.Places) || len(again.Skipped) != 1 {
		t.Errorf("round trip: %+v", again)
	}
	if withLearned(out, parseLearned(out)) != out {
		t.Error("written twice, not the same")
	}
	if cuts := parseCuts(out); len(cuts) != 1 || cuts[0].Nth != 2 {
		t.Errorf("cuts %+v", cuts)
	}
	// new sections go in the block's order, before the cuts
	l2 := parseLearned("cut:\n  \"The rain\"")
	l2.addNot("Monday")
	l2.setPlace("kitchen", "INT. KOVAČ KITCHEN")
	if got := withLearned("cut:\n  \"The rain\"", l2); got != "not characters: Monday\nplaces: kitchen = INT. KOVAČ KITCHEN\ncut:\n  \"The rain\"" {
		t.Errorf("new sections:\n%s", got)
	}
	// a learned cue renamed: its names go with it
	l3 := parseLearned("characters: ANNIE = Anna, Annie\nTOM = Thomas")
	if !l3.renameCue("ANNIE", "TOM") || l3.aliasOf("Anna") != "TOM" || l3.aliasOf("Thomas") != "TOM" || len(l3.Aliases) != 1 {
		t.Errorf("renamed cue: %+v", l3.Aliases)
	}
	// everything taken away: nothing left
	if got := withLearned("characters: A = B", learned{}); got != "" {
		t.Errorf("emptied: %q", got)
	}
}

// learnedBlock is the script's adaptation block, inside.
func learnedBlock(w *MainWindow) string {
	in, _ := adaptationBlock(w.textEditor.Text())
	return in
}

// body is the script without its tail blocks.
func body(w *MainWindow) string {
	text := w.textEditor.Text()
	return text[:bodyEndByte(text)]
}

// moveTo puts the cursor at the end of the line holding s (the end of
// the script for ""), as a tap there does.
func moveTo(t *testing.T, w *MainWindow, s string) {
	t.Helper()
	if s == "" {
		toEnd(w)
	} else {
		l := lineWith(t, w, s)
		w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineEnd(l), false) })
	}
	w.onCursorChanged()
}

// toScriptEnd puts the cursor on the empty line after the script, before
// its adaptation block (a blank line between).
func toScriptEnd(w *MainWindow) {
	text := w.textEditor.Text()
	_, a := tailBlocks(text)
	if !a.ok {
		toEnd(w)
		return
	}
	at := a.start + 2
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(at, false) })
	w.onCursorChanged()
}

// forgetRow taps the remove button of the learned view's row that says
// text.
func forgetRow(t *testing.T, w *MainWindow, text string) {
	t.Helper()
	v := w.sourcePane().learned
	for _, o := range v.rows.Objects {
		row := o.(*fyne.Container)
		var label *widget.Label
		var del *widget.Button
		for _, c := range row.Objects {
			switch c := c.(type) {
			case *widget.Label:
				label = c
			case *widget.Button:
				del = c
			}
		}
		if label != nil && strings.HasPrefix(label.Text, text) {
			test.Tap(del)
			return
		}
	}
	t.Fatalf("no row %q in %d", text, len(v.rows.Objects))
}

// rowTexts are the learned view's rows.
func rowTexts(w *MainWindow) []string {
	var out []string
	for _, o := range w.sourcePane().learned.rows.Objects {
		for _, c := range o.(*fyne.Container).Objects {
			if l, ok := c.(*widget.Label); ok {
				out = append(out, l.Text)
			}
		}
	}
	return out
}

// A cue Tab put in, renamed: the source's name becomes another name of
// the new cue, once the cursor has left it (half typed, nothing); the
// suggestions then use the cue. Undo takes the rename and the lesson;
// the view's remove button takes the lesson, and it is not learned again.
func TestLearnRenamedCue(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	tabKey(w) // the action
	tabKey(w) // ANNA's line
	if got := tail(w); got != "ANNA\n(whispering)\nWhere were you?" {
		t.Fatalf("taken: %q", got)
	}
	// renamed: the cue's line, its letters taken away and others typed
	moveTo(t, w, "ANNA")
	for range 4 {
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	typeText(w, "ANNIE")
	if b := learnedBlock(w); b != "" {
		t.Fatalf("learned while the cue is written: %q", b)
	}
	before := body(w)
	moveTo(t, w, "Title")
	cursor := w.textEditor.CursorOffset()
	if b := learnedBlock(w); b != "characters: ANNIE = Anna" {
		t.Fatalf("block %q", b)
	}
	if body(w) != before || w.textEditor.CursorOffset() != cursor {
		t.Errorf("the writer's text or cursor changed: %q", body(w))
	}
	if got := rowTexts(w); !slices.Equal(got, []string{"“Anna” is ANNIE"}) {
		t.Errorf("view %q", got)
	}
	// one undo: the rename and the lesson
	w.textEditor.Undo()
	if b := learnedBlock(w); b != "" || strings.Contains(w.textEditor.Text(), "ANNIE") {
		t.Errorf("undone: block %q\n%s", b, w.textEditor.Text())
	}
	w.textEditor.Redo()
	if b := learnedBlock(w); b != "characters: ANNIE = Anna" {
		t.Fatalf("redone: %q", b)
	}
	// used afterwards: Anna's next line comes under ANNIE
	moveTo(t, w, "Where were you?")
	typeText(w, "\n\nANNIE")
	if h := status(w); !strings.HasPrefix(h, "Tab: ANNIE’s line: “No. There is never tea.”") {
		t.Errorf("hint %q", h)
	}
	for range 7 {
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	// taken away in the view: gone, and not learned again
	w.sourcePane().learned.open = true
	forgetRow(t, w, "“Anna” is ANNIE")
	if b := learnedBlock(w); b != "" {
		t.Errorf("forgotten: %q", b)
	}
	if v := w.sourcePane().learned; v.box.Visible() {
		t.Error("the view is there with nothing learned")
	}
	moveTo(t, w, "ANNIE")
	moveTo(t, w, "")
	typeText(w, "\n")
	moveTo(t, w, "Title")
	if b := learnedBlock(w); b != "" {
		t.Errorf("learned again: %q", b)
	}
}

// A cue the writer types over a line the source reads as another name's
// makes that name another of the cue; not when the source's own name for
// them is a cue in the script (the reading was wrong), nor when the cue
// is another speaker's.
func TestLearnTypedCue(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	typeText(w, "THOMAS\nThen I will make some.\n\n")
	if b := learnedBlock(w); b != "characters: THOMAS = Tom" {
		t.Fatalf("block %q", b)
	}
	typeText(w, "THOMAS")
	if h := status(w); !strings.HasPrefix(h, "Tab: THOMAS’s line: “Out. Is there tea?”") {
		t.Errorf("hint %q", h)
	}
	// TOM in the script: Anna over Tom's line is a correction
	w2 := suggestWindow(t, suggestScript+"TOM\nOut. Is there tea?\n\n")
	typeText(w2, "MAID\nThen I will make some.\n\n")
	if b := learnedBlock(w2); b != "" {
		t.Errorf("Tom speaks in the script, still: %q", b)
	}
	// ANNA over Tom's line: Anna's cue
	// a cue Tab put in, corrected to another who speaks too: ANNA still
	// speaks elsewhere, so the line was read as the wrong one's
	script := suggestScript + "ANNA\nNo. There is never tea.\n\n"
	w3 := suggestWindow(t, script)
	w3.showSuggestionHint()
	tabKey(w3)
	tabKey(w3)
	if got := tailOf(w3, script); got != "ANNA\n(whispering)\nWhere were you?" {
		t.Fatalf("taken: %q", got)
	}
	l := len([]rune(script))
	w3.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(l, l+4, "TOM") })
	moveTo(t, w3, "Title")
	if b := learnedBlock(w3); b != "" {
		t.Errorf("a correction, still: %q", b)
	}
	// what was in the script when it opened teaches nothing
	w4 := suggestWindow(t, suggestScript+"THOMAS\nThen I will make some.\n\n")
	moveTo(t, w4, "Title")
	moveTo(t, w4, "THOMAS")
	moveTo(t, w4, "")
	if b := learnedBlock(w4); b != "" {
		t.Errorf("as opened: %q", b)
	}
}

// A name Tab put in as a cue, thrown away twice (the line deleted, or
// the name over it), is no one: the source's reading drops it, and its
// lines come with no cue.
func TestLearnNotCharacter(t *testing.T) {
	w := suggestWindow(t, suggestScript)
	takeAndDelete := func(whole bool) {
		t.Helper()
		toScriptEnd(w)
		w.showSuggestionHint()
		tabKey(w)
		tabKey(w)
		if got := tail(w); got != "ANNA\n(whispering)\nWhere were you?" {
			t.Fatalf("taken: %q", got)
		}
		e := w.textEditor
		r := []rune(e.Text())
		from := len([]rune(suggestScript))
		if !whole { // only the name: the line is action now
			e.Edit(func(b *buffer.Buffer) { b.Replace(from, from+len("ANNA\n(whispering)\n"), "") })
		} else {
			e.Edit(func(b *buffer.Buffer) { b.Replace(from, len(r), "") })
		}
		moveTo(t, w, "Title")
		if !whole {
			e.Edit(func(b *buffer.Buffer) { b.Replace(from, from+len("Where were you?"), "") })
		}
	}
	takeAndDelete(false)
	if b := learnedBlock(w); b != "" {
		t.Fatalf("once: %q", b)
	}
	takeAndDelete(false)
	if b := learnedBlock(w); b != "not characters: Anna" {
		t.Fatalf("twice: %q", b)
	}
	if body(w) != strings.TrimRight(suggestScript, "\n") {
		t.Errorf("the writer's text: %q", body(w))
	}
	// used: her lines are no one's
	toScriptEnd(w)
	typeText(w, "ANNA")
	if h := status(w); strings.Contains(h, "ANNA’s line") {
		t.Errorf("still Anna's: %q", h)
	}
	forgetRow(t, w, "“Anna” is no one")
	if b := learnedBlock(w); b != "" {
		t.Errorf("forgotten: %q", b)
	}
	// a line deleted whole, or reworded under its cue, says nothing about
	// the name, however often
	w2 := suggestWindow(t, suggestScript)
	for range 2 {
		toScriptEnd(w2)
		w2.showSuggestionHint()
		tabKey(w2)
		tabKey(w2)
		e := w2.textEditor
		from := len([]rune(suggestScript))
		e.Edit(func(b *buffer.Buffer) { b.Replace(from, len([]rune(e.Text())), "") })
		moveTo(t, w2, "Title")
		toScriptEnd(w2)
		tabKey(w2)
		tabKey(w2)
		at := from + len([]rune("ANNA\n(whispering)\n"))
		e.Edit(func(b *buffer.Buffer) { b.Replace(at, at+len("Where were you?"), "Where on earth were you?") })
		moveTo(t, w2, "Title")
		e.Edit(func(b *buffer.Buffer) { b.Replace(from, len([]rune(e.Text())), "") })
		moveTo(t, w2, "Title")
	}
	if b := learnedBlock(w2); b != "" {
		t.Errorf("deleted whole and reworded: %q", b)
	}
}

// learnStory has a second scene in the street, in the evening.
const learnStory = suggestStory + `
* * *

That evening the street was quiet again, and the snow was grey.
`

// A heading written for a scene linked to a passage, other than any the
// source's place makes, maps the place to it (without the time); a later
// scene in that place is offered it.
func TestLearnPlace(t *testing.T) {
	w, _ := sourceWindow(t, suggestScript, learnStory)
	toEnd(w)
	typeText(w, "EXT. KOVAČ ROAD - MORNING")
	if b := learnedBlock(w); b != "" {
		t.Fatalf("learned while the heading is written: %q", b)
	}
	typeText(w, "\n\n")
	if b := learnedBlock(w); b != "" {
		t.Fatalf("no link yet, still: %q", b)
	}
	before := body(w)
	typeText(w, `[[from: "The next morning the street was white"]]`)
	typeText(w, "\n\n")
	if b := learnedBlock(w); b != "places: street = EXT. KOVAČ ROAD" {
		t.Fatalf("block %q", b)
	}
	if !strings.HasPrefix(body(w), before) {
		t.Errorf("the writer's text changed: %q", body(w))
	}
	typeText(w, "ext")
	if h := status(w); !strings.HasPrefix(h, "Tab: EXT. KOVAČ ROAD - EVENING") {
		t.Errorf("hint %q", h)
	}
	// a heading as the source has it teaches nothing
	w2, _ := sourceWindow(t, suggestScript, learnStory)
	toEnd(w2)
	typeText(w2, "INT. STREET - DAY\n\n[[from: \"The next morning the street was white\"]]\n\n")
	if b := learnedBlock(w2); b != "" {
		t.Errorf("as read: %q", b)
	}
	// Adapt…'s new scene, its placeholder heading written over
	w3, _ := sourceWindow(t, suggestScript, learnStory)
	toEnd(w3)
	w3.nextSourcePassage()
	w3.adaptSourcePassage()
	d := w3.sourcePane().adapting
	d.how.choose(0)
	d.heading.SetText("INT. PLACE - DAY")
	d.accept()
	if sel := w3.textEditor.Buffer().SelectedText(); sel != "PLACE - DAY" {
		t.Fatalf("Adapt… selected %q", sel)
	}
	typeText(w3, "KOVAČ ROAD - MORNING")
	moveTo(t, w3, "Title")
	if b := learnedBlock(w3); b != "places: street = INT. KOVAČ ROAD" {
		t.Errorf("after Adapt…: %q", b)
	}
	forgetRow(t, w, "street: EXT. KOVAČ ROAD")
	if b := learnedBlock(w); b != "" {
		t.Errorf("forgotten: %q", b)
	}
}

// A heading typed above a scene has that scene's links under it until
// the blank line after it is typed: they are not put under it, so it
// teaches nothing (the sample's 1A heading once taught "god", then
// "the timber-yard", its own place). Links the script had when opened
// and links put under a heading since are both another's.
func TestLearnNotFromAnotherScenesLinks(t *testing.T) {
	w, _ := sourceWindow(t, suggestScript, learnStory)
	typeAbove := func(at, heading string) {
		l := lineWith(t, w, at)
		w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.LineStart(l), false) })
		w.onCursorChanged()
		typeText(w, heading+"\n\nA puppy slips out.\n\n")
		moveTo(t, w, "Title")
	}
	typeAbove("INT. KITCHEN - NIGHT", "EXT. GENERAL'S YARD - DAY") // the script's links
	if b := learnedBlock(w); b != "" {
		t.Fatalf("above the kitchen: %q", b)
	}
	toEnd(w)
	typeText(w, "EXT. KOVAČ ROAD - MORNING\n\n[[from: \"The next morning the street was white\"]]\n\n")
	moveTo(t, w, "Title")
	if b := learnedBlock(w); b != "places: street = EXT. KOVAČ ROAD" {
		t.Fatalf("the road: %q", b)
	}
	typeAbove("EXT. KOVAČ ROAD - MORNING", "EXT. KENNEL - DAY") // links put under a heading now
	if b := learnedBlock(w); b != "places: street = EXT. KOVAČ ROAD" {
		t.Errorf("above the road: %q", b)
	}

	// a heading renamed does teach: the old one is gone
	w2, _ := sourceWindow(t, suggestScript, learnStory)
	replaceText(t, w2, "INT. KITCHEN - NIGHT", "INT. KOVAČ KITCHEN - NIGHT")
	moveTo(t, w2, "Anna opened the kitchen door")
	moveTo(t, w2, "Title")
	if b := learnedBlock(w2); b != "places: kitchen = INT. KOVAČ KITCHEN" {
		t.Errorf("renamed: %q", b)
	}
	// and Adapt…'s placeholder written over after the cursor was in its scene
	w3, _ := sourceWindow(t, suggestScript, learnStory)
	toEnd(w3)
	w3.nextSourcePassage()
	w3.adaptSourcePassage()
	d := w3.sourcePane().adapting
	d.how.choose(0)
	d.heading.SetText("INT. PLACE - DAY")
	d.accept()
	moveTo(t, w3, "The next morning the street was white")
	l := lineWith(t, w3, "INT. PLACE - DAY")
	w3.textEditor.Navigate(func(b *buffer.Buffer) { b.Select(b.LineStart(l)+5, b.LineEnd(l)) })
	w3.onCursorChanged()
	typeText(w3, "KOVAČ ROAD - MORNING")
	moveTo(t, w3, "Title")
	if b := learnedBlock(w3); b != "places: street = INT. KOVAČ ROAD" {
		t.Errorf("the placeholder written over later: %q", b)
	}
}

// A suggested link skipped is remembered in the block and not offered
// to that scene again; taken away in the view, it is.
func TestLearnSkippedLink(t *testing.T) {
	w, _ := sourceWindow(t, comboNoLinks, comboStory)
	s := w.sourcePane()
	if !strings.Contains(s.prompt.label.Text, "1 of 3: INT. LIGHTHOUSE STAIRS") {
		t.Fatalf("asked %q", s.prompt.label.Text)
	}
	s.prompt.skip.OnTapped()
	if b := learnedBlock(w); b != "skipped links:\n  INT. LIGHTHOUSE STAIRS - NIGHT = #:~:text=Mara climbed the stairs of the old lighthouse,every step out loud.\n"+
		"  INT. LIGHTHOUSE STAIRS - NIGHT = #:~:text=\"Is anyone up there?\" Mara called into the dark." {
		t.Fatalf("block %q", b)
	}
	if body(w) != strings.TrimRight(comboNoLinks, "\n") {
		t.Errorf("the script changed: %q", body(w))
	}
	s.prompt.no.OnTapped()
	w.suggestSourceLinks()
	if !strings.Contains(s.prompt.label.Text, "1 of 2: INT. LAMP ROOM") {
		t.Errorf("again: %q", s.prompt.label.Text)
	}
	s.prompt.no.OnTapped()
	s.learned.open = true
	s.learned.layout()
	forgetRow(t, w, "Not linked: INT. LIGHTHOUSE STAIRS - NIGHT, “Mara climbed")
	forgetRow(t, w, "Not linked: INT. LIGHTHOUSE STAIRS - NIGHT, “Is anyone")
	w.suggestSourceLinks()
	if !strings.Contains(s.prompt.label.Text, "1 of 3: INT. LIGHTHOUSE STAIRS") {
		t.Errorf("forgotten: %q", s.prompt.label.Text)
	}
}

// With the source pane hidden (a phone), learning runs on the
// suggestions' reload after typing.
func TestLearnPaneHidden(t *testing.T) {
	w, p := scriptWindow(t, suggestScript)
	os.WriteFile(filepath.Join(filepath.Dir(p), "story.md"), []byte(suggestStory), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	if w.paneInSight("source") {
		t.Fatal("the pane is shown")
	}
	toEnd(w)
	w.showSuggestionHint()
	typeText(w, "THOMAS\nThen I will make some.\n\n")
	if b := learnedBlock(w); b != "characters: THOMAS = Tom" {
		t.Errorf("block %q", b)
	}
}

// A block written by hand is used as well: its names as one, its places.
func TestLearnedByHand(t *testing.T) {
	script := suggestScript + "\n\n/* Adaptation\nCharacters:\n - THOMAS = Tom\nplaces: the street → EXT. KOVAČ ROAD\n*/\n"
	w := suggestWindow(t, script)
	w.textEditor.Navigate(func(b *buffer.Buffer) { b.SetCursor(len([]rune(suggestScript)), false) })
	typeText(w, "THOMAS")
	if h := status(w); !strings.HasPrefix(h, "Tab: THOMAS’s line: “Out. Is there tea?”") {
		t.Errorf("hint %q", h)
	}
	typeText(w, "\n\n")
	for range 2 {
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	for range len("THOMAS") {
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	typeText(w, "ext")
	if h := status(w); !strings.HasPrefix(h, "Tab: EXT. KOVAČ ROAD - MORNING") {
		t.Errorf("heading hint %q", h)
	}
}

// Screenshots of the learned view, open, at a desktop's width and a
// phone's (SHOT_DIR).
func TestLearnedScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("SHOT_DIR not set")
	}
	os.MkdirAll(dir, 0o755)
	script := suggestScript + "/* Adaptation\ncharacters: ANNA = Anna, Annie, Miss Kovač\nnot characters: Monday, Lord\n" +
		"places: the house = INT. KOVAČ HOUSE\nskipped links:\n  INT. KITCHEN - NIGHT = \"The next morning the street was white…\"\n*/\n"
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w := suggestWindow(t, script)
		w.fyneWindow.Resize(size.size)
		s := w.sourcePane()
		s.follow()
		s.learned.open = true
		s.learned.layout()
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "learned-"+size.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
}

// benchLearned is the bench script with a learned block: names, people
// who are no one, places, skipped links.
func benchLearned() (md, script string) {
	md, script = benchFixture()
	var b strings.Builder
	b.WriteString("/* Adaptation\ncharacters:\n")
	for i := range 20 {
		b.WriteString("  CUE" + string(rune('A'+i)) + " = Name" + string(rune('a'+i)) + ", Other" + string(rune('a'+i)) + "\n")
	}
	b.WriteString("not characters: Monday, Lord, Sunday, Christmas\nplaces:\n")
	for i := range 20 {
		b.WriteString("  place" + string(rune('a'+i)) + " = INT. PLACE " + string(rune('A'+i)) + "\n")
	}
	b.WriteString("skipped links:\n  INT. PLACE 3 - DAY = \"nothing like it\"\n")
	return md, strings.Replace(script, "/* Adaptation\n", b.String(), 1)
}

// A keystroke with suggestions on, without and with a learned block
// (BenchmarkSuggestKeystroke's): learning adds nothing to it, it runs on
// the follow after typing pauses (BenchmarkLearnTick).
func BenchmarkLearnedKeystroke(b *testing.B) {
	for _, with := range []bool{false, true} {
		b.Run(map[bool]string{false: "none", true: "learned"}[with], func(b *testing.B) {
			md, script := benchFixture()
			if with {
				md, script = benchLearned()
			}
			w, s, off := benchWindowWith(b, md, script, false)
			w.textEditor.Replace(off, off, "\n")
			w.textEditor.Navigate(func(bf *buffer.Buffer) { bf.SetCursor(off, false) })
			w.suggester().offersFor(s)
			w.showSuggestionHint()
			if h := status(w); !strings.Contains(h, "from the source") {
				b.Fatalf("hint %q", h)
			}
			realTimers(b)
			e := w.textEditor
			i := 0
			for b.Loop() {
				e.Edit(func(bf *buffer.Buffer) {
					if i%2 == 0 {
						bf.Replace(off+1, off+1, "x")
					} else {
						bf.Replace(off+1, off+2, "")
					}
					bf.SetCursor(off, false)
				})
				cancel(&w.followTimer)
				cancel(&w.suggester().loadTimer)
				i++
			}
		})
	}
}

// What learning costs when the follow runs after a pause: the cursor's
// scene looked at, with a learned block, nothing new to learn (the
// edit, and the text it makes, outside the timing).
func BenchmarkLearnTick(b *testing.B) {
	md, script := benchLearned()
	w, s, off := benchWindowWith(b, md, script, false)
	w.suggester().offersFor(s)
	realTimers(b)
	bf := w.textEditor.Buffer()
	before := learnedBlock(w)
	i := 0
	for b.Loop() {
		b.StopTimer()
		if i%2 == 0 {
			bf.Replace(off+1, off+1, "x")
		} else {
			bf.Replace(off+1, off+2, "")
		}
		bf.SetCursor(off, false)
		bf.Text()
		i++
		b.StartTimer()
		w.learnFromWriting()
	}
	if !w.suggester().learn.ran || learnedBlock(w) != before {
		b.Errorf("ran %v, block changed", w.suggester().learn.ran)
	}
}

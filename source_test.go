package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/layout"

	"github.com/LaPingvino/accolade/internal/prose"
)

const sourceStory = `---
title: The Garden
---
# Part One

## Chapter 1

She opened the door to the garden and stepped out into the *rain*.

"Who's there?" Anna called, but nobody answered.

- a list item of its own
- and another

The rain kept falling.

## Chapter 2

The next morning the garden was white with frost, and the door stood open.

The rain kept falling.
`

const sourceScript = `Title: The Garden

INT. HOUSE - NIGHT

Anna at the door. [[from: "She opened the door to the garden and stepped…"]]

EXT. GARDEN - NIGHT

[[from: "“Who’s there?” Anna called, but nobody answered."]]

ANNA
Who's there?

EXT. GARDEN - DAY

[[from: "The next morning the garden was white…"]]
[[from: "The rain kept falling."]]
[[from: "Nothing like this is in the story at all."]]
`

// passageNamed is the index of the passage opening with these words.
func passageNamed(t *testing.T, ps []passage, opening string) int {
	t.Helper()
	for _, p := range ps {
		if strings.HasPrefix(p.Plain, opening) {
			return p.Index
		}
	}
	t.Fatalf("no passage %q in %v", opening, ps)
	return -1
}

// The source is read as passages: headings, paragraphs, each list item;
// its front matter left out.
func TestPassages(t *testing.T) {
	ps := passagesOf(sourceStory)
	var kinds []prose.Kind
	for _, p := range ps {
		kinds = append(kinds, p.Kind)
	}
	want := []prose.Kind{prose.Heading, prose.Heading, prose.Para, prose.Para, prose.Item, prose.Item,
		prose.Para, prose.Heading, prose.Para, prose.Para}
	if !slices.Equal(kinds, want) {
		t.Fatalf("kinds %v, want %v", kinds, want)
	}
	if ps[0].Plain != "Part One" || ps[0].Level != 1 || ps[1].Level != 2 {
		t.Errorf("headings: %+v %+v", ps[0], ps[1])
	}
	if ps[2].Plain != "She opened the door to the garden and stepped out into the rain." || !strings.Contains(ps[2].Markdown, "*rain*") {
		t.Errorf("a paragraph: %+v", ps[2])
	}
	if ps[4].Plain != "a list item of its own" {
		t.Errorf("a list item: %+v", ps[4])
	}
	if got := normWords(`“Who’s  there?” Anna—called…`); !slices.Equal(got, []string{"whos", "there", "anna", "called"}) {
		t.Errorf("normWords: %q", got)
	}
}

// Links are notes [[from: "…"]] in the script, not in a boneyard or the
// notes; the quote is from its first quote mark to its last.
func TestParseLinks(t *testing.T) {
	links := parseLinks(sourceScript + "\n/* [[from: \"in a boneyard\"]] */\n" + notesText(`[[from: "in the notes"]]`))
	var quotes []string
	for _, l := range links {
		quotes = append(quotes, l.Quote)
	}
	want := []string{"She opened the door to the garden and stepped", "“Who’s there?” Anna called, but nobody answered",
		"The next morning the garden was white", "The rain kept falling", "Nothing like this is in the story at all"}
	if !slices.Equal(quotes, want) {
		t.Fatalf("links %q", quotes)
	}
	r := []rune(sourceScript)
	if got := string(r[links[0].Start:links[0].End]); got != `[[from: "She opened the door to the garden and stepped…"]]` || links[0].Line != 4 {
		t.Errorf("where the first is: %q line %d", got, links[0].Line)
	}
	if len(parseLinks("[[a colour note]] [[ FROM: 'Hi there' ]]")) != 1 {
		t.Error("only from: notes are links (in any case)")
	}
}

// Matching: exact, after edits elsewhere, after an edit in the quoted
// words (near), ambiguous, missing.
func TestLinkMatching(t *testing.T) {
	ps := passagesOf(sourceStory)
	match := func(ps []passage, quote string) linkMatch { return matchLink(normWords(quote), ps) }
	door := passageNamed(t, ps, "She opened")

	// exact, whatever the case, quotes and spacing
	if m := match(ps, `SHE  opened the door to the "garden"`); m.Kind != linkFound || m.Passages[0] != door {
		t.Errorf("exact: %+v", m)
	}
	// edits elsewhere: paragraphs changed, added, taken out
	edited := strings.Replace(sourceStory, "Who's there?", "Hello?", 1)
	edited = strings.Replace(edited, "# Part One\n", "# Part One\n\nA new opening paragraph.\n\nAnd another.\n", 1)
	edited = strings.Replace(edited, "- a list item of its own\n", "", 1)
	eps := passagesOf(edited)
	if m := match(eps, "She opened the door to the garden and stepped…"); m.Kind != linkFound || eps[m.Passages[0]].Plain != ps[door].Plain {
		t.Errorf("after edits elsewhere: %+v", m)
	}
	// an edit in the quoted words: a word changed, or one added
	for _, change := range [][2]string{{"stepped out", "walked out"}, {"the door to", "the old door to"}} {
		cps := passagesOf(strings.Replace(sourceStory, change[0], change[1], 1))
		if m := match(cps, "She opened the door to the garden and stepped…"); m.Kind != linkNear || m.Passages[0] != door {
			t.Errorf("after %q: %+v", change, m)
		}
	}
	// too far from any passage: missing, never the nearest anyway
	if m := match(ps, "Nothing like this is in the story at all"); m.Kind != linkMissing {
		t.Errorf("missing: %+v", m)
	}
	if m := match(ps, "She sat"); m.Kind != linkMissing {
		t.Errorf("a short link is not matched nearly: %+v", m)
	}
	// identical paragraphs: ambiguous, both
	if m := match(ps, "The rain kept falling"); m.Kind != linkAmbiguous || len(m.Passages) != 2 {
		t.Errorf("identical paragraphs: %+v", m)
	}
	// a new paragraph opening as the linked one: ambiguous
	twice := strings.Replace(sourceStory, "## Chapter 2\n", "## Chapter 2\n\nShe opened the door to the garden and stepped back in.\n", 1)
	if m := match(passagesOf(twice), "She opened the door to the garden and stepped…"); m.Kind != linkAmbiguous || len(m.Passages) != 2 {
		t.Errorf("a new paragraph with the same opening: %+v", m)
	}
	// two near ones as near: ambiguous
	near := strings.Replace(twice, "stepped out", "went out", 1)
	near = strings.Replace(near, "stepped back", "came back", 1)
	if m := match(passagesOf(near), "She opened the door to the garden and stepped…"); m.Kind != linkAmbiguous {
		t.Errorf("two as near: %+v", m)
	}
}

// A new link quotes enough to be unique: more than its first words if
// another passage opens with them; the words before it if that does not
// do.
func TestFragmentForUnique(t *testing.T) {
	src := "The door was open and the wind came in from the sea.\n\nThe door was open and the wind came in from the hills.\n\nShort.\n\nShort.\n"
	ps := passagesOf(src)
	lm := newLinkMatcher(ps)
	d, unique := fragmentFor(lm, 1, 1)
	if !unique || d != (textDirective{Start: "The door was open and the wind came in from the hills."}) {
		t.Errorf("fragment %+v %v", d, unique)
	}
	if m := lm.matchRef(linkRef{Dir: d}); m.Kind != linkFound || m.Passages[0] != 1 {
		t.Errorf("found as %+v", m)
	}
	d, _ = fragmentFor(newLinkMatcher(passagesOf(sourceStory)), 2, 2)
	if d.Start != "She opened the door to the garden and" || d.End == "" {
		t.Errorf("eight words and the end: %+v", d)
	}
	if d, unique := fragmentFor(lm, 3, 3); !unique || d.Prefix != "Short." || d.Start != "Short." {
		t.Errorf("the second Short. by its prefix: %+v %v", d, unique)
	}
	if note := fragmentNote("", []textDirective{{Start: `a "b" ]] c, d-e & 5%`}}); note != `[[from: #:~:text=a "b" %5D%5D c%2C d%2De %26 5%25]]` {
		t.Errorf("note %q", note)
	}
}

// sourceWindow is a window with story.fountain open and story.md beside
// it, the source shown.
func sourceWindow(t *testing.T, script, story string) (*MainWindow, string) {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir()) // the kept sources
	dir := t.TempDir()
	p := filepath.Join(dir, "story.fountain")
	os.WriteFile(p, []byte(script), 0o644)
	os.WriteFile(filepath.Join(dir, "story.md"), []byte(story), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.togglePane("source")
	if !w.sourcePane().shown() {
		t.Fatal("the source is not shown")
	}
	return w, p
}

// lineWith is the line of the script that holds s.
func lineWith(t *testing.T, w *MainWindow, s string) int {
	t.Helper()
	for i, l := range strings.Split(w.textEditor.Text(), "\n") {
		if strings.Contains(l, s) {
			return i
		}
	}
	t.Fatalf("no %q in the script", s)
	return -1
}

// Layer 4: story.md (or .markdown) beside story.fountain is its source.
func TestSourceBeside(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	dir := filepath.Dir(p)
	if r := w.sourceRef(); r != (sourceRef{Origin: sourceBeside, Path: filepath.Join(dir, "story.md")}) {
		t.Fatalf("source %+v", r)
	}
	s := w.sourcePane()
	if s.title.Text != "story.md · beside the script" || len(s.passages) != 10 {
		t.Errorf("pane: %q, %d passages", s.title.Text, len(s.passages))
	}
	// none beside it: no source, and the pane says how to have one
	lone := filepath.Join(dir, "lone.fountain")
	os.WriteFile(lone, []byte(sourceScript), 0o644)
	w.LoadFile(lone)
	s.follow()
	if w.sourceRef().Origin != sourceNone || len(s.passages) != 0 || s.body.Objects[0] != s.empty {
		t.Errorf("without a source: %+v, %d passages", w.sourceRef(), len(s.passages))
	}
	os.WriteFile(filepath.Join(dir, "lone.markdown"), []byte("Hi."), 0o644)
	if r := w.sourceRef(); r.Path != filepath.Join(dir, "lone.markdown") {
		t.Errorf("beside, .markdown: %+v", r)
	}
	// a rename takes it along
	moved := filepath.Join(dir, "moved.fountain")
	w.sourceAfterRename(p, moved)
	if w.chosenSource(moved) != filepath.Join(dir, "story.md") {
		t.Errorf("after a rename: %q", w.chosenSource(moved))
	}
	// a Markdown document has no source, nor the menu items
	w.setDocumentType(true)
	if w.sourceRef().Origin != sourceNone || slices.Contains(menuLabels(w, "View"), "Source") ||
		slices.Contains(menuLabels(w, "Adaptation"), "Link to Source Passage") || slices.Contains(menuLabels(w, "File"), "Choose Source…") {
		t.Error("a Markdown document with a source")
	}
}

// Layer 3: Choose Source keeps a copy (converted to Markdown) in
// Accolade's folder, remembered for the script; it wins over the file
// beside the script, is edited in place, and Save Source As gets it out.
func TestSourceKept(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	dir := t.TempDir()
	page := filepath.Join(dir, "novel.html")
	os.WriteFile(page, []byte("<h1>The Novel</h1><p>It was a <b>dark</b> night.</p>"), 0o644)
	if err := w.useSource(page); err != nil {
		t.Fatal(err)
	}
	r := w.sourceRef()
	if r.Origin != sourceKept || !strings.HasPrefix(r.Path, keptSourcesDir()) || filepath.Base(r.Path) != "novel.md" {
		t.Fatalf("kept as %+v (in %s)", r, keptSourcesDir())
	}
	if md, _ := w.sourceText(r); md != "# The Novel\n\nIt was a **dark** night.\n" && !strings.Contains(md, "It was a **dark** night.") {
		t.Errorf("kept %q", md)
	}
	os.Remove(page) // the original is not needed any more
	s := w.sourcePane()
	s.follow()
	if s.title.Text != "novel.md · kept by Accolade" || len(s.passages) != 2 {
		t.Errorf("pane: %q, %d passages", s.title.Text, len(s.passages))
	}
	// remembered for the script, after opening another
	other := filepath.Join(filepath.Dir(p), "other.fountain")
	os.WriteFile(other, []byte("INT. X - DAY\n"), 0o644)
	w.LoadFile(other)
	if w.sourceRef().Origin != sourceNone {
		t.Errorf("another script has it: %+v", w.sourceRef())
	}
	w.LoadFile(p)
	if w.sourceRef() != r {
		t.Errorf("not remembered: %+v", w.sourceRef())
	}
	// an unsaved script keeps it until saved, then by its name
	w.NewEmptyFile()
	w.useSource(filepath.Join(filepath.Dir(p), "story.md"))
	unsaved := w.sourceRef()
	if unsaved.Origin != sourceKept {
		t.Fatalf("unsaved: %+v", unsaved)
	}
	saved := filepath.Join(dir, "new.fountain")
	w.sourceAfterRename("", saved)
	if w.chosenSource(saved) != unsaved.Path {
		t.Errorf("after saving: %q", w.chosenSource(saved))
	}
}

// Layer 1: Keep Source in the Script moves it into a /* Source */ block
// at the script's end, before the notes; it wins over the others, is
// edited there, and moves out again into a kept copy. The links and the
// outline do not read it, nor is it printed.
func TestSourceInside(t *testing.T) {
	script := sourceScript + notesText("Call Bram.")
	w, _ := sourceWindow(t, script, sourceStory)
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	text := w.textEditor.Text()
	want := strings.TrimRight(sourceScript, "\n") + "\n\n/* Source\n" + sourceStory + "*/\n" + notesText("Call Bram.") // the blank lines before the notes as they were
	if text != want {
		t.Fatalf("inside:\n%s\nwant\n%s", text, want)
	}
	if w.sourceRef().Origin != sourceInside || notesOf(text) != "Call Bram." {
		t.Errorf("ref %+v, notes %q", w.sourceRef(), notesOf(text))
	}
	s := w.sourcePane()
	s.follow()
	if s.title.Text != "Kept in the script" || len(s.passages) != 10 {
		t.Errorf("pane: %q, %d passages", s.title.Text, len(s.passages))
	}
	if n := len(parseLinks(text)); n != 5 {
		t.Errorf("%d links", n)
	}
	for _, it := range outlineOf(text, nil) {
		if !it.Scene {
			t.Errorf("the source's heading %q in the outline", it.Text)
		}
	}
	for _, l := range layout.Lay(exportJob{}.script(text), nil) {
		if strings.Contains(l.Text(), "Chapter") {
			t.Errorf("the source printed: %q", l.Text())
		}
	}
	if !slices.ContainsFunc(adaptationItems(w), func(it string) bool { return it == "Keep Source in the Script ✓" }) {
		t.Errorf("not checked: %q", adaptationItems(w))
	}
	// edited: in the block; a */ in it is kept apart
	s.toggleEdit()
	s.w.srcDoc.SetText("New */ text.\n")
	s.toggleEdit()
	if _, _, md, _ := sourceBlock(w.textEditor.Text()); md != "New */ text.\n" || !strings.Contains(w.textEditor.Text(), "New * / text.") {
		t.Errorf("edited: %q", w.textEditor.Text())
	}
	// out again: the script as it was, the source kept by Accolade
	if err := w.keepSourceInside(false); err != nil {
		t.Fatal(err)
	}
	if got := w.textEditor.Text(); got != script {
		t.Errorf("out:\n%q\nwant\n%q", got, script)
	}
	r := w.sourceRef()
	if md, _ := w.sourceText(r); r.Origin != sourceKept || md != "New */ text.\n" {
		t.Errorf("out: %+v %q", r, md)
	}
	// in and out at the very end (no notes)
	w.textEditor.SetText(sourceScript)
	w.keepSourceInside(true)
	if !strings.HasSuffix(w.textEditor.Text(), "*/\n") {
		t.Errorf("no new line at the end: %q", w.textEditor.Text())
	}
	w.keepSourceInside(false)
	if w.textEditor.Text() != sourceScript {
		t.Errorf("at the end, out: %q", w.textEditor.Text())
	}
}

// The layers' order: in the script, then kept, then beside.
func TestSourceOrder(t *testing.T) {
	w, p := sourceWindow(t, sourceScript, sourceStory)
	if w.sourceRef().Origin != sourceBeside {
		t.Fatal("not beside")
	}
	kept := filepath.Join(t.TempDir(), "kept.md")
	os.WriteFile(kept, []byte("Kept."), 0o644)
	w.useSource(kept)
	if w.sourceRef().Origin != sourceKept {
		t.Errorf("kept does not win over beside: %+v", w.sourceRef())
	}
	w.keepSourceInside(true)
	if w.sourceRef().Origin != sourceInside {
		t.Errorf("inside does not win: %+v", w.sourceRef())
	}
	if _, _, md, _ := sourceBlock(w.textEditor.Text()); md != "Kept.\n" && md != "Kept." {
		t.Errorf("moved in: %q", md)
	}
	w.textEditor.Undo()
	if w.sourceRef().Origin != sourceKept {
		t.Errorf("undone: %+v", w.sourceRef())
	}
	os.RemoveAll(filepath.Dir(w.sourceRef().Path)) // the kept copy gone: beside again
	if r := w.sourceRef(); r.Origin != sourceBeside || r.Path != filepath.Join(filepath.Dir(p), "story.md") {
		t.Errorf("kept copy gone: %+v", r)
	}
}

// adaptationItems are the Adaptation menu's items, a checked one marked.
func adaptationItems(w *MainWindow) []string {
	var out []string
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "Adaptation" {
			for _, it := range m.Items {
				if it.Checked {
					out = append(out, it.Label+" ✓")
				} else {
					out = append(out, it.Label)
				}
			}
		}
	}
	return out
}

// Link to Source Passage puts the link for the picked passage at the
// cursor: on an empty line as that line, after a line of dialogue on it,
// below a heading.
func TestLinkInserted(t *testing.T) {
	script := "INT. HOUSE - NIGHT\n\nShe looks out.\n\n\n\nANNA\nWho's there?\n"
	w, _ := sourceWindow(t, script, sourceStory)
	s := w.sourcePane()
	door := passageNamed(t, s.passages, "She opened")
	who := passageNamed(t, s.passages, `"Who's`)

	w.linkToSource()
	if w.textEditor.Text() != script || !strings.Contains(w.statusLabel.Text, "Pick the passage") {
		t.Fatalf("linked with nothing picked: %q", w.statusLabel.Text)
	}
	// an empty line between two others
	s.list.Select(door)
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(4))
	w.linkToSource()
	note := `[[from: #:~:text=She opened the door to the garden and,out into the rain.]]`
	want := "INT. HOUSE - NIGHT\n\nShe looks out.\n\n" + note + "\n\nANNA\nWho's there?\n"
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("on an empty line:\n%q\nwant\n%q", got, want)
	}
	// dialogue: on its line, so it stays with its character
	s.list.Select(who)
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineEnd(lineWith(t, w, "Who's there?")))
	w.linkToSource()
	if l := w.textEditor.Buffer().Line(lineWith(t, w, "Who's there?")); l != `Who's there? [[from: #:~:text="Who's there?" Anna called%2C but nobody answered.]]` {
		t.Errorf("after dialogue: %q", l)
	}
	// a scene heading: below it
	s.list.Select(door)
	w.textEditor.SetCursorOffset(3)
	w.linkToSource()
	if lines := strings.Split(w.textEditor.Text(), "\n"); lines[0] != "INT. HOUSE - NIGHT" || lines[1] != "" || lines[2] != note || lines[3] != "" {
		t.Errorf("below a heading: %q", lines[:4])
	}
	// every link is found again, and lights its passage
	m := s.current()
	for i, mt := range m.matches {
		if mt.Kind != linkFound {
			t.Errorf("link %q: %+v", m.links[i].Quote, mt)
		}
	}
	if !s.lit[door] {
		t.Error("the new link's passage is not lit")
	}
	// the output does not print the links
	for _, l := range layout.Lay(exportJob{}.script(w.textEditor.Text()), nil) {
		if strings.Contains(l.Text(), "from:") {
			t.Errorf("a link printed: %q", l.Text())
		}
	}
}

// The cursor in a scene lights the passages it links to; ambiguous and
// missing links are said, their candidates marked as doubtful.
func TestCursorLightsPassages(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	s := w.sourcePane()
	door := passageNamed(t, s.passages, "She opened")
	who := passageNamed(t, s.passages, `"Who's`)
	morning := passageNamed(t, s.passages, "The next morning")
	at := func(line int) {
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(line))
	}
	at(lineWith(t, w, "Anna at the door"))
	if !sameSet(s.lit, map[int]bool{door: true}) || len(s.doubt) != 0 {
		t.Errorf("INT. HOUSE: lit %v doubt %v", s.lit, s.doubt)
	}
	at(lineWith(t, w, "ANNA"))
	if !sameSet(s.lit, map[int]bool{who: true}) {
		t.Errorf("EXT. GARDEN - NIGHT: lit %v", s.lit)
	}
	at(lineWith(t, w, "EXT. GARDEN - DAY"))
	if !sameSet(s.lit, map[int]bool{morning: true}) || len(s.doubt) != 2 {
		t.Errorf("EXT. GARDEN - DAY: lit %v doubt %v", s.lit, s.doubt)
	}
	for _, said := range []string{"fits 2 passages", "is not in the source", "1 not found"} {
		if !strings.Contains(s.info.Text, said) {
			t.Errorf("info does not say %q:\n%s", said, s.info.Text)
		}
	}
	at(0) // the title page: before any scene, nothing
	if len(s.lit) != 0 {
		t.Errorf("title page: lit %v", s.lit)
	}
	// a source changed on disk is read again
	os.WriteFile(s.ref.Path, []byte("New text.\n"), 0o644)
	s.modTime = s.modTime.Add(-1) // as if it were written later
	at(lineWith(t, w, "Anna at the door"))
	if len(s.passages) != 1 || len(s.lit) != 0 || !strings.Contains(s.info.Text, "is not in the source") {
		t.Errorf("after the source changed: %d passages, lit %v, %q", len(s.passages), s.lit, s.info.Text)
	}
}

// Tapping a passage picks it, the cursor staying; the → under it (Go
// to the Passage's Scene) goes to the scene linking to it, with several
// the next each time.
func TestTapPassageJumps(t *testing.T) {
	script := sourceScript + "\nINT. HOUSE - LATER\n\nAgain. [[from: \"She opened the door to the garden\"]]\n"
	w, _ := sourceWindow(t, script, sourceStory)
	s := w.sourcePane()
	door := passageNamed(t, s.passages, "She opened")
	first, second := lineWith(t, w, "INT. HOUSE - NIGHT"), lineWith(t, w, "INT. HOUSE - LATER")
	w.textEditor.SetCursorOffset(len([]rune(script)))
	s.list.Select(door)
	if got := cursorLine(w); got != strings.Count(script, "\n") || s.selected != door {
		t.Errorf("a tap went to line %d, picked %d", got, s.selected)
	}
	for _, want := range []int{first, second, first} {
		w.goToPassageScene()
		if got := cursorLine(w); got != want {
			t.Errorf("to its scene: line %d, want %d", got, want)
		}
		if s.selected != door {
			t.Errorf("picked %d", s.selected)
		}
	}
	if !s.lit[door] {
		t.Error("its passage is not lit in its scene")
	}
	// a passage no scene links to: picked, the cursor stays
	before := w.textEditor.CursorOffset()
	s.list.Select(passageNamed(t, s.passages, "a list item"))
	if w.textEditor.CursorOffset() != before || !strings.Contains(w.statusLabel.Text, "Adapt…") {
		t.Errorf("unlinked: moved %v, %q", w.textEditor.CursorOffset() != before, w.statusLabel.Text)
	}
}

// The source's headings become sections: all of them at the cursor in a
// script without any, else only those missing, in their places; never
// twice.
func TestSectionsFromSource(t *testing.T) {
	w, _ := sourceWindow(t, "Title: The Garden\n\nINT. HOUSE - NIGHT\n\nRain.\n", sourceStory)
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(2))
	w.sectionsFromSourceHere()
	want := "Title: The Garden\n\n# Part One\n\n## Chapter 1\n\n## Chapter 2\n\nINT. HOUSE - NIGHT\n\nRain.\n"
	if got := w.textEditor.Text(); got != want {
		t.Fatalf("outline:\n%s\nwant\n%s", got, want)
	}
	w.sectionsFromSourceHere()
	if got := w.textEditor.Text(); got != want || !strings.Contains(w.statusLabel.Text, "all in the script") {
		t.Errorf("twice:\n%s", got)
	}
	// some there: the rest in their places
	script := "INT. START - DAY\n\n## Chapter 1\n\nINT. HOUSE - NIGHT\n\nRain.\n\n/* Notes\nkeep\n*/\n"
	w.textEditor.SetText(script)
	w.textEditor.SetCursorOffset(len([]rune(script)))
	w.sectionsFromSourceHere()
	want = "INT. START - DAY\n\n# Part One\n\n## Chapter 1\n\nINT. HOUSE - NIGHT\n\nRain.\n\n## Chapter 2\n\n/* Notes\nkeep\n*/\n"
	if got := w.textEditor.Text(); got != want {
		t.Errorf("in their places:\n%s\nwant\n%s", got, want)
	}
	// one undo step
	w.textEditor.Undo()
	if got := w.textEditor.Text(); got != script {
		t.Errorf("undone:\n%s", got)
	}
}

// Edit opens the source as text in the pane; Done saves it and shows the
// passages again. Everything in the pane has a name.
func TestSourceEdit(t *testing.T) {
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	s := w.sourcePane()
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("no name for %q", u)
	}
	s.toggleEdit()
	if !s.editing || s.body.Objects[0] != s.w.srcDoc.view || s.w.srcDoc.Text() != sourceStory {
		t.Fatal("not editing")
	}
	if u := unnamed(w.fyneWindow.Canvas()); len(u) > 0 {
		t.Errorf("editing: no name for %q", u)
	}
	const only = "# Only\n\nShe opened the door to the garden and stepped out.\n"
	s.w.srcDoc.SetText(only)
	s.toggleEdit()
	data, _ := os.ReadFile(s.ref.Path)
	if s.editing || string(data) != only || len(s.passages) != 2 || w.srcDoc != nil {
		t.Errorf("after Done: editing %v, %d passages, file %q", s.editing, len(s.passages), data)
	}
}

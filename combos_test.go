package main

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor"
)

// Every combination of source and screenplay, as a writer meets it
// first: no source, one beside the script, kept by Accolade, in the
// script, in a bundle, one still being written; no screenplay, an empty
// one, one with no links, one covering part of the source. Each test is
// what the writer sees first and does next; with SHOT_DIR set, the
// first view is rendered at a desktop's and a phone's size
// (combo-NAME-wide.png, combo-NAME-phone.png).

const comboStory = `---
title: The Lighthouse
---
# The Lighthouse

## Chapter 1

Mara climbed the stairs of the old lighthouse, counting every step out loud.

"Is anyone up there?" Mara called into the dark.

"Only the gulls," said Tomas from the lamp room. "Come up, the kettle is on."

## Chapter 2

The storm came in the night and tore the roof from the boathouse.

Tomas stood at the window and watched the fishing boats break loose.

"We have to go down," Mara said. "The boats are ours to save."

## Chapter 3

In the morning the harbour was full of wreckage, and the whole town came down to look.
`

// comboPartial is a script that covers the first chapter.
const comboPartial = `Title: The Lighthouse

## Chapter 1

INT. LIGHTHOUSE STAIRS - NIGHT

[[from: "Mara climbed the stairs of the old lighthouse, counting…"]]

Mara climbs, counting under her breath.

[[from: "“Is anyone up there?” Mara called into the dark."]]

MARA
Is anyone up there?

INT. LAMP ROOM - NIGHT

[[from: "“Only the gulls,” said Tomas from the lamp room."]]

TOMAS
Only the gulls. Come up, the kettle is on.
`

// comboNoLinks is a script written from the story without links.
const comboNoLinks = `Title: The Lighthouse

INT. LIGHTHOUSE STAIRS - NIGHT

Mara climbs the stairs of the old lighthouse, counting every step.

MARA
Is anyone up there?

INT. LAMP ROOM - NIGHT

TOMAS
Only the gulls. Come up, the kettle is on.

EXT. HARBOUR - NIGHT

The storm tears the roof from the boathouse. The fishing boats break loose.

MARA
We have to go down. The boats are ours to save.

INT. PUB - NIGHT

Nobody here has read the story. A darts match, loud and long.
`

// comboShot renders the window's first view at a desktop's width and
// at a phone's, with SHOT_DIR set; the window is left at the desktop's.
func comboShot(t *testing.T, w *MainWindow, name string) {
	t.Helper()
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"wide", fyne.NewSize(1100, 650)}, {"phone", fyne.NewSize(380, 800)}} {
		w.fyneWindow.Resize(size.size)
		if w.source != nil {
			w.source.follow()
		}
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "combo-"+name+"-"+size.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		img := w.fyneWindow.Canvas().Capture()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		if got := img.Bounds().Dx(); got != int(size.size.Width) {
			t.Errorf("%s at %s: the window is %d wide", name, size.name, got)
		}
	}
	w.fyneWindow.Resize(fyne.NewSize(1100, 650))
}

// scriptWindow is a window with story.fountain open, nothing beside it.
func scriptWindow(t *testing.T, script string) (*MainWindow, string) {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	p := filepath.Join(t.TempDir(), "story.fountain")
	os.WriteFile(p, []byte(script), 0o644)
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	return w, p
}

// storyFile is the story written to a file of its own, elsewhere.
func storyFile(t *testing.T, name, story string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	os.WriteFile(p, []byte(story), 0o644)
	return p
}

// textAfter reports whether, in the script, b comes after a (both there).
func textAfter(t *testing.T, w *MainWindow, a, b string) bool {
	t.Helper()
	text := w.textEditor.Text()
	i, j := strings.Index(text, a), strings.Index(text, b)
	if i < 0 || j < 0 {
		t.Fatalf("%q at %d, %q at %d in\n%s", a, i, b, j, text)
	}
	return j > i
}

// selected is the editor's selection.
func selected(w *MainWindow) string {
	e := w.textEditor
	a, b := e.Buffer().Selection()
	r := []rune(e.Text())
	return string(r[min(a, b):max(a, b)])
}

// No source: the pane says what to do (it was blank), with nothing to
// count or go to; a source chosen for an empty script offers to start
// from it, its chapters as sections and the first scene.
func TestComboNoSource(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		w, _ := newDialogTestWindow(t)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		w.createNewFile(true)
		w.togglePane("source")
		s := w.sourcePane()
		if s.body.Objects[0] != s.empty || s.tools.Visible() || s.edit.Visible() {
			t.Fatalf("no source: body %T, tools %v, edit %v", s.body.Objects[0], s.tools.Visible(), s.edit.Visible())
		}
		comboShot(t, w, "none-new")

		// Adaptation > Choose Source… for the empty script
		if err := w.useSource(storyFile(t, "lighthouse.md", comboStory)); err != nil {
			t.Fatal(err)
		}
		w.showSource()
		if !s.prompt.box.Visible() || !strings.Contains(s.prompt.label.Text, "Nothing written yet") {
			t.Fatalf("no offer to start from the source: %q", s.prompt.label.Text)
		}
		comboShot(t, w, "kept-new")
		s.prompt.yes.OnTapped()
		text := w.textEditor.Text()
		for _, want := range []string{"# The Lighthouse", "## Chapter 1", "## Chapter 2", "## Chapter 3", `[[from: #:~:text=Mara,`} {
			if !strings.Contains(text, want) {
				t.Errorf("no %q in\n%s", want, text)
			}
		}
		if !textAfter(t, w, "## Chapter 1", "[[from: #:~:text=Mara,") || textAfter(t, w, "## Chapter 2", "[[from: #:~:text=Mara,") {
			t.Errorf("the first scene is not under Chapter 1:\n%s", text)
		}
		if sel := selected(w); sel == "" || !strings.Contains(text, ". "+sel+"\n") {
			t.Errorf("selected %q", sel)
		}
		// a new script after it does not keep its source
		w.createNewFile(true)
		if r := w.sourceRef(); r.Origin != sourceNone {
			t.Errorf("a new script has the last one's source: %+v", r)
		}
	})
	t.Run("script", func(t *testing.T) {
		w, _ := scriptWindow(t, comboNoLinks)
		w.togglePane("source")
		s := w.sourcePane()
		if s.body.Objects[0] != s.empty || s.tools.Visible() {
			t.Fatalf("no source: body %T, tools %v", s.body.Objects[0], s.tools.Visible())
		}
		comboShot(t, w, "none-script")
	})
}

// Only a source, open as Markdown (opened, or just imported): Start a
// Screenplay makes a script with it as the source, its title page naming
// the story, its chapters as sections, the first scene started.
func TestComboSourceOnly(t *testing.T) {
	t.Run("opened", func(t *testing.T) {
		w, app := newDialogTestWindow(t)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		md := storyFile(t, "lighthouse.md", comboStory)
		if err := w.LoadFile(md); err != nil {
			t.Fatal(err)
		}
		comboShot(t, w, "only-md")
		if !slices.ContainsFunc(w.fyneWindow.MainMenu().Items[0].Items, func(it *fyne.MenuItem) bool { return it.Label == startScreenplayLabel }) {
			t.Error("File has no " + startScreenplayLabel)
		}
		w.startScreenplayFromSource()
		if len(app.windows) != 2 || !w.markdown || w.currentFile != md {
			t.Fatalf("the story is not kept open in its window: %d windows", len(app.windows))
		}
		n := app.windows[1]
		n.fyneWindow.Resize(fyne.NewSize(1100, 650))
		checkStarted(t, n)
		if r := n.sourceRef(); r.Origin != sourceKept || r.Path != absPath(md) || r.describe() != "lighthouse.md · chosen" {
			t.Errorf("the story itself is not the source: %+v", r)
		}
		comboShot(t, n, "only-md-started")
	})
	t.Run("imported", func(t *testing.T) {
		w, app := newDialogTestWindow(t)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		w.showImportedMarkdown("/somewhere/lighthouse.docx", comboStory, "Word", 0)
		if !strings.Contains(w.statusLabel.Text, "Start a Screenplay") {
			t.Errorf("the import does not say what next: %q", w.statusLabel.Text)
		}
		comboShot(t, w, "only-import")
		w.startScreenplayFromSource()
		if len(app.windows) != 1 || w.markdown {
			t.Fatalf("the import is not made the script: %d windows, markdown %v", len(app.windows), w.markdown)
		}
		checkStarted(t, w)
		if r := w.sourceRef(); r.Origin != sourceKept || !strings.HasPrefix(r.Path, keptSourcesDir()) {
			t.Errorf("the import is not kept as the source: %+v", r)
		}
		if d := w.draft(); d.Source == "" {
			t.Error("the browser's draft has no source")
		}
	})
}

// checkStarted checks a script started from comboStory.
func checkStarted(t *testing.T, w *MainWindow) {
	t.Helper()
	text := w.textEditor.Text()
	for _, want := range []string{"Title: The Lighthouse", "Source: Based on “The Lighthouse”", "## Chapter 1", "## Chapter 3"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if !textAfter(t, w, "## Chapter 1", "[[from: #:~:text=Mara,") || textAfter(t, w, "## Chapter 2", "[[from: #:~:text=Mara,") {
		t.Errorf("the first scene is not under Chapter 1:\n%s", text)
	}
	// the heading the story suggests, its words after INT. selected to write over
	if sel := selected(w); sel == "" || !strings.Contains(text, ". "+sel+"\n") || w.suggestedName != "lighthouse.fountain" || !w.sourcePane().shown() {
		t.Errorf("selected %q, name %q, shown %v", selected(w), w.suggestedName, w.sourcePane().shown())
	}
	// a scene for a later chapter goes under its section, wherever the cursor is
	s := w.sourcePane()
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	adaptNew(s, passageNamed(t, s.passages, "The storm came"))
	if !textAfter(t, w, "## Chapter 2", "[[from: #:~:text=The storm,") || textAfter(t, w, "## Chapter 3", "[[from: #:~:text=The storm,") {
		t.Errorf("Chapter 2's scene is not under it:\n%s", w.textEditor.Text())
	}
}

// A script written without links, the source added: each scene the
// source reads like gets a faint "↗ link?" tag after its heading, the one
// from nowhere in the story none; nothing is written until one is
// accepted (one undo step); Not This is remembered; the source pane
// labels the chapters adapted, and Next goes on after the last.
func TestComboNoLinks(t *testing.T) {
	w, _ := sourceWindow(t, comboNoLinks, comboStory)
	s := w.sourcePane()
	ps := s.passages
	if w.textEditor.Text() != comboNoLinks || w.textEditor.Buffer().CanUndo() {
		t.Fatal("the script changed before anything was accepted")
	}
	want := map[string][2]string{
		"INT. LIGHTHOUSE STAIRS - NIGHT": {"Mara climbed", ""},
		"INT. LAMP ROOM - NIGHT":         {"\"Only the gulls", ""},
		"EXT. HARBOUR - NIGHT":           {"The storm came", "\"We have to go"},
	}
	tags := w.guessTags()
	if len(tags) != len(want) {
		t.Errorf("%d tags: %v", len(tags), tags)
	}
	byLine := s.guessesByLine()
	for line, tag := range tags {
		g := byLine[line]
		ws, ok := want[g.Heading]
		if !ok || tag.Label != "↗ link?" || !strings.HasPrefix(ps[g.First].Plain, ws[0]) || ws[1] != "" && !strings.HasPrefix(ps[g.Last].Plain, ws[1]) {
			t.Errorf("line %d: %s %q, %q … %q", line, g.Heading, tag.Label, ps[g.First].Plain, ps[g.Last].Plain)
		}
	}
	// the chapters: the first two adapted (by a guess), the third not yet
	labels := map[string]string{}
	for p, l := range s.up.labelAt {
		labels[ps[p].Plain] = l
	}
	if labels["Chapter 1"] != "adapted? (scenes 1–2)" || labels["Chapter 2"] != "adapted? (scene 3)" || labels["Chapter 3"] != "not adapted yet" {
		t.Errorf("sections %q", labels)
	}
	if os.Getenv("SHOT_DIR") != "" {
		s.scrollToTop(passageNamed(t, ps, "Chapter 1"))
		comboShot(t, w, "linkup-sections")
	}
	// Next goes on after the last scene placed, not from the top
	toEnd(w)
	w.nextSourcePassage()
	if s.selected < 0 || !strings.HasPrefix(ps[s.selected].Plain, "In the morning") {
		t.Errorf("Next went to %d", s.selected)
	}
	comboShot(t, w, "beside-nolinks")
	if os.Getenv("SHOT_DIR") != "" { // the tag's menu, from the keyboard (Alt+L)
		w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "INT. LAMP ROOM")))
		w.linkGuessHere()
		comboShot(t, w, "linkup-menu")
		for o := w.fyneWindow.Canvas().Overlays().Top(); o != nil; o = w.fyneWindow.Canvas().Overlays().Top() {
			w.fyneWindow.Canvas().Overlays().Remove(o)
		}
	}
	// Accept: the link under the heading, one undo step
	stairs := lineWith(t, w, "INT. LIGHTHOUSE STAIRS")
	s.acceptGuess(stairs, byLine[stairs])
	if !strings.Contains(w.textEditor.Text(), "INT. LIGHTHOUSE STAIRS - NIGHT\n\n[[from: #:~:text=Mara,loud.]]\n\nMara climbs") {
		t.Errorf("linked:\n%s", w.textEditor.Text())
	}
	if _, still := w.guessTags()[stairs]; still {
		t.Error("the linked scene keeps its tag")
	}
	if c := countOf(s.statuses); c.Adapted != 1 {
		t.Errorf("count %+v", c)
	}
	labels = map[string]string{}
	for p, l := range s.up.labelAt {
		labels[ps[p].Plain] = l
	}
	if labels["Chapter 1"] != "adapted (scenes 1–2)" {
		t.Errorf("with a link: %q", labels)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != comboNoLinks {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	// Not This: remembered, its tag gone
	lamp := lineWith(t, w, "INT. LAMP ROOM")
	s.notThisGuess(s.guessesByLine()[lamp])
	if _, still := w.guessTags()[lamp]; still || !strings.Contains(learnedBlock(w), "skipped links:\n  INT. LAMP ROOM - NIGHT = ") {
		t.Errorf("Not This: block %q", learnedBlock(w))
	}
	if body(w) != strings.TrimRight(comboNoLinks, "\n") {
		t.Errorf("Not This changed the script's body: %q", body(w))
	}
	w.textEditor.Undo()
	if _, back := w.guessTags()[lamp]; !back || w.textEditor.Text() != comboNoLinks {
		t.Error("undone, the guess is not back")
	}
	// Link Up the Script…: all of them, checked, one undo step
	w.linkUpScript()
	if len(s.up.checks) != 3 || s.up.apply == nil {
		t.Fatalf("Link Up lists %d", len(s.up.checks))
	}
	s.up.apply()
	if n := strings.Count(w.textEditor.Text(), "[[from:"); n != 3 {
		t.Errorf("%d links:\n%s", n, w.textEditor.Text())
	}
	if len(w.guessTags()) != 0 {
		t.Error("tags left after Link Up")
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != comboNoLinks {
		t.Errorf("one undo:\n%s", w.textEditor.Text())
	}
}

// The tag is drawn after the heading, faint, and is not text: the
// cursor goes from the heading's end to the next line, a tap on it opens
// its menu (here: told), the text never changes.
func TestGuessTagNotText(t *testing.T) {
	w, _ := sourceWindow(t, comboNoLinks, comboStory)
	e := w.textEditor
	lamp := lineWith(t, w, "INT. LAMP ROOM")
	found := false
	for _, r := range shownRows(w) {
		if strings.HasSuffix(strings.TrimRight(r, " "), "INT. LAMP ROOM - NIGHT ↗ link?") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no tag drawn: %q", shownRows(w))
	}
	end := e.Buffer().LineEnd(lamp)
	e.SetCursorOffset(end)
	e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got, _ := e.Buffer().Position(e.CursorOffset()); got != lamp+1 {
		t.Errorf("Right from the heading's end: line %d, want %d", got, lamp+1)
	}
	at, cell, ok := e.TagRect(lamp)
	if !ok {
		t.Fatal("no place for the tag")
	}
	tapped := -1
	e.OnTagTapped = func(line int, _ editor.Tag, _ fyne.Position, _ fyne.Size) { tapped = line }
	e.Tapped(&fyne.PointEvent{Position: at.AddXY(cell.Width*2, cell.Height/2)})
	if tapped != lamp {
		t.Errorf("tap on the tag: %d", tapped)
	}
	if e.Text() != comboNoLinks {
		t.Error("the text changed")
	}
}

// R1: a source chosen for a script, its guesses made: the script is as
// it was, byte for byte, not changed, nothing to undo.
func TestLinkUpLeavesScript(t *testing.T) {
	for _, c := range comboOrigins {
		t.Run(c.name, func(t *testing.T) {
			if c.name == "inside" {
				return // keeping it inside is a change asked for
			}
			w := c.open(t, comboNoLinks)
			s := w.sourcePane()
			s.follow()
			if len(s.up.guesses) == 0 || len(w.guessTags()) == 0 {
				t.Fatal("no guesses made")
			}
			if w.textEditor.Text() != comboNoLinks || w.textEditor.Buffer().CanUndo() || w.hasChanges {
				t.Errorf("the script changed (changes %v):\n%s", w.hasChanges, w.textEditor.Text())
			}
		})
	}
}

// A long part of the story before the first adapted chapter is offered
// as cut, once, on a line of its own; Mark as Cut cuts its passages, one
// undo step.
func TestLinkUpOffersCut(t *testing.T) {
	filler := strings.Repeat("A paragraph of the prologue, which the script does not adapt yet.\n\n", 9)
	story := strings.Replace(comboStory, "## Chapter 1\n", "## Prologue\n\n"+filler+"## Chapter 1\n", 1)
	w, _ := sourceWindow(t, comboNoLinks, story)
	s := w.sourcePane()
	if !s.prompt.box.Visible() || s.prompt.label.Text != "Prologue (9 paragraphs) isn't in the script. Mark as Cut?" {
		t.Fatalf("asked %q", s.prompt.label.Text)
	}
	if w.textEditor.Text() != comboNoLinks {
		t.Fatal("asking wrote")
	}
	comboShot(t, w, "linkup-cut")
	s.prompt.yes.OnTapped()
	if c := countOf(s.statuses); c.Cut != 9 {
		t.Errorf("count %+v", c)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != comboNoLinks {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	s.follow()
	if s.prompt.box.Visible() {
		t.Errorf("asked again: %q", s.prompt.label.Text)
	}
}

// comboOrigins open a script with comboStory as its source, kept in
// each of the places a source can be.
var comboOrigins = []struct {
	name string
	open func(t *testing.T, script string) *MainWindow
}{
	{"beside", func(t *testing.T, script string) *MainWindow {
		w, _ := sourceWindow(t, script, comboStory)
		return w
	}},
	{"kept", func(t *testing.T, script string) *MainWindow {
		w, _ := scriptWindow(t, script)
		if err := w.useSource(storyFile(t, "lighthouse.md", comboStory)); err != nil {
			t.Fatal(err)
		}
		w.showSource()
		return w
	}},
	{"inside", func(t *testing.T, script string) *MainWindow {
		w, _ := sourceWindow(t, script, comboStory)
		if err := w.keepSourceInside(true); err != nil {
			t.Fatal(err)
		}
		w.textEditor.SetCursorOffset(0)
		w.sourcePane().follow()
		return w
	}},
	{"bundle", func(t *testing.T, script string) *MainWindow {
		w, _ := newDialogTestWindow(t)
		t.Setenv("XDG_DATA_HOME", t.TempDir())
		p := filepath.Join(t.TempDir(), "Lighthouse.highland")
		items := bundleItems("Lighthouse.textbundle/")
		for i := range items {
			if strings.HasSuffix(items[i].name, "text.fountain") {
				items[i].data = script
			}
		}
		os.WriteFile(p, makeZip(t, items), 0o644)
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		if err := w.useSource(storyFile(t, "lighthouse.md", comboStory)); err != nil {
			t.Fatal(err)
		}
		w.showSource()
		return w
	}},
}

// The source wherever it is kept, a script covering its first chapter:
// the statuses at once, the list on where the adapting goes on, and
// Next, from the top of the script as it opens, goes on after the last
// passage covered, its scene after the scene that covers it.
func TestComboPartial(t *testing.T) {
	for _, c := range comboOrigins {
		t.Run(c.name, func(t *testing.T) {
			w := c.open(t, comboPartial)
			s := w.sourcePane()
			w.textEditor.SetCursorOffset(0)
			s.follow()
			if got := s.count.Text; got != "42% adapted · 0 cut · 4 to go" {
				t.Errorf("count %q", got)
			}
			if s.prompt.box.Visible() {
				t.Errorf("asked %q of a script with links", s.prompt.label.Text)
			}
			comboShot(t, w, c.name+"-partial")
			w.textEditor.SetCursorOffset(0)
			s.stepPassage(1)
			adaptNew(s, s.selected)
			if !textAfter(t, w, "Only the gulls. Come up", "[[from: #:~:text=The storm,") ||
				!textAfter(t, w, "Only the gulls. Come up", "## Chapter 2") || !textAfter(t, w, "## Chapter 2", "[[from: #:~:text=The storm,") {
				t.Errorf("Next from the top, then Adapt…:\n%s", w.textEditor.Text())
			}
			if c.name == "inside" && w.sourceRef().Origin != sourceInside {
				t.Errorf("the source left the script: %+v", w.sourceRef())
			}
		})
	}
}

// The source wherever it is kept, a script with only its title page:
// offered to start from the source, which puts the chapters in as
// sections and starts the first scene, the source where it was.
func TestComboEmptyScript(t *testing.T) {
	for _, c := range comboOrigins {
		t.Run(c.name, func(t *testing.T) {
			w := c.open(t, "Title: The Lighthouse\n")
			s := w.sourcePane()
			if !s.prompt.box.Visible() || !strings.Contains(s.prompt.label.Text, "Nothing written yet") {
				t.Fatalf("not offered to start: %q", s.prompt.label.Text)
			}
			if c.name != "kept" { // kept-new above
				comboShot(t, w, c.name+"-empty")
			}
			origin := w.sourceRef().Origin
			s.prompt.yes.OnTapped()
			if !textAfter(t, w, "Title: The Lighthouse", "## Chapter 1") || !textAfter(t, w, "## Chapter 1", "[[from: #:~:text=Mara,") ||
				textAfter(t, w, "## Chapter 2", "[[from: #:~:text=Mara,") {
				t.Errorf("started:\n%s", w.textEditor.Text())
			}
			if w.sourceRef().Origin != origin {
				t.Errorf("the source moved: %v, was %v", w.sourceRef().Origin, origin)
			}
			if c.name == "inside" && !textAfter(t, w, "## Chapter 3", "/* Source") {
				t.Errorf("the sections are not before the source:\n%s", w.textEditor.Text())
			}
			w.textEditor.Undo()
			w.textEditor.Undo()
			if got := w.textEditor.Text(); !strings.HasPrefix(got, "Title: The Lighthouse\n") || strings.Contains(got[:bodyEndByte(got)], "## Chapter") {
				t.Errorf("two undos:\n%s", got)
			}
		})
	}
}

// The source wherever it is kept, a script without links: guessed.
func TestComboNoLinksOrigins(t *testing.T) {
	for _, c := range comboOrigins {
		t.Run(c.name, func(t *testing.T) {
			w := c.open(t, comboNoLinks)
			s := w.sourcePane()
			s.follow()
			stairs := lineWith(t, w, "INT. LIGHTHOUSE STAIRS")
			g, ok := s.guessesByLine()[stairs]
			if !ok || !g.guessed() {
				t.Fatalf("not guessed: %+v", s.up.guesses)
			}
			s.acceptGuess(stairs, g)
			if c := countOf(s.statuses); c.Adapted != 1 {
				t.Errorf("count %+v in\n%s", c, w.textEditor.Text())
			}
		})
	}
}

// A script just opened, its source long: the list is on where the
// adapting goes on, not at the story's start.
func TestComboScrollsToNext(t *testing.T) {
	filler := strings.Repeat("A paragraph of the prologue, which the script does not adapt yet.\n\n", 40)
	story := strings.Replace(comboStory, "## Chapter 1\n", filler+"## Chapter 1\n", 1)
	w, _ := sourceWindow(t, comboPartial, story)
	s := w.sourcePane()
	if s.list.GetScrollOffset() == 0 {
		t.Error("the list is at the start")
	}
	storm := passageNamed(t, s.passages, "The storm came")
	if !s.inSight(map[int]bool{storm: true}) {
		t.Error("the next passage to adapt is not in sight")
	}
}

// A story in another language (with scene headings Accolade knows)
// starts a script in it: its scene headings, and the status says so.
func TestComboStoryLanguage(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	data, err := os.ReadFile(filepath.Join("internal", "storyread", "testdata", "nl-multatuli-a.md"))
	if err != nil {
		t.Fatal(err)
	}
	story := string(data)
	w.showImportedMarkdown("havelaar.md", story, "Markdown", 0)
	w.startScreenplayFromSource()
	if w.language != "nl" || !strings.Contains(w.statusLabel.Text, "nl (Edit > Script Language…)") {
		t.Errorf("language %q, status %q", w.language, w.statusLabel.Text)
	}
	if start, _ := headingPlaceholder(w.sceneStarts()); start == "INT." || !strings.Contains(w.textEditor.Text(), "\n"+start+" ") {
		t.Errorf("the first scene's heading is not Dutch (%q):\n%s", start, w.textEditor.Text())
	}
}

// A passage under the story's title, before its first chapter (an
// epigraph): its scene goes under the title's section, before Chapter 1.
func TestComboEpigraph(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	story := strings.Replace(comboStory, "## Chapter 1\n", "An old song of the coast, sung by the keepers' wives.\n\n## Chapter 1\n", 1)
	w.showImportedMarkdown("lighthouse.md", story, "Markdown", 0)
	w.startScreenplayFromSource()
	if !textAfter(t, w, "# The Lighthouse", "[[from: #:~:text=An,") || !textAfter(t, w, "[[from: #:~:text=An,", "## Chapter 1") {
		t.Errorf("the epigraph's scene:\n%s", w.textEditor.Text())
	}
}

// A source still being written: what is added later appears and Next
// finds it, the links hold, and a linked passage changed since says so.
func TestComboGrowing(t *testing.T) {
	half := comboStory[:strings.Index(comboStory, "## Chapter 2")]
	w, p := sourceWindow(t, comboPartial, half)
	s := w.sourcePane()
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	s.follow()
	if got := s.count.Text; got != "100% adapted · 0 cut · 0 to go" {
		t.Errorf("count %q", got)
	}
	comboShot(t, w, "growing-partial")
	s.stepPassage(1)
	if s.selected >= 0 && s.statusOf(s.selected) == statusToDo {
		adaptNew(s, s.selected)
	}
	if st := w.statusLabel.Text; !strings.Contains(st, "added to the source later") {
		t.Errorf("Next at the end of a source being written: %q", st)
	}
	// the story goes on, its first paragraph reworded in the words a link
	// quotes, Tomas's after them
	grown := strings.Replace(comboStory, "Mara climbed the stairs of the old lighthouse", "Mara climbed the narrow stairs of the old lighthouse", 1)
	grown = strings.Replace(grown, "the kettle is on", "the teapot is warm", 1)
	story := filepath.Join(filepath.Dir(p), "story.md")
	os.WriteFile(story, []byte(grown), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(story, later, later)
	s.follow()
	c := countOf(s.statuses)
	if c.Adapted != 3 || c.ToDo != 4 {
		t.Errorf("after the story grew: %+v", c)
	}
	if !strings.Contains(s.count.Text, "2 changed since linked") {
		t.Errorf("count %q", s.count.Text)
	}
	first := passageNamed(t, s.passages, "Mara climbed")
	row := newPassageRow()
	s.show(row, first)
	if m := rowParts(row).mark; m.status != statusAdapted || !m.changed {
		t.Errorf("the changed passage's mark: %v %v", m.status, m.changed)
	}
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "Mara climbs")))
	s.follow()
	if !strings.Contains(s.info.Text, "changed since it was linked") {
		t.Errorf("info %q", s.info.Text)
	}
	comboShot(t, w, "growing-changed")
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	s.stepPassage(1)
	if s.selected >= 0 && s.statusOf(s.selected) == statusToDo {
		adaptNew(s, s.selected)
	}
	if !strings.Contains(w.textEditor.Text(), `[[from: #:~:text=The storm,`) {
		t.Errorf("Next did not go on into what was added:\n%s", w.textEditor.Text())
	}
}

// A source with nothing in it yet says how to write it, not a blank list.
func TestComboEmptySource(t *testing.T) {
	w, _ := sourceWindow(t, comboPartial, "---\ntitle: Nothing yet\n---\n")
	s := w.sourcePane()
	if s.body.Objects[0] != s.blank || !s.edit.Visible() {
		t.Fatalf("empty source: body %T", s.body.Objects[0])
	}
	comboShot(t, w, "empty-source")
	s.toggleEdit()
	s.w.srcDoc.SetText("# Chapter 1\n\nThe first line written.\n")
	s.toggleEdit()
	if s.body.Objects[0] == s.blank || len(s.passages) != 2 {
		t.Errorf("written: body %T, %d passages", s.body.Objects[0], len(s.passages))
	}
}

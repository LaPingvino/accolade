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
		if w.notes != nil {
			w.notes.source.follow()
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
		w.toggleSource()
		s := w.sourcePane()
		if s.body.Objects[0] != s.empty || s.tools.Visible() || s.edit.Visible() {
			t.Fatalf("no source: body %T, tools %v, edit %v", s.body.Objects[0], s.tools.Visible(), s.edit.Visible())
		}
		comboShot(t, w, "none-new")

		// File > Choose Source… for the empty script
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
		for _, want := range []string{"# The Lighthouse", "## Chapter 1", "## Chapter 2", "## Chapter 3", `[[from: "Mara climbed`} {
			if !strings.Contains(text, want) {
				t.Errorf("no %q in\n%s", want, text)
			}
		}
		if !textAfter(t, w, "## Chapter 1", "[[from: \"Mara climbed") || textAfter(t, w, "## Chapter 2", "[[from: \"Mara climbed") {
			t.Errorf("the first scene is not under Chapter 1:\n%s", text)
		}
		if selected(w) != "PLACE - DAY" {
			t.Errorf("selected %q", selected(w))
		}
		// a new script after it does not keep its source
		w.createNewFile(true)
		if r := w.sourceRef(); r.Origin != sourceNone {
			t.Errorf("a new script has the last one's source: %+v", r)
		}
	})
	t.Run("script", func(t *testing.T) {
		w, _ := scriptWindow(t, comboNoLinks)
		w.toggleSource()
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
	if strings.Contains(text, "\n# The Lighthouse") {
		t.Errorf("the title is a section too:\n%s", text)
	}
	if !textAfter(t, w, "## Chapter 1", "[[from: \"Mara climbed") || textAfter(t, w, "## Chapter 2", "[[from: \"Mara climbed") {
		t.Errorf("the first scene is not under Chapter 1:\n%s", text)
	}
	if selected(w) != "PLACE - DAY" || w.suggestedName != "lighthouse.fountain" || !w.sourcePane().shown() {
		t.Errorf("selected %q, name %q, shown %v", selected(w), w.suggestedName, w.sourcePane().shown())
	}
	// a scene for a later chapter goes under its section, wherever the cursor is
	s := w.sourcePane()
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
	s.startScene(passageNamed(t, s.passages, "The storm came"))
	if !textAfter(t, w, "## Chapter 2", "[[from: \"The storm came") || textAfter(t, w, "## Chapter 3", "[[from: \"The storm came") {
		t.Errorf("Chapter 2's scene is not under it:\n%s", w.textEditor.Text())
	}
}

// A script written without links, the source added: its scenes' links
// are suggested one scene at a time, Link or Skip; nothing goes in by
// itself, and a scene from nowhere in the story gets none.
func TestComboNoLinks(t *testing.T) {
	w, _ := sourceWindow(t, comboNoLinks, comboStory)
	s := w.sourcePane()
	ps := s.passages
	got := suggestLinks(comboNoLinks, w.sceneStarts(), ps, s.current().status, s.currentReading())
	want := map[string][]string{
		"INT. LIGHTHOUSE STAIRS - NIGHT": {"Mara climbed", "\"Is anyone"},
		"INT. LAMP ROOM - NIGHT":         {"\"Only the gulls"},
		"EXT. HARBOUR - NIGHT":           {"The storm came", "\"We have to go"}, // Tomas at the window: too few words in it
	}
	if len(got) != len(want) {
		t.Errorf("%d scenes suggested: %+v", len(got), got)
	}
	for _, sg := range got {
		var names []string
		for _, p := range sg.Passages {
			names = append(names, ps[p].Plain)
		}
		ws, ok := want[sg.Heading]
		if !ok || len(ws) != len(names) {
			t.Errorf("%s: %q", sg.Heading, names)
			continue
		}
		for i := range ws {
			if !strings.HasPrefix(names[i], ws[i]) {
				t.Errorf("%s: %q, want %q", sg.Heading, names, ws)
			}
		}
	}
	// the pane asked at once, about the first scene; the script as it was
	if !s.prompt.box.Visible() || !strings.Contains(s.prompt.label.Text, "1 of 3: INT. LIGHTHOUSE STAIRS") || !s.prompt.skip.Visible() {
		t.Fatalf("not asked: %q", s.prompt.label.Text)
	}
	if w.textEditor.Text() != comboNoLinks {
		t.Fatal("the script changed before anything was accepted")
	}
	comboShot(t, w, "beside-nolinks")
	s.prompt.yes.OnTapped() // Link
	if !strings.Contains(w.textEditor.Text(), "INT. LIGHTHOUSE STAIRS - NIGHT\n\n[[from: \"Mara climbed the stairs of the old lighthouse,…\"]]\n[[from: \"“Is anyone up there?” Mara called into the…\"]]\n\nMara climbs") {
		t.Errorf("linked:\n%s", w.textEditor.Text())
	}
	if !strings.Contains(s.prompt.label.Text, "2 of 3: INT. LAMP ROOM") {
		t.Errorf("then: %q", s.prompt.label.Text)
	}
	s.prompt.skip.OnTapped() // Skip
	if !strings.Contains(s.prompt.label.Text, "3 of 3: EXT. HARBOUR") || strings.Contains(w.textEditor.Text(), "Only the gulls,”") {
		t.Errorf("after Skip: %q", s.prompt.label.Text)
	}
	s.prompt.no.OnTapped() // Not Now
	if s.prompt.box.Visible() {
		t.Error("Not Now left it")
	}
	if c := countOf(s.statuses); c.Adapted != 2 {
		t.Errorf("count %+v", c)
	}
	w.textEditor.Undo() // the scene's links, one step
	if w.textEditor.Text() != comboNoLinks {
		t.Errorf("undo:\n%s", w.textEditor.Text())
	}
	// asked again from the menu: the scenes still without links
	w.suggestSourceLinks()
	if !strings.Contains(s.prompt.label.Text, "1 of 3") {
		t.Errorf("again: %q", s.prompt.label.Text)
	}
}

// The source wherever it is kept, a script covering its first chapter:
// the statuses at once, and Next, from the top of the script as it
// opens, goes on after the last passage covered, its scene after the
// scene that covers it.
func TestComboPartial(t *testing.T) {
	for _, c := range []struct {
		name string
		make func(t *testing.T) *MainWindow
	}{
		{"beside", func(t *testing.T) *MainWindow {
			w, _ := sourceWindow(t, comboPartial, comboStory)
			return w
		}},
		{"kept", func(t *testing.T) *MainWindow {
			w, _ := scriptWindow(t, comboPartial)
			if err := w.useSource(storyFile(t, "lighthouse.docx.md", comboStory)); err != nil {
				t.Fatal(err)
			}
			w.showSource()
			return w
		}},
		{"inside", func(t *testing.T) *MainWindow {
			w, _ := sourceWindow(t, comboPartial, comboStory)
			if err := w.keepSourceInside(true); err != nil {
				t.Fatal(err)
			}
			return w
		}},
		{"bundle", func(t *testing.T) *MainWindow {
			w, _ := newDialogTestWindow(t)
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			p := filepath.Join(t.TempDir(), "Lighthouse.highland")
			items := bundleItems("Lighthouse.textbundle/")
			for i := range items {
				if strings.HasSuffix(items[i].name, "text.fountain") {
					items[i].data = comboPartial
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
	} {
		t.Run(c.name, func(t *testing.T) {
			w := c.make(t)
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
			s.nextPassage()
			if !textAfter(t, w, "Only the gulls. Come up", "[[from: \"The storm came") ||
				!textAfter(t, w, "Only the gulls. Come up", "## Chapter 2") || !textAfter(t, w, "## Chapter 2", "[[from: \"The storm came") {
				t.Errorf("Next from the top:\n%s", w.textEditor.Text())
			}
			if selected(w) != "PLACE - DAY" {
				t.Errorf("selected %q", selected(w))
			}
		})
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
	s.nextPassage()
	if st := w.statusLabel.Text; !strings.Contains(st, "added to the source later") {
		t.Errorf("Next at the end of a source being written: %q", st)
	}
	// the story goes on, and its first paragraph is reworded
	grown := strings.Replace(comboStory, "Mara climbed the stairs of the old lighthouse", "Mara climbed the narrow stairs of the old lighthouse", 1)
	story := filepath.Join(filepath.Dir(p), "story.md")
	os.WriteFile(story, []byte(grown), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(story, later, later)
	s.follow()
	c := countOf(s.statuses)
	if c.Adapted != 3 || c.ToDo != 4 {
		t.Errorf("after the story grew: %+v", c)
	}
	if !strings.Contains(s.count.Text, "1 changed since linked") {
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
	s.nextPassage()
	if !strings.Contains(w.textEditor.Text(), `[[from: "The storm came`) {
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
	s.editor.SetText("# Chapter 1\n\nThe first line written.\n")
	s.toggleEdit()
	if s.body.Objects[0] == s.blank || len(s.passages) != 2 {
		t.Errorf("written: body %T, %d passages", s.body.Objects[0], len(s.passages))
	}
}

package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// longNote is a note of many lines, long enough to reach past a phone's
// pane.
const longNote = "The neighbour. Knows more than he says.\nHe was there the night the letters stopped.\nHe keeps a key to her flat and has never used it.\nHe wants her to leave the city with him.\nHe will not say so until the last scene."

// detailScroll is the scroll of the character opened.
func detailScroll(t *testing.T, p *charactersPane) *container.Scroll {
	t.Helper()
	if len(p.body.Objects) != 1 {
		t.Fatal("no character opened")
	}
	s, ok := p.body.Objects[0].(*container.Scroll)
	if !ok {
		t.Fatalf("the detail is a %T", p.body.Objects[0])
	}
	return s
}

// strip is the canvas's pixels in r, captured.
func strip(w *MainWindow, r image.Rectangle) []byte {
	img := w.fyneWindow.Canvas().Capture()
	var out []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := img.At(x, y)
			cr, cg, cb, ca := c.RGBA()
			out = append(out, byte(cr>>8), byte(cg>>8), byte(cb>>8), byte(ca>>8))
		}
	}
	return out
}

// A character opened on a phone, their note long: what lies under the
// pane's scroll stays there, drawn nowhere past its edge (the status bar
// below is the same scrolled or not), and a tap below the edge, where a
// field lies out of sight, does not reach it.
func TestCharactersScrollEdge(t *testing.T) {
	w, _ := scriptWindow(t, charsScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	w.showPane("characters", false)
	p := w.charactersPane()
	p.openChar("BEN")
	p.detail.note.SetText(longNote)
	p.detail.fields[fieldAge].SetText("40")
	p.detail.fields[fieldLook].SetText("A grey coat, a cap")
	w.fyneWindow.Content().Refresh()
	s := detailScroll(t, p)
	d := fyne.CurrentApp().Driver()
	at := d.AbsolutePositionForObject(s)
	bottom := at.Y + s.Size().Height
	if bottom >= w.fyneWindow.Canvas().Size().Height-4 {
		t.Fatalf("the scroll reaches the window's bottom (%v): nothing to see past it", bottom)
	}
	// a field under the edge, out of sight
	var hidden *widget.Entry
	for _, f := range p.detail.fields {
		if y := d.AbsolutePositionForObject(f).Y; y > bottom && y < w.fyneWindow.Canvas().Size().Height-8 {
			hidden = f
			break
		}
	}
	if hidden == nil {
		t.Fatal("no field lies past the edge: the scenario does not reach it")
	}
	scale := w.fyneWindow.Canvas().Scale()
	r := image.Rect(int(at.X*scale), int(bottom*scale)+1, int((at.X+s.Size().Width)*scale), int(w.fyneWindow.Canvas().Size().Height*scale))
	before := strip(w, r)
	s.ScrollToOffset(fyne.NewPos(0, 60))
	w.fyneWindow.Content().Refresh()
	if after := strip(w, r); string(after) != string(before) {
		t.Error("what lies under the pane's edge changed with its scroll: the detail draws past the edge")
	}
	s.ScrollToOffset(fyne.NewPos(0, 0))
	hp := d.AbsolutePositionForObject(hidden)
	if hp.Y <= bottom {
		t.Fatal("the field came into sight")
	}
	w.fyneWindow.Canvas().Focus(nil)
	test.TapCanvas(w.fyneWindow.Canvas(), hp.Add(fyne.NewPos(10, 4)))
	if w.fyneWindow.Canvas().Focused() == fyne.Focusable(hidden) {
		t.Error("a tap below the pane's edge reached a field out of sight")
	}
}

// TestCharactersLeftScreenshots renders the Characters items of
// 2026-11-21 (SHOT_DIR): a long note at the scroll's edge, desktop and
// phone.
func TestCharactersLeftScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1100, 560}, {"phone", 380, 800}} {
		w, _ := scriptWindow(t, charsScript)
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.onWindowWidth(size.w)
		snap := func(name string) {
			t.Helper()
			w.fyneWindow.Content().Refresh()
			snapWindow(t, w, filepath.Join(dir, "characters-"+size.name+"-"+name+".png"))
		}
		w.showPane("characters", false)
		p := w.charactersPane()
		p.openChar("BEN")
		p.detail.note.SetText(longNote)
		p.detail.fields[fieldAge].SetText("40")
		p.detail.fields[fieldLook].SetText("A grey coat, a cap")
		snap("5-edge")
		// Rename Everywhere with the notes and the source to choose
		w2 := charsWindow(t, renameScript)
		w2.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w2.onWindowWidth(size.w)
		w2.charactersPane().openChar("ANNA")
		w2.charactersPane().askRename("ANNA")
		w2.fyneWindow.Content().Refresh()
		snapWindow(t, w2, filepath.Join(dir, "characters-"+size.name+"-6-rename-blocks.png"))
		closeOverlays(w2)
		// a Markdown document's character, opened
		w3 := mdCharsWindow(t, mdDoc)
		w3.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w3.onWindowWidth(size.w)
		p3 := w3.charactersPane()
		p3.openChar("ANNA")
		p3.detail.note.SetText("The one who stays")
		p3.detail.fields[fieldAge].SetText("34")
		p3.detail.fields[fieldLook].SetText("A red coat")
		w3.fyneWindow.Content().Refresh()
		snapWindow(t, w3, filepath.Join(dir, "characters-"+size.name+"-7-markdown-notes.png"))
	}
}

// A Markdown document without notes gets its list at its end; one with
// notes, before them; taken out, the text is as it was.
func TestMarkdownListPlace(t *testing.T) {
	for _, doc := range []string{"# A\n\nAnna.\n", "# A\n\nAnna.", "# A\n\nAnna.\n\n<!-- Notes\nn\n-->\n"} {
		s, e, block := markdownList.with(doc, "Anna: x")
		rs := []rune(doc)
		got := string(rs[:s]) + block + string(rs[e:])
		if l := markdownList.list(got); len(l.entries) != 1 || l.entries[0].Note != "x" {
			t.Errorf("%q: no list in %q", doc, got)
		}
		if strings.Contains(doc, "Notes") && (markdownNotes.of(got) != "n" || !strings.HasSuffix(got, "-->\n\n<!-- Notes\nn\n-->\n")) {
			t.Errorf("%q: the notes after it: %q", doc, got)
		}
		if strings.Contains(markdownBody(got), "Characters") {
			t.Errorf("%q: the list in the body", doc)
		}
		s, e, block = markdownList.with(got, "")
		rs = []rune(got)
		if back := string(rs[:s]) + block + string(rs[e:]); strings.TrimRight(back, "\n") != strings.TrimRight(doc, "\n") {
			t.Errorf("%q: taken out: %q", doc, back)
		}
	}
}

// renameScript has the name in its body, a source link to a passage
// with it, a cut entry naming another, and in its source and notes.
const renameScript = `INT. FLAT - NIGHT

Anna reads. [[from: #:~:text=Anna opened the door]]

ANNA
Who is it?

/* Adaptation
cut:
  #:~:text=Anna sat down
*/

/* Source
# The visit

Anna opened the door.

Anna sat down and wept.

Ben left.
*/

/* Notes
Anna should be older.
*/
`

// Rename Everywhere counts the notes and the source apart, and leaves
// them unless asked; asked, the source's links and cut entries follow it
// and still find their passages. One undo step takes it all back.
func TestRenameAlsoBlocks(t *testing.T) {
	w := charsWindow(t, renameScript)
	p := w.charactersPane()
	c, _ := p.charOf("ANNA")
	places := renamePlacesIn(w.textEditor.Text(), false, w.sceneStarts(), "ANNA", "Hanna", renameEverywhere, p.spellings(c))
	if len(places.body) != 2 || len(places.notes) != 1 || len(places.source) != 2 || len(places.links) != 1 {
		t.Fatalf("places: body %d, notes %d, source %d, links %d", len(places.body), len(places.notes), len(places.source), len(places.links))
	}
	notes, source := renameChecks(places)
	if notes == nil || source == nil || notes.Checked || source.Checked ||
		notes.Text != "the notes (1 place)" || source.Text != "the source (2 places)" {
		t.Fatalf("the choices: %v %v", notes, source)
	}
	before := w.textEditor.Text()
	p.renameIn("ANNA", "Hanna", renameEverywhere, renameAlso{})
	got := w.textEditor.Text()
	if !strings.Contains(got, "HANNA\nWho") || !strings.Contains(got, "Hanna reads.") ||
		!strings.Contains(got, "text=Anna opened") || !strings.Contains(got, "Anna should be older") || !strings.Contains(got, "\nAnna opened the door.") {
		t.Errorf("the body only:\n%s", got)
	}
	w.undo()
	if w.textEditor.Text() != before {
		t.Fatal("Undo left it renamed")
	}
	p.read()
	src, notesDoc := w.sourceDoc(), w.notesDocument() // open as documents: they follow
	p.renameIn("ANNA", "Hanna", renameEverywhere, renameAlso{notes: true, source: true})
	got = w.textEditor.Text()
	if !strings.Contains(src.Text(), "Hanna opened the door.") || !strings.Contains(notesDoc.Text(), "Hanna should be older") {
		t.Errorf("the open documents did not follow: %q, %q", src.Text(), notesDoc.Text())
	}
	for _, want := range []string{"Hanna reads.", "text=Hanna opened the door]]", "text=Hanna sat down", "\nHanna opened the door.", "Hanna sat down and wept", "Hanna should be older"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "Anna") {
		t.Errorf("an Anna left:\n%s", got)
	}
	_, _, md, _ := sourceBlock(got)
	ps := passagesOf(md)
	links := parseLinks(got)
	if len(links) != 1 || !matchRefIn(links[0].Ref, ps).found() {
		t.Errorf("the link does not find its passage: %+v", links)
	}
	inside, _ := adaptationBlock(got)
	cuts := parseCuts(inside)
	if len(cuts) != 1 || resolveCut(cuts[0], ps).first < 0 {
		t.Errorf("the cut entry does not find its passage: %+v", cuts)
	}
	w.undo()
	if w.textEditor.Text() != before {
		t.Errorf("one Undo did not take it all back:\n%s", w.textEditor.Text())
	}
}

// Undo of a Merge whose breakdown changed since says that the cast
// stays merged there; with nothing changed it puts it back and says
// nothing of it (TestMergeUndoTags).
func TestMergeUndoSays(t *testing.T) {
	w, _ := scriptWindow(t, charsScript)
	tagNow(t, w, "BEN", 0, "Cast")
	tagNow(t, w, "Anna", 0, "Cast")
	w.showPane("characters", false)
	w.charactersPane().rename("BEN", "ANNA", renameCues)
	w.undo()
	if s := w.statusLabel.Text; strings.Contains(s, "left as it is") {
		t.Errorf("nothing changed since, yet: %q", s)
	}
	w.redo()
	tagNow(t, w, "window", 0, "Props") // the breakdown changed since the merge
	w.undo()
	if s := w.statusLabel.Text; !strings.Contains(s, "Merge undone in the script only") || !strings.Contains(s, "the breakdown changed since") {
		t.Errorf("status %q", s)
	}
	tagNow(t, w, "reads", 0, "Props")
	w.redo()
	if s := w.statusLabel.Text; !strings.Contains(s, "Merge redone in the script only") {
		t.Errorf("status after Redo %q", s)
	}
}

const mdDoc = "# The visit\n\nAnna opened the door. \"Who is it?\" Anna asked.\n\n\"Only me,\" said Ben.\n\nAnna laughed, and Ben came in. Anna made tea while Ben talked about the rain.\n\n\"You look tired,\" Anna said.\n\n\"I am,\" said Ben.\n\n<!-- Notes\nAnna is older here.\n-->\n"

// mdCharsWindow is a window with a Markdown document and the Characters pane.
func mdCharsWindow(t *testing.T, doc string) *MainWindow {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "visit.md")
	os.WriteFile(path, []byte(doc), 0o644)
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	w.showPane("characters", false)
	return w
}

// A Markdown document's characters have notes and fields, kept in a
// comment before its notes: not read as its text (no mentions, not
// shown, not printed), the notes still the notes; a name only in the
// list is listed; a rename renames the entry; the document type carries
// the list over.
func TestMarkdownCharacterNotes(t *testing.T) {
	w := mdCharsWindow(t, mdDoc)
	p := w.charactersPane()
	if p.model == nil || p.model.find("ANNA") < 0 {
		t.Fatalf("no Anna: %v", p.model)
	}
	p.openChar("ANNA")
	if p.focusTarget() != fyne.Focusable(p.detail.note) || !p.detail.form.Visible() {
		t.Fatal("no note to type in")
	}
	p.detail.note.SetText("The one who stays")
	p.detail.fields[fieldAge].SetText("34")
	text := w.textEditor.Text()
	want := "\"I am,\" said Ben.\n\n<!-- Characters\nAnna: The one who stays\n  age: 34\n-->\n\n<!-- Notes\nAnna is older here.\n-->\n"
	if !strings.HasSuffix(text, want) {
		t.Fatalf("the list:\n%s", text)
	}
	if markdownNotes.of(text) != "Anna is older here." || strings.Contains(markdownBody(text), "Characters") {
		t.Errorf("notes %q, body %q", markdownNotes.of(text), markdownBody(text))
	}
	w.undo() // one step for the field's typing, one for the note's
	w.undo()
	if w.textEditor.Text() != mdDoc {
		t.Errorf("Undo:\n%s", w.textEditor.Text())
	}
	w.redo()
	w.redo()
	// a name in the list only; the mentions are the text's
	e := w.textEditor
	text = e.Text()
	at := strings.Index(text, "  age: 34\n") + len("  age: 34\n")
	e.SetText(text[:at] + "Clara: the sister\n" + text[at:])
	p.read()
	m := p.model
	ci, ai := m.find("CLARA"), m.find("ANNA")
	if ci < 0 || ai < 0 {
		t.Fatalf("names %q", rowNames(p))
	}
	if got := p.charStats(m.chars[ci]); got != "in the list, not named yet" {
		t.Errorf("Clara: %q", got)
	}
	if n := len(m.chars[ai].Mentions); n != 5 {
		t.Errorf("Anna's mentions %d, want 5 (not the list's or the notes')", n)
	}
	p.rename("ANNA", "Hanna", renameEverywhere)
	text = e.Text()
	if !strings.Contains(text, "<!-- Characters\nHanna: The one who stays") || strings.Count(text, "Hanna") != 6 || !strings.Contains(text, "Anna is older") {
		t.Errorf("renamed:\n%s", text)
	}
	w.setDocumentType(false)
	text = e.Text()
	if l := charListOf(text); len(l.entries) != 2 || l.entries[0].Name != "Hanna" || notesOf(text) != "Anna is older here." {
		t.Errorf("as a screenplay:\n%s", text)
	}
	w.setDocumentType(true)
	if l := markdownList.list(e.Text()); len(l.entries) != 2 {
		t.Errorf("back to Markdown:\n%s", e.Text())
	}
}

// A Markdown document's character has their cast photo, kept beside it.
func TestMarkdownCastPhoto(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "anna.png")
	os.WriteFile(pic, testPicture(t, 200, 200), 0o644)
	doc := filepath.Join(dir, "visit.md")
	os.WriteFile(doc, []byte(mdDoc), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(doc); err != nil {
		t.Fatal(err)
	}
	w.showPane("characters", false)
	p := w.charactersPane()
	old := chooseFileIn
	t.Cleanup(func() { chooseFileIn = old })
	chooseFileIn = func(_ fyne.Window, _ []string, then func(string)) { then(pic) }
	p.choosePhoto("ANNA")
	inside, ok := markdownList.inside(w.textEditor.Text())
	if !ok || !strings.Contains(inside, "photo: visit.assets/anna.png") {
		t.Fatalf("the list: %q", inside)
	}
	p.read()
	c, _ := p.charOf("ANNA")
	if p.photoOf(c, 80) == nil {
		t.Error("the photo is not shown")
	}
}

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

const charsScript = `Title: The Letters

INT. ANNA'S FLAT - NIGHT

Anna reads by the window. BEN knocks.

ANNA
Who is it?

BEN (O.S.)
Only me, Anna.

@McCLANE
Yippee.

EXT. STREET - NIGHT

ANNA
Go home, Ben.

BEN ^
I am home.

ANNIE
Same Anna, another cue.

/* Adaptation
characters: ANNA = Annie
not characters: Monday
*/

/* Characters
ANNA: the lead, thirty-four
  age: 34
  look: a red coat
  She keeps the letters.

zoe
  actor: Ines Ruiz
*/
`

// The block: read as written, written back the same; one entry changed
// is the only one written anew, in Accolade's form.
func TestCharListRoundTrip(t *testing.T) {
	inside := "ANNA: the lead\n  age: 34\n\tlook:   a red coat\n  She keeps the letters.\n\n  Burns them in the end.\n\n# kept as written\nzoe\n  Played By: Ines Ruiz\nBEN"
	l := parseCharList(inside)
	if got := l.inside(l.entries); got != inside {
		t.Fatalf("round trip:\n%q\nwant\n%q", got, inside)
	}
	names := []string{}
	for _, e := range l.entries {
		names = append(names, e.Name)
	}
	if strings.Join(names, "|") != "ANNA|# kept as written|zoe|BEN" {
		t.Errorf("names %q", names)
	}
	a := l.entries[0]
	if a.Note != "the lead\nShe keeps the letters.\n\nBurns them in the end." || a.Fields[fieldAge] != "34" || a.Fields[fieldLook] != "a red coat" {
		t.Errorf("ANNA %+v", a)
	}
	if z := l.entries[2]; z.Fields[fieldActor] != "Ines Ruiz" || z.Note != "" {
		t.Errorf("zoe %+v", z)
	}
	// zoe's age set: only zoe's lines change
	es := slices.Clone(l.entries)
	es[2].Fields[fieldAge] = "9"
	got := l.inside(es)
	want := strings.Replace(inside, "zoe\n  Played By: Ines Ruiz", "zoe\n  age: 9\n  actor: Ines Ruiz", 1)
	if got != want {
		t.Errorf("zoe's age:\n%q\nwant\n%q", got, want)
	}
	// a new entry last; one taken out
	es = append(slices.Clone(l.entries[:3]), charEntry{Name: "小明", from: -1})
	if got := l.inside(es); !strings.HasSuffix(got, "  Played By: Ines Ruiz\n小明") || strings.Contains(got, "BEN") {
		t.Errorf("added, taken out: %q", got)
	}
}

// The block among the others at the end, in any order: each found, the
// body before all of them.
func TestCharactersAmongTailBlocks(t *testing.T) {
	body := "INT. ROOM - DAY\n\nANNA\nHi.\n"
	blocks := []string{
		"/* Characters\nANNA: the lead\n*/",
		"/* Production\nscene numbers: locked, 10A\n*/",
		"/* Adaptation\nnot characters: Monday\n*/",
		"/* Source\n# Story\n\nOnce.\n*/",
	}
	orders := [][]int{{0, 1, 2, 3}, {3, 2, 1, 0}, {1, 0, 3, 2}, {2, 3, 0, 1}}
	for _, o := range orders {
		text := body
		for _, i := range o {
			text += "\n" + blocks[i] + "\n"
		}
		text += "\n/* Notes\nnotes here\n*/\n"
		ts := tailSetOf(text)
		if !ts.chars.ok || ts.chars.inside != "ANNA: the lead" || !ts.prod.ok || !ts.adapt.ok || !ts.src.ok {
			t.Errorf("order %v: %+v", o, ts)
		}
		if got := text[:bodyEndByte(text)]; strings.TrimRight(got, "\n") != strings.TrimRight(body, "\n") {
			t.Errorf("order %v: body %q", o, got)
		}
		if n := listedNames(text); len(n) != 1 || n[0] != "ANNA" {
			t.Errorf("order %v: listed %q", o, n)
		}
	}
}

// The list, for the language rules' list cue rule: names as written,
// matched by their words in capitals; a name added once, before the notes.
func TestAddedName(t *testing.T) {
	text := "INT. ROOM - DAY\n\nben\nHi.\n\n/* Notes\nn\n*/\n"
	if isListed(text, "ben") {
		t.Fatal("listed before")
	}
	s, e, block, ok := addedName(text, "ben (V.O.)")
	if !ok {
		t.Fatal("not added")
	}
	rs := []rune(text)
	text = string(rs[:s]) + block + string(rs[e:])
	if !isListed(text, "BEN") || !isListed(text, "@ben") || listedNames(text)[0] != "ben" {
		t.Errorf("after: %q", text)
	}
	if !strings.Contains(text, "Hi.\n\n/* Characters\nben\n*/\n\n/* Notes") {
		t.Errorf("where: %q", text)
	}
	if _, _, _, ok := addedName(text, "BEN"); ok {
		t.Error("added twice")
	}
}

// Who speaks and who is listed: cues with their extension, @ and ^, the
// learned names merged into their cue, the list's own, the title page
// left out; the most present first.
func TestScreenplayChars(t *testing.T) {
	m := screenplayChars(charsScript, nil)
	var got []string
	for _, c := range m.chars {
		got = append(got, c.Name+":"+itoa(len(c.Scenes))+"/"+itoa(c.Speeches)+"/"+itoa(c.Words))
	}
	want := "ANNA:002/003/010|BEN:002/002/006|McCLANE:001/001/001|zoe:000/000/000"
	if strings.Join(got, "|") != want {
		t.Errorf("chars\n%s\nwant\n%s", strings.Join(got, "|"), want)
	}
	anna := m.chars[m.find("ANNA")]
	if anna.Entry != 0 || len(anna.Aliases) != 1 || anna.Aliases[0] != "Annie" || len(anna.Cues) != 3 {
		t.Errorf("ANNA %+v", anna)
	}
	for _, c := range anna.Cues {
		if s := string([]rune(charsScript)[c.start:c.end]); s != "ANNA" && s != "ANNIE" {
			t.Errorf("cue span %q", s)
		}
	}
	ben := m.chars[m.find("BEN")]
	if s := string([]rune(charsScript)[ben.Cues[0].start:ben.Cues[0].end]); s != "BEN" {
		t.Errorf("BEN's span %q", s)
	}
	if len(m.scenes) != 2 || m.scenes[0].heading != "INT. ANNA'S FLAT - NIGHT" {
		t.Errorf("scenes %+v", m.scenes)
	}
}

// charsWindow is a window with charsScript and the Characters pane.
func charsWindow(t *testing.T, script string) *MainWindow {
	t.Helper()
	w, _ := scriptWindow(t, script)
	w.showPane("characters", false)
	if !w.paneInSight("characters") || w.charactersPane().model == nil {
		t.Fatal("the pane is not shown")
	}
	return w
}

// rowNames are the pane's rows.
func rowNames(p *charactersPane) []string {
	var out []string
	for _, i := range p.shown {
		out = append(out, p.model.chars[i].Name)
	}
	return out
}

// The pane: the rows, the filter, a tap to the next cue and the one
// after, the character opened with its note and fields written into the
// list as typed (one undo step), the scenes.
func TestCharactersPane(t *testing.T) {
	w := charsWindow(t, charsScript)
	p := w.charactersPane()
	if got := strings.Join(rowNames(p), "|"); got != "ANNA|BEN|McCLANE|zoe" {
		t.Fatalf("rows %q", got)
	}
	if !strings.Contains(p.count.Text, "4 characters · 1 in the list only") {
		t.Errorf("count %q", p.count.Text)
	}
	p.filter.SetText("letters") // the note
	if got := strings.Join(rowNames(p), "|"); got != "ANNA" {
		t.Errorf("filtered %q", got)
	}
	p.filter.SetText("")
	// tapped: the next cue, then the one after, then round again
	w.textEditor.SetCursorOffset(0)
	text := w.textEditor.Text()
	var at []string
	for range 4 {
		p.list.Select(0)
		a, b := w.textEditor.Buffer().Selection()
		at = append(at, string([]rune(text)[min(a, b):max(a, b)])+"@"+itoa(lineAt(text, min(a, b))))
	}
	if got := strings.Join(at, " "); got != "ANNA@006 ANNA@017 ANNIE@023 ANNA@006" {
		t.Errorf("next cues %s", got)
	}
	if !strings.Contains(w.statusText(), "ANNA: cue 1 of 3") {
		t.Errorf("status %q", w.statusText())
	}
	// opened: the scenes, a note typed in, a field
	p.openChar("BEN")
	if n := len(p.detail.scenes.Objects); n != 2 {
		t.Errorf("BEN's scenes: %d", n)
	}
	steps := w.textEditor.UndoSteps()
	for _, s := range []string{"T", "Th", "The", "The neighbour"} {
		p.detail.note.SetText(s)
	}
	if n := w.textEditor.UndoSteps() - steps; n != 1 {
		t.Errorf("the note's typing is %d undo steps, want 1", n)
	}
	p.detail.fields[fieldAge].SetText("4")
	p.detail.fields[fieldAge].SetText("40")
	if !strings.Contains(w.textEditor.Text(), "zoe\n  actor: Ines Ruiz\nBEN: The neighbour\n  age: 40\n*/") {
		t.Errorf("BEN in the list: %q", w.textEditor.Text()[strings.Index(w.textEditor.Text(), "/* Characters"):])
	}
	if n := w.textEditor.UndoSteps() - steps; n != 2 {
		t.Errorf("the note and the age are %d undo steps, want 2", n)
	}
	w.textEditor.Undo()
	w.textEditor.Undo()
	if strings.Contains(w.textEditor.Text(), "BEN: ") {
		t.Error("Undo left BEN's entry")
	}
	// the actor is Production's
	w.workAs(roleWriter)
	p.showDetail()
	if slices.Contains(p.detail.form.Items, p.detail.fieldRow[fieldActor]) {
		t.Error("the actor shown as Writer")
	}
	w.workAs(roleProduction)
	p.showDetail()
	if !slices.Contains(p.detail.form.Items, p.detail.fieldRow[fieldActor]) {
		t.Error("the actor hidden as Production")
	}
	p.closeDetail()
	// not a character, and back
	p.setNot("MCCLANE", true)
	if got := strings.Join(rowNames(p), "|"); got != "ANNA|BEN|zoe|McCLANE" || !p.model.chars[p.model.find("MCCLANE")].Not {
		t.Errorf("not a character: %q", got)
	}
	if !strings.Contains(w.textEditor.Text(), "not characters: Monday, McCLANE") {
		t.Errorf("learned: %q", w.textEditor.Text())
	}
	p.setNot("MCCLANE", false)
	// an alias
	p.addAlias("BEN", "Benjamin, Mr. Hale")
	if !strings.Contains(w.textEditor.Text(), "BEN = Benjamin, Mr. Hale") {
		t.Errorf("aliases: %q", w.textEditor.Text())
	}
}

// lineAt is the line of rune offset off.
func lineAt(text string, off int) int { return strings.Count(string([]rune(text)[:off]), "\n") }

// Rename Everywhere: every place the name stands, each in its case, the
// cue's @, extension and ^ kept, the list's entry and the learned notes'
// cue; one undo step, the least edits; Undo puts it all back.
func TestRenameEverywhere(t *testing.T) {
	w := charsWindow(t, charsScript)
	p := w.charactersPane()
	steps := w.textEditor.UndoSteps()
	p.rename("BEN", "Hale", renameEverywhere)
	got := w.textEditor.Text()
	want := strings.NewReplacer("BEN knocks", "HALE knocks", "\nBEN (O.S.)", "\nHALE (O.S.)", "Go home, Ben.", "Go home, Hale.", "\nBEN ^", "\nHALE ^").Replace(charsScript)
	if got != want {
		t.Errorf("renamed:\n%s", got)
	}
	if w.textEditor.UndoSteps()-steps != 1 {
		t.Errorf("%d undo steps", w.textEditor.UndoSteps()-steps)
	}
	if !strings.Contains(w.statusText(), "BEN renamed Hale in 4 places") {
		t.Errorf("status %q", w.statusText())
	}
	if got := strings.Join(rowNames(p), "|"); got != "ANNA|HALE|McCLANE|zoe" {
		t.Errorf("rows %q", got)
	}
	w.textEditor.Undo()
	if w.textEditor.Text() != charsScript {
		t.Error("Undo did not put it back")
	}
	// the list's spelling and the learned cue follow; a heading too
	p.rename("ANNA", "Hanna", renameEverywhere)
	got = w.textEditor.Text()
	for _, s := range []string{"INT. HANNA'S FLAT", "Hanna reads", "Only me, Hanna.", "\nHANNA\nWho", "characters: HANNA = Annie", "/* Characters\nHANNA: the lead", "ANNIE\nSame Hanna"} {
		if !strings.Contains(got, s) {
			t.Errorf("no %q in\n%s", s, got)
		}
	}
	// onto another character's name: refused, Merge is the way
	w.textEditor.SetText(charsScript)
	p.read()
	p.rename("BEN", "Anna", renameEverywhere)
	if w.textEditor.Text() != charsScript || !strings.Contains(w.statusText(), "Merge Into") {
		t.Errorf("renamed onto ANNA: %q", w.statusText())
	}
	// Merge: the cues only, the entries together
	w.textEditor.SetText(charsScript)
	p.read()
	p.rename("ZOE", "ANNA", renameCues) // no cues: the entry goes into ANNA's
	got = w.textEditor.Text()
	if !strings.Contains(got, "/* Characters\nANNA: the lead, thirty-four\n  age: 34\n  look: a red coat\n  actor: Ines Ruiz\n  She keeps the letters.\n*/") {
		t.Errorf("merged entries:\n%s", got[strings.Index(got, "/* Characters"):])
	}
	p.rename("MCCLANE", "BEN", renameCues)
	if got = w.textEditor.Text(); !strings.Contains(got, "\n@BEN\nYippee.") {
		t.Errorf("merged cue: %q", got)
	}
}

// Renaming while a co-writer types: their keys beside a cue and in the
// list are kept, the two scripts the same (the rename is the least edits,
// docs/COLLAB.md).
func TestRenameTogether(t *testing.T) {
	var sb strings.Builder
	for i := range 600 { // a feature's lead: 1200 places, past the Doc's own diff's bound
		sb.WriteString("INT. ROOM " + itoa(i) + " - DAY\n\nBen waits.\n\nBEN\nLine " + itoa(i) + ".\n\nANNA\nYes.\n\n")
	}
	sb.WriteString("/* Characters\nBEN: the neighbour\nANNA\n*/\n")
	start := sb.String()
	a, b, _ := sharedPair(t, start)
	cut := connect(t, a, b)
	textIs(t, b, "b", start)
	cut() // both edit before they hear from each other
	a.showPane("characters", false)
	a.charactersPane().rename("BEN", "HALE", renameEverywhere)
	mid := strings.Index(start, "Line 250.")
	b.textEditor.Edit(func(buf *buffer.Buffer) {
		buf.Replace(len([]rune(start[:mid+4])), len([]rune(start[:mid+4])), " (soft)")
	})
	at := strings.Index(b.textEditor.Text(), "the neighbour") + len("the neighbour")
	b.textEditor.Edit(func(buf *buffer.Buffer) {
		buf.Replace(len([]rune(b.textEditor.Text()[:at])), len([]rune(b.textEditor.Text()[:at])), ", tall")
	})
	connect(t, a, b)
	want := strings.NewReplacer("Ben waits", "Hale waits", "\nBEN\n", "\nHALE\n", "BEN: the neighbour", "HALE: the neighbour, tall").Replace(start)
	want = strings.Replace(want, "Line 250.", "Line (soft) 250.", 1)
	for _, side := range []struct {
		name string
		w    *MainWindow
	}{{"a", a}, {"b", b}} {
		var got string
		ok := false
		for range 200 {
			if got = side.w.textEditor.Text(); got == want {
				ok = true
				break
			}
			uiSleep(25e6)
		}
		if !ok {
			i := 0
			for i < len(got) && i < len(want) && got[i] == want[i] {
				i++
			}
			t.Errorf("%s differs at %d: %q, want %q", side.name, i, got[max(0, i-40):min(len(got), i+60)], want[max(0, i-40):min(len(want), i+60)])
		}
	}
}

// The breakdown follows a rename: the cast member's name, a tag whose
// words or context changed anchored again, Undo and Redo back and forth.
func TestRenameTags(t *testing.T) {
	w, _ := scriptWindow(t, charsScript)
	tagNow(t, w, "BEN", 0, "Cast") // in the action: a cast member who is seen
	tagNow(t, w, "the window", 0, "Props")
	before := found(w)
	if j := strings.Join(before, " "); !strings.Contains(j, "BEN@BEN@") || !strings.Contains(j, "@the window@") {
		t.Fatalf("tagged %q", before)
	}
	w.showPane("characters", false)
	w.charactersPane().rename("BEN", "Hale", renameEverywhere)
	after := found(w)
	if strings.Join(after, " ") != strings.Replace(strings.Join(before, " "), "BEN@BEN@", "HALE@HALE@", 1) {
		t.Errorf("tags\n%q\nwant like\n%q", after, before)
	}
	w.undo()
	if got := found(w); strings.Join(got, " ") != strings.Join(before, " ") {
		t.Errorf("after Undo %q, want %q", got, before)
	}
	w.redo()
	if got := found(w); strings.Join(got, " ") != strings.Join(after, " ") {
		t.Errorf("after Redo %q, want %q", got, after)
	}
}

// In a Markdown document the pane lists the names the text reads as
// someone's; a tap goes to where the name next stands.
func TestCharactersMarkdown(t *testing.T) {
	doc := "# The visit\n\nAnna opened the door. \"Who is it?\" Anna asked.\n\n\"Only me,\" said Ben.\n\nAnna laughed, and Ben came in. Anna made tea while Ben talked about the rain.\n\n\"You look tired,\" Anna said.\n\n\"I am,\" said Ben.\n"
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "visit.md")
	os.WriteFile(path, []byte(doc), 0o644)
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	w.showPane("characters", false)
	p := w.charactersPane()
	if p.model == nil || !p.model.markdown {
		t.Fatal("no Markdown model")
	}
	names := rowNames(p)
	if !slices.Contains(names, "Anna") || !slices.Contains(names, "Ben") {
		t.Fatalf("names %q", names)
	}
	w.textEditor.SetCursorOffset(0)
	p.next("ANNA", 1)
	p.next("ANNA", 1)
	a, _ := w.textEditor.Buffer().Selection()
	if want := len([]rune(doc[:strings.Index(doc, "Anna asked")])); a != want && a != want+4 {
		t.Errorf("second Anna at %d, want %d", a, want)
	}
	p.rename("BEN", "Tom", renameEverywhere)
	if got := w.textEditor.Text(); strings.Contains(got, "Ben") || strings.Count(got, "Tom") != 4 {
		t.Errorf("renamed: %q", got)
	}
}

// On a phone every pane shown is one tab row: those that fit, the one in
// front always among them, the rest in the More menu; the arrow keys
// reach every tab.
func TestDockSheetOneRow(t *testing.T) {
	w, _ := scriptWindow(t, charsScript)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	for _, p := range w.dock.panes {
		w.showPane(p.id, false)
	}
	r := w.dock.sheetRow
	if r == nil {
		t.Fatal("no sheet row")
	}
	w.fyneWindow.Content().Refresh()
	h := w.dock.tabs["outline"].Size().Height
	if r.Container.Size().Height > h+1 {
		t.Errorf("the row is %v high, a tab %v", r.Container.Size().Height, h)
	}
	if r.first >= len(r.items) || !r.more.Visible() {
		t.Fatal("all tabs fit at 380 px: nothing in More")
	}
	vis := w.dock.visible(dockSheet)
	for range len(vis) + 1 {
		front := w.dock.inFront(dockSheet)
		tab := w.dock.tabs[front]
		if !tab.Visible() || tab.Position().X+tab.Size().Width > r.Container.Size().Width+1 {
			t.Errorf("the tab in front, %s, is not on the row", front)
		}
		w.fyneWindow.Canvas().Focus(tab)
		tab.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	}
}

// The keyboard's place is seen in the pane: the filter, the rows' ›, a
// character opened (its buttons and fields); on a phone, the sheet's row
// with every pane shown, its More button and each tab the arrows reach.
func TestCharactersFocusVisible(t *testing.T) {
	if raceDetector {
		t.Skip("pixel capture of focus: checked by the runs without -race (as TestFocusIsVisible)")
	}
	w, _ := scriptWindow(t, charsScript)
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	w.showPane("characters", false)
	c := w.fyneWindow.Canvas()
	if u := invisibleFocus(c); len(u) > 0 {
		t.Errorf("the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	w.fyneWindow.Resize(fyne.NewSize(1000, 1100)) // all of it in sight
	w.charactersPane().openChar("ANNA")
	if u := invisibleFocus(c); len(u) > 0 {
		t.Errorf("opened: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	w.charactersPane().closeDetail()
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	for _, p := range w.dock.panes {
		if p.id != "help" { // its page's links run off the screen (TestDockFocusVisible)
			w.showPane(p.id, false)
		}
	}
	if u := invisibleFocus(c); len(u) > 0 {
		t.Errorf("phone: the focus is not seen on\n  %s", strings.Join(u, "\n  "))
	}
	for range len(w.dock.visible(dockSheet)) {
		tab := w.dock.tabs[w.dock.inFront(dockSheet)]
		c.Focus(tab)
		tab.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		if f, ok := c.Focused().(*dockTab); !ok || !f.Visible() || f.Size().Width == 0 {
			t.Errorf("the arrow went to a tab not on the row: %v", c.Focused())
		}
	}
}

// TestCharactersScreenshots renders the pane (SHOT_DIR): desktop and
// phone, the list and a character opened, a Markdown document's names,
// and the phone's tab row with every pane shown.
func TestCharactersScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "# The visit\n\nAnna opened the door. \"Who is it?\" Anna asked.\n\n\"Only me,\" said Ben.\n\nAnna laughed, and Ben came in. Anna made tea while Ben talked about the rain.\n\n\"You look tired,\" Anna said.\n\n\"I am,\" said Ben.\n"
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1100, 900}, {"phone", 380, 800}} {
		w, _ := scriptWindow(t, charsScript)
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		w.onWindowWidth(size.w)
		snap := func(name string) {
			t.Helper()
			w.fyneWindow.Content().Refresh()
			snapWindow(t, w, filepath.Join(dir, "characters-"+size.name+"-"+name+".png"))
		}
		w.showPane("outline", false)
		w.showPane("characters", false)
		p := w.charactersPane()
		p.readCast(w.textEditor.Text())
		w.textEditor.SetCursorOffset(0)
		p.list.Select(1)
		snap("1-list")
		p.openChar("BEN")
		p.detail.note.SetText("The neighbour. Knows more than he says.")
		p.detail.fields[fieldAge].SetText("40")
		w.hidePane("outline") // the pane alone: the whole of it
		snap("2-opened")
		w.workAs(roleWriter) // as new users start: no cast number, no actor
		p.openChar("ANNA")
		snap("2b-opened-writer")
		w.workAs(roleProduction)
		p.closeDetail()
		for _, q := range w.dock.panes {
			w.showPane(q.id, false)
		}
		w.showPane("characters", false)
		snap("3-all-panes")
		// a Markdown document's names
		path := filepath.Join(t.TempDir(), "visit.md")
		os.WriteFile(path, []byte(doc), 0o644)
		if err := w.LoadFile(path); err != nil {
			t.Fatal(err)
		}
		w.showPane("characters", false)
		p.read()
		snap("4-markdown")
	}
}

// A rename leaves alone what only looks like the name: the word in small
// letters (a character called WILL), the title page, a source link's quote.
func TestRenameLeavesAlone(t *testing.T) {
	script := "Title: Will and Anna\nAuthor: Will Smith\n\nINT. ROOM - DAY\n\nWILL enters. I will go, says Will.\n\n[[from: #:~:text=Will opened the door]]\n\nWILL\nWill you?\n"
	w := charsWindow(t, script)
	p := w.charactersPane()
	p.rename("WILL", "Tom", renameEverywhere)
	want := "Title: Will and Anna\nAuthor: Will Smith\n\nINT. ROOM - DAY\n\nTOM enters. I will go, says Tom.\n\n[[from: #:~:text=Will opened the door]]\n\nTOM\nTom you?\n"
	if got := w.textEditor.Text(); got != want {
		t.Errorf("renamed:\n%s\nwant\n%s", got, want)
	}
}

// Undo of a Merge puts the breakdown back as it was kept: the other
// character's cast member keeps its own name.
func TestMergeUndoTags(t *testing.T) {
	w, _ := scriptWindow(t, charsScript)
	tagNow(t, w, "BEN", 0, "Cast")
	tagNow(t, w, "Anna", 0, "Cast")
	before := found(w)
	w.showPane("characters", false)
	w.charactersPane().rename("BEN", "ANNA", renameCues)
	w.undo()
	if got := found(w); strings.Join(got, " ") != strings.Join(before, " ") {
		t.Errorf("after Undo of the merge %q, want %q", got, before)
	}
	var names []string
	for _, e := range w.breakdownNow().Elements {
		names = append(names, e.Name)
	}
	if slices.Sort(names); strings.Join(names, "|") != "ANNA|BEN" {
		t.Errorf("cast members %q", names)
	}
}

// Swapped with the main area, the pane still goes to the cues: the
// script's cursor moves and the Script tab lights the scene.
func TestCharactersSwapped(t *testing.T) {
	w := charsWindow(t, charsScript)
	p := w.charactersPane()
	w.swapWithMain("characters")
	if w.swapped != "characters" || !w.swappedIsView() {
		t.Fatalf("swapped %q", w.swapped)
	}
	w.textEditor.SetCursorOffset(0)
	p.next("BEN", 1)
	a, _ := w.textEditor.Buffer().Selection()
	if lineAt(w.textEditor.Text(), a) != 9 {
		t.Errorf("the cursor is on line %d", lineAt(w.textEditor.Text(), a))
	}
	w.swapBack()
	if w.swapped != "" {
		t.Error("not put back")
	}
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/collab"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// trSample is a small script for the Table Read's tests: three scenes,
// two speakers, a frame, a boneyard and a note left out.
const trSample = "Title: The Harbour\n\n" +
	"FADE IN:\n\n" +
	"EXT. HARBOUR - DAWN [[blue]]\n\nThe boats come in, *one by one*.\n[[board: assets/boats.png]]\n\n" +
	"ANNA\n(quietly)\nThere he is.\n\n" +
	"/* cut\nBEN\nNot here.\n*/\n\n" +
	"INT. KITCHEN - DAY\n\nBEN\nTea?\n\nANNA (V.O.)\nAlways.\n\n" +
	"INT. HALL - NIGHT [[green]]\n\nThe clock strikes. [[a note]]\n\nCUT TO:\n\n" +
	"/* Characters\nANNA: waits for her brother\n  photo: assets/anna.png\n*/\n"

func trRead(t *testing.T) *trScript {
	t.Helper()
	return readTable(trSample, nil, nil, nil)
}

// The script read: the opening, three scenes, the cast in order, each
// line's kind and speaker; notes, boneyard and markers left out.
func TestTableReadReads(t *testing.T) {
	s := trRead(t)
	if len(s.Scenes) != 4 || s.Scenes[0].Heading != "" || s.Scenes[1].Heading != "EXT. HARBOUR - DAWN" {
		t.Fatalf("scenes %+v", s.Scenes)
	}
	if !slices.Equal(s.Cast, []string{"ANNA", "BEN"}) {
		t.Errorf("cast %v", s.Cast)
	}
	if s.Scenes[1].Color != "blue" || s.Scenes[3].Color != "green" {
		t.Errorf("colours %q %q", s.Scenes[1].Color, s.Scenes[3].Color)
	}
	var kinds []trKind
	var texts []string
	for _, l := range s.Scenes[1].Lines {
		kinds = append(kinds, l.Kind)
		texts = append(texts, l.Text)
	}
	if !slices.Equal(kinds, []trKind{trHeading, trAction, trFrame, trCue, trParen, trSpeech}) {
		t.Errorf("kinds %v (%q)", kinds, texts)
	}
	if texts[1] != "The boats come in, one by one." {
		t.Errorf("action read as %q", texts[1])
	}
	if f := s.Scenes[1].Lines[2]; f.Frame != "assets/boats.png" {
		t.Errorf("frame %+v", f)
	}
	if strings.Contains(strings.Join(texts, "|"), "Not here") {
		t.Error("the boneyard is read")
	}
	if s.Scenes[3].Lines[1].Text != "The clock strikes." {
		t.Errorf("the note is read: %q", s.Scenes[3].Lines[1].Text)
	}
	if s.Scenes[2].Lines[3].Text != "ANNA (V.O.)" || s.Scenes[2].Lines[3].Speaker != "ANNA" {
		t.Errorf("a cue with its extension: %+v", s.Scenes[2].Lines[3])
	}
	// the first cue is marked (the photo goes there), not the later ones
	var firsts []string
	for _, sc := range s.Scenes {
		for _, l := range sc.Lines {
			if l.First {
				firsts = append(firsts, l.Speaker)
			}
		}
	}
	if !slices.Equal(firsts, []string{"ANNA", "BEN"}) {
		t.Errorf("first cues %v", firsts)
	}
	if s.Scenes[1].Rows == 0 || !strings.Contains(trEstimate(s.rowsOf([]int{1, 2, 3})), "pg") {
		t.Errorf("no estimate: rows %d", s.Scenes[1].Rows)
	}
}

// A scene whose frames all have a time is as long as its boards; one
// with some untimed is never shorter than the timed ones; one without
// frames goes by its pages.
func TestTableReadBoardTimes(t *testing.T) {
	text := "INT. A - DAY\n\nShort.\n[[board: a.png, duration: 12]]\n[[board: b.png, duration: 2.5]]\n\n" +
		"INT. B - DAY\n\nShort.\n[[board: c.png, duration: 30]]\n[[board: d.png]]\n\n" +
		"INT. C - DAY\n\nShort.\n[[board: e.png]]\n"
	s := readTable(text, nil, nil, nil)
	if secs, by := s.Scenes[0].secs(); secs != 14.5 || !by {
		t.Errorf("all timed: %v %v", secs, by)
	}
	if secs, by := s.Scenes[1].secs(); secs != 30 || !by || s.Scenes[1].Untimed != 1 {
		t.Errorf("some timed, longer than read: %v %v", secs, by)
	}
	if secs, by := s.Scenes[2].secs(); by || secs <= 0 {
		t.Errorf("untimed: %v %v", secs, by)
	}
	if got := trEstimate(s.Scenes[0].Rows, 14.5); !strings.HasSuffix(got, "~15 s") {
		t.Errorf("estimate %q", got)
	}
	if rows, secs := s.rowsOf([]int{0, 1}); rows == 0 || secs != 44.5 {
		t.Errorf("the pick: %d rows, %v s", rows, secs)
	}
	sec, _ := s.Scenes[2].secs()
	if _, secs := s.rowsOf([]int{0, 2}); secs != 14.5+sec {
		t.Errorf("mixed pick %v", secs)
	}
}

// The pick: a range, a character's scenes, a tag's, together.
func TestTableReadPick(t *testing.T) {
	s := trRead(t)
	tags := &trTags{}
	tags.add("day:0", "Shoot day 1", 3)
	tags.add("day:0", "Shoot day 1", 1)
	for _, c := range []struct {
		p    trPick
		want []int
	}{
		{trPick{To: -1}, []int{0, 1, 2, 3}},
		{trPick{From: 1, To: 2}, []int{1, 2}},
		{trPick{To: -1, Character: "BEN"}, []int{2}},
		{trPick{To: -1, Character: "ANNA"}, []int{1, 2}},
		{trPick{To: -1, Tag: "day:0"}, []int{1, 3}},
		{trPick{To: -1, Tag: "day:0", Character: "ANNA"}, []int{1}},
		{trPick{From: 2, To: -1, Tag: "day:0"}, []int{3}},
	} {
		if got := s.picked(c.p, tags); !slices.Equal(got, c.want) {
			t.Errorf("%+v: %v, want %v", c.p, got, c.want)
		}
	}
	if sc := s.sceneAtLine(s.Scenes[2].Line + 3); sc != 2 {
		t.Errorf("line in scene %d", sc)
	}
}

// Each speaker keeps one tint, by the order they first speak; the
// narrator's lines are marked at the start of each run.
func TestTableReadTintAndNarrator(t *testing.T) {
	s := trRead(t)
	if s.tintOf("ANNA") != 0 || s.tintOf("BEN") != 1 || s.tintOf("NOBODY") != -1 {
		t.Errorf("tints %d %d", s.tintOf("ANNA"), s.tintOf("BEN"))
	}
	var marks []string
	for _, l := range s.Scenes[3].Lines {
		if !l.Narrator {
			t.Errorf("%q not the narrator's", l.Text)
		}
		if l.NarratorStart {
			marks = append(marks, l.Text)
		}
	}
	if !slices.Equal(marks, []string{"INT. HALL - NIGHT"}) {
		t.Errorf("narrator marks %q: one for the run", marks)
	}
	// after a speech the narrator is marked again
	sc := s.Scenes[2]
	if sc.Lines[0].Kind != trHeading || !sc.Lines[0].NarratorStart {
		t.Errorf("heading %+v", sc.Lines[0])
	}
	for _, l := range sc.Lines[1:] {
		if l.Narrator {
			t.Errorf("%q read by the narrator", l.Text)
		}
	}
}

// trWindow is a test window with the sample script, saved beside its
// pictures.
func trWindow(t *testing.T, size fyne.Size) *MainWindow {
	t.Helper()
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "assets", "boats.png"), harbourPicture(t, 320, 180, 1), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "anna.png"), harbourPicture(t, 200, 200, 0.8), 0o644)
	path := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(path, []byte(trSample), 0o644)
	w.fyneWindow.Resize(size)
	if err := w.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	return w
}

// richTexts are the texts of the view's rows.
func (v *tableReadView) shownTexts() []string {
	var out []string
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch o := o.(type) {
		case *widget.RichText:
			out = append(out, o.String())
		case *trLineRow:
			walk(o.body)
		case *fyne.Container:
			for _, c := range o.Objects {
				walk(c)
			}
		}
	}
	walk(v.rows)
	return out
}

// On a desktop the Table Read is a window of its own; Next and Previous
// go through the scenes picked; the script is left as it was.
func TestTableReadWindow(t *testing.T) {
	w := trWindow(t, fyne.NewSize(1200, 800))
	text, changed := w.textEditor.Text(), w.hasChanges
	w.textEditor.SetCursorOffset(len([]rune(trSample[:strings.Index(trSample, "INT. KITCHEN")])))
	cmd("view.tableRead").run(w)
	v := w.tableRead
	if v == nil || v.win == nil || w.mid.has("tableread") {
		t.Fatal("no window of its own")
	}
	if v.scene != 2 {
		t.Errorf("opened at scene %d, not the cursor's", v.scene)
	}
	if !w.cmdEnv().tableReadOpen || w.cmdEnv().tableRead {
		t.Error("open, but no tab while not in the middle")
	}
	cmd("tableread.next").run(w)
	if v.scene != 3 {
		t.Errorf("next: scene %d", v.scene)
	}
	v.key(&fyne.KeyEvent{Name: fyne.KeyLeft})
	v.key(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if v.scene != 1 || !strings.Contains(v.title.Text, "EXT. HARBOUR") {
		t.Errorf("back twice: scene %d, %q", v.scene, v.title.Text)
	}
	got := strings.Join(v.shownTexts(), "|")
	if !strings.Contains(got, "NARRATOR") || !strings.Contains(got, "There he is.") {
		t.Errorf("shown: %s", got)
	}
	cmd("tableread.narrator").run(w)
	if strings.Contains(strings.Join(v.shownTexts(), "|"), "NARRATOR") {
		t.Error("narrator mode off, still marked")
	}
	// the storyboard's frame between the lines, the cast photo on the first cue
	cmd("tableread.storyboard").run(w)
	frames, beside, photos := 0, 0, 0
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch o := o.(type) {
		case *trLineRow:
			walk(o.body)
		case *canvas.Image:
			switch o.MinSize().Width {
			case 400:
				frames++
			case 300:
				beside++
			case 72:
				photos++
			}
		case *fyne.Container:
			for _, c := range o.Objects {
				walk(c)
			}
		}
	}
	walk(v.rows)
	if frames != 1 || photos != 1 {
		t.Errorf("%d frames, %d photos", frames, photos)
	}
	// beside: the frame by the line it follows, not between the lines
	texts := v.shownTexts()
	cmd("tableread.beside").run(w)
	frames, photos = 0, 0
	walk(v.rows)
	if frames != 0 || beside != 1 || !v.boards {
		t.Errorf("beside: %d between, %d beside", frames, beside)
	}
	if got := v.shownTexts(); !slices.Equal(got, texts) {
		t.Errorf("the lines beside: %q, between: %q", got, texts)
	}
	cmd("tableread.beside").run(w)
	// only Ben's scenes
	v.pickCharacter("BEN")
	if !slices.Equal(v.order, []int{2}) || v.scene != 2 {
		t.Errorf("Ben's: %v at %d", v.order, v.scene)
	}
	if v.castRow.Objects[0].(*widget.Button).Importance == widget.HighImportance {
		t.Error("Everyone still chosen")
	}
	if w.textEditor.Text() != text || w.hasChanges != changed {
		t.Error("the Table Read changed the script")
	}
	cmd("tableread.close").run(w)
	if w.tableReadOpen() {
		t.Error("still open")
	}
}

// On a phone (and in the browser) it is a middle tab, with its own
// ribbon tab brought forward; edits to the script show in it.
func TestTableReadMiddle(t *testing.T) {
	w := trWindow(t, fyne.NewSize(380, 800))
	if !w.narrow {
		t.Skip("not narrow")
	}
	cmd("view.tableRead").run(w)
	if w.tableRead.win != nil || !w.tableReadInFront() {
		t.Fatal("not a middle tab")
	}
	if !w.cmdEnv().tableRead {
		t.Error("the Table Read tab's rule says no")
	}
	if w.ribbon != nil && w.ribbonOn() && w.ribbon.selected != "Table Read" {
		t.Errorf("ribbon on %q", w.ribbon.selected)
	}
	w.textEditor.SetText(strings.Replace(trSample, "Tea?", "Coffee?", 1))
	w.tableRead.update()
	w.tableRead.goLine(w.tableRead.script.Scenes[2].Line)
	if got := strings.Join(w.tableRead.shownTexts(), "|"); !strings.Contains(got, "Coffee?") {
		t.Errorf("the edit not read: %s", got)
	}
	cmd("tableread.close").run(w)
	if w.mid.has("tableread") || w.cmdEnv().tableRead {
		t.Error("still in the middle")
	}
}

// TestTableReadScreenshots: the Table Read's window on a desktop, its
// middle tab on a phone and in a browser-sized page (SHOT_DIR).
func TestTableReadScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(os.Getenv("SHOT_DIR"), 0o755)
	w := trWindow(t, fyne.NewSize(1200, 800))
	cmd("view.tableRead").run(w)
	v := w.tableRead
	v.setBoards(true)
	v.goLine(v.script.Scenes[1].Line)
	v.togglePanel()
	v.win.Resize(fyne.NewSize(1280, 760))
	capture(t, v.win.Canvas(), "tableread-desktop.png")
	v.togglePanel()
	v.goLine(v.script.Scenes[2].Line + 3)
	capture(t, v.win.Canvas(), "tableread-desktop-line.png")
	v.setBeside(true)
	v.goLine(v.script.Scenes[1].Line + 2)
	capture(t, v.win.Canvas(), "tableread-desktop-beside.png")
	v.setBeside(false)
	cmd("tableread.close").run(w)

	p := trWindow(t, fyne.NewSize(380, 800))
	cmd("view.tableRead").run(p)
	p.tableRead.goLine(p.tableRead.script.Scenes[1].Line)
	capture(t, p.fyneWindow.Canvas(), "tableread-phone.png")
	p.tableRead.togglePanel()
	capture(t, p.fyneWindow.Canvas(), "tableread-phone-pick.png")

	b := trWindow(t, fyne.NewSize(1280, 800)) // as the browser has it: a middle tab
	b.tableReadView().update()
	b.openMiddle("tableread")
	b.tableRead.goLine(b.tableRead.script.Scenes[1].Line)
	capture(t, b.fyneWindow.Canvas(), "tableread-browser.png")
}

// Follow mode: a second window joins with the follow link, as a viewer.
// Its Table Read opens and follows the leader: the leader's Table Read
// going to the next scene, a line tapped there, its cursor once its Table
// Read is closed. The viewer edits nothing: its typing is put back and
// never reaches the leader, nor does its language.
func TestTableReadFollow(t *testing.T) {
	noTrackers(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, app := newDialogTestWindow(t)
	holdUI(t)
	a, b := app.newWindow(), app.newWindow()
	a.fyneWindow.Resize(fyne.NewSize(1200, 800))
	b.fyneWindow.Resize(fyne.NewSize(1200, 800))
	a.textEditor.SetText(trSample)
	key, _ := collab.NewKey()
	a.startSharing(key, false)
	_ = os.Remove(keptSharedPath(key))
	_ = os.Remove(ownMarkPath(key))
	if !isFollowLink(followLink(key)) {
		t.Fatal("the follow link is not one")
	}
	b.join(followLink(key))
	t.Cleanup(func() { a.stopSharing(); b.stopSharing() })
	if !b.followsOnly() || !b.shared.session.Viewer || b.tableRead == nil || !b.tableReadOpen() {
		t.Fatal("b is no viewer with its Table Read open")
	}
	connect(t, a, b)
	textIs(t, b, "b", trSample)
	waitFor(t, "b follows a", func() bool { return b.tableRead.follow != "" })
	waitFor(t, "a sees b as a viewer", func() bool {
		h := a.hereNow()
		return len(h) == 1 && h[0].View
	})

	// the leader's Table Read leads
	cmd("view.tableRead").run(a)
	if len(a.tableRead.followIDs) != 1 {
		t.Errorf("a's Follow list offers the viewer: %v", a.tableRead.followIDs)
	}
	av, bv := a.tableRead, b.tableRead
	av.goLine(av.script.Scenes[1].Line)
	av.moved()
	waitFor(t, "b at scene 1", func() bool { return bv.scene == 1 })
	cmd("tableread.next").run(a)
	waitFor(t, "b at scene 2", func() bool { return bv.scene == 2 })
	tea := av.script.Scenes[2].Lines[2].Line // Ben's "Tea?"
	av.tapLine(tea)
	waitFor(t, "b on the line tapped", func() bool { return bv.line == tea })
	if !strings.Contains(bv.estimateText(), "following") {
		t.Errorf("b's estimate line %q", bv.estimateText())
	}

	// its cursor, once its Table Read is closed
	cmd("tableread.close").run(a)
	hall := av.script.Scenes[3].Line + 2
	a.textEditor.SetCursorOffset(len([]rune(strings.Join(strings.Split(trSample, "\n")[:hall], "\n") + "\n")))
	a.onCursorChanged()
	waitFor(t, "b in the hall", func() bool { return bv.scene == 3 && bv.line == hall })

	// the viewer edits nothing
	b.fyneWindow.Canvas().Focus(b.textEditor)
	b.textEditor.SetCursorOffset(0)
	test.Type(b.textEditor, "x")
	textIs(t, b, "b (put back)", trSample)
	b.setLanguage("eo")
	a.textEditor.SetText(strings.Replace(trSample, "Tea?", "Coffee?", 1))
	textIs(t, b, "b", strings.Replace(trSample, "Tea?", "Coffee?", 1))
	if a.textEditor.Text() != strings.Replace(trSample, "Tea?", "Coffee?", 1) || a.language == "eo" {
		t.Errorf("the viewer reached the leader: language %q", a.language)
	}
}

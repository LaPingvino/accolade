package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

const framesFountain = "INT. HARBOUR - DAWN\n\nThe boats come in.\n[[board: boats.png, duration: 2.50]]\n\nANNA\nThere he is.\n[[board: missing.png]]\n"

// The board notes are found, their chips have the picture (a missing one
// warns), and Insert writes the same notation back under the line.
func TestFrameChips(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "boats.png"), testPicture(t, 320, 180), 0o644)
	script := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(script, []byte(framesFountain), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	chips := w.editorChips(w.textEditor.Text())
	if len(chips) != 2 {
		t.Fatalf("%d chips, want 2", len(chips))
	}
	if chips[0].Image == nil || chips[0].Warn || !strings.Contains(chips[0].Label, "boats · 2.5s") {
		t.Errorf("the frame's chip: %+v", chips[0])
	}
	if chips[1].Image != nil || !chips[1].Warn {
		t.Errorf("the missing frame's chip: %+v", chips[1])
	}
	text := []rune(w.textEditor.Text())
	if got := string(text[chips[0].Start:chips[0].End]); got != "[[board: boats.png, duration: 2.50]]" {
		t.Errorf("the chip covers %q", got)
	}
	if again := w.editorChips(w.textEditor.Text()); &again[0] != &chips[0] {
		t.Error("not the same slice for the same text")
	}
	// the round trip: written as read
	b := buffer.New("INT. A - DAY\n\nShe runs.\n[[board: a.png]]\n\nShe stops.\n")
	b.SetCursor(len([]rune("INT. A - DAY\n\nSh")), false)
	insertFrame(b, lex.Board{File: "b.png", Duration: 1.5})
	if got, want := b.Text(), "INT. A - DAY\n\nShe runs.\n[[board: a.png]]\n[[board: b.png, duration: 1.50]]\n\nShe stops.\n"; got != want {
		t.Errorf("inserted:\n%q\nwant\n%q", got, want)
	}
	fs := findFrames(b.Text())
	if len(fs) != 2 || fs[1].board != (lex.Board{File: "b.png", Duration: 1.5}) {
		t.Errorf("read back %+v", fs)
	}
	if boardSrc(lex.Board{File: "x.png"}, "boards") != "boards/x.png" || boardDataPath("Title: X\n[[dataPath: boards]]\n") != "boards" {
		t.Error("the data path not followed")
	}
}

// Page Setup's Print storyboard frames puts them in the PDF, under
// their lines; off (the default), the PDF has none.
func TestPrintFrames(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "boats.png"), testPicture(t, 320, 180), 0o644)
	script := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(script, []byte(framesFountain), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	count := func(p pageSetup) (int, string) {
		var d pdf.Drawing
		j := w.outputJob(p, "pdf", w.textEditor.Text(), nil)
		j.record = &d
		out := filepath.Join(t.TempDir(), "x.pdf")
		if err := exportAs(w.textEditor.Text(), out, "pdf", j); err != nil {
			t.Fatal(err)
		}
		n, text := 0, ""
		for _, pg := range d.Pages {
			for _, m := range pg.Marks {
				switch m := m.(type) {
				case pdf.Image:
					n++
				case pdf.Text:
					text += m.Text + " "
				}
			}
		}
		return n, text
	}
	p := w.pageSetup()
	if p.frames {
		t.Fatal("frames printed by default")
	}
	if n, _ := count(p); n != 0 {
		t.Errorf("%d pictures with frames off", n)
	}
	p.frames = true
	n, text := count(p)
	if n != 1 || !strings.Contains(text, "missing.png") {
		t.Errorf("%d pictures with frames on, text %q", n, text)
	}
}

// A frame added to an .accolade goes into its assets/, travels with it
// and is read from it when the bundle is opened again.
func TestBundleCarriesPictures(t *testing.T) {
	holdUI(t)
	w, _ := newDialogTestWindow(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	dir := t.TempDir()
	pic := filepath.Join(dir, "my shot.png")
	os.WriteFile(pic, testPicture(t, 64, 36), 0o644)
	file := filepath.Join(dir, "Script.accolade")
	os.WriteFile(file, makeZip(t, bundleItems("Script.textbundle/")), 0o644)
	if err := w.LoadFile(file); err != nil {
		t.Fatal(err)
	}
	realPaths(t)
	var rel string
	var kept error
	finished := false
	w.keepPicture(pic, w.pictureFolder(""), func(r string, err error) { rel, kept, finished = r, err, true })
	waitFor(t, "the picture kept", func() bool { return finished && !w.bundleBusy() })
	if kept != nil || rel != "assets/my-shot.png" {
		t.Fatalf("kept as %q: %v", rel, kept)
	}
	data, _ := os.ReadFile(file)
	if got, ok := zipFile(t, data, "Script.textbundle/assets/my-shot.png"); !ok || got == "" {
		t.Fatal("the picture is not in the bundle")
	}
	if p := w.pictures().Load(rel); !p.OK() {
		t.Errorf("not read from the bundle: %v", p.Err)
	}
	w2, _ := newDialogTestWindow(t)
	if err := w2.LoadFile(file); err != nil {
		t.Fatal(err)
	}
	if p := w2.pictures().Load("assets/my-shot.png"); !p.OK() {
		t.Errorf("not read from the bundle opened again: %v", p.Err)
	}
}

// Storyboarder's boards beside a script are read by their scene ids:
// the heading's chip has the first board and the count.
func TestStoryboarderBoards(t *testing.T) {
	dir := t.TempDir()
	scene := "Scene-1-INT-HARBOUR-DAWN-1-x2k9q"
	images := filepath.Join(dir, "storyboards", scene, "images")
	os.MkdirAll(images, 0o755)
	sb, _ := json.Marshal(map[string]any{"version": "1.16.0", "aspectRatio": 2.333, "fps": 24, "defaultBoardTiming": 2000,
		"boards": []map[string]any{{"uid": "AAAAA", "url": "board-1-AAAAA.png", "number": 1, "shot": "1A", "action": "Boats."},
			{"uid": "BBBBB", "url": "board-2-BBBBB.png", "number": 2, "shot": "2A", "duration": 1500}}})
	os.WriteFile(filepath.Join(dir, "storyboards", scene, scene+".storyboarder"), sb, 0o644)
	os.WriteFile(filepath.Join(images, "board-1-AAAAA-posterframe.jpg"), testPicture(t, 100, 43), 0o644) // (a PNG, read by its bytes)
	script := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(script, []byte("INT. HARBOUR - DAWN #1-x2k9q#\n\nThe boats come in.\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	chips := w.editorChips(w.textEditor.Text())
	if len(chips) != 1 || chips[0].Image == nil || !strings.Contains(chips[0].Label, "#1 · 2 boards") {
		t.Fatalf("chips %+v", chips)
	}
	s := w.storyboarder().scenes["x2k9q"]
	if s == nil || len(s.boards) != 2 || s.boards[0].Action != "Boats." || s.boards[1].Duration != 1500 {
		t.Errorf("scene %+v", s)
	}
	if data, _ := os.ReadFile(script); !strings.Contains(string(data), "#1-x2k9q#") {
		t.Error("the script changed")
	}
}

// A character's cast photo: kept beside the script, named in the
// character list, shown in the Characters tab.
func TestCastPhoto(t *testing.T) {
	dir := t.TempDir()
	pic := filepath.Join(dir, "anna.png")
	os.WriteFile(pic, testPicture(t, 200, 200), 0o644)
	script := filepath.Join(dir, "harbour.fountain")
	os.WriteFile(script, []byte("INT. HARBOUR - DAWN\n\nANNA\nThere he is.\n"), 0o644)
	w := newTestWindow(t, "")
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	p := w.charactersPane()
	p.update(w.textEditor.Text())
	p.read()
	old := chooseFileIn
	t.Cleanup(func() { chooseFileIn = old })
	chooseFileIn = func(_ fyne.Window, _ []string, then func(string)) { then(pic) }
	p.choosePhoto("ANNA")
	block, ok := charactersBlock(w.textEditor.Text())
	if !ok || !strings.Contains(block, "photo: harbour.assets/anna.png") {
		t.Fatalf("the list: %q", block)
	}
	if _, err := os.Stat(filepath.Join(dir, "harbour.assets", "anna.png")); err != nil {
		t.Fatal(err)
	}
	p.read()
	c, _ := p.charOf("ANNA")
	if p.photoOf(c, 80) == nil {
		t.Error("the photo is not shown")
	}
}

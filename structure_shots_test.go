package main

import (
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// structureBeats is a script written as beats: three acts, each with its
// own synopsis and beats standing on their own (docs/help/structure.md).
const structureBeats = `Title: The Lighthouse

# Act One

= The keeper's daughter comes home.

= Mara finds her father's letter in the lamp room.

= She decides to stay the winter.

# Act Two

= The storm, and what it takes.

= The storm tears the roof from the boathouse.

= Tomas goes missing on the water.

= Mara takes the old boat out alone.

# Act Three

= Bringing him back.

= She finds Tomas on the rocks.

= Morning: the harbour full of wreckage.
`

// From beats to script, end to end, with the tools: the beats in the
// outline and on the cards, moved, taken into the next act, folded, one
// grown into two scenes, the cards and the outline following, and the
// PDF without sections or synopses. With SHOT_DIR set, each step is
// rendered at a desktop's and a phone's size (combo-structure-*.png).
func TestStructureBeatsToScript(t *testing.T) {
	w, _ := scriptWindow(t, structureBeats)
	text := func() string { return w.textEditor.Text() }
	items := func() []outlineItem { return w.outlineItems(text()) }
	w.togglePane("outline")
	if it := items(); len(it) != 3 || len(it[0].Loose) != 2 || len(it[1].Loose) != 3 || it[1].Synopsis != "The storm, and what it takes." {
		t.Fatalf("the beats: %+v", it)
	}
	comboShot(t, w, "structure-1-outline")
	runCmd(w, "view.cards")
	if !w.cardsShown() {
		t.Fatal("no cards")
	}
	beats := 0
	for _, g := range w.cards.list.Objects {
		if grid, ok := g.(*fyne.Container); ok {
			for _, o := range grid.Objects {
				if _, ok := o.(*synopsisCard); ok {
					beats++
				}
			}
		}
	}
	if beats != 7 {
		t.Errorf("%d beat cards, want 7", beats)
	}
	comboShot(t, w, "structure-2-cards")
	w.showMiddle("")

	// a beat up, and the last of Act One into Act Two
	cursorTo(t, w.textEditor, text(), "= She decides")
	runCmd(w, "structure.up")
	if !textAfter(t, w, "= She decides", "= Mara finds") {
		t.Errorf("not moved up:\n%s", text())
	}
	cursorTo(t, w.textEditor, text(), "= Mara finds")
	runCmd(w, "structure.intoNext")
	if !textAfter(t, w, "# Act Two", "= Mara finds") || !textAfter(t, w, "= The storm, and what it takes.", "= Mara finds") || !textAfter(t, w, "= Mara finds", "= The storm tears") {
		t.Errorf("not into Act Two after its own synopsis:\n%s", text())
	}
	w.textEditor.Undo()
	w.textEditor.Undo()
	if text() != structureBeats {
		t.Fatalf("two undos:\n%s", text())
	}

	w.foldSections()
	comboShot(t, w, "structure-3-folded")
	w.foldAll(true)

	// grow "Tomas goes missing" into two scenes
	cursorTo(t, w.textEditor, text(), "= Tomas goes")
	d, at := w.newSceneDraft()
	d.values["place"] = "HARBOUR"
	d.values["time"] = "NIGHT"
	w.insertScene(at, d.heading())
	typeIn(w, "\n\nTomas casts off into the dark.")
	d, at = w.newSceneDraft()
	d.values["place"] = "LAMP ROOM"
	d.values["time"] = "NIGHT"
	w.insertScene(at, d.heading())
	typeIn(w, "Mara watches the light sweep the empty water.")
	it := items()
	var harbour, lamp *outlineItem
	for k := range it {
		switch {
		case strings.Contains(it[k].Text, "HARBOUR"):
			harbour = &it[k]
		case strings.Contains(it[k].Text, "LAMP ROOM"):
			lamp = &it[k]
		}
	}
	if harbour == nil || lamp == nil || harbour.Synopsis != "Tomas goes missing on the water." || len(lamp.Loose) != 1 || lamp.Loose[0].Text != "Mara takes the old boat out alone." {
		t.Fatalf("grown:\n%s\n%+v", text(), it)
	}
	comboShot(t, w, "structure-4-grown-outline")
	runCmd(w, "view.cards")
	comboShot(t, w, "structure-5-grown-cards")
	w.showMiddle("")

	// the PDF: the scenes, no sections, synopses or notes (a structure's
	// note on how to use it among them)
	out := filepath.Join(t.TempDir(), "b.pdf")
	if err := exportAs(structureText(structureTemplates[0])+"\n"+text(), out, "pdf", exportJob{format: "default"}); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("pdftotext", "-layout", out, "-").Output(); err == nil {
		pdf := string(b)
		if !strings.Contains(pdf, "HARBOUR") || strings.Contains(pdf, "Act One") || strings.Contains(pdf, "Mara finds") || strings.Contains(pdf, "Help > Structure") {
			t.Errorf("the PDF:\n%s", pdf)
		}
	}
}

// The structures: File > New from Structure as a menu, and one opened,
// in the outline and on the cards (SHOT_DIR only).
func TestStructureTemplateScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := newDialogTestWindow(t)
	c := w.fyneWindow.Canvas()
	var file *fyne.Menu
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "File" {
			file = m
		}
	}
	main := widget.NewPopUpMenu(file, c)
	main.ShowAtPosition(fyne.NewPos(4, 4))
	sub := widget.NewPopUpMenu(templateMenu("New from Structure", w.newFromStructure), c)
	sub.ShowAtPosition(fyne.NewPos(4+main.MinSize().Width, 200))
	f, err := os.Create(filepath.Join(dir, "combo-structure-0-new-menu.png"))
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, c.Capture())
	f.Close()
	main.Hide()
	sub.Hide()
	w.newFromStructure(structureTemplates[5]) // the beat sheet
	w.togglePane("outline")
	comboShot(t, w, "structure-6-template-outline")
	runCmd(w, "view.cards")
	comboShot(t, w, "structure-7-template-cards")
}

// typeIn types s into the script's editor, a key at a time (\n: Enter).
func typeIn(w *MainWindow, s string) {
	for _, r := range s {
		if r == '\n' {
			w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			continue
		}
		w.textEditor.TypedRune(r)
	}
}

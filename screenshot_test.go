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
)

// TestScreenshots renders the main window in each colour scheme to PNG
// files for reviewing the design. It only runs when SHOT_DIR is set:
//
//	SHOT_DIR=/tmp/shots go test -run TestScreenshots .
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)
	w := NewMainWindow(&Application{fyneApp: a})
	t.Cleanup(w.stopTimers)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) / 3)

	for _, scheme := range []string{"light", "dark", "sepia"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		img := w.fyneWindow.Canvas().Capture()
		f, err := os.Create(filepath.Join(dir, "accolade-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

// TestMarkdownScreenshots renders a Markdown document open, with its
// outline and preview, in the light and dark schemes (SHOT_DIR as above).
func TestMarkdownScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)
	w := NewMainWindow(&Application{fyneApp: a})
	t.Cleanup(w.stopTimers)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	md := filepath.Join(t.TempDir(), "story.md")
	if err := os.WriteFile(md, []byte(story), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(md); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetCursorOffset(len([]rune(story)) / 3)
	w.togglePane("outline")
	w.preview()
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, "accolade-markdown-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

// TestDocumentTypeScreenshot renders View > Document Type open over a
// Markdown document (SHOT_DIR as above). The test window draws no main
// menu, so the View menu and its submenu are shown as pop-ups where they
// would open.
func TestDocumentTypeScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	p := filepath.Join(t.TempDir(), "story.txt")
	if err := os.WriteFile(p, []byte(story), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(p); err != nil {
		t.Fatal(err)
	}
	w.setDocumentType(true)
	c := w.fyneWindow.Canvas()
	var view *fyne.Menu
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "View" {
			view = m
		}
	}
	main := widget.NewPopUpMenu(view, c)
	main.ShowAtPosition(fyne.NewPos(120, 4))
	types := documentTypeItems(t, w)
	sub := widget.NewPopUpMenu(fyne.NewMenu("", types...), c)
	sub.ShowAtPosition(fyne.NewPos(120+main.MinSize().Width, main.MinSize().Height-40)) // beside its row
	f, err := os.Create(filepath.Join(dir, "accolade-document-type.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, c.Capture()); err != nil {
		t.Fatal(err)
	}
}

// TestImportExportScreenshots renders, for a Markdown document, the File
// menu with Import, and the Export dialog (SHOT_DIR as above); the
// Export dialog with pandoc's formats too, and without them.
func TestImportExportScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shot := func(w *MainWindow, name string) {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	open := func() *MainWindow {
		w, _ := newDialogTestWindow(t)
		w.fyneWindow.Resize(fyne.NewSize(1000, 650))
		p := filepath.Join(t.TempDir(), "story.md")
		if err := os.WriteFile(p, []byte(story), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := w.LoadFile(p); err != nil {
			t.Fatal(err)
		}
		return w
	}
	w := open()
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == "File" {
			widget.NewPopUpMenu(m, w.fyneWindow.Canvas()).ShowAtPosition(fyne.NewPos(4, 4))
		}
	}
	shot(w, "accolade-markdown-import-menu.png")

	w = open()
	ed := newMarkdownExportDialog(w)
	ed.format.SetSelected("Word (DOCX)")
	ed.Show()
	shot(w, "accolade-markdown-export.png")
	ed.dialog.Hide()
	ed.format.OnChanged = nil
	ed.format.SetSelected("PDF")
	ed.formatChanged()
	ed.Show()
	ed.format.Tapped(&fyne.PointEvent{}) // its list open
	shot(w, "accolade-markdown-export-formats.png")

	old := pandocPath
	pandocPath = func() string { return "" }
	defer func() { pandocPath = old }()
	w = open()
	ed = newMarkdownExportDialog(w)
	ed.Show()
	ed.format.Tapped(&fyne.PointEvent{})
	shot(w, "accolade-markdown-export-no-pandoc.png")
}

// TestExportStructureScreenshots renders the Export dialog for the outline
// (OPML: no Page Setup card) and a treatment (the paper alone), SHOT_DIR
// as above.
func TestExportStructureScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ id, name string }{{"opml", "outline"}, {"treatment-pdf", "treatment"}} {
		w, _ := newDialogTestWindow(t)
		w.fyneWindow.Resize(fyne.NewSize(1000, 760))
		ed := NewExportDialog(w)
		ed.formatSelect.SetSelected(exportFormatLabel(c.id))
		ed.Show()
		f, err := os.Create(filepath.Join(dir, "accolade-export-"+c.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}

// TestSourceScreenshots renders a script with its source beside it, the
// cursor in a scene whose linked passage is lit, another picked, in the
// light and dark schemes (SHOT_DIR as above).
func TestSourceScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, sourceScript, sourceStory)
	w.fyneWindow.Resize(fyne.NewSize(1200, 700))
	s := w.sourcePane()
	s.selected = passageNamed(t, s.passages, "The next morning")
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "ANNA")))
	for _, scheme := range []string{"light", "dark"} {
		w.app.setColorScheme(scheme)
		w.fyneWindow.Content().Refresh()
		s.list.Refresh()
		f, err := os.Create(filepath.Join(dir, "accolade-source-"+scheme+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// a scene with an ambiguous and a missing link
	w.textEditor.SetCursorOffset(w.textEditor.Buffer().LineStart(lineWith(t, w, "EXT. GARDEN - DAY")))
	f, err := os.Create(filepath.Join(dir, "accolade-source-doubt.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, w.fyneWindow.Canvas().Capture())
	// the source kept in the script
	if err := w.keepSourceInside(true); err != nil {
		t.Fatal(err)
	}
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) - 40)
	s.follow()
	g, err := os.Create(filepath.Join(dir, "accolade-source-inside.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	png.Encode(g, w.fyneWindow.Canvas().Capture())
}

// cardBeatsText has beats before the first heading, a section's own and
// a scene's at its end: the cards' marks (cards.go, cards_pane.go).
const cardBeatsText = `= Anna finds the key.

= The key fits no door.

# Act One

= Bram leaves town.

= Anna stays behind.

INT. KITCHEN - DAY

Anna waits.

= She hears a car.

INT. ROAD - DAY

Bram drives.
`

// The cards mark their beats without a scene, a tap stepping through
// them, and the beats before the first heading are an Unplaced group.
func TestCardsBeats(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	keepDockSettings(t, w)
	w.textEditor.SetText(cardBeatsText)
	w.showPane("cards", false)
	p := w.cardsPane()
	p.update(w.textEditor.Text())
	if got := p.beats; len(got[-1]) != 2 || len(got[4]) != 1 || len(got[10]) != 1 || len(got[16]) != 0 {
		t.Fatalf("the pane's beats: %v", got)
	}
	if !p.unplaced.Visible() || len(p.unplaced.Objects) != 4 {
		t.Fatalf("the pane's Unplaced: %d rows", len(p.unplaced.Objects))
	}
	if p.rowHeight(1) <= p.rowHeight(2) {
		t.Error("a card with beats is no taller than one without")
	}
	w.textEditor.SetCursorOffset(len([]rune(cardBeatsText)) - 1)
	p.unplaced.Objects[1].(*linkLabel).Tapped(nil)
	if line := strings.Count(string([]rune(w.textEditor.Text())[:w.textEditor.CursorOffset()]), "\n"); line != 0 {
		t.Errorf("the Unplaced mark stepped to line %d", line)
	}
	w.toggleCards()
	var marks []string
	found := false
	walk(w.cards.list, func(o fyne.CanvasObject) {
		if l, ok := o.(*linkLabel); ok {
			marks = append(marks, l.Text)
		}
		if l, ok := o.(*widget.Label); ok && l.Text == "Unplaced" {
			found = true
		}
	})
	if !found || len(marks) != 3 {
		t.Errorf("the cards: Unplaced %v, marks %q", found, marks)
	}
	w.textEditor.SetText("INT. A - DAY\n\nHi.\n")
	p.update(w.textEditor.Text())
	if p.unplaced.Visible() {
		t.Error("no beats, and still Unplaced")
	}
}

// TestCardsBeatsScreenshot renders the Cards pane and the full cards
// with their beats' marks and the Unplaced group (SHOT_DIR as above).
func TestCardsBeatsScreenshot(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := newDialogTestWindow(t)
	keepDockSettings(t, w)
	w.textEditor.SetText(cardBeatsText)
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	w.showPane("cards", false)
	w.cardsPane().update(w.textEditor.Text())
	w.fyneWindow.Content().Refresh()
	capture(t, w.fyneWindow.Canvas(), "cards-beats-pane.png")
	w.toggleCards()
	w.fyneWindow.Content().Refresh()
	capture(t, w.fyneWindow.Canvas(), "cards-beats-full.png")
}

// beatsLandText is a Three Acts script whose Midpoint comes late: on the
// last of its pages.
func beatsLandText() string {
	var b strings.Builder
	b.WriteString("Title: The Long Rain\n\n# Act One\n\n= Anna waits for a letter.\n\nINT. KITCHEN - DAY\n\n")
	for range 60 {
		b.WriteString("The rain goes on, and Anna watches the road for the postman.\n\n")
	}
	b.WriteString("# Act Two\n\nEXT. ROAD - DAY\n\n")
	for range 60 {
		b.WriteString("The wipers beat, the road shines, Bram does not slow down.\n\n")
	}
	b.WriteString("# Midpoint\n\nINT. HALL - NIGHT\n\nThe letter on the mat.\n\n# Act Three\n\nEXT. LAKE - DAWN\n\nThey read it together.\n")
	return b.String()
}

// Where the beats land shows on the Cards' section headers, the
// timeline's bands and the outline's rows: a flag on the late Midpoint,
// its tap going there with what to do.
func TestBeatsLandInCards(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	text := beatsLandText()
	w.textEditor.SetText(text)
	waitFor(t, "the layout", func() bool { _, _, ok := w.beatPlaces(); return ok && !w.beats.busy })
	items := w.outlineItems(text)
	lands := w.landings(items, true)
	mid := strings.Count(text[:strings.Index(text, "# Midpoint")], "\n")
	p, ok := lands[mid]
	if !ok || !p.off() || p.landWord() != "Late" {
		t.Fatalf("the Midpoint: %+v %v (%v)", p, ok, lands)
	}
	w.toggleCards()
	var marks []*linkLabel
	walk(w.cards.list, func(o fyne.CanvasObject) {
		if l, ok := o.(*linkLabel); ok && strings.Contains(l.Text, "p. ") {
			marks = append(marks, l)
		}
	})
	var late *linkLabel
	for _, l := range marks {
		if strings.HasPrefix(l.Text, "⚑ Late") {
			late = l
		}
	}
	if late == nil {
		t.Fatalf("no Late mark on the cards: %d marks", len(marks))
	}
	w.textEditor.SetCursorOffset(0)
	late.Tapped(nil)
	if w.scriptCursorLine() != mid {
		t.Errorf("the mark went to line %d, not %d", w.scriptCursorLine(), mid)
	}
	w.setCardsLook(cardsLook{timeline: true})
	tl := w.cards.tl
	if q, ok := tl.lands[slices.IndexFunc(tl.m.items, func(it outlineItem) bool { return it.Line == mid })]; !ok || !q.off() {
		t.Errorf("the timeline's Midpoint: %+v %v", q, ok)
	}
	o := w.outlinePane()
	o.refresh()
	if q, ok := o.lands[mid]; !ok || !q.off() {
		t.Errorf("the outline's Midpoint: %+v %v", q, ok)
	}
}

// TestBeatsLandScreenshot renders where the beats land on the cards and
// on the timeline (SHOT_DIR as above).
func TestBeatsLandScreenshot(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, _ := newDialogTestWindow(t)
	keepDockSettings(t, w)
	w.textEditor.SetText(beatsLandText())
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	waitFor(t, "the layout", func() bool { _, _, ok := w.beatPlaces(); return ok && !w.beats.busy })
	w.togglePane("outline")
	w.toggleCards()
	w.fyneWindow.Content().Refresh()
	capture(t, w.fyneWindow.Canvas(), "beats-land-cards.png")
	w.setCardsLook(cardsLook{timeline: true})
	w.cards.tl.update()
	w.fyneWindow.Content().Refresh()
	capture(t, w.fyneWindow.Canvas(), "beats-land-timeline.png")
}

// TestMarkdownRevisionScreenshot renders the Revision tab on a Markdown
// document's prose finding, the Notes pane beside (SHOT_DIR as above);
// it checks too that Dismiss writes the finding into the notes block,
// which alone keeps it out when the file is opened again.
func TestMarkdownRevisionScreenshot(t *testing.T) {
	w := newTestWindow(t, "")
	keepDockSettings(t, w)
	dismissInDocFor(t, w) // the notes block's way
	r := ribbonOnFor(t, w)
	md := filepath.Join(t.TempDir(), "story.md")
	text := "# The Letter\n\nAnna stands at the window and watches the road, the empty road. The postman is late again.\n\n" +
		"She felt the cold through the glass. She saw the gate move. She noticed the dog had gone quiet, and she wondered why.\n\n" +
		"Bram comes up the hill at last.\n"
	if err := os.WriteFile(md, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadFile(md); err != nil {
		t.Fatal(err)
	}
	w.fyneWindow.Resize(fyne.NewSize(1200, 720))
	r.selectTab("Revision")
	w.togglePane("notes")
	w.textEditor.SetCursorOffset(0)
	w.stepOpen(1)
	it, ok := w.currentItem("test")
	if !ok || it.Check != checkRepeat {
		t.Fatalf("not on the repeat: %+v", it)
	}
	w.ribbonRefresh()
	w.fyneWindow.Content().Refresh()
	capture(t, w.fyneWindow.Canvas(), "revision-markdown.png")
	w.dismissFinding(it)
	if !strings.Contains(w.textEditor.Text(), "- "+it.dismissKey()) {
		t.Fatalf("not in the notes:\n%s", w.textEditor.Text())
	}
	again := newTestWindow(t, "")
	again.markdown = true
	again.textEditor.SetText(w.textEditor.Text())
	for _, f := range again.revItems() {
		if f.Check == checkRepeat {
			t.Errorf("dismissed, back after a reload: %+v", f)
		}
	}
	w.textEditor.Undo()
	if _, ok := w.dismissedOf()[it.dismissKey()]; ok {
		t.Error("Undo did not bring it back")
	}
}

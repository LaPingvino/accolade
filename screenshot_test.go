// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
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

// The Live Preview on a 380-wide phone keeps the window at 380: nothing
// in it asks for more (it once widened it to ~429). SHOT_DIR, as above,
// keeps the text and pages shots.
func TestLivePreviewPhoneWidth(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.currentFile = filepath.Join(t.TempDir(), "story.md")
	w.textEditor.SetText(story)
	w.fyneWindow.Resize(fyne.NewSize(380, 800))
	w.onWindowWidth(380)
	w.preview()
	for _, mode := range []string{previewText, previewPages} {
		w.previewPane.mode.SetSelected(mode)
		time.Sleep(200 * time.Millisecond)
		fyne.DoAndWait(func() {})
		w.fyneWindow.Content().Refresh()
		min := w.fyneWindow.Content().MinSize().Width
		got := w.fyneWindow.Canvas().Size().Width
		t.Logf("%s: content min width %.0f, window %.0f", mode, min, got)
		if min > 380 || got > 380 {
			t.Errorf("%s: the Live Preview widens a 380 phone: min %.0f, window %.0f", mode, min, got)
		}
		if os.Getenv("SHOT_DIR") != "" {
			capture(t, w.fyneWindow.Canvas(), "livepreview-phone-"+mode+".png")
		}
	}
}

// TestScreenplaySourcePageScreenshot renders a script with its PDF
// screenplay source beside it: the deck scene's differences and its page
// as printed, with its tick (SHOT_DIR as above).
func TestScreenplaySourcePageScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := sourceWindow(t, spConverted, "")
	os.Remove(filepath.Join(filepath.Dir(w.currentFile), "story.md"))
	pdf := filepath.Join(t.TempDir(), "ferry.pdf")
	if err := exportAs(spOriginal, pdf, "pdf", exportJob{}); err != nil {
		t.Fatal(err)
	}
	holdUI(t)
	w.chooseSourceFile(pdf)
	waitForText(t, w)
	fyne.DoAndWait(func() {
		w.fyneWindow.Resize(fyne.NewSize(1300, 1500)) // tall: the page below the differences in sight
		spCursorTo(t, w, "EXT. FERRY DECK")
		f, err := os.Create(filepath.Join(dir, "accolade-source-page.png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		png.Encode(f, w.fyneWindow.Canvas().Capture())
	})
}

// TestFilePickerScreenshots renders File > Open's dialog with its kind
// toggles and file types, at desktop and phone width, and Save As's file
// types open, SHOT_DIR as above.
func TestFilePickerScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := t.TempDir()
	for name, text := range map[string]string{
		"The Barn.fountain": "INT. BARN - DAY\n", "The Barn draft.txt": "INT. BARN - NIGHT\n\nJOHN\nIt's coming.\n\nMARY\nWhat is?\n",
		"Chapter one.md": "# One\n\nIt was a dark night.\n", "Letter.txt": "Dear Ann,\n\nThe barn is finished.\n",
		"Outline.opml": "<opml/>", "Final.fdx": "<FinalDraft/>",
	} {
		os.WriteFile(filepath.Join(files, name), []byte(text), 0o644)
	}
	if b, err := os.ReadFile("testdata/scanned-tv-episode.pdf"); err == nil {
		os.WriteFile(filepath.Join(files, "Episode scan.pdf"), b, 0o644)
	}
	loc, err := storage.ListerForURI(storage.NewFileURI(files))
	if err != nil {
		t.Fatal(err)
	}
	shot := func(w *MainWindow, name string) {
		time.Sleep(300 * time.Millisecond) // the kinds read
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
	}
	for _, size := range []struct {
		name string
		w, h float32
	}{{"desktop", 1000, 650}, {"phone", 400, 760}} {
		w, _ := newDialogTestWindow(t)
		w.fyneWindow.Resize(fyne.NewSize(size.w, size.h))
		d := dialog.NewFileOpen(func(fyne.URIReadCloser, error) {}, w.fyneWindow)
		setPickFilters(d, openExtensions(), w.preferredKind())
		d.SetLocation(loc)
		d.SetView(dialog.ListView)
		showPicker(d, w.fyneWindow)
		shot(w, "accolade-open-kinds-"+size.name+".png")
		d.SelectFileKind("")
		shot(w, "accolade-open-everything-"+size.name+".png")
	}

	w, _ := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	d := dialog.NewFileSave(func(fyne.URIWriteCloser, error) {}, w.fyneWindow)
	setSaveFilters(d, saveExtensions(), ".fountain")
	d.SetFileName("The Barn.fountain")
	d.SetLocation(loc)
	showPicker(d, w.fyneWindow)
	if sel := findSelect(w.fyneWindow.Canvas().Overlays().Top()); sel != nil {
		sel.Tapped(&fyne.PointEvent{})
	}
	shot(w, "accolade-saveas-types.png")
}

// findSelect is the first select in o.
func findSelect(o fyne.CanvasObject) *widget.Select {
	var found *widget.Select
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil || o == nil {
			return
		}
		switch x := o.(type) {
		case *widget.Select:
			found = x
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			r := test.WidgetRenderer(x)
			for _, c := range r.Objects() {
				walk(c)
			}
		}
	}
	walk(o)
	return found
}

// TestStoryboardFramesScreenshot: frames drawn in the open programs — an
// SVG (Inkscape), a .kra (Krita) and an animated Lottie (its time from
// its file) as chips, and a frame's dialog with Edit in Inkscape
// (SHOT_DIR as above).
func TestStoryboardFramesScreenshot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	fakeTools(t, []string{"inkscape", "krita"}, []string{"org.mattbas.Glaxnimate"})
	w, app := newDialogTestWindow(t)
	w.fyneWindow.Resize(fyne.NewSize(1000, 650))
	src := t.TempDir()
	frame := strings.Replace(string(blankFrame(2.35)), `id="drawing"/>`, `id="drawing"><circle cx="700" cy="420" r="260" fill="#3a6ea5"/><rect x="1100" y="250" width="500" height="320" fill="#d9822b"/></g>`, 1)
	os.WriteFile(filepath.Join(src, "harbour.svg"), []byte(frame), 0o644)
	os.WriteFile(filepath.Join(src, "anna.kra"), krita(t, testPicture(t, 320, 136)), 0o644)
	os.WriteFile(filepath.Join(src, "waves.json"), []byte(lottieJSON), 0o644)
	script := filepath.Join(src, "harbour.fountain")
	os.WriteFile(script, []byte("[[aspectRatio: 2.35]]\n\nINT. HARBOUR - DAWN\n\nThe boats come in.\n[[board: harbour.svg, duration: 2.50]]\n\nANNA\nThere he is.\n[[board: anna.kra]]\n\nEXT. SEA - DAY\n\nThe waves roll.\n[[board: waves.json]]\n"), 0o644)
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	shot := func(name string) {
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	w.textEditor.SetCursorOffset(0)
	shot("frames-chips.png")
	text := w.textEditor.Text()
	for _, f := range findFrames(text) {
		if f.board.File == "harbour.svg" {
			w.showFrame(f.board, f.board.File, f.start)
		}
	}
	shot("frames-dialog.png")
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.Show()
	pd.tabs.SelectIndex(2)
	pd.framesFold.Open(0)
	pd.win.Resize(fyne.NewSize(640, 1500))
	f, _ := os.Create(filepath.Join(dir, "frames-preferences.png"))
	png.Encode(f, pd.win.Canvas().Capture())
	f.Close()
	pd.close()
}

// TestFunScreenshot: a new window, its drop line over the status bar and
// Help > Fun Things to Try beside the script (SHOT_DIR as above).
func TestFunScreenshot(t *testing.T) {
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
	w.fyneWindow.Resize(fyne.NewSize(1200, 750))
	w.welcomeOnce()
	w.showHelp("fun", "")
	w.fyneWindow.Content().Refresh()
	f, err := os.Create(filepath.Join(dir, "accolade-fun.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

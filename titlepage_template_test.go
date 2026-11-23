package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/lexington/pdf"
)

const templateScript = "Contact:\n    Ann Writer\nTitle: Rain\nAuthor: Ann\n\nINT. HOUSE - DAY\n\nRain.\n"

// The Title Page tab chooses the template (the page setup's, so Export,
// Print and Preview follow it) and adds the keys it names that the tab
// has no button for; the export is given it.
func TestTitleTemplateTab(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, templateScript)
	t.Cleanup(func() { w.setTitleTemplate("") })
	r := ribbonOnFor(t, w)
	w.textEditor.SetCursorOffset(len("Contact:\n    Ann Writer\nTi"))
	w.ribbonRefresh()
	r.selectTab("Title Page")
	if r.titleTpl == nil || r.titleTpl.Selected != "Standard" {
		t.Fatalf("the template's choice: %v", r.titleTpl)
	}
	if b := r.byID["view.preview"]; len(b) == 0 {
		t.Error("no Preview on the tab")
	}
	r.titleTpl.SetSelected("Poster")
	if got := w.pageSetup().titleTemplate; got != "poster" {
		t.Fatalf("the page setup's template: %q", got)
	}
	if j := w.layoutJob(); j.titleTemplate != "poster" || j.options("pdf", "x.pdf").TitlePage != "poster" {
		t.Errorf("the export's template: %q", j.titleTemplate)
	}
	var labels []string
	for _, b := range r.buttons {
		labels = append(labels, b.label)
	}
	all := strings.Join(labels, "|")
	if !strings.Contains(all, "|Picture|") || !strings.Contains(all, "|Logline|") {
		t.Errorf("the template's keys are not on the tab: %s", all)
	}
	if r.titleTpl.Selected != "Poster" {
		t.Errorf("the tab shows %q", r.titleTpl.Selected)
	}
	w.textEditor.SetCursorOffset(len("Contact:\n    Ann Writer\nTi"))
	file, bundle := w.currentFile, w.bundle
	w.currentFile, w.bundle = "", nil // (nowhere to keep a picture: its name to fill in)
	for _, k := range w.titleTemplateKeys() {
		if k.Key == "Picture" {
			w.addTemplateKey(k)
		}
	}
	w.currentFile, w.bundle = file, bundle
	if !strings.Contains(w.textEditor.Text(), "Title: Rain\nPicture: ![](picture.png)\nAuthor: Ann") {
		t.Errorf("the picture key:\n%s", w.textEditor.Text())
	}
	// a template the configuration no longer has: the standard one
	p := w.pageSetup()
	p.titleTemplate = "gone"
	w.setPageSetup(p)
	if w.titleTemplate() != "" || w.layoutJob().titleTemplate != "" {
		t.Error("an unknown template is used")
	}
}

// The preview draws the title page as the PDF does, with the template.
func TestTitlePageImage(t *testing.T) {
	w, _ := scriptWindow(t, templateScript)
	std, err := w.titlePageImage(templateScript, "", 0.5)
	if err != nil {
		t.Fatal(err)
	}
	bold, err := w.titlePageImage(templateScript, "bold", 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if std.Bounds() != bold.Bounds() || std.Bounds().Dx() < 300 {
		t.Fatalf("sizes %v %v", std.Bounds(), bold.Bounds())
	}
	same := true
	for y := 0; y < std.Bounds().Dy() && same; y += 3 {
		for x := 0; x < std.Bounds().Dx(); x += 3 {
			if std.At(x, y) != bold.At(x, y) {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("the bold template draws as the standard one")
	}
}

// artPNG is a small picture to print: a blue field with a gold sun.
func artPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 160, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 160; x++ {
			c := color.RGBA{30, 60, 110, 255}
			if (x-80)*(x-80)+(y-50)*(y-50) < 600 {
				c = color.RGBA{240, 200, 90, 255}
			}
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

// A title page picture beside the script reaches the page: read by the
// window's picture loader, printed by the template, and painted when it
// comes in an edit while the pages are shown (the page view's painter
// takes the new drawing's pictures).
func TestTitlePagePicture(t *testing.T) {
	holdUI(t)
	w, p := scriptWindow(t, templateScript)
	artPNG(t, filepath.Join(filepath.Dir(p), "art.png"))
	w.pictures().Forget()
	t.Cleanup(func() { w.setTitleTemplate("") })
	w.setTitleTemplate("poster")
	v := previewAndWait(t, w)
	if hasBlue(painted(t, v, 0)) {
		t.Fatal("a picture before there is one")
	}
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(0, 0, "Picture: ![Art](art.png)\n") })
	j := w.outputJob(w.pageSetup(), "pdf", w.textEditor.Text(), nil)
	if len(j.titlePictures["art.png"]) == 0 {
		t.Fatalf("the picture is not given to Lexington: %v", j.titlePictures)
	}
	w.previewPDF()
	v = awaitPages(t, shown)
	found := false
	for _, m := range v.d.Pages[0].Marks {
		if _, ok := m.(pdf.Image); ok {
			found = true
		}
	}
	if !found {
		t.Error("no picture on the title page")
	}
	if !hasBlue(painted(t, v, 0)) {
		t.Errorf("the picture is not painted on the page (images %d)", len(v.d.Images))
	}
}

// hasBlue tells whether a page shows artPNG's blue.
func hasBlue(img image.Image) bool {
	if img == nil {
		return false
	}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 2 {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
			if r, g, b, _ := img.At(x, y).RGBA(); b>>8 > 90 && r>>8 < 60 && g>>8 < 80 {
				return true
			}
		}
	}
	return false
}

// TestTitleTemplateScreenshots renders, with SHOT_DIR set (as
// TestScreenshots): the Title Page tab with the bold template chosen and
// the pages previewed from it (the title page first), the same with the
// poster template, the Title Page dialog with its preview, and Page
// Setup's new row.
func TestTitleTemplateScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	holdUI(t)
	os.MkdirAll(dir, 0o755)
	script := "Title: The Lighthouse Keeper\nCredit: Written by\nAuthor: Ann Writer\nLogline: A keeper who never sleeps finds a ship that never sails.\n" +
		"Draft date: 23 November 2026\nContact:\n    Ann Writer\n    ann@example.com\n\nEXT. LIGHTHOUSE - NIGHT\n\nThe lamp turns.\n"
	w, _ := scriptWindow(t, script)
	t.Cleanup(func() { w.setTitleTemplate("") })
	r := ribbonOnFor(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1200, 760))
	w.textEditor.SetCursorOffset(3)
	w.ribbonRefresh()
	r.selectTab("Title Page")
	shot := func(name string) {
		w.fyneWindow.Content().Refresh()
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	r.titleTpl.SetSelected("Bold")
	v := previewAndWait(t, w)
	if v.d == nil || len(v.d.Pages) < 2 {
		t.Fatal("no pages")
	}
	painted(t, v, 0)
	w.textEditor.SetCursorOffset(3)
	w.ribbonRefresh()
	r.selectTab("Title Page")
	shot("titletpl-1-tab-bold-preview.png")
	shown := make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	r.titleTpl.SetSelected("Poster") // the pages follow the page setup
	v = awaitPages(t, shown)
	painted(t, v, 0)
	w.textEditor.SetCursorOffset(3)
	w.ribbonRefresh()
	r.selectTab("Title Page")
	shot("titletpl-2-tab-poster-preview.png")

	// a picture beside the script, above the poster's title
	artPNG(t, filepath.Join(filepath.Dir(w.currentFile), "art.png"))
	w.pictures().Forget()
	shown = make(chan *pageView, 4)
	onPagesShown(t, func(v *pageView) { shown <- v })
	w.textEditor.Edit(func(b *buffer.Buffer) { b.Replace(0, 0, "Picture: ![Key art](art.png)\n") })
	w.previewPDF()
	v = awaitPages(t, shown)
	if pi := painted(t, v, 0); !hasBlue(pi) {
		t.Error("the picture added is not painted on the page shown")
	}
	w.textEditor.SetCursorOffset(3)
	w.ribbonRefresh()
	r.selectTab("Title Page")
	shot("titletpl-6-poster-picture.png")

	d := NewTitlePageDialog(w)
	d.Show()
	shot("titletpl-3-dialog.png")
	d.previewTitlePage()
	shot("titletpl-4-dialog-preview.png")
	for _, o := range w.fyneWindow.Canvas().Overlays().List() {
		w.fyneWindow.Canvas().Overlays().Remove(o)
	}
	p := w.showPageSetup("pdf", nil)
	shot("titletpl-5-pagesetup.png")
	p.d.Hide()
}

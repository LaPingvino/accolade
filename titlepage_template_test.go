package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
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

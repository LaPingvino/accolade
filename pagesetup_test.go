// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/pdfin"
)

const setupScript = "Title: The Barn\nAuthor: Anna\n\n===\n\n# Act One [[blue]]\n\nINT. BARN - DAY\n\nRain falls.\n\nANNA\nHello.\n"

// Each setting of Page Setup is read by Export, Print and Preview PDF
// from one place: set in the dialog, each output's options have it, the
// same in all three.
func TestPageSetupOnePlace(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(setupScript)
	defaults := w.pageSetup()
	t.Cleanup(func() { defaults.writeTo(w.settings) })
	// what a job prints: Lexington's options, and the script as it gives it
	printed := func(j exportJob, format string) any {
		return struct {
			o           any
			omit, renum bool
		}{j.options(format, "x"), j.omitTitlePage, j.numberScenes}
	}
	opts := func(format string) [3]any {
		ed := NewExportDialog(w)
		ed.formatSelect.SetSelected(exportFormatLabel(format))
		export := printed(ed.job(), format)
		preview := printed(w.outputJob(w.pageSetup(), format, w.textEditor.Text(), nil), format)
		return [3]any{export, preview, printed(w.layoutJob(), format)}
	}
	for _, c := range []struct {
		name, format string
		set          func(p *pageSetupDlg)
		got          func(any) any
	}{
		{"paper", "pdf", func(p *pageSetupDlg) { p.paperSelect.SetSelected("A5") }, nil},
		{"title page", "pdf", func(p *pageSetupDlg) { p.includeTitlePageCheck.SetChecked(false) }, nil},
		{"scene numbers", "html", func(p *pageSetupDlg) { p.sceneNumbersCheck.SetChecked(true) }, nil},
		{"continueds", "pdf", func(p *pageSetupDlg) { p.continuedsCheck.SetChecked(true) }, nil},
		{"colours", "html", func(p *pageSetupDlg) {
			p.colorsCheck.SetChecked(true)
			p.colorStyleSelect.SetSelected("Marker")
			p.colorBlockCheck.SetChecked(true)
		}, nil},
		{"page breaks", "pdf", func(p *pageSetupDlg) { p.paginationSelect.SetSelected("None (as before)") }, nil},
		{"template", "fdx", func(p *pageSetupDlg) { p.templateEntry.SetText("/t/mine.fdx") }, nil},
	} {
		defaults.writeTo(w.settings)
		before := opts(c.format)
		p := w.newPageSetup(c.format, nil)
		c.set(p)
		p.ok()
		after := opts(c.format)
		if reflect.DeepEqual(before[0], after[0]) {
			t.Errorf("%s: the export's options did not change", c.name)
		}
		// Print's is the HTML's, Preview's the PDF's: the same builder
		if !reflect.DeepEqual(after[0], after[1]) {
			t.Errorf("%s: Export %+v\nPrint/Preview %+v", c.name, after[0], after[1])
		}
		if c.name != "template" && c.name != "paper" && !reflect.DeepEqual(after[0], after[2]) {
			t.Errorf("%s: the page counts' layout %+v\nExport %+v", c.name, after[2], after[0])
		}
	}
}

// The settings reach what Print and Preview PDF make: the title page in
// the page printed, and in the preview's pages.
func TestPageSetupPrintAndPreview(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(setupScript)
	t.Cleanup(func() { w.settings.SetBoolean("include-title-page", true) })
	printed := make(chan string, 1)
	oldPrint := printFile
	printFile = func(p string) { printed <- p }
	t.Cleanup(func() { printFile = oldPrint })
	shown := make(chan *pageView, 4)
	pagesInto(t, shown)
	wait := func(c chan string) string {
		t.Helper()
		select {
		case p := <-c:
			return p
		case <-time.After(30 * time.Second):
			t.Fatal("nothing made")
		}
		return ""
	}
	waitPages := func() *pageView {
		t.Helper()
		select {
		case v := <-shown:
			return v
		case <-time.After(30 * time.Second):
			t.Fatal("no pages shown")
		}
		return nil
	}
	pages := func() int {
		t.Helper()
		w.previewPDF()
		v := waitPages()
		ps, err := pdfin.Pages(v.pdfPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(ps) != len(v.d.Pages) {
			t.Errorf("the PDF has %d pages, the page view %d", len(ps), len(v.d.Pages))
		}
		return len(ps)
	}
	printedTitle := func() bool {
		t.Helper()
		w.printScript()
		b, _ := os.ReadFile(wait(printed))
		return strings.Contains(string(b), "The Barn")
	}
	with, withPrint := pages(), printedTitle()
	p := w.newPageSetup(printFormat, nil)
	p.includeTitlePageCheck.SetChecked(false)
	p.ok()
	if without := pages(); without != with-1 {
		t.Errorf("preview: %d pages with the title page, %d without", with, without)
	}
	if !withPrint || printedTitle() {
		t.Error("print: the title page did not follow the page setup")
	}
	// Preview's line over the pages says so, and a change in Page Setup
	// makes the pages again
	ps := w.previewSetup
	if ps == nil || !strings.Contains(strings.ToLower(ps.line.summary.Text), "no title page") {
		t.Fatalf("Preview's line: %+v", ps)
	}
	p = w.newPageSetup("pdf", nil)
	p.includeTitlePageCheck.SetChecked(true)
	p.ok()
	if v := waitPages(); len(v.d.Pages) != with {
		t.Errorf("after Page Setup: %d pages, want %d", len(v.d.Pages), with)
	}
}

// Page Setup shows only what the format uses, as Export did.
func TestPageSetupPerFormat(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.textEditor.SetText(setupScript)
	shown := func(p *pageSetupDlg, o fyne.CanvasObject) bool {
		for _, x := range p.box.Objects {
			if x == o {
				return true
			}
			if c, ok := x.(*fyne.Container); ok && len(c.Objects) > 0 {
				for _, y := range c.Objects {
					if y == o {
						return true
					}
				}
			}
		}
		return false
	}
	p := w.newPageSetup("pdf", nil)
	if !shown(p, p.paperSelect) || !shown(p, p.includeTitlePageCheck) || shown(p, p.templateEntry) || shown(p, p.revisionsCheck) {
		t.Error("PDF: paper and title page, no template, no revisions before an issue")
	}
	p.formatSelect.SetSelected(pageFormatLabel("fdx"))
	if shown(p, p.paperSelect) || !shown(p, p.templateEntry) || shown(p, p.scriptFormatSelect) {
		t.Error("FDX: a template, no paper, no script format")
	}
	p.formatSelect.SetSelected(pageFormatLabel(printFormat))
	if shown(p, p.paperSelect) || !shown(p, p.colorsCheck) {
		t.Error("Print: colours, no paper")
	}
	p.formatSelect.SetSelected(pageFormatLabel("txt"))
	if len(p.box.Objects) != 0 || !p.note.Visible() {
		t.Error("plain text: a page setup")
	}
	if a := pageSetupApplies("pdf", true); !a.revisions || !a.revisedPages {
		t.Error("an issued script's PDF: no revisions")
	}
	if a := pageSetupApplies("fdx", true); !a.revisions || a.revisedPages {
		t.Error("FDX: the marks, not revised pages")
	}
	if !strings.Contains(w.pageSummary(w.pageSetup(), "txt"), "no page setup") {
		t.Error("plain text's line")
	}
}

// The ribbon's Scene Numbers and CONTINUEDs are Page Setup's: each shows
// what the other set.
func TestPageSetupFollowsRibbon(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	w.workAs(roleProduction)
	t.Cleanup(func() {
		w.settings.SetBoolean("fountain-scene-numbers", false)
		w.settings.SetBoolean(continuedsPref, false)
	})
	r.selectTab("Production")
	buttonOf(r, "set.sceneNumbers").press()
	p := w.newPageSetup("pdf", nil)
	if !p.sceneNumbersCheck.Checked {
		t.Error("Scene Numbers on the ribbon: not in Page Setup")
	}
	p.continuedsCheck.SetChecked(true)
	p.sceneNumbersCheck.SetChecked(false)
	p.ok()
	r.refresh()
	if !buttonOf(r, "set.continueds").checked || buttonOf(r, "set.sceneNumbers").checked {
		t.Error("Page Setup's CONTINUEDs and numbers: not on the ribbon")
	}
	if buttonOf(r, "file.pageSetup") == nil {
		t.Error("no Page Setup in Production > Options")
	}
}

// A Markdown document's page setup: the paper and a title page in Word,
// ODT and the web page; the defaults make what they made before.
func TestMarkdownPageSetup(t *testing.T) {
	doc := "---\ntitle: The Keeper\nauthor: Anna\n---\n\n# One\n\nThe light.\n"
	dir := t.TempDir()
	part := func(file, name string) string {
		t.Helper()
		z, err := zip.OpenReader(file)
		if err != nil {
			t.Fatal(err)
		}
		defer z.Close()
		for _, f := range z.File {
			if f.Name == name {
				r, _ := f.Open()
				b, _ := io.ReadAll(r)
				return string(b)
			}
		}
		t.Fatalf("%s: no %s", file, name)
		return ""
	}
	for _, f := range []string{"docx", "odt", "html"} {
		plain, setUp := filepath.Join(dir, "a."+f), filepath.Join(dir, "b."+f)
		if err := exportMarkdown(doc, "The Keeper", plain, f); err != nil {
			t.Fatal(err)
		}
		again := filepath.Join(dir, "c."+f)
		if err := exportMarkdownWith(doc, "The Keeper", again, f, mdPageSetup{}, nil); err != nil {
			t.Fatal(err)
		}
		a, _ := os.ReadFile(plain)
		c, _ := os.ReadFile(again)
		if f == "html" && string(a) != string(c) {
			t.Errorf("%s: the defaults changed the page", f)
		}
		p := mdPageSetup{paper: "a5", titlePage: true}
		if err := exportMarkdownWith(doc, "The Keeper", setUp, f, p, p.titlePageOf(doc, "keeper.md")); err != nil {
			t.Fatal(err)
		}
		switch f {
		case "docx":
			if d := part(plain, "word/document.xml"); !strings.Contains(d, `w:w="11906" w:h="16838"`) {
				t.Error("docx: not A4 by default")
			}
			d := part(setUp, "word/document.xml")
			if !strings.Contains(d, `w:w="8391" w:h="11906"`) || !strings.Contains(d, ">The Keeper<") || !strings.Contains(d, ">Anna<") || !strings.Contains(d, `w:type="page"`) {
				t.Errorf("docx: A5 and a title page: %s", d[:min(len(d), 600)])
			}
		case "odt":
			if s := part(setUp, "styles.xml"); !strings.Contains(s, `fo:page-width="5.83in" fo:page-height="8.27in"`) {
				t.Error("odt: not A5")
			}
			if c := part(setUp, "content.xml"); !strings.Contains(c, `P_title">The Keeper<`) || !strings.Contains(c, `P_title_end">Anna<`) {
				t.Error("odt: no title page")
			}
		case "html":
			b, _ := os.ReadFile(setUp)
			if !strings.Contains(string(b), "@page{size:A5}") || !strings.Contains(string(b), `<p class="title">The Keeper</p>`) {
				t.Error("html: no paper or title page")
			}
		}
	}
	w, _ := newDialogTestWindow(t)
	w.markdown = true
	w.textEditor.SetText(doc)
	t.Cleanup(func() {
		w.settings.SetString(mdPaperPref, "")
		w.settings.SetBoolean(mdTitlePagePref, false)
	})
	p := w.newMarkdownPageSetup(nil)
	p.paper.SetSelected("Letter")
	p.titlePage.SetChecked(true)
	p.d.Show()
	capture(t, w.fyneWindow.Canvas(), "pagesetup-markdown.png")
	p.ok.OnTapped()
	if got := w.markdownPageSetup(); got.paper != "letter" || !got.titlePage {
		t.Errorf("saved %+v", got)
	}
	if !strings.Contains(w.markdownPageSummary(), "Letter · title page") {
		t.Errorf("line %q", w.markdownPageSummary())
	}
}

// Screenshots (SHOT_DIR=… go test -run TestPageSetupScreenshots):
// pagesetup-*, export-*, print-*, preview-* for a desktop and a phone.
func TestPageSetupScreenshots(t *testing.T) {
	if os.Getenv("SHOT_DIR") == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(os.Getenv("SHOT_DIR"), 0o755)
	for _, size := range []struct {
		name string
		size fyne.Size
	}{{"desktop", fyne.NewSize(1100, 760)}, {"phone", fyne.NewSize(380, 800)}} {
		w, _ := newDialogTestWindow(t)
		w.textEditor.SetText(setupScript)
		w.fyneWindow.Resize(size.size)
		w.settings.SetBoolean("export-colors", true)
		p := w.newPageSetup("pdf", nil)
		p.d.Show()
		capture(t, w.fyneWindow.Canvas(), "pagesetup-"+size.name+".png")
		p.scroll.ScrollToBottom()
		p.scroll.Refresh()
		capture(t, w.fyneWindow.Canvas(), "pagesetup-bottom-"+size.name+".png")
		p.d.Hide()
		ed := NewExportDialog(w)
		ed.dialog.Resize(fitDialog(w.fyneWindow, 600, 700))
		ed.Show()
		capture(t, w.fyneWindow.Canvas(), "export-"+size.name+".png")
		ed.characterSelect.SetSelected("ANNA")
		ed.sidesRadio.SetSelected("Cue lines")
		w.fyneWindow.Resize(fyne.NewSize(size.size.Width, 1300)) // tall, to see the sides' rows whole
		ed.dialog.Resize(fitDialog(w.fyneWindow, 600, 1200))
		capture(t, w.fyneWindow.Canvas(), "export-sides-"+size.name+".png")
		w.fyneWindow.Resize(size.size)
		ed.dialog.Hide()
		d := w.showPrint()
		capture(t, w.fyneWindow.Canvas(), "print-"+size.name+".png")
		d.Hide()
		previewAndWait(t, w)
		capture(t, w.fyneWindow.Canvas(), "preview-"+size.name+".png")
		w.hidePages()
		w.settings.SetBoolean("export-colors", false)
		// Preferences' card, and a Markdown document's Export and Live Preview
		pd := NewPreferencesDialog(w.app, w.fyneWindow)
		w.app.windows = []*MainWindow{w}
		pd.showPageSetup()
		if pd.dlg != nil {
			pd.tabs.SelectIndex(2)
			pd.dlg.Show()
			capture(t, w.fyneWindow.Canvas(), "preferences-pagesetup-"+size.name+".png")
			pd.dlg.Hide()
		} else if pd.win != nil {
			pd.tabs.SelectIndex(2)
			pd.win.Resize(size.size)
			capture(t, pd.win.Canvas(), "preferences-pagesetup-"+size.name+".png")
			pd.win.Close()
		}
		w.app.windows = nil
		w.markdown = true
		w.settings.SetBoolean(mdTitlePagePref, true)
		w.textEditor.SetText("---\ntitle: The Keeper\nauthor: Anna\n---\n\n# One\n\nThe light.\n")
		md := newMarkdownExportDialog(w)
		md.Show()
		capture(t, w.fyneWindow.Canvas(), "md-export-"+size.name+".png")
		md.dialog.Hide()
		w.preview()
		capture(t, w.fyneWindow.Canvas(), "md-preview-"+size.name+".png")
		w.preview()
		w.markdown = false
		w.settings.SetBoolean(mdTitlePagePref, false)
	}
}

// Once issued, the revisions are Page Setup's too: the marks, revised pages
// only and their colour's paper reach Export's and Preview's PDF alike,
// the marks an FDX; the page counts and Print never take them.
func TestPageSetupRevisions(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	issueNow(t, w, "White", time.Now())
	replaceText(t, w, "Mud everywhere.", "Mud.")
	defaults := w.pageSetup()
	t.Cleanup(func() { defaults.writeTo(w.settings) })
	p := w.newPageSetup("pdf", nil)
	for _, c := range []fyne.CanvasObject{p.revisionsCheck, p.revisedOnlyCheck, p.pageColoursCheck} {
		found := false
		for _, o := range p.box.Objects {
			found = found || o == c
		}
		if !found {
			t.Fatal("an issued script's PDF: the revisions not in Page Setup")
		}
	}
	p.revisionsCheck.SetChecked(true)
	p.revisedOnlyCheck.SetChecked(true)
	p.pageColoursCheck.SetChecked(true)
	p.ok()
	text := w.textEditor.Text()
	ed := NewExportDialog(w)
	ed.formatSelect.SetSelected(exportFormatLabel("pdf"))
	export, preview := ed.job(), w.outputJob(w.pageSetup(), "pdf", text, nil)
	for _, j := range []exportJob{export, preview} {
		if j.revisions == nil || !j.revisions.PagesOnly || !j.revisions.PageColors {
			t.Errorf("PDF: revisions %+v", j.revisions)
		}
	}
	ed.formatSelect.SetSelected(exportFormatLabel("fdx"))
	if j := ed.job(); j.revisions == nil || j.revisions.PagesOnly {
		t.Errorf("FDX: revisions %+v", j.revisions)
	}
	if w.layoutJob().revisions != nil || w.outputJob(w.pageSetup(), "html", text, nil).revisions != nil {
		t.Error("the page counts or Print took the revisions")
	}
	if s := w.pageSummary(w.pageSetup(), "pdf"); !strings.Contains(s, "revised pages only") {
		t.Errorf("line %q", s)
	}
	p = w.newPageSetup("pdf", nil)
	p.revisionsCheck.SetChecked(false)
	if !p.revisedOnlyCheck.Disabled() {
		t.Error("revised pages only without the marks")
	}
	p.ok()
	if j := w.outputJob(w.pageSetup(), "pdf", text, nil); j.revisions != nil {
		t.Error("marks off: revisions in the PDF")
	}
}

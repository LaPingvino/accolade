package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/pdf"
)

// The page view in the window: beside the script on a wide window, in
// its place on a phone; Close (or Escape on the pages) puts the script
// back.

// showPages shows a drawing in the window's page view: title over it,
// the PDF it is of (name its file's name, for saving), extra over the
// pages (a page setup line, a way back to a dialog).
func (w *MainWindow) showPages(d pdf.Drawing, title, pdfPath, name string, extra ...fyne.CanvasObject) *pageView {
	rebuilt := false
	if w.pages == nil || w.pages.narrow != w.narrow { // (a phone's bar, or a desktop's)
		v := newPageView(w)
		v.onClose = w.hidePages
		w.pages, rebuilt = v, true
	}
	v := w.pages
	v.remake = nil
	v.extra.Objects = extra
	v.extra.Refresh()
	v.setNote("")
	wasVisible := w.pagesVisible
	w.pagesVisible = true
	if !wasVisible || rebuilt {
		w.relayout()
	}
	// the last preview's PDF goes now that these pages are shown (not
	// before: if making them failed, the pages shown keep their PDF)
	if dir := filepath.Dir(pdfPath); pdfPath != "" && strings.HasPrefix(filepath.Base(dir), "accolade-preview-") {
		if w.previewDir != "" && w.previewDir != dir {
			os.RemoveAll(w.previewDir)
		}
		w.previewDir = dir
	}
	v.show(d, title, pdfPath, name)
	pagesInSight = v
	if !wasVisible {
		if c := w.fyneWindow.Canvas(); c != nil {
			c.Focus(v.area)
		}
	}
	return v
}

// pagesInSight is the page view shown (two fingers zoom it in the
// browser), nil if none.
var pagesInSight *pageView

// hidePages closes the page view.
func (w *MainWindow) hidePages() {
	if !w.pagesVisible {
		return
	}
	w.pagesVisible = false
	if pagesInSight == w.pages {
		pagesInSight = nil
	}
	w.relayout()
	w.focusEditor()
}

// focusEditor gives the script the focus.
func (w *MainWindow) focusEditor() {
	if c := w.fyneWindow.Canvas(); c != nil && w.textEditor != nil {
		c.Focus(w.textEditor)
	}
}

// openInSystem opens a file in the computer's viewer for its kind (the
// page view's Open in Viewer).
var openInSystem = func(path string) { systemOpen(path) }

// previewTemp is a new folder for a preview's PDF; the last one's goes
// when its pages are shown (showPages), as the page view shows one PDF
// at a time.
func previewTemp() (string, error) { return os.MkdirTemp("", "accolade-preview-") }

// browserNote says why the pages shown are not the PDF's: a browser
// made the PDF (scripts the PDF writer cannot print as written).
const browserNote = "Your browser made this PDF, for the scripts Lexington's PDF cannot print as written (its letters " +
	"joined, or CJK): the pages shown are Lexington's own. Open in Viewer shows the browser's."

// drawScript prints the script as the job says to out (a PDF), and gives
// its pages as drawn. A PDF a browser made (exportAs) has none of its
// own: Lexington's pages are drawn then, and note says so.
func drawScript(content, out string, job exportJob) (d pdf.Drawing, note string, err error) {
	job.record = &d
	if err := exportAs(content, out, "pdf", job); err != nil {
		return d, "", err
	}
	if len(d.Pages) > 0 {
		return d, "", nil
	}
	own := out + ".lexington.pdf"
	defer os.Remove(own)
	job = withLockedPages(content, "pdf", job)
	o := job.options("pdf", own)
	o.Record = &d
	if err := cli.Write(context.Background(), job.script(content), o); err != nil {
		return d, "", err
	}
	return d, browserNote, nil
}

// pdfName is the name of the script's (or document's) PDF.
func (w *MainWindow) pdfName() string {
	name := w.untitledName()
	if w.currentFile != "" {
		name = filepath.Base(w.currentFile)
	} else if w.suggestedName != "" {
		name = w.suggestedName
	}
	return strings.TrimSuffix(name, filepath.Ext(name)) + ".pdf"
}

// previewLine is Preview PDF's page setup line: a change in Page Setup
// makes the pages again.
func (w *MainWindow) previewLine() *pageSetupLine {
	if w.previewSetup == nil {
		// (its Page Setup… makes the pages again, as File > Page Setup…
		// does: pagesAgain)
		w.previewSetup = &previewSetup{line: w.newPageSetupLine(func() string { return "pdf" }, nil)}
	}
	w.previewSetup.line.refresh()
	return w.previewSetup.line
}

// printedLayout is what the printed page goes by: the script's page
// setup and format, a Markdown document's page setup.
func (w *MainWindow) printedLayout() any {
	if w.settings == nil {
		return nil
	}
	return struct {
		p      pageSetup
		md     mdPageSetup
		format string
	}{w.pageSetup(), w.markdownPageSetup(), w.scriptFormatKey()}
}

// pagesAgain makes the page view's pages again after a change in Page
// Setup, if they follow it (Preview PDF's).
func (w *MainWindow) pagesAgain() {
	if w.pagesVisible && w.pages != nil && w.pages.remake != nil {
		w.pages.remake()
	}
}

// backButton is a button over the pages back to a dialog.
func backButton(label string, back func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, theme.NavigateBackIcon(), back)
	b.Importance = widget.LowImportance
	return b
}

// showPDFError says a PDF could not be made (a revised pages' PDF with no
// revised page says so).
func (w *MainWindow) showPDFError(err error) {
	if errors.Is(err, pdf.ErrNoRevisedPages) {
		dialog.ShowInformation("No revised pages", "No page has a change of this revision on it.", w.fyneWindow)
		return
	}
	dialog.ShowError(fmt.Errorf("could not make the PDF: %v", err), w.fyneWindow)
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/pdf"
)

// The page view in the window: beside the script on a wide window; on a
// phone a middle tab of its own, Pages, beside the script's (middle.go),
// so the way back to the script is a tap; Close (or Escape on the pages,
// or the tab's ×) closes it.

// showPages shows a drawing in the window's page view: title over it,
// the PDF it is of (name its file's name, for saving), extra over the
// pages (a page setup line, a way back to a dialog).
func (w *MainWindow) showPages(d pdf.Drawing, title, pdfPath, name string, extra ...fyne.CanvasObject) *pageView {
	w.pagesPreview = false // the script's Preview says so after (state.go)
	rebuilt := false
	if w.pages == nil || w.pages.narrow != w.narrow { // (a phone's bar, or a desktop's)
		v := newPageView(w)
		v.onClose = w.hidePages
		w.pages, rebuilt = v, true
	}
	v := w.pages
	if v.follow != nil {
		v.follow.stop()
	}
	v.remake, v.follow, v.freshPDF, v.setupLine = nil, nil, nil, nil
	v.extra.Objects = extra
	v.extra.Refresh()
	v.setNote("")
	wasVisible := w.pagesVisible
	w.pagesVisible = true
	back := w.mid.pagesBack
	w.mid.pagesBack = false
	w.pagesTabFollow(!back)
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
	v.followLook()
	pagesInSight.Store(v)
	if !wasVisible {
		if c := w.fyneWindow.Canvas(); c != nil {
			c.Focus(v.area)
		}
	}
	w.ribbonRefresh() // the Preview tab
	return v
}

// pagesInSight is the page view shown (two fingers zoom it in the
// browser), nil if none. Atomic, as pagesShownHook.
var pagesInSight atomic.Pointer[pageView]

// hidePages closes the page view.
func (w *MainWindow) hidePages() {
	if !w.pagesVisible {
		return
	}
	if w.mid.shown == "pages" && !w.mid.showing {
		w.mid.show("") // the script's tab to the front first
	}
	w.pagesVisible = false
	w.pagesTabFollow(false)
	if w.pages.follow != nil {
		w.pages.follow.stop()
		w.pages.follow = nil
	}
	pagesInSight.CompareAndSwap(w.pages, nil)
	w.relayout()
	w.focusEditor()
	w.ribbonRefresh() // the Preview tab goes
}

// pagesTabFollow puts the page view where the window's width says: on a
// phone, while it is shown, a middle tab "Pages" right after the script's
// (front: brought to the front as it comes); else none, the pages beside
// the script (relayout).
func (w *MainWindow) pagesTabFollow(front bool) {
	m := &w.mid
	want := w.narrow && w.pagesVisible && w.pages != nil
	switch {
	case want && !m.has("pages"):
		m.tabs = append([]string{"pages"}, m.tabs...)
		if front {
			m.show("pages")
		}
	case !want && m.has("pages"):
		if m.shown == "pages" && !m.showing {
			m.show("") // (a tab between the script and a view: the pages stay)
		}
		m.tabs = slices.DeleteFunc(m.tabs, func(s string) bool { return s == "pages" })
	}
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

// drawScript prints the script as the job says to out (a PDF), and gives
// its pages as drawn.
func drawScript(content, out string, job exportJob) (d pdf.Drawing, err error) {
	job.record = &d
	err = exportAs(content, out, "pdf", job)
	return d, err
}

// shapingNote is the note over a script's pages when the PDF cannot shape
// the joining letters of some of its scripts (Syriac, N'Ko…: Lexington's
// Unshaped, unshapedScripts), "" when it shapes all: the pages are the
// best the PDF can do; the actions make the PDF through a browser on the
// computer instead, or open the script as HTML, in which the browser
// shapes them.
func (w *MainWindow) shapingNote(scripts []string, content string, job exportJob) (string, []fyne.CanvasObject) {
	if len(scripts) == 0 {
		return "", nil
	}
	note := fmt.Sprintf("Shaping for %s is missing in the PDF: its letters print unjoined. These pages are the best the PDF can do.",
		joinNames(scripts))
	var acts []fyne.CanvasObject
	if b := localBrowser(); b != "" {
		acts = append(acts, noteLink("Make the PDF with the Browser", func() { w.browserPDFOf(content, job, b) }))
	}
	acts = append(acts, noteLink("Open as HTML", func() { w.htmlOf(content, job) }))
	return note, acts
}

// joinNames is scripts' Unicode names as a list in prose ("Syriac",
// "N'Ko and Syriac", "Mongolian, N'Ko and Syriac").
func joinNames(names []string) string {
	names = slices.Clone(names)
	for i, n := range names {
		if n == "Nko" {
			n = "N'Ko"
		}
		names[i] = strings.ReplaceAll(n, "_", " ") // Hanifi_Rohingya
	}
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// noteLink is an action in a note over the pages.
func noteLink(label string, act func()) *widget.Button {
	b := widget.NewButton(label, act)
	b.Importance = widget.LowImportance
	return b
}

// browserPDFOf makes the script's PDF with a browser on the computer
// from Lexington's HTML, and opens it in the system's viewer.
func (w *MainWindow) browserPDFOf(content string, job exportJob, browser string) {
	dir, err := os.MkdirTemp("", "accolade-pdf-")
	if err != nil {
		w.showPDFError(err)
		return
	}
	page, out := filepath.Join(dir, "script.html"), filepath.Join(dir, w.pdfName())
	w.setStatus("Making the PDF with the browser...")
	inBackground(func() error {
		if err := exportAs(content, page, "html", job); err != nil {
			return err
		}
		return browserPDF(browser, page, out)
	}, func(err error) {
		w.setStatus("")
		if err != nil {
			w.showPDFError(err)
			return
		}
		openInSystem(out)
	})
}

// htmlOf opens the script as Lexington's HTML (in the browser, which
// shapes every script).
func (w *MainWindow) htmlOf(content string, job exportJob) {
	dir, err := os.MkdirTemp("", "accolade-html-")
	if err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	page := filepath.Join(dir, strings.TrimSuffix(w.pdfName(), ".pdf")+".html")
	if err := exportAs(content, page, "html", job); err != nil {
		dialog.ShowError(err, w.fyneWindow)
		return
	}
	openFile(page)
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

package main

import (
	"bytes"
	"cmp"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/prose"
)

// A Markdown document's PDF, written by Accolade itself (internal/prose,
// through Lexington's drawing): no Chromium on the desktop, a real PDF in
// the browser, its pages in the page view (the Live Preview's Pages).
// What it cannot print yet as written — text read right to left, scripts
// its fonts have not got — goes through a browser as before
// (exportMarkdownWith).

// proseFonts are the PDF's text face: Noto Sans, Fyne's own (SIL Open
// Font License 1.1, which lets it be embedded), in its four styles; code
// is in Courier Badi, Lexington's (OFL too).
func proseFonts() prose.PDFFonts {
	return prose.PDFFonts{
		Regular: theme.DefaultTextFont().Content(), Bold: theme.DefaultTextBoldFont().Content(),
		Italic: theme.DefaultTextItalicFont().Content(), BoldItalic: theme.DefaultTextBoldItalicFont().Content(),
	}
}

// markdownDoc is the document to print: its body, title, paper and title
// page.
func markdownDoc(text, title string, p mdPageSetup, titlePage []string) *prose.Doc {
	d := prose.FromMarkdown(markdownBody(text))
	d.Title = cmp.Or(d.Title, title)
	d.Paper, d.TitlePage = p.paper, titlePage
	d.NotesAt, d.NotesPerChapter = p.notesAt, p.notesPerChapter
	return d
}

// markdownPDFMissing says what of the document Accolade's PDF cannot
// print as written ("" if it prints all of it).
func markdownPDFMissing(text string) string {
	return prose.FromMarkdown(markdownBody(text)).PDFMissing(proseFonts())
}

// markdownPDF is the document's PDF as Accolade writes it, and its pages.
func markdownPDF(text, title string, p mdPageSetup, titlePage []string) ([]byte, pdf.Drawing, error) {
	var b bytes.Buffer
	d, err := markdownDoc(text, title, p, titlePage).WritePDF(&b, proseFonts())
	return b.Bytes(), d, err
}

// mdPDFByPrinting: the document's PDF is made in the print dialog — one
// Accolade cannot print, with no Chromium on the computer to make it.
func mdPDFByPrinting(text string) bool {
	return markdownPDFMissing(text) != "" && pdfByPrinting()
}

// mdPages is the Live Preview's Pages: the document's PDF in a page
// view, made again a moment after the typing stops.
type mdPages struct {
	w     *MainWindow
	view  *pageView
	mu    sync.Mutex
	timer *time.Timer
	gen   int
	dir   string
}

func newMDPages(w *MainWindow) *mdPages {
	m := &mdPages{w: w, view: newPageView(w)}
	m.view.closeB.Hide() // the preview's own: Text shows the text again
	// Refresh: the pages made at once
	m.view.remake = func() { m.updateAfter(w.writeEditor().Text(), 0) }
	return m
}

// mdPagesDelay is how long after the last change the pages are made.
var mdPagesDelay = 400 * time.Millisecond

// mdPagesAfter runs f after d (the tests run it at once).
var mdPagesAfter = time.AfterFunc

// update makes the pages of text, soon.
func (m *mdPages) update(text string) { m.updateAfter(text, mdPagesDelay) }

// updateAfter makes the pages of text after delay.
func (m *mdPages) updateAfter(text string, delay time.Duration) {
	m.mu.Lock()
	m.gen++
	gen := m.gen
	if m.timer != nil {
		m.timer.Stop()
	}
	m.mu.Unlock()
	w := m.w
	name := strings.TrimSuffix(w.pdfName(), ".pdf")
	title := markdownTitle(text, w.scriptName())
	page := w.markdownPageSetup()
	titlePage := page.titlePageOf(text, w.scriptName())
	t := mdPagesAfter(delay, func() {
		data, d, err := markdownPDF(text, title, page, titlePage)
		missing := markdownPDFMissing(text)
		m.mu.Lock()
		stale := gen != m.gen
		if m.dir == "" {
			m.dir, _ = os.MkdirTemp("", "accolade-pages-")
		}
		out := filepath.Join(m.dir, name+".pdf")
		m.mu.Unlock()
		if stale || err != nil {
			return
		}
		if os.WriteFile(out, data, 0o644) != nil {
			out = ""
		}
		fyne.Do(func() {
			if gen != m.gen {
				return
			}
			note := ""
			if missing != "" {
				note = "This PDF cannot print the document as written yet (" + missing + "): Export makes it through a browser."
			}
			m.view.setNote(note)
			m.view.show(d, name+".pdf", out, name+".pdf")
			pagesShown(m.view)
		})
	})
	m.mu.Lock()
	if gen == m.gen {
		m.timer = t
	} else {
		t.Stop()
	}
	m.mu.Unlock()
}

func (m *mdPages) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.timer != nil {
		m.timer.Stop()
	}
	m.gen++
}

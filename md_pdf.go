package main

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/theme"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/cite"
	"github.com/LaPingvino/accolade/internal/prose"
)

// A Markdown document's PDF, written by Accolade itself (internal/prose,
// through Lexington's drawing): no Chromium on the desktop, a real PDF in
// the browser, its pages in the page view (the Live Preview's Pages).
// Text read right to left it prints too (Hebrew, Arabic: ordered,
// joined and set from the right, as Lexington's screenplay PDF prints
// it). What it cannot print yet as written — scripts its fonts have not
// got (CJK) — goes through a browser as before (exportMarkdownWith).

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
	d := prose.FromMarkdown(citedBody(text, p.lib))
	d.Title = cmp.Or(d.Title, title)
	d.Paper, d.TitlePage = p.paper, titlePage
	d.NotesAt, d.NotesPerChapter = p.notesAt, p.notesPerChapter
	d.Images, d.AltCaptions = p.pics, p.altCaptions
	return d
}

// markdownPDFMissing says what of the document Accolade's PDF cannot
// print as written ("" if it prints all of it), its citations cited
// from lib.
func markdownPDFMissing(text string, lib *cite.Library) string {
	return prose.FromMarkdown(citedBody(text, lib)).PDFMissing(proseFonts())
}

// markdownPDF is the document's PDF as Accolade writes it, and its pages.
func markdownPDF(text, title string, p mdPageSetup, titlePage []string) ([]byte, pdf.Drawing, error) {
	var b bytes.Buffer
	d, err := markdownDoc(text, title, p, titlePage).WritePDF(&b, proseFonts())
	return b.Bytes(), d, err
}

// mdPDFByPrinting: the document's PDF is made in the print dialog — one
// Accolade cannot print, with no Chromium on the computer to make it.
func mdPDFByPrinting(text string, lib *cite.Library) bool {
	return markdownPDFMissing(text, lib) != "" && pdfByPrinting()
}

// mdPages is the Live Preview's Pages: the document's PDF in a page
// view, made again a moment after the typing stops.
type mdPages struct {
	w     *MainWindow
	view  *pageView
	timer *uiTimer // the pause before they are made
	gen   int      // the latest making: an older one's pages are dropped
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

// update makes the pages of text, soon.
func (m *mdPages) update(text string) { m.updateAfter(text, mdPagesDelay) }

// updateAfter makes the pages of text after delay, in the background
// (background.go: the tests' at once).
func (m *mdPages) updateAfter(text string, delay time.Duration) {
	m.gen++
	gen := m.gen
	w := m.w
	name := strings.TrimSuffix(w.pdfName(), ".pdf")
	title := markdownTitle(text, w.scriptName())
	page := w.markdownPageSetup()
	titlePage := page.titlePageOf(text, w.scriptName())
	if m.dir == "" {
		m.dir, _ = os.MkdirTemp("", "accolade-pages-")
	}
	out := filepath.Join(m.dir, name+".pdf")
	type made struct {
		d       pdf.Drawing
		missing string
		out     string
		err     error
	}
	w.debounce(&m.timer, delay, func() {
		if gen != m.gen {
			return
		}
		// written beside it first: a making finishing late, made stale by
		// a newer one, does not replace the newer one's PDF
		tmp := fmt.Sprintf("%s.%d.tmp", out, gen)
		inBackground(func() made {
			data, d, err := markdownPDF(text, title, page, titlePage)
			r := made{d: d, missing: markdownPDFMissing(text, page.lib), out: tmp, err: err}
			if err == nil && os.WriteFile(tmp, data, 0o644) != nil {
				r.out = ""
			}
			return r
		}, func(r made) {
			if gen != m.gen || r.err != nil {
				os.Remove(tmp)
				return
			}
			if r.out != "" && os.Rename(tmp, out) == nil {
				r.out = out
			}
			note := ""
			if r.missing != "" {
				note = "This PDF cannot print the document as written yet (" + r.missing + "): Export makes it through a browser."
			}
			m.view.setNote(note)
			m.view.show(r.d, name+".pdf", r.out, name+".pdf")
			pagesShown(m.view)
		})
	})
}

// stop drops the pages being made.
func (m *mdPages) stop() {
	cancel(&m.timer)
	m.gen++
}

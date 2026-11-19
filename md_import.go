package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"github.com/LaPingvino/accolade/internal/prose"
	"github.com/LaPingvino/lexington/pdfin"
)

// File > Import (docs/ADAPTATION.md, step 2): a story, an article, a book
// into a new Markdown document, to adapt. Word, OpenDocument, web pages
// and PDFs are read by Accolade itself (internal/prose, Lexington's pdfin
// for PDFs), in the browser too; with pandoc on the computer, File >
// Import with pandoc reads what pandoc reads (EPUB, RTF, LaTeX ...).

// importExtensions are the files Import reads itself.
var importExtensions = []string{".docx", ".odt", ".html", ".htm", ".xhtml", ".pdf", ".txt", ".md", ".markdown"}

// pandocFormats are the files Import with pandoc offers, and pandoc's
// names for them.
var pandocFormats = map[string]string{
	".epub": "epub", ".rtf": "rtf", ".tex": "latex", ".latex": "latex", ".docx": "docx", ".odt": "odt",
	".html": "html", ".htm": "html", ".org": "org", ".rst": "rst", ".fb2": "fb2", ".textile": "textile",
	".dbk": "docbook", ".typ": "typst", ".md": "markdown", ".txt": "markdown",
}

// pandocExtensions are pandocFormats' extensions, in order.
func pandocExtensions() []string {
	var exts []string
	for e := range pandocFormats {
		exts = append(exts, e)
	}
	slices.Sort(exts)
	return exts
}

// hasPandoc: pandoc is there to import and export with (never in the
// browser).
func hasPandoc() bool { return !inBrowser && pandocPath() != "" }

// toMarkdown reads a document (not a PDF: importProsePDF) as Markdown;
// dropped counts the images and objects left out, from says what it was.
func toMarkdown(path string) (md string, dropped int, from string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, "", err
	}
	var d *prose.Doc
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		d, err = prose.ReadDOCX(bytes.NewReader(data), int64(len(data)))
		from = "a Word document"
	case ".odt":
		d, err = prose.ReadODT(bytes.NewReader(data), int64(len(data)))
		from = "an OpenDocument text"
	case ".html", ".htm", ".xhtml":
		d, err = prose.ReadHTML(bytes.NewReader(data))
		from = "a web page"
	case ".txt", ".md", ".markdown":
		text, _ := decodeScript(data)
		return text, 0, "a text file", nil
	default:
		return "", 0, "", fmt.Errorf("Accolade does not read %s files itself", filepath.Ext(path))
	}
	if err != nil {
		return "", 0, "", err
	}
	return d.Markdown(), d.Dropped, from, nil
}

// pandocToMarkdown reads a document with pandoc as Markdown (GitHub's,
// without raw HTML: pandoc's anchors and spans would be clutter).
func pandocToMarkdown(path string) (string, error) {
	format, ok := pandocFormats[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", fmt.Errorf("pandoc is not asked to read %s files", filepath.Ext(path))
	}
	dir, err := os.MkdirTemp("", "accolade-pandoc-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "imported.md")
	if err := runPandoc("-f", format, "-t", "gfm-raw_html", "--wrap=none", "-o", out, path); err != nil {
		return "", err
	}
	b, err := os.ReadFile(out)
	return string(b), err
}

// importProsePDF reads a PDF as prose, as Markdown; scanned pages are read
// with OCR in ocrLang. skipped are the pages that could not be read.
func importProsePDF(ctx context.Context, path, ocrLang string, progress func(done, pages int)) (md string, skipped []string, err error) {
	opts := pdfin.Options{Context: ctx, Progress: progress, Skipped: func(page int, err error) {
		skipped = append(skipped, fmt.Sprintf("page %d: %v", page, err))
	}}
	if ocr, done := pdfOCR(ocrLang); ocr != nil {
		defer done()
		opts.OCR = ocr
	}
	p, err := pdfin.ReadProseFile(path, opts)
	if err != nil {
		return "", skipped, err
	}
	return p.Markdown(), skipped, nil
}

// chooseFile lets the user pick a file with one of exts: the browser's
// file picker, or Fyne's file dialog.
func (w *MainWindow) chooseFile(exts []string, then func(path string)) {
	if inBrowser {
		pickFile(exts, then)
		return
	}
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		r.Close()
		then(r.URI().Path())
	}, w.fyneWindow)
	d.SetFilter(storage.NewExtensionFileFilter(exts))
	d.Show()
}

// ImportDocument is File > Import: a document into a new Markdown one.
func (w *MainWindow) ImportDocument() {
	w.chooseFile(importExtensions, func(p string) { w.withTarget(func(t *MainWindow) { t.importDocument(p) }) })
}

// ImportWithPandoc is File > Import with pandoc.
func (w *MainWindow) ImportWithPandoc() {
	w.chooseFile(pandocExtensions(), func(p string) {
		w.withTarget(func(t *MainWindow) {
			t.setStatus("Importing " + filepath.Base(p) + " with pandoc...")
			go func() {
				md, err := pandocToMarkdown(p)
				fyne.Do(func() {
					if err != nil {
						dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(p), err), t.fyneWindow)
						return
					}
					t.showImportedMarkdown(p, md, "pandoc", 0)
				})
			}()
		})
	})
}

// importDocument reads a document into this window as a new Markdown
// document: a PDF in the background, the rest at once.
func (w *MainWindow) importDocument(path string) {
	if strings.EqualFold(filepath.Ext(path), ".pdf") {
		w.importProsePDFAsync(path)
		return
	}
	md, dropped, from, err := toMarkdown(path)
	if err != nil {
		dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(path), err), w.fyneWindow)
		return
	}
	w.showImportedMarkdown(path, md, from, dropped)
}

// importProsePDFAsync reads a PDF as prose in the background, asking
// first what language scanned pages are in.
func (w *MainWindow) importProsePDFAsync(path string) {
	read := func(lang string) {
		w.readPDFInBackground(path, "the text", func(ctx context.Context, progress func(done, pages int)) (string, []string, error) {
			return importProsePDF(ctx, path, lang, progress)
		}, func(md string) {
			w.showImportedMarkdown(path, md, "a PDF: check the result, a PDF only shows how the text looked", 0)
		})
	}
	if scanned, pages, err := pdfin.ScannedPages(path); err == nil && scanned > 0 {
		if langs, tesseract := ocrLanguages(); len(langs) > 0 {
			w.askOCRLanguage(path, scanned, pages, w.guessOCRLanguage(languageOf(path), langs), langs, tesseract, read)
			return
		}
	}
	read("")
}

// showImportedMarkdown puts an imported document in the window as a new
// Markdown document, unsaved, named after the file it came from.
func (w *MainWindow) showImportedMarkdown(path, md, from string, dropped int) {
	w.cancelAutoSave()
	w.format = fileFormat{}
	w.setMarkdown(true)
	w.textEditor.SetText(md)
	w.textEditor.SetCursorOffset(0)
	w.filledWith = w.textEditor.Text()
	w.currentFile = ""
	w.suggestedName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".md"
	w.hasChanges = true
	w.updateTitle()
	w.updateStatusBar()
	status := "Imported from " + from
	if dropped > 0 {
		status += fmt.Sprintf("; %s left out, marked <!-- image left out --> in the text", plural(dropped, "image"))
	}
	w.setStatus(status)
}

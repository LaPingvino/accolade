package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/LaPingvino/accolade/internal/images"
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
	return toMarkdownSaving(path, nil)
}

// toMarkdownSaving is toMarkdown keeping the document's pictures: save
// (nil: none saved, the Markdown names files not there) puts each where
// the Markdown links it from. A web page's pictures beside it are
// copied too; one on the web stays a link, never fetched.
func toMarkdownSaving(path string, save func(name string, data []byte) (string, error)) (md string, dropped int, from string, err error) {
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
	if save != nil {
		dir := images.Dir(filepath.Dir(path))
		d.EachImage(func(img *prose.Image) {
			if img.Data == nil && !images.IsRemote(img.Src) && !strings.HasPrefix(strings.ToLower(img.Src), "data:") {
				if p := images.New(dir).Load(img.Src); p.Data != nil {
					img.Data, img.Src = p.Orig, filepath.Base(filepath.FromSlash(img.Src))
				}
			}
		})
		_ = d.SaveImages(save) // one that could not be saved is left out (a comment), as before
	}
	return d.Markdown(), d.Dropped + d.Unsaved(), from, nil
}

// importedPictures saves an imported document's pictures: in a folder
// named after it beside the file it came from (Name.assets/, as Typora
// keeps them), linked by that folder's name — the document, saved in
// that folder, finds them; until then they are read from there. In the
// browser, with no folder, each is kept in the text as a data: URI.
func importedPictures(path string) func(name string, data []byte) (string, error) {
	if inBrowser {
		return func(_ string, data []byte) (string, error) {
			return "data:" + mimeOf(data) + ";base64," + base64.StdEncoding.EncodeToString(data), nil
		}
	}
	folder := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".assets"
	dir := filepath.Join(filepath.Dir(path), folder)
	return func(name string, data []byte) (string, error) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		name = uniqueName(dir, name, data)
		if err := writeFileAtomic(filepath.Join(dir, name), data); err != nil {
			return "", err
		}
		return (&url.URL{Path: folder + "/" + name}).String(), nil
	}
}

// uniqueName is name in dir, numbered if a different file has it (the
// same file there already: its name).
func uniqueName(dir, name string, data []byte) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		old, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || bytes.Equal(old, data) {
			return name
		}
		name = fmt.Sprintf("%s-%d%s", base, i+1, ext)
	}
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
// with OCR in ocrLang, and the scans whose text looks garbled too if
// ocrText. skipped are the pages that could not be read.
func importProsePDF(ctx context.Context, path, ocrLang string, ocrText bool, progress func(done, pages int)) (md string, skipped []string, err error) {
	opts := pdfin.Options{Context: ctx, Progress: progress, Skipped: func(page int, err error) {
		skipped = append(skipped, fmt.Sprintf("page %d: %v", page, err))
	}}
	if ocr, done := pdfOCR(ocrLang); ocr != nil {
		defer done()
		opts.OCR, opts.OCRText = ocr, ocrText
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
	chooseFileIn(w.fyneWindow, exts, then)
}

// chooseFileIn is chooseFile over any window (Preferences' too): the
// browser's own file picker there, Fyne's open dialog on the desktop. A
// test stands in for the person choosing.
var chooseFileIn = func(win fyne.Window, exts []string, then func(path string)) {
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
	}, win)
	setPickFilters(d, exts, "")
	showPicker(d, win)
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
			type read struct {
				md  string
				err error
			}
			inBackground(func() read {
				md, err := pandocToMarkdown(p)
				return read{md, err}
			}, func(r read) {
				if r.err != nil {
					dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(p), r.err), t.fyneWindow)
					return
				}
				t.showImportedMarkdown(p, r.md, "pandoc", 0)
			})
		})
	})
}

// importDocument reads a document into this window as a new Markdown
// document, in the background (background.go).
func (w *MainWindow) importDocument(path string) {
	if strings.EqualFold(filepath.Ext(path), ".pdf") {
		w.importProsePDFAsync(path)
		return
	}
	type read struct {
		md, from string
		dropped  int
		err      error
	}
	w.importGen++
	gen := w.importGen
	w.setStatus("Reading " + filepath.Base(path) + "…")
	inBackground(func() read { // converting a novel took seconds in the browser
		var r read
		r.md, r.dropped, r.from, r.err = toMarkdownSaving(path, importedPictures(path))
		return r
	}, func(r read) {
		if gen != w.importGen {
			return // another import since
		}
		if r.err != nil {
			dialog.ShowError(fmt.Errorf("could not import %s: %v", filepath.Base(path), r.err), w.fyneWindow)
			return
		}
		w.showImportedMarkdown(path, r.md, r.from, r.dropped)
	})
}

// importProsePDFAsync reads a PDF as prose in the background, asking
// first what language scanned pages are in.
func (w *MainWindow) importProsePDFAsync(path string) {
	read := func(lang string, ocrText bool) {
		w.readPDFInBackground(path, "the text", func(ctx context.Context, progress func(done, pages int)) (string, []string, error) {
			return importProsePDF(ctx, path, lang, ocrText, progress)
		}, func(md string) {
			w.showImportedMarkdown(path, md, "a PDF: check the result, a PDF only shows how the text looked", 0)
		})
	}
	w.readPDFAsking(path, languageOf(path), read, func() { read("", false) })
}

// showImportedMarkdown puts an imported document in the window as a new
// Markdown document, unsaved, named after the file it came from.
func (w *MainWindow) showImportedMarkdown(path, md, from string, dropped int) {
	// the views follow at the end (state.go)
	defer w.scriptChange()()
	w.newDocument()
	w.cancelAutoSave()
	w.forgetUnsavedSource()
	w.format = fileFormat{}
	w.setMarkdown(true)
	w.textEditor.SetText(md)
	w.textEditor.SetCursorOffset(0)
	w.filledWith = w.textEditor.Text()
	w.currentFile, w.plainBundle = "", ""
	w.setBundle(nil)
	w.setCarried(nil, "")
	w.suggestedName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".md"
	w.unsavedDir = filepath.Dir(path) // its pictures are beside it (importedPictures)
	w.hasChanges = true
	status := "Imported from " + from + ". To adapt it: Start a Screenplay, on the toolbar"
	if dropped > 0 {
		status += fmt.Sprintf("; %s left out, marked <!-- image left out --> in the text", plural(dropped, "object"))
	}
	w.setStatus(status)
}

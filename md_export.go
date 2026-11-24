package main

import (
	"bytes"
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/prose"
)

// Export for a Markdown document (docs/ADAPTATION.md, step 2): a web
// page, Word, OpenDocument, a PDF and the Markdown as written, made by
// Accolade itself (internal/prose, goldmark), in the browser too; a PDF
// of what Accolade's cannot print yet (CJK: letters its fonts have not
// got) through a browser (headless Chromium on the computer, else the
// print dialog's Save as PDF); with pandoc on the computer, what pandoc writes too. A
// screenplay's Export is the other dialog (export_dialog.go), unchanged.

// mdExportFormat is a format a Markdown document is exported to.
type mdExportFormat struct {
	id, label, ext string
	pandoc         string // pandoc's -t for it, "" if Accolade writes it
}

var builtinMarkdownExports = []mdExportFormat{
	{id: "html", label: "Web page (HTML)", ext: "html"},
	{id: "docx", label: "Word (DOCX)", ext: "docx"},
	{id: "odt", label: "OpenDocument (ODT)", ext: "odt"},
	{id: "pdf", label: "PDF", ext: "pdf"},
	{id: "md", label: "Markdown (as written)", ext: "md"},
	{id: "txt", label: "Plain text", ext: "txt"},
}

var pandocMarkdownExports = []mdExportFormat{
	{id: "pandoc-epub", label: "EPUB (pandoc)", ext: "epub", pandoc: "epub"},
	{id: "pandoc-rtf", label: "RTF (pandoc)", ext: "rtf", pandoc: "rtf"},
	{id: "pandoc-latex", label: "LaTeX (pandoc)", ext: "tex", pandoc: "latex"},
	{id: "pandoc-docx", label: "Word (DOCX, pandoc)", ext: "docx", pandoc: "docx"},
	{id: "pandoc-odt", label: "OpenDocument (ODT, pandoc)", ext: "odt", pandoc: "odt"},
	{id: "pandoc-org", label: "Org mode (pandoc)", ext: "org", pandoc: "org"},
}

// markdownExports are the formats offered: Accolade's own, and pandoc's
// when pandoc is there.
func markdownExports() []mdExportFormat {
	list := append([]mdExportFormat(nil), builtinMarkdownExports...)
	if hasPandoc() {
		list = append(list, pandocMarkdownExports...)
	}
	return list
}

func markdownExportByLabel(label string) mdExportFormat {
	list := markdownExports()
	for _, f := range list {
		if f.label == label || f.id == label {
			return f
		}
	}
	return list[0]
}

// pdfByPrinting: a PDF is made in the print dialog (no Chromium on the
// computer, or the browser version), not written by the export.
func pdfByPrinting() bool { return inBrowser || localBrowser() == "" }

// markdownTitle is a document's title: its first heading, else name.
func markdownTitle(text, name string) string {
	if t := prose.FromMarkdown(markdownBody(text)).Title; t != "" {
		return t
	}
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
}

// exportMarkdown writes the Markdown document text as format id to out;
// the front matter and the notes are left out (as in the preview), except
// in the Markdown as written.
func exportMarkdown(text, title, out, id string) error {
	return exportMarkdownWith(text, title, out, id, mdPageSetup{}, nil)
}

// exportMarkdownWith is exportMarkdown with the page setup p (the web
// page, Word, ODT and the PDF use it), titlePage its title page's lines.
func exportMarkdownWith(text, title, out, id string, p mdPageSetup, titlePage []string) error {
	f := markdownExportByLabel(id)
	if f.id != id {
		return fmt.Errorf("unknown export format %q", id)
	}
	body := markdownBody(text)
	switch id {
	case "md":
		return writeFileAtomic(out, []byte(text))
	case "txt":
		return writeFileAtomic(out, []byte(prose.FromMarkdown(citedBody(text, p.lib)).Plain()))
	case "html":
		page, err := markdownHTMLWith(text, title, p, titlePage)
		if err != nil {
			return err
		}
		return writeFileAtomic(out, page)
	case "docx", "odt":
		d := markdownDoc(text, title, p, titlePage)
		var b bytes.Buffer
		var err error
		if id == "docx" {
			err = d.WriteDOCX(&b)
		} else {
			err = d.WriteODT(&b)
		}
		if err != nil {
			return err
		}
		return writeFileAtomic(out, b.Bytes())
	case "pdf":
		// Accolade's own (md_pdf.go); a browser's for what it cannot print
		missing := markdownPDFMissing(text, p.lib)
		if missing == "" {
			data, _, err := markdownPDF(text, title, p, titlePage)
			if err != nil {
				return err
			}
			return writeFileAtomic(out, data)
		}
		browser := localBrowser()
		if browser == "" {
			return fmt.Errorf("the PDF cannot print the document as written (%s), and no Chromium browser (Chrome, Chromium, Edge, Brave) is installed to make it: use Print and Save as PDF", missing)
		}
		return withPage(text, title, p, titlePage, func(page string) error {
			return writeViaTemp(out, func(tmp string) error { return browserPDF(browser, page, tmp) })
		})
	}
	// pandoc's: from the Markdown as GitHub writes it, with footnotes
	dir, err := os.MkdirTemp("", "accolade-pandoc-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "document.md")
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		return err
	}
	if err := pandocFiles(dir, p.cite); err != nil { // the library it cites from (citations.go)
		return err
	}
	return writeViaTemp(out, func(tmp string) error {
		return runPandoc(pandocArgs(f.pandoc, title, src, tmp, dir, p.cite)...)
	})
}

// withPage writes the document as a web page to a temporary file for do.
func withPage(text, title string, p mdPageSetup, titlePage []string, do func(page string) error) error {
	dir, err := os.MkdirTemp("", "accolade-page-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	page, err := markdownHTMLWith(text, title, p, titlePage)
	if err != nil {
		return err
	}
	page = inlinePictures(page, p.pics) // (from a temporary folder)
	path := filepath.Join(dir, "document.html")
	if err := os.WriteFile(path, page, 0o644); err != nil {
		return err
	}
	return do(path)
}

// markdownExportDialog is File > Export for a Markdown document.
type markdownExportDialog struct {
	w        *MainWindow
	dialog   *dialog.CustomDialog
	format   *widget.Select
	dir      *widget.Entry
	name     *widget.Entry
	dirRow   fyne.CanvasObject
	note     *widget.Label
	status   *widget.Label
	export   *widget.Button
	settings *Settings
	pageLine *pageSetupLine // the document's page setup
}

func newMarkdownExportDialog(w *MainWindow) *markdownExportDialog {
	w.countsNow()
	ed := &markdownExportDialog{w: w, settings: GetSettings()}
	var labels []string
	for _, f := range markdownExports() {
		labels = append(labels, f.label)
	}
	ed.dir = widget.NewEntry()
	ed.dir.SetText(cmp.Or(ed.settings.GetString("export-directory"), defaultExportDir()))
	ed.name = widget.NewEntry()
	ed.note = widget.NewLabel("")
	ed.note.Wrapping = fyne.TextWrapWord
	ed.status = widget.NewLabel("")
	ed.format = widget.NewSelect(labels, func(string) { ed.formatChanged() })
	browse := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(u fyne.ListableURI, err error) {
			if err == nil && u != nil {
				ed.dir.SetText(u.Path())
			}
		}, w.fyneWindow)
	})
	fyne.SetAccessibleLabel(browse, "Choose the folder")
	ed.dirRow = labelledWith("Directory:", ed.dir, browse)
	ed.export = widget.NewButtonWithIcon("Export", theme.DocumentSaveIcon(), ed.run)
	ed.export.Importance = widget.HighImportance
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { ed.dialog.Hide() })
	ed.pageLine = w.newPageSetupLine(func() string { return "" }, nil)
	form := container.NewVBox(
		labelled("Format:", ed.format),
		ed.dirRow,
		labelled("Filename:", ed.name),
		ed.note,
		ed.pageLine.box,
		ed.status,
	)
	if inBrowser { // the file is downloaded
		ed.dirRow.Hide()
	}
	if !inBrowser && !hasPandoc() {
		hint := widget.NewLabel("With pandoc installed, EPUB, RTF, LaTeX and more are offered too.")
		hint.Wrapping = fyne.TextWrapWord
		hint.Importance = widget.LowImportance
		form.Add(hint)
	}
	content := container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), cancel, ed.export)),
		nil, nil, container.NewPadded(form))
	ed.dialog = dialog.NewCustomWithoutButtons("Export Document", content, w.fyneWindow)
	ed.dialog.SetOnEscape(func() { ed.dialog.Hide() })
	ed.dialog.Resize(fitDialog(w.fyneWindow, 560, 440))
	ed.format.SetSelected(markdownExportByLabel(ed.settings.GetString("markdown-export-format")).label)
	return ed
}

func defaultExportDir() string {
	if inBrowser {
		return "/files"
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Documents")
	}
	return "."
}

func (ed *markdownExportDialog) Show() { ed.dialog.Show() }

// formatChanged names the file for the format and says what it makes.
func (ed *markdownExportDialog) formatChanged() {
	f := markdownExportByLabel(ed.format.Selected)
	base := strings.TrimSuffix(ed.w.scriptName(), filepath.Ext(ed.w.scriptName()))
	ed.name.SetText(base + "." + f.ext)
	notes := map[string]string{
		"html": "A web page, as the preview shows the document.",
		"docx": "Headings, quotes and lists in Word's own styles, footnotes as footnotes: restyle them in Word or LibreOffice.",
		"odt":  "Headings, quotes and lists in LibreOffice's own styles, footnotes as footnotes.",
		"md":   "The document as it is, a copy of the .md file.",
		"txt":  "The text without Markdown's marks: headings, paragraphs and lists, footnotes at the end.",
	}
	note := notes[f.id]
	if f.pandoc != "" {
		note = "Written by pandoc, from the document's Markdown."
	}
	enable := func(o fyne.Disableable, on bool) {
		if on {
			o.Enable()
		} else {
			o.Disable()
		}
	}
	text := ed.w.textEditor.Text()
	printing := f.id == "pdf" && mdPDFByPrinting(text, ed.w.library())
	if f.id == "pdf" {
		note = "Written by Accolade: headings, emphasis, lists, quotes, code, tables and footnotes, on the page setup's paper; Live Preview's Pages shows it."
		if missing := markdownPDFMissing(text, ed.w.library()); missing != "" {
			note = "Made by Chromium from the web page, without a dialog (Accolade's PDF cannot print " + missing + " yet)."
		}
		if printing {
			note = "Opens the print dialog with the document as a web page: choose Save as PDF there."
			if !inBrowser {
				note += " (Install Chrome, Chromium, Edge or Brave to have Export write the PDF itself.)"
			}
		}
	}
	enable(ed.dir, !printing)
	enable(ed.name, !printing)
	ed.note.SetText(note)
	ed.status.SetText("")
}

// run exports, asking first before a file is replaced.
func (ed *markdownExportDialog) run() {
	f := markdownExportByLabel(ed.format.Selected)
	ed.settings.SetString("markdown-export-format", f.id)
	if f.id == "pdf" && mdPDFByPrinting(ed.w.textEditor.Text(), ed.w.library()) {
		_ = ed.settings.Save()
		ed.dialog.Hide()
		ed.w.printMarkdown()
		return
	}
	dir := strings.TrimSpace(ed.dir.Text)
	if inBrowser {
		dir = "/files"
		_ = os.MkdirAll(dir, 0o755)
	}
	name := strings.TrimSpace(ed.name.Text)
	if dir == "" || name == "" {
		dialog.ShowError(fmt.Errorf("choose a directory and a file name"), ed.w.fyneWindow)
		return
	}
	if _, err := os.Stat(dir); err != nil {
		dialog.ShowError(fmt.Errorf("the directory %s does not exist", dir), ed.w.fyneWindow)
		return
	}
	out := filepath.Join(dir, name)
	if ed.w.currentFile != "" && sameFile(out, ed.w.currentFile) {
		dialog.ShowError(fmt.Errorf("%s is the document itself: choose another name or directory", name), ed.w.fyneWindow)
		return
	}
	ed.settings.SetString("export-directory", dir)
	_ = ed.settings.Save()
	start := func() {
		text, title := ed.w.textEditor.Text(), markdownTitle(ed.w.textEditor.Text(), ed.w.scriptName())
		page := ed.w.markdownPageSetup()
		titlePage := page.titlePageOf(text, ed.w.scriptName())
		ed.export.Disable()
		ed.status.SetText("Exporting...")
		cites := ed.w.citeInputsFor(text) // where the library may be: read in the background
		inBackground(func() error {
			switch {
			case f.pandoc != "":
				c, err := citeExportFor(text, cites)
				if err != nil {
					return err
				}
				page.cite = c
			case page.lib == nil && f.id != "md": // Accolade's own: cites from the library, read now if the window has not yet
				page.lib = libraryNow(text, cites)
			}
			return exportMarkdownWith(text, title, out, f.id, page, titlePage)
		}, func(err error) {
			ed.export.Enable()
			if err != nil {
				ed.status.SetText("Export failed")
				dialog.ShowError(fmt.Errorf("export failed: %v", err), ed.w.fyneWindow)
				return
			}
			ed.dialog.Hide()
			if inBrowser {
				offerDownload(out)
			}
			ed.w.setStatus("Exported to " + out)
		})
	}
	if _, err := os.Stat(out); err == nil && !inBrowser {
		dialog.ShowConfirm("Replace File", fmt.Sprintf("%s exists. Replace it?", name), func(ok bool) {
			if ok {
				start()
			}
		}, ed.w.fyneWindow)
		return
	}
	start()
}

package main

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// A PDF is a screenplay's or a novel's: Open reads it as the kind it is
// (fileKindOf, file_picker.go) and offers the other in a line under the
// script; File > Import PDF as Screenplay… and as Markdown… choose.

// openedAsBar is that line: what the PDF was opened as, and the other way.
type openedAsBar struct {
	box   *fyne.Container
	text  *widget.Label
	other *widget.Button
	path  string
	asMD  bool // opened as Markdown: the other way is as a screenplay
}

func (w *MainWindow) newOpenedAsBar() *fyne.Container {
	b := &openedAsBar{text: widget.NewLabel("")}
	b.text.Truncation = fyne.TextTruncateEllipsis
	b.other = widget.NewButton("", func() { w.openPDFTheOtherWay() })
	b.other.Importance = widget.LowImportance
	closeBar := widget.NewButtonWithIcon("", theme.CancelIcon(), w.hideOpenedAs)
	closeBar.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(closeBar, "Close")
	b.box = container.NewBorder(nil, nil, nil, container.NewHBox(b.other, closeBar), b.text)
	b.box.Hide()
	w.openedAs = b
	return b.box
}

// openPDF opens a PDF as the kind it reads as: a novel's as Markdown,
// any other (a screenplay's, a scan, one not told in time) as a
// screenplay; the line offers the other way.
func (w *MainWindow) openPDF(path string) {
	w.setStatus("Reading " + filepath.Base(path) + "…")
	inBackground(func() string { return fileKindOf(path) }, func(kind string) {
		w.openPDFAs(path, kind == kindNovel)
		w.showOpenedAs(path, kind == kindNovel)
	})
}

func (w *MainWindow) openPDFAs(path string, asMarkdown bool) {
	if asMarkdown {
		w.importProsePDFAsync(path)
		return
	}
	w.importPDFAsync(path)
}

func (w *MainWindow) showOpenedAs(path string, asMarkdown bool) {
	b := w.openedAs
	if b == nil {
		return
	}
	b.path, b.asMD = path, asMarkdown
	if asMarkdown {
		b.text.SetText("Opened as Markdown: it reads as a novel")
		b.other.SetText("Open as a Screenplay Instead")
	} else {
		b.text.SetText("Opened as a screenplay")
		b.other.SetText("Open as Markdown Instead")
	}
	b.box.Show()
}

func (w *MainWindow) hideOpenedAs() {
	if w.openedAs != nil {
		w.openedAs.box.Hide()
	}
}

// openPDFTheOtherWay reads the PDF again as the other kind, in place of
// the first reading.
func (w *MainWindow) openPDFTheOtherWay() {
	b := w.openedAs
	if b == nil || b.path == "" {
		return
	}
	w.hideOpenedAs()
	path, asMD := b.path, !b.asMD
	// this window if the reading is unchanged, else as a new script asks
	w.withTarget(func(t *MainWindow) { t.openPDFAs(path, asMD) })
}

// ImportPDFAsScreenplay is File > Import PDF as Screenplay….
func (w *MainWindow) ImportPDFAsScreenplay() {
	w.choosePDF(kindScreenplay, func(p string) { w.withTarget(func(t *MainWindow) { t.hideOpenedAs(); t.importPDFAsync(p) }) })
}

// ImportPDFAsMarkdown is File > Import PDF as Markdown….
func (w *MainWindow) ImportPDFAsMarkdown() {
	w.choosePDF(kindNovel, func(p string) { w.withTarget(func(t *MainWindow) { t.hideOpenedAs(); t.importProsePDFAsync(p) }) })
}

// choosePDF lets the person pick a PDF, the PDFs of kind shown first.
func (w *MainWindow) choosePDF(kind string, then func(path string)) {
	if inBrowser {
		pickFile([]string{".pdf"}, then)
		return
	}
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil || r == nil {
			return
		}
		r.Close()
		then(r.URI().Path())
	}, w.fyneWindow)
	pdf := storage.NewExtensionFileFilter([]string{".pdf"})
	d.SetFilters([]dialog.NamedFilter{
		{Name: "Screenplay (PDF)", Filter: pdf, Kind: kindScreenplay},
		{Name: "Novel (PDF)", Filter: pdf, Kind: kindNovel},
	})
	d.SetFileKinds([]string{kindScreenplay, kindNovel}, func(u fyne.URI) string { return fileKindOf(u.Path()) })
	d.SelectFileKind(kind)
	d.Show()
}

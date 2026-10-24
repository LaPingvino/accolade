package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
)

// pdfWarning says what the PDF writer cannot print of a script ("" if
// it prints well).
func pdfWarning(content string, scenes []string) string {
	return cli.PDFWarning(fountain.Parse(scenes, strings.NewReader(content)))
}

// pdfOrPrint makes a PDF with makePDF, unless the script has text the
// PDF cannot print as written (Hebrew, CJK: boxes; right to left: the
// wrong way round): then it offers to print instead, through the browser,
// whose fonts and layout have them, or to make the PDF anyway. With a
// Chromium browser on the computer there is no need to ask: exportAs has
// it make the PDF.
func (w *MainWindow) pdfOrPrint(content string, makePDF func()) {
	warning := pdfWarning(content, sceneHeaders(w.language))
	if warning == "" || localBrowser() != "" { // a browser makes it (exportAs)
		makePDF()
		return
	}
	warning = strings.Replace(warning, "Export HTML and print it from a browser (Save as PDF) instead.",
		"Print it instead (File > Print…), and choose Save as PDF in the print dialog.", 1)
	label := widget.NewLabel(warning)
	label.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm("The PDF cannot print this script", "Print Instead", "Make the PDF Anyway",
		label, func(print bool) {
			if print {
				w.printScript()
			} else {
				makePDF()
			}
		}, w.fyneWindow)
	d.Resize(fyne.NewSize(520, 240))
	d.Show()
}

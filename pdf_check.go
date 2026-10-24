package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
)

// pdfOrPrint makes a PDF with makePDF, unless the script has text the
// PDF cannot print as written (Hebrew, CJK: boxes; right to left: the
// wrong way round): then it offers to print instead, through the browser,
// whose fonts and layout have them, or to make the PDF anyway.
func (w *MainWindow) pdfOrPrint(content string, makePDF func()) {
	warning := cli.PDFWarning(fountain.Parse(sceneHeaders(w.language), strings.NewReader(content)))
	if warning == "" {
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

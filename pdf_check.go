package main

import (
	"github.com/LaPingvino/lexington/rules"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
)

// pdfWarning says what the PDF writer cannot print of a script ("" if
// it prints well).
func pdfWarning(content string, lang rules.Language) string {
	return cli.PDFWarning(fountain.ParseIn(lang, strings.NewReader(content)))
}

// unshapedScripts are the scripts of a script whose joining letters the
// PDF writer cannot shape (Lexington's names: "Syriac", "Nko"), none if
// it shapes all.
func unshapedScripts(content string, lang rules.Language) []string {
	return pdf.Unshaped(fountain.ParseIn(lang, strings.NewReader(content)))
}

// pdfOrPrint makes a PDF with makePDF, unless the script has text the
// PDF cannot print as written (letters that join that it cannot shape,
// as Syriac; a character no font on the computer has): then it offers to
// print instead, through the browser, whose fonts and shaping have them,
// or to make the PDF anyway.
func (w *MainWindow) pdfOrPrint(content string, makePDF func()) {
	warning := pdfWarning(content, w.lex().Language(w.language))
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

package main

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
)

// Page Setup (LOOP.md, Joop 2026-11-21: "split out the export settings
// from Export, Print and Preview, as the latter two use these settings,
// but you can only set them while also exporting"): what the printed
// page looks like, in one dialog of its own (File > Page Setup…), and
// read from one place by everything that prints: Export, Print, Preview
// PDF, and the page counts (reports, the schedule, locked pages). Export
// keeps what is its own (the format, the folder, the file name, the part
// of the script); it, Print and Preview each show a line saying the page
// setup, with Page Setup… beside it. The settings keys are the ones the
// Export dialog had, so nothing moves.

// The settings keys Page Setup adds: the export had these only while it
// was open.
const (
	revisedOnlyPref = "export-revised-only" // revised pages only (PDF)
	templatePref    = "export-template"     // FDX's and LaTeX's template, "" Lexington's
)

// pageSetup is what the printed page looks like, as the settings have
// it (the script format is the script's own: scriptFormatKey).
type pageSetup struct {
	paper         string // "page-size": "" the script format's, else letter, a4, a5
	titlePage     bool
	numberScenes  bool // the scenes that have no number
	continueds    bool // (CONTINUED) where a scene breaks over pages
	colors        bool // headings and sections in their notes' colors
	colorStyle    lex.ColorStyle
	colorBlock    bool // a marker, background or text color over the whole block
	colorTint     bool // a line's block tinted
	colorPatterns bool // a line's bar patterned
	pagination    string
	revisions     bool // the revision marks, once issued
	revisedOnly   bool
	pageColours   bool // the changed pages on their colour's paper
	template      string
}

// pageSetupOf is the page setup the settings s have.
func pageSetupOf(s *Settings) pageSetup {
	return pageSetup{
		paper:         s.GetString("page-size"),
		titlePage:     s.GetBoolean("include-title-page"),
		numberScenes:  s.GetBoolean("fountain-scene-numbers"),
		continueds:    s.GetBoolean(continuedsPref),
		colors:        s.GetBoolean("export-colors"),
		colorStyle:    lex.ColorStyle(s.GetString("export-color-style")),
		colorBlock:    s.GetBoolean("export-color-block"),
		colorTint:     s.GetBoolean("export-color-tint"),
		colorPatterns: s.GetBoolean("export-color-patterns"),
		pagination:    s.GetString("export-pagination"),
		revisions:     s.GetBoolean("export-revisions"),
		revisedOnly:   s.GetBoolean(revisedOnlyPref),
		pageColours:   s.GetBoolean("export-page-colours"),
		template:      s.GetString(templatePref),
	}
}

// writeTo puts the page setup in the settings s.
func (p pageSetup) writeTo(s *Settings) {
	s.SetString("page-size", p.paper)
	s.SetBoolean("include-title-page", p.titlePage)
	s.SetBoolean("fountain-scene-numbers", p.numberScenes)
	s.SetBoolean(continuedsPref, p.continueds)
	s.SetBoolean("export-colors", p.colors)
	s.SetString("export-color-style", string(p.colorStyle))
	s.SetBoolean("export-color-block", p.colorBlock)
	s.SetBoolean("export-color-tint", p.colorTint)
	s.SetBoolean("export-color-patterns", p.colorPatterns)
	s.SetString("export-pagination", p.pagination)
	s.SetBoolean("export-revisions", p.revisions)
	s.SetBoolean(revisedOnlyPref, p.revisedOnly)
	s.SetBoolean("export-page-colours", p.pageColours)
	s.SetString(templatePref, strings.TrimSpace(p.template))
}

// pageSetupPref is the key changeSetting is given for a page setup's
// change (every key above at once): Preferences shows its line again.
const pageSetupPref = "page-setup"

// pageSetup is the window's page setup.
func (w *MainWindow) pageSetup() pageSetup { return pageSetupOf(w.settings) }

// setPageSetup saves p and shows it everywhere: the ribbon's Scene
// Numbers and CONTINUEDs, Preferences, the other windows.
func (w *MainWindow) setPageSetup(p pageSetup) {
	w.changeSetting(pageSetupPref, p.writeTo)
}

// layoutJob is an export laid out as the page setup says, without what
// only an output has (revisions, a template, the part of the script):
// what the page counts and locks go by (reports, the schedule, locked
// pages), and what Print and Preview start from.
func (w *MainWindow) layoutJob() exportJob { return w.layoutJobOf(w.pageSetup()) }

func (w *MainWindow) layoutJobOf(p pageSetup) exportJob {
	return exportJob{
		format:        w.scriptFormatKey(),
		conf:          w.lex(),
		page:          p.paper,
		omitTitlePage: !p.titlePage,
		numberScenes:  p.numberScenes,
		colors:        p.colors,
		colorStyle:    p.colorStyle,
		colorBlock:    p.colorBlock,
		colorTint:     p.colorTint,
		colorPatterns: p.colorPatterns,
		pagination:    p.pagination,
		scenes:        w.sceneHeaders(),
		language:      w.language,
		continueds:    p.continueds,
	}
}

// outputJob is the export of text as format (an export format's id;
// "html" for Print) with the page setup p: Export, Print and Preview
// PDF all make theirs here. part sets the part of the script (Export's
// filters) before the pages are matched to an issue's; nil for all of it.
func (w *MainWindow) outputJob(p pageSetup, format, text string, part func(*exportJob)) exportJob {
	j := w.layoutJobOf(p)
	j.page = w.paperOf(p.paper)
	a := pageSetupApplies(format, w.scriptIssued())
	if a.template {
		j.template = strings.TrimSpace(p.template)
	}
	if part != nil {
		part(&j)
	}
	switch {
	case format == "pdf" && a.revisions && p.revisions:
		j = w.withRevisions(j, text, p.revisedOnly, p.pageColours)
	case format == "fdx" && a.revisions && p.revisions:
		j = w.withAllRevisions(j, text)
	}
	if format == "fdx" { // the scenes' lengths, and the breakdown's tags
		j = w.withBreakdown(j, text)
	}
	if format == "pdf" { // in the pages its revision prints against
		j = w.withIssuedPages(j, text)
	}
	return j
}

// paperOf is the paper's name for Lexington: the setting's, or the
// script format's for "".
func (w *MainWindow) paperOf(paper string) string {
	if paper == "" {
		return w.lex().format(w.scriptFormatKey()).Page
	}
	return strings.ToLower(paper)
}

// pageApplies is what of the page setup a format uses.
type pageApplies struct {
	scriptFormat bool // the script format's layout
	paper        bool
	template     bool
	script       bool // title page, scene numbers, colours, page breaks: what Lexington lays out
	continueds   bool
	revisions    bool // the marks (PDF, FDX), once issued
	revisedPages bool // revised pages only, the pages' colours (PDF)
}

// none reports whether nothing of the page setup applies.
func (a pageApplies) none() bool { return a == pageApplies{} }

// pageSetupApplies is what format (an export format's id or label,
// "print" for Print) uses of the page setup; issued: the script has an
// issue, so its revisions print.
func pageSetupApplies(format string, issued bool) pageApplies {
	if format == printFormat {
		format = "html"
	}
	f := exportFormatByLabel(format)
	var a pageApplies
	if !f.lexington {
		return a // the text as it is
	}
	a.script = true
	switch f.id {
	case "pdf", "html", "docx", "odt", "latex", "htmlpdf", "latexpdf":
		a.scriptFormat = true
	}
	switch f.id {
	case "pdf", "docx", "odt":
		a.paper = true
	}
	switch f.id {
	case "fdx", "latex", "latexpdf":
		a.template = true
	}
	a.continueds = f.id == "pdf"
	a.revisions = issued && (f.id == "pdf" || f.id == "fdx")
	a.revisedPages = issued && f.id == "pdf"
	return a
}

// printFormat is Print's place among the formats Page Setup shows the
// settings of: the HTML the browser prints.
const printFormat = "print"

// pageFormats are the formats Page Setup can show the settings of: Print
// first, then the export formats (ids).
func pageFormats() []string {
	ids := []string{printFormat}
	for _, f := range exportFormatList() {
		ids = append(ids, f.id)
	}
	return ids
}

// pageFormatLabel is a format's name in Page Setup's list.
func pageFormatLabel(id string) string {
	if id == printFormat {
		return "Print (File > Print…)"
	}
	return exportFormatByLabel(id).label
}

// pageFormatID is the format a label of Page Setup's list names.
func pageFormatID(label string) string {
	if label == pageFormatLabel(printFormat) {
		return printFormat
	}
	return exportFormatByLabel(label).id
}

// pageSummary is the page setup p in a line, as format uses it: Export,
// Print and Preview show it beside Page Setup….
func (w *MainWindow) pageSummary(p pageSetup, format string) string {
	a := pageSetupApplies(format, w.scriptIssued())
	if a.none() {
		return "The script's text as it is: no page setup."
	}
	var parts []string
	if a.scriptFormat {
		parts = append(parts, w.lex().format(w.scriptFormatKey()).Name)
	}
	if a.paper {
		if p.paper == "" {
			parts = append(parts, paperName(w.paperOf(""))+" (the format's)")
		} else {
			parts = append(parts, paperName(p.paper))
		}
	}
	if a.script {
		if p.titlePage {
			parts = append(parts, "title page")
		} else {
			parts = append(parts, "no title page")
		}
		if p.numberScenes {
			parts = append(parts, "scenes numbered")
		}
		if p.colors {
			parts = append(parts, "colours: "+strings.ToLower(colorStyleLabel(string(p.colorStyle))))
		}
		if p.pagination != "" {
			parts = append(parts, "page breaks: "+strings.ToLower(strings.TrimSuffix(paginationLabel(p.pagination), ":")))
		}
	}
	if a.continueds && p.continueds {
		parts = append(parts, "CONTINUEDs")
	}
	if a.revisions && p.revisions {
		r := "revision marks"
		if a.revisedPages && p.revisedOnly {
			r += ", revised pages only"
		}
		if a.revisedPages && p.pageColours {
			r += ", on their colour's paper"
		}
		parts = append(parts, r)
	}
	if a.template && strings.TrimSpace(p.template) != "" {
		parts = append(parts, "template "+filepath.Base(strings.TrimSpace(p.template)))
	}
	if w.textEditor != nil && pageLockOf(w.textEditor.Text()) != nil && (format == "pdf" || format == "fdx") {
		parts = append(parts, "pages locked")
	}
	s := strings.Join(parts, " · ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

// paperName is a paper's name as people write it.
func paperName(p string) string {
	switch strings.ToLower(p) {
	case "a4":
		return "A4"
	case "a5":
		return "A5"
	case "letter", "":
		return "Letter"
	}
	return p
}

// pageSetupLine is the line Export, Print, Preview and Preferences show:
// the page setup as format (given by the caller, as it is then) uses it,
// and Page Setup… opening it for that format; changed runs after a change
// in it. refresh shows the line again.
type pageSetupLine struct {
	w       *MainWindow
	box     fyne.CanvasObject
	summary *widget.Label
	button  *widget.Button
	format  func() string
	changed func()
	setup   func() pageSetup // the page setup shown, if not the window's (Export's revised pages)
}

func (w *MainWindow) newPageSetupLine(format func() string, changed func()) *pageSetupLine {
	l := &pageSetupLine{w: w, format: format, changed: changed}
	l.summary = widget.NewLabel("")
	l.summary.Wrapping = fyne.TextWrapWord
	l.button = widget.NewButtonWithIcon("Page Setup…", theme.DocumentIcon(), func() {
		w.showPageSetup(l.format(), func() {
			l.refresh()
			if l.changed != nil {
				l.changed()
			}
		})
	})
	l.box = container.NewBorder(nil, nil, nil, container.NewVBox(l.button), l.summary)
	if c := w.fyneWindow.Canvas().Size(); c.Width > 0 && c.Width < 600 { // a phone: the button under the line
		l.box = container.NewVBox(l.summary, container.NewHBox(l.button))
	}
	l.refresh()
	return l
}

func (l *pageSetupLine) refresh() {
	if l.w.markdown {
		l.summary.SetText(l.w.markdownPageSummary())
		return
	}
	p := l.w.pageSetup()
	if l.setup != nil {
		p = l.setup()
	}
	l.summary.SetText(l.w.pageSummary(p, l.format()))
}

// The dialog.

// pageSetupDlg is File > Page Setup…: the page setup, the settings shown
// those the format chosen at its top uses (the caller's: Export's
// format, Print, Preview's PDF).
type pageSetupDlg struct {
	w    *MainWindow
	d    *dialog.CustomDialog
	done func() // after OK

	formatSelect *widget.Select // whose settings show
	box          *fyne.Container
	scroll       *container.Scroll
	note         *widget.Label // what the format does with none, or a lock

	scriptFormatSelect *widget.Select
	paperSelect        *widget.Select
	templateEntry      *widget.Entry

	includeTitlePageCheck *widget.Check
	sceneNumbersCheck     *widget.Check
	continuedsCheck       *widget.Check
	colorsCheck           *widget.Check
	colorStyleSelect      *widget.Select
	colorOptions          *fyne.Container
	colorBlockCheck       *widget.Check // with marker, background or text
	colorTintCheck        *widget.Check // with the line
	colorPatternsCheck    *widget.Check // with the line

	paginationSelect                                 *widget.Select
	paginationKeep, paginationWidows, paginationMore *widget.Check
	paginationBox                                    fyne.CanvasObject

	revisionsCheck, revisedOnlyCheck, pageColoursCheck *widget.Check
}

// showPageSetup opens Page Setup with format's settings (an export
// format's id, printFormat; "" the export's); done runs after OK.
func (w *MainWindow) showPageSetup(format string, done func()) *pageSetupDlg {
	if w.markdown {
		w.showMarkdownPageSetup(done)
		return nil
	}
	p := w.newPageSetup(format, done)
	p.d.Show()
	return p
}

// newPageSetup is the dialog, not shown.
func (w *MainWindow) newPageSetup(format string, done func()) *pageSetupDlg {
	if format == "" {
		format = exportFormatByLabel(w.settings.GetString("export-format")).id
	}
	p := &pageSetupDlg{w: w, done: done}
	var labels []string
	for _, id := range pageFormats() {
		labels = append(labels, pageFormatLabel(id))
	}
	p.formatSelect = widget.NewSelect(labels, func(string) { p.show() })
	p.note = widget.NewLabel("")
	p.note.Wrapping = fyne.TextWrapWord
	p.note.Importance = widget.LowImportance

	p.scriptFormatSelect = newScriptFormatSelect(w.lex, nil, func(string) { p.enable() })
	fyne.SetAccessibleLabel(p.scriptFormatSelect, "Script format")
	p.paperSelect = widget.NewSelect(documentPages, nil)
	p.templateEntry = widget.NewEntry()
	p.templateEntry.SetPlaceHolder("Lexington's own")
	fyne.SetAccessibleLabel(p.templateEntry, "Template")

	p.includeTitlePageCheck = widget.NewCheck("Title page", nil)
	p.sceneNumbersCheck = widget.NewCheck("Number the scenes not numbered", nil)
	p.continuedsCheck = widget.NewCheck("CONTINUED where a scene breaks", nil)
	p.colorsCheck = widget.NewCheck("Colours: headings in their notes' colours", func(bool) { p.enable() })
	var styles []string
	for _, s := range colorStyles {
		styles = append(styles, s.label)
	}
	p.colorStyleSelect = widget.NewSelect(styles, func(string) { p.enable() })
	p.colorBlockCheck = widget.NewCheck("Whole block, not only the heading", nil)
	p.colorTintCheck = widget.NewCheck("Tint the block", nil)
	p.colorPatternsCheck = widget.NewCheck("Patterns on the line (colour blindness)", nil)
	p.colorOptions = container.NewVBox(labelled("Style:", p.colorStyleSelect),
		p.colorBlockCheck, p.colorTintCheck, p.colorPatternsCheck)
	p.paginationBox = p.paginationWidgets()
	p.revisionsCheck = widget.NewCheck("Revision marks (* and the page's header)", func(bool) { p.enable() })
	p.revisedOnlyCheck = widget.NewCheck("Only the pages with a change", nil)
	p.pageColoursCheck = widget.NewCheck("Changed pages on their colour's paper", nil)

	p.load(w.pageSetup())
	w.lex().selectFormat(p.scriptFormatSelect, w.scriptFormatKey())
	p.box = container.NewVBox()
	p.formatSelect.SetSelected(pageFormatLabel(format))

	intro := widget.NewLabel("How the script prints: Export, Print and Preview PDF all use these.")
	intro.Wrapping = fyne.TextWrapWord
	top := container.NewVBox(intro, labelled("Settings for:", p.formatSelect), widget.NewSeparator())
	cancel := widget.NewButton("Cancel", func() { p.d.Hide() })
	defaults := widget.NewButton("Defaults", p.defaults)
	ok := widget.NewButton("OK", p.ok)
	ok.Importance = widget.HighImportance
	p.scroll = container.NewVScroll(container.NewVBox(top, p.box, p.note))
	p.d = dialog.NewCustomWithoutButtons("Page Setup", container.NewBorder(nil, w.dialogButtons(cancel, defaults, ok), nil, nil, p.scroll), w.fyneWindow)
	p.d.Resize(fitDialog(w.fyneWindow, 600, 640))
	p.d.SetOnEscape(p.d.Hide)
	return p
}

// format is the format whose settings show.
func (p *pageSetupDlg) format() string { return pageFormatID(p.formatSelect.Selected) }

// show shows the settings the format uses, and only those (as the Export
// dialog did).
func (p *pageSetupDlg) show() {
	if p.box == nil {
		return
	}
	a := pageSetupApplies(p.format(), p.w.scriptIssued())
	var objs []fyne.CanvasObject
	if a.scriptFormat {
		objs = append(objs, labelled("Script format:", p.scriptFormatSelect))
	}
	if a.paper {
		objs = append(objs, labelled("Paper:", p.paperSelect))
	}
	if a.script {
		objs = append(objs, p.includeTitlePageCheck, p.sceneNumbersCheck)
	}
	if a.continueds && p.w.role() == roleProduction {
		objs = append(objs, p.continuedsCheck)
	}
	if a.script {
		objs = append(objs, p.colorsCheck, p.colorOptions, p.paginationBox)
	}
	if a.revisions {
		objs = append(objs, p.revisionsCheck)
	}
	if a.revisedPages {
		objs = append(objs, p.revisedOnlyCheck, p.pageColoursCheck)
	}
	if a.template {
		objs = append(objs, labelled("Template:", p.templateEntry))
	}
	p.box.Objects = objs
	note := ""
	switch {
	case a.none():
		note = "The script's text as it is: no page setup applies to it."
	case (p.format() == "pdf" || p.format() == "fdx") && pageLockOf(p.w.textEditor.Text()) != nil:
		note = "The pages are locked (Production > Lock Pages): the whole script prints in the lock's format, " +
			"page breaks and CONTINUEDs; the rest is as set here."
	}
	p.note.SetText(note)
	setVisible(p.note, note != "")
	p.enable()
	p.box.Refresh()
}

// enable lets what goes with another setting be chosen with it.
func (p *pageSetupDlg) enable() {
	if p.revisedOnlyCheck == nil || p.paginationMore == nil {
		return
	}
	on := func(o fyne.Disableable, yes bool) {
		if yes {
			o.Enable()
		} else {
			o.Disable()
		}
	}
	colors := p.colorsCheck.Checked
	style := p.chosenColorStyle()
	on(p.colorStyleSelect, colors)
	on(p.colorBlockCheck, colors && (style == lex.ColorMarker || style == lex.ColorBackground || style == lex.ColorText))
	on(p.colorTintCheck, colors && style == lex.ColorLine)
	on(p.colorPatternsCheck, colors && style == lex.ColorLine)
	some := p.paginationSelect.Selected == paginationChoices[3].label
	for _, o := range []fyne.Disableable{p.paginationKeep, p.paginationWidows, p.paginationMore} {
		on(o, some)
	}
	on(p.revisedOnlyCheck, p.revisionsCheck.Checked)
	on(p.pageColoursCheck, p.revisionsCheck.Checked)
}

// load shows the page setup s.
func (p *pageSetupDlg) load(s pageSetup) {
	p.paperSelect.SetSelected(documentPageLabel(s.paper))
	p.includeTitlePageCheck.SetChecked(s.titlePage)
	p.sceneNumbersCheck.SetChecked(s.numberScenes)
	p.continuedsCheck.SetChecked(s.continueds)
	p.colorsCheck.SetChecked(s.colors)
	p.colorStyleSelect.SetSelected(colorStyleLabel(string(s.colorStyle)))
	p.colorBlockCheck.SetChecked(s.colorBlock)
	p.colorTintCheck.SetChecked(s.colorTint)
	p.colorPatternsCheck.SetChecked(s.colorPatterns)
	p.setPagination(s.pagination)
	p.revisionsCheck.SetChecked(s.revisions)
	p.revisedOnlyCheck.SetChecked(s.revisedOnly)
	p.pageColoursCheck.SetChecked(s.pageColours)
	p.templateEntry.SetText(s.template)
	p.enable()
}

// chosen is the page setup as the dialog shows it.
func (p *pageSetupDlg) chosen() pageSetup {
	return pageSetup{
		paper:         chosenPaper(p.paperSelect.Selected),
		titlePage:     p.includeTitlePageCheck.Checked,
		numberScenes:  p.sceneNumbersCheck.Checked,
		continueds:    p.continuedsCheck.Checked,
		colors:        p.colorsCheck.Checked,
		colorStyle:    p.chosenColorStyle(),
		colorBlock:    p.colorBlockCheck.Checked,
		colorTint:     p.colorTintCheck.Checked,
		colorPatterns: p.colorPatternsCheck.Checked,
		pagination:    p.chosenPagination(),
		revisions:     p.revisionsCheck.Checked,
		revisedOnly:   p.revisedOnlyCheck.Checked,
		pageColours:   p.pageColoursCheck.Checked,
		template:      strings.TrimSpace(p.templateEntry.Text),
	}
}

// defaults shows the page setup as it starts (saved with OK).
func (p *pageSetupDlg) defaults() {
	s := &Settings{data: map[string]interface{}{}}
	s.loadDefaults()
	p.load(pageSetupOf(s))
}

// ok saves the page setup and the script format, if another.
func (p *pageSetupDlg) ok() {
	w := p.w
	if key := w.lex().selectedFormat(p.scriptFormatSelect); key != w.scriptFormatKey() {
		w.setScriptFormat(key)
	}
	w.setPageSetup(p.chosen())
	p.d.Hide()
	if p.done != nil {
		p.done()
	}
}

// documentPages are the paper choices for PDF, DOCX and ODT; the first
// is the script format's suggestion (A5 for a musical), else Letter.
var documentPages = []string{"Script format's", "Letter", "A4", "A5"}

// documentPageLabel is the choice for a "page-size" setting.
func documentPageLabel(name string) string {
	for _, l := range documentPages[1:] {
		if strings.EqualFold(l, name) {
			return l
		}
	}
	return documentPages[0]
}

// chosenPaper is the "page-size" setting for a choice ("" for the script
// format's).
func chosenPaper(choice string) string {
	if choice == documentPages[0] || choice == "" {
		return ""
	}
	return strings.ToLower(choice)
}

// colorStyles are how colors can show (Lexington's color styles).
var colorStyles = []struct {
	label string
	style lex.ColorStyle
}{
	{"Line in the margin", lex.ColorLine},
	{"Marker", lex.ColorMarker},
	{"Background", lex.ColorBackground},
	{"Text color", lex.ColorText},
}

// colorStyleLabel is the choice for an "export-color-style" setting (a
// line in the margin for one it does not know).
func colorStyleLabel(name string) string {
	for _, s := range colorStyles {
		if string(s.style) == name {
			return s.label
		}
	}
	return colorStyles[0].label
}

// chosenColorStyle is the color style chosen.
func (p *pageSetupDlg) chosenColorStyle() lex.ColorStyle {
	for _, s := range colorStyles {
		if s.label == p.colorStyleSelect.Selected {
			return s.style
		}
	}
	return lex.ColorLine
}

// The page setup's own commands.

// pageSetupCommand is File > Page Setup….
func pageSetupCommand() *command {
	return &command{id: "file.pageSetup", label: "Page Setup…", tip: "How the script prints: paper, title page, numbers, colours…",
		mdTip: "How the document prints: paper, title page", icon: theme.DocumentIcon,
		run: func(w *MainWindow) { w.showPageSetup("", nil) }}
}

// Print and Preview PDF: each says the page setup it prints with, Page
// Setup… beside it.

// showPrint is File > Print…: the page setup's line, then the browser's
// print dialog (printScript, printMarkdown).
func (w *MainWindow) showPrint() *dialog.CustomDialog {
	run := w.printScript
	if w.markdown {
		run = w.printMarkdown
	}
	line := w.newPageSetupLine(func() string { return printFormat }, nil)
	about := widget.NewLabel("The browser's print dialog opens: choose a printer there, or Save as PDF.")
	about.Wrapping = fyne.TextWrapWord
	var d *dialog.CustomDialog
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	print := widget.NewButtonWithIcon("Print", theme.DocumentPrintIcon(), func() {
		d.Hide()
		run()
	})
	print.Importance = widget.HighImportance
	d = dialog.NewCustomWithoutButtons("Print", container.NewBorder(nil, w.dialogButtons(cancel, print), nil, nil,
		container.NewVBox(about, line.box)), w.fyneWindow)
	d.Resize(fitDialog(w.fyneWindow, 540, 240))
	d.SetOnEscape(d.Hide)
	d.Show()
	w.fyneWindow.Canvas().Focus(print) // Space prints
	return d
}

// previewSetup is Preview PDF's dialog while it is open: a change in Page
// Setup makes the preview again.
type previewSetup struct {
	d    *dialog.CustomDialog
	line *pageSetupLine
}

// showPreviewSetup shows, beside the PDF just opened, the page setup it
// is made with; Page Setup… makes it again with the change.
func (w *MainWindow) showPreviewSetup() {
	if w.previewSetup != nil {
		w.previewSetup.line.refresh()
		w.previewSetup.d.Show()
		return
	}
	ps := &previewSetup{}
	ps.line = w.newPageSetupLine(func() string { return "pdf" }, func() {
		w.pdfOrPrint(w.textEditor.Text(), w.makePreviewPDF)
	})
	about := widget.NewLabel("The PDF opens in your PDF viewer. A change in Page Setup makes it again.")
	about.Wrapping = fyne.TextWrapWord
	closeB := widget.NewButton("Close", func() { ps.d.Hide() })
	again := widget.NewButtonWithIcon("Again", theme.ViewRefreshIcon(), func() {
		w.pdfOrPrint(w.textEditor.Text(), w.makePreviewPDF)
	})
	ps.d = dialog.NewCustomWithoutButtons("Preview PDF", container.NewBorder(nil, w.dialogButtons(closeB, again), nil, nil,
		container.NewVBox(about, ps.line.box)), w.fyneWindow)
	ps.d.Resize(fitDialog(w.fyneWindow, 540, 240))
	ps.d.SetOnEscape(ps.d.Hide)
	w.previewSetup = ps
	ps.d.Show()
}

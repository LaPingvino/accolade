package main

import (
	"fmt"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/lex"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ExportDialog struct {
	window   *MainWindow
	dialog   *dialog.CustomDialog
	settings *Settings

	// Format selection
	formatSelect *widget.Select
	// templateEntry is a template for FDX and LaTeX (Lexington's -template)
	templateEntry *widget.Entry

	// Output settings
	outputDirEntry *widget.Entry
	filenameEntry  *widget.Entry

	// Fountain settings
	includeTitlePageCheck *widget.Check
	sceneNumbersCheck     *widget.Check
	scriptFormatSelect    *widget.Select
	documentPageSelect    *widget.Select
	colorsCheck           *widget.Check
	colorStyleSelect      *widget.Select // how the colors show, with Colors on
	// colorOptions holds how colors show, under Colors: the style, and
	// room for what goes with a style (a marker's scope, a line's tint)
	colorOptions                    *fyne.Container
	colorBlockCheck, colorTintCheck *widget.Check // with marker, background or text; with the line
	colorPatternsCheck              *widget.Check // with the line

	// page breaks (export_pagination.go)
	paginationSelect                                 *widget.Select
	paginationKeep, paginationWidows, paginationMore *widget.Check
	paginationBox                                    fyne.CanvasObject

	// Filters (export_filters.go)
	tagChecks        []*widget.Check // one per tag, its Text the tag
	characterSelect  *widget.Select  // whose sides; noSides for none
	sidesRadio       *widget.RadioGroup
	cueRadio         *widget.RadioGroup // the cue: the whole speech or its last words
	cueWordsEntry    *widget.Entry
	sidesBox, cueBox *fyne.Container // shown with a character, with cue lines

	// Progress
	progressBar *widget.ProgressBar
	statusLabel *widget.Label

	// Buttons
	exportButton  *widget.Button
	cancelButton  *widget.Button
	previewButton *widget.Button

	// Format settings container
	formatSettingsContainer *fyne.Container
	scroll                  *container.Scroll // the settings

	// State
	isExporting bool
}

func NewExportDialog(window *MainWindow) *ExportDialog {
	ed := &ExportDialog{
		window:      window,
		settings:    GetSettings(),
		isExporting: false,
	}

	ed.createWidgets()
	ed.loadSettings()
	content := ed.createContent()

	// its own buttons (NewCustom would add an empty dismiss button)
	ed.dialog = dialog.NewCustomWithoutButtons(
		"Export Document",
		content,
		window.fyneWindow,
	)

	ed.dialog.Resize(fyne.NewSize(600, 700))
	ed.dialog.SetOnEscape(ed.cancel) // as its Cancel button

	return ed
}

func (ed *ExportDialog) createWidgets() {
	// Format selection
	ed.formatSelect = widget.NewSelect(exportFormats, ed.onFormatChanged)
	// Don't set selected yet - wait until after containers are created

	// Output settings
	ed.outputDirEntry = widget.NewEntry()
	ed.outputDirEntry.SetText(ed.getDefaultOutputDir())

	ed.filenameEntry = widget.NewEntry()
	ed.filenameEntry.SetText(ed.getDefaultFilename())

	// PDF, HTML, DOCX and ODT: the script format (Lexington's presets)
	ed.scriptFormatSelect = newScriptFormatSelect(nil, nil)
	fyne.SetAccessibleLabel(ed.scriptFormatSelect, "Script format")
	// DOCX and ODT: the paper, or the one the script format suggests
	ed.documentPageSelect = widget.NewSelect(documentPages, nil)
	ed.documentPageSelect.SetSelected(documentPages[0])

	// FDX and LaTeX take a template (Lexington's -template)
	ed.templateEntry = widget.NewEntry()
	ed.templateEntry.SetPlaceHolder("Lexington's own")
	fyne.SetAccessibleLabel(ed.templateEntry, "Template")

	// Fountain settings
	ed.includeTitlePageCheck = widget.NewCheck("Include title page", nil)
	ed.includeTitlePageCheck.SetChecked(true)

	ed.sceneNumbersCheck = widget.NewCheck("Number the scenes (those without a number)", nil)
	ed.colorsCheck = widget.NewCheck("Colors: headings and sections in their notes' colors", func(bool) { ed.enableColorStyle() })
	var styles []string
	for _, s := range colorStyles {
		styles = append(styles, s.label)
	}
	ed.colorStyleSelect = widget.NewSelect(styles, func(string) { ed.enableColorStyle() })
	ed.colorStyleSelect.SetSelected(styles[0])
	ed.colorBlockCheck = widget.NewCheck("Whole block, not only the heading", nil)
	ed.colorTintCheck = widget.NewCheck("Tint the block", nil)
	ed.colorPatternsCheck = widget.NewCheck("Patterns on the line (for colour blindness)", nil)
	ed.colorOptions = container.NewVBox(labelled("Style:", ed.colorStyleSelect),
		container.NewHBox(ed.colorBlockCheck, ed.colorTintCheck), ed.colorPatternsCheck)

	ed.paginationBox = ed.paginationWidgets()
	ed.createFilterWidgets()

	// Progress
	ed.progressBar = widget.NewProgressBar()
	ed.progressBar.Hide()

	ed.statusLabel = widget.NewLabel("Ready to export")

	// Buttons
	ed.exportButton = widget.NewButtonWithIcon("Export", theme.DocumentSaveIcon(), ed.exportDocument)
	ed.cancelButton = widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), ed.cancel)
	ed.previewButton = widget.NewButtonWithIcon("Preview", theme.VisibilityIcon(), ed.previewExport)
}

func (ed *ExportDialog) createContent() fyne.CanvasObject {
	// Output section
	browseButton := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), ed.browseOutputDir)
	fyne.SetAccessibleLabel(browseButton, "Choose the folder")

	outputSection := widget.NewCard("Output", "",
		container.NewVBox(
			labelled("Format:", ed.formatSelect),
			labelledWith("Directory:", ed.outputDirEntry, browseButton),
			labelled("Filename:", ed.filenameEntry),
		),
	)

	// Format-specific settings
	ed.formatSettingsContainer = container.NewVBox()

	// Now it's safe to set the selected format which will call updateFormatSettings
	// the format of the settings (loadSettings), PDF without one; set
	// again now that the format settings have their container
	format := ed.formatSelect.Selected
	if format == "" {
		format = exportFormats[0]
	}
	ed.formatSelect.SetSelected("")
	ed.formatSelect.SetSelected(format)

	formatSection := widget.NewCard("Format Settings", "", ed.formatSettingsContainer)

	// Fountain options
	fountainSection := widget.NewCard("Script", "",
		container.NewVBox(
			ed.includeTitlePageCheck,
			ed.sceneNumbersCheck,
			ed.colorsCheck,
			ed.colorOptions,
			ed.paginationBox,
		),
	)
	filterSection := widget.NewCard("Part of the Script", "", ed.filterContent())

	// Progress section
	progressSection := container.NewVBox(
		ed.progressBar,
		ed.statusLabel,
	)

	// Buttons
	buttonContainer := container.NewHBox(
		ed.previewButton,
		layout.NewSpacer(),
		ed.cancelButton,
		ed.exportButton,
	)
	// the buttons stay below the settings, which scroll
	scrollContent := container.NewVScroll(container.NewVBox(outputSection, formatSection, fountainSection, filterSection, progressSection))
	scrollContent.SetMinSize(fyne.NewSize(580, 560))
	ed.scroll = scrollContent
	return container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), buttonContainer), nil, nil, scrollContent)
}

func (ed *ExportDialog) updateFormatSettings() {
	// Safety check in case this is called before container is initialized
	if ed.formatSettingsContainer == nil {
		return
	}

	ed.formatSettingsContainer.Objects = nil

	labelled := func(label string, o fyne.CanvasObject) {
		ed.formatSettingsContainer.Add(labelled(label, o))
	}
	note := func(text string) {
		l := widget.NewLabel(text)
		l.Wrapping = fyne.TextWrapWord
		ed.formatSettingsContainer.Add(l)
	}
	f := exportFormatByLabel(ed.formatSelect.Selected)
	switch f.id {
	case "pdf", "html", "docx", "odt", "latex", "htmlpdf", "latexpdf":
		labelled("Script format:", ed.scriptFormatSelect)
	}
	switch f.id {
	case "pdf", "docx", "odt":
		labelled("Paper:", ed.documentPageSelect)
	}
	switch f.id {
	case "fdx", "latex", "latexpdf":
		labelled("Template:", ed.templateEntry)
	}
	switch f.id {
	case "docx", "odt":
		note("Every element is a paragraph style, to restyle in Word or LibreOffice.")
	case "fountain":
		note("Fountain as Lexington writes it: tidied up, every element marked as it is meant.")
	case "text-fountain":
		note("The script's text as it is: a copy of the .fountain file.")
	case "txt":
		note("The script's text as it is, as a .txt file.")
	}
	available := !f.lexington || f.format.Available()
	if !available {
		if inBrowser {
			note(f.format.Name + " is made with " + f.format.Needs + ": only in Accolade on your computer " +
				"(github.com/LaPingvino/accolade), with " + f.format.Needs + " installed.")
		} else {
			note(f.format.Name + " is made with " + f.format.Needs + ", which is not installed.")
		}
	}
	for _, o := range append([]fyne.Disableable{ed.includeTitlePageCheck, ed.sceneNumbersCheck, ed.colorsCheck, ed.previewButton}, ed.filterWidgets()...) {
		if o == nil {
			continue
		}
		if f.lexington && available {
			o.Enable()
		} else {
			o.Disable()
		}
	}
	ed.enablePagination()
	ed.enableColorStyle()
	if ed.exportButton != nil {
		if available {
			ed.exportButton.Enable()
		} else {
			ed.exportButton.Disable()
		}
	}

	ed.formatSettingsContainer.Refresh()
}

func (ed *ExportDialog) onFormatChanged(format string) {
	ed.updateFormatSettings()
	ed.updateDefaultFilename()
}

func (ed *ExportDialog) loadSettings() {
	// Load export settings from preferences
	ed.formatSelect.SetSelected(exportFormatLabel(ed.settings.GetString("export-format")))

	if dir := ed.settings.GetString("export-directory"); dir != "" {
		ed.outputDirEntry.SetText(dir)
	}

	selectScriptFormat(ed.scriptFormatSelect, ed.settings.GetString("script-format"))
	ed.documentPageSelect.SetSelected(documentPageLabel(ed.settings.GetString("page-size")))

	ed.includeTitlePageCheck.SetChecked(ed.settings.GetBoolean("include-title-page"))
	ed.sceneNumbersCheck.SetChecked(ed.settings.GetBoolean("fountain-scene-numbers"))
	ed.colorsCheck.SetChecked(ed.settings.GetBoolean("export-colors"))
	ed.colorStyleSelect.SetSelected(colorStyleLabel(ed.settings.GetString("export-color-style")))
	ed.colorBlockCheck.SetChecked(ed.settings.GetBoolean("export-color-block"))
	ed.colorTintCheck.SetChecked(ed.settings.GetBoolean("export-color-tint"))
	ed.colorPatternsCheck.SetChecked(ed.settings.GetBoolean("export-color-patterns"))
	ed.setPagination(ed.settings.GetString("export-pagination"))
	ed.enableColorStyle()
}

func (ed *ExportDialog) saveSettings() {
	// Save export settings to preferences
	ed.settings.SetString("export-format", ed.formatSelect.Selected)
	ed.settings.SetString("export-directory", ed.outputDirEntry.Text)
	ed.settings.SetString("page-size", ed.chosenPage())
	ed.settings.SetString("script-format", selectedScriptFormat(ed.scriptFormatSelect))
	ed.settings.SetBoolean("include-title-page", ed.includeTitlePageCheck.Checked)
	ed.settings.SetBoolean("fountain-scene-numbers", ed.sceneNumbersCheck.Checked)
	ed.settings.SetBoolean("export-colors", ed.colorsCheck.Checked)
	ed.settings.SetString("export-color-style", string(ed.chosenColorStyle()))
	ed.settings.SetBoolean("export-color-block", ed.colorBlockCheck.Checked)
	ed.settings.SetBoolean("export-color-tint", ed.colorTintCheck.Checked)
	ed.settings.SetBoolean("export-color-patterns", ed.colorPatternsCheck.Checked)
	ed.settings.SetString("export-pagination", ed.chosenPagination())

	ed.settings.Save()
}

func (ed *ExportDialog) getDefaultOutputDir() string {
	if dir := ed.settings.GetString("export-directory"); dir != "" {
		return dir
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	return filepath.Join(homeDir, "Documents")
}

func (ed *ExportDialog) getDefaultFilename() string {
	if ed.window.currentFile != "" {
		base := filepath.Base(ed.window.currentFile)
		ext := filepath.Ext(base)
		if ext != "" {
			base = base[:len(base)-len(ext)]
		}
		return base
	}
	return "screenplay"
}

func (ed *ExportDialog) updateDefaultFilename() {
	base := ed.getDefaultFilename()
	if ed.characterSelect != nil { // (not yet while the widgets are made)
		base += filterSuffix(ed.chosenTags(), ed.chosenCharacter())
	}
	ext := "." + exportFormatByLabel(ed.formatSelect.Selected).ext
	ed.filenameEntry.SetText(base + ext)
}

func (ed *ExportDialog) browseOutputDir() {
	dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
		if err == nil && folder != nil {
			ed.outputDirEntry.SetText(folder.Path())
		}
	}, ed.window.fyneWindow)
}

// previewExport writes the export to a temporary file and opens it in
// the system's viewer for that kind of file.
func (ed *ExportDialog) previewExport() {
	if err := ed.validateFilters(); err != nil {
		dialog.ShowError(err, ed.window.fyneWindow)
		return
	}
	f := exportFormatByLabel(ed.formatSelect.Selected)
	format := f.id
	dir, err := os.MkdirTemp("", "accolade-preview-")
	if err != nil {
		dialog.ShowError(err, ed.window.fyneWindow)
		return
	}
	out := filepath.Join(dir, "preview."+f.ext)
	content := ed.window.textEditor.Text()
	ed.statusLabel.SetText("Preparing the preview...")
	job := ed.job() // read on the UI thread
	go func() {
		err := ed.performExportJob(content, out, format, job)
		fyne.Do(func() {
			if err != nil {
				ed.statusLabel.SetText("Preview failed")
				dialog.ShowError(err, ed.window.fyneWindow)
				return
			}
			ed.statusLabel.SetText("Ready to export")
			openFile(out)
		})
	}()
}

func (ed *ExportDialog) exportDocument() {
	if ed.isExporting {
		return
	}

	// Validate inputs
	if err := ed.validateInputs(); err != nil {
		dialog.ShowError(err, ed.window.fyneWindow)
		return
	}
	out := filepath.Join(ed.outputDirEntry.Text, ed.filenameEntry.Text)
	if ed.window.currentFile != "" && sameFile(out, ed.window.currentFile) {
		dialog.ShowError(fmt.Errorf("%s is the script itself: choose another name or directory", filepath.Base(out)),
			ed.window.fyneWindow)
		return
	}
	if _, err := os.Stat(out); err == nil {
		dialog.ShowConfirm("Replace File", fmt.Sprintf("%s exists. Replace it?", filepath.Base(out)), func(ok bool) {
			if ok {
				ed.startExport()
			}
		}, ed.window.fyneWindow)
		return
	}
	ed.startExport()
}

// sameFile reports whether two paths name one file.
func sameFile(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA == nil && errB == nil {
		return os.SameFile(ia, ib)
	}
	pa, _ := filepath.Abs(a)
	pb, _ := filepath.Abs(b)
	return pa == pb
}

func (ed *ExportDialog) validateInputs() error {
	if ed.outputDirEntry.Text == "" {
		return fmt.Errorf("please select an output directory")
	}

	if ed.filenameEntry.Text == "" {
		return fmt.Errorf("please enter a filename")
	}
	if err := ed.validateFilters(); err != nil {
		return err
	}

	// Check if output directory exists
	if _, err := os.Stat(ed.outputDirEntry.Text); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", ed.outputDirEntry.Text)
	}

	return nil
}

// startExport exports, a PDF only once it is clear it can print the
// script (pdfOrPrint).
func (ed *ExportDialog) startExport() {
	if exportFormatByLabel(ed.formatSelect.Selected).id == "pdf" {
		ed.window.pdfOrPrint(ed.window.textEditor.Text(), ed.runExport)
		return
	}
	ed.runExport()
}

func (ed *ExportDialog) runExport() {
	ed.isExporting = true
	ed.exportButton.Disable()
	ed.progressBar.Show()
	ed.statusLabel.SetText("Exporting...")

	// Save settings
	ed.saveSettings()

	// Get export parameters
	outputPath := filepath.Join(ed.outputDirEntry.Text, ed.filenameEntry.Text)
	format := exportFormatByLabel(ed.formatSelect.Selected).id
	content := ed.window.textEditor.Text()

	// Perform export in goroutine
	job := ed.job() // the options, read here on the UI thread
	go func() {
		err := ed.performExportJob(content, outputPath, format, job)

		// Update UI on main thread
		fyne.Do(func() { ed.finishExport(err, outputPath) })
	}()
}

// job is the export as the dialog sets it up.
func (ed *ExportDialog) job() exportJob {
	return exportJob{
		format:        selectedScriptFormat(ed.scriptFormatSelect),
		page:          ed.documentPage(),
		template:      strings.TrimSpace(ed.templateEntry.Text),
		omitTitlePage: !ed.includeTitlePageCheck.Checked,
		numberScenes:  ed.sceneNumbersCheck.Checked,
		scenes:        sceneHeaders(ed.window.language),
		colors:        ed.colorsCheck.Checked,
		colorStyle:    ed.chosenColorStyle(),
		colorBlock:    ed.colorBlockCheck.Checked,
		colorTint:     ed.colorTintCheck.Checked,
		colorPatterns: ed.colorPatternsCheck.Checked,
		pagination:    ed.chosenPagination(),
		tags:          ed.chosenTags(),
		character:     ed.chosenCharacter(),
		sides:         ed.chosenSides(),
		cueWords:      ed.chosenCueWords(),
	}
}

func (ed *ExportDialog) performExport(content, outputPath, format string) error {
	return ed.performExportJob(content, outputPath, format, ed.job())
}

// performExportJob exports with options read beforehand (off the UI
// thread it must not read the widgets).
func (ed *ExportDialog) performExportJob(content, outputPath, format string, job exportJob) error {
	switch format {
	case "txt":
		return ed.exportToTXT(content, outputPath)
	case "text-fountain":
		return ed.exportToFountain(content, outputPath)
	}
	if _, ok := cli.FormatFor(format); !ok {
		return fmt.Errorf("unsupported export format: %s", format)
	}
	return exportAs(content, outputPath, format, job)
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

// chosenPage is the "page-size" setting for the choice ("" for the
// script format's).
func (ed *ExportDialog) chosenPage() string {
	if ed.documentPageSelect.Selected == documentPages[0] {
		return ""
	}
	return strings.ToLower(ed.documentPageSelect.Selected)
}

// documentPage is the chosen paper's name for Lexington.
func (ed *ExportDialog) documentPage() string {
	if ed.documentPageSelect.Selected == documentPages[0] {
		return scriptFormat(selectedScriptFormat(ed.scriptFormatSelect)).Page
	}
	return strings.ToLower(ed.documentPageSelect.Selected)
}

func (ed *ExportDialog) exportToTXT(content, outputPath string) error {
	return writeFileAtomic(outputPath, []byte(content))
}

// exportToFountain writes the script as it is: Fountain already. (It
// used to trim every line, which turned a dialogue's "  " blank line
// into the end of the speech.)
func (ed *ExportDialog) exportToFountain(content, outputPath string) error {
	return writeFileAtomic(outputPath, []byte(content))
}

func (ed *ExportDialog) finishExport(err error, outputPath string) {
	if err == nil && inBrowser {
		offerDownload(outputPath)
	}
	ed.isExporting = false
	ed.exportButton.Enable()
	ed.progressBar.Hide()

	if err != nil {
		ed.statusLabel.SetText("Export failed")
		dialog.ShowError(fmt.Errorf("export failed: %v", err), ed.window.fyneWindow)
	} else {
		ed.statusLabel.SetText("Export completed successfully")

		// Show success dialog with option to open file
		confirmDialog := dialog.NewConfirm(
			"Export Complete",
			fmt.Sprintf("Document exported successfully to:\n%s\n\nWould you like to open the containing folder?", outputPath),
			func(open bool) {
				if open {
					ed.openContainingFolder(outputPath)
				}
				ed.dialog.Hide()
			},
			ed.window.fyneWindow,
		)
		confirmDialog.Show()
	}
}

func (ed *ExportDialog) openContainingFolder(filePath string) {
	dir := filepath.Dir(filePath)
	// This is platform-specific and simplified
	// In a real implementation, you'd want to handle different operating systems
	dialog.ShowInformation("Folder Location", fmt.Sprintf("File saved to: %s", dir), ed.window.fyneWindow)
}

func (ed *ExportDialog) cancel() {
	if ed.isExporting {
		// TODO: Implement export cancellation
		return
	}
	ed.dialog.Hide()
}

func (ed *ExportDialog) Show() {
	ed.dialog.Show()
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
func (ed *ExportDialog) chosenColorStyle() lex.ColorStyle {
	for _, s := range colorStyles {
		if s.label == ed.colorStyleSelect.Selected {
			return s.style
		}
	}
	return lex.ColorLine
}

// enableColorStyle lets the style be chosen while Colors is on (and can
// be); what goes with one style only is enabled with that style.
func (ed *ExportDialog) enableColorStyle() {
	if ed.colorStyleSelect == nil {
		return
	}
	on := ed.colorsCheck.Checked && !ed.colorsCheck.Disabled()
	style := ed.chosenColorStyle()
	enable := func(o fyne.Disableable, yes bool) {
		if yes {
			o.Enable()
		} else {
			o.Disable()
		}
	}
	enable(ed.colorStyleSelect, on)
	if ed.colorBlockCheck != nil {
		enable(ed.colorBlockCheck, on && (style == lex.ColorMarker || style == lex.ColorBackground || style == lex.ColorText))
		enable(ed.colorTintCheck, on && style == lex.ColorLine)
		enable(ed.colorPatternsCheck, on && style == lex.ColorLine)
	}
}

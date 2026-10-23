package main

import (
	"fmt"
	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/rules"
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

	// Progress
	progressBar *widget.ProgressBar
	statusLabel *widget.Label

	// Buttons
	exportButton  *widget.Button
	cancelButton  *widget.Button
	previewButton *widget.Button

	// Format settings container
	formatSettingsContainer *fyne.Container

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
		),
	)

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
	scrollContent := container.NewVScroll(container.NewVBox(outputSection, formatSection, fountainSection, progressSection))
	scrollContent.SetMinSize(fyne.NewSize(580, 560))
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
		note(f.format.Name + " is made with " + f.format.Needs + ", which is not installed.")
	}
	for _, o := range []fyne.Disableable{ed.includeTitlePageCheck, ed.sceneNumbersCheck, ed.previewButton} {
		if o == nil {
			continue
		}
		if f.lexington && available {
			o.Enable()
		} else {
			o.Disable()
		}
	}
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
}

func (ed *ExportDialog) saveSettings() {
	// Save export settings to preferences
	ed.settings.SetString("export-format", ed.formatSelect.Selected)
	ed.settings.SetString("export-directory", ed.outputDirEntry.Text)
	ed.settings.SetString("page-size", ed.chosenPage())
	ed.settings.SetString("script-format", selectedScriptFormat(ed.scriptFormatSelect))
	ed.settings.SetBoolean("include-title-page", ed.includeTitlePageCheck.Checked)
	ed.settings.SetBoolean("fountain-scene-numbers", ed.sceneNumbersCheck.Checked)

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

	// Check if output directory exists
	if _, err := os.Stat(ed.outputDirEntry.Text); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", ed.outputDirEntry.Text)
	}

	return nil
}

func (ed *ExportDialog) startExport() {
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
		if p, ok := rules.GetPreset(selectedScriptFormat(ed.scriptFormatSelect)); ok {
			return p.Page
		}
		return ""
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

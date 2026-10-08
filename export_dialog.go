package main

import (
	"github.com/LaPingvino/lexington/rules"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ExportDialog struct {
	window   *MainWindow
	dialog   *dialog.CustomDialog
	settings *Settings
	
	// Format selection
	formatSelect   *widget.Select
	
	// Output settings
	outputDirEntry *widget.Entry
	filenameEntry  *widget.Entry
	
	// PDF settings
	pageSizeSelect     *widget.Select
	orientationSelect  *widget.Select
	fontSelect         *widget.Select
	fontSizeEntry      *widget.Entry
	marginsEntry       *widget.Entry
	
	// HTML settings
	cssStyleSelect     *widget.Select
	embedCSSCheck      *widget.Check
	includeJSCheck     *widget.Check
	
	// Fountain settings
	includeTitlePageCheck  *widget.Check
	sceneNumbersCheck      *widget.Check
	dualDialogueCheck      *widget.Check
	scriptFormatSelect     *widget.Select
	documentPageSelect     *widget.Select
	
	// Progress
	progressBar        *widget.ProgressBar
	statusLabel        *widget.Label
	
	// Buttons
	exportButton       *widget.Button
	cancelButton       *widget.Button
	previewButton      *widget.Button
	
	// Format settings container
	formatSettingsContainer *fyne.Container
	
	// State
	isExporting        bool
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
	
	return ed
}

func (ed *ExportDialog) createWidgets() {
	// Format selection
	ed.formatSelect = widget.NewSelect(
		[]string{"PDF", "HTML", "DOCX", "ODT", "FDX", "TXT", "Fountain"},
		ed.onFormatChanged,
	)
	// Don't set selected yet - wait until after containers are created
	
	// Output settings
	ed.outputDirEntry = widget.NewEntry()
	ed.outputDirEntry.SetText(ed.getDefaultOutputDir())
	
	ed.filenameEntry = widget.NewEntry()
	ed.filenameEntry.SetText(ed.getDefaultFilename())
	
	// PDF, HTML, DOCX and ODT: the script format (Lexington's presets)
	ed.scriptFormatSelect = newScriptFormatSelect(nil, nil)
	// DOCX and ODT: the paper, or the one the script format suggests
	ed.documentPageSelect = widget.NewSelect(documentPages, nil)
	ed.documentPageSelect.SetSelected(documentPages[0])

	// PDF settings
	ed.pageSizeSelect = widget.NewSelect(
		[]string{"Letter", "A4", "Legal", "Executive"},
		nil,
	)
	ed.pageSizeSelect.SetSelected("Letter")
	
	ed.orientationSelect = widget.NewSelect(
		[]string{"Portrait", "Landscape"},
		nil,
	)
	ed.orientationSelect.SetSelected("Portrait")
	
	ed.fontSelect = widget.NewSelect(
		[]string{"Courier", "Courier New", "Times New Roman", "Arial", "Helvetica"},
		nil,
	)
	ed.fontSelect.SetSelected("Courier")
	
	ed.fontSizeEntry = widget.NewEntry()
	ed.fontSizeEntry.SetText("12")
	
	ed.marginsEntry = widget.NewEntry()
	ed.marginsEntry.SetText("1.0")
	ed.marginsEntry.SetPlaceHolder("Inches")
	
	// HTML settings
	ed.cssStyleSelect = widget.NewSelect(
		[]string{"Default", "Minimal", "Classic", "Modern"},
		nil,
	)
	ed.cssStyleSelect.SetSelected("Default")
	
	ed.embedCSSCheck = widget.NewCheck("Embed CSS", nil)
	ed.embedCSSCheck.SetChecked(true)
	
	ed.includeJSCheck = widget.NewCheck("Include JavaScript", nil)
	
	// Fountain settings
	ed.includeTitlePageCheck = widget.NewCheck("Include title page", nil)
	ed.includeTitlePageCheck.SetChecked(true)
	
	ed.sceneNumbersCheck = widget.NewCheck("Include scene numbers", nil)
	
	ed.dualDialogueCheck = widget.NewCheck("Format dual dialogue", nil)
	ed.dualDialogueCheck.SetChecked(true)
	
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
	
	outputSection := widget.NewCard("Output", "",
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Format:"), nil, ed.formatSelect),
			container.NewBorder(nil, nil, widget.NewLabel("Directory:"), browseButton, ed.outputDirEntry),
			container.NewBorder(nil, nil, widget.NewLabel("Filename:"), nil, ed.filenameEntry),
		),
	)
	
	// Format-specific settings
	ed.formatSettingsContainer = container.NewVBox()
	
	// Now it's safe to set the selected format which will call updateFormatSettings
	ed.formatSelect.SetSelected("PDF")
	
	formatSection := widget.NewCard("Format Settings", "", ed.formatSettingsContainer)
	
	// Fountain options
	fountainSection := widget.NewCard("Fountain Options", "",
		container.NewVBox(
			ed.includeTitlePageCheck,
			ed.sceneNumbersCheck,
			ed.dualDialogueCheck,
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
	
	switch ed.formatSelect.Selected {
	case "PDF", "HTML", "DOCX", "ODT":
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Script format:"), nil, ed.scriptFormatSelect),
		)
	}
	switch ed.formatSelect.Selected {
	case "PDF":
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Page size:"), nil, ed.pageSizeSelect),
		)
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Orientation:"), nil, ed.orientationSelect),
		)
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Font:"), nil, ed.fontSelect),
		)
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Font size:"), nil, ed.fontSizeEntry),
		)
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Margins:"), nil, ed.marginsEntry),
		)
		
	case "HTML":
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Style:"), nil, ed.cssStyleSelect),
		)
		ed.formatSettingsContainer.Add(ed.embedCSSCheck)
		ed.formatSettingsContainer.Add(ed.includeJSCheck)
		
	case "DOCX", "ODT":
		ed.formatSettingsContainer.Add(
			container.NewBorder(nil, nil, widget.NewLabel("Paper:"), nil, ed.documentPageSelect),
		)
		ed.formatSettingsContainer.Add(widget.NewLabel("Every element is a paragraph style, to restyle in Word or LibreOffice."))
		
	case "FDX":
		ed.formatSettingsContainer.Add(
			widget.NewLabel("Final Draft document (.fdx)."),
		)
		
	case "TXT":
		ed.formatSettingsContainer.Add(
			widget.NewLabel("Plain text export with Fountain formatting preserved."),
		)
		
	case "Fountain":
		ed.formatSettingsContainer.Add(
			widget.NewLabel("Export as standard Fountain format."),
		)
	}
	
	ed.formatSettingsContainer.Refresh()
}

func (ed *ExportDialog) onFormatChanged(format string) {
	ed.updateFormatSettings()
	ed.updateDefaultFilename()
}

func (ed *ExportDialog) loadSettings() {
	// Load export settings from preferences
	ed.formatSelect.SetSelected(ed.settings.GetString("export-format"))
	
	if dir := ed.settings.GetString("export-directory"); dir != "" {
		ed.outputDirEntry.SetText(dir)
	}
	
	ed.pageSizeSelect.SetSelected(ed.settings.GetString("page-size"))
	selectScriptFormat(ed.scriptFormatSelect, ed.settings.GetString("script-format"))
	ed.fontSelect.SetSelected(ed.settings.GetString("font-name"))
	ed.fontSizeEntry.SetText(strconv.Itoa(ed.settings.GetInt("font-size-export")))
	
	ed.includeTitlePageCheck.SetChecked(ed.settings.GetBoolean("include-title-page"))
	ed.sceneNumbersCheck.SetChecked(ed.settings.GetBoolean("fountain-scene-numbers"))
	ed.dualDialogueCheck.SetChecked(ed.settings.GetBoolean("fountain-dual-dialogue"))
}

func (ed *ExportDialog) saveSettings() {
	// Save export settings to preferences
	ed.settings.SetString("export-format", ed.formatSelect.Selected)
	ed.settings.SetString("export-directory", ed.outputDirEntry.Text)
	ed.settings.SetString("page-size", ed.pageSizeSelect.Selected)
	ed.settings.SetString("script-format", selectedScriptFormat(ed.scriptFormatSelect))
	ed.settings.SetString("font-name", ed.fontSelect.Selected)
	
	if fontSize, err := strconv.Atoi(ed.fontSizeEntry.Text); err == nil {
		ed.settings.SetInt("font-size-export", fontSize)
	}
	
	ed.settings.SetBoolean("include-title-page", ed.includeTitlePageCheck.Checked)
	ed.settings.SetBoolean("fountain-scene-numbers", ed.sceneNumbersCheck.Checked)
	ed.settings.SetBoolean("fountain-dual-dialogue", ed.dualDialogueCheck.Checked)
	
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
	var ext string
	
	switch ed.formatSelect.Selected {
	case "PDF":
		ext = ".pdf"
	case "HTML":
		ext = ".html"
	case "DOCX":
		ext = ".docx"
	case "ODT":
		ext = ".odt"
	case "FDX":
		ext = ".fdx"
	case "TXT":
		ext = ".txt"
	case "Fountain":
		ext = ".fountain"
	default:
		ext = ".pdf"
	}
	
	ed.filenameEntry.SetText(base + ext)
}

func (ed *ExportDialog) browseOutputDir() {
	dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
		if err == nil && folder != nil {
			ed.outputDirEntry.SetText(folder.Path())
		}
	}, ed.window.fyneWindow)
}

func (ed *ExportDialog) previewExport() {
	// TODO: Implement export preview
	dialog.ShowInformation("Preview", "Export preview is not yet implemented.", ed.window.fyneWindow)
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
	
	ed.startExport()
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
	
	// Validate font size for PDF/DOCX
	if ed.formatSelect.Selected == "PDF" {
		if fontSize, err := strconv.Atoi(ed.fontSizeEntry.Text); err != nil || fontSize < 6 || fontSize > 72 {
			return fmt.Errorf("font size must be between 6 and 72")
		}
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
	format := strings.ToLower(ed.formatSelect.Selected)
	content := ed.window.textEditor.Text()
	
	// Perform export in goroutine
	go func() {
		err := ed.performExport(content, outputPath, format)
		
		// Update UI on main thread
		fyne.Do(func() { ed.finishExport(err, outputPath) })
	}()
}

func (ed *ExportDialog) performExport(content, outputPath, format string) error {
	switch format {
	case "pdf":
		return ed.exportToPDF(content, outputPath)
	case "html":
		return ed.exportToHTML(content, outputPath)
	case "fdx":
		return exportFDX(content, outputPath)
	case "docx", "odt":
		set := scriptFormatElements(selectedScriptFormat(ed.scriptFormatSelect))
		return exportDocument(content, outputPath, format, set, ed.documentPage())
	case "txt":
		return ed.exportToTXT(content, outputPath)
	case "fountain":
		return ed.exportToFountain(content, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

func (ed *ExportDialog) exportToPDF(content, outputPath string) error {
	return exportPDF(content, outputPath, scriptFormatElements(selectedScriptFormat(ed.scriptFormatSelect)))
}

func (ed *ExportDialog) exportToHTML(content, outputPath string) error {
	return exportHTML(content, outputPath, scriptFormatElements(selectedScriptFormat(ed.scriptFormatSelect)))
}

// documentPages are the paper choices for DOCX and ODT; the first is the
// script format's suggestion (A5 for a musical), else Letter.
var documentPages = []string{"Script format's", "Letter", "A4", "A5"}

// documentPage is the chosen paper's name for Lexington ("" for the
// script format's).
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
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func (ed *ExportDialog) exportToFountain(content, outputPath string) error {
	// Clean up the content for proper Fountain format
	cleaned := ed.cleanFountainContent(content)
	return os.WriteFile(outputPath, []byte(cleaned), 0644)
}

func (ed *ExportDialog) cleanFountainContent(content string) string {
	// Basic cleaning for Fountain export
	lines := strings.Split(content, "\n")
	var cleaned strings.Builder
	
	for _, line := range lines {
		// Remove excessive whitespace but preserve structure
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			cleaned.WriteString("\n")
		} else {
			cleaned.WriteString(trimmed + "\n")
		}
	}
	
	return cleaned.String()
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
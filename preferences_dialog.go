package main

import (
	"fyne.io/fyne/v2/driver/desktop"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// PreferencesDialog is the Preferences window. It shows only settings
// that do something: each one is read by the window, the editor or the
// Export dialog.
type PreferencesDialog struct {
	app      *Application
	win      fyne.Window // a window of its own, not an overlay on the editor
	tabs     *container.AppTabs
	settings *Settings

	// General
	themeSelect           *widget.Select
	toolbarVisibleCheck   *widget.Check
	statusbarVisibleCheck *widget.Check
	autoSaveCheck         *widget.Check
	autoSaveInterval      *widget.Entry

	// Editor
	fontFamilyEntry        *widget.Entry
	fontSizeEntry          *widget.Entry
	lineNumbersCheck       *widget.Check
	autoIndentCheck        *widget.Check
	autoCloseBracketsCheck *widget.Check
	keysSelect             *widget.Select

	// Script and export
	languageSelect        *widget.Select
	scriptFormatSelect    *widget.Select
	scriptFormatDesc      *widget.Label
	exportFormatSelect    *widget.Select
	exportDirEntry        *widget.Entry
	paperSelect           *widget.Select
	includeTitlePageCheck *widget.Check
	numberScenesCheck     *widget.Check
}

// exportFormats are the Export dialog's formats, as it names them.
var exportFormats = []string{"PDF", "HTML", "DOCX", "ODT", "FDX", "TXT", "Fountain"}

// exportFormatLabel is the format for an "export-format" setting
// (older settings have "pdf").
func exportFormatLabel(setting string) string {
	for _, f := range exportFormats {
		if strings.EqualFold(f, setting) {
			return f
		}
	}
	return exportFormats[0]
}

func NewPreferencesDialog(app *Application, parent fyne.Window) *PreferencesDialog {
	pd := &PreferencesDialog{
		app:      app,
		settings: GetSettings(),
	}

	pd.createWidgets()
	pd.loadCurrentSettings()
	content := pd.createContent()

	pd.win = app.fyneApp.NewWindow("Preferences")
	pd.win.SetContent(content)
	pd.keys()
	pd.win.Resize(fyne.NewSize(640, 540))
	pd.win.CenterOnScreen()
	return pd
}

func (pd *PreferencesDialog) createWidgets() {
	pd.themeSelect = widget.NewSelect([]string{"system", "light", "dark", "sepia"}, nil)
	pd.toolbarVisibleCheck = widget.NewCheck("Show toolbar", nil)
	pd.statusbarVisibleCheck = widget.NewCheck("Show status bar", nil)
	pd.autoSaveCheck = widget.NewCheck("Auto-save documents", nil)
	pd.autoSaveInterval = widget.NewEntry()
	pd.autoSaveInterval.SetPlaceHolder("30")

	pd.fontFamilyEntry = widget.NewEntry()
	pd.fontFamilyEntry.SetPlaceHolder("monospace, system, or a font file")
	pd.fontSizeEntry = widget.NewEntry()
	pd.fontSizeEntry.SetPlaceHolder("12")
	pd.lineNumbersCheck = widget.NewCheck("Show line numbers", nil)
	pd.autoIndentCheck = widget.NewCheck("Format the line and indent the next one on Enter", nil)
	pd.autoCloseBracketsCheck = widget.NewCheck("Close brackets as you type them: ( and [", nil)
	pd.keysSelect = widget.NewSelect([]string{"standard", "vim", "helix"}, nil)

	pd.languageSelect = widget.NewSelect(scriptLanguages(), nil)
	pd.scriptFormatDesc = widget.NewLabel("")
	pd.scriptFormatDesc.Wrapping = fyne.TextWrapWord
	pd.scriptFormatSelect = newScriptFormatSelect(pd.scriptFormatDesc, nil)
	fyne.SetAccessibleLabel(pd.scriptFormatSelect, "Script format")
	pd.exportFormatSelect = widget.NewSelect(exportFormats, nil)
	pd.exportDirEntry = widget.NewEntry()
	pd.exportDirEntry.SetPlaceHolder("Choose export directory...")
	pd.paperSelect = widget.NewSelect(documentPages, nil)
	pd.includeTitlePageCheck = widget.NewCheck("Include the title page", nil)
	pd.numberScenesCheck = widget.NewCheck("Number the scenes (those without a number)", nil)
}

func (pd *PreferencesDialog) createContent() *fyne.Container {
	tabs := container.NewAppTabs(
		container.NewTabItem("General", container.NewVScroll(pd.createGeneralTab())),
		container.NewTabItem("Editor", container.NewVScroll(pd.createEditorTab())),
		container.NewTabItem("Script", container.NewVScroll(pd.createScriptTab())),
	)
	pd.tabs = tabs

	// below the tabs, which scroll: the buttons never cover a setting
	buttons := container.NewHBox(
		widget.NewButton("Reset to Defaults", pd.resetToDefaults),
		layout.NewSpacer(),
		widget.NewButton("Cancel", pd.cancel),
		widget.NewButton("Apply", pd.apply),
		widget.NewButton("OK", pd.ok),
	)
	return container.NewBorder(nil, buttons, nil, nil, tabs)
}

func labelled(label string, o fyne.CanvasObject) fyne.CanvasObject {
	return labelledWith(label, o, nil)
}

// labelledWith is a field with its label before it and trailing (a button)
// after it; screen readers call the field by the label.
func labelledWith(label string, o, trailing fyne.CanvasObject) fyne.CanvasObject {
	fyne.SetAccessibleLabel(o, strings.TrimSuffix(strings.TrimSpace(label), ":"))
	return container.NewBorder(nil, nil, widget.NewLabel(label), trailing, o)
}

func (pd *PreferencesDialog) createGeneralTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewCard("Appearance", "", container.NewVBox(
			labelled("Theme:", pd.themeSelect),
			pd.toolbarVisibleCheck,
			pd.statusbarVisibleCheck,
		)),
		widget.NewCard("Saving", "", container.NewVBox(
			pd.autoSaveCheck,
			labelled("Auto-save interval (seconds):", pd.autoSaveInterval),
		)),
	)
}

func (pd *PreferencesDialog) createEditorTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewCard("Font", "", container.NewVBox(
			labelled("Font:", pd.fontFamilyEntry),
			labelled("Size:", pd.fontSizeEntry),
		)),
		widget.NewCard("Writing", "", container.NewVBox(
			pd.lineNumbersCheck,
			pd.autoIndentCheck,
			pd.autoCloseBracketsCheck,
			labelled("Keys (vim, helix: Escape for normal mode):", pd.keysSelect),
		)),
	)
}

func (pd *PreferencesDialog) createScriptTab() fyne.CanvasObject {
	browse := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
			if err == nil && folder != nil {
				pd.exportDirEntry.SetText(folder.Path())
			}
		}, pd.win)
	})
	return container.NewVBox(
		widget.NewCard("Script Format", "How the preview and the exports lay the script out",
			container.NewVBox(pd.scriptFormatSelect, pd.scriptFormatDesc),
		),
		widget.NewCard("Language", "Of a new script: its scene headings (INT./EXT., EN./EKST., BIN./BUI., ...). A file named like scene.eo.fountain says its own.",
			labelled("Language:", pd.languageSelect),
		),
		widget.NewCard("Export", "The Export dialog starts with these", container.NewVBox(
			labelled("Format:", pd.exportFormatSelect),
			labelledWith("Directory:", pd.exportDirEntry, browse),
			labelled("Paper (PDF, DOCX, ODT):", pd.paperSelect),
			pd.includeTitlePageCheck,
			pd.numberScenesCheck,
		)),
	)
}

func (pd *PreferencesDialog) loadCurrentSettings() { pd.loadFrom(pd.settings) }

// loadFrom shows the settings of s in the window.
func (pd *PreferencesDialog) loadFrom(s *Settings) {
	pd.themeSelect.SetSelected(s.GetString("theme"))
	pd.toolbarVisibleCheck.SetChecked(s.GetBoolean("toolbar-visible"))
	pd.statusbarVisibleCheck.SetChecked(s.GetBoolean("statusbar-visible"))
	pd.autoSaveCheck.SetChecked(s.GetBoolean("auto-save"))
	pd.autoSaveInterval.SetText(strconv.Itoa(s.GetInt("auto-save-interval")))

	pd.fontFamilyEntry.SetText(s.GetString("font-family"))
	pd.fontSizeEntry.SetText(strconv.Itoa(s.GetInt("font-size")))
	pd.lineNumbersCheck.SetChecked(s.GetBoolean("show-line-numbers"))
	pd.autoIndentCheck.SetChecked(s.GetBoolean("auto-indent"))
	pd.autoCloseBracketsCheck.SetChecked(s.GetBoolean("auto-close-brackets"))
	pd.keysSelect.SetSelected((&MainWindow{settings: s}).editorKeys())

	selectScriptFormat(pd.scriptFormatSelect, s.GetString("script-format"))
	pd.languageSelect.SetSelected(s.GetString("script-language"))
	pd.exportFormatSelect.SetSelected(exportFormatLabel(s.GetString("export-format")))
	pd.exportDirEntry.SetText(s.GetString("export-directory"))
	pd.paperSelect.SetSelected(documentPageLabel(s.GetString("page-size")))
	pd.includeTitlePageCheck.SetChecked(s.GetBoolean("include-title-page"))
	pd.numberScenesCheck.SetChecked(s.GetBoolean("fountain-scene-numbers"))
}

func (pd *PreferencesDialog) saveSettings() {
	s := pd.settings
	if pd.themeSelect.Selected != "" {
		s.SetString("theme", pd.themeSelect.Selected)
		pd.app.setColorScheme(pd.themeSelect.Selected)
	}
	s.SetBoolean("toolbar-visible", pd.toolbarVisibleCheck.Checked)
	s.SetBoolean("statusbar-visible", pd.statusbarVisibleCheck.Checked)
	s.SetBoolean("auto-save", pd.autoSaveCheck.Checked)
	if interval, err := strconv.Atoi(pd.autoSaveInterval.Text); err == nil && interval > 0 {
		s.SetInt("auto-save-interval", interval)
	}

	s.SetString("font-family", pd.fontFamilyEntry.Text)
	if size, err := strconv.Atoi(pd.fontSizeEntry.Text); err == nil && size > 0 {
		s.SetInt("font-size", size)
	}
	s.SetBoolean("show-line-numbers", pd.lineNumbersCheck.Checked)
	s.SetBoolean("auto-indent", pd.autoIndentCheck.Checked)
	s.SetBoolean("auto-close-brackets", pd.autoCloseBracketsCheck.Checked)
	if pd.keysSelect.Selected != "" {
		s.SetString("editor-keys", pd.keysSelect.Selected)
		s.SetBoolean("vim-mode", false) // the older setting
	}

	s.SetString("script-format", selectedScriptFormat(pd.scriptFormatSelect))
	if pd.languageSelect.Selected != "" {
		s.SetString("script-language", pd.languageSelect.Selected)
	}
	s.SetString("export-format", exportFormatLabel(pd.exportFormatSelect.Selected))
	s.SetString("export-directory", pd.exportDirEntry.Text)
	page := ""
	if pd.paperSelect.Selected != documentPages[0] {
		page = strings.ToLower(pd.paperSelect.Selected)
	}
	s.SetString("page-size", page)
	s.SetBoolean("include-title-page", pd.includeTitlePageCheck.Checked)
	s.SetBoolean("fountain-scene-numbers", pd.numberScenesCheck.Checked)

	s.Save()
}

func (pd *PreferencesDialog) resetToDefaults() {
	dialog.ShowConfirm(
		"Reset Settings",
		"Are you sure you want to reset all settings to their defaults? This cannot be undone.",
		func(confirmed bool) {
			if confirmed {
				// the defaults shown: saved with OK or Apply, not before,
				// and the recent files and contact details are not
				// preferences, so they stay
				defaults := &Settings{data: map[string]interface{}{}}
				defaults.loadDefaults()
				pd.loadFrom(defaults)
			}
		},
		pd.win,
	)
}

func (pd *PreferencesDialog) apply() {
	pd.saveSettings()
	pd.app.applySettingsToWindows()
	dialog.ShowInformation("Settings Applied", "Your preferences have been saved.", pd.win)
}

func (pd *PreferencesDialog) ok() {
	pd.saveSettings()
	pd.app.applySettingsToWindows()
	pd.win.Close()
}

func (pd *PreferencesDialog) cancel() {
	pd.win.Close()
}

func (pd *PreferencesDialog) Show() {
	pd.win.Show()
}

// keys are the window's keys, whatever has the focus: Escape cancels,
// Ctrl+PageUp and Ctrl+PageDown go to the page before or after.
func (pd *PreferencesDialog) keys() {
	c, ok := pd.win.Canvas().(desktop.KeyPreviewCanvas)
	if !ok {
		return
	}
	c.SetOnKeyPreview(func(key fyne.KeyName, mod fyne.KeyModifier) bool {
		switch {
		case key == fyne.KeyEscape && mod == 0:
			pd.cancel()
		case key == fyne.KeyPageDown && mod == fyne.KeyModifierControl:
			pd.turnPage(1)
		case key == fyne.KeyPageUp && mod == fyne.KeyModifierControl:
			pd.turnPage(-1)
		default:
			return false
		}
		return true
	})
}

// turnPage shows the page step pages on (wrapping around).
func (pd *PreferencesDialog) turnPage(step int) {
	n := len(pd.tabs.Items)
	pd.tabs.SelectIndex(((pd.tabs.SelectedIndex()+step)%n + n) % n)
}

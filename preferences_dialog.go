package main

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type PreferencesDialog struct {
	app      *Application
	win      fyne.Window // a window of its own, not an overlay on the editor
	settings *Settings
	
	// Theme settings
	themeSelect      *widget.Select
	colorSchemeSelect *widget.Select
	
	// Editor settings
	fontFamilyEntry  *widget.Entry
	fontSizeEntry    *widget.Entry
	lineNumbersCheck *widget.Check
	autoIndentCheck  *widget.Check
	showLineNumbersCheck *widget.Check
	tabWidthEntry    *widget.Entry
	useSpacesCheck   *widget.Check
	
	// Writing settings
	autoSaveCheck    *widget.Check
	autoSaveInterval *widget.Entry
	spellCheckCheck  *widget.Check
	spellCheckLang   *widget.Select
	smartQuotesCheck *widget.Check
	
	// Window settings
	toolbarVisibleCheck   *widget.Check
	statusbarVisibleCheck *widget.Check
	autohideHeaderCheck   *widget.Check
	
	// Export settings
	exportFormatSelect   *widget.Select
	exportDirEntry       *widget.Entry
	includeTitlePageCheck *widget.Check
	pageSizeSelect       *widget.Select
	exportFontEntry      *widget.Entry
	exportFontSizeEntry  *widget.Entry
	
	// Fountain settings
	fountainAutoFormatCheck  *widget.Check
	fountainSceneNumbersCheck *widget.Check
	fountainDualDialogueCheck *widget.Check
	fountainTitlePageCheck   *widget.Check
	scriptFormatSelect       *widget.Select
	scriptFormatDesc         *widget.Label
	
	// Advanced settings
	smoothScrollingCheck      *widget.Check
	showWhitespaceCheck      *widget.Check
	highlightMatchingCheck   *widget.Check
	autoCloseBracketsCheck   *widget.Check
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
	pd.win.Resize(fyne.NewSize(640, 540))
	pd.win.CenterOnScreen()
	return pd
}

func (pd *PreferencesDialog) createWidgets() {
	// Theme settings
	pd.themeSelect = widget.NewSelect([]string{"system", "light", "dark", "sepia"}, nil)
	pd.colorSchemeSelect = widget.NewSelect([]string{"system", "light", "dark"}, nil)
	
	// Editor settings
	pd.fontFamilyEntry = widget.NewEntry()
	pd.fontFamilyEntry.SetPlaceHolder("monospace")
	
	pd.fontSizeEntry = widget.NewEntry()
	pd.fontSizeEntry.SetPlaceHolder("12")
	
	pd.lineNumbersCheck = widget.NewCheck("Show line numbers", nil)
	pd.autoIndentCheck = widget.NewCheck("Auto-indent", nil)
	pd.showLineNumbersCheck = widget.NewCheck("Show line numbers", nil)
	
	pd.tabWidthEntry = widget.NewEntry()
	pd.tabWidthEntry.SetPlaceHolder("4")
	
	pd.useSpacesCheck = widget.NewCheck("Use spaces instead of tabs", nil)
	
	// Writing settings
	pd.autoSaveCheck = widget.NewCheck("Auto-save documents", nil)
	
	pd.autoSaveInterval = widget.NewEntry()
	pd.autoSaveInterval.SetPlaceHolder("30")
	
	pd.spellCheckCheck = widget.NewCheck("Enable spell checking", nil)
	pd.spellCheckLang = widget.NewSelect([]string{"en_US", "en_GB", "es_ES", "fr_FR", "de_DE"}, nil)
	pd.smartQuotesCheck = widget.NewCheck("Use smart quotes", nil)
	
	// Window settings
	pd.toolbarVisibleCheck = widget.NewCheck("Show toolbar", nil)
	pd.statusbarVisibleCheck = widget.NewCheck("Show status bar", nil)
	pd.autohideHeaderCheck = widget.NewCheck("Auto-hide header bar in fullscreen", nil)
	
	// Export settings
	pd.exportFormatSelect = widget.NewSelect([]string{"pdf", "html", "docx", "fountain"}, nil)
	pd.exportDirEntry = widget.NewEntry()
	pd.exportDirEntry.SetPlaceHolder("Choose export directory...")
	
	pd.includeTitlePageCheck = widget.NewCheck("Include title page", nil)
	pd.pageSizeSelect = widget.NewSelect([]string{"letter", "a4", "legal"}, nil)
	
	pd.exportFontEntry = widget.NewEntry()
	pd.exportFontEntry.SetPlaceHolder("Courier")
	
	pd.exportFontSizeEntry = widget.NewEntry()
	pd.exportFontSizeEntry.SetPlaceHolder("12")
	
	// Fountain settings
	pd.fountainAutoFormatCheck = widget.NewCheck("Auto-format Fountain syntax", nil)
	pd.fountainSceneNumbersCheck = widget.NewCheck("Show scene numbers", nil)
	pd.fountainDualDialogueCheck = widget.NewCheck("Enable dual dialogue", nil)
	pd.fountainTitlePageCheck = widget.NewCheck("Generate title page", nil)
	pd.scriptFormatDesc = widget.NewLabel("")
	pd.scriptFormatDesc.Wrapping = fyne.TextWrapWord
	pd.scriptFormatSelect = newScriptFormatSelect(pd.scriptFormatDesc, nil)
	
	// Advanced settings
	pd.smoothScrollingCheck = widget.NewCheck("Smooth scrolling", nil)
	pd.showWhitespaceCheck = widget.NewCheck("Show whitespace", nil)
	pd.highlightMatchingCheck = widget.NewCheck("Highlight matching brackets", nil)
	pd.autoCloseBracketsCheck = widget.NewCheck("Auto-close brackets", nil)
}

func (pd *PreferencesDialog) createContent() *fyne.Container {
	tabs := container.NewAppTabs()
	
	// General tab
	generalTab := pd.createGeneralTab()
	tabs.Append(container.NewTabItem("General", container.NewVScroll(generalTab)))
	
	// Editor tab
	editorTab := pd.createEditorTab()
	tabs.Append(container.NewTabItem("Editor", container.NewVScroll(editorTab)))
	
	// Export tab
	exportTab := pd.createExportTab()
	tabs.Append(container.NewTabItem("Export", container.NewVScroll(exportTab)))
	
	// Fountain tab
	fountainTab := pd.createFountainTab()
	tabs.Append(container.NewTabItem("Fountain", container.NewVScroll(fountainTab)))
	
	// Advanced tab
	advancedTab := pd.createAdvancedTab()
	tabs.Append(container.NewTabItem("Advanced", container.NewVScroll(advancedTab)))
	
	// Buttons at the bottom
	// below the tabs, which scroll: the buttons never cover a setting
	buttonsContainer := container.NewHBox(
		widget.NewButton("Reset to Defaults", pd.resetToDefaults),
		layout.NewSpacer(),
		widget.NewButton("Cancel", pd.cancel),
		widget.NewButton("Apply", pd.apply),
		widget.NewButton("OK", pd.ok),
	)
	
	content := container.NewBorder(
		nil,           // top
		buttonsContainer, // bottom
		nil,           // left
		nil,           // right
		tabs,          // center
	)
	
	return content
}

func (pd *PreferencesDialog) createGeneralTab() *fyne.Container {
	return container.NewVBox(
		widget.NewCard("Theme", "",
			container.NewVBox(
				container.NewHBox(widget.NewLabel("Theme:"), pd.themeSelect),
				container.NewHBox(widget.NewLabel("Color scheme:"), pd.colorSchemeSelect),
			),
		),
		widget.NewCard("Window", "",
			container.NewVBox(
				pd.toolbarVisibleCheck,
				pd.statusbarVisibleCheck,
				pd.autohideHeaderCheck,
			),
		),
		widget.NewCard("Writing", "",
			container.NewVBox(
				pd.autoSaveCheck,
				container.NewHBox(widget.NewLabel("Auto-save interval (seconds):"), pd.autoSaveInterval),
				pd.spellCheckCheck,
				container.NewHBox(widget.NewLabel("Spell check language:"), pd.spellCheckLang),
				pd.smartQuotesCheck,
			),
		),
	)
}

func (pd *PreferencesDialog) createEditorTab() *fyne.Container {
	return container.NewVBox(
		widget.NewCard("Font", "",
			container.NewVBox(
				container.NewHBox(widget.NewLabel("Font family:"), pd.fontFamilyEntry),
				container.NewHBox(widget.NewLabel("Font size:"), pd.fontSizeEntry),
			),
		),
		widget.NewCard("Text Editing", "",
			container.NewVBox(
				pd.lineNumbersCheck,
				pd.autoIndentCheck,
				pd.showLineNumbersCheck,
				container.NewHBox(widget.NewLabel("Tab width:"), pd.tabWidthEntry),
				pd.useSpacesCheck,
			),
		),
	)
}

func (pd *PreferencesDialog) createExportTab() *fyne.Container {
	browseButton := widget.NewButton("Browse...", func() {
		dialog.ShowFolderOpen(func(folder fyne.ListableURI, err error) {
			if err == nil && folder != nil {
				pd.exportDirEntry.SetText(folder.Path())
			}
		}, pd.app.windows[0].fyneWindow)
	})
	
	return container.NewVBox(
		widget.NewCard("Export Settings", "",
			container.NewVBox(
				container.NewHBox(widget.NewLabel("Default format:"), pd.exportFormatSelect),
				container.NewHBox(
					widget.NewLabel("Export directory:"),
					container.NewHBox(pd.exportDirEntry, browseButton),
				),
				pd.includeTitlePageCheck,
				container.NewHBox(widget.NewLabel("Page size:"), pd.pageSizeSelect),
			),
		),
		widget.NewCard("Export Font", "",
			container.NewVBox(
				container.NewHBox(widget.NewLabel("Font name:"), pd.exportFontEntry),
				container.NewHBox(widget.NewLabel("Font size:"), pd.exportFontSizeEntry),
			),
		),
	)
}

func (pd *PreferencesDialog) createFountainTab() *fyne.Container {
	return container.NewVBox(
		widget.NewCard("Fountain Format", "",
			container.NewVBox(
				pd.fountainAutoFormatCheck,
				pd.fountainSceneNumbersCheck,
				pd.fountainDualDialogueCheck,
				pd.fountainTitlePageCheck,
			),
		),
		widget.NewCard("Script Format", "How the preview, PDF and HTML lay the script out",
			container.NewVBox(pd.scriptFormatSelect, pd.scriptFormatDesc),
		),
	)
}

func (pd *PreferencesDialog) createAdvancedTab() *fyne.Container {
	return container.NewVBox(
		widget.NewCard("Editor Behavior", "",
			container.NewVBox(
				pd.smoothScrollingCheck,
				pd.showWhitespaceCheck,
				pd.highlightMatchingCheck,
				pd.autoCloseBracketsCheck,
			),
		),
		widget.NewCard("Performance", "",
			container.NewVBox(
				widget.NewLabel("Performance settings will be added here."),
			),
		),
	)
}

func (pd *PreferencesDialog) loadCurrentSettings() {
	// Theme settings
	pd.themeSelect.SetSelected(pd.settings.GetString("theme"))
	pd.colorSchemeSelect.SetSelected(pd.settings.GetString("color-scheme"))
	
	// Editor settings
	pd.fontFamilyEntry.SetText(pd.settings.GetString("font-family"))
	pd.fontSizeEntry.SetText(strconv.Itoa(pd.settings.GetInt("font-size")))
	pd.lineNumbersCheck.SetChecked(pd.settings.GetBoolean("show-line-numbers"))
	pd.autoIndentCheck.SetChecked(pd.settings.GetBoolean("auto-indent"))
	pd.showLineNumbersCheck.SetChecked(pd.settings.GetBoolean("show-line-numbers"))
	pd.tabWidthEntry.SetText(strconv.Itoa(pd.settings.GetInt("tab-width")))
	pd.useSpacesCheck.SetChecked(pd.settings.GetBoolean("use-spaces"))
	
	// Writing settings
	pd.autoSaveCheck.SetChecked(pd.settings.GetBoolean("auto-save"))
	pd.autoSaveInterval.SetText(strconv.Itoa(pd.settings.GetInt("auto-save-interval")))
	pd.spellCheckCheck.SetChecked(pd.settings.GetBoolean("spell-check"))
	pd.spellCheckLang.SetSelected(pd.settings.GetString("spell-check-language"))
	pd.smartQuotesCheck.SetChecked(pd.settings.GetBoolean("smart-quotes"))
	
	// Window settings
	pd.toolbarVisibleCheck.SetChecked(pd.settings.GetBoolean("toolbar-visible"))
	pd.statusbarVisibleCheck.SetChecked(pd.settings.GetBoolean("statusbar-visible"))
	pd.autohideHeaderCheck.SetChecked(pd.settings.GetBoolean("autohide-headerbar"))
	
	// Export settings
	pd.exportFormatSelect.SetSelected(pd.settings.GetString("export-format"))
	pd.exportDirEntry.SetText(pd.settings.GetString("export-directory"))
	pd.includeTitlePageCheck.SetChecked(pd.settings.GetBoolean("include-title-page"))
	pd.pageSizeSelect.SetSelected(pd.settings.GetString("page-size"))
	pd.exportFontEntry.SetText(pd.settings.GetString("font-name"))
	pd.exportFontSizeEntry.SetText(strconv.Itoa(pd.settings.GetInt("font-size-export")))
	
	// Fountain settings
	pd.fountainAutoFormatCheck.SetChecked(pd.settings.GetBoolean("fountain-auto-format"))
	pd.fountainSceneNumbersCheck.SetChecked(pd.settings.GetBoolean("fountain-scene-numbers"))
	pd.fountainDualDialogueCheck.SetChecked(pd.settings.GetBoolean("fountain-dual-dialogue"))
	pd.fountainTitlePageCheck.SetChecked(pd.settings.GetBoolean("fountain-title-page"))
	selectScriptFormat(pd.scriptFormatSelect, pd.settings.GetString("script-format"))
	
	// Advanced settings
	pd.smoothScrollingCheck.SetChecked(pd.settings.GetBoolean("smooth-scrolling"))
	pd.showWhitespaceCheck.SetChecked(pd.settings.GetBoolean("show-whitespace"))
	pd.highlightMatchingCheck.SetChecked(pd.settings.GetBoolean("highlight-matching-brackets"))
	pd.autoCloseBracketsCheck.SetChecked(pd.settings.GetBoolean("auto-close-brackets"))
}

func (pd *PreferencesDialog) saveSettings() {
	// Theme settings
	if pd.themeSelect.Selected != "" {
		pd.settings.SetString("theme", pd.themeSelect.Selected)
		pd.app.setColorScheme(pd.themeSelect.Selected)
	}
	if pd.colorSchemeSelect.Selected != "" {
		pd.settings.SetString("color-scheme", pd.colorSchemeSelect.Selected)
	}
	
	// Editor settings
	pd.settings.SetString("font-family", pd.fontFamilyEntry.Text)
	if fontSize, err := strconv.Atoi(pd.fontSizeEntry.Text); err == nil {
		pd.settings.SetInt("font-size", fontSize)
	}
	pd.settings.SetBoolean("show-line-numbers", pd.lineNumbersCheck.Checked)
	pd.settings.SetBoolean("auto-indent", pd.autoIndentCheck.Checked)
	pd.settings.SetBoolean("show-line-numbers", pd.showLineNumbersCheck.Checked)
	if tabWidth, err := strconv.Atoi(pd.tabWidthEntry.Text); err == nil {
		pd.settings.SetInt("tab-width", tabWidth)
	}
	pd.settings.SetBoolean("use-spaces", pd.useSpacesCheck.Checked)
	
	// Writing settings
	pd.settings.SetBoolean("auto-save", pd.autoSaveCheck.Checked)
	if interval, err := strconv.Atoi(pd.autoSaveInterval.Text); err == nil {
		pd.settings.SetInt("auto-save-interval", interval)
	}
	pd.settings.SetBoolean("spell-check", pd.spellCheckCheck.Checked)
	if pd.spellCheckLang.Selected != "" {
		pd.settings.SetString("spell-check-language", pd.spellCheckLang.Selected)
	}
	pd.settings.SetBoolean("smart-quotes", pd.smartQuotesCheck.Checked)
	
	// Window settings
	pd.settings.SetBoolean("toolbar-visible", pd.toolbarVisibleCheck.Checked)
	pd.settings.SetBoolean("statusbar-visible", pd.statusbarVisibleCheck.Checked)
	pd.settings.SetBoolean("autohide-headerbar", pd.autohideHeaderCheck.Checked)
	
	// Export settings
	if pd.exportFormatSelect.Selected != "" {
		pd.settings.SetString("export-format", pd.exportFormatSelect.Selected)
	}
	pd.settings.SetString("export-directory", pd.exportDirEntry.Text)
	pd.settings.SetBoolean("include-title-page", pd.includeTitlePageCheck.Checked)
	if pd.pageSizeSelect.Selected != "" {
		pd.settings.SetString("page-size", pd.pageSizeSelect.Selected)
	}
	pd.settings.SetString("font-name", pd.exportFontEntry.Text)
	pd.settings.SetString("script-format", selectedScriptFormat(pd.scriptFormatSelect))
	if exportFontSize, err := strconv.Atoi(pd.exportFontSizeEntry.Text); err == nil {
		pd.settings.SetInt("font-size-export", exportFontSize)
	}
	
	// Fountain settings
	pd.settings.SetBoolean("fountain-auto-format", pd.fountainAutoFormatCheck.Checked)
	pd.settings.SetBoolean("fountain-scene-numbers", pd.fountainSceneNumbersCheck.Checked)
	pd.settings.SetBoolean("fountain-dual-dialogue", pd.fountainDualDialogueCheck.Checked)
	pd.settings.SetBoolean("fountain-title-page", pd.fountainTitlePageCheck.Checked)
	
	// Advanced settings
	pd.settings.SetBoolean("smooth-scrolling", pd.smoothScrollingCheck.Checked)
	pd.settings.SetBoolean("show-whitespace", pd.showWhitespaceCheck.Checked)
	pd.settings.SetBoolean("highlight-matching-brackets", pd.highlightMatchingCheck.Checked)
	pd.settings.SetBoolean("auto-close-brackets", pd.autoCloseBracketsCheck.Checked)
	
	// Save to disk
	pd.settings.Save()
}

func (pd *PreferencesDialog) resetToDefaults() {
	dialog.ShowConfirm(
		"Reset Settings",
		"Are you sure you want to reset all settings to their defaults? This cannot be undone.",
		func(confirmed bool) {
			if confirmed {
				pd.settings.Reset()
				pd.loadCurrentSettings()
			}
		},
		pd.app.windows[0].fyneWindow,
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
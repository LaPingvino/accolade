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

	"github.com/LaPingvino/lexington/rules"
)

// PreferencesDialog is the Preferences window. It shows only settings
// that do something: each one is read by the window, the editor or the
// Export dialog.
type PreferencesDialog struct {
	app *Application
	win fyne.Window // a window of its own, not an overlay on the editor
	// in the web version (one window): a dialog in the page instead
	dlg dialog.Dialog
	// host is where its messages show: its window, or the page's
	host     fyne.Window
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
	removeFormatButton    *widget.Button
	exportFormatSelect    *widget.Select
	exportDirEntry        *widget.Entry
	paperSelect           *widget.Select
	includeTitlePageCheck *widget.Check
	numberScenesCheck     *widget.Check
}

func NewPreferencesDialog(app *Application, parent fyne.Window) *PreferencesDialog {
	pd := &PreferencesDialog{
		app:      app,
		settings: GetSettings(),
	}

	pd.createWidgets()
	pd.loadCurrentSettings()
	content := pd.createContent()

	if inBrowser && parent != nil {
		pd.dlg = dialog.NewCustomWithoutButtons("Preferences", content, parent)
		pd.dlg.Resize(fyne.NewSize(640, 540))
		pd.host = parent
		return pd
	}
	pd.win = app.fyneApp.NewWindow("Preferences")
	pd.win.SetContent(content)
	pd.keys()
	pd.win.Resize(fyne.NewSize(640, 540))
	pd.win.CenterOnScreen()
	pd.host = pd.win
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
	pd.autoIndentCheck = widget.NewCheck("Format the line on Enter (capitals for scene headings and transitions)", nil)
	pd.autoCloseBracketsCheck = widget.NewCheck("Close brackets as you type them: ( and [", nil)
	pd.keysSelect = widget.NewSelect([]string{"standard", "vim", "helix"}, nil)

	pd.languageSelect = widget.NewSelect(scriptLanguages(), nil)
	pd.scriptFormatDesc = widget.NewLabel("")
	pd.scriptFormatDesc.Wrapping = fyne.TextWrapWord
	pd.scriptFormatSelect = newScriptFormatSelect(pd.scriptFormatDesc, func(string) { pd.showFormatButtons() })
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
		}, pd.host)
	})
	return container.NewVBox(
		widget.NewCard("Script Format", "How the editor and the exports lay the script out",
			container.NewVBox(pd.scriptFormatSelect, pd.scriptFormatDesc,
				container.NewHBox(widget.NewButton("Edit Format…", pd.editFormat),
					widget.NewButton("New Format from This…", pd.newFormat), pd.formatRemoveButton()),
				container.NewHBox(widget.NewButton("Edit Lexington Settings…", pd.editLexington))),
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

// formatRemoveButton removes one's own format, or one's changes to a
// built-in one.
func (pd *PreferencesDialog) formatRemoveButton() *widget.Button {
	pd.removeFormatButton = widget.NewButton("Remove", func() {
		key := selectedScriptFormat(pd.scriptFormatSelect)
		what := "Remove the format " + scriptFormat(key).Name + "?"
		if builtinFormat(key) {
			what = "Forget your changes to " + scriptFormat(key).Name + ", and use the built-in one?"
		}
		dialog.ShowConfirm("Remove", what, func(ok bool) {
			if !ok {
				return
			}
			if err := removeFormat(key); err != nil {
				dialog.ShowError(err, pd.host)
				return
			}
			if !builtinFormat(key) {
				key = rules.ScreenplayPreset
			}
			pd.formatsChanged(key)
		}, pd.host)
	})
	pd.showFormatButtons()
	return pd.removeFormatButton
}

// showFormatButtons: Remove for one's own formats and changed built-in ones.
func (pd *PreferencesDialog) showFormatButtons() {
	b := pd.removeFormatButton
	if b == nil {
		return
	}
	key := selectedScriptFormat(pd.scriptFormatSelect)
	b.SetText("Remove")
	if builtinFormat(key) {
		b.SetText("Use the Built-in")
	}
	if userFormat(key) {
		b.Enable()
	} else {
		b.Disable()
	}
}

// editFormat edits the chosen format's file (a built-in one: a changed
// copy of one's own).
func (pd *PreferencesDialog) editFormat() {
	key := selectedScriptFormat(pd.scriptFormatSelect)
	editTOML(pd.host, "Format: "+scriptFormat(key).Name, formatText(key),
		"Written as Lexington's presets are: a Name, the format it starts from (Base) and the [Elements.…] that differ, with Accolade's toolbar buttons ([[Accolade.Buttons]]), which Lexington skips. Saved as "+key+".toml in your formats folder.",
		nil, func(text string) error { return saveFormat(key, text) }, func() { pd.formatsChanged(key) })
}

// newFormat makes a format of one's own from the chosen one.
func (pd *PreferencesDialog) newFormat() {
	from := selectedScriptFormat(pd.scriptFormatSelect)
	name := widget.NewEntry()
	name.SetPlaceHolder("My " + scriptFormat(from).Name)
	dialog.ShowForm("New Format from "+scriptFormat(from).Name, "Make", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Name", name)}, func(ok bool) {
			if !ok {
				return
			}
			key, err := newFormat(strings.TrimSpace(name.Text), from)
			if err != nil {
				dialog.ShowError(err, pd.host)
				return
			}
			pd.formatsChanged(key)
			pd.editFormat()
		}, pd.host)
}

// formatsChanged shows the formats again, key chosen.
func (pd *PreferencesDialog) formatsChanged(key string) {
	pd.scriptFormatSelect.SetOptions(scriptFormatNames())
	selectScriptFormat(pd.scriptFormatSelect, key)
	pd.showFormatButtons()
	pd.app.scriptFormatsChanged()
}

// editLexington edits Lexington's settings: formats and languages of
// one's own show here and in the windows after saving.
func (pd *PreferencesDialog) editLexington() {
	editLexingtonSettings(pd.host, func() {
		format, lang := selectedScriptFormat(pd.scriptFormatSelect), pd.languageSelect.Selected
		pd.scriptFormatSelect.SetOptions(scriptFormatNames())
		selectScriptFormat(pd.scriptFormatSelect, format)
		pd.languageSelect.SetOptions(scriptLanguages())
		pd.languageSelect.SetSelected(lang)
		pd.app.scriptFormatsChanged()
	})
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
		pd.host,
	)
}

func (pd *PreferencesDialog) apply() {
	pd.saveSettings()
	pd.app.applySettingsToWindows()
	dialog.ShowInformation("Settings Applied", "Your preferences have been saved.", pd.host)
}

func (pd *PreferencesDialog) ok() {
	pd.saveSettings()
	pd.app.applySettingsToWindows()
	pd.close()
}

func (pd *PreferencesDialog) cancel() { pd.close() }

func (pd *PreferencesDialog) close() {
	if pd.dlg != nil {
		pd.dlg.Hide()
		return
	}
	pd.win.Close()
}

func (pd *PreferencesDialog) Show() {
	if pd.dlg != nil {
		pd.dlg.Show()
		return
	}
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

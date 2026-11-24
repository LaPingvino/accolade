package main

import (
	"fyne.io/fyne/v2/driver/desktop"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/lex"
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
	roleSelect            *widget.Select // Writer or Production (role.go)
	productionCard        *widget.Card   // the Script tab's production settings: Production's only
	themeSelect           *widget.Select
	variantSelect         *widget.Select // light or dark (theme-variant)
	themeModes            *widget.Label  // what the chosen theme has: light and dark, or one
	lookSelect            *widget.Select // the header's rows or the ribbon
	toolbarVisibleCheck   *widget.Check
	statusbarVisibleCheck *widget.Check
	quickRowCheck         *widget.Check // the ribbon's Quick row (quick.go)
	autoSaveCheck         *widget.Check
	autoSaveInterval      *widget.Entry

	// Editor
	fontSelect             *widget.Select // the fonts the ribbon offers (fontOptions)
	fontValue              string         // the font setting fontSelect shows
	fontSizeEntry          *widget.Entry
	lineNumbersCheck       *widget.Check
	autoIndentCheck        *widget.Check
	autoCloseBracketsCheck *widget.Check
	keysSelect             *widget.Select

	// Script and export
	languageSelect      *widget.Select
	scriptFormatSelect  *widget.Select
	scriptFormatDesc    *widget.Label
	removeFormatButton  *widget.Button
	exportFormatSelect  *widget.Select
	exportDirEntry      *widget.Entry
	libraryEntry        *widget.Entry // the main library for citations (citations.go)
	yourNameEntry       *widget.Entry
	pageSummary         *widget.Label // the page setup (Page Setup… sets it: pagesetup.go)
	sceneLettersSelect  *widget.Select
	revisionMarksSelect *widget.Select // the current revision's marks, or all issues'
	revisionHeaderEntry *widget.Entry  // a changed page's header
	revisionDateSelect  *widget.Select // its date's layout
	suggestCheck        *widget.Check
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
		pd.dlg.Resize(fitDialog(parent, 640, 540))
		pd.host = parent
		return pd
	}
	pd.win = app.fyneApp.NewWindow("Preferences")
	pd.win.SetOnClosed(func() {
		if app.prefs == pd {
			app.prefs = nil
		}
	})
	pd.win.SetContent(content)
	pd.keys()
	pd.win.Resize(fyne.NewSize(640, 540))
	pd.win.CenterOnScreen()
	pd.host = pd.win
	return pd
}

func (pd *PreferencesDialog) createWidgets() {
	pd.roleSelect = widget.NewSelect(roleLabels, func(string) { pd.showRole() })
	pd.themeSelect = widget.NewSelect(themeLabels(), func(string) { pd.showThemeModes() })
	pd.variantSelect = widget.NewSelect(choiceLabels(themeVariantChoices), nil)
	pd.themeModes = hint("")
	pd.lookSelect = widget.NewSelect(lookChoices, nil)
	pd.toolbarVisibleCheck = widget.NewCheck("Show toolbar", nil)
	pd.statusbarVisibleCheck = widget.NewCheck("Show status bar", nil)
	pd.quickRowCheck = widget.NewCheck("Quick row under the ribbon (the Fixed order)", nil)
	pd.autoSaveCheck = widget.NewCheck("Auto-save documents", nil)
	pd.autoSaveInterval = widget.NewEntry()
	pd.autoSaveInterval.SetPlaceHolder("30")

	pd.fontSelect = widget.NewSelect(nil, func(label string) {
		for _, c := range fontOptions(pd.fontValue) {
			if c[1] == label {
				pd.fontValue = c[0]
			}
		}
	})
	pd.fontSizeEntry = widget.NewEntry()
	pd.fontSizeEntry.SetPlaceHolder("12")
	pd.lineNumbersCheck = widget.NewCheck("Show line numbers", nil)
	pd.autoIndentCheck = widget.NewCheck("Format the line on Enter (capitals for scene headings and transitions)", nil)
	pd.autoCloseBracketsCheck = widget.NewCheck("Close brackets as you type them: ( and [", nil)
	pd.keysSelect = widget.NewSelect([]string{"standard", "vim", "helix"}, nil)

	pd.languageSelect = widget.NewSelect(nil, nil)
	pd.offerLanguages()
	pd.scriptFormatDesc = widget.NewLabel("")
	pd.scriptFormatDesc.Wrapping = fyne.TextWrapWord
	pd.scriptFormatSelect = newScriptFormatSelect(machineLex, pd.scriptFormatDesc, func(string) { pd.showFormatButtons() })
	fyne.SetAccessibleLabel(pd.scriptFormatSelect, "Script format")
	pd.exportFormatSelect = widget.NewSelect(exportFormats, nil)
	pd.yourNameEntry = widget.NewEntry()
	pd.yourNameEntry.SetPlaceHolder(os.Getenv("USER"))
	pd.exportDirEntry = widget.NewEntry()
	pd.exportDirEntry.SetPlaceHolder("Choose export directory...")
	pd.libraryEntry = widget.NewEntry()
	pd.libraryEntry.SetPlaceHolder("~/Zotero/library.json")
	pd.pageSummary = widget.NewLabel("")
	pd.pageSummary.Wrapping = fyne.TextWrapWord
	pd.sceneLettersSelect = widget.NewSelect(sceneLetterings, nil)
	pd.revisionMarksSelect = widget.NewSelect(revisionMarkChoices, nil)
	pd.revisionHeaderEntry = widget.NewEntry()
	pd.revisionHeaderEntry.SetPlaceHolder(lex.DefaultRevisionHeader)
	fyne.SetAccessibleLabel(pd.revisionHeaderEntry, "Revised page header")
	var dates []string
	for _, d := range revisionDates {
		dates = append(dates, d.label)
	}
	pd.revisionDateSelect = widget.NewSelect(dates, nil)
	pd.suggestCheck = widget.NewCheck("Suggest from the source while writing: lines, action, headings (Tab takes them)", nil)
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

// lookChoices are the header's looks, as Preferences names them.
var lookChoices = []string{"Menus", "Ribbon"}

func labelled(label string, o fyne.CanvasObject) fyne.CanvasObject {
	return labelledWith(label, o, nil)
}

// labelledWith is a field with its label before it and trailing (a button)
// after it; screen readers call the field by the label.
func labelledWith(label string, o, trailing fyne.CanvasObject) fyne.CanvasObject {
	fyne.SetAccessibleLabel(o, strings.TrimSuffix(strings.TrimSpace(label), ":"))
	return container.NewBorder(nil, nil, widget.NewLabel(label), trailing, o)
}

// hint is a quiet line of explanation under a setting.
func hint(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	l.Importance = widget.LowImportance
	return l
}

// quickRowRow is the Quick row's checkbox with its hint: the desktop's
// (a phone has the Quick tab; hidden there).
func (pd *PreferencesDialog) quickRowRow() fyne.CanvasObject {
	box := container.NewVBox(pd.quickRowCheck,
		hint("The Quick tab's commands as a row of small buttons under the ribbon, like Office's Quick Access toolbar, while the ribbon's lock is shut (Fixed). Also View > Quick Row, or a ribbon button's right-click menu."))
	if fyne.CurrentDevice().IsMobile() {
		box.Hide()
	}
	return box
}

func (pd *PreferencesDialog) createGeneralTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewCard("Working As", "", container.NewVBox(
			labelled("I am working as:", pd.roleSelect),
			hint("Writer puts production away: the Production menu, its pane and ribbon tab, and its settings here. A script locked or issued for production stays as it is, and prints as it is. View > Working As switches, for every script or only one."),
		)),
		widget.NewCard("Appearance", "", container.NewVBox(
			labelled("Theme:", pd.themeSelect),
			labelled("Light or dark:", pd.variantSelect),
			pd.themeModes,
			container.NewHBox(widget.NewButtonWithIcon("Customize Colours…", theme.ColorPaletteIcon(), pd.customizeTheme)),
			labelled("Look:", pd.lookSelect),
			hint("Menus: the menu bar and the toolbar's two rows. Ribbon: tabs of large buttons in their place, the menus under ☰ at the tab row's end (a Mac keeps its menu bar); F10, or Alt pressed alone, shows their keys."),
			pd.toolbarVisibleCheck,
			pd.statusbarVisibleCheck,
			pd.quickRowRow(),
		)),
		widget.NewCard("Saving", "", container.NewVBox(
			pd.autoSaveCheck,
			labelled("Auto-save interval (seconds):", pd.autoSaveInterval),
		)),
		widget.NewCard("Writing Together", "", container.NewVBox(
			labelled("Your name:", pd.yourNameEntry),
			hint("What the others see by your changes when you share a script."),
		)),
	)
}

func (pd *PreferencesDialog) createEditorTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewCard("Font", "", container.NewVBox(
			labelledWith("Font:", pd.fontSelect, widget.NewButton("Choose File…", func() {
				chooseFileIn(pd.host, fontExtensions, pd.showFont)
			})),
			labelled("Size:", pd.fontSizeEntry),
		)),
		widget.NewCard("Writing", "", container.NewVBox(
			pd.lineNumbersCheck,
			pd.autoIndentCheck,
			pd.autoCloseBracketsCheck,
			labelled("Keys (vim, helix: Escape for normal mode):", pd.keysSelect),
		)),
		pd.citationsCard(),
	)
}

// citationsCard is the main library a Markdown document cites from when
// it names none (citations.go); none in the browser, whose files have no
// path to keep.
func (pd *PreferencesDialog) citationsCard() fyne.CanvasObject {
	card := widget.NewCard("Citations", "Markdown: [@key] and @key cite from a library", container.NewVBox(
		labelledWith("Main library:", pd.libraryEntry, widget.NewButton("Choose File…", func() {
			chooseFileIn(pd.host, []string{".json", ".bib"}, func(p string) { pd.libraryEntry.SetText(p) })
		})),
		hint("CSL-JSON (Zotero's Better BibTeX: Better CSL JSON, kept up to date) or BibTeX/BibLaTeX. A document's front matter (bibliography: refs.bib) comes first, then a .bib or .json beside the document or in its bundle, then this. A re-exported file is read again."),
	))
	if inBrowser {
		card.Hide()
	}
	return card
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
				container.NewHBox(widget.NewButton("Edit Lexington Settings…", pd.editLexington), pd.addSettingsButton())),
		),
		widget.NewCard("Language", "Of a new script: its scene headings (INT./EXT., EN./EKST., BIN./BUI., ...). A file named like scene.eo.fountain says its own.",
			labelled("Language:", pd.languageSelect),
		),
		widget.NewCard("Export", "The Export dialog starts with these", container.NewVBox(
			labelled("Format:", pd.exportFormatSelect),
			labelledWith("Directory:", pd.exportDirEntry, browse),
		)),
		widget.NewCard("Page Setup", "How Export, Print and Preview PDF lay the page out", container.NewBorder(nil, nil, nil,
			container.NewVBox(widget.NewButtonWithIcon("Page Setup…", theme.DocumentIcon(), pd.openPageSetup)), pd.pageSummary)),
		widget.NewCard("Adapting", "A script made from a story beside it (View > Source)", container.NewVBox(
			pd.suggestCheck,
		)),
		pd.newProductionCard(container.NewVBox(
			labelled("A new scene's number:", pd.sceneLettersSelect),
			hint("A scene written between 10 and 11: a letter after the scene before (10A) or before the scene after (A11). For scripts locked from now on; a locked script keeps its own."),
			labelled("Revision marks since:", pd.revisionMarksSelect),
			hint("Once issued (Production > Issue Draft…): the changed lines, in the editor's margin and as * in the PDF. Since the last issue: the current revision's, in its colour; all issues: each line in the colour of the last set it changed in."),
			labelled("Revised page header:", pd.revisionHeaderEntry),
			labelled("Its date:", pd.revisionDateSelect),
			hint("{set} is the revision's colour, {date} its date: \"{set} Revision – {date}\" prints \"Blue Revision – 20/11/26\". \"-\" prints none."),
		)),
	)
}

// newProductionCard is the Production card, shown as the role chosen says.
func (pd *PreferencesDialog) newProductionCard(content fyne.CanvasObject) *widget.Card {
	pd.productionCard = widget.NewCard("Production", "Once the scene numbers are locked (Production > Lock Scene Numbers)", content)
	pd.showRole()
	return pd.productionCard
}

// showRole shows the Production card for Production, not for Writer.
func (pd *PreferencesDialog) showRole() {
	if pd.productionCard != nil {
		setVisible(pd.productionCard, pd.roleSelect.Selected != roleLabels[0])
	}
}

func (pd *PreferencesDialog) loadCurrentSettings() { pd.loadFrom(pd.settings) }

// loadFrom shows the settings of s in the window.
func (pd *PreferencesDialog) loadFrom(s *Settings) {
	pd.roleSelect.SetSelected(roleLabel(s.GetString(rolePref)))
	pd.themeSelect.SetSelected(themeLabel(s.GetString("theme")))
	pd.variantSelect.SetSelected(choiceLabel(themeVariantChoices, s.GetString(themeVariantPref)))
	pd.lookSelect.SetSelected(lookChoices[0])
	if s.GetString(lookPref) == "ribbon" {
		pd.lookSelect.SetSelected(lookChoices[1])
	}
	pd.toolbarVisibleCheck.SetChecked(s.GetBoolean("toolbar-visible"))
	pd.statusbarVisibleCheck.SetChecked(s.GetBoolean("statusbar-visible"))
	pd.quickRowCheck.SetChecked(s.GetBoolean(quickRowPref))
	pd.autoSaveCheck.SetChecked(s.GetBoolean("auto-save"))
	pd.autoSaveInterval.SetText(strconv.Itoa(s.GetInt("auto-save-interval")))

	pd.showFont(s.GetString("font-family"))
	pd.fontSizeEntry.SetText(strconv.Itoa(s.GetInt("font-size")))
	pd.lineNumbersCheck.SetChecked(s.GetBoolean("show-line-numbers"))
	pd.autoIndentCheck.SetChecked(s.GetBoolean("auto-indent"))
	pd.autoCloseBracketsCheck.SetChecked(s.GetBoolean("auto-close-brackets"))
	pd.keysSelect.SetSelected((&MainWindow{settings: s}).editorKeys())

	machineLex().selectFormat(pd.scriptFormatSelect, s.GetString("script-format"))
	selectOption(pd.languageSelect, s.GetString("script-language"), scriptLanguages())
	pd.exportFormatSelect.SetSelected(exportFormatLabel(s.GetString("export-format")))
	pd.exportDirEntry.SetText(s.GetString("export-directory"))
	pd.libraryEntry.SetText(s.GetString(citeLibraryPref))
	pd.yourNameEntry.SetText(s.GetString("your-name"))
	pd.showPageSetup()
	pd.suggestCheck.SetChecked(s.GetBoolean(suggestPref))
	pd.sceneLettersSelect.SetSelected(sceneLettering(lex.ParseNumberStyle(s.GetString(sceneNumberStylePref))))
	pd.revisionMarksSelect.SetSelected(revisionMarkChoices[0])
	if s.GetString(revisionMarksPref) == "all" {
		pd.revisionMarksSelect.SetSelected(revisionMarkChoices[1])
	}
	pd.revisionHeaderEntry.SetText(s.GetString(revisionHeaderPref))
	pd.revisionDateSelect.SetSelected(revisionDates[0].label)
	for _, d := range revisionDates {
		if d.layout == s.GetString(revisionDatePref) {
			pd.revisionDateSelect.SetSelected(d.label)
		}
	}
}

func (pd *PreferencesDialog) saveSettings() {
	s := pd.settings
	if pd.roleSelect.Selected != "" {
		s.SetString(rolePref, roleOfLabel(pd.roleSelect.Selected))
	}
	if pd.variantSelect.Selected != "" {
		s.SetString(themeVariantPref, choiceID(themeVariantChoices, pd.variantSelect.Selected))
	}
	if pd.themeSelect.Selected != "" {
		s.SetString("theme", themeIDOf(pd.themeSelect.Selected))
		pd.app.setColorScheme(themeIDOf(pd.themeSelect.Selected))
	}
	s.SetString(lookPref, "menus")
	if pd.lookSelect.Selected == lookChoices[1] {
		s.SetString(lookPref, "ribbon")
	}
	s.SetBoolean("toolbar-visible", pd.toolbarVisibleCheck.Checked)
	s.SetBoolean("statusbar-visible", pd.statusbarVisibleCheck.Checked)
	s.SetBoolean(quickRowPref, pd.quickRowCheck.Checked)
	s.SetBoolean("auto-save", pd.autoSaveCheck.Checked)
	if interval, err := strconv.Atoi(pd.autoSaveInterval.Text); err == nil && interval > 0 {
		s.SetInt("auto-save-interval", interval)
	}

	s.SetString("font-family", pd.fontValue)
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

	s.SetString("script-format", machineLex().selectedFormat(pd.scriptFormatSelect))
	if l := pd.languageSelect.Selected; l != "" {
		if l != s.GetString("script-language") && pd.app != nil {
			for _, w := range pd.app.windows {
				w.newLanguage = l // New's next, as New in: (window.go)
			}
		}
		s.SetString("script-language", l)
	}
	s.SetString("export-format", exportFormatLabel(pd.exportFormatSelect.Selected))
	s.SetString("export-directory", pd.exportDirEntry.Text)
	s.SetString(citeLibraryPref, strings.TrimSpace(pd.libraryEntry.Text))
	s.SetString("your-name", strings.TrimSpace(pd.yourNameEntry.Text))
	s.SetBoolean(suggestPref, pd.suggestCheck.Checked)
	s.SetString(sceneNumberStylePref, letteringStyle(pd.sceneLettersSelect.Selected).String())
	marks := "last"
	if pd.revisionMarksSelect.Selected == revisionMarkChoices[1] {
		marks = "all"
	}
	s.SetString(revisionMarksPref, marks)
	s.SetString(revisionHeaderPref, strings.TrimSpace(pd.revisionHeaderEntry.Text))
	for _, d := range revisionDates {
		if d.label == pd.revisionDateSelect.Selected {
			s.SetString(revisionDatePref, d.layout)
		}
	}

	s.Save()
}

// formatRemoveButton removes one's own format, or one's changes to a
// built-in one.
func (pd *PreferencesDialog) formatRemoveButton() *widget.Button {
	pd.removeFormatButton = widget.NewButton("Remove", func() {
		key := machineLex().selectedFormat(pd.scriptFormatSelect)
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
	key := machineLex().selectedFormat(pd.scriptFormatSelect)
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

// editFormat edits the chosen format's file in the format editor (a
// built-in one: a changed copy goes into one's own folder).
func (pd *PreferencesDialog) editFormat() {
	key := machineLex().selectedFormat(pd.scriptFormatSelect)
	path := formatFile(key)
	if path == "" || !userFormat(key) {
		path = filepath.Join(userFormatDir(), key+".toml")
	}
	openTOMLFile(pd.app, pd.host, path, formatText(key))
}

// newFormat makes a format of one's own from the chosen one.
func (pd *PreferencesDialog) newFormat() {
	from := machineLex().selectedFormat(pd.scriptFormatSelect)
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
	machineLex().selectFormat(pd.scriptFormatSelect, key)
	pd.showFormatButtons()
	pd.app.scriptFormatsChanged()
}

// showFont shows the font setting f in the font list: a built-in choice
// (anything unknown is the screenplay font, as newEditorTheme reads it) or
// a font file by its name.
func (pd *PreferencesDialog) showFont(f string) {
	f = strings.TrimSpace(f)
	if !isFontFile(f) && !strings.EqualFold(f, "system") {
		f = fontChoices[0][0]
	} else if strings.EqualFold(f, "system") {
		f = "system"
	}
	pd.fontValue = f
	var labels []string
	selected := ""
	for _, c := range fontOptions(f) {
		labels = append(labels, c[1])
		if c[0] == f {
			selected = c[1]
		}
	}
	pd.fontSelect.SetOptions(labels)
	pd.fontSelect.SetSelected(selected)
}

// addSettingsButton is Add to My Settings for an open script that brought
// settings of its own (an .accolade's, lexington_carried.go); hidden if
// none did.
func (pd *PreferencesDialog) addSettingsButton() *widget.Button {
	var from *MainWindow
	if pd.app != nil {
		for _, w := range pd.app.windows {
			if !w.carried.empty() {
				from = w
				break
			}
		}
	}
	b := widget.NewButton("Add the Script's Settings…", func() {
		if from != nil {
			from.addToMySettings()
		}
	})
	if from == nil {
		b.Hide()
	}
	return b
}

// editLexington edits Lexington's settings: formats and languages of
// one's own show here and in the windows after saving.
func (pd *PreferencesDialog) editLexington() {
	openTOMLFile(pd.app, pd.host, lexingtonConfig(), lexingtonSettingsText())
}

// settingsFilesChanged shows the formats and languages again after the
// settings or a format file was saved.
func (pd *PreferencesDialog) settingsFilesChanged() {
	format, lang := machineLex().selectedFormat(pd.scriptFormatSelect), pd.languageSelect.Selected
	pd.scriptFormatSelect.SetOptions(scriptFormatNames())
	machineLex().selectFormat(pd.scriptFormatSelect, format)
	pd.offerLanguages()
	selectOption(pd.languageSelect, lang, scriptLanguages())
	pd.showFormatButtons()
}

// pageWindow is the window whose page setup Preferences shows: the
// first (nil without one).
func (pd *PreferencesDialog) pageWindow() *MainWindow {
	if pd.app == nil || len(pd.app.windows) == 0 {
		return nil
	}
	return pd.app.windows[0]
}

// showPageSetup shows the page setup's line.
func (pd *PreferencesDialog) showPageSetup() {
	w := pd.pageWindow()
	if w == nil {
		pd.pageSummary.SetText("")
		return
	}
	if w.markdown {
		pd.pageSummary.SetText(w.markdownPageSummary())
		return
	}
	pd.pageSummary.SetText(w.pageSummary(w.pageSetup(), exportFormatByLabel(pd.settings.GetString("export-format")).id))
}

// openPageSetup opens Page Setup, in the script's window.
func (pd *PreferencesDialog) openPageSetup() {
	if w := pd.pageWindow(); w != nil {
		w.showPageSetup("", pd.showPageSetup)
		w.fyneWindow.RequestFocus()
	}
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
				if pd.roleSelect.Selected == roleLabels[0] {
					// production's settings are out of sight: kept as
					// they are, as a writer's locked script prints them
					for _, k := range productionPrefs {
						defaults.data[k] = pd.settings.get(k)
					}
				}
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
	if pd.app != nil && pd.app.prefs == pd {
		pd.app.prefs = nil
	}
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

// sceneLetterings are the Preferences' choices for a new scene's number
// in a locked script (lex.NumberStyle), named by what they make.
var sceneLetterings = []string{
	"10A, 10B (after 10)",
	"A11, B11 (before 11)",
}

func sceneLettering(s lex.NumberStyle) string {
	if s == lex.NumberBefore {
		return sceneLetterings[1]
	}
	return sceneLetterings[0]
}

func letteringStyle(choice string) lex.NumberStyle {
	if choice == sceneLetterings[1] {
		return lex.NumberBefore
	}
	return lex.NumberAfter
}

// offerLanguages gives the language for new scripts its options: the
// languages, their variants behind More variants… (language_variants.go).
func (pd *PreferencesDialog) offerLanguages() {
	all := scriptLanguages()
	shown, _ := shownLanguages(all, "")
	offerVariants(pd.languageSelect, shown, all)
}

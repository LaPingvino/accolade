package main

import (
	"cmp"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/LaPingvino/lexington/lex"
)

// The preferences changed while working, as registry commands: the
// ribbon's buttons and the palette's entries read and write the same
// settings as the Preferences window, which stays their full list. A
// change is saved and applied to every window at once, and an open
// Preferences window shows it (followSetting), as a change made there
// shows here once applied (refreshMenu and the ribbon follow the
// settings).

// The font sizes Larger Text and Smaller Text step between.
const (
	minFontSize = 8
	maxFontSize = 32
)

// themeChoices are the themes, as Preferences names them, and their labels.
var themeChoices = [][2]string{{"system", "System"}, {"light", "Light"}, {"dark", "Dark"}, {"sepia", "Sepia"}}

// changeSetting runs set on the settings, saves them, and applies them to
// every window (and to an open Preferences window: key).
func (w *MainWindow) changeSetting(key string, set func(s *Settings)) {
	before := w.printedLayout()
	defer func() { // a preview's pages follow a change in how the page prints (the ribbon's too)
		if !reflect.DeepEqual(before, w.printedLayout()) {
			w.pagesAgain()
		}
	}()
	set(w.settings)
	_ = w.settings.Save()
	if w.app != nil {
		w.app.applySettingsToWindows()
	}
	w.followInPreferences(key)
	if w.app == nil || !slices.Contains(w.app.windows, w) { // a window of its own (a test's)
		w.applyEditorSettings()
		w.applyFontSettings()
	}
	w.refreshMenu()
}

// toggleSetting is a command turning a boolean setting on and off.
func toggleSetting(id, label, key string, ic fyne.Resource, after func(w *MainWindow)) *command {
	return &command{id: id, label: label, icon: res(ic),
		checked: func(w *MainWindow) bool { return w.settings != nil && w.settings.GetBoolean(key) },
		run: func(w *MainWindow) {
			w.changeSetting(key, func(s *Settings) { s.SetBoolean(key, !s.GetBoolean(key)) })
			if after != nil {
				after(w)
			}
		}}
}

// choiceSetting is a command whose submenu chooses a string setting's
// value: choices are value and label.
func choiceSetting(id, label, key string, ic fyne.Resource, choices [][2]string, after func(w *MainWindow, value string)) *command {
	return &command{id: id, label: label, icon: res(ic),
		submenu: func(w *MainWindow) *fyne.Menu {
			var items []*fyne.MenuItem
			for _, c := range choices {
				value := c[0]
				it := fyne.NewMenuItem(c[1], func() {
					w.changeSetting(key, func(s *Settings) { s.SetString(key, value) })
					if after != nil {
						after(w, value)
					}
				})
				it.Checked = w.settings != nil && w.settings.GetString(key) == value
				items = append(items, it)
			}
			return fyne.NewMenu(label, items...)
		}}
}

// textSize is a command making the editor's text a point larger or
// smaller.
func textSize(id, label string, step int, ic fyne.Resource) *command {
	return &command{id: id, label: label, icon: res(ic), editing: false,
		enabled: func(e cmdEnv) bool {
			if step > 0 {
				return e.fontSize < maxFontSize
			}
			return e.fontSize > minFontSize
		},
		run: func(w *MainWindow) {
			w.changeSetting("font-size", func(s *Settings) {
				s.SetInt("font-size", min(max(s.GetInt("font-size")+step, minFontSize), maxFontSize))
			})
		}}
}

// settingCommands are the preferences changed while working.
func settingCommands() []*command {
	lookRibbon := toggleSetting("set.ribbon", "Ribbon", lookPref, ribbonIcon, nil)
	lookRibbon.checked = func(w *MainWindow) bool { return w.ribbonOn() }
	lookRibbon.run = func(w *MainWindow) {
		w.changeSetting(lookPref, func(s *Settings) {
			if s.GetString(lookPref) == "ribbon" {
				s.SetString(lookPref, "menus")
			} else {
				s.SetString(lookPref, "ribbon")
			}
		})
	}
	marks := func(w *MainWindow, _ string) {
		for _, x := range w.windows() {
			x.revisions.key = "" // read again with the new setting
			x.scheduleRevisionMarks()
		}
	}
	lettering := [][2]string{{lex.NumberAfter.String(), sceneLetterings[0]}, {lex.NumberBefore.String(), sceneLetterings[1]}}
	cs := []*command{
		choiceSetting("set.theme", "Theme", "theme", themeIcon, themeChoices, func(w *MainWindow, v string) {
			if w.app != nil {
				w.app.setColorScheme(v)
			}
		}),
		textSize("set.textLarger", "Larger Text", 1, textLargerIcon),
		textSize("set.textSmaller", "Smaller Text", -1, textSmallerIcon),
		toggleSetting("set.lineNumbers", "Line Numbers", "show-line-numbers", lineNumbersIcon, nil),
		toggleSetting("set.statusBar", "Status Bar", "statusbar-visible", statusBarIcon, nil),
		lookRibbon,
		toggleSetting("set.suggest", "Suggest from the Source While Writing", suggestPref, suggestWhileIcon, nil),
		choiceSetting("set.revisionMarks", "Revision Marks Since", revisionMarksPref, marksIcon,
			[][2]string{{"last", revisionMarkChoices[0]}, {"all", revisionMarkChoices[1]}}, marks),
		choiceSetting("set.lettering", "A New Scene's Number", sceneNumberStylePref, letteringIcon, lettering, nil),
		toggleSetting("set.continueds", "CONTINUED Where a Scene Breaks", continuedsPref, continuedIcon, nil),
	}
	for _, c := range cs[6:8] { // the source's
		c.when = screenplayOnly
	}
	for _, c := range cs[8:] { // production's
		c.when, c.role = screenplayOnly, productionRole
	}
	numbers := toggleSetting("set.sceneNumbers", "Number the Scenes in Exports", "fountain-scene-numbers", sceneNumbersIcon, nil)
	numbers.when, numbers.role = screenplayOnly, productionRole
	toolbar := toggleSetting("set.toolbar", "Toolbar", "toolbar-visible", toolbarIcon, func(w *MainWindow) {
		if !w.settings.GetBoolean("toolbar-visible") {
			w.setStatus("Toolbar hidden: View > Toolbar shows it again")
		}
	})
	return append(cs, editorKeysCommand(), fontCommand(), newLanguageCommand(), defaultLanguageCommand(), toolbar, numbers, draftsSideBySideCommand())
}

// editorKeysChoices are the editor's keys, as Preferences names them, and
// their labels.
var editorKeysChoices = [][2]string{{"standard", "Standard"}, {"vim", "Vim"}, {"helix", "Helix"}}

// editorKeysCommand chooses the editor's keys: standard, vim or helix
// (switching while working is safe: applyEditorSettings starts their
// state again and says which).
func editorKeysCommand() *command {
	return &command{id: "set.editorKeys", label: "Editor Keys", icon: res(keysIcon),
		submenu: func(w *MainWindow) *fyne.Menu {
			var items []*fyne.MenuItem
			for _, c := range editorKeysChoices {
				value := c[0]
				it := fyne.NewMenuItem(c[1], func() {
					w.changeSetting("editor-keys", func(s *Settings) {
						s.SetString("editor-keys", value)
						s.SetBoolean("vim-mode", false) // the older setting
					})
				})
				it.Checked = w.editorKeys() == value
				items = append(items, it)
			}
			return fyne.NewMenu("Editor Keys", items...)
		}}
}

// editorKeysStatus is the status line saying which keys the editor has.
func editorKeysStatus(keys string) string {
	for _, c := range editorKeysChoices {
		if c[0] == keys && keys != "standard" {
			return "Editor keys: " + c[1] + ", in normal mode (i types, Escape comes back)"
		}
	}
	return "Editor keys: Standard"
}

// fontChoices are the editor's fonts: the screenplay font, the system's.
var fontChoices = [][2]string{{"monospace", "Courier Badi (Screenplay)"}, {"system", "System"}}

// fontCommand chooses the editor's font, beside the text size; a chosen
// font file shows as itself, and Font File… picks one.
func fontCommand() *command {
	return &command{id: "set.font", label: "Font", icon: res(fontIcon),
		submenu: func(w *MainWindow) *fyne.Menu {
			now := strings.TrimSpace(w.settings.GetString("font-family"))
			choices := fontOptions(now)
			var items []*fyne.MenuItem
			for _, c := range choices {
				value := c[0]
				it := fyne.NewMenuItem(c[1], func() {
					w.changeSetting("font-family", func(s *Settings) { s.SetString("font-family", value) })
				})
				// anything else is the screenplay font (newEditorTheme)
				it.Checked = strings.EqualFold(now, value) || value == "monospace" && !strings.EqualFold(now, "system") && !isFontFile(now)
				items = append(items, it)
			}
			items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Font File…", func() {
				chooseFileIn(w.fyneWindow, fontExtensions, func(p string) {
					w.changeSetting("font-family", func(s *Settings) { s.SetString("font-family", p) })
				})
			}))
			return fyne.NewMenu("Font", items...)
		}}
}

// fontExtensions are the font files the editor reads.
var fontExtensions = []string{".ttf", ".otf"}

// fontOptions are the editor's fonts as Preferences and the ribbon offer
// them: the built-in choices, and now's file when it is one.
func fontOptions(now string) [][2]string {
	choices := fontChoices
	if isFontFile(now) {
		choices = append(slices.Clone(choices), [2]string{now, filepath.Base(now)})
	}
	return choices
}

// isFontFile reports whether the font setting f is a font file's path.
func isFontFile(f string) bool {
	f = strings.ToLower(f)
	return strings.HasSuffix(f, ".ttf") || strings.HasSuffix(f, ".otf")
}

// newScriptLanguage is the language new scripts are in (Preferences >
// Script: Language).
func (w *MainWindow) newScriptLanguage() string {
	if w.settings == nil {
		return ""
	}
	return w.settings.GetString("script-language")
}

// newFileLanguage is the language New starts a script in here: the one
// picked last (New in:, else the script's), as the ribbon shows it.
func (w *MainWindow) newFileLanguage() string {
	return cmp.Or(w.newLanguage, w.language, w.newScriptLanguage())
}

// newLanguageCommand chooses the language of new scripts, on the File
// tab's New group as "New in: English".
func newLanguageCommand() *command {
	return &command{id: "set.newLanguage", label: "Language for New Scripts", icon: res(languageIcon),
		ribbonLabelFor: func(w *MainWindow) string {
			name := []rune(languageName(w.newFileLanguage()))
			if len(name) > 14 { // a language of one's own with a long name: the File tab still fits
				name = append(name[:13], '…')
			}
			return "New in: " + string(name)
		},
		submenu: func(w *MainWindow) *fyne.Menu {
			// a language picked is New's next here, only (window.go);
			// the default changes with the toggle below it
			item := func(l string) *fyne.MenuItem {
				it := fyne.NewMenuItem(defaultLanguageLabel(w, l), func() {
					w.newLanguage = l
					w.ribbonRefresh()
					w.refreshMenu()
				})
				it.Checked = w.newFileLanguage() == l
				return it
			}
			shown, hidden := shownLanguages(scriptLanguages(), w.newFileLanguage())
			var items []*fyne.MenuItem
			for _, l := range shown {
				items = append(items, item(l))
			}
			if len(hidden) > 0 { // hi-Latn, hi-Deva (language_variants.go)
				var more []*fyne.MenuItem
				for _, l := range hidden {
					more = append(more, item(l))
				}
				it := fyne.NewMenuItem(moreVariants, nil)
				it.ChildMenu = fyne.NewMenu(moreVariants, more...)
				items = append(items, fyne.NewMenuItemSeparator(), it)
			}
			lang := w.newFileLanguage()
			def := fyne.NewMenuItem(useByDefault, func() { w.toggleDefaultLanguage(lang) })
			def.Checked = lang == w.preferredLanguage()
			items = append(items, fyne.NewMenuItemSeparator(), def)
			return fyne.NewMenu("Language for New Scripts", items...)
		}}
}

// useByDefault is the toggle making a language picked the preferences'
// (Preferences > Script: Language), next to the pick: New in:'s, Script
// Language's.
const useByDefault = "Use for New Scripts by Default"

// defaultLanguageLabel is language l's label in a list to pick from,
// saying when it is the preferences' language.
func defaultLanguageLabel(w *MainWindow, l string) string {
	if l == w.preferredLanguage() {
		return languageLabel(l) + " (default)"
	}
	return languageLabel(l)
}

// toggleDefaultLanguage makes lang the preferences' language for new
// scripts; if it is already, the default goes back to English.
func (w *MainWindow) toggleDefaultLanguage(lang string) {
	if lang == w.preferredLanguage() {
		lang = "en"
	}
	w.setDefaultLanguage(lang)
}

// setDefaultLanguage makes lang the preferences' language for new scripts.
func (w *MainWindow) setDefaultLanguage(lang string) {
	if lang == "" || lang == w.preferredLanguage() {
		return
	}
	w.changeSetting("script-language", func(s *Settings) { s.SetString("script-language", lang) })
	w.setStatus("New scripts are in " + languageLabel(lang) + " by default")
}

// defaultLanguageCommand makes the script's language the default for new
// scripts (or English again, when it is), for the palette and menus.
func defaultLanguageCommand() *command {
	return &command{id: "set.defaultLanguage", label: "Script's Language for New Scripts by Default", icon: res(languageIcon),
		when:    screenplayOnly,
		checked: func(w *MainWindow) bool { return w.language == w.preferredLanguage() },
		run:     func(w *MainWindow) { w.toggleDefaultLanguage(w.language) }}
}

// draftsSidePref is Compare's side by side, remembered.
const draftsSidePref = "drafts-side-by-side"

// draftsSideBySide reports whether Compare shows the drafts side by side
// (the default) or one under the other.
func (w *MainWindow) draftsSideBySide() bool {
	return w.settings != nil && (!w.settings.HasKey(draftsSidePref) || w.settings.GetBoolean(draftsSidePref))
}

// draftsSideBySideCommand is Compare's Side by side, on Collaboration >
// Drafts as in the Compare window.
func draftsSideBySideCommand() *command {
	return &command{id: "set.draftsSide", label: "Compare Drafts Side by Side", icon: res(sideBySideIcon),
		checked: (*MainWindow).draftsSideBySide,
		run: func(w *MainWindow) {
			on := !w.draftsSideBySide()
			w.changeSetting(draftsSidePref, func(s *Settings) { s.SetBoolean(draftsSidePref, on) })
			if cv := w.compare; cv != nil { // the Compare tab open: it follows
				cv.sideBySide.SetChecked(on)
				cv.show()
			}
		}}
}

// followInPreferences shows setting key's new value in an open
// Preferences window.
func (w *MainWindow) followInPreferences(key string) {
	if w.app != nil && w.app.prefs != nil {
		w.app.prefs.followSetting(key)
	}
}

// windows is every window of w's application (w alone without one).
func (w *MainWindow) windows() []*MainWindow {
	if w.app == nil {
		return []*MainWindow{w}
	}
	return w.app.windows
}

// followSetting shows the setting key as it is now (a change made on the
// ribbon or in the palette), leaving the other fields as they are.
func (pd *PreferencesDialog) followSetting(key string) {
	s := pd.settings
	switch key {
	case rolePref:
		pd.roleSelect.SetSelected(roleLabel(s.GetString(rolePref)))
	case "script-format":
		machineLex().selectFormat(pd.scriptFormatSelect, s.GetString("script-format"))
	case "theme":
		pd.themeSelect.SetSelected(s.GetString("theme"))
	case lookPref:
		pd.lookSelect.SetSelected(lookChoices[0])
		if s.GetString(lookPref) == "ribbon" {
			pd.lookSelect.SetSelected(lookChoices[1])
		}
	case "font-size":
		pd.fontSizeEntry.SetText(strconv.Itoa(s.GetInt("font-size")))
	case "show-line-numbers":
		pd.lineNumbersCheck.SetChecked(s.GetBoolean(key))
	case "statusbar-visible":
		pd.statusbarVisibleCheck.SetChecked(s.GetBoolean(key))
	case quickRowPref:
		pd.quickRowCheck.SetChecked(s.GetBoolean(key))
	case suggestPref:
		pd.suggestCheck.SetChecked(s.GetBoolean(key))
	case revisionMarksPref:
		pd.revisionMarksSelect.SetSelected(revisionMarkChoices[0])
		if s.GetString(key) == "all" {
			pd.revisionMarksSelect.SetSelected(revisionMarkChoices[1])
		}
	case sceneNumberStylePref:
		pd.sceneLettersSelect.SetSelected(sceneLettering(lex.ParseNumberStyle(s.GetString(key))))
	case continuedsPref, "fountain-scene-numbers", pageSetupPref:
		pd.showPageSetup()
	case "editor-keys":
		pd.keysSelect.SetSelected((&MainWindow{settings: s}).editorKeys())
	case "font-family":
		pd.showFont(s.GetString(key))
	case "script-language":
		pd.languageSelect.SetSelected(s.GetString(key))
	case "toolbar-visible":
		pd.toolbarVisibleCheck.SetChecked(s.GetBoolean(key))
	}
}

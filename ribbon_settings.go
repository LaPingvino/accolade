package main

import (
	"slices"
	"strconv"

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
	set(w.settings)
	_ = w.settings.Save()
	if w.app != nil {
		w.app.applySettingsToWindows()
		if w.app.prefs != nil {
			w.app.prefs.followSetting(key)
		}
	}
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
		toggleSetting("set.nextInserts", "Next Source Passage Starts a Scene", "next-passage-inserts", nextSceneOptIcon, nil),
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
	return cs
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
	case suggestPref:
		pd.suggestCheck.SetChecked(s.GetBoolean(key))
	case "next-passage-inserts":
		pd.nextInsertsCheck.SetChecked(s.GetBoolean(key))
	case revisionMarksPref:
		pd.revisionMarksSelect.SetSelected(revisionMarkChoices[0])
		if s.GetString(key) == "all" {
			pd.revisionMarksSelect.SetSelected(revisionMarkChoices[1])
		}
	case sceneNumberStylePref:
		pd.sceneLettersSelect.SetSelected(sceneLettering(lex.ParseNumberStyle(s.GetString(key))))
	case continuedsPref:
		pd.continuedsCheck.SetChecked(s.GetBoolean(key))
	}
}

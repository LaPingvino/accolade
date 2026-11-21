package main

import (
	"github.com/LaPingvino/lexington/lex"
	"strconv"
	"strings"
	"testing"
)

// pressMenuItem runs the item labelled label of command id's submenu.
func pressMenuItem(t *testing.T, w *MainWindow, id, label string) {
	t.Helper()
	for _, it := range cmd(id).submenu(w).Items {
		if strings.HasPrefix(it.Label, label) {
			it.Action()
			return
		}
	}
	t.Fatalf("%s has no %q", id, label)
}

// The settings on the ribbon write what Preferences writes, apply at
// once, show checked as they are, and an open Preferences window follows
// them; Preferences' changes show on the ribbon in turn.
func TestRibbonSettings(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	s := w.settings
	t.Cleanup(func() {
		s.SetBoolean("show-line-numbers", false)
		s.SetBoolean("statusbar-visible", true)
		s.SetInt("font-size", 12)
		s.SetString("theme", "system")
		s.SetString(revisionMarksPref, "last")
		s.SetString(sceneNumberStylePref, "10A")
		s.SetBoolean(continuedsPref, false)
		s.SetBoolean(suggestPref, true)
	})
	pd := NewPreferencesDialog(w.app, w.fyneWindow)
	w.app.prefs = pd
	t.Cleanup(func() { w.app.prefs = nil; pd.cancel() })

	r.selectTab("View")
	lines := buttonOf(r, "set.lineNumbers")
	if lines == nil || lines.checked {
		t.Fatal("no Line Numbers button, or checked with them off")
	}
	lines.press()
	if !s.GetBoolean("show-line-numbers") || !lines.checked || !pd.lineNumbersCheck.Checked {
		t.Errorf("Line Numbers: setting %v, button %v, Preferences %v", s.GetBoolean("show-line-numbers"), lines.checked, pd.lineNumbersCheck.Checked)
	}
	buttonOf(r, "set.statusBar").press()
	if s.GetBoolean("statusbar-visible") || w.statusBar.Visible() || pd.statusbarVisibleCheck.Checked {
		t.Error("Status Bar did not hide the status bar everywhere")
	}

	size := s.GetInt("font-size")
	buttonOf(r, "set.textLarger").press()
	if s.GetInt("font-size") != size+1 || pd.fontSizeEntry.Text != strconv.Itoa(size+1) {
		t.Errorf("Larger Text: %d, Preferences %q", s.GetInt("font-size"), pd.fontSizeEntry.Text)
	}
	buttonOf(r, "set.textSmaller").press()
	s.SetInt("font-size", maxFontSize)
	r.refresh()
	if !buttonOf(r, "set.textLarger").Disabled() {
		t.Error("Larger Text enabled at the largest size")
	}
	s.SetInt("font-size", size)

	pressMenuItem(t, w, "set.theme", "Dark")
	if s.GetString("theme") != "dark" || pd.themeSelect.Selected != "dark" {
		t.Errorf("Theme: %q, Preferences %q", s.GetString("theme"), pd.themeSelect.Selected)
	}

	r.selectTab("Production")
	pressMenuItem(t, w, "set.revisionMarks", "All")
	if s.GetString(revisionMarksPref) != "all" || pd.revisionMarksSelect.Selected != revisionMarkChoices[1] {
		t.Error("Revision Marks: not all issues'")
	}
	pressMenuItem(t, w, "set.lettering", "A11")
	if w.sceneNumberStyle() != lex.NumberBefore || pd.sceneLettersSelect.Selected != sceneLetterings[1] {
		t.Error("Scene Lettering: not A11")
	}
	cont := buttonOf(r, "set.continueds")
	cont.press()
	if !s.GetBoolean(continuedsPref) || !cont.checked || !pd.continuedsCheck.Checked {
		t.Error("CONTINUEDs")
	}

	// Preferences' own change, applied there, shows on the ribbon
	r.selectTab("View")
	pd.loadCurrentSettings()
	pd.lineNumbersCheck.SetChecked(false)
	pd.saveSettings()
	w.app.applySettingsToWindows()
	w.applyEditorSettings()
	if buttonOf(r, "set.lineNumbers").checked {
		t.Error("Preferences turned line numbers off; the ribbon still shows them on")
	}

	// Ribbon off: the menus, and back
	buttonOf(r, "set.ribbon").press()
	if w.ribbonOn() || pd.lookSelect.Selected != lookChoices[0] {
		t.Error("Ribbon did not switch to the menus")
	}
	w.do(cmd("set.ribbon"))()
	if !w.ribbonShown() {
		t.Error("the ribbon did not come back")
	}

	// Writer: the production settings are away from the ribbon, and the
	// palette still finds them
	w.workAs(roleWriter)
	t.Cleanup(func() { w.workAs(roleProduction) })
	if buttonOf(r, "set.continueds") != nil {
		t.Error("a production setting on a writer's ribbon")
	}
	if !cmd("set.continueds").appliesForRole(w.cmdEnv()) {
		t.Error("the palette would not find CONTINUEDs")
	}
}

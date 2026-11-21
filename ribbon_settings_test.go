package main

import (
	"fyne.io/fyne/v2"
	"github.com/LaPingvino/lexington/lex"
	"slices"
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
	// as in the app: the window among the application's, with another
	other := newTestWindow(t, "")
	other.app = w.app
	w.app.windows = append(w.app.windows, w, other)
	t.Cleanup(func() { w.app.windows = nil })
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
	if other.statusBar.Visible() {
		t.Error("Status Bar did not reach the other window")
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
	if !s.GetBoolean(continuedsPref) || !cont.checked || !strings.Contains(pd.pageSummary.Text, "CONTINUEDs") {
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

	// the palette has each, a choice's values one by one
	labels := map[string]paletteEntry{}
	for _, e := range w.paletteEntries(w.cmdEnv()) {
		labels[e.label] = e
	}
	for _, l := range []string{"Line Numbers", "Status Bar", "Ribbon", "Larger Text", "Smaller Text", "Theme > Light",
		"Suggest from the Source While Writing", "Next Source Passage Starts a Scene", "CONTINUED Where a Scene Breaks"} {
		if _, ok := labels[l]; !ok {
			t.Errorf("the palette has no %q", l)
		}
	}
	labels["Theme > Light"].action()
	if s.GetString("theme") != "light" {
		t.Error("the palette's Theme > Light")
	}

	// Writer: the production settings are away from the ribbon, and the
	// palette still finds them
	w.workAs(roleWriter)
	t.Cleanup(func() { w.workAs(roleProduction) })
	if buttonOf(r, "set.continueds") != nil {
		t.Error("a production setting on a writer's ribbon")
	}
	found := false
	for _, e := range w.paletteEntries(w.cmdEnv()) {
		found = found || strings.HasPrefix(e.label, "CONTINUED Where a Scene Breaks") && e.away
	}
	if !found {
		t.Error("a writer's palette does not find CONTINUEDs")
	}
}

// A tooltip goes when its button is pressed by a key, or the groups are
// made again under it.
func TestRibbonTooltipGoes(t *testing.T) {
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	r.selectTab("View")
	b := buttonOf(r, "view.outline")
	w.fyneWindow.Canvas().Focus(b) // key tips off: the tip at once
	if w.tooltips.Text() == "" {
		t.Fatal("no tip on focus")
	}
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if w.tooltips.Text() != "" {
		t.Errorf("the tip stayed after Enter: %q", w.tooltips.Text())
	}
	w.fyneWindow.Canvas().Focus(buttonOf(r, "view.notes"))
	r.selectTab("File") // made again
	if w.tooltips.Text() != "" {
		t.Errorf("the tip stayed with its tab gone: %q", w.tooltips.Text())
	}
}

// Live Preview shows checked however it is given: a key or the menu too.
func TestLivePreviewChecked(t *testing.T) {
	w, _ := scriptWindow(t, "# Story\n\nOnce.\n")
	r := ribbonOnFor(t, w)
	w.setDocumentType(true)
	r.selectTab("View")
	w.do(cmd("view.preview"))()
	if b := buttonOf(r, "view.preview"); b == nil || !b.checked {
		t.Error("Live Preview not checked on the ribbon after Ctrl+Shift+P")
	}
	w.do(cmd("view.preview"))()
	if buttonOf(r, "view.preview").checked {
		t.Error("Live Preview still checked once hidden")
	}
}

// menuItemChecked reports whether the item labelled label of command id's
// submenu is checked.
func menuItemChecked(t *testing.T, w *MainWindow, id, label string) bool {
	t.Helper()
	for _, it := range cmd(id).submenu(w).Items {
		if it.Label == label {
			return it.Checked
		}
	}
	t.Fatalf("%s has no %q", id, label)
	return false
}

// The borderline settings on the ribbon too (editor keys, font, new
// scripts' language, toolbar, printed scene numbers, drafts side by
// side): each writes what Preferences writes, an open Preferences window
// follows, and Preferences' changes show on the ribbon.
func TestRibbonMoreSettings(t *testing.T) {
	holdUI(t)
	w, _ := scriptWindow(t, revScript)
	r := ribbonOnFor(t, w)
	s := w.settings
	w.app.windows = append(w.app.windows, w)
	t.Cleanup(func() { w.app.windows = nil })
	t.Cleanup(func() {
		s.SetString("editor-keys", "standard")
		s.SetBoolean("vim-mode", false)
		s.SetString("font-family", "monospace")
		s.SetString("script-language", "en")
		s.SetBoolean("toolbar-visible", true)
		s.SetBoolean("fountain-scene-numbers", false)
		s.SetBoolean(draftsSidePref, true)
		w.applyEditorSettings()
	})
	pd := NewPreferencesDialog(w.app, w.fyneWindow)
	w.app.prefs = pd
	t.Cleanup(func() { w.app.prefs = nil; pd.cancel() })

	// editor keys, on Edit: switched mid-operator, the state starts again
	r.selectTab("Edit")
	if buttonOf(r, "set.editorKeys") == nil {
		t.Fatal("no Editor Keys on the Edit tab")
	}
	s.SetBoolean("vim-mode", true) // the older setting: Editor Keys clears it
	w.applyEditorSettings()
	if w.editorKeys() != "vim" || !menuItemChecked(t, w, "set.editorKeys", "Vim") {
		t.Fatal("the older vim-mode setting is not Vim")
	}
	w.vim.op, w.vim.count, w.vim.g = 'd', 3, true
	pressMenuItem(t, w, "set.editorKeys", "Helix")
	if s.GetString("editor-keys") != "helix" || s.GetBoolean("vim-mode") || pd.keysSelect.Selected != "helix" {
		t.Errorf("Helix: setting %q (vim-mode %v), Preferences %q", s.GetString("editor-keys"), s.GetBoolean("vim-mode"), pd.keysSelect.Selected)
	}
	if v := w.vim; v.op != 0 || v.count != 0 || v.g || v.mode != vimNormal {
		t.Errorf("switched mid-operator, the state stayed: op %q count %d g %v mode %v", v.op, v.count, v.g, v.mode)
	}
	if got := w.statusLabel.Text; !strings.Contains(got, "Helix") {
		t.Errorf("the status line does not say Helix: %q", got)
	}
	w.vim.mode = vimVisual
	pressMenuItem(t, w, "set.editorKeys", "Standard")
	if w.vim.mode != vimInsert || !strings.Contains(w.statusLabel.Text, "Standard") || pd.keysSelect.Selected != "standard" {
		t.Errorf("Standard: mode %v, status %q, Preferences %q", w.vim.mode, w.statusLabel.Text, pd.keysSelect.Selected)
	}
	pd.loadCurrentSettings()
	pd.keysSelect.SetSelected("vim")
	pd.saveSettings()
	w.app.applySettingsToWindows()
	if !menuItemChecked(t, w, "set.editorKeys", "Vim") || w.vim.mode != vimNormal {
		t.Error("Preferences' Vim does not show in Editor Keys, or vim is not in normal mode")
	}

	// the font, beside the text size on View
	r.selectTab("View")
	if b := buttonOf(r, "set.font"); b == nil || !b.compact {
		t.Fatal("no Font beside the text size")
	}
	if !menuItemChecked(t, w, "set.font", fontChoices[0][1]) {
		t.Error("the screenplay font is not checked")
	}
	pressMenuItem(t, w, "set.font", "System")
	if s.GetString("font-family") != "system" || pd.fontValue != "system" || pd.fontSelect.Selected != "System" {
		t.Errorf("Font > System: %q, Preferences %q (%q)", s.GetString("font-family"), pd.fontValue, pd.fontSelect.Selected)
	}
	// Font File… picks a file (it used to open Preferences, which had no picker)
	var asked []string
	old := chooseFileIn
	chooseFileIn = func(_ fyne.Window, exts []string, then func(string)) { asked = exts; then("/fonts/Picked.otf") }
	t.Cleanup(func() { chooseFileIn = old })
	pressMenuItem(t, w, "set.font", "Font File…")
	if s.GetString("font-family") != "/fonts/Picked.otf" || !slices.Equal(asked, fontExtensions) {
		t.Errorf("Font File…: %q, asked for %v", s.GetString("font-family"), asked)
	}
	if pd.fontValue != "/fonts/Picked.otf" || pd.fontSelect.Selected != "Picked.otf" {
		t.Errorf("Preferences after Font File…: %q (%q)", pd.fontValue, pd.fontSelect.Selected)
	}
	s.SetString("font-family", "/fonts/Mine.ttf") // a file chosen earlier
	if !menuItemChecked(t, w, "set.font", "Mine.ttf") || menuItemChecked(t, w, "set.font", fontChoices[0][1]) {
		t.Error("a font file is not shown as itself, checked")
	}

	// new scripts' language, in File > New
	r.selectTab("File")
	if b := buttonOf(r, "set.newLanguage"); b == nil || b.label != "New in: English" {
		t.Fatalf("New in: %+v", b)
	}
	pressMenuItem(t, w, "set.newLanguage", languageLabel("nl"))
	if s.GetString("script-language") != "nl" || pd.languageSelect.Selected != "nl" {
		t.Errorf("New in Dutch: %q, Preferences %q", s.GetString("script-language"), pd.languageSelect.Selected)
	}
	if b := buttonOf(r, "set.newLanguage"); b == nil || b.label != "New in: "+languageName("nl") {
		t.Errorf("the button does not say Dutch: %+v", b)
	}
	pd.loadCurrentSettings()
	pd.languageSelect.SetSelected("en")
	pd.saveSettings()
	w.app.applySettingsToWindows()
	if b := buttonOf(r, "set.newLanguage"); b == nil || b.label != "New in: English" {
		t.Error("Preferences' English does not show on the ribbon")
	}

	// printed scene numbers, in Production > Options
	r.selectTab("Production")
	nums := buttonOf(r, "set.sceneNumbers")
	if nums == nil || nums.checked {
		t.Fatal("no Scene Numbers in Production, or checked while off")
	}
	nums.press()
	if !s.GetBoolean("fountain-scene-numbers") || !nums.checked || !strings.Contains(pd.pageSummary.Text, "scenes numbered") {
		t.Error("Scene Numbers did not number the scenes everywhere")
	}

	// drafts side by side, in Collaboration > Drafts: on by default
	r.selectTab("Collaboration")
	side := buttonOf(r, "set.draftsSide")
	if side == nil || !side.checked {
		t.Fatal("Side by Side not there, or not on by default")
	}
	side.press()
	if w.draftsSideBySide() || side.checked {
		t.Error("Side by Side did not turn off")
	}
	side.press()
	if !w.draftsSideBySide() {
		t.Error("Side by Side did not turn on again")
	}

	// the toolbar: in View's menu (and ☰'s, the same menus), not on the ribbon
	found := false
	for _, m := range w.menusMenu().Items {
		if m.Label == "View" {
			for _, it := range m.ChildMenu.Items {
				found = found || it.Label == "Toolbar" && it.Checked
			}
		}
	}
	if !found {
		t.Error("☰ > View has no Toolbar, checked")
	}
	w.do(cmd("set.toolbar"))()
	if s.GetBoolean("toolbar-visible") || pd.toolbarVisibleCheck.Checked || w.headerArea.Visible() {
		t.Error("Toolbar did not hide the toolbar")
	}
	if w.menuGone { // the ribbon gone with it: the menu bar is back
		t.Error("the menu bar did not come back with the ribbon hidden")
	}
	w.do(cmd("set.toolbar"))()
	if !w.ribbonShown() || !pd.toolbarVisibleCheck.Checked {
		t.Error("Toolbar did not bring the ribbon back")
	}
	// the Menus look: View > Toolbar hides the header bar's rows and shows them again
	s.SetString(lookPref, "menus")
	w.applyEditorSettings()
	w.do(cmd("set.toolbar"))()
	if w.headerArea.Visible() || s.GetBoolean("toolbar-visible") {
		t.Error("Toolbar did not hide the header bar in the Menus look")
	}
	w.do(cmd("set.toolbar"))()
	if !w.headerArea.Visible() || w.headerArea.Objects[0] != w.headerBar.container {
		t.Error("Toolbar did not bring the header bar back in the Menus look")
	}
}

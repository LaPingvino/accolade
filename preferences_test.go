package main

import (
	"fyne.io/fyne/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
)

// Every setting the Preferences window saves is read somewhere else: no
// more checkboxes that do nothing.
func TestPreferencesOnlyShowWorkingSettings(t *testing.T) {
	prefs, _ := os.ReadFile("preferences_dialog.go")
	saved := regexp.MustCompile(`s\.Set(?:String|Boolean|Int)\("([a-z-]+)"`).FindAllStringSubmatch(string(prefs), -1)
	files, _ := filepath.Glob("*.go")
	var rest strings.Builder
	for _, f := range files {
		if f == "preferences_dialog.go" || f == "settings.go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, _ := os.ReadFile(f)
		rest.Write(b)
	}
	if len(saved) < 10 {
		t.Fatalf("found only %d saved settings", len(saved))
	}
	for _, m := range saved {
		if !strings.Contains(rest.String(), `"`+m[1]+`"`) {
			t.Errorf("Preferences saves %q, which nothing reads", m[1])
		}
	}
}

func TestToolbarAndStatusBarSettings(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetBoolean("toolbar-visible", false)
	w.settings.SetBoolean("statusbar-visible", false)
	t.Cleanup(func() {
		w.settings.SetBoolean("toolbar-visible", true)
		w.settings.SetBoolean("statusbar-visible", true)
	})
	w.app.applySettingsToWindows()
	if w.headerArea.Visible() || w.statusBar.Visible() {
		t.Error("toolbar or status bar still shown")
	}
	w.settings.SetBoolean("toolbar-visible", true)
	w.app.applySettingsToWindows()
	if !w.headerArea.Visible() {
		t.Error("toolbar not shown again")
	}
}

func TestAutoCloseBrackets(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	e := w.textEditor
	e.SetText("ANNA\n")
	w.fyneWindow.Canvas().Focus(e)
	e.Navigate(func(b *buffer.Buffer) { b.SetCursor(b.Len(), false) })
	test.Type(e, "(beat")
	if got := e.Text(); got != "ANNA\n(beat)" {
		t.Fatalf("typed ( gives %q", got)
	}
	test.Type(e, ")")
	if got := e.Text(); got != "ANNA\n(beat)" || e.CursorOffset() != len([]rune(got)) {
		t.Errorf("typing ) over the closing one: %q, cursor %d", got, e.CursorOffset())
	}
	w.settings.SetBoolean("auto-close-brackets", false)
	t.Cleanup(func() { w.settings.SetBoolean("auto-close-brackets", true) })
	test.Type(e, "(")
	if got := e.Text(); got != "ANNA\n(beat)(" {
		t.Errorf("with the setting off: %q", got)
	}
}

// Reset to Defaults shows the defaults; nothing changes before OK or
// Apply, and the recent files are no preference.
func TestResetToDefaultsWaitsForOK(t *testing.T) {
	w, app := newDialogTestWindow(t)
	w.settings.SetString("theme", "dark")
	w.settings.SetStringSlice("recent-files", []string{"/tmp/a.fountain"})
	t.Cleanup(func() {
		w.settings.SetString("theme", "system")
		w.settings.SetStringSlice("recent-files", nil)
	})
	pd := NewPreferencesDialog(app, w.fyneWindow)
	defaults := &Settings{data: map[string]interface{}{}}
	defaults.loadDefaults()
	pd.loadFrom(defaults) // what the confirmed reset does
	if pd.themeSelect.Selected != "system" {
		t.Errorf("shown theme %q", pd.themeSelect.Selected)
	}
	if w.settings.GetString("theme") != "dark" {
		t.Error("reset changed the settings before OK")
	}
	pd.ok()
	if w.settings.GetString("theme") != "system" || len(w.settings.GetStringSlice("recent-files")) != 1 {
		t.Errorf("after OK: theme %q, recent %v", w.settings.GetString("theme"), w.settings.GetStringSlice("recent-files"))
	}
}

// Keyboard: Ctrl+PageDown / PageUp turn the Preferences pages; Escape in
// the Export dialog does what its Cancel does.
func TestKeyboardPagesAndEscape(t *testing.T) {
	w, app := newDialogTestWindow(t)
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.turnPage(1)
	if pd.tabs.SelectedIndex() != 1 {
		t.Errorf("Ctrl+PageDown: page %d", pd.tabs.SelectedIndex())
	}
	pd.turnPage(-1)
	pd.turnPage(-1)
	if pd.tabs.SelectedIndex() != len(pd.tabs.Items)-1 {
		t.Errorf("Ctrl+PageUp from the first: page %d", pd.tabs.SelectedIndex())
	}
	pd.cancel()

	ed := NewExportDialog(w)
	ed.Show()
	top, ok := w.fyneWindow.Canvas().Overlays().Top().(fyne.EscapeHandler)
	if !ok || !top.HandleEscape() {
		t.Fatal("Escape not handled by the Export dialog")
	}
	if w.fyneWindow.Canvas().Overlays().Top() != nil {
		t.Error("Escape left the Export dialog open")
	}
}

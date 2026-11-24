package main

import (
	"fyne.io/fyne/v2"
	"image/png"
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
	if pd.themeSelect.Selected != "System" {
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

// TestPreferencesFontShot renders Preferences' Editor page with a font
// file chosen (SHOT_DIR as in TestScreenshots).
func TestPreferencesFontShot(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	w, app := newDialogTestWindow(t)
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.showFont("/fonts/Mine.ttf")
	pd.tabs.SelectIndex(1)
	pd.win.Resize(fyne.NewSize(640, 540))
	f, err := os.Create(filepath.Join(dir, "preferences-font.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, pd.win.Canvas().Capture())
}

// The saved theme is the window's from the start (set before the window
// is built), and a second window does not theme the app again.
func TestNewWindowTakesTheSavedTheme(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := GetSettings()
	s.SetString("theme", "dark")
	t.Cleanup(func() { s.SetString("theme", "system") })
	a := test.NewApp()
	t.Cleanup(a.Quit)
	app := &Application{fyneApp: a}
	w := NewMainWindow(app)
	t.Cleanup(w.stopTimers)
	if shownThemeID() != "dark" {
		t.Fatalf("the new window's theme: %T", a.Settings().Theme())
	}
	other := newAccoladeTheme("light", "", nil)
	a.Settings().SetTheme(other) // (not through setColorScheme: what a second window would undo)
	w2 := NewMainWindow(app)
	t.Cleanup(w2.stopTimers)
	if a.Settings().Theme() != other {
		t.Errorf("a second window set the theme again: %T", a.Settings().Theme())
	}
	app.setColorScheme("light")
	app.setColorScheme("dark")
	if shownThemeID() != "dark" {
		t.Errorf("a changed scheme is not applied: %T", a.Settings().Theme())
	}
}

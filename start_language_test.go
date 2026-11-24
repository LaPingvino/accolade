package main

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// A new window starts with New's text in the preferences' language: the
// Esperanto sample with Esperanto set, the English template with English;
// unchanged, as a sample left alone is.
func TestNewWindowFollowsLanguageSetting(t *testing.T) {
	for _, tc := range []struct{ lang, has string }{
		{"eo", languageSample("eo")},
		{"en", "INT. LIVING ROOM - DAY"},
	} {
		s := GetSettings()
		was := s.GetString("script-language")
		s.SetString("script-language", tc.lang)
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		a := test.NewApp()
		w := NewMainWindow(&Application{fyneApp: a})
		text := w.textEditor.Text()
		w.stopTimers()
		a.Quit()
		s.SetString("script-language", was)
		if tc.has == "" {
			t.Fatalf("%s: no sample to compare", tc.lang)
		}
		if w.language != tc.lang || !strings.Contains(text, strings.TrimSpace(tc.has)) {
			t.Errorf("%s: a new window is in %q with\n%.120s", tc.lang, w.language, text)
		}
		if w.hasChanges || w.filledWith != text {
			t.Errorf("%s: the first text counts as a change (%v) or not as the sample", tc.lang, w.hasChanges)
		}
	}
}

// Out of the gate the language is English: a fresh settings folder, and
// Preferences saved untouched on it; the default text reads as English.
func TestFreshSettingsAreEnglish(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if l := NewSettings().GetString("script-language"); l != "en" {
		t.Fatalf("fresh settings: %q", l)
	}
	if l := headingsLanguage(defaultScript(time.Now()), "eo"); l != "en" {
		t.Errorf("the English template read as %q", l)
	}
	w, app := newDialogTestWindow(t)
	s := w.settings
	was := s.GetString("script-language")
	t.Cleanup(func() { s.SetString("script-language", was) })
	s.SetString("script-language", "en")
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.loadCurrentSettings()
	pd.saveSettings()
	if l := s.GetString("script-language"); l != "en" {
		t.Errorf("Preferences saved untouched: %q", l)
	}
}

// The tests keep their settings to themselves: none written where the
// user's are (TestMain's folders, not the real home's).
func TestSettingsNotTheUsers(t *testing.T) {
	if realHome == "" || strings.HasPrefix(getConfigDir(), realHome+string(filepath.Separator)) {
		t.Errorf("the tests' settings are the user's: %s (home %q)", getConfigDir(), realHome)
	}
}

// New in: picks this window's next language only; Use for New Scripts by
// Default makes it the preferences' (marked "(default)"), and again goes
// back to English; the palette's makes the script's language the default.
func TestDefaultLanguageToggle(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	s := w.settings
	was := s.GetString("script-language")
	s.SetString("script-language", "en")
	t.Cleanup(func() { s.SetString("script-language", was) })

	pressMenuItem(t, w, "set.newLanguage", languageLabel("eo"))
	if s.GetString("script-language") != "en" || w.newFileLanguage() != "eo" {
		t.Fatalf("New in Esperanto: default %q, next %q", s.GetString("script-language"), w.newFileLanguage())
	}
	if !menuItemChecked(t, w, "set.newLanguage", languageLabel("eo")) || menuItemChecked(t, w, "set.newLanguage", useByDefault) {
		t.Error("Esperanto not checked, or checked as the default")
	}
	if menuItemChecked(t, w, "set.newLanguage", languageLabel("en")+" (default)") { // there, not picked
		t.Error("English checked, Esperanto picked")
	}
	pressMenuItem(t, w, "set.newLanguage", useByDefault)
	if s.GetString("script-language") != "eo" || !menuItemChecked(t, w, "set.newLanguage", useByDefault) {
		t.Errorf("Use by Default: %q", s.GetString("script-language"))
	}
	pressMenuItem(t, w, "set.newLanguage", useByDefault)
	if s.GetString("script-language") != "en" {
		t.Errorf("Use by Default again: %q", s.GetString("script-language"))
	}

	// the palette's: the script's own language
	w.setLanguage("nl")
	cmd("set.defaultLanguage").run(w)
	if s.GetString("script-language") != "nl" || !cmd("set.defaultLanguage").checked(w) {
		t.Errorf("the script's language as the default: %q", s.GetString("script-language"))
	}
	s.SetString("script-language", "en")
}

// TestLanguageMenuScreenshots renders New in:'s menu, the place button's
// and the Script Language dialog, with the default toggle (SHOT_DIR as
// TestScreenshots).
func TestLanguageMenuScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	os.MkdirAll(dir, 0o755)
	w, _ := newDialogTestWindow(t)
	s := w.settings
	was := s.GetString("script-language")
	s.SetString("script-language", "en")
	t.Cleanup(func() { s.SetString("script-language", was) })
	w.newLanguage = "eo" // picked, not the default
	shot := func(name string) {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		png.Encode(f, w.fyneWindow.Canvas().Capture())
		f.Close()
	}
	c := w.fyneWindow.Canvas()
	m := widget.NewPopUpMenu(cmd("set.newLanguage").submenu(w), c)
	m.ShowAtPosition(fyne.NewPos(20, 20))
	shot("new-in-menu.png")
	m.Hide()
	w.textEditor.SetText("INT. KITCHEN - DAY\n\nRain.\n")
	w.pickPlace(nil)
	shot("place-menu.png")
	for _, o := range c.Overlays().List() {
		o.Hide()
	}
	w.pickLanguage()
	shot("script-language-dialog.png")
}

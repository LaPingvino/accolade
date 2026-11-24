package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
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


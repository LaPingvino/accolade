package main

import (
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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func editorTextSize(w *MainWindow) float32 {
	return w.textEditor.Theme().Size(theme.SizeNameText)
}

func editorFont(w *MainWindow) fyne.Resource {
	return w.textEditor.Theme().Font(fyne.TextStyle{Monospace: true})
}

func TestEditorUsesScreenplayFontAndSize(t *testing.T) {
	w := newTestWindow(t, "")
	if editorFont(w) != courierRegular {
		t.Errorf("editor font = %v, want the bundled screenplay font", editorFont(w))
	}
	if got := w.textEditor.Theme().Font(fyne.TextStyle{Monospace: true, Italic: true}); got != courierItalic {
		t.Errorf("italic font = %v", got)
	}
	if got, want := editorTextSize(w), float32(w.settings.GetInt("font-size"))*4/3; got != want {
		t.Errorf("text size = %v, want %v", got, want)
	}
	// the rest of the window keeps the normal UI font
	if w.searchBar.searchEntry.Theme().Font(fyne.TextStyle{}) == courierRegular {
		t.Error("the search field should not use the screenplay font")
	}
}

func TestFontPreferencesApplyToOpenWindows(t *testing.T) {
	w := newTestWindow(t, "")
	w.app.windows = append(w.app.windows, w) // as Application.newWindow does
	prevFamily, prevSize := w.settings.GetString("font-family"), w.settings.GetInt("font-size")
	t.Cleanup(func() {
		w.settings.SetString("font-family", prevFamily)
		w.settings.SetInt("font-size", prevSize)
	})

	w.settings.SetInt("font-size", 15)
	w.settings.SetString("font-family", "system")
	w.app.applySettingsToWindows()
	if got := editorTextSize(w); got != 20 {
		t.Errorf("text size after preferences = %v, want 20", got)
	}
	if editorFont(w) != theme.DefaultTheme().Font(fyne.TextStyle{Monospace: true}) {
		t.Error(`"system" should use Fyne's monospace font`)
	}

	fontFile := filepath.Join(t.TempDir(), "My Font.ttf")
	os.WriteFile(fontFile, courierRegular.Content(), 0o644)
	w.settings.SetString("font-family", fontFile)
	w.app.applySettingsToWindows()
	if f := editorFont(w); f == nil || f.Name() != "My Font.ttf" {
		t.Errorf("font file not used: %v", f)
	}

	w.settings.SetString("font-family", filepath.Join(t.TempDir(), "missing.ttf"))
	w.app.applySettingsToWindows()
	if editorFont(w) != courierRegular {
		t.Error("a missing font file should fall back to the screenplay font")
	}
}

func TestAutoIndentOff(t *testing.T) {
	w := newTestWindow(t, "")
	prev := w.settings.GetBoolean("auto-indent")
	w.settings.SetBoolean("auto-indent", false)
	t.Cleanup(func() { w.settings.SetBoolean("auto-indent", prev) })

	typeScript(w, "int. barn", "", "JOHN", "hi")
	if got, want := w.textEditor.Text(), "int. barn\n\nJOHN\nhi\n"; got != want {
		t.Errorf("with auto-indent off: %q, want %q", got, want)
	}
}

func TestLineNumbersPreference(t *testing.T) {
	w := newTestWindow(t, "one\ntwo")
	w.app.windows = append(w.app.windows, w)
	prev := w.settings.GetBoolean("show-line-numbers")
	t.Cleanup(func() { w.settings.SetBoolean("show-line-numbers", prev) })

	w.settings.SetBoolean("show-line-numbers", true)
	w.app.applySettingsToWindows()
	if got := w.textEditor.GridRow(0).Cells[0].Rune; got != '1' {
		t.Errorf("first cell %q, want the line number", got)
	}
	w.settings.SetBoolean("show-line-numbers", false)
	w.app.applySettingsToWindows()
	if got := w.textEditor.GridRow(0).Cells[0].Rune; got != 'o' {
		t.Errorf("first cell %q without line numbers", got)
	}
}

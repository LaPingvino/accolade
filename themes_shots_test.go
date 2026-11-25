package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// TestThemeScreenshots renders every theme in every variant it has, at a
// desktop's size and a phone's: the ribbon, the script with its notes
// and marks, the Notes pane, and a dialog over them; then the theme
// editor (SHOT_DIR as in screenshot_test.go).
func TestThemeScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, app := newDialogTestWindow(t)
	s := w.settings
	s.SetString(lookPref, "ribbon")
	t.Cleanup(func() {
		s.SetString(lookPref, "menus")
		s.SetString("theme", "system")
		s.SetString(themeVariantPref, "system")
	})
	w.applyLook()
	w.textEditor.SetText(themeShotScript)
	w.togglePane("notes")
	w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())) / 4)
	snap := func(c fyne.Canvas, name string) {
		t.Helper()
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, c.Capture()); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range themeDefs {
		for _, v := range themeVariants(d) {
			s.SetString("theme", d.id)
			s.SetString(themeVariantPref, variantName(v))
			app.setColorScheme(d.id)
			w.updateOpen()
			for _, size := range []struct {
				name string
				size fyne.Size
			}{{"desktop", fyne.NewSize(1200, 760)}, {"phone", fyne.NewSize(380, 800)}} {
				w.fyneWindow.Resize(size.size)
				w.fyneWindow.Content().Refresh()
				name := fmt.Sprintf("theme-%s-%s-%s", d.id, variantName(v), size.name)
				snap(w.fyneWindow.Canvas(), name)
				dl := dialog.NewConfirm("Save changes?", "The script has changes that are not saved yet.", func(bool) {}, w.fyneWindow)
				dl.Show()
				snap(w.fyneWindow.Canvas(), name+"-dialog")
				dl.Hide()
			}
		}
	}
	// the theme editor, from Slate, in Preferences
	s.SetString("theme", "slate")
	s.SetString(themeVariantPref, "dark")
	app.setColorScheme("slate")
	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.win.Resize(fyne.NewSize(1000, 700))
	pd.customizeTheme()
	snap(pd.win.Canvas(), "theme-editor-slate")
	te := &themeEditor{pd: pd, custom: &customTheme{Base: "slate", Dark: map[string]string{"muted": "#4e5c76", "accent": "#7a6a9a"}}, dark: true}
	te.show()
	snap(pd.win.Canvas(), "theme-editor-warnings")
	// at a phone's width: the preview over the list, no wider than the phone
	phone := NewPreferencesDialog(app, w.fyneWindow)
	phone.win.Resize(fyne.NewSize(380, 800))
	phone.customizeTheme()
	snap(phone.win.Canvas(), "theme-editor-phone")
	if top := phone.win.Canvas().Overlays().Top(); top == nil {
		t.Error("no theme editor at a phone's width")
	} else if mw := top.MinSize().Width; mw > 380 {
		t.Errorf("the theme editor is %v wide at a phone's width", mw)
	}
}

const themeShotScript = `Title: The Barn
Author: Mara Lind

# Act One

= Mara comes home to claim the farm.

INT. BARN - DAY

Rain hammers the roof. MARA (30s) drags the door open. [[todo: is the barn hers yet?]]

MARA
(to herself)
It was always ours.

[[Mother's letter goes here, unread.]]

/* EXT. ROAD - NIGHT

A cut scene: the drive in. */

EXT. FIELD - NIGHT

= She finds the deed.

Wind in the wheat. Mara walks the fence line with a torch.

TOM (O.S.)
You came back.

MARA
For the land. Not for you.

CUT TO:

INT. KITCHEN - CONTINUOUS

The kettle screams. [[todo: give Tom a reason to stay]]
`

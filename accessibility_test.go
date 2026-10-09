package main

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// unnamed lists what a screen reader would find in a canvas without a
// real name: focusable objects with no name, a placeholder for one
// ("(Select one)") or a file name ("foreground_cancel.svg"), walking the
// objects as the accessibility bridge does.
func unnamed(c fyne.Canvas) []string {
	var out []string
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		if a, ok := o.(fyne.Accessible); ok && a.AccessibilityRole() != fyne.AccessibleRoleContainer {
			name := strings.TrimSpace(fyne.AccessibleLabel(o))
			if _, focusable := o.(fyne.Focusable); focusable &&
				(name == "" || strings.HasPrefix(name, "(") || strings.Contains(name, ".svg")) {
				out = append(out, fmt.Sprintf("%T %q", o, name))
			}
		}
		if ac, ok := o.(fyne.AccessibleChildren); ok {
			for _, k := range ac.AccessibilityChildren() {
				walk(k)
			}
		} else if ct, ok := o.(*fyne.Container); ok {
			for _, k := range ct.Objects {
				walk(k)
			}
		}
	}
	walk(c.Content())
	for _, o := range c.Overlays().List() {
		walk(o)
	}
	return out
}

// Everything a keyboard or screen reader user reaches has a name: the
// window with all its panes, every Preferences page and the dialogs.
func TestEverythingHasAName(t *testing.T) {
	w, app := newDialogTestWindow(t)
	t.Cleanup(func() {
		w.settings.SetBoolean("preview-visible", false)
		w.settings.SetBoolean("outline-visible", false)
		w.settings.SetBoolean("notes-visible", false)
	})
	w.textEditor.SetText("Title: X\n\nINT. BARN - DAY\n\nRain.\n\nANNA\nHi.\n")
	check := func(what string, c fyne.Canvas) {
		if u := unnamed(c); len(u) > 0 {
			t.Errorf("%s: no name for\n  %s", what, strings.Join(u, "\n  "))
		}
	}
	w.showPreview()
	w.toggleOutline()
	w.showFindReplace()
	check("window with preview, outline and search", w.fyneWindow.Canvas())
	w.hideFindReplace()
	w.toggleNotes()
	check("window with notes", w.fyneWindow.Canvas())

	pd := NewPreferencesDialog(app, w.fyneWindow)
	for i := range pd.tabs.Items {
		pd.tabs.SelectIndex(i)
		check("Preferences, "+pd.tabs.Items[i].Text, pd.win.Canvas())
	}
	pd.cancel()

	ed := NewExportDialog(w)
	ed.Show()
	check("Export", w.fyneWindow.Canvas())
	ed.cancel()

	tp := NewTitlePageDialog(w)
	tp.Show()
	check("Title Page", w.fyneWindow.Canvas())
}

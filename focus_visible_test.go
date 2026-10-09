package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// invisibleFocus lists the objects of a canvas that Tab reaches but that
// look the same with the focus as without it: a keyboard user cannot see
// where they are.
func invisibleFocus(c fyne.Canvas) []string {
	scale := c.Scale()
	area := func(o fyne.CanvasObject) image.Rectangle {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
		size := o.Size()
		return image.Rect(int(pos.X*scale), int(pos.Y*scale), int((pos.X+size.Width)*scale), int((pos.Y+size.Height)*scale))
	}
	var out []string
	c.Unfocus()
	seen := map[fyne.Focusable]bool{}
	for i := 0; i < 80; i++ {
		c.FocusNext()
		f := c.Focused()
		if f == nil || seen[f] {
			break
		}
		seen[f] = true
		o, ok := f.(fyne.CanvasObject)
		if !ok {
			continue
		}
		focused := c.Capture()
		c.Unfocus()
		unfocused := c.Capture()
		c.Focus(f)
		r := area(o).Intersect(focused.Bounds())
		changed := 0
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if focused.At(x, y) != unfocused.At(x, y) {
					changed++
				}
			}
		}
		if changed == 0 && os.Getenv("FOCUS_SHOTS") != "" {
			for name, img := range map[string]image.Image{"focused": focused, "unfocused": unfocused} {
				label := strings.Map(func(r rune) rune {
					if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
						return r
					}
					return '_'
				}, fyne.AccessibleLabel(o))
				f, _ := os.Create(fmt.Sprintf("%s/%s-%s.png", os.Getenv("FOCUS_SHOTS"), label, name))
				png.Encode(f, img)
				f.Close()
			}
		}
		if changed == 0 {
			dis := false
			if d, ok := o.(fyne.Disableable); ok {
				dis = d.Disabled()
			}
			out = append(out, fmt.Sprintf("%T %q at %v (disabled %v)", o, fyne.AccessibleLabel(o), area(o), dis))
		}
	}
	return out
}

// Everything Tab reaches shows that it has the focus.
func TestFocusIsVisible(t *testing.T) {
	w, app := newDialogTestWindow(t)
	t.Cleanup(func() {
		w.settings.SetBoolean("outline-visible", false)
		w.settings.SetBoolean("notes-visible", false)
	})
	w.textEditor.SetText("Title: X\n\nINT. BARN - DAY\n\nRain.\n")
	w.fyneWindow.Resize(fyne.NewSize(1000, 700))
	check := func(what string, c fyne.Canvas) {
		if u := invisibleFocus(c); len(u) > 0 {
			t.Errorf("%s: the focus is not seen on\n  %s", what, strings.Join(u, "\n  "))
		}
	}
	w.toggleOutline()
	w.showFindReplace()
	check("window with outline and search", w.fyneWindow.Canvas())
	w.hideFindReplace()

	pd := NewPreferencesDialog(app, w.fyneWindow)
	pd.win.Resize(fyne.NewSize(900, 820)) // all of every page in view
	for i := range pd.tabs.Items {
		pd.tabs.SelectIndex(i)
		check("Preferences, "+pd.tabs.Items[i].Text, pd.win.Canvas())
	}
	pd.cancel()
}

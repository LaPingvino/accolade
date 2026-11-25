package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// TestThemeSmallTextPixels renders the window in every theme and reads
// the pixels of its small text (the ribbon's captions and shortcuts, the
// Notes items' kind labels, the status bar): each text's darkest (or
// lightest) glyph pixel against the background around it, as drawn,
// antialiased. The colour checker (theme_contrast.go) reads the colours
// a theme names; this reads what is on the screen, so a colour that
// passes on paper and reads faint fails here.
func TestThemeSmallTextPixels(t *testing.T) {
	for _, mac := range []bool{false, true} {
		name := "ctrl"
		if mac {
			name = "mac"
		}
		t.Run(name, func(t *testing.T) {
			was := macKeyLabels
			macKeyLabels = mac // the Mac's ⌘ labels draw thinner than Ctrl+…: both, on every OS
			t.Cleanup(func() { macKeyLabels = was })
			smallTextPixels(t)
		})
	}
}

func smallTextPixels(t *testing.T) {
	w, app := newDialogTestWindow(t)
	s := w.settings
	s.SetString(lookPref, "ribbon")
	s.SetBoolean("show-line-numbers", true)
	t.Cleanup(func() {
		s.SetString(lookPref, "menus")
		s.SetBoolean("show-line-numbers", false)
		s.SetString("theme", "system")
		s.SetString(themeVariantPref, "system")
	})
	w.applyLook()
	app.applySettingsToWindows()
	w.textEditor.SetText(themeShotScript)
	w.togglePane("notes")
	w.fyneWindow.Resize(fyne.NewSize(1200, 760))
	for _, d := range themeDefs {
		for _, v := range themeVariants(d) {
			s.SetString("theme", d.id)
			s.SetString(themeVariantPref, variantName(v))
			app.setColorScheme(d.id)
			w.updateOpen()
			w.fyneWindow.Content().Refresh()
			ms := smallTextContrast(w.fyneWindow.Canvas())
			if len(ms) < 5 {
				t.Errorf("%s (%s): only %d small texts measured", d.label, variantName(v), len(ms))
			}
			if os.Getenv("SMALL_TEXT_ALL") != "" {
				t.Logf("%s (%s): %v", d.label, variantName(v), ms)
			} else {
				t.Logf("%s (%s): %d small texts, the faintest %v", d.label, variantName(v), len(ms), ms[:min(3, len(ms))])
			}
			for _, m := range ms {
				if m.ratio < contrastSmall {
					t.Errorf("%s (%s): %q reads %.2f:1 as drawn (%s on %s; needs %.0f:1)",
						d.label, variantName(v), m.text, m.ratio, hexOf(m.core), hexOf(m.bg), contrastSmall)
				}
			}
		}
	}
}

// smallTextMeasure is one small text as drawn.
type smallTextMeasure struct {
	text     string
	core, bg color.NRGBA
	ratio    float64
}

// smallTextContrast measures each small text in sight on c: at the
// caption size or smaller, or italic (the thin ones).
func smallTextContrast(c fyne.Canvas) []smallTextMeasure {
	img := c.Capture()
	scale := c.Scale()
	caption := theme.CaptionTextSize() + 0.5
	var out []smallTextMeasure
	seen := map[string]bool{}
	for _, o := range shownObjects(c) {
		tx, ok := o.(*canvas.Text)
		if !ok || strings.TrimSpace(tx.Text) == "" || tx.TextSize > caption && !tx.TextStyle.Italic {
			continue
		}
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(tx)
		size := tx.MinSize()
		r := image.Rect(int(pos.X*scale), int(pos.Y*scale), int((pos.X+size.Width)*scale), int((pos.Y+size.Height)*scale)).Intersect(img.Bounds())
		if r.Dx() < 4 || r.Dy() < 4 {
			continue
		}
		m, ok := glyphContrast(img, r)
		if !ok {
			continue
		}
		key := tx.Text + hexOf(m.core) + hexOf(m.bg)
		if seen[key] {
			continue
		}
		seen[key] = true
		m.text = tx.Text
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ratio < out[j].ratio })
	return out
}

// shownObjects are the objects drawn on c: visible, and all that hold
// them visible too (a widget's by its renderer).
func shownObjects(c fyne.Canvas) []fyne.CanvasObject {
	var out []fyne.CanvasObject
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		out = append(out, o)
		switch x := o.(type) {
		case *fyne.Container:
			for _, k := range x.Objects {
				walk(k)
			}
		case fyne.Widget:
			for _, k := range test.WidgetRenderer(x).Objects() {
				walk(k)
			}
		}
	}
	walk(c.Content())
	return out
}

// glyphContrast is a text's area: its background (the commonest colour)
// and its glyphs' core (the pixel standing out most from it).
func glyphContrast(img image.Image, r image.Rectangle) (smallTextMeasure, bool) {
	counts := map[color.NRGBA]int{}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			counts[color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)]++
		}
	}
	var bg color.NRGBA
	best := 0
	for c, n := range counts {
		if n > best {
			bg, best = c, n
		}
	}
	core, ratio := bg, 1.0
	for c := range counts {
		if r := contrastRatio(c, bg); r > ratio {
			core, ratio = c, r
		}
	}
	return smallTextMeasure{core: core, bg: bg, ratio: ratio}, ratio > 1.05
}

// (for a look at the measures: go test -run TestThemeSmallTextPixels -v)
func (m smallTextMeasure) String() string {
	return fmt.Sprintf("%q %.2f:1 %s on %s", m.text, m.ratio, hexOf(m.core), hexOf(m.bg))
}

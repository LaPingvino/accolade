package main

import (
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	lexfont "github.com/LaPingvino/lexington/font"
)

// editorTheme is the theme of the script editor: the app's theme, with the
// screenplay font and the font size from the settings.
type editorTheme struct {
	size           float32 // text size; 0 keeps the app theme's
	regular, slant fyne.Resource
}

var (
	courierRegular = fyne.NewStaticResource("CourierBadi-Regular.ttf", lexfont.CourierBadiRegular)
	courierItalic  = fyne.NewStaticResource("CourierBadi-Italic.ttf", lexfont.CourierBadiItalic)
)

// newEditorTheme builds the editor theme for a font family and size in
// points. The family is "monospace" or a Courier name for the bundled
// screenplay font (Courier Badi, a Courier Prime derivative), "system" for
// Fyne's own monospace font, or the path of a .ttf/.otf file.
func newEditorTheme(family string, sizePt int) *editorTheme {
	t := &editorTheme{}
	if sizePt > 0 {
		t.size = float32(sizePt) * 4 / 3 // points to device-independent pixels
	}

	switch f := strings.TrimSpace(family); {
	case strings.EqualFold(f, "system"):
	case strings.HasSuffix(strings.ToLower(f), ".ttf") || strings.HasSuffix(strings.ToLower(f), ".otf"):
		data, err := os.ReadFile(f)
		if err != nil {
			log.Printf("Editor font %s: %v; using the screenplay font", f, err)
			t.regular, t.slant = courierRegular, courierItalic
			break
		}
		t.regular = fyne.NewStaticResource(filepath.Base(f), data)
		t.slant = t.regular
	default:
		t.regular, t.slant = courierRegular, courierItalic
	}
	return t
}

func (t *editorTheme) base() fyne.Theme {
	if app := fyne.CurrentApp(); app != nil && app.Settings().Theme() != nil {
		return app.Settings().Theme()
	}
	return theme.DefaultTheme()
}

func (t *editorTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return t.base().Color(n, v)
}

func (t *editorTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return t.base().Icon(n)
}

func (t *editorTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace && t.regular != nil {
		if s.Italic {
			return t.slant
		}
		return t.regular
	}
	return t.base().Font(s)
}

func (t *editorTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText && t.size > 0 {
		return t.size
	}
	return t.base().Size(n)
}

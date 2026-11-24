package editor

import (
	"image/color"

	"fyne.io/fyne/v2"
)

// The editor's own colours, which a theme may set so the writing surface
// fits its chrome. A theme without them (Fyne's own) gets what the editor
// derives from its standard colours: the accent for headings, the text
// dimmed for notes, the placeholder colour for the fold marks.
const (
	ColorNameSceneHeading fyne.ThemeColorName = "accolade-scene-heading" // scene headings, Markdown headings
	ColorNameNote         fyne.ThemeColorName = "accolade-note"          // [[notes]], sections, synopses, comments
	ColorNameFold         fyne.ThemeColorName = "accolade-fold"          // the fold marks and a folded block's summary
)

// Themed is the theme's colour n, or fallback when the theme has none
// (nil, or fully transparent: what Fyne's theme gives an unknown name).
func Themed(th fyne.Theme, n fyne.ThemeColorName, v fyne.ThemeVariant, fallback color.Color) color.Color {
	if th == nil {
		return fallback
	}
	c := th.Color(n, v)
	if c == nil {
		return fallback
	}
	if _, _, _, a := c.RGBA(); a == 0 {
		return fallback
	}
	return c
}

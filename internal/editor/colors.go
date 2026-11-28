// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package editor

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The editor's own colours, which a theme may set so the writing surface
// fits its chrome. A theme without them (Fyne's own) gets what the editor
// derives from its standard colours: the accent for headings, the text
// dimmed for notes, the placeholder colour for the fold marks.
const (
	ColorNameSceneHeading fyne.ThemeColorName = "accolade-scene-heading" // scene headings, Markdown headings
	ColorNameNote         fyne.ThemeColorName = "accolade-note"          // [[notes]], sections, synopses, comments
	ColorNameFold         fyne.ThemeColorName = "accolade-fold"          // the fold marks and a folded block's summary
	ColorNameDim          fyne.ThemeColorName = "accolade-dim"           // parentheticals, transitions, the title page, code
)

// ElementColors are the colours the editor draws a script's landmarks
// and quiet parts in: scene (and Markdown) headings, notes, dimmed text.
func ElementColors(th fyne.Theme, v fyne.ThemeVariant) (heading, note, dim color.Color) {
	fg := th.Color(theme.ColorNameForeground, v)
	derived := mix(fg, th.Color(theme.ColorNameBackground, v), 0.45)
	dim = Themed(th, ColorNameDim, v, derived)
	return Themed(th, ColorNameSceneHeading, v, th.Color(theme.ColorNamePrimary, v)), Themed(th, ColorNameNote, v, dim), dim
}

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

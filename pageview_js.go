//go:build js

package main

import (
	"image"
	"image/draw"

	"github.com/LaPingvino/accolade/internal/editor"
)

// In the browser the PDF writer has no fonts of the computer's: a
// character Courier Badi lacks takes its cells blank in the PDF. The page
// view draws it in the browser's fonts, as the editor does, in those
// cells: an em high, its middle as an ideograph's (0.38 em over the
// baseline).
func init() {
	pageFallback = func(s string, size, w float64) *image.Alpha {
		if w < 1 || size < 1 {
			return nil
		}
		pic := editor.PixelGlyph(s, size, w, size)
		b := pic.Bounds()
		top := int(-0.88 * size)
		m := image.NewAlpha(image.Rect(0, top, b.Dx(), top+b.Dy()))
		draw.Draw(m, m.Rect, pic, b.Min, draw.Src)
		return m
	}
}

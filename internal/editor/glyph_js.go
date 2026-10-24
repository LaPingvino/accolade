//go:build js

package editor

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"syscall/js"
)

// In the browser, wide characters (CJK, emoji) are drawn by the browser
// with its own fonts: the font Accolade bundles has none of them.
func init() { wideGlyph = browserGlyph }

type glyphKey struct {
	s          string
	c          color.NRGBA
	bold       bool
	size, w, h float32
}

var (
	glyphs      = map[glyphKey]image.Image{}
	glyphCanvas js.Value
)

// browserGlyph draws s centred in a picture of w by h (in Fyne's units, at
// the screen's pixel ratio so it stays sharp), kept for the next time.
func browserGlyph(s string, c color.Color, bold bool, size, w, h float32) image.Image {
	key := glyphKey{s, color.NRGBAModel.Convert(c).(color.NRGBA), bold, size, w, h}
	if img, ok := glyphs[key]; ok {
		return img
	}
	ratio := js.Global().Get("devicePixelRatio").Float()
	if ratio <= 0 {
		ratio = 1
	}
	pw, ph := int(math.Ceil(float64(w)*ratio)), int(math.Ceil(float64(h)*ratio))
	if !glyphCanvas.Truthy() {
		glyphCanvas = js.Global().Get("document").Call("createElement", "canvas")
	}
	glyphCanvas.Set("width", pw)
	glyphCanvas.Set("height", ph)
	ctx := glyphCanvas.Call("getContext", "2d", map[string]any{"willReadFrequently": true})
	ctx.Call("clearRect", 0, 0, pw, ph)
	weight := "normal"
	if bold {
		weight = "bold"
	}
	ctx.Set("font", fmt.Sprintf("%s %.1fpx monospace", weight, float64(size)*ratio))
	ctx.Set("textAlign", "center")
	ctx.Set("textBaseline", "middle")
	k := key.c
	ctx.Set("fillStyle", fmt.Sprintf("rgba(%d,%d,%d,%.3f)", k.R, k.G, k.B, float64(k.A)/255))
	ctx.Call("fillText", s, float64(pw)/2, float64(ph)/2)
	data := ctx.Call("getImageData", 0, 0, pw, ph).Get("data")
	img := image.NewNRGBA(image.Rect(0, 0, pw, ph))
	js.CopyBytesToGo(img.Pix, js.Global().Get("Uint8Array").New(data.Get("buffer")))
	glyphs[key] = img
	return img
}

// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package images

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"path"
	"strings"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

// The drawing programs' files a storyboard frame may be, beside the
// plain pictures: SVG (Inkscape), drawn here; Krita's .kra, by the
// picture it keeps of itself (mergedimage.png); Synfig's .sif/.sifz and
// a Lottie .json (Glaxnimate), drawn by the program's own command line
// when Render has one (the desktop's; else ErrNoRenderer). Their Picture
// keeps Data as the file's PNG, so an export reads it as any picture.

// More reasons a picture is not shown.
var (
	ErrNoRenderer = errors.New("drawn by its program, which is not installed")
	ErrRendering  = errors.New("being drawn")
)

// Kind is a drawing program's file kind ("" a plain picture).
type Kind string

const (
	KindSVG    Kind = "svg"
	KindKrita  Kind = "kra"
	KindSynfig Kind = "sif"
	KindLottie Kind = "lottie"
)

// KindOf is the kind of file data is, by its name (and, for a .json, its
// keys: a Lottie animation's).
func KindOf(name string, data []byte) Kind {
	switch strings.ToLower(path.Ext(strings.SplitN(name, "?", 2)[0])) {
	case ".svg":
		return KindSVG
	case ".kra":
		return KindKrita
	case ".sif", ".sifz":
		return KindSynfig
	case ".json", ".lottie":
		if data == nil || IsLottie(data) {
			return KindLottie
		}
		return ""
	}
	if isSVG(data, name) {
		return KindSVG
	}
	return ""
}

// IsLottie tells a Lottie animation's JSON: a frame rate, its in and out
// points and layers.
func IsLottie(data []byte) bool {
	var l struct {
		Fr     *float64          `json:"fr"`
		Op     *float64          `json:"op"`
		Layers []json.RawMessage `json:"layers"`
	}
	return json.Unmarshal(data, &l) == nil && l.Fr != nil && l.Op != nil && l.Layers != nil
}

// Render draws a Synfig or Lottie file (data, named name) as a picture,
// set by the desktop to its programs' command lines; nil: none.
var Render func(kind Kind, name string, data []byte) (image.Image, error)

// decodeKind draws a drawing program's file.
func decodeKind(k Kind, data []byte, name string) (image.Image, error) {
	switch k {
	case KindSVG:
		return DrawSVG(data, 1920)
	case KindKrita:
		return kritaPicture(data)
	case KindSynfig, KindLottie:
		if Render == nil {
			return nil, ErrNoRenderer
		}
		return Render(k, name, data)
	}
	return nil, ErrFormat
}

// DrawSVG draws an SVG at its own size, the longer side at most max
// pixels.
func DrawSVG(data []byte, maxSide int) (img image.Image, err error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data))
	if err != nil {
		return nil, ErrFormat
	}
	vw, vh := icon.ViewBox.W, icon.ViewBox.H
	if vw <= 0 || vh <= 0 {
		return nil, ErrFormat
	}
	s := min(1, float64(maxSide)/max(vw, vh))
	w, h := max(1, int(vw*s+0.5)), max(1, int(vh*s+0.5))
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	defer func() {
		if recover() != nil {
			img, err = nil, ErrFormat
		}
	}()
	icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)
	out := image.NewNRGBA(rgba.Bounds())
	draw.Draw(out, out.Bounds(), rgba, image.Point{}, draw.Src)
	return out, nil
}

// kritaPicture is the picture a .kra keeps of itself (Krita writes the
// flattened image as mergedimage.png, its thumbnail as preview.png).
func kritaPicture(data []byte) (image.Image, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrFormat
	}
	for _, name := range []string{"mergedimage.png", "preview.png"} {
		if f, err := z.Open(name); err == nil {
			img, err := png.Decode(f)
			f.Close()
			if err == nil {
				return img, nil
			}
		}
	}
	return nil, ErrFormat
}

// Unzipped is a .sifz's XML (gzipped), other data as it is.
func Unzipped(data []byte) []byte {
	if len(data) > 2 && data[0] == 0x1f && data[1] == 0x8b {
		if r, err := gzip.NewReader(bytes.NewReader(data)); err == nil {
			if out, err := io.ReadAll(io.LimitReader(r, 64<<20)); err == nil {
				return out
			}
		}
	}
	return data
}

// encodePNG is a picture's PNG.
func encodePNG(img image.Image) []byte {
	var b bytes.Buffer
	if png.Encode(&b, img) != nil {
		return nil
	}
	return b.Bytes()
}

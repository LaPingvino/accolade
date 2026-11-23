package prose

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"path"
	"strings"
)

// Pictures in Word and ODT: each picture read once, its file put in the
// document (word/media/, Pictures/) as PNG or JPEG — a JPEG and a PNG as
// they are, a GIF or WebP as a PNG —, placed where the text has it: a
// block picture on a line of its own, centred, as wide as the text at
// most (never larger than it is, at 96 dpi), its caption under it; one
// in a line as high as the line. One that cannot be read is its alt
// text in brackets, as in the PDF.

// embedded is a picture put in a document.
type embedded struct {
	n    int // from 1
	data []byte
	ext  string // "png", "jpeg"
	w, h int    // in pixels
}

// embedder reads a document's pictures for Word or ODT.
type embedder struct {
	doc  *Doc
	list []*embedded
	by   map[*Image]*embedded
	src  map[string]*embedded
	none map[*Image]bool
}

func newEmbedder(d *Doc) *embedder {
	return &embedder{doc: d, by: map[*Image]*embedded{}, src: map[string]*embedded{}, none: map[*Image]bool{}}
}

// get is the picture img names, put in the document (nil: it cannot be
// read).
func (e *embedder) get(img *Image) *embedded {
	if em, ok := e.by[img]; ok {
		return em
	}
	if e.none[img] {
		return nil
	}
	if img.Data == nil {
		if em, ok := e.src[img.Src]; ok {
			e.by[img] = em
			return em
		}
	}
	p := e.doc.picture(img)
	if p.Err != nil || p.Image == nil {
		e.none[img] = true
		return nil
	}
	b := p.Image.Bounds()
	em := &embedded{n: len(e.list) + 1, data: p.Data, w: b.Dx(), h: b.Dy()}
	switch {
	case bytes.HasPrefix(p.Data, []byte("\xff\xd8")):
		em.ext = "jpeg"
	case bytes.HasPrefix(p.Data, []byte("\x89PNG")):
		em.ext = "png"
	default:
		var buf bytes.Buffer
		if err := png.Encode(&buf, flatImage(p.Image)); err != nil {
			e.none[img] = true
			return nil
		}
		em.data, em.ext = buf.Bytes(), "png"
	}
	e.list = append(e.list, em)
	e.by[img] = em
	if img.Data == nil {
		e.src[img.Src] = em
	}
	return em
}

// flatImage is img as an image PNG writes (its own if it is one).
func flatImage(img image.Image) image.Image {
	switch img.(type) {
	case *image.NRGBA, *image.RGBA, *image.Gray, *image.Paletted:
		return img
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

// blockSize is a block picture's size in points: the text's width at
// most, never larger than at 96 dpi.
func (e *embedder) blockSize(em *embedded) (w, h float64) {
	pw, _ := paperSize(e.doc.Paper)
	text := (pw - 2) * 72 // an inch's margins
	w = math.Min(float64(em.w)*0.75, text)
	return w, w * float64(em.h) / float64(em.w)
}

// inlineSize is a picture's size in a line of text, in points.
func inlineSize(em *embedded) (w, h float64) {
	h = textSize * 1.15
	return h * float64(em.w) / float64(em.h), h
}

// caption is a block picture's caption ("" none).
func (d *Doc) caption(img *Image) string {
	if img.Title == "" && d.AltCaptions {
		return img.Alt
	}
	return img.Title
}

// altText is what a picture that cannot be read shows: its alt text in
// brackets.
func altText(img *Image) string {
	alt := strings.TrimSpace(img.Alt)
	if alt == "" {
		alt = "picture"
	}
	return "[" + alt + "]"
}

// docxDrawing is a picture as Word places it in a run, w × h points,
// from the relation rel.
func docxDrawing(em *embedded, rel string, w, h float64, alt string) string {
	cx, cy := int(w*12700), int(h*12700) // EMUs
	name := fmt.Sprintf("image%d.%s", em.n, em.ext)
	return fmt.Sprintf(`<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/>`+
		`<wp:docPr id="%d" name="Picture %d" descr="%s"/><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>`+
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic>`+
		`<pic:nvPicPr><pic:cNvPr id="%d" name="%s" descr="%s"/><pic:cNvPicPr/></pic:nvPicPr>`+
		`<pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>`+
		`</pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`,
		cx, cy, em.n, em.n, xmlAttr(alt), em.n, name, xmlAttr(alt), rel, cx, cy)
}

// odtFrame is a picture as ODT places it in a paragraph, w × h points.
func odtFrame(em *embedded, w, h float64, alt string) string {
	return fmt.Sprintf(`<draw:frame draw:style-name="Fr_pic" draw:name="Picture %d" text:anchor-type="as-char" svg:width="%.2fpt" svg:height="%.2fpt" draw:z-index="0">`+
		`<draw:image xlink:href="Pictures/image%d.%s" xlink:type="simple" xlink:show="embed" xlink:actuate="onLoad"/>`+
		`<svg:desc>%s</svg:desc></draw:frame>`, em.n, w, h, em.n, em.ext, xmlEsc(alt))
}

// extOf is the extension a picture's file has by its bytes (".png" for
// one it does not know).
func extOf(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\xff\xd8")):
		return ".jpg"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return ".gif"
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp"
	case bytes.Contains(data[:min(len(data), 512)], []byte("<svg")):
		return ".svg"
	}
	return ".png"
}

// EachImage calls f with each picture in the document, notes and tables
// too.
func (d *Doc) EachImage(f func(*Image)) {
	var spans func([]Span)
	var blocks func([]*Block)
	spans = func(ss []Span) {
		for _, s := range ss {
			if s.Image != nil {
				f(s.Image)
			}
			blocks(s.Note)
		}
	}
	blocks = func(bs []*Block) {
		for _, b := range bs {
			spans(b.Spans)
			for _, r := range b.Rows {
				for _, c := range r {
					spans(c)
				}
			}
			blocks(b.Children)
		}
	}
	blocks(d.Blocks)
}

// Unsaved is how many of the document's pictures are an import's not
// saved (Markdown leaves them out, a comment in their place).
func (d *Doc) Unsaved() int {
	n := 0
	d.EachImage(func(img *Image) {
		if img.Data != nil {
			n++
		}
	})
	return n
}

// SaveImages saves the pictures an import brought (their Data): save
// puts a file named as the picture's Src suggests and gives the name
// the Markdown names it by (assets/image1.png), and the picture then
// names that. A picture that cannot be saved keeps its Data (Word, ODT
// and the PDF still have it), the first error given back.
func (d *Doc) SaveImages(save func(name string, data []byte) (string, error)) error {
	var first error
	saved := map[*Image]bool{}
	d.EachImage(func(img *Image) {
		if img.Data == nil || saved[img] {
			return
		}
		saved[img] = true
		name := path.Base(strings.ReplaceAll(img.Src, "\\", "/"))
		if name == "." || name == "/" || name == "" {
			name = "image" + extOf(img.Data)
		}
		src, err := save(name, img.Data)
		if err != nil {
			if first == nil {
				first = err
			}
			return
		}
		img.Src, img.Data = src, nil
	})
	return first
}

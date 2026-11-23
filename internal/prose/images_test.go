package prose

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/images"
)

// testPNG is a w × h picture in two colours, as a PNG.
func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{30, 90, 160, 255}
			if x < w/2 {
				c = color.NRGBA{220, 140, 30, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// picturesDoc is a document with a block picture (titled), one in a
// line, one missing, one on the web and one as a data: URI, in a folder.
func picturesDoc(t *testing.T) (string, *images.Loader) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "assets", "harbour.png"), testPNG(t, 800, 400), 0o644)
	os.WriteFile(filepath.Join(dir, "dot.png"), testPNG(t, 20, 20), 0o644)
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(testPNG(t, 60, 30))
	md := "# Pictures\n\n" +
		"![The harbour at dawn](assets/harbour.png \"The harbour, 1920\")\n\n" +
		"A dot ![dot](dot.png) in a line.\n\n" +
		"![A lost one](nowhere/lost.png)\n\n" +
		"![On the web](https://example.com/web.png)\n\n" +
		"![Inline data](" + data + ")\n"
	return md, images.New(images.Dir(dir))
}

// The Markdown's pictures are in the model, written back as they were.
func TestImagesInModel(t *testing.T) {
	md, _ := picturesDoc(t)
	d := FromMarkdown(md)
	block := d.Blocks[1].Picture()
	if block == nil || block.Src != "assets/harbour.png" || block.Alt != "The harbour at dawn" || block.Title != "The harbour, 1920" {
		t.Fatalf("block picture %+v", block)
	}
	if d.Blocks[2].Picture() != nil {
		t.Error("a picture in a line taken as a block")
	}
	again := FromMarkdown(d.Markdown())
	if got, want := again.Markdown(), d.Markdown(); got != want {
		t.Errorf("not kept through Markdown:\n%s\nthen\n%s", want, got)
	}
	if !strings.Contains(d.Markdown(), `![The harbour at dawn](assets/harbour.png "The harbour, 1920")`) {
		t.Errorf("written as %q", d.Markdown())
	}
	if !strings.Contains(d.Plain(), "The harbour at dawn") {
		t.Error("plain text without the alt text")
	}
}

// The PDF has the pictures: the block one at the text's width (it is
// larger), under the heading, its caption under it; the line's as high
// as its line; the missing and the remote one as placeholders with their
// alt text and where they are — the remote one never fetched.
func TestImagesInPDF(t *testing.T) {
	md, loader := picturesDoc(t)
	d := FromMarkdown(md)
	d.Images = loader
	var b bytes.Buffer
	drawing, err := d.WritePDF(&b, PDFFonts{})
	if err != nil {
		t.Fatal(err)
	}
	var pics []pdf.Image
	var text strings.Builder
	for _, p := range drawing.Pages {
		for _, m := range p.Marks {
			switch m := m.(type) {
			case pdf.Image:
				pics = append(pics, m)
			case pdf.Text:
				text.WriteString(m.Text + " ")
			}
		}
	}
	if len(pics) != 3 {
		t.Fatalf("%d pictures drawn, want 3 (block, inline, data)", len(pics))
	}
	pageW := drawing.Pages[0].Width
	if w := pics[0].W; w < pageW-2*72-1 || w > pageW-2*72+1 {
		t.Errorf("the block picture %.0fpt wide, not the text's width", w)
	}
	if r := pics[0].W / pics[0].H; r < 1.99 || r > 2.01 {
		t.Errorf("its shape changed: %.2f", r)
	}
	if h := pics[1].H; h > textSize*1.5 {
		t.Errorf("the picture in a line %.0fpt high", h)
	}
	if w := pics[2].W; w > 60*0.75+0.01 {
		t.Errorf("a small picture made larger: %.1fpt", w)
	}
	all := strings.Join(strings.Fields(text.String()), " ")
	for _, want := range []string{"The harbour, 1920", "A lost one", "nowhere/lost.png (not found)", "On the web", "https://example.com/web.png (on the web, not fetched)"} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in %q", want, all)
		}
	}
	// it paints
	img := pdf.NewRasterizer(&drawing).Page(0, 1)
	if c := img.At(int(pics[0].X+10), int(pics[0].Y+10)); c == (color.RGBA{255, 255, 255, 255}) {
		t.Error("the picture not painted")
	}
}

// A picture too tall for a page is made smaller to fit one.
func TestTallImageFitsAPage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "tall.png"), testPNG(t, 300, 3000), 0o644)
	d := FromMarkdown("Text.\n\n![tall](tall.png)\n")
	d.Images = images.New(images.Dir(dir))
	var b bytes.Buffer
	drawing, err := d.WritePDF(&b, PDFFonts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range drawing.Pages {
		for _, m := range p.Marks {
			if im, ok := m.(pdf.Image); ok && (im.Y < 71 || im.Y+im.H > p.Height-71) {
				t.Errorf("the picture from %.0f to %.0f, past the margins", im.Y, im.Y+im.H)
			}
		}
	}
}

// Word and ODT carry the pictures: their files in the document, placed
// where the text has them, read back with their alt text; an import's
// pictures saved where the caller puts them, the Markdown naming them.
func TestImagesInWordAndODT(t *testing.T) {
	md, loader := picturesDoc(t)
	for _, kind := range []string{"docx", "odt"} {
		t.Run(kind, func(t *testing.T) {
			d := FromMarkdown(md)
			d.Images = loader
			var b bytes.Buffer
			var err error
			if kind == "docx" {
				err = d.WriteDOCX(&b)
			} else {
				err = d.WriteODT(&b)
			}
			if err != nil {
				t.Fatal(err)
			}
			z, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
			if err != nil {
				t.Fatal(err)
			}
			n := 0
			for _, f := range z.File {
				if strings.HasPrefix(f.Name, "word/media/") || strings.HasPrefix(f.Name, "Pictures/") {
					n++
				}
			}
			if n != 3 {
				t.Errorf("%d pictures in the file, want 3", n)
			}
			var back *Doc
			if kind == "docx" {
				back, err = ReadDOCX(bytes.NewReader(b.Bytes()), int64(b.Len()))
			} else {
				back, err = ReadODT(bytes.NewReader(b.Bytes()), int64(b.Len()))
			}
			if err != nil {
				t.Fatal(err)
			}
			var got []*Image
			back.EachImage(func(img *Image) { got = append(got, img) })
			if len(got) != 3 || got[0].Alt != "The harbour at dawn" || got[0].Data == nil || back.Dropped != 0 {
				t.Fatalf("read back %d pictures (%+v), %d dropped", len(got), got, back.Dropped)
			}
			if text := back.Plain(); !strings.Contains(text, "[A lost one]") || !strings.Contains(text, "The harbour, 1920") {
				t.Errorf("the missing picture's alt text or the caption not kept: %q", text)
			}
			saved := map[string][]byte{}
			if err := back.SaveImages(func(name string, data []byte) (string, error) {
				saved[name] = data
				return "assets/" + name, nil
			}); err != nil {
				t.Fatal(err)
			}
			out := back.Markdown()
			if len(saved) != 3 || !strings.Contains(out, "](assets/image1.png") {
				t.Errorf("saved %d, Markdown %q", len(saved), out)
			}
			if p, _ := png.Decode(bytes.NewReader(saved["image1.png"])); p == nil || p.Bounds().Dx() != 800 {
				t.Error("the first picture not saved as it was")
			}
		})
	}
}

// A web page's pictures: a data: URI's saved as a file, an address kept.
func TestImagesFromHTML(t *testing.T) {
	data := base64.StdEncoding.EncodeToString(testPNG(t, 10, 10))
	d, err := ReadHTML(strings.NewReader(`<p><img src="data:image/png;base64,` + data + `" alt="dot"></p><p><img src="pics/a b.png" alt="beside" title="T"></p>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SaveImages(func(name string, _ []byte) (string, error) { return "assets/" + name, nil }); err != nil {
		t.Fatal(err)
	}
	if got, want := d.Markdown(), "![dot](assets/image1.png)\n\n![beside](<pics/a b.png> \"T\")\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

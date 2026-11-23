package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/images"
)

// testPicture is a w × h PNG in two colours.
func testPicture(t testing.TB, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{40, 90, 170, 255}
			if y < h/2 {
				c = color.NRGBA{230, 150, 40, 255}
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

// picturesMarkdown is a document with a picture beside it, one in its
// assets/, one missing and one on the web.
const picturesMarkdown = "# Harbour\n\nThe boats came in.\n\n![The harbour](harbour.png \"Dawn at the harbour\")\n\n" +
	"A gull ![gull](assets/gull.png) in a line.\n\n![Lost](lost.png)\n\n![Web](https://example.com/w.png)\n"

func writePicturesDoc(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	os.WriteFile(filepath.Join(dir, "harbour.png"), testPicture(t, 600, 300), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "gull.png"), testPicture(t, 30, 30), 0o644)
	doc := filepath.Join(dir, "harbour.md")
	os.WriteFile(doc, []byte(picturesMarkdown), 0o644)
	return doc
}

// The Live Preview shows the document's pictures, read beside it, in
// their order, the missing and the remote one as placeholders: Fyne's
// own image segments (which would read the working folder, or fetch)
// are all replaced.
func TestPreviewPictures(t *testing.T) {
	doc := writePicturesDoc(t)
	l := images.New(images.Dir(filepath.Dir(doc)))
	segs, _ := markdownSegments(picturesMarkdown, -1, l)
	var pics []*pictureSegment
	var walk func([]widget.RichTextSegment)
	walk = func(ss []widget.RichTextSegment) {
		for _, s := range ss {
			switch s := s.(type) {
			case *widget.ImageSegment:
				t.Errorf("a Fyne image segment left: %v", s.Source)
			case *pictureSegment:
				pics = append(pics, s)
			case *widget.ParagraphSegment:
				walk(s.Texts)
			}
		}
	}
	walk(segs)
	if len(pics) != 4 {
		t.Fatalf("%d pictures, want 4", len(pics))
	}
	if !pics[0].pic.OK() || pics[0].img.title != "Dawn at the harbour" || !pics[1].pic.OK() {
		t.Errorf("the pictures beside and in assets/ not shown: %v %v", pics[0].pic.Err, pics[1].pic.Err)
	}
	if pics[2].pic.OK() || pics[3].pic.OK() || !strings.Contains(images.Why(pics[3].pic.Err), "not fetched") {
		t.Error("a missing or remote picture shown")
	}
	for _, p := range pics {
		if p.Visual() == nil {
			t.Error("no visual")
		}
	}
}

// The document opened: its PDF and its page view have the pictures;
// Word too; the page a browser prints stands alone (data: URIs), the
// remote picture a link.
func TestDocumentPictures(t *testing.T) {
	doc := writePicturesDoc(t)
	w := newTestWindow(t, "")
	if err := w.LoadFile(doc); err != nil {
		t.Fatal(err)
	}
	p := w.markdownPageSetup()
	_, d, err := markdownPDF(w.textEditor.Text(), "Harbour", p, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, pg := range d.Pages {
		for _, m := range pg.Marks {
			if _, ok := m.(pdf.Image); ok {
				n++
			}
		}
	}
	if n != 2 {
		t.Errorf("%d pictures in the PDF, want 2", n)
	}
	out := filepath.Join(t.TempDir(), "harbour.docx")
	if err := exportMarkdownWith(w.textEditor.Text(), "Harbour", out, "docx", p, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(out); !bytes.Contains(data, []byte("word/media/image1.png")) {
		t.Error("Word has no picture")
	}
	page, _ := markdownHTMLWith(w.textEditor.Text(), "Harbour", p, nil)
	alone := string(inlinePictures(page, p.pics))
	if strings.Count(alone, `src="data:image/png;base64,`) != 2 || !strings.Contains(alone, `<a href="https://example.com/w.png">Web</a>`) ||
		!strings.Contains(alone, "lost.png") {
		t.Errorf("the page to print: %s", alone)
	}
	if !strings.Contains(string(page), `src="harbour.png"`) {
		t.Error("the exported web page does not keep the tag")
	}
}

// An imported Word document's pictures are saved beside it, in
// Name.assets/, the Markdown linking them there, and shown before the
// document is saved.
func TestImportKeepsPictures(t *testing.T) {
	doc := writePicturesDoc(t)
	w := newTestWindow(t, "")
	if err := w.LoadFile(doc); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	word := filepath.Join(dir, "Story.docx")
	if err := exportMarkdownWith(w.textEditor.Text(), "Harbour", word, "docx", w.markdownPageSetup(), nil); err != nil {
		t.Fatal(err)
	}
	md, dropped, _, err := toMarkdownSaving(word, importedPictures(word))
	if err != nil {
		t.Fatal(err)
	}
	if dropped != 0 || !strings.Contains(md, "](Story.assets/image1.png") || !strings.Contains(md, "](Story.assets/image2.png)") {
		t.Errorf("%d dropped, Markdown %q", dropped, md)
	}
	if _, err := os.Stat(filepath.Join(dir, "Story.assets", "image1.png")); err != nil {
		t.Error(err)
	}
	w.showImportedMarkdown(word, md, "a Word document", dropped)
	if p := w.pictures().Load("Story.assets/image1.png"); !p.OK() {
		t.Errorf("the unsaved import's picture not read: %v", p.Err)
	}
}

// A linked picture and another after it: each segment its own picture.
func TestPreviewPicturesMatched(t *testing.T) {
	segs := richMarkdown("[![a](a.png)](https://example.com)\n\nText.\n\n![b](b.png)\n\n| x |\n| --- |\n| ![c](c.png) |\n", nil)
	var srcs []string
	var walk func([]widget.RichTextSegment)
	walk = func(ss []widget.RichTextSegment) {
		for _, s := range ss {
			switch s := s.(type) {
			case *pictureSegment:
				srcs = append(srcs, s.img.src)
			case *widget.ParagraphSegment:
				walk(s.Texts)
			case *widget.TableSegment:
				for _, r := range s.Rows {
					for _, c := range r {
						walk(c)
					}
				}
			case *widget.ImageSegment:
				t.Errorf("a Fyne image segment left: %v", s.Source)
			}
		}
	}
	walk(segs)
	// (Fyne makes a linked picture its link)
	for i, want := range []string{"b.png", "c.png"} {
		if i >= len(srcs) || srcs[i] != want {
			t.Errorf("pictures %v", srcs)
			break
		}
	}
}

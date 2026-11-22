package prose

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
)

// more is what the story has not got: code, a break, a table, a line
// break, struck and code text, a long word.
const more = "\n## Three\n\n```\nfunc light() {\n\treturn \"on\"\n}\n```\n\n* * *\n\n" +
	"| Room | Things |\n| --- | --- |\n| Kitchen | a kettle, two cups |\n| Lamp room | the **lens** |\n\n" +
	"A line,  \nbroken. Some ~~struck~~ and `code` text, and Llanfairpwllgwyngyllgogerychwyrndrobwllllantysiliogogogoch.\n"

// The PDF of a document: its text in order, a page for each of the
// drawing's, the title page first.
func TestWritePDF(t *testing.T) {
	d := FromMarkdown(story + more)
	d.TitlePage = []string{"The Lighthouse Keeper", "A. Writer"}
	d.Paper = "a5"
	var b bytes.Buffer
	drawing, err := d.WritePDF(&b, PDFFonts{})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(file, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != len(drawing.Pages) || len(pages) < 3 {
		t.Fatalf("%d pages in the PDF, %d drawn", len(pages), len(drawing.Pages))
	}
	if w := drawing.Pages[0].Width; w < 415 || w > 422 {
		t.Errorf("A5's width %v", w)
	}
	var text strings.Builder
	for _, p := range pages {
		for _, l := range p {
			text.WriteString(l.Text + "\n")
		}
	}
	all := strings.Join(strings.Fields(text.String()), " ")
	for _, want := range []string{
		"The Lighthouse Keeper A. Writer", "The wind came in from the west, and the rain followed it", "Nobody went out that night.",
		"• a brass compass", "1. read once", "The sea gives back what it takes", "func light() {", `return "on"`,
		"Room Things", "Kitchen a kettle, two cups", "A line, broken.", "1. Not even the fishermen.",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("no %q in\n%s", want, all)
		}
	}
	if dir := os.Getenv("SHOT_DIR"); dir != "" {
		r := pdf.NewRasterizer(&drawing)
		for i := range drawing.Pages {
			f, _ := os.Create(filepath.Join(dir, "prose-page-"+string(rune('1'+i))+".png"))
			png.Encode(f, r.Page(i, 1.5))
			f.Close()
		}
	}
}

// What the PDF cannot print yet: text read right to left, characters
// none of its fonts has.
func TestPDFMissing(t *testing.T) {
	if m := FromMarkdown(story).PDFMissing(PDFFonts{}); m != "" {
		t.Errorf("the story: %q", m)
	}
	if m := FromMarkdown("שָׁלוֹם").PDFMissing(PDFFonts{}); !strings.Contains(m, "right to left") {
		t.Errorf("Hebrew: %q", m)
	}
	if m := FromMarkdown("日本語").PDFMissing(PDFFonts{}); !strings.Contains(m, "日") {
		t.Errorf("Japanese: %q", m)
	}
}

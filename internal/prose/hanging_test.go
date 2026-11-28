// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package prose

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/pdf"
)

// A paragraph starting with HangingMark (a bibliography's entry) is set
// with a hanging indent: in the PDF its lines after the first start
// further in; Word and ODT get the indent as a paragraph property.
func TestHangingParagraph(t *testing.T) {
	long := strings.Repeat("word ", 60)
	d := FromMarkdown("Plain " + long + "\n\n" + HangingMark + "*Title* Hanging " + long + "\n")
	if len(d.Blocks) != 2 || d.Blocks[0].Hanging || !d.Blocks[1].Hanging {
		t.Fatalf("blocks %+v", d.Blocks)
	}
	if s := d.Blocks[1].Spans[0]; !s.Italic || s.Text != "Title" {
		t.Errorf("the mark left a span: %+v", d.Blocks[1].Spans)
	}
	_, drawing := written(t, d)
	// the x of each line's first word, per line (y)
	starts := map[float64]float64{}
	var ys []float64
	for _, m := range drawing.Pages[0].Marks {
		if tx, ok := m.(pdf.Text); ok && strings.TrimSpace(tx.Text) != "" {
			if x, ok := starts[tx.Y]; !ok || tx.X < x {
				if !ok {
					ys = append(ys, tx.Y)
				}
				starts[tx.Y] = tx.X
			}
		}
	}
	_, hy := textAt(drawing, "Title")
	var plainRest, hangFirst, hangRest float64
	for i, y := range ys {
		switch {
		case y < hy && i > 0:
			plainRest = starts[y]
		case y == hy:
			hangFirst = starts[y]
		case y > hy:
			hangRest = starts[y]
		}
	}
	if hangFirst != starts[ys[0]] || plainRest != hangFirst || hangRest <= hangFirst+30 {
		t.Errorf("indents: plain %v/%v, hanging %v/%v", starts[ys[0]], plainRest, hangFirst, hangRest)
	}
	var b bytes.Buffer
	if err := d.WriteDOCX(&b); err != nil {
		t.Fatal(err)
	}
	if x := zipEntry(t, b.Bytes(), "word/document.xml"); strings.Count(x, `w:hanging="720"`) != 1 {
		t.Errorf("docx: no hanging indent")
	}
	b.Reset()
	if err := d.WriteODT(&b); err != nil {
		t.Fatal(err)
	}
	if x := zipEntry(t, b.Bytes(), "content.xml"); strings.Count(x, `text:style-name="P_hanging"`) != 1 || !strings.Contains(x, `fo:text-indent="-0.5in"`) {
		t.Errorf("odt: no hanging indent")
	}
}

// zipEntry is the text of a file in a zip.
func zipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range z.File {
		if f.Name == name {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			b, _ := io.ReadAll(r)
			return string(b)
		}
	}
	t.Fatalf("no %s", name)
	return ""
}

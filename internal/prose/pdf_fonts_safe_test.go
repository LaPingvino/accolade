package prose

import (
	"bytes"
	"crypto/sha256"
	"testing"

	lexfont "github.com/LaPingvino/lexington/font"
	"golang.org/x/image/font/gofont/goregular"
)

// Writing a Markdown PDF leaves the fonts it borrows untouched: the
// interface's own (Fyne's) and the embedded Courier Badi.
func TestPDFLeavesFontsAlone(t *testing.T) {
	sum := func(b []byte) [32]byte { return sha256.Sum256(b[:cap(b)]) }
	mono := lexfont.GetFont(lexfont.CourierBadiName, "")
	before, text := sum(mono), sum(goregular.TTF)
	d := FromMarkdown("# Title\n\nSome *text* and `code`, a list:\n\n- one\n- two\n")
	for range 2 {
		var b bytes.Buffer
		if _, err := d.WritePDF(&b, PDFFonts{}); err != nil {
			t.Fatal(err)
		}
	}
	if sum(mono) != before {
		t.Error("Courier Badi changed while writing PDFs")
	}
	if sum(goregular.TTF) != text {
		t.Error("the text font changed while writing PDFs")
	}
}

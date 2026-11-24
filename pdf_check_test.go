package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/pdf"
	"github.com/LaPingvino/lexington/pdfin"
)

func TestPDFWarning(t *testing.T) {
	warn := func(text string) string {
		return cli.PDFWarning(fountain.Parse(sceneHeaders("en"), strings.NewReader(text)))
	}
	if w := warn("INT. HOUSE - DAY\n\nTiếng Việt, Esperanto ĉ.\n"); w != "" {
		t.Errorf("Latin script warned: %s", w)
	}
	if w := warn("פנים. סדנה. יום.\n\nבחור צעיר.\n"); w != "" { // Courier Badi 1.100, right to left since Lexington 1.21.0
		t.Errorf("Hebrew warned: %s", w)
	}
	if w := warn("داخلي. ورشة. نهار.\n\nشاب صغير.\n"); w != "" { // shaped by Lexington since the Hekaya item
		t.Errorf("Arabic warned: %s", w)
	}
	if w := warn("INT. HOUSE - DAY\n\nܫܠܡܐ\n"); w == "" { // Syriac joins, and is not shaped
		t.Error("Syriac: no warning")
	}
}

// A Chinese script's PDF is Lexington's own, not a browser's: its
// characters from a font on the computer, its text the script's.
func TestChinesePDF(t *testing.T) {
	const script = "内景 咖啡馆 夜\n\n张三坐在窗边。\n"
	if m, _ := pdf.Unsupported(fountain.Parse(nil, strings.NewReader(script))); len(m) > 0 {
		t.Skipf("no font on this computer has %s", string(m))
	}
	out := filepath.Join(t.TempDir(), "zh.pdf")
	job := exportJob{scenes: sceneHeaders("zh")}
	if err := exportAs(script, out, "pdf", job); err != nil {
		t.Fatal(err)
	}
	pages, err := pdfin.Pages(out)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, p := range pages {
		for _, l := range p {
			text.WriteString(l.Text + "\n")
		}
	}
	if !strings.Contains(text.String(), "张三坐在窗边") {
		t.Errorf("the PDF's text: %q", text.String())
	}
}

// Make the PDF with the Browser (a script the PDF cannot shape): a
// Chromium browser on the computer prints Lexington's HTML.
func TestBrowserPDF(t *testing.T) {
	if localBrowser() == "" || testing.Short() {
		t.Skip("no Chromium browser")
	}
	dir := t.TempDir()
	page, out := filepath.Join(dir, "syr.html"), filepath.Join(dir, "syr.pdf")
	if err := exportAs("INT. HOUSE - DAY\n\nܫܠܡܐ\n", page, "html", exportJob{}); err != nil {
		t.Fatal(err)
	}
	if err := browserPDF(localBrowser(), page, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("no PDF: %v", err)
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
	"github.com/LaPingvino/lexington/fountain"
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
	if w := warn("داخلي. ورشة. نهار.\n\nشاب صغير.\n"); w == "" { // needs shaping (the Hekaya item)
		t.Error("Arabic: no warning")
	}
}

// An Arabic script's PDF is made by a browser on the computer, when there
// is one: its letters joined, which the PDF writer does not do yet.
func TestBrowserPDF(t *testing.T) {
	if localBrowser() == "" || testing.Short() {
		t.Skip("no Chromium browser")
	}
	out := filepath.Join(t.TempDir(), "ar.pdf")
	job := exportJob{scenes: sceneHeaders("ar")}
	if err := exportAs("داخلي. ورشة. نهار.\n\nشاب صغير يمشي في الشارع.\n", out, "pdf", job); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("no PDF: %v", err)
	}
}

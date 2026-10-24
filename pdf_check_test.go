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
	if w := warn("פנים. סדנה. יום.\n\nבחור צעיר.\n"); !strings.Contains(w, "boxes") || !strings.Contains(w, "right-to-left") {
		t.Errorf("Hebrew: %q", w)
	}
}

// A Hebrew script's PDF is made by a browser on the computer, when there
// is one: its text is in it, not boxes.
func TestBrowserPDF(t *testing.T) {
	if localBrowser() == "" || testing.Short() {
		t.Skip("no Chromium browser")
	}
	out := filepath.Join(t.TempDir(), "he.pdf")
	job := exportJob{scenes: sceneHeaders("he")}
	if err := exportAs("פנים. סדנה. יום.\n\nבחור צעיר הולך ברחוב.\n", out, "pdf", job); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("no PDF: %v", err)
	}
}

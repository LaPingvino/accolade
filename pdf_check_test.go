package main

import (
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

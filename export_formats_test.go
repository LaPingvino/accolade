package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LaPingvino/lexington/cli"
)

// The Export dialog offers every format Lexington's command line has.
func TestExportOffersLexingtonsFormats(t *testing.T) {
	ids := map[string]bool{}
	for _, f := range exportFormatList() {
		ids[f.id] = true
	}
	for _, f := range cli.Formats {
		if !ids[f.Key] {
			t.Errorf("%s (%s) is not offered", f.Key, f.Name)
		}
	}
	for _, id := range []string{"text-fountain", "txt"} {
		if !ids[id] {
			t.Errorf("%s is not offered", id)
		}
	}
}

// Settings of earlier versions name formats by their old labels.
func TestOldExportFormatSettings(t *testing.T) {
	for old, id := range map[string]string{"PDF": "pdf", "DOCX": "docx", "ODT": "odt", "FDX": "fdx",
		"HTML": "html", "TXT": "txt", "Fountain": "text-fountain", "pdf": "pdf", "nonsense": "pdf"} {
		if got := exportFormatByLabel(old).id; got != id {
			t.Errorf("%q: %s, want %s", old, got, id)
		}
	}
	for _, f := range exportFormatList() {
		if exportFormatByLabel(f.label).id != f.id {
			t.Errorf("%s: its label %q is not its own", f.id, f.label)
		}
	}
}

// A Kindle book through the dialog's export, where Calibre is installed.
func TestExportKindle(t *testing.T) {
	f, _ := cli.FormatFor("azw3")
	if !f.Available() {
		t.Skip("Calibre is not installed")
	}
	w := newTestWindow(t, accoladeScript)
	ed := NewExportDialog(w)
	out := filepath.Join(t.TempDir(), "barn.azw3")
	if err := ed.performExportJob(accoladeScript, out, "azw3", ed.job()); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Errorf("no book: %v", err)
	}
}

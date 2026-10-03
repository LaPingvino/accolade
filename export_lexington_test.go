package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// accoladeScript is laid out the way Accolade's editor and title page
// dialog write scripts.
var accoladeScript = strings.Join([]string{
	"Title: The Barn",
	"Author: Jane Smith",
	"Contact:",
	"    Jane Smith",
	"    1 Writer's Lane",
	"",
	"FADE IN:",
	"",
	"INT. BARN - DAY",
	"",
	"Rain hammers the roof.",
	"",
	pad(CharacterIndent, "JOHN"),
	pad(ParentheticalIndent, "(beat)"),
	pad(DialogueIndent, "It's coming."),
	"",
	pad(TransitionIndent, "CUT TO:"),
	"",
}, "\n")

func TestParseForExportElementTypes(t *testing.T) {
	var got []string
	for _, l := range parseForExport(accoladeScript) {
		if l.Type != "empty" {
			got = append(got, string(l.Type)+"="+string(l.Contents))
		}
	}
	want := []string{
		"titlepage=", "Title=The Barn", "Author=Jane Smith", "metasection=",
		"Contact=Jane Smith", "Contact=1 Writer's Lane", "newpage=",
		"action=FADE IN:", "scene=INT. BARN - DAY", "action=Rain hammers the roof.",
		"speaker=JOHN", "paren=(beat)", "dialog=It's coming.", "trans=CUT TO:",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("elements:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestExportPDF(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "barn.pdf")
	if err := exportPDF(accoladeScript, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
		t.Errorf("not a PDF (%d bytes): %q", len(data), data[:min(len(data), 20)])
	}
	if text, err := exec.Command("pdftotext", "-layout", out, "-").Output(); err == nil {
		for _, s := range []string{"The Barn", "1 Writer's Lane", "FADE IN:", "INT. BARN - DAY", pad(26, "JOHN"), "(beat)", "It's coming.", "CUT TO:"} {
			if !strings.Contains(string(text), s) {
				t.Errorf("PDF text is missing %q:\n%s", s, text)
			}
		}
	} else {
		t.Log("pdftotext not available; PDF contents not checked")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("leftover files: %v", entries)
	}
}

func TestExportHTML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "barn.html")
	if err := exportHTML(accoladeScript, out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	page := string(data)
	for _, s := range []string{"The Barn", "Jane Smith", "1 Writer&#39;s Lane", "FADE IN:", "INT. BARN - DAY", "JOHN", "(beat)", "It&#39;s coming.", "CUT TO:"} {
		if !strings.Contains(page, s) {
			t.Errorf("HTML is missing %q", s)
		}
	}
}

func TestExportToMissingDirectoryFails(t *testing.T) {
	out := filepath.Join(t.TempDir(), "nope", "barn.pdf")
	if err := exportPDF(accoladeScript, out); err == nil {
		t.Error("expected an error writing into a missing directory")
	}
}

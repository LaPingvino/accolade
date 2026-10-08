package main

import (
	"archive/zip"
	"bytes"
	"github.com/LaPingvino/lexington/rules"
	"io"
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
	if err := exportPDF(accoladeScript, out, rules.Default); err != nil {
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
	if err := exportHTML(accoladeScript, out, rules.Default); err != nil {
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
	if err := exportPDF(accoladeScript, out, rules.Default); err == nil {
		t.Error("expected an error writing into a missing directory")
	}
}

// DOCX and ODT go through Lexington: zip documents with the script's
// styles, on the chosen paper.
func TestExportDocuments(t *testing.T) {
	m, _ := rules.GetPreset("musical")
	for _, c := range []struct{ format, part, want string }{
		{"docx", "word/document.xml", `<w:pgSz w:w="8395"`}, // A5
		{"odt", "styles.xml", `fo:page-width="5.830in"`},    // A5
	} {
		out := filepath.Join(t.TempDir(), "script."+c.format)
		if err := exportDocument(accoladeScript, out, c.format, m.Elements, m.Page); err != nil {
			t.Fatal(err)
		}
		z, err := zip.OpenReader(out)
		if err != nil {
			t.Fatalf("%s: %v", c.format, err)
		}
		found := false
		for _, f := range z.File {
			if f.Name == c.part {
				r, _ := f.Open()
				b, _ := io.ReadAll(r)
				r.Close()
				found = strings.Contains(string(b), c.want)
			}
		}
		z.Close()
		if !found {
			t.Errorf("%s: %s lacks %s", c.format, c.part, c.want)
		}
	}
}

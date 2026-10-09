package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
	"io"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
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
	if err := exportPDF(accoladeScript, out, exportJob{elements: rules.Default}); err != nil {
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
	if err := exportHTML(accoladeScript, out, exportJob{elements: rules.Default}); err != nil {
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
	if err := exportPDF(accoladeScript, out, exportJob{elements: rules.Default}); err == nil {
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
		if err := exportDocument(accoladeScript, out, c.format, exportJob{elements: m.Elements, page: m.Page}); err != nil {
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

// The Export dialog's script options: without the title page, scenes
// numbered after the last given number.
func TestExportJobScript(t *testing.T) {
	text := "Title: T\nAuthor: A\n\nINT. ONE - DAY\n\nHi.\n\nINT. TWO - DAY #7#\n\nHo.\n\nEXT. THREE - NIGHT\n"
	s := exportJob{omitTitlePage: true, numberScenes: true}.script(text)
	var scenes []string
	for _, l := range s {
		if l.Type == lex.TypeTitlePage || l.Type == "Title" {
			t.Errorf("title page left: %+v", l)
		}
		if l.Type == lex.TypeScene {
			scenes = append(scenes, l.Contents)
		}
	}
	want := []string{"INT. ONE - DAY #1#", "INT. TWO - DAY #7#", "EXT. THREE - NIGHT #8#"}
	if strings.Join(scenes, "|") != strings.Join(want, "|") {
		t.Errorf("scenes %q", scenes)
	}
	if s := (exportJob{}).script(text); s[0].Type != lex.TypeTitlePage {
		t.Error("the title page is kept by default")
	}
}

// A PDF opens as its script, imported: unsaved, with a .fountain name.
func TestOpenPDF(t *testing.T) {
	out := filepath.Join(t.TempDir(), "script.pdf")
	if err := exportPDF(accoladeScript, out, exportJob{elements: rules.Default}); err != nil {
		t.Fatal(err)
	}
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile(out); err != nil {
		t.Fatal(err)
	}
	text := waitForText(t, w)
	for _, want := range []string{"Title: The Barn", "Contact: Jane Smith\n    1 Writer's Lane", "INT. BARN - DAY",
		"Rain hammers the roof.", "\nJOHN\n(beat)\nIt's coming.", "CUT TO:"} {
		if !strings.Contains(text, want) {
			t.Errorf("imported script lacks %q:\n%s", want, text)
		}
	}
	if w.currentFile != "" || w.suggestedName != "script.fountain" || !w.hasChanges {
		t.Errorf("imported: file %q, name %q, changed %v", w.currentFile, w.suggestedName, w.hasChanges)
	}
}

// Dropping files opens them; a window with work in it is not replaced.
func TestDropOpens(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.fountain"), filepath.Join(dir, "b.fountain")
	os.WriteFile(a, []byte("INT. A - DAY\n"), 0o644)
	os.WriteFile(b, []byte("INT. B - DAY\n"), 0o644)
	w, app := newDialogTestWindow(t)
	w.onDropped(fyne.Position{}, []fyne.URI{storage.NewFileURI(a)})
	if w.textEditor.Text() != "INT. A - DAY\n" {
		t.Fatalf("dropped file not opened: %q", w.textEditor.Text())
	}
	w.onDropped(fyne.Position{}, []fyne.URI{storage.NewFileURI(b)})
	if w.textEditor.Text() != "INT. A - DAY\n" || len(app.windows) != 2 || app.windows[1].textEditor.Text() != "INT. B - DAY\n" {
		t.Errorf("second drop: this window %q, %d windows", w.textEditor.Text(), len(app.windows))
	}
}

// waitForText waits for a PDF import (in the background) to fill the
// editor.
func waitForText(t *testing.T, w *MainWindow) string {
	t.Helper()
	select {
	case <-w.imported:
	case <-time.After(2 * time.Minute):
		t.Fatal("the import did not finish")
	}
	var text string
	fyne.DoAndWait(func() { text = w.textEditor.Text() })
	return text
}

// A scanned script opens through OCR: tesseract or the built-in one.
func TestOpenScannedPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("OCR takes seconds a page")
	}
	w, _ := newDialogTestWindow(t)
	if err := w.LoadFile("testdata/scanned-tv-episode.pdf"); err != nil {
		t.Fatal(err)
	}
	text := waitForText(t, w)
	for _, want := range []string{"Title: BISCUITS", "INT. OFFICE KITCHEN - DAY", "\nANNA\nJust one left.", "BRAM ^"} {
		if !strings.Contains(text, want) {
			t.Errorf("OCRed script lacks %q:\n%s", want, text)
		}
	}
}

// The Export dialog starts with the format of the preferences, refuses
// the open script as its target, and the Fountain export is the script.
func TestExportDialogFormatAndTarget(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.settings.SetString("export-format", "DOCX")
	t.Cleanup(func() { w.settings.SetString("export-format", "PDF") })
	ed := NewExportDialog(w)
	if ed.formatSelect.Selected != "DOCX" {
		t.Errorf("opens on %q", ed.formatSelect.Selected)
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "film.fountain")
	text := "INT. A - DAY\n\nANNA\nHello.\n  \nStill Anna.\n"
	os.WriteFile(script, []byte(text), 0o644)
	if err := w.LoadFile(script); err != nil {
		t.Fatal(err)
	}
	if !sameFile(filepath.Join(dir, ".", "film.fountain"), script) {
		t.Error("sameFile")
	}
	out := filepath.Join(dir, "copy.fountain")
	if err := ed.exportToFountain(w.textEditor.Text(), out); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(out); string(b) != text {
		t.Errorf("Fountain export %q, want the script %q", b, text)
	}
}

// Cancelling an import (its dialog's Cancel) stops it.
func TestImportPDFCancel(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "s.pdf")
	if err := exportPDF("INT. A - DAY\n\nAction.\n", pdfPath, exportJob{}); err != nil {
		t.Fatal(err)
	}
	if text, _, err := importPDF(context.Background(), pdfPath, nil); err != nil || !strings.Contains(text, "INT. A - DAY") {
		t.Fatalf("import: %q %v", text, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := importPDF(ctx, pdfPath, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled import: %v", err)
	}
}

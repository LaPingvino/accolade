package main

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportFDX(t *testing.T) {
	out := filepath.Join(t.TempDir(), "barn.fdx")
	if err := exportFDX(accoladeScript, out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	var doc struct {
		XMLName      xml.Name `xml:"FinalDraft"`
		DocumentType string   `xml:"DocumentType,attr"`
		Paragraphs   []struct {
			Type string `xml:"Type,attr"`
			Text string `xml:"Text"`
		} `xml:"Content>Paragraph"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("not valid XML: %v", err)
	}
	if doc.DocumentType != "Script" {
		t.Errorf("DocumentType = %q", doc.DocumentType)
	}
	var got []string
	for _, p := range doc.Paragraphs {
		got = append(got, p.Type+":"+p.Text)
	}
	want := "Action:FADE IN:|Scene Heading:INT. BARN - DAY|Action:Rain hammers the roof.|Character:JOHN|Parenthetical:(beat)|Dialogue:It's coming.|Transition:CUT TO:"
	if strings.Join(got, "|") != want {
		t.Errorf("paragraphs:\n%s\nwant:\n%s", strings.Join(got, "|"), want)
	}
}

func TestOpenFDXConvertsToFountain(t *testing.T) {
	dir := t.TempDir()
	fdxPath := filepath.Join(dir, "barn.fdx")
	if err := exportFDX(accoladeScript, fdxPath); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(fdxPath)

	w := newTestWindow(t, "")
	if err := w.LoadFile(fdxPath); err != nil {
		t.Fatal(err)
	}
	text := w.textEditor.Text
	for _, s := range []string{"Title: The Barn", "INT. BARN - DAY", "\nJOHN\n(beat)\nIt's coming.\n", "CUT TO:"} {
		if !strings.Contains(text, s) {
			t.Errorf("imported text is missing %q:\n%s", s, text)
		}
	}
	if w.currentFile != "" || w.suggestedName != "barn.fountain" || !w.hasChanges {
		t.Errorf("imported document: currentFile=%q suggestedName=%q hasChanges=%v", w.currentFile, w.suggestedName, w.hasChanges)
	}
	if title := w.fyneWindow.Title(); !strings.Contains(title, "barn.fountain") {
		t.Errorf("title = %q", title)
	}
	if w.autoSaveTimer != nil {
		t.Error("an imported document must not auto-save over the .fdx")
	}
	if now, _ := os.ReadFile(fdxPath); string(now) != string(original) {
		t.Error("opening the .fdx changed it")
	}
}

func TestOpenBrokenFDX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.fdx")
	os.WriteFile(path, []byte("<FinalDraft><Content>"), 0o644)
	w := newTestWindow(t, "keep me")
	err := w.LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), "broken.fdx") {
		t.Errorf("err = %v", err)
	}
	if w.textEditor.Text != "keep me" {
		t.Errorf("failed import replaced the text: %q", w.textEditor.Text)
	}
}

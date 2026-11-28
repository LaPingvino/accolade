// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/phpdave11/gofpdf"

	"github.com/LaPingvino/accolade/internal/prose"
)

// storyDraft is a story draft of our own, to import and export.
const storyDraft = `# The Lighthouse Keeper

## One

The wind came in from the *west*, and the rain followed it **hard and grey**.[^1]

- a brass compass
- a letter

> The sea gives back what it takes.

[^1]: Not even the fishermen went out.

<!-- Notes
a note of the writer's, not exported
-->
`

// Markdown goes out as Word and comes back as it was: headings,
// emphasis, lists, the quote and the footnote.
func TestMarkdownDOCXRoundTrip(t *testing.T) {
	for _, f := range []string{"docx", "odt"} {
		out := filepath.Join(t.TempDir(), "story."+f)
		if err := exportMarkdown(storyDraft, "The Lighthouse Keeper", out, f); err != nil {
			t.Fatal(err)
		}
		md, _, _, err := toMarkdown(out)
		if err != nil {
			t.Fatal(err)
		}
		want := "# The Lighthouse Keeper\n\n## One\n\nThe wind came in from the *west*, and the rain followed it **hard and grey**.[^1]\n\n" +
			"- a brass compass\n- a letter\n\n> The sea gives back what it takes.\n\n[^1]: Not even the fishermen went out.\n"
		if md != want {
			t.Errorf("%s: came back as\n%s\nwant\n%s", f, md, want)
		}
	}
}

// A document's DOCX is prose: none of a screenplay's styles.
func TestMarkdownDOCXIsProse(t *testing.T) {
	out := filepath.Join(t.TempDir(), "story.docx")
	if err := exportMarkdown(storyDraft, "Story", out, "docx"); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var all strings.Builder
	for _, f := range z.File {
		r, _ := f.Open()
		b := new(bytes.Buffer)
		b.ReadFrom(r)
		r.Close()
		all.Write(b.Bytes())
	}
	for _, not := range []string{"Scene Heading", "Character", "Dialogue", "writer's"} {
		if strings.Contains(all.String(), not) {
			t.Errorf("the DOCX has %q", not)
		}
	}
	if !strings.Contains(all.String(), `w:styleId="Heading1"`) {
		t.Error("no Heading 1 style")
	}
}

func TestMarkdownExportsText(t *testing.T) {
	dir := t.TempDir()
	for f, want := range map[string]string{
		"md":   storyDraft,
		"txt":  "The Lighthouse Keeper\n\nOne\n\nThe wind came in from the west, and the rain followed it hard and grey.[1]\n\n- a brass compass",
		"html": `<h1 id="the-lighthouse-keeper">The Lighthouse Keeper</h1>`, // its anchor for [links](#the-lighthouse-keeper)
	} {
		out := filepath.Join(dir, "story."+f)
		if err := exportMarkdown(storyDraft, "The Lighthouse Keeper", out, f); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(out)
		if !strings.Contains(string(b), want) {
			t.Errorf("%s:\n%s\nlacks %q", f, b, want)
		}
		if f == "html" && (!strings.Contains(string(b), "footnote") || strings.Contains(string(b), "writer's")) {
			t.Errorf("html: footnotes not made, or the notes kept:\n%s", b)
		}
	}
}

// File > Import puts a Word document in a new Markdown document,
// unsaved, named after it; images left out are counted.
func TestImportDOCX(t *testing.T) {
	src := filepath.Join(t.TempDir(), "chapter.docx")
	var b bytes.Buffer
	if err := prose.FromMarkdown("# Chapter\n\nIt *rained*.\n").WriteDOCX(&b); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(src, b.Bytes(), 0o644)
	w, _ := newDialogTestWindow(t)
	w.importDocument(src)
	if !w.markdown || w.textEditor.Text() != "# Chapter\n\nIt *rained*.\n" {
		t.Errorf("markdown %v, text %q", w.markdown, w.textEditor.Text())
	}
	if w.currentFile != "" || w.suggestedName != "chapter.md" || !w.hasChanges {
		t.Errorf("imported: file %q, name %q, changed %v", w.currentFile, w.suggestedName, w.hasChanges)
	}
}

func TestImportHTMLAndText(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "page.html")
	os.WriteFile(page, []byte(`<html><body><h2>Two</h2><p>A <b>bold</b> <img alt="x"> move.</p></body></html>`), 0o644)
	md, dropped, _, err := toMarkdown(page)
	if err != nil || md != "## Two\n\nA **bold** <!-- image left out: x --> move.\n" || dropped != 1 {
		t.Errorf("%q, %d, %v", md, dropped, err)
	}
	if _, _, _, err := toMarkdown(filepath.Join(dir, "a.epub")); err == nil {
		t.Error("an EPUB read without pandoc")
	}
}

// A PDF of prose comes in as paragraphs, not as a screenplay.
func TestImportProsePDF(t *testing.T) {
	f := gofpdf.New("P", "pt", "A4", "")
	f.AddPage()
	f.SetFont("Times", "B", 20)
	f.Cell(0, 30, "The Lighthouse Keeper")
	f.Ln(40)
	f.SetFont("Times", "", 12)
	f.MultiCell(0, 15, "The wind came in from the west and the rain followed it, hard and grey, across the bay where the boats "+
		"lay tied at the harbour wall, and nobody went out that night.", "", "L", false)
	f.Ln(8)
	f.MultiCell(0, 15, "In the morning the sea was calm.", "", "L", false)
	path := filepath.Join(t.TempDir(), "story.pdf")
	if err := f.OutputFileAndClose(path); err != nil {
		t.Fatal(err)
	}
	md, _, err := importProsePDF(context.Background(), path, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "# The Lighthouse Keeper\n\nThe wind came in from the west and the rain followed it, hard and grey, across the bay where the boats " +
		"lay tied at the harbour wall, and nobody went out that night.\n\nIn the morning the sea was calm.\n"
	if md != want {
		t.Errorf("got\n%q\nwant\n%q", md, want)
	}
}

// Without pandoc there is no pandoc: no menu item, no formats.
func TestWithoutPandoc(t *testing.T) {
	old := pandocPath
	pandocPath = func() string { return "" }
	t.Cleanup(func() { pandocPath = old })
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	if slices.Contains(menuLabels(w, "File"), "Import with pandoc…") {
		t.Error("Import with pandoc without pandoc")
	}
	for _, f := range markdownExports() {
		if f.pandoc != "" {
			t.Errorf("%s offered without pandoc", f.label)
		}
	}
	if !slices.Contains(menuLabels(w, "File"), "Import as Markdown…") {
		t.Error("no Import")
	}
}

// With pandoc, it imports and exports what Accolade does not.
func TestWithPandoc(t *testing.T) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		t.Skip("no pandoc")
	}
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	if !slices.Contains(menuLabels(w, "File"), "Import with pandoc…") {
		t.Error("no Import with pandoc")
	}
	dir := t.TempDir()
	epub := filepath.Join(dir, "story.epub")
	if err := exportMarkdown(storyDraft, "The Lighthouse Keeper", epub, "pandoc-epub"); err != nil {
		t.Fatal(err)
	}
	md, err := pandocToMarkdown(epub)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"The Lighthouse Keeper", "*west*", "**hard and grey**", "Not even the fishermen"} {
		if !strings.Contains(md, want) {
			t.Errorf("the EPUB came back without %q:\n%s", want, md)
		}
	}
}

// In Markdown mode Export is the document's dialog, with the document's
// formats; a screenplay keeps its own.
func TestMarkdownExportDialog(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	ed := newMarkdownExportDialog(w)
	if len(ed.format.Options) < len(builtinMarkdownExports) || slices.Contains(ed.format.Options, "Final Draft (FDX)") {
		t.Errorf("formats: %v", ed.format.Options)
	}
	ed.format.SetSelected("Word (DOCX)")
	if !strings.HasSuffix(ed.name.Text, ".docx") {
		t.Errorf("file name %q", ed.name.Text)
	}
	if exportFormatLabel(GetSettings().GetString("export-format")) == "" {
		t.Error("the screenplay's export format is gone")
	}
}

// A PDF of a document Accolade's own PDF cannot print as written (CJK,
// which its fonts have not got), made by Chromium (when there is one; it cannot run
// in some sandboxes, as for TestBrowserPDF).
func TestBrowserPDFMarkdown(t *testing.T) {
	if localBrowser() == "" {
		t.Skip("no Chromium browser")
	}
	text := storyDraft + "\n日本語の文章。\n"
	if markdownPDFMissing(text, nil) == "" {
		t.Fatal("Accolade's PDF prints Japanese prose now: give this test a document it cannot")
	}
	out := filepath.Join(t.TempDir(), "story.pdf")
	if err := exportMarkdown(text, "Story", out, "pdf"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Error("not a PDF")
	}
}

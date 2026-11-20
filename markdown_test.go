package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

const story = `---
title: The Barn
---
# Chapter One

Rain hammered the **roof**. *Anna* waited.

- the dog
    still barking
- the cat

` + "```" + `
# not a heading
` + "```" + `

## The Storm

int. barn - day, she wrote.
`

// menuLabels are the labels in the window's menu called name.
func menuLabels(w *MainWindow, name string) []string {
	var out []string
	for _, m := range w.fyneWindow.MainMenu().Items {
		if m.Label == name {
			for _, it := range m.Items {
				out = append(out, it.Label)
			}
		}
	}
	return out
}

// A .md file opens in Markdown mode, as it is (its indents too), with the
// screenplay's tools put away; a .fountain file brings them back.
func TestMarkdownModeSwitches(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	dir := t.TempDir()
	md := filepath.Join(dir, "story.md")
	fountain := filepath.Join(dir, "script.fountain")
	os.WriteFile(md, []byte(story), 0o644)
	os.WriteFile(fountain, []byte("INT. BARN - DAY\n\nRain.\n"), 0o644)

	if err := w.LoadFile(md); err != nil {
		t.Fatal(err)
	}
	if !w.markdown || !w.textEditor.Markdown {
		t.Fatal("a .md file is not in Markdown mode")
	}
	if w.textEditor.Text() != story {
		t.Errorf("the text changed on opening:\n%q", w.textEditor.Text())
	}
	if w.textEditor.Format != nil || w.textEditor.PageColumns != markdownColumns {
		t.Error("laid out as a screenplay")
	}
	hb := w.headerBar
	if hb.container.Objects[1] != hb.markdownRow || hb.button("file.export").Disabled() || !hb.button("view.cards").Disabled() {
		t.Error("the screenplay's toolbar is still there")
	}
	file, format, view := menuLabels(w, "File"), menuLabels(w, "Format"), menuLabels(w, "View")
	if slices.Contains(file, "Title Page…") || !slices.Contains(file, "Export…") || !slices.Contains(file, "Import as Markdown…") {
		t.Errorf("File: %v", file)
	}
	for _, gone := range []string{"Scene Heading", "Character", "Script Language…", "Underline"} {
		if slices.Contains(format, gone) {
			t.Errorf("Format has %s", gone)
		}
	}
	if !slices.Contains(format, "Bold") || !slices.Contains(view, "Live Preview") || slices.Contains(view, "Cards") {
		t.Errorf("menus: %v %v", format, view)
	}
	if got := w.statsLabel.Text; !strings.Contains(got, "words") || strings.Contains(got, "cene") {
		t.Errorf("status bar: %q", got)
	}

	if err := w.LoadFile(fountain); err != nil {
		t.Fatal(err)
	}
	if w.markdown || w.textEditor.Markdown || w.textEditor.Format == nil {
		t.Error("a .fountain file is not a screenplay again")
	}
	if hb.container.Objects[1] != hb.screenplayRow || hb.button("file.export").Disabled() {
		t.Error("the screenplay's toolbar did not come back")
	}
	if !slices.Contains(menuLabels(w, "Format"), "Scene Heading") {
		t.Error("the elements did not come back")
	}
}

// Enter in Markdown formats no screenplay line, and lists go on.
func TestMarkdownEnter(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.NewMarkdownFile()
	enter := func(text string) string {
		w.textEditor.SetText(text)
		w.textEditor.SetCursorOffset(len([]rune(text)))
		w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		return w.textEditor.Text()
	}
	for _, c := range []struct{ in, out string }{
		{"int. barn - day", "int. barn - day\n"},
		{"\nTHE END", "\nTHE END\n"},
		{"- milk", "- milk\n- "},
		{"  * eggs", "  * eggs\n  * "},
		{"9. flour", "9. flour\n10. "},
		{"> a quote", "> a quote\n> "},
		{"- milk\n- ", "- milk\n"}, // an empty item ends the list
	} {
		if got := enter(c.in); got != c.out {
			t.Errorf("Enter after %q: %q, want %q", c.in, got, c.out)
		}
	}
	// Tab completes no character names
	w.textEditor.SetText("ANNA\nHi.\n\nAN")
	w.textEditor.SetCursorOffset(len("ANNA\nHi.\n\nAN"))
	w.textEditor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if strings.Contains(w.textEditor.Text(), "\nANNA") {
		t.Error("Tab completed a character's name")
	}
}

// The outline is the headings, not what is in code or a #tag.
func TestMarkdownOutline(t *testing.T) {
	items := markdownOutline(story)
	var got []string
	for _, it := range items {
		got = append(got, strings.Repeat("#", it.Level)+" "+it.Text)
	}
	if want := []string{"# Chapter One", "## The Storm"}; !slices.Equal(got, want) {
		t.Errorf("outline %q, want %q", got, want)
	}
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.textEditor.SetText(story)
	w.toggleOutline()
	if len(w.outline.all) != 2 || w.outline.all[1].Line != 15 {
		t.Errorf("outline panel: %+v", w.outline.all)
	}
	w.outline.goTo(w.outline.all[1].Line)
	if line, _ := w.textEditor.Buffer().Position(w.textEditor.CursorOffset()); line != 15 {
		t.Errorf("went to line %d", line)
	}
}

// The preview shows the document as prose: without the front matter and
// the notes, following the text.
func TestMarkdownPreview(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.textEditor.SetText(story + markdownNotes.text("call Bram"))
	w.preview()
	if !w.previewVisible {
		t.Fatal("no preview")
	}
	shown := func() string {
		var b strings.Builder
		for _, s := range w.previewPane.text.Segments {
			if t, ok := s.(interface{ Textual() string }); ok {
				b.WriteString(t.Textual() + "\n")
			}
			if l, ok := s.(*widget.ListSegment); ok {
				for _, it := range l.Items {
					if p, ok := it.(*widget.ParagraphSegment); ok {
						for _, t := range p.Texts {
							b.WriteString(t.Textual() + "\n")
						}
					}
				}
			}
		}
		return b.String()
	}
	got := shown()
	for _, want := range []string{"Chapter One", "roof", "The Storm"} {
		if !strings.Contains(got, want) {
			t.Errorf("preview has no %q:\n%s", want, got)
		}
	}
	for _, not := range []string{"title: The Barn", "call Bram"} {
		if strings.Contains(got, not) {
			t.Errorf("preview shows %q", not)
		}
	}
	w.textEditor.SetText("# Changed\n")
	if !strings.Contains(shown(), "Changed") {
		t.Error("the preview does not follow the text")
	}
	w.setMarkdown(false) // a screenplay's preview is the PDF
	if w.previewVisible {
		t.Error("the preview stayed for a screenplay")
	}
}

// Notes in Markdown are kept in an HTML comment.
func TestMarkdownNotes(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.setMarkdown(true)
	w.textEditor.SetText("# One\n\nText.\n")
	w.toggleNotes()
	test.Type(w.notes.entry, "Call Bram.")
	if got := w.textEditor.Text(); got != "# One\n\nText.\n\n<!-- Notes\nCall Bram.\n-->\n" {
		t.Fatalf("with notes: %q", got)
	}
	if markdownNotes.of(w.textEditor.Text()) != "Call Bram." || fountainNotes.of(w.textEditor.Text()) != "" {
		t.Error("notes not read back")
	}
}

// New Markdown Document makes one, saved as .md; naming it .fountain
// makes it a screenplay.
func TestNewMarkdownDocument(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	if !slices.Contains(menuLabels(w, "File"), "New Markdown Document") {
		t.Fatal("no New Markdown Document")
	}
	w.NewMarkdownFile()
	if !w.markdown || w.untitledName() != "untitled.md" || !strings.HasPrefix(w.textEditor.Text(), "# ") {
		t.Errorf("new Markdown document: %v %q %q", w.markdown, w.untitledName(), w.textEditor.Text())
	}
	if renamedName("story", w.untitledName()) != "story.md" {
		t.Error("a bare name is not .md")
	}
	if err := w.renameTo("story.fountain"); err != nil || w.markdown {
		t.Errorf("named .fountain: still Markdown (%v)", err)
	}
	w.NewFile()
	if w.markdown {
		t.Error("New is a screenplay")
	}
}

// The browser keeps an unsaved Markdown document as Markdown; editing
// together, the kind is shared.
func TestRestoreMarkdownDraft(t *testing.T) {
	w, _ := newDialogTestWindow(t)
	w.restore(kept{Text: "# Draft\n", Markdown: true})
	if !w.markdown || w.textEditor.Text() != "# Draft\n" {
		t.Error("the draft is not Markdown")
	} // a peer's document is the kind the peer says
	w.applyShared("markdown", "false")
	if w.markdown {
		t.Error("a shared screenplay stayed Markdown")
	}
}

func TestMarkdownStatsAndPrint(t *testing.T) {
	if got := markdownStats(story); got != "21 words · ~1 page" {
		t.Errorf("stats %q", got)
	}
	page, err := markdownHTML(story, "The Barn")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1>Chapter One</h1>", "<strong>roof</strong>", "<li>the dog"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("no %s in\n%s", want, page)
		}
	}
	if strings.Contains(string(page), "title:") {
		t.Error("front matter printed")
	}
}

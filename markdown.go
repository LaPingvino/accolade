package main

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/prose"
)

// Markdown mode (docs/ADAPTATION.md, step 1): a .md file opens with
// Markdown's colours, an outline of its headings and a preview of it as
// prose. The screenplay's own tools (elements, completion, the title page,
// the export formats, cards and colours) are left out; search, vim, notes,
// autosave and editing together are the same.

// markdownColumns is how wide the editor's column is for prose.
const markdownColumns = 72

// kindOfName tells from a file's name whether it is Markdown; known is
// false when its name does not say (no extension, an unknown one).
func kindOfName(name string) (markdown, known bool) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown":
		return true, true
	case ".fountain", ".spmd", ".txt", ".fdx", ".pdf", ".highland", ".accolade":
		return false, true
	}
	return false, false
}

// openExtensions are the files Open offers (the browser's file picker
// too). A .textbundle is a folder: Open goes into it and opens its text
// (bundleOf), so it is not among them.
var openExtensions = []string{".fountain", ".spmd", ".txt", ".md", ".markdown", ".highland", ".accolade", ".textpack", ".fdx", ".pdf"}

func isMarkdownFile(name string) bool {
	md, _ := kindOfName(name)
	return md
}

// modeFromName makes the window's mode the one a file's name says, if it
// says one.
func (w *MainWindow) modeFromName(name string) {
	if md, ok := kindOfName(name); ok {
		w.setMarkdown(md)
	}
}

// untitledName is what a document without a name is saved as.
func (w *MainWindow) untitledName() string {
	if w != nil && w.markdown {
		return "untitled.md"
	}
	return "untitled.fountain"
}

// setMarkdown switches the window between a screenplay and a Markdown
// document: the editor's colours and layout, the toolbar, the menu, the
// outline and the status bar.
func (w *MainWindow) setMarkdown(on bool) {
	if w.markdown == on {
		return
	}
	w.markdown = on
	w.textEditor.Markdown = on
	w.applyScriptFormat() // the screenplay's columns, or none
	if w.headerBar != nil {
		w.headerBar.setMarkdown(on)
	}
	if on {
		w.cardsVisible = false // cards are a screenplay's scenes
	} else {
		w.previewVisible = false // a screenplay's preview is a PDF
	}
	w.refreshMenu()
	text := w.textEditor.Text()
	if w.outlineVisible && w.outline != nil {
		w.outline.update(text)
	}
	if w.notesVisible && w.notes != nil {
		w.notes.update(text)
	}
	w.updateCurrentElement()
	w.updateStatusBar()
	w.relayout()
	w.shareMeta("markdown", strconv.FormatBool(on))
}

// NewMarkdownFile starts a Markdown document, asking first if the one
// open has unsaved changes.
func (w *MainWindow) NewMarkdownFile() {
	create := func() {
		w.cancelAutoSave()
		w.forgetUnsavedSource()
		w.setMarkdown(true)
		w.format = fileFormat{}
		w.textEditor.SetText("# Untitled\n\n")
		w.textEditor.SetCursorOffset(len([]rune(w.textEditor.Text())))
		w.currentFile, w.suggestedName = "", ""
		w.plainBundle = ""
		w.setBundle(nil)
		w.hasChanges = false
		w.updateTitle()
		w.updateStatusBar()
	}
	if !w.hasChanges {
		create()
		return
	}
	dialog.ShowConfirm("Unsaved Changes", "You have unsaved changes. Create a new file anyway?",
		func(ok bool) {
			if ok {
				create()
			}
		}, w.fyneWindow)
}

// outlineItems are the outline of the window's document: a screenplay's
// sections and scenes, or a Markdown document's headings.
func (w *MainWindow) outlineItems(text string) []outlineItem {
	if w.markdown {
		return markdownOutline(text)
	}
	return outlineOf(text, w.sceneStarts())
}

// markdownOutline is a Markdown text's headings, as sections.
func markdownOutline(text string) []outlineItem {
	var items []outlineItem
	for _, h := range syntax.MarkdownHeadings(strings.Split(text, "\n")) {
		items = append(items, outlineItem{Line: h.Line, Text: h.Text, Level: h.Level})
	}
	return items
}

// markdownStats is the status bar's text for a Markdown document: its
// words (the front matter, code and comments left out) and its pages, at
// a manuscript's 250 words a page.
func markdownStats(text string) string {
	lines := strings.Split(text, "\n")
	words := 0
	for i, k := range syntax.ClassifyMarkdown(lines) {
		switch k {
		case syntax.Action, syntax.MdHeading, syntax.MdQuote, syntax.MdListItem:
			words += countWordsIn(lines[i])
		}
	}
	s := plural(words, "word")
	if words > 0 {
		s += " · ~" + plural((words+249)/250, "page")
	}
	return s
}

// markdownNames are Markdown's kinds of line in the status bar.
var markdownNames = map[syntax.Kind]string{
	syntax.Action: "Text", syntax.Empty: "Empty", syntax.MdHeading: "Heading",
	syntax.MdQuote: "Quote", syntax.MdListItem: "List", syntax.MdCode: "Code",
	syntax.MdCodeFence: "Code", syntax.MdRule: "Break", syntax.MdFrontMatter: "Front Matter",
	syntax.MdComment: "Comment",
}

// listMarker is a list item's or quote's mark, with the spaces around it:
// "- ", "  2. ", "> ".
var listMarker = regexp.MustCompile(`^(\s*)(?:([-+*]|\d{1,9}[.)])|>)(\s+|$)`)

// continueList handles Enter at the end of a list item or quote: the next
// line starts with its mark (a number one up); on an item with nothing in
// it, the mark goes and the list ends. It reports false for other lines.
func continueList(e *editor.ScriptEditor) bool {
	if e.SelectedText() != "" {
		return false
	}
	text := []rune(e.Text())
	cursor := e.CursorOffset()
	start, end := lineBounds(text, cursor)
	line := string(text[start:end])
	m := listMarker.FindStringSubmatch(line)
	if cursor != end || m == nil {
		return false
	}
	if strings.TrimSpace(line[len(m[0]):]) == "" { // an empty item: the list ends
		e.Edit(func(b *buffer.Buffer) { b.Replace(start, end, "") })
		return true
	}
	next := m[0]
	if n, err := strconv.Atoi(strings.TrimRight(m[2], ".)")); err == nil && m[2] != "" {
		next = m[1] + strconv.Itoa(n+1) + m[2][len(m[2])-1:] + m[3]
	}
	e.Edit(func(b *buffer.Buffer) { b.Insert("\n" + next) })
	return true
}

// markdownBody is the part of a Markdown document that is shown and
// printed: without the front matter and the notes block.
func markdownBody(text string) string {
	text = string([]rune(text)[:markdownNotes.start(text)])
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	i := 0
	for i < len(kinds) && kinds[i] == syntax.MdFrontMatter {
		i++
	}
	return strings.Join(lines[i:], "\n")
}

// previewPanel shows a Markdown document as formatted prose beside the
// editor (View > Preview), following the text as it is written.
type previewPanel struct {
	text *widget.RichText
	box  fyne.CanvasObject
}

func newPreviewPanel() *previewPanel {
	p := &previewPanel{text: widget.NewRichText()}
	p.text.Wrapping = fyne.TextWrapWord
	fyne.SetAccessibleLabel(p.text, "Preview")
	width := canvas.NewRectangle(color.Transparent) // a nil colour crashes the painter
	width.SetMinSize(fyne.NewSize(320, 0))
	heading := widget.NewLabelWithStyle("Preview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.box = container.NewStack(width, container.NewBorder(heading, nil, nil, nil, container.NewVScroll(p.text)))
	return p
}

// update shows text; a table as lines, not in a scroller of its own that
// would catch a drag meant for the preview's (source_rows.go).
func (p *previewPanel) update(text string) {
	p.text.Segments, _ = markdownSegments(markdownBody(text), -1)
	p.text.Refresh()
}

// preview shows the document as it will look: a screenplay as a PDF, a
// Markdown document in the preview beside the editor (or hides it).
func (w *MainWindow) preview() {
	if !w.markdown {
		w.previewPDF()
		return
	}
	if w.previewPane == nil {
		w.previewPane = newPreviewPanel()
	}
	w.previewVisible = !w.previewVisible
	if w.previewVisible {
		w.previewPane.update(w.textEditor.Text())
	}
	w.relayout()
}

// markdownHTML is a Markdown document as a web page, for printing.
func markdownHTML(text, title string) ([]byte, error) {
	var body bytes.Buffer
	if err := prose.Markdown.Convert([]byte(markdownBody(text)), &body); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, `<!doctype html><html><head><meta charset="utf-8"><title>%s</title>`+
		`<style>body{font:12pt/1.5 Georgia,serif;max-width:40em;margin:2em auto;padding:0 1em}`+
		`pre,code{font-family:monospace}blockquote{margin-left:1.5em;font-style:italic}</style></head><body>`,
		strings.NewReplacer("&", "&amp;", "<", "&lt;").Replace(title))
	out.Write(body.Bytes())
	out.WriteString("</body></html>\n")
	return out.Bytes(), nil
}

// printMarkdown prints a Markdown document as a web page, through the
// browser, as printScript does a screenplay.
func (w *MainWindow) printMarkdown() {
	dir, err := os.MkdirTemp("", "accolade-print-")
	if err == nil {
		var page []byte
		name := strings.TrimSuffix(w.scriptName(), filepath.Ext(w.scriptName()))
		if page, err = markdownHTML(w.textEditor.Text(), name); err == nil {
			out := filepath.Join(dir, "document.html")
			if err = os.WriteFile(out, page, 0o644); err == nil {
				w.setStatus("Print: choose a printer, or Save as PDF")
				printFile(out)
				return
			}
		}
	}
	dialog.ShowError(fmt.Errorf("could not prepare the document to print: %v", err), w.fyneWindow)
}

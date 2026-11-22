package main

import (
	"html"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/prose"
)

// A Markdown document's page setup (pagesetup.go has a screenplay's): the
// paper, and a title page, as its Export, Print and Live Preview use them;
// where the PDF's notes print and how they are numbered.
// Its own keys: a screenplay's paper "" is its script format's, which a
// document has none of.

const (
	mdPaperPref     = "markdown-page-size"  // "" as before: A4 in Word, ODT and PDF, the printer's for a web page
	mdTitlePagePref = "markdown-title-page" // a page with the title (and the author) before the text
	// where the PDF's notes print: "" at the page's foot, "chapter" after
	// each chapter, "end" after the text; numbered again in each chapter
	mdNotesPref           = "markdown-notes"
	mdNotesPerChapterPref = "markdown-notes-per-chapter"
)

// mdPageSetup is a Markdown document's page setup.
type mdPageSetup struct {
	paper           string // "", letter, a4, a5
	titlePage       bool
	notesAt         prose.NotesPlace
	notesPerChapter bool
}

// mdNotesPlaces are the notes' places as kept, in prose.NotesPlace's
// order, and as Page Setup says them.
var (
	mdNotesPlaces = []string{"", "chapter", "end"}
	mdNotesLabels = []string{"At the foot of the page", "After each chapter", "After the text"}
	mdNumberings  = []string{"Through the document", "Again in each chapter"}
)

func (w *MainWindow) markdownPageSetup() mdPageSetup {
	if w.settings == nil {
		return mdPageSetup{}
	}
	p := mdPageSetup{paper: w.settings.GetString(mdPaperPref), titlePage: w.settings.GetBoolean(mdTitlePagePref),
		notesPerChapter: w.settings.GetBoolean(mdNotesPerChapterPref)}
	if i := slices.Index(mdNotesPlaces, w.settings.GetString(mdNotesPref)); i > 0 {
		p.notesAt = prose.NotesPlace(i)
	}
	return p
}

// mdPapers are the paper choices, the first the default.
var mdPapers = []string{"Default (A4 in Word, ODT and PDF; the printer's when printed)", "Letter", "A4", "A5"}

// markdownPageSummary is the line saying the document's page setup.
func (w *MainWindow) markdownPageSummary() string {
	p := w.markdownPageSetup()
	paper := "Default paper"
	if p.paper != "" {
		paper = paperName(p.paper)
	}
	s := paper + " · no title page"
	if p.titlePage {
		s = paper + " · title page"
	}
	switch p.notesAt { // the notes only when not at the page's foot, numbered through
	case prose.NotesAtChapterEnd:
		s += " · notes after each chapter"
	case prose.NotesAtEnd:
		s += " · notes after the text"
	}
	if p.notesPerChapter {
		s += " · notes numbered by chapter"
	}
	return s + "."
}

// markdownTitlePage is the title page's lines for text (named name): its
// title — the front matter's title:, else its first heading, else the
// name — and the front matter's author:, if it has one.
func markdownTitlePage(text, name string) []string {
	lines := []string{markdownTitle(text, name)}
	if t := frontMatterValue(text, "title"); t != "" {
		lines[0] = t
	}
	if a := frontMatterValue(text, "author"); a != "" {
		lines = append(lines, a)
	}
	return lines
}

// frontMatterValue is the front matter's value for key ("" without one).
func frontMatterValue(text, key string) string {
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	for i := 0; i < len(kinds) && kinds[i] == syntax.MdFrontMatter; i++ {
		if k, v, ok := strings.Cut(lines[i], ":"); ok && strings.EqualFold(strings.TrimSpace(k), key) {
			if v = strings.Trim(strings.TrimSpace(v), `"'`); v != "" {
				return v
			}
		}
	}
	return ""
}

// titlePageOf is the title page's lines with the page setup p (nil: none).
func (p mdPageSetup) titlePageOf(text, name string) []string {
	if !p.titlePage {
		return nil
	}
	return markdownTitlePage(text, name)
}

// pageCSS is the web page's paper and title page style: nothing for the
// defaults, so the page is as it was.
func (p mdPageSetup) pageCSS() string {
	var css string
	if p.paper != "" {
		css += "@page{size:" + map[string]string{"letter": "letter", "a4": "A4", "a5": "A5"}[strings.ToLower(p.paper)] + "}"
	}
	if p.titlePage {
		css += ".title-page{text-align:center;padding-top:30vh;break-after:page;page-break-after:always}" +
			".title-page .title{font-size:2em;margin-bottom:1em}"
	}
	return css
}

// titlePageHTML is the title page's lines as the web page's.
func titlePageHTML(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="title-page">`)
	for i, l := range lines {
		class := "by"
		if i == 0 {
			class = "title"
		}
		b.WriteString(`<p class="` + class + `">` + html.EscapeString(l) + `</p>`)
	}
	b.WriteString("</section>")
	return b.String()
}

// mdPageSetupDlg is Page Setup for a Markdown document.
type mdPageSetupDlg struct {
	w         *MainWindow
	d         *dialog.CustomDialog
	paper     *widget.Select
	titlePage *widget.Check
	notes     *widget.Select
	numbering *widget.Select
	ok        *widget.Button
}

// showMarkdownPageSetup opens the document's Page Setup; done runs after OK.
func (w *MainWindow) showMarkdownPageSetup(done func()) *mdPageSetupDlg {
	p := w.newMarkdownPageSetup(done)
	p.d.Show()
	return p
}

func (w *MainWindow) newMarkdownPageSetup(done func()) *mdPageSetupDlg {
	p := &mdPageSetupDlg{w: w}
	p.paper = widget.NewSelect(mdPapers, nil)
	p.titlePage = widget.NewCheck("Title page", nil)
	s := w.markdownPageSetup()
	p.paper.SetSelected(mdPapers[0])
	for _, l := range mdPapers[1:] {
		if strings.EqualFold(l, s.paper) {
			p.paper.SetSelected(l)
		}
	}
	p.titlePage.SetChecked(s.titlePage)
	p.notes = widget.NewSelect(mdNotesLabels, nil)
	p.notes.SetSelected(mdNotesLabels[s.notesAt])
	p.numbering = widget.NewSelect(mdNumberings, nil)
	p.numbering.SetSelected(mdNumberings[0])
	if s.notesPerChapter {
		p.numbering.SetSelected(mdNumberings[1])
	}
	intro := widget.NewLabel("How the document prints: Export (web page, Word, ODT, PDF), Print and Live Preview use these.")
	intro.Wrapping = fyne.TextWrapWord
	about := hint("The title page has the title — the front matter's title:, else the first heading — and its author: from the front matter, centred, on a page of its own.")
	notesAbout := hint("Notes: where the PDF and the pages print the footnotes, and how they are numbered (Word and ODT keep them at the page's foot, numbered through). A chapter starts at a top-level heading; under a single title heading, at the next level's.")
	cancel := widget.NewButton("Cancel", func() { p.d.Hide() })
	p.ok = widget.NewButton("OK", func() {
		chosen := mdPageSetup{paper: chosenPaper(p.paper.Selected), titlePage: p.titlePage.Checked,
			notesAt: prose.NotesPlace(max(slices.Index(mdNotesLabels, p.notes.Selected), 0)), notesPerChapter: p.numbering.Selected == mdNumberings[1]}
		if p.paper.Selected == mdPapers[0] {
			chosen.paper = ""
		}
		w.changeSetting(pageSetupPref, func(s *Settings) {
			s.SetString(mdPaperPref, chosen.paper)
			s.SetBoolean(mdTitlePagePref, chosen.titlePage)
			s.SetString(mdNotesPref, mdNotesPlaces[chosen.notesAt])
			s.SetBoolean(mdNotesPerChapterPref, chosen.notesPerChapter)
		})
		p.d.Hide()
		if w.previewVisible && w.previewPane != nil { // the preview shows it at once
			w.previewPane.update(w.writeEditor().Text())
		}
		if done != nil {
			done()
		}
	})
	p.ok.Importance = widget.HighImportance
	body := container.NewVScroll(container.NewVBox(intro, labelled("Paper:", p.paper), p.titlePage, about,
		labelled("Notes:", p.notes), labelled("Numbering:", p.numbering), notesAbout))
	p.d = dialog.NewCustomWithoutButtons("Page Setup", container.NewBorder(nil, w.dialogButtons(cancel, p.ok), nil, nil, body), w.fyneWindow)
	p.d.Resize(fitDialog(w.fyneWindow, 520, 470))
	p.d.SetOnEscape(p.d.Hide)
	return p
}

// withPageSetup puts the page setup's line at the top of the preview, its
// title page in it.
func (p *previewPanel) withPageSetup(w *MainWindow) {
	p.w = w
	p.line = w.newPageSetupLine(func() string { return "" }, nil)
	p.line.markdown = w.editorMarkdown // a source in the middle is a document too
	p.line.refresh()
	p.box = p.layout()
}

// titleSegments is the title page in the preview: the lines centred, a
// rule under them.
func titleSegments(lines []string) []widget.RichTextSegment {
	var segs []widget.RichTextSegment
	for i, l := range lines {
		style := widget.RichTextStyle{Alignment: fyne.TextAlignCenter, ColorName: theme.ColorNameForeground}
		if i == 0 {
			style = widget.RichTextStyleHeading
			style.Alignment = fyne.TextAlignCenter
			style.Inline = false // a line of its own
		}
		segs = append(segs, &widget.TextSegment{Text: l, Style: style})
	}
	if len(segs) > 0 {
		segs = append(segs, &widget.SeparatorSegment{})
	}
	return segs
}

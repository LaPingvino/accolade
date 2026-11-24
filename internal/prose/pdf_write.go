package prose

import (
	"bytes"
	"cmp"
	_ "embed"
	"fmt"
	"io"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	lexfont "github.com/LaPingvino/lexington/font"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/pdf"

	"github.com/LaPingvino/accolade/internal/images"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/unicode/bidi"
)

// The PDF of a document, written by Accolade itself (no browser): its
// pages drawn through Lexington's drawing interface (pdf.Drawer), so
// that a Recorder keeps them as drawn and the page view shows them. The
// layout is the web page's (markdownHTML): a title page if asked,
// headings, paragraphs with their emphasis, links, code and line breaks,
// quotes in italic and indented, lists with their marks, code blocks on
// grey, a break as a short rule, tables with their header; the
// footnotes at the foot of the page that refers to them, under a short
// rule (a long one goes on at the next page's foot), or after each
// chapter, or the text (Doc.NotesAt); page numbers at the foot. Links
// work: an address's, a heading's anchor (#…), a note's number to the
// note and back; the headings are the PDF's outline.
//
// Pictures (Doc.Images): one alone in its paragraph on a line of its own,
// as wide as the text at most (never larger than it is, at 96 dpi as a
// browser shows it), kept on one page (made smaller to fit one), its
// title under it as a caption; one in a line of text as high as the
// line. One that cannot be read (missing, on the web — never fetched —,
// an SVG) is a quiet placeholder: its alt text and where it is.
//
// Devanagari is shaped as Lexington's screenplay PDF shapes it
// (pdf.Shaper: conjuncts, the i-matra, reph, marks on their letters; the
// text read back as written), a run in one face, a long word broken
// between clusters. It prints in Noto Sans Devanagari, proportional as
// the text is, when the text's face has no Devanagari (scriptFaces).
//
// Text read right to left (Hebrew, Arabic, Persian) is shaped and
// ordered as Lexington's screenplay PDF does it (pdf.Shaper): Arabic
// joined and pointed Hebrew composed before the text is wrapped; a
// paragraph's direction from its first letter, its characters' levels
// over the whole paragraph (UAX #9), each line wrapped in logical order
// and then shown in visual order, its right-to-left runs turned around
// with their marks placed by the font's GPOS as the text as written
// shapes (pdf.Shaper.ComposeMarks: the fatha of لَا on the lam's half of
// the ligature, cut with the text). A paragraph read right to
// left is set from the right: its lines right-aligned, its indents, list
// marks and note numbers on the right. Hebrew is in Noto Sans Hebrew,
// Arabic in Courier Badi, unless the text's face has them (scriptFaces).
//
// Not yet: scripts no font here has (CJK…). PDFMissing says what a
// document has of those; the caller prints such a document through a
// browser instead.

// PDFFonts are the proportional face's files for the text: regular, bold,
// italic, bold italic (TrueType). nil: the Go fonts (BSD licence). Code
// is in Courier Badi (Lexington's, OFL), which is also the face for a
// character the text's face has not got.
type PDFFonts struct {
	Regular, Bold, Italic, BoldItalic []byte
}

func (f PDFFonts) orGo() PDFFonts {
	if f.Regular == nil {
		return PDFFonts{goregular.TTF, gobold.TTF, goitalic.TTF, gobolditalic.TTF}
	}
	if f.Bold == nil {
		f.Bold = f.Regular
	}
	if f.Italic == nil {
		f.Italic = f.Regular
	}
	if f.BoldItalic == nil {
		f.BoldItalic = f.Bold
	}
	return f
}

// The faces in the PDF.
const (
	faceText = "text"
	faceMono = "mono"
)

// Script faces: Noto Sans Devanagari and Hebrew (SIL Open Font License
// 1.1, fonts/OFL-*.txt), the faces for those scripts in proportional
// text when the text's face has not got them (else they would be in
// Courier Badi, monospaced, stretched beside the text). Each is subset to
// its script (its blocks and presentation forms, the joiners, the dotted
// circle), every layout feature kept, without hinting (pyftsubset
// --layout-features='*' --no-hinting), to keep the browser build small.
// They have no italic.
//
// Arabic stays in Courier Badi: its shaping prints each letter's form as
// its presentation form (pdf.Shaper.Compose), which Courier Badi maps;
// Noto Sans Arabic builds the forms of beh, noon and the like from a
// dotless letter and its dots, glyphs no character maps to, so they
// would print unjoined.
var (
	//go:embed fonts/NotoSansDevanagari-Regular.ttf
	devaRegular []byte
	//go:embed fonts/NotoSansDevanagari-Bold.ttf
	devaBold []byte
	//go:embed fonts/NotoSansHebrew-Regular.ttf
	hebrewRegular []byte
	//go:embed fonts/NotoSansHebrew-Bold.ttf
	hebrewBold []byte
)

// scriptFace is a face for one script: its name in the PDF, its files,
// the characters it is for.
type scriptFace struct {
	name          string
	regular, bold []byte
	covers        func(rune) bool
}

var scriptFaces = []scriptFace{
	{"deva", devaRegular, devaBold, func(r rune) bool {
		return r >= 0x0900 && r <= 0x097F || r >= 0x1CD0 && r <= 0x1CFF || r >= 0xA8E0 && r <= 0xA8FF
	}},
	{"hebrew", hebrewRegular, hebrewBold, func(r rune) bool { return r >= 0x0590 && r <= 0x05FF || r >= 0xFB1D && r <= 0xFB4F }},
}

// font is the face's file in a style.
func (f scriptFace) font(style string) []byte {
	if strings.Contains(style, "B") {
		return f.bold
	}
	return f.regular
}

// pdfPapers are gofpdf's names of the papers.
var pdfPapers = map[string]string{"letter": "Letter", "a4": "A4", "a5": "A5"}

// Sizes, in points.
const (
	textSize  = 11.0
	leading   = 1.45 // a line's height, in sizes
	codeSize  = 9.0
	noteSize  = 9.0
	indentStp = 20.0
)

var headingSizes = [7]float64{0, 20, 16, 13.5, 12, 11, 11}

// glyphs tells which characters a font has.
func glyphs(data []byte) func(rune) bool {
	f, err := sfnt.Parse(data)
	if err != nil {
		return func(rune) bool { return false }
	}
	var b sfnt.Buffer
	return func(r rune) bool {
		i, err := f.GlyphIndex(&b, r)
		return err == nil && i != 0
	}
}

// PDFMissing says what of the document its PDF cannot print as written:
// characters no font of the PDF has ("" if it prints well).
func (d *Doc) PDFMissing(f PDFFonts) string {
	f = f.orGo()
	has := []func(rune) bool{glyphs(f.Regular), glyphs(lexfont.GetFont(lexfont.CourierBadiName, ""))}
	for _, sf := range scriptFaces {
		has = append(has, glyphs(sf.regular))
	}
	var missing []rune
	seen := map[rune]bool{}
	for _, r := range d.Plain() + strings.Join(d.TitlePage, "\n") {
		if unicode.IsSpace(r) || unicode.IsControl(r) || seen[r] || slices.ContainsFunc(has, func(h func(rune) bool) bool { return h(r) }) {
			continue
		}
		seen[r] = true
		missing = append(missing, r)
	}
	if len(missing) == 0 {
		return ""
	}
	s := string(missing)
	if len(missing) > 12 {
		s = string(missing[:12]) + "…"
	}
	return "characters its fonts have not got (" + s + ")"
}

// WritePDF writes the document as a PDF in fonts f, and gives its pages
// as drawn.
func (d *Doc) WritePDF(w io.Writer, f PDFFonts) (pdf.Drawing, error) {
	f = f.orGo()
	size, ok := pdfPapers[strings.ToLower(d.Paper)]
	if !ok {
		size = "A4"
	}
	rec := pdf.NewRecorder(pdf.NewFpdf("P", "pt", size))
	p := &pdfWriter{d: rec, f: f, doc: d}
	p.setup()
	p.prepare()
	if d.Title != "" {
		rec.SetTitle(d.Title, true)
	}
	if len(d.TitlePage) > 0 {
		p.titlePage(d.TitlePage)
	}
	p.newPage()
	for i, b := range d.Blocks {
		if c := p.chapterAt[b]; c > 0 && d.NotesAt == NotesAtChapterEnd {
			p.endnotes(c) // the chapters' before
		}
		p.block(b, 0, false, i == 0)
	}
	p.finished = true
	if d.NotesAt == NotesAtPageFoot {
		for len(p.carry) > 0 { // a long note's rest, on pages of their own
			p.newPage()
		}
	} else {
		p.endnotes(len(p.chapterTitles))
	}
	p.finishPage()
	p.pageNumbers(len(d.TitlePage) > 0)
	var b bytes.Buffer
	if err := rec.Output(&b); err != nil {
		return pdf.Drawing{}, err
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		return pdf.Drawing{}, err
	}
	return rec.Drawing(), nil
}

// pdfWriter lays a document out on pages.
type pdfWriter struct {
	d                        pdf.Drawer
	f                        PDFFonts
	doc                      *Doc
	has                      map[string]func(rune) bool
	pageW, pageH             float64
	left, right, top, bottom float64
	y                        float64
	atTop                    bool // nothing printed on the page yet

	// the notes, numbered in the order they are referred to (a note
	// referred to again keeps its number), by their first block's place
	notes  []*note
	noteOf map[**Block]*note
	// chapterAt is the chapter a top-level heading starts (from 1);
	// chapterTitles the chapters' headings' text (the first, "": before
	// any)
	chapterAt     map[*Block]int
	chapterTitles []string
	// anchors are the headings' internal links, by their anchor (ID), and
	// by the heading; minLevel is the top-level headings' highest level,
	// outline the last level put in the PDF's outline (-1: none yet)
	anchors   map[string]int
	anchorOf  map[*Block]int
	minLevel  int
	outline   int
	foot      []footLine // the notes at this page's foot so far
	carry     []footLine // the rest of a note too long for its page's foot
	collect   *[]footLine
	collected *[]*note               // the notes a note being laid out refers to
	gapNext   float64                // the gap before the next line laid out for a note
	decor     []func(float64, bool)  // drawn with the next line (a list item's mark), told its direction
	lineRTL   bool                   // the lines being emitted read right to left
	finished  bool                   // the text is all laid out: only notes are left
	pics      map[string]*pic        // the pictures read, by their Image
	shapers   map[string]*pdf.Shaper // the faces' shapers, by face and style (made when first needed)
}

// pic is a picture in the PDF: its name there and its size in pixels, or
// why it is not shown.
type pic struct {
	name string
	w, h int
	err  error
}

// pic is img's picture, read into the PDF once.
func (p *pdfWriter) pic(img *Image) *pic {
	key := img.Src
	if img.Data != nil {
		key = fmt.Sprintf("%p", img)
	}
	if pc, ok := p.pics[key]; ok {
		return pc
	}
	if p.pics == nil {
		p.pics = map[string]*pic{}
	}
	pc := &pic{name: "pic" + strconv.Itoa(len(p.pics)+1)}
	p.pics[key] = pc
	got := p.doc.picture(img)
	if pc.err = got.Err; pc.err == nil {
		pc.w, pc.h, pc.err = pdf.AddImage(p.d, pc.name, got.Data)
	}
	return pc
}

// note is a footnote: its number and text, the chapter of its first
// reference; its place's link and its first reference's.
type note struct {
	num     int
	blocks  []*Block
	chapter int
	dest    int // the link to it
	back    int // the link to its (first) reference
	backSet bool
	placed  bool       // put at a page's foot (or carried to the next)
	printed bool       // printed among the notes after a chapter or the text
	lines   []footLine // laid out, once
	laid    bool
	nested  []*note // the notes its text refers to
}

// footLine is a line of a note: the gap before it (none at the top of a
// page's notes), its height, and how it is drawn with its top at y.
type footLine struct {
	gap, h float64
	draw   func(y float64)
}

func (p *pdfWriter) setup() {
	d := p.d
	for _, s := range []struct {
		style string
		data  []byte
	}{{"", p.f.Regular}, {"B", p.f.Bold}, {"I", p.f.Italic}, {"BI", p.f.BoldItalic}} {
		// copies (pdf.AddFont): gofpdf writes into the bytes it gets, and
		// these are Fyne's own interface font and the embedded Courier Badi
		pdf.AddFont(d, faceText, s.style, s.data)
		pdf.AddFont(d, faceMono, s.style, lexfont.GetFont(lexfont.CourierBadiName, s.style))
		for _, sf := range scriptFaces {
			pdf.AddFont(d, sf.name, s.style, sf.font(s.style))
		}
	}
	p.has = map[string]func(rune) bool{faceText: glyphs(p.f.Regular), faceMono: glyphs(lexfont.GetFont(lexfont.CourierBadiName, ""))}
	for _, sf := range scriptFaces {
		p.has[sf.name] = glyphs(sf.regular)
	}
	d.SetAutoPageBreak(false, 0)
	d.SetCellMargin(0)
	p.pageW, p.pageH = d.GetPageSize()
	margin := 72.0
	if p.pageW < 450 { // A5
		margin = 54
	}
	p.left, p.right, p.top, p.bottom = margin, p.pageW-margin, margin, p.pageH-margin
	d.SetMargins(p.left, p.top, p.pageW-p.right)
	d.SetFont(faceText, "", textSize)
}

// prepare numbers the notes, finds the chapters and the headings'
// anchors, before anything is laid out.
func (p *pdfWriter) prepare() {
	p.noteOf, p.chapterAt, p.anchors, p.anchorOf = map[**Block]*note{}, map[*Block]int{}, map[string]int{}, map[*Block]int{}
	p.outline = -1
	// a chapter starts at a top-level heading of the highest level; one
	// such heading over headings of lower levels is the title, and the
	// chapters are the next level's
	count := map[int]int{}
	for _, b := range p.doc.Blocks {
		if b.Kind == Heading {
			count[b.Level]++
		}
	}
	levels := []int{}
	for lv := 1; lv <= 6; lv++ {
		if count[lv] > 0 {
			levels = append(levels, lv)
		}
	}
	chapter := 0
	if len(levels) > 0 {
		p.minLevel = levels[0]
		if count[levels[0]] == 1 && len(levels) > 1 {
			chapter = levels[1]
		} else {
			chapter = levels[0]
		}
	}
	p.chapterTitles = []string{""}
	numbered := 0
	var walk func(bs []*Block, top bool)
	var spans func(ss []Span)
	spans = func(ss []Span) {
		for i := range ss {
			if ss[i].Note == nil || len(ss[i].Note) == 0 || p.noteOf[&ss[i].Note[0]] != nil {
				continue
			}
			numbered++
			n := &note{num: numbered, blocks: ss[i].Note, chapter: len(p.chapterTitles) - 1, dest: p.d.AddLink(), back: p.d.AddLink()}
			p.noteOf[&ss[i].Note[0]] = n
			p.notes = append(p.notes, n)
			walk(n.blocks, false) // the notes it refers to, after it
		}
	}
	walk = func(bs []*Block, top bool) {
		for _, b := range bs {
			if b.Kind == Heading {
				if top && b.Level == chapter {
					p.chapterTitles = append(p.chapterTitles, spansText(b.Spans))
					p.chapterAt[b] = len(p.chapterTitles) - 1
					if p.doc.NotesPerChapter {
						numbered = 0
					}
				}
				if b.ID != "" {
					if _, ok := p.anchors[b.ID]; !ok {
						p.anchors[b.ID] = p.d.AddLink()
						p.anchorOf[b] = p.anchors[b.ID]
					}
				}
			}
			spans(b.Spans)
			for _, r := range b.Rows {
				for _, c := range r {
					spans(c)
				}
			}
			walk(b.Children, false)
		}
	}
	walk(p.doc.Blocks, true)
}

// noteAt is the note a span refers to (nil: none).
func (p *pdfWriter) noteAt(s *Span) *note {
	if len(s.Note) == 0 {
		return nil
	}
	return p.noteOf[&s.Note[0]]
}

func (p *pdfWriter) newPage() {
	p.finishPage()
	p.d.AddPage()
	p.y, p.atTop = p.top, true
	if len(p.carry) > 0 { // a note's rest first, as much as the page's share for notes
		n := 0
		for n < len(p.carry) && (n == 0 || p.footHeight(append(p.foot, p.carry[:n+1]...)) <= p.footShare()) {
			n++
		}
		p.foot = append(p.foot, p.carry[:n]...)
		p.carry = p.carry[n:]
	}
}

// footHeight is the height the lines take at a page's foot, the space
// over them with the rule included.
func (p *pdfWriter) footHeight(lines []footLine) float64 {
	if len(lines) == 0 {
		return 0
	}
	h := noteSep
	for i, l := range lines {
		if i > 0 {
			h += l.gap
		}
		h += l.h
	}
	return h
}

// noteSep is the space over a page's notes, with their rule.
const noteSep = noteSize * 1.2

// limit is where the text must end on this page: over its notes.
func (p *pdfWriter) limit() float64 { return p.bottom - p.footHeight(p.foot) }

// finishPage draws the page's notes at its foot, under a short rule.
func (p *pdfWriter) finishPage() {
	if len(p.foot) == 0 {
		return
	}
	y := p.bottom - p.footHeight(p.foot)
	p.d.SetFillColor(140, 140, 140)
	p.d.Rect(p.left, y+noteSize*0.4, (p.right-p.left)/3, 0.5, "F")
	y += noteSep
	for i, l := range p.foot {
		if i > 0 {
			y += l.gap
		}
		l.draw(y)
		y += l.h
	}
	p.foot = nil
}

// room makes room for h points, on the next page if this one has not.
func (p *pdfWriter) room(h float64) {
	if p.y+h > p.limit()+0.01 && !p.atTop {
		p.newPage()
	}
}

// keep keeps h points together (a heading with what follows it).
func (p *pdfWriter) keep(h float64) {
	if p.collect == nil {
		p.room(h)
	}
}

// gap is space before a block (none at a page's top).
func (p *pdfWriter) gap(h float64) {
	if p.collect != nil {
		if len(*p.collect) > 0 || p.gapNext > 0 {
			p.gapNext += h
		}
		return
	}
	if !p.atTop {
		p.y += h
	}
}

// emit lays out a line h high where the text has got to, on the next
// page if it has not room for it and the notes it refers to (refs: those
// at the page's foot first appear here); draw draws it with its top at
// y. A note being laid out keeps its lines instead (collect).
func (p *pdfWriter) emit(h float64, refs []*note, draw func(y float64)) {
	decor, rtl := p.decor, p.lineRTL
	p.decor = nil
	line := func(y float64) {
		for _, d := range decor {
			d(y, rtl)
		}
		draw(y)
	}
	if p.collect != nil {
		*p.collect = append(*p.collect, footLine{gap: p.gapNext, h: h, draw: line})
		p.gapNext = 0
		*p.collected = append(*p.collected, refs...)
		return
	}
	news := p.newNotes(refs, nil)
	if len(news) == 0 {
		p.room(h)
	} else if p.y+h+p.notesRoom(news) > p.limit()+0.01 && !p.atTop {
		p.newPage()
	}
	line(p.y)
	p.y += h
	p.atTop = false
	p.placeNotes(news)
}

// newNotes are the notes of refs to put at the page's foot, not put yet,
// each with the notes it refers to after it.
func (p *pdfWriter) newNotes(refs []*note, out []*note) []*note {
	if p.doc.NotesAt != NotesAtPageFoot {
		return nil
	}
	for _, n := range refs {
		if n.placed || slices.Contains(out, n) {
			continue
		}
		out = append(out, n)
		out = p.newNotes(p.noteLines(n).nested, out)
	}
	return out
}

// notesRoom is the room notes need at the page's foot: the rule's if
// there are none yet, the notes whole, the last's first line only (it
// may go on at the next page's foot); none while a note is being carried
// on (they follow it).
func (p *pdfWriter) notesRoom(ns []*note) float64 {
	if len(p.carry) > 0 {
		return 0
	}
	var lines []footLine
	for i, n := range ns {
		ls := p.noteLines(n).lines
		if i == len(ns)-1 && len(ls) > 0 {
			ls = ls[:1]
		}
		lines = append(lines, ls...)
	}
	return p.footHeight(append(slices.Clone(p.foot), lines...)) - p.footHeight(p.foot)
}

// placeNotes puts notes at the page's foot, as much of them as there is
// room for under the text, and no more than half the page's (a note's
// first line always, with its reference); the rest is carried to the
// next page's.
func (p *pdfWriter) placeNotes(ns []*note) {
	for _, n := range ns {
		n.placed = true
		for i, l := range n.lines {
			h := p.footHeight(append(slices.Clone(p.foot), l))
			if len(p.carry) == 0 && p.y+h <= p.bottom+0.01 && (i == 0 || h <= p.footShare()) {
				p.foot = append(p.foot, l)
				continue
			}
			p.carry = append(p.carry, l)
		}
	}
}

// footShare is the most of a page its notes take while text is to come
// (all of it when only notes are left).
func (p *pdfWriter) footShare() float64 {
	if p.finished {
		return p.bottom - p.top
	}
	return (p.bottom - p.top) / 2
}

// noteLines lays a note out in lines, once: its number, then its text.
func (p *pdfWriter) noteLines(n *note) *note {
	if n.laid {
		return n
	}
	n.laid = true
	var lines []footLine
	var refs []*note
	saved, savedRefs, savedGap, savedDecor := p.collect, p.collected, p.gapNext, p.decor
	p.collect, p.collected, p.gapNext = &lines, &refs, noteSize*0.3
	p.decor = []func(float64, bool){p.noteMark(n)}
	p.noteBlocks(n.blocks, indentStp)
	if len(lines) == 0 || len(p.decor) > 0 {
		p.emit(noteSize*leading, nil, func(float64) {})
	}
	p.collect, p.collected, p.gapNext, p.decor = saved, savedRefs, savedGap, savedDecor
	lines[0].gap = noteSize * 0.3
	n.lines = lines
	for _, r := range refs {
		if r != n && !slices.Contains(n.nested, r) {
			n.nested = append(n.nested, r)
		}
	}
	return n
}

// noteMark draws a note's number before its first line (on its right,
// ".1", when it reads right to left), a link back to its reference, and
// is where the references link to.
func (p *pdfWriter) noteMark(n *note) func(y float64, rtl bool) {
	return func(y float64, rtl bool) {
		st := style{size: noteSize}
		mark := strconv.Itoa(n.num) + "."
		if rtl {
			mark = "." + strconv.Itoa(n.num)
		}
		p.setFont(faceText, st)
		mw := p.d.GetStringWidth(mark)
		x := p.left + indentStp - 6 - mw
		if rtl {
			x = p.right - indentStp + 6
		}
		p.draw(piece{text: mark, face: faceText, st: st, w: mw}, x, y, noteSize*leading)
		p.d.SetLink(n.dest, y, -1)
		if n.backSet {
			p.d.Link(x, y, mw, noteSize*leading, n.back)
		}
	}
}

// endnotes prints the notes of the chapters before chapter not printed
// yet, under a rule: after a chapter, or the text. Numbered again in each
// chapter, the notes after the text have their chapter's heading over
// them.
func (p *pdfWriter) endnotes(chapter int) {
	var list []*note
	for _, n := range p.notes {
		if !n.printed && n.chapter < chapter {
			n.printed = true
			list = append(list, n)
		}
	}
	if len(list) == 0 {
		return
	}
	p.gap(textSize)
	p.keep(3 * noteSize * leading)
	p.emit(noteSize, nil, func(y float64) {
		p.d.SetFillColor(140, 140, 140)
		p.d.Rect(p.left, y, (p.right-p.left)/3, 0.75, "F")
	})
	heads := p.doc.NotesPerChapter && list[0].chapter != list[len(list)-1].chapter
	at := -1
	for i, n := range list {
		if heads && n.chapter != at {
			at = n.chapter
			if t := p.chapterTitles[at]; t != "" {
				if i > 0 {
					p.gap(noteSize * 0.8)
				}
				p.keep(3 * noteSize * leading)
				p.flow([]Span{{Text: t}}, style{size: noteSize, bold: true}, p.left, p.right, alignLeft, nil)
			}
		}
		for j, l := range p.noteLines(n).lines {
			if i > 0 || j > 0 {
				p.gap(l.gap)
			}
			p.emit(l.h, nil, l.draw)
		}
	}
}

func (p *pdfWriter) titlePage(lines []string) {
	p.newPage()
	y := p.pageH * 0.3
	for i, l := range lines {
		st := style{size: 14}
		if i == 0 {
			st = style{size: 26, bold: true}
		}
		for _, line := range p.lines(p.pieces([]Span{{Text: l}}, st), p.right-p.left) {
			p.drawLine(line, p.left+p.centred(line, p.right-p.left), y, st.size*leading)
			y += st.size * leading
		}
		y += st.size * 0.8
	}
}

// centred is how far in a line starts to be centred in width.
func (p *pdfWriter) centred(line []piece, width float64) float64 {
	total := 0.0
	for _, pc := range line {
		total += pc.w
	}
	return (width - total) / 2
}

// style is how inline text prints.
type style struct {
	size                       float64
	bold, italic, mono, strike bool
	href                       string // where it links to ("": nowhere)
	color                      [3]int
	rise                       float64 // raised by, in points (a note's number)
}

func (s style) fontStyle() string {
	st := ""
	if s.bold {
		st += "B"
	}
	if s.italic {
		st += "I"
	}
	return st
}

type align int

const (
	alignLeft align = iota
	alignCenter
)

// piece is a run of text in one style and font, a space, or a break; a
// note's number is the note's.
type piece struct {
	text  string
	face  string
	st    style
	space bool
	brk   bool
	w     float64
	note  *note
	pic   *pic    // a picture in the line, ph high
	ph    float64 //
	off   int     // where its text starts in its paragraph's (in characters: hangingLines)
	rtl   bool    // printed right to left (a line's run at an odd level)
	// where its composed marks go (pdf.Shaper.ComposeMarks: the fatha of
	// لَا on the lam), one for each character of text; nil: not known
	marks []pdf.MarkPlace
}

// cut is the piece's characters a..b (of its text), its marks' places
// with them.
func (pc piece) cut(a, b int) piece {
	rs := []rune(pc.text)
	pc.text = string(rs[a:b])
	if len(pc.marks) == len(rs) {
		pc.marks = pc.marks[a:b:b]
	} else {
		pc.marks = nil
	}
	return pc
}

// join is the piece with next's text after its own, the marks' places
// of both (a new slice: neither's is written to).
func (pc piece) join(next piece) piece {
	a, b := utf8.RuneCountInString(pc.text), utf8.RuneCountInString(next.text)
	var marks []pdf.MarkPlace
	if pc.marks != nil || next.marks != nil {
		marks = make([]pdf.MarkPlace, a+b) // (a piece's not known: its zero places)
		if len(pc.marks) == a {
			copy(marks, pc.marks)
		}
		if len(next.marks) == b {
			copy(marks[a:], next.marks)
		}
	}
	pc.text, pc.marks = pc.text+next.text, marks
	return pc
}

// pieces are spans as runs: words and spaces, each in the face that has
// its characters, a note as its number raised.
func (p *pdfWriter) pieces(spans []Span, base style) []piece {
	var out []piece
	for i := range spans {
		s := spans[i]
		if s.Comment != "" && s.Text == "" {
			continue
		}
		st := base
		st.bold = st.bold || s.Bold
		st.italic = st.italic || s.Italic
		st.strike = s.Strike
		st.mono = s.Code
		if s.Link != "" {
			st.href, st.color = s.Link, [3]int{26, 13, 171}
		}
		if s.Break {
			out = append(out, piece{brk: true, st: st})
			continue
		}
		if s.Image != nil {
			if pc := p.pic(s.Image); pc.err == nil {
				h := st.size * 1.15 // as high as the line's letters
				out = append(out, piece{pic: pc, st: st, ph: h, w: h * float64(pc.w) / float64(pc.h)})
				continue
			}
			// its alt text in grey, [in brackets]
			alt := st
			alt.color = [3]int{110, 110, 110}
			if st.href != "" {
				alt.color = [3]int{26, 13, 171}
			}
			text := "[" + cmp.Or(strings.TrimSpace(s.Image.Alt), "picture") + "]"
			for k, word := range strings.Fields(text) {
				if k > 0 {
					out = append(out, piece{text: " ", face: p.face(' ', alt), st: alt, space: true})
				}
				out = append(out, p.runs(word, alt)...)
			}
			continue
		}
		if s.Note != nil {
			n := p.noteAt(&spans[i])
			if n == nil {
				continue
			}
			ns := st
			ns.size, ns.rise = st.size*0.7, st.size*0.35
			for _, r := range p.runs(strconv.Itoa(n.num), ns) {
				r.note = n
				out = append(out, r)
			}
			continue
		}
		text := strings.ReplaceAll(s.Text, "\n", " ")
		if !s.Code {
			text = strings.Join(strings.FieldsFunc(text, func(r rune) bool { return r == '\t' }), " ")
		}
		var word strings.Builder
		flush := func() {
			if word.Len() > 0 {
				out = append(out, p.runs(word.String(), st)...)
				word.Reset()
			}
		}
		for _, r := range text {
			if r == ' ' {
				flush()
				out = append(out, piece{text: " ", face: p.face(' ', st), st: st, space: true})
				continue
			}
			word.WriteRune(r)
		}
		flush()
	}
	// a space between words read right to left in their face (a cell of
	// Courier Badi's, as wide as its letters' spacing wants)
	for i := 1; i+1 < len(out); i++ {
		a, b := out[i-1], out[i+1]
		if out[i].space && a.face == b.face && a.face != out[i].face && p.has[a.face](' ') &&
			strings.IndexFunc(a.text, rtlLetter) >= 0 && strings.IndexFunc(b.text, rtlLetter) >= 0 {
			out[i].face = a.face
		}
	}
	for i := range out {
		if !out[i].brk && out[i].pic == nil {
			out[i].w = p.width(out[i])
		}
	}
	return out
}

// refs are the notes a line refers to.
func refs(line []piece) []*note {
	var out []*note
	for _, pc := range line {
		if pc.note != nil && !slices.Contains(out, pc.note) {
			out = append(out, pc.note)
		}
	}
	return out
}

// face is the face a character prints in: the text's or code's, else
// the other if it has it.
func (p *pdfWriter) face(r rune, st style) string {
	first, other := faceText, faceMono
	if st.mono {
		first, other = faceMono, faceText
	}
	if p.has[first](r) {
		return first
	}
	if !st.mono { // a script's face: proportional, as the text is
		for _, sf := range scriptFaces {
			if sf.covers(r) && p.has[sf.name](r) {
				return sf.name
			}
		}
	}
	if p.has[other](r) {
		return other
	}
	return first
}

// rtlLetter tells whether a character is a letter read right to left
// (Hebrew, Arabic, Syriac, Thaana…: bidi class R or AL).
func rtlLetter(r rune) bool {
	c, _ := bidi.LookupRune(r)
	return c.Class() == bidi.R || c.Class() == bidi.AL
}

// rtlRunEnd is where the run of right-to-left letters starting at rs[i]
// ends: over its letters, the marks on them and the joiners (a Persian
// word's ZWNJ), so that it is in one face, composed whole.
func rtlRunEnd(rs []rune, i int) int {
	j := i
	for j < len(rs) && (rtlLetter(rs[j]) || unicode.Is(unicode.Mn, rs[j]) || rs[j] == 0x200C || rs[j] == 0x200D) {
		j++
	}
	return j
}

// runs is a word in runs of one face each. A Devanagari run (its
// letters, signs, joiners and marks) is shaped whole, so it is in one
// face, its first letter's: the text face has the joiners, not the
// letters, and क्‍ष split there would print unjoined. So is a run of
// right-to-left letters (with their marks and joiners), composed
// (pdf.Shaper.Compose: Arabic joined, pointed Hebrew as its forms); a
// mark goes with its letter.
func (p *pdfWriter) runs(word string, st style) []piece {
	var out []piece
	rs := []rune(word)
	for i := 0; i < len(rs); {
		j := i + 1
		switch {
		case pdf.DevanagariRune(rs[i]):
			j = max(pdf.DevanagariRunEnd(rs, i), j)
		case rtlLetter(rs[i]):
			j = max(rtlRunEnd(rs, i), j)
		}
		for j < len(rs) && unicode.Is(unicode.Mn, rs[j]) {
			j++
		}
		f := p.face(p.letter(rs[i:j]), st)
		if n := len(out); n > 0 && out[n-1].face == f {
			out[n-1].text += string(rs[i:j])
		} else {
			out = append(out, piece{text: string(rs[i:j]), face: f, st: st})
		}
		i = j
	}
	// signs at a right-to-left run's ends (its word's period, comma,
	// brackets) in its face when it has them: one face with the letters,
	// as a reader takes it for one word, turned around with them
	for k := range out {
		if strings.IndexFunc(out[k].text, unicode.IsLetter) >= 0 || strings.IndexFunc(out[k].text, unicode.IsDigit) >= 0 {
			continue
		}
		for _, n := range []int{k - 1, k + 1} {
			if n < 0 || n >= len(out) || strings.IndexFunc(out[n].text, rtlLetter) < 0 {
				continue
			}
			if !strings.ContainsFunc(out[k].text, func(r rune) bool { return !p.has[out[n].face](r) }) {
				out[k].face = out[n].face
				break
			}
		}
	}
	for k := 1; k < len(out); k++ { // (runs of one face again)
		if out[k].face == out[k-1].face {
			out[k-1].text += out[k].text
			out = slices.Delete(out, k, k+1)
			k--
		}
	}
	for k := range out {
		if strings.IndexFunc(out[k].text, rtlLetter) >= 0 {
			out[k].text, out[k].marks = p.shaper(out[k].face, st).ComposeMarks(out[k].text)
		}
	}
	return out
}

// letter is the character whose face a Devanagari run (or a single
// character) prints in: its first that is not a joiner.
func (p *pdfWriter) letter(run []rune) rune {
	for _, r := range run {
		if r != 0x200C && r != 0x200D {
			return r
		}
	}
	return run[0]
}

// shaper is the shaper of a face in a style: the file given to AddFont.
func (p *pdfWriter) shaper(face string, st style) *pdf.Shaper {
	key := face + st.fontStyle()
	if s, ok := p.shapers[key]; ok {
		return s
	}
	var data []byte
	switch face {
	case faceMono:
		data = lexfont.GetFont(lexfont.CourierBadiName, st.fontStyle())
	case faceText:
		data = map[string][]byte{"": p.f.Regular, "B": p.f.Bold, "I": p.f.Italic, "BI": p.f.BoldItalic}[st.fontStyle()]
	default:
		for _, sf := range scriptFaces {
			if sf.name == face {
				data = sf.font(st.fontStyle())
			}
		}
	}
	if p.shapers == nil {
		p.shapers = map[string]*pdf.Shaper{}
	}
	s := pdf.ShaperOf(data)
	p.shapers[key] = s
	return s
}

func (p *pdfWriter) setFont(face string, st style) {
	p.d.SetFont(face, st.fontStyle(), st.size)
}

// width is a piece's width as draw prints it: its Devanagari shaped,
// without what takes no room (marks, joiners, direction controls).
func (p *pdfWriter) width(pc piece) float64 {
	p.setFont(pc.face, pc.st)
	return p.shaper(pc.face, pc.st).Width(p.d, pc.text)
}

// lines breaks pieces into lines width wide: at spaces, a word longer
// than the line broken where it must.
func (p *pdfWriter) lines(ps []piece, width float64) [][]piece {
	return p.hangingLines(ps, width, width)
}

// hangingLines are lines of ps, the first width wide, the others rest:
// wrapped in logical order, each shown in visual order (visual).
func (p *pdfWriter) hangingLines(ps []piece, first, rest float64) [][]piece {
	ps = slices.Clone(ps)
	text := paraText(ps)
	var lv []pdf.Level
	if needsBidi(text) {
		lv = pdf.Levels(text, lex.LineRTL(string(text), false))
	}
	var lines [][]piece
	var line []piece
	w := 0.0
	width := first
	end := func() {
		for len(line) > 0 && line[len(line)-1].space {
			line = line[:len(line)-1]
		}
		if lv != nil {
			line = p.visual(line, lv)
		}
		lines = append(lines, line)
		line, w = nil, 0
		width = rest
	}
	for i := 0; i < len(ps); {
		if ps[i].brk {
			end()
			i++
			continue
		}
		if ps[i].space {
			if len(line) > 0 {
				line = append(line, ps[i])
				w += ps[i].w
			}
			i++
			continue
		}
		j, ww := i, 0.0 // the word: the pieces up to the next space or break
		for ; j < len(ps) && !ps[j].space && !ps[j].brk; j++ {
			ww += ps[j].w
		}
		if w+ww <= width || len(line) == 0 && ww <= width {
			line = append(line, ps[i:j]...)
			w += ww
			i = j
			continue
		}
		if len(line) > 0 {
			end()
			continue
		}
		// a word longer than the line: as much of it as fits, broken
		// between clusters (a conjunct, a letter and its marks are one),
		// each part measured whole (shaped, its clusters' widths add up
		// to no more than it)
		for k := i; k < j; k++ {
			pc := ps[k]
			if pc.pic != nil {
				if w+pc.w > width && len(line) > 0 {
					end()
				}
				line = append(line, pc)
				w += pc.w
				continue
			}
			off, at := pc.off, 0
			for _, cluster := range clusters(pc.text) {
				size := utf8.RuneCountInString(cluster)
				c := pc.cut(at, at+size)
				c.off = off
				off, at = off+size, at+size
				c.w = p.width(c)
				n := len(line)
				joins := n > 0 && line[n-1].face == c.face && line[n-1].st == c.st && line[n-1].note == c.note && !line[n-1].space
				if joins {
					merged := line[n-1].join(c)
					merged.w = p.width(merged)
					if w-line[n-1].w+merged.w <= width {
						w += merged.w - line[n-1].w
						line[n-1] = merged
						continue
					}
				}
				if w+c.w > width && len(line) > 0 {
					end()
				}
				line = append(line, c)
				w += c.w
			}
		}
		i = j
	}
	if len(line) > 0 {
		end()
	}
	return lines
}

// paraText is the text of a paragraph's pieces, each piece's offset in
// it set: a picture is one neutral character, a break a space.
func paraText(ps []piece) []rune {
	var text []rune
	for i := range ps {
		ps[i].off = len(text)
		switch {
		case ps[i].pic != nil:
			text = append(text, 0xFFFC)
		case ps[i].brk:
			text = append(text, ' ')
		default:
			text = append(text, []rune(ps[i].text)...)
		}
	}
	return text
}

// needsBidi tells whether a paragraph has to be ordered: it has letters
// read right to left, or direction controls.
func needsBidi(text []rune) bool {
	return slices.ContainsFunc(text, func(r rune) bool {
		return rtlLetter(r) || r == 0x200E || r == 0x200F || r == 0x061C || r >= 0x202A && r <= 0x202E || r >= 0x2066 && r <= 0x2069
	})
}

// rtlPieces tells whether a paragraph's pieces read right to left: by its
// first letter (or direction mark).
func rtlPieces(ps []piece) bool {
	var b strings.Builder
	for _, pc := range ps {
		b.WriteString(pc.text)
	}
	return lex.LineRTL(b.String(), false)
}

// visual is a line of a paragraph (its pieces in logical order) in the
// order it is shown, left to right, by the paragraph's levels lv
// (pdf.VisualRuns): a piece split where its levels change, each part of
// a right-to-left run marked to print so (pdf.Shaper.CellRTL).
func (p *pdfWriter) visual(line []piece, lv []pdf.Level) []piece {
	if len(line) == 0 {
		return line
	}
	size := func(pc piece) int {
		if pc.pic != nil {
			return 1
		}
		return utf8.RuneCountInString(pc.text)
	}
	from, last := line[0].off, line[len(line)-1]
	to := last.off + size(last)
	var out []piece
	for _, r := range pdf.VisualRuns(lv, from, to) {
		var parts []piece
		for _, pc := range line {
			n := size(pc)
			a, b := max(r.Start, pc.off), min(r.End, pc.off+n)
			if a >= b {
				continue
			}
			part := pc
			if pc.pic == nil && (a > pc.off || b < pc.off+n) {
				part = pc.cut(a-pc.off, b-pc.off)
				part.off = a
				part.w = p.width(part)
			}
			part.rtl = r.RTL && pc.pic == nil
			parts = append(parts, part)
		}
		if r.RTL {
			slices.Reverse(parts)
		}
		out = append(out, parts...)
	}
	return out
}

// lineWidth is a line's width.
func lineWidth(line []piece) float64 {
	w := 0.0
	for _, pc := range line {
		w += pc.w
	}
	return w
}

// clusters are text's clusters: a character with the marks and joiners
// on it, a Devanagari conjunct (pdf.JoinsBefore), where a word too long
// for its line may break.
func clusters(text string) []string {
	var out []string
	var prev rune
	for i, r := range text {
		if len(out) > 0 && pdf.JoinsBefore(prev, r) {
			out[len(out)-1] += text[i : i+utf8.RuneLen(r)]
		} else {
			out = append(out, string(r))
		}
		prev = r
	}
	return out
}

// paragraph prints spans from left to right where the text has got to,
// wrapping, on as many pages as it takes.
func (p *pdfWriter) paragraph(spans []Span, base style, left, right float64, a align) {
	p.flow(spans, base, left, right, a, nil)
}

// flow lays spans out from left to right where the text has got to,
// wrapping, a line at a time (emit); first, if set, is told where the
// first line's top is, on its page.
func (p *pdfWriter) flow(spans []Span, base style, left, right float64, a align, first func(y float64)) {
	p.flowHanging(spans, base, left, right, 0, a, first)
}

// flowHanging is flow with the lines after the first indented by hang.
// A paragraph read right to left is set from the right: its indents
// (left of the text column's) on the right, its lines right-aligned.
func (p *pdfWriter) flowHanging(spans []Span, base style, left, right, hang float64, a align, first func(y float64)) {
	lh := base.size * leading
	ps := p.pieces(spans, base)
	rtl := rtlPieces(ps)
	if rtl {
		left, right = p.left+p.right-right, p.left+p.right-left
	}
	defer func(was bool) { p.lineRTL = was }(p.lineRTL)
	for i, line := range p.hangingLines(ps, right-left, right-left-hang) {
		x := left
		if i > 0 && !rtl {
			x += hang
		}
		switch {
		case a == alignCenter:
			x += p.centred(line, right-left)
		case rtl:
			x = right - lineWidth(line)
			if i > 0 {
				x -= hang
			}
		}
		p.lineRTL = rtl // (set again: a note laid out by the line before sets its own)
		p.emit(lh, refs(line), func(y float64) {
			if i == 0 && first != nil {
				first(y)
			}
			p.drawLine(line, x, y, lh)
		})
	}
}

// drawLine draws a line's pieces from x on the line at y, lh high, and
// links them: a link's text where it is on the line, a note's number to
// the note (the note's number back to its first reference).
func (p *pdfWriter) drawLine(line []piece, x, y, lh float64) {
	for i := 0; i < len(line); {
		pc := line[i]
		if pc.note != nil {
			p.draw(pc, x, y, lh)
			p.d.Link(x, y-pc.st.rise, pc.w, lh, pc.note.dest)
			if !pc.note.backSet {
				pc.note.backSet = true
				p.d.SetLink(pc.note.back, y, -1)
			}
			x += pc.w
			i++
			continue
		}
		at, w, j := x, 0.0, i
		for ; j < len(line) && line[j].note == nil && line[j].st.href == pc.st.href; j++ {
			p.draw(line[j], x, y, lh)
			x += line[j].w
			w += line[j].w
		}
		p.link(pc.st.href, at, y, w, lh)
		i = j
	}
}

// link links a rectangle to href: a heading's anchor (#…) in the
// document, or a web or mail address (http, https, mailto); nothing for an
// anchor no heading has, nor for any other scheme (javascript:, file:, …),
// which a PDF reader might act on.
func (p *pdfWriter) link(href string, x, y, w, h float64) {
	switch {
	case href == "" || w <= 0:
	case strings.HasPrefix(href, "#"):
		if id, ok := p.anchors[href[1:]]; ok {
			p.d.Link(x, y, w, h, id)
		}
	case safeAddress(href):
		p.d.LinkString(x, y, w, h, href)
	}
}

// safeAddress reports whether href is a web or mail address.
func safeAddress(href string) bool {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	case "mailto":
		return u.Opaque != "" || u.Path != ""
	}
	return false
}

// draw prints a piece at x on the line at y, lh high.
func (p *pdfWriter) draw(pc piece, x, y, lh float64) {
	d := p.d
	if pc.pic != nil { // on the line's baseline, a little below
		base := y + lh/2 + 0.3*pc.st.size
		pdf.DrawImage(d, pc.pic.name, x, base+0.15*pc.st.size-pc.ph, pc.w, pc.ph)
		return
	}
	p.setFont(pc.face, pc.st)
	d.SetTextColor(pc.st.color[0], pc.st.color[1], pc.st.color[2])
	d.SetXY(x, y-pc.st.rise)
	// shaped, its text read back as written; right to left turned around
	if sh := p.shaper(pc.face, pc.st); pc.rtl {
		sh.CellRTLMarks(d, pc.face, pc.st.fontStyle(), pc.st.size, pc.w, lh, pc.text, pc.marks, "L")
	} else {
		sh.Cell(d, pc.face, pc.st.fontStyle(), pc.st.size, pc.w, lh, pc.text, "L")
	}
	base := y + lh/2 + 0.3*pc.st.size // gofpdf's baseline in a cell
	if pc.st.href != "" || pc.st.strike {
		d.SetFillColor(pc.st.color[0], pc.st.color[1], pc.st.color[2])
		if pc.st.href != "" {
			d.Rect(x, base+pc.st.size*0.12, pc.w, pc.st.size*0.06, "F")
		}
		if pc.st.strike {
			d.Rect(x, base-pc.st.size*0.3, pc.w, pc.st.size*0.06, "F")
		}
	}
	d.SetTextColor(0, 0, 0)
}

// blocks prints blocks indent points in; quoted text is in italic.
func (p *pdfWriter) blocks(bs []*Block, indent float64, quoted bool) {
	for i, b := range bs {
		p.block(b, indent, quoted, i == 0)
	}
}

func (p *pdfWriter) block(b *Block, indent float64, quoted, first bool) {
	left := p.left + indent
	body := style{size: textSize, italic: quoted}
	switch b.Kind {
	case Para:
		if !first {
			p.gap(textSize * 0.6)
		}
		if img := b.Picture(); img != nil {
			p.picture(img, b.Spans, left)
			return
		}
		if b.Hanging { // a bibliography's entry
			p.flowHanging(b.Spans, body, left, p.right, 2*indentStp, alignLeft, nil)
			return
		}
		p.paragraph(b.Spans, body, left, p.right, alignLeft)
	case Heading:
		lv := min(max(b.Level, 1), 6)
		st := style{size: headingSizes[lv], bold: true, italic: quoted || lv == 6}
		if !first {
			p.gap(st.size * 0.9)
		}
		// with two lines of what follows on its page
		p.keep(st.size*leading + 2*textSize*leading)
		p.flow(b.Spans, st, left, p.right, alignLeft, p.headingPlace(b, indent == 0 && !quoted && p.collect == nil))
		p.gap(st.size * 0.2)
	case Quote:
		if !first {
			p.gap(textSize * 0.6)
		}
		p.blocks(b.Children, indent+indentStp, true)
	case List:
		if !first {
			p.gap(textSize * 0.6)
		}
		n := b.Start
		if n == 0 {
			n = 1
		}
		for i, it := range b.Children {
			num := strconv.Itoa(n + i)
			if i > 0 {
				p.gap(textSize * 0.25)
			}
			p.keep(textSize * leading)
			// the mark with the item's first line, wherever it lands: on its
			// right when it reads right to left (".1")
			p.decor = append(p.decor, func(y float64, rtl bool) {
				mark := "•"
				switch {
				case b.Ordered && rtl:
					mark = "." + num
				case b.Ordered:
					mark = num + "."
				}
				p.setFont(faceText, body)
				mw := p.d.GetStringWidth(mark)
				x := left + indentStp - 6 - mw
				if rtl {
					x = p.right - (left - p.left) - indentStp + 6
				}
				p.draw(piece{text: mark, face: faceText, st: body, w: mw}, x, y, textSize*leading)
			})
			if len(it.Children) == 0 {
				p.emit(textSize*leading, nil, func(float64) {})
			}
			p.blocks(it.Children, indent+indentStp, quoted)
		}
	case Code:
		if !first {
			p.gap(textSize * 0.6)
		}
		st := style{size: codeSize, mono: true}
		lh := codeSize * leading
		for _, l := range strings.Split(strings.TrimRight(b.Text, "\n"), "\n") {
			l = strings.ReplaceAll(l, "\t", "    ")
			ps := p.runs(l, st)
			for k := range ps {
				ps[k].w = p.width(ps[k])
			}
			lines := p.lines(ps, p.right-left-8)
			if len(lines) == 0 {
				lines = [][]piece{nil}
			}
			for _, line := range lines {
				p.emit(lh, nil, func(y float64) {
					p.d.SetFillColor(242, 242, 242)
					p.d.Rect(left, y, p.right-left, lh, "F")
					p.drawLine(line, left+4, y, lh)
				})
			}
		}
	case Rule:
		p.gap(textSize * 0.6)
		p.emit(textSize*leading, nil, func(y float64) {
			w := (p.right - left) / 3
			p.d.SetFillColor(140, 140, 140)
			p.d.Rect(left+(p.right-left-w)/2, y+textSize*leading/2, w, 0.75, "F")
		})
	case Table:
		if !first {
			p.gap(textSize * 0.6)
		}
		p.table(b, left)
	}
}

// picture prints a picture on a line of its own, from left to the text's
// right edge: as wide as that at most and never larger than it is (at 96
// dpi), on one page, its caption under it. One that cannot be read is a
// grey box with its alt text and where it is (a link, for one on the web).
func (p *pdfWriter) picture(img *Image, spans []Span, left float64) {
	width := p.right - left
	caption := img.Title
	if caption == "" && p.doc.AltCaptions {
		caption = img.Alt
	}
	capSt := style{size: noteSize, italic: true, color: [3]int{90, 90, 90}}
	href := ""
	for _, s := range spans {
		if s.Image == img {
			href = s.Link
		}
	}
	pc := p.pic(img)
	if pc.err != nil {
		lh := noteSize * leading
		label := cmp.Or(strings.TrimSpace(img.Alt), "Picture")
		where := img.Src
		if strings.HasPrefix(strings.ToLower(where), "data:") {
			where = "an embedded picture"
		}
		if len(where) > 120 {
			where = where[:117] + "…"
		}
		st := style{size: noteSize, color: [3]int{110, 110, 110}}
		lines := p.lines(p.pieces([]Span{{Text: label, Italic: true}}, st), width-16)
		srcSt := st
		if images.IsRemote(img.Src) && safeAddress(img.Src) {
			srcSt.href, srcSt.color = img.Src, [3]int{26, 13, 171}
		}
		lines = append(lines, p.lines(p.pieces([]Span{{Text: where + " (" + images.Why(pc.err) + ")"}}, srcSt), width-16)...)
		h := float64(len(lines))*lh + 12
		p.emit(h, nil, func(y float64) {
			p.d.SetDrawColor(190, 190, 190)
			p.d.SetLineWidth(0.5)
			p.d.Rect(left, y+1, width, h-2, "D")
			ly := y + 6
			for _, l := range lines {
				p.drawLine(l, left+8, ly, lh)
				ly += lh
			}
		})
		return
	}
	w := math.Min(width, float64(pc.w)*0.75)
	h := w * float64(pc.h) / float64(pc.w)
	capLines := [][]piece(nil)
	if caption != "" {
		capLines = p.lines(p.pieces([]Span{{Text: caption}}, capSt), width)
	}
	capH := float64(len(capLines)) * noteSize * leading
	if most := p.limit() - p.top - capH - 4; h > most && most > 0 { // a page's height at most
		h = most
		w = h * float64(pc.w) / float64(pc.h)
	}
	p.room(h + capH + 4)
	x := left + (width-w)/2
	p.emit(h, nil, func(y float64) {
		pdf.DrawImage(p.d, pc.name, x, y, w, h)
		p.link(href, x, y, w, h)
	})
	if caption != "" {
		p.gap(4)
		p.flow([]Span{{Text: caption}}, capSt, left, p.right, alignCenter, nil)
	}
}

// headingPlace is told where a heading's first line is: the place its
// anchor's links go to, and (outline) its entry in the PDF's outline.
func (p *pdfWriter) headingPlace(b *Block, outline bool) func(y float64) {
	id, linked := p.anchorOf[b]
	if !linked && !outline {
		return nil
	}
	return func(y float64) {
		if linked {
			p.d.SetLink(id, y, -1)
		}
		if outline {
			// a level under the one before at most (the outline has no gaps)
			lv := min(max(b.Level-p.minLevel, 0), p.outline+1)
			p.outline = lv
			p.d.Bookmark(spansText(b.Spans), lv, y)
		}
	}
}

// table prints a table: its columns as wide as their text needs, shared
// out when the page is too narrow; the header row in bold on grey; a row
// whole on a page.
func (p *pdfWriter) table(b *Block, left float64) {
	cols := 0
	for _, r := range b.Rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return
	}
	const pad = 4.0
	avail := p.right - left
	natural := make([]float64, cols)
	for ri, r := range b.Rows {
		for c, cell := range r {
			st := style{size: textSize, bold: ri == 0}
			w := 0.0
			for _, pc := range p.pieces(cell, st) {
				w += pc.w
			}
			natural[c] = max(natural[c], w+2*pad)
		}
	}
	total := 0.0
	for _, w := range natural {
		total += w
	}
	widths := natural
	if total > avail { // shared out: no column under its share's half
		widths = make([]float64, cols)
		even := avail / float64(cols)
		for c := range widths {
			widths[c] = math.Max(natural[c]/total*avail, math.Min(natural[c], even/2))
		}
		sum := 0.0
		for _, w := range widths {
			sum += w
		}
		for c := range widths {
			widths[c] *= avail / sum
		}
	}
	lh := textSize * leading
	// a table read right to left (its first row's first letter) runs from
	// the right
	var first []piece
	for _, cell := range b.Rows[0] {
		first = append(first, p.pieces(cell, style{size: textSize, bold: true})...)
	}
	rtl := rtlPieces(first)
	for ri, r := range b.Rows {
		st := style{size: textSize, bold: ri == 0}
		cells := make([][][]piece, cols)
		cellRTL := make([]bool, cols)
		h := lh
		var notes []*note
		for c := 0; c < cols; c++ {
			var spans []Span
			if c < len(r) {
				spans = r[c]
			}
			ps := p.pieces(spans, st)
			cells[c], cellRTL[c] = p.lines(ps, widths[c]-2*pad), rtlPieces(ps)
			h = max(h, float64(len(cells[c]))*lh)
			for _, line := range cells[c] {
				for _, n := range refs(line) {
					if !slices.Contains(notes, n) {
						notes = append(notes, n)
					}
				}
			}
		}
		h += 2 * pad
		p.emit(h, notes, func(top float64) {
			x := left
			if rtl { // its first column on the right, from the right edge (mirrored)
				x = p.left + p.right - left
			}
			p.d.SetDrawColor(160, 160, 160)
			p.d.SetLineWidth(0.5)
			for c := 0; c < cols; c++ {
				if rtl {
					x -= widths[c]
				}
				if ri == 0 {
					p.d.SetFillColor(235, 235, 235)
					p.d.Rect(x, top, widths[c], h, "FD")
				} else {
					p.d.Rect(x, top, widths[c], h, "D")
				}
				y := top + pad
				for _, line := range cells[c] {
					at := x + pad
					if cellRTL[c] { // set from the right
						at = x + widths[c] - pad - lineWidth(line)
					}
					p.drawLine(line, at, y, lh)
					y += lh
				}
				if !rtl {
					x += widths[c]
				}
			}
		})
	}
}

// noteBlocks prints a note's blocks in the notes' size.
func (p *pdfWriter) noteBlocks(bs []*Block, indent float64) {
	for i, b := range bs {
		if b.Kind == Para || b.Kind == Heading {
			if i > 0 {
				p.gap(noteSize * 0.5)
			}
			p.paragraph(b.Spans, style{size: noteSize}, p.left+indent, p.right, alignLeft)
			continue
		}
		p.block(b, indent, false, i == 0)
	}
}

// pageNumbers numbers the pages at their foot (not a title page).
func (p *pdfWriter) pageNumbers(titlePage bool) {
	d := p.d
	n := d.PageCount()
	first := 1
	if titlePage {
		first = 2
	}
	if n-first < 1 { // one page: no number
		return
	}
	st := style{size: 9, color: [3]int{90, 90, 90}}
	for i := first; i <= n; i++ {
		d.SetPage(i)
		s := strconv.Itoa(i - first + 1)
		p.setFont(faceText, st)
		w := d.GetStringWidth(s)
		p.draw(piece{text: s, face: faceText, st: st, w: w}, (p.pageW-w)/2, p.bottom+(p.pageH-p.bottom)/2-9, 18)
	}
}

package prose

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"

	lexfont "github.com/LaPingvino/lexington/font"
	"github.com/LaPingvino/lexington/pdf"
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
// grey, a break as a short rule, tables with their header, the
// footnotes numbered at the end under a rule; page numbers at the foot.
//
// Not yet: text read right to left (Hebrew, Arabic: their order on the
// line), scripts no font here has (CJK…), images (their alt text is in
// the model, as in Word and ODT). PDFMissing says what a document has of
// those; the caller prints such a document through a browser instead.

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
// text read right to left, characters no font of the PDF has ("" if it
// prints well).
func (d *Doc) PDFMissing(f PDFFonts) string {
	f = f.orGo()
	text, mono := glyphs(f.Regular), glyphs(lexfont.GetFont(lexfont.CourierBadiName, ""))
	var missing []rune
	seen := map[rune]bool{}
	rtl := false
	for _, r := range d.Plain() + strings.Join(d.TitlePage, "\n") {
		if p, _ := bidi.LookupRune(r); p.Class() == bidi.R || p.Class() == bidi.AL {
			rtl = true
		}
		if unicode.IsSpace(r) || unicode.IsControl(r) || seen[r] || text(r) || mono(r) {
			continue
		}
		seen[r] = true
		missing = append(missing, r)
	}
	var why []string
	if rtl {
		why = append(why, "text read right to left")
	}
	if len(missing) > 0 {
		s := string(missing)
		if len(missing) > 12 {
			s = string(missing[:12]) + "…"
		}
		why = append(why, "characters its fonts have not got ("+s+")")
	}
	return strings.Join(why, "; ")
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
	p := &pdfWriter{d: rec, f: f}
	p.setup()
	if d.Title != "" {
		rec.SetTitle(d.Title, true)
	}
	if len(d.TitlePage) > 0 {
		p.titlePage(d.TitlePage)
	}
	p.newPage()
	p.blocks(d.Blocks, 0, false)
	p.footnotes()
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
	has                      map[string]func(rune) bool
	pageW, pageH             float64
	left, right, top, bottom float64
	y                        float64
	atTop                    bool // nothing printed on the page yet
	notes                    [][]*Block
}

func (p *pdfWriter) setup() {
	d := p.d
	for _, s := range []struct {
		style string
		data  []byte
	}{{"", p.f.Regular}, {"B", p.f.Bold}, {"I", p.f.Italic}, {"BI", p.f.BoldItalic}} {
		d.AddUTF8FontFromBytes(faceText, s.style, s.data)
		d.AddUTF8FontFromBytes(faceMono, s.style, lexfont.GetFont(lexfont.CourierBadiName, s.style))
	}
	p.has = map[string]func(rune) bool{faceText: glyphs(p.f.Regular), faceMono: glyphs(lexfont.GetFont(lexfont.CourierBadiName, ""))}
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

func (p *pdfWriter) newPage() {
	p.d.AddPage()
	p.y, p.atTop = p.top, true
}

// room makes room for h points, on the next page if this one has not.
func (p *pdfWriter) room(h float64) {
	if p.y+h > p.bottom+0.01 && !p.atTop {
		p.newPage()
	}
}

// gap is space before a block (none at a page's top).
func (p *pdfWriter) gap(h float64) {
	if !p.atTop {
		p.y += h
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
		y = p.place([]Span{{Text: l}}, st, p.left, p.right, y, alignCenter, false)
		y += st.size * 0.8
	}
}

// style is how inline text prints.
type style struct {
	size                       float64
	bold, italic, mono, strike bool
	link                       bool
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

// piece is a run of text in one style and font, a space, or a break.
type piece struct {
	text  string
	face  string
	st    style
	space bool
	brk   bool
	w     float64
}

// pieces are spans as runs: words and spaces, each in the face that has
// its characters, a note as its number raised.
func (p *pdfWriter) pieces(spans []Span, base style) []piece {
	var out []piece
	for _, s := range spans {
		if s.Comment != "" && s.Text == "" {
			continue
		}
		st := base
		st.bold = st.bold || s.Bold
		st.italic = st.italic || s.Italic
		st.strike = s.Strike
		st.mono = s.Code
		if s.Link != "" {
			st.link, st.color = true, [3]int{26, 13, 171}
		}
		if s.Break {
			out = append(out, piece{brk: true, st: st})
			continue
		}
		if s.Note != nil {
			p.notes = append(p.notes, s.Note)
			n := st
			n.size, n.rise = st.size*0.7, st.size*0.35
			out = append(out, p.runs(strconv.Itoa(len(p.notes)), n)...)
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
	for i := range out {
		if !out[i].brk {
			out[i].w = p.width(out[i])
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
	if p.has[first](r) || !p.has[other](r) {
		return first
	}
	return other
}

// runs is a word in runs of one face each.
func (p *pdfWriter) runs(word string, st style) []piece {
	var out []piece
	for _, r := range word {
		f := p.face(r, st)
		if n := len(out); n > 0 && out[n-1].face == f {
			out[n-1].text += string(r)
			continue
		}
		out = append(out, piece{text: string(r), face: f, st: st})
	}
	return out
}

func (p *pdfWriter) setFont(face string, st style) {
	p.d.SetFont(face, st.fontStyle(), st.size)
}

func (p *pdfWriter) width(pc piece) float64 {
	p.setFont(pc.face, pc.st)
	return p.d.GetStringWidth(pc.text)
}

// lines breaks pieces into lines width wide: at spaces, a word longer
// than the line broken where it must.
func (p *pdfWriter) lines(ps []piece, width float64) [][]piece {
	var lines [][]piece
	var line []piece
	w := 0.0
	end := func() {
		for len(line) > 0 && line[len(line)-1].space {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
		line, w = nil, 0
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
		// a word longer than the line: as much of it as fits
		for k := i; k < j; k++ {
			pc := ps[k]
			for _, r := range pc.text {
				c := pc
				c.text = string(r)
				c.w = p.width(c)
				if w+c.w > width && len(line) > 0 {
					end()
				}
				if n := len(line); n > 0 && line[n-1].face == c.face && line[n-1].st == c.st && !line[n-1].space {
					line[n-1].text += c.text
					line[n-1].w += c.w
				} else {
					line = append(line, c)
				}
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

// paragraph prints spans from left to right where the text has got to,
// wrapping, on as many pages as it takes.
func (p *pdfWriter) paragraph(spans []Span, base style, left, right float64, a align) {
	p.place(spans, base, left, right, p.y, a, true)
}

// place prints spans from left to right at y, wrapping (on the flow of
// the pages if follow), and gives the y after them.
func (p *pdfWriter) place(spans []Span, base style, left, right, y float64, a align, follow bool) float64 {
	ps := p.pieces(spans, base)
	lh := base.size * leading
	lines := p.lines(ps, right-left)
	for _, line := range lines {
		if follow {
			p.y = y
			p.room(lh)
			y = p.y
		}
		x := left
		if a == alignCenter {
			total := 0.0
			for _, pc := range line {
				total += pc.w
			}
			x += (right - left - total) / 2
		}
		for _, pc := range line {
			p.draw(pc, x, y, lh)
			x += pc.w
		}
		y += lh
		if follow {
			p.y, p.atTop = y, false
		}
	}
	return y
}

// draw prints a piece at x on the line at y, lh high.
func (p *pdfWriter) draw(pc piece, x, y, lh float64) {
	d := p.d
	p.setFont(pc.face, pc.st)
	d.SetTextColor(pc.st.color[0], pc.st.color[1], pc.st.color[2])
	d.SetXY(x, y-pc.st.rise)
	d.CellFormat(pc.w, lh, pc.text, "", 0, "L", false, 0, "")
	base := y + lh/2 + 0.3*pc.st.size // gofpdf's baseline in a cell
	if pc.st.link || pc.st.strike {
		d.SetFillColor(pc.st.color[0], pc.st.color[1], pc.st.color[2])
		if pc.st.link {
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
		p.paragraph(b.Spans, body, left, p.right, alignLeft)
	case Heading:
		lv := min(max(b.Level, 1), 6)
		st := style{size: headingSizes[lv], bold: true, italic: quoted || lv == 6}
		if !first {
			p.gap(st.size * 0.9)
		}
		// with two lines of what follows on its page
		p.room(st.size*leading + 2*textSize*leading)
		p.paragraph(b.Spans, st, left, p.right, alignLeft)
		p.y += st.size * 0.2
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
			mark := "•"
			if b.Ordered {
				mark = strconv.Itoa(n+i) + "."
			}
			if i > 0 {
				p.gap(textSize * 0.25)
			}
			p.room(textSize * leading)
			at := p.y
			p.setFont(faceText, body)
			mw := p.d.GetStringWidth(mark)
			p.draw(piece{text: mark, face: faceText, st: body, w: mw}, left+indentStp-6-mw, at, textSize*leading)
			if len(it.Children) == 0 {
				p.y, p.atTop = at+textSize*leading, false
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
				p.room(lh)
				p.d.SetFillColor(242, 242, 242)
				p.d.Rect(left, p.y, p.right-left, lh, "F")
				x := left + 4
				for _, pc := range line {
					p.draw(pc, x, p.y, lh)
					x += pc.w
				}
				p.y += lh
				p.atTop = false
			}
		}
	case Rule:
		p.gap(textSize * 0.6)
		p.room(textSize * leading)
		w := (p.right - left) / 3
		p.d.SetFillColor(140, 140, 140)
		p.d.Rect(left+(p.right-left-w)/2, p.y+textSize*leading/2, w, 0.75, "F")
		p.y += textSize * leading
		p.atTop = false
	case Table:
		if !first {
			p.gap(textSize * 0.6)
		}
		p.table(b, left)
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
	for ri, r := range b.Rows {
		st := style{size: textSize, bold: ri == 0}
		cells := make([][][]piece, cols)
		h := lh
		for c := 0; c < cols; c++ {
			var spans []Span
			if c < len(r) {
				spans = r[c]
			}
			cells[c] = p.lines(p.pieces(spans, st), widths[c]-2*pad)
			h = max(h, float64(len(cells[c]))*lh)
		}
		h += 2 * pad
		p.room(h)
		x := left
		p.d.SetDrawColor(160, 160, 160)
		p.d.SetLineWidth(0.5)
		for c := 0; c < cols; c++ {
			if ri == 0 {
				p.d.SetFillColor(235, 235, 235)
				p.d.Rect(x, p.y, widths[c], h, "FD")
			} else {
				p.d.Rect(x, p.y, widths[c], h, "D")
			}
			y := p.y + pad
			for _, line := range cells[c] {
				lx := x + pad
				for _, pc := range line {
					p.draw(pc, lx, y, lh)
					lx += pc.w
				}
				y += lh
			}
			x += widths[c]
		}
		p.y += h
		p.atTop = false
	}
}

// footnotes prints the notes at the end, under a rule, numbered.
func (p *pdfWriter) footnotes() {
	if len(p.notes) == 0 {
		return
	}
	p.gap(textSize)
	p.room(3 * noteSize * leading)
	p.d.SetFillColor(140, 140, 140)
	p.d.Rect(p.left, p.y, (p.right-p.left)/3, 0.75, "F")
	p.y += noteSize
	st := style{size: noteSize}
	for i := 0; i < len(p.notes); i++ { // (a note may add notes)
		mark := fmt.Sprintf("%d.", i+1)
		p.room(noteSize * leading)
		p.setFont(faceText, st)
		mw := p.d.GetStringWidth(mark)
		p.draw(piece{text: mark, face: faceText, st: st, w: mw}, p.left+indentStp-6-mw, p.y, noteSize*leading)
		p.noteBlocks(p.notes[i], indentStp)
		p.gap(noteSize * 0.3)
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

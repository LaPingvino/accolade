package prose

import (
	"archive/zip"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// ReadDOCX reads a Word document: its paragraphs, headings (by their
// style, whatever its language: its name or outline level), lists,
// quotes, tables, bold, italic, links and footnotes. Images and other
// objects are left out, a comment in their place.
func ReadDOCX(r io.ReaderAt, size int64) (*Doc, error) {
	z, err := zip.NewReader(r, size)
	if err != nil {
		return nil, errors.New("not a Word document (.docx): " + err.Error())
	}
	doc, err := zipPart(z, "word/document.xml")
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, errors.New("not a Word document (.docx): it has no word/document.xml")
	}
	x := &docxReader{styles: map[string]*docxStyle{}, lists: map[string]map[int]bool{}, links: map[string]string{}, notes: map[string]*node{}}
	if n, err := zipPart(z, "word/styles.xml"); err == nil {
		x.readStyles(n)
	}
	if n, err := zipPart(z, "word/numbering.xml"); err == nil {
		x.readNumbering(n)
	}
	if n, err := zipPart(z, "word/_rels/document.xml.rels"); err == nil && n != nil {
		for _, rel := range n.all("Relationship") {
			if strings.HasSuffix(rel.attr("Type"), "/hyperlink") {
				x.links[rel.attr("Id")] = rel.attr("Target")
			}
		}
	}
	for _, part := range []string{"word/footnotes.xml", "word/endnotes.xml"} {
		if n, err := zipPart(z, part); err == nil && n != nil {
			for _, f := range n.Children {
				if t := f.attr("type"); f.Name.Local != "" && t != "separator" && t != "continuationSeparator" && t != "continuationNotice" {
					x.notes[f.Name.Local+":"+f.attr("id")] = f
				}
			}
		}
	}
	d := &Doc{}
	if core, err := zipPart(z, "docProps/core.xml"); err == nil && core != nil {
		d.Title = strings.TrimSpace(core.child("title").text())
	}
	x.doc = d
	body := doc.find("body")
	d.Blocks = x.body(body)
	return d, nil
}

type docxStyle struct {
	name, basedOn string
	outline       int // the heading level, 0 for none
	bold, italic  *bool
	numID         string
	numLevel      int
	code          bool
}

type docxReader struct {
	doc    *Doc
	styles map[string]*docxStyle
	lists  map[string]map[int]bool // numId: the levels that are numbered (not bullets)
	links  map[string]string
	notes  map[string]*node // "footnote:3", "endnote:1"
}

var headingName = regexp.MustCompile(`(?i)^(heading|überschrift|titre|kop|título|titolo|rubrik|nagłówek|заголовок)\s*(\d)$`)

func (x *docxReader) readStyles(n *node) {
	if n == nil {
		return
	}
	for _, s := range n.all("style") {
		st := &docxStyle{name: s.child("name").attr("val"), basedOn: s.child("basedOn").attr("val")}
		if p := s.child("pPr"); p != nil {
			if o := p.child("outlineLvl"); o != nil {
				if v, err := strconv.Atoi(o.attr("val")); err == nil && v < 9 {
					st.outline = v + 1
				}
			}
			if np := p.child("numPr"); np != nil {
				st.numID = np.child("numId").attr("val")
				st.numLevel, _ = strconv.Atoi(np.child("ilvl").attr("val"))
			}
		}
		if m := headingName.FindStringSubmatch(st.name); m != nil {
			st.outline, _ = strconv.Atoi(m[2])
		}
		if strings.EqualFold(st.name, "title") {
			st.outline = 1
		}
		if rp := s.child("rPr"); rp != nil {
			st.bold, st.italic = onOff(rp.child("b")), onOff(rp.child("i"))
			if f := rp.child("rFonts"); f != nil && monospace(f.attr("ascii")) {
				st.code = true
			}
		}
		x.styles[s.attr("styleId")] = st
	}
}

// onOff is a w:b or w:i: nil if absent, else on unless its value says off.
func onOff(n *node) *bool {
	if n == nil {
		return nil
	}
	v := n.attr("val")
	on := v != "0" && v != "false" && v != "off" && v != "none"
	return &on
}

func monospace(font string) bool {
	f := strings.ToLower(font)
	return strings.Contains(f, "courier") || strings.Contains(f, "mono") || strings.Contains(f, "consolas") || strings.Contains(f, "menlo")
}

func (x *docxReader) readNumbering(n *node) {
	if n == nil {
		return
	}
	abstract := map[string]map[int]bool{}
	for _, a := range n.all("abstractNum") {
		levels := map[int]bool{}
		for _, l := range a.all("lvl") {
			i, _ := strconv.Atoi(l.attr("ilvl"))
			f := l.child("numFmt").attr("val")
			levels[i] = f != "bullet" && f != "none" && f != ""
		}
		abstract[a.attr("abstractNumId")] = levels
	}
	for _, num := range n.all("num") {
		x.lists[num.attr("numId")] = abstract[num.child("abstractNumId").attr("val")]
	}
}

// style is a paragraph style with what it is based on: its heading
// level, whether it is a quote, its list.
func (x *docxReader) style(id string) (outline int, quote bool, numID string, numLevel int, code bool) {
	for i := 0; id != "" && i < 20; i++ {
		st := x.styles[id]
		if st == nil {
			break
		}
		if outline == 0 {
			outline = st.outline
		}
		name := strings.ToLower(st.name)
		if strings.Contains(name, "quote") || name == "block text" || strings.Contains(name, "citation") || strings.Contains(name, "zitat") {
			quote = true
		}
		if numID == "" && st.numID != "" {
			numID, numLevel = st.numID, st.numLevel
		}
		if st.code || strings.Contains(name, "code") || strings.Contains(name, "preformatted") {
			code = true
		}
		id = st.basedOn
	}
	return
}

// runStyle is a character style's bold and italic, with what it is
// based on.
func (x *docxReader) runStyle(id string) (bold, italic *bool, code bool) {
	for i := 0; id != "" && i < 20; i++ {
		st := x.styles[id]
		if st == nil {
			break
		}
		if bold == nil {
			bold = st.bold
		}
		if italic == nil {
			italic = st.italic
		}
		name := strings.ToLower(st.name)
		if bold == nil && (name == "strong" || name == "strong emphasis") {
			t := true
			bold = &t
		}
		if italic == nil && name == "emphasis" {
			t := true
			italic = &t
		}
		code = code || st.code || strings.Contains(name, "code") || strings.Contains(name, "source text")
		id = st.basedOn
	}
	return
}

// docxPara is a paragraph read, before lists and quotes are put together.
type docxPara struct {
	block    *Block
	numID    string
	level    int // the list's level
	quote    bool
	code     bool
	codeText string
	indented bool // indented: in a list, the item's next paragraph
}

// body reads the blocks of a body (the document's, a cell's, a note's).
func (x *docxReader) body(n *node) []*Block {
	var paras []docxPara
	var walk func(n *node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		for _, c := range n.Children {
			switch c.Name.Local {
			case "p":
				paras = append(paras, x.para(c))
			case "tbl":
				paras = append(paras, docxPara{block: x.table(c)})
			case "sdt", "customXml", "ins", "smartTag":
				walk(c.child("sdtContent"))
				if c.Name.Local != "sdt" {
					walk(c)
				}
			}
		}
	}
	if n != nil {
		walk(n)
	}
	return x.assemble(paras)
}

func (x *docxReader) para(p *node) docxPara {
	pPr := p.child("pPr")
	outline, quote, numID, level, code := x.style(pPr.child("pStyle").attr("val"))
	if o := pPr.child("outlineLvl"); o != nil {
		if v, err := strconv.Atoi(o.attr("val")); err == nil && v < 9 {
			outline = v + 1
		}
	}
	if np := pPr.child("numPr"); np != nil {
		if id := np.child("numId").attr("val"); id != "" {
			numID = id
		}
		level, _ = strconv.Atoi(np.child("ilvl").attr("val"))
	}
	if numID == "0" || outline > 0 { // numbering taken off; a numbered heading is a heading
		numID = ""
	}
	if _, ok := x.lists[numID]; !ok {
		numID = ""
	}
	spans := x.spans(p, Span{})
	if code {
		return docxPara{code: true, codeText: spansText(spans)}
	}
	b := &Block{Kind: Para, Spans: trimSpans(spans)}
	if outline > 0 {
		b.Kind, b.Level = Heading, min(outline, 6)
		for i := range b.Spans { // a heading's own boldness is its style
			b.Spans[i].Bold = false
		}
	}
	left, _ := strconv.Atoi(cmpOr(pPr.child("ind").attr("left"), pPr.child("ind").attr("start")))
	return docxPara{block: b, numID: numID, level: level, quote: quote, indented: left > 0}
}

// spans reads the runs in n (a paragraph, a link, an insertion) in the
// format f.
func (x *docxReader) spans(n *node, f Span) []Span {
	var out []Span
	for _, c := range n.Children {
		switch c.Name.Local {
		case "r":
			out = append(out, x.run(c, f)...)
		case "hyperlink":
			g := f
			if t := x.links[c.attr("id")]; t != "" {
				g.Link = t
			} else if a := c.attr("anchor"); a != "" {
				g.Link = ""
			}
			out = append(out, x.spans(c, g)...)
		case "ins", "smartTag", "fldSimple", "customXml", "bdo", "dir":
			out = append(out, x.spans(c, f)...)
		case "sdt":
			out = append(out, x.spans(c.child("sdtContent"), f)...)
		}
	}
	return out
}

func (x *docxReader) run(r *node, f Span) []Span {
	s := f
	rPr := r.child("rPr")
	bold, italic, code := x.runStyle(rPr.child("rStyle").attr("val"))
	if b := onOff(rPr.child("b")); b != nil {
		bold = b
	}
	if i := onOff(rPr.child("i")); i != nil {
		italic = i
	}
	if bold != nil {
		s.Bold = *bold
	}
	if italic != nil {
		s.Italic = *italic
	}
	if st := onOff(rPr.child("strike")); st != nil {
		s.Strike = *st
	}
	if fn := rPr.child("rFonts"); fn != nil && monospace(fn.attr("ascii")) {
		code = true
	}
	s.Code = code
	var out []Span
	add := func(t string) {
		if n := len(out); n > 0 && !out[n-1].Break && out[n-1].Note == nil && out[n-1].Comment == "" {
			out[n-1].Text += t
			return
		}
		t2 := s
		t2.Text = t
		out = append(out, t2)
	}
	for _, c := range r.Children {
		switch c.Name.Local {
		case "t":
			add(c.text())
		case "tab", "ptab":
			add("\t")
		case "noBreakHyphen":
			add("-")
		case "br", "cr":
			if c.attr("type") != "page" && c.attr("type") != "column" {
				out = append(out, Span{Break: true})
			}
		case "footnoteReference", "endnoteReference":
			kind := strings.TrimSuffix(c.Name.Local, "Reference")
			if note := x.notes[kind+":"+c.attr("id")]; note != nil {
				blocks := x.body(note)
				if len(blocks) == 0 {
					blocks = []*Block{{Kind: Para}}
				}
				out = append(out, Span{Note: blocks})
			}
		case "drawing", "pict", "object", "AlternateContent":
			x.doc.Dropped++
			alt := ""
			if d := c.find("docPr"); d != nil {
				alt = cmpOr(d.attr("descr"), d.attr("title"), d.attr("name"))
			}
			out = append(out, Span{Comment: strings.TrimSpace("image left out: " + alt)})
		}
	}
	return out
}

func cmpOr(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// trimSpans leaves out spaces at a paragraph's ends, and the footnote
// mark that starts a footnote's text (the number is the note itself).
func trimSpans(spans []Span) []Span {
	for len(spans) > 0 && !spans[0].Break && spans[0].Note == nil && spans[0].Comment == "" && strings.TrimSpace(spans[0].Text) == "" {
		spans = spans[1:]
	}
	for len(spans) > 0 && (spans[len(spans)-1].Break || (spans[len(spans)-1].Note == nil && spans[len(spans)-1].Comment == "" && strings.TrimSpace(spans[len(spans)-1].Text) == "")) {
		spans = spans[:len(spans)-1]
	}
	if len(spans) > 0 && spans[0].Note == nil && spans[0].Comment == "" {
		spans[0].Text = strings.TrimLeft(spans[0].Text, " \t")
	}
	if n := len(spans); n > 0 && spans[n-1].Note == nil && spans[n-1].Comment == "" {
		spans[n-1].Text = strings.TrimRight(spans[n-1].Text, " \t")
	}
	return spans
}

func (x *docxReader) table(t *node) *Block {
	b := &Block{Kind: Table}
	for _, tr := range t.all("tr") {
		var row [][]Span
		for _, tc := range tr.all("tc") {
			var cell []Span
			for _, bl := range x.body(tc) {
				if len(cell) > 0 {
					cell = append(cell, Span{Text: " "})
				}
				cell = append(cell, flatSpans(bl)...)
			}
			row = append(row, cell)
		}
		b.Rows = append(b.Rows, row)
	}
	if len(b.Rows) > 0 { // the header's bold is the header's
		for i := range b.Rows[0] {
			for j := range b.Rows[0][i] {
				b.Rows[0][i][j].Bold = false
			}
		}
	}
	return b
}

// flatSpans are a block's spans, its children's too.
func flatSpans(b *Block) []Span {
	if b.Kind == Code {
		return []Span{{Text: b.Text, Code: true}}
	}
	out := append([]Span(nil), b.Spans...)
	for _, c := range b.Children {
		if len(out) > 0 {
			out = append(out, Span{Text: " "})
		}
		out = append(out, flatSpans(c)...)
	}
	return out
}

// assemble puts paragraphs together into blocks: list items into lists
// (nested by their level), quotes into quotes, code lines into code, and
// empty paragraphs left out.
func (x *docxReader) assemble(ps []docxPara) []*Block {
	var out []*Block
	type open struct {
		list  *Block
		numID string
		level int
	}
	var stack []open // the lists open, innermost last
	var quote *Block
	var code *Block
	count := map[string]int{} // the items so far of a list ("numId:level"): Word goes on counting
	for _, p := range ps {
		if p.code {
			if code == nil {
				code = &Block{Kind: Code}
				out = append(out, code)
			}
			code.Text += p.codeText + "\n"
			stack, quote = nil, nil
			continue
		}
		code = nil
		b := p.block
		if b.Kind == Para && len(b.Spans) == 0 {
			continue
		}
		if p.numID == "" && p.indented && !p.quote && len(stack) > 0 && b.Kind == Para {
			top := stack[len(stack)-1].list
			last := top.Children[len(top.Children)-1]
			last.Children = append(last.Children, b)
			continue
		}
		if p.numID == "" {
			stack = nil
			if p.quote && b.Kind == Para {
				if quote == nil {
					quote = &Block{Kind: Quote}
					out = append(out, quote)
				}
				quote.Children = append(quote.Children, b)
				continue
			}
			quote = nil
			out = append(out, b)
			continue
		}
		quote = nil
		for len(stack) > 0 && (stack[len(stack)-1].level > p.level || (stack[len(stack)-1].level == p.level && stack[len(stack)-1].numID != p.numID)) {
			stack = stack[:len(stack)-1]
		}
		key := p.numID + ":" + strconv.Itoa(p.level)
		for k := range count { // a higher item restarts the levels below it
			if id, lv, ok := strings.Cut(k, ":"); ok && id == p.numID && lv > strconv.Itoa(p.level) {
				delete(count, k)
			}
		}
		if len(stack) == 0 || stack[len(stack)-1].level < p.level {
			l := &Block{Kind: List, Ordered: x.lists[p.numID][p.level], Start: count[key] + 1}
			if len(stack) == 0 {
				out = append(out, l)
			} else {
				parent := stack[len(stack)-1].list
				last := parent.Children[len(parent.Children)-1]
				last.Children = append(last.Children, l)
			}
			stack = append(stack, open{l, p.numID, p.level})
		}
		top := stack[len(stack)-1].list
		top.Children = append(top.Children, &Block{Kind: Item, Children: []*Block{b}})
		count[key]++
	}
	return out
}

package prose

import (
	"archive/zip"
	"errors"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/LaPingvino/accolade/internal/pace"
)

// ReadODT reads an OpenDocument text (LibreOffice, Google Docs, Word's
// .odt): its paragraphs, headings, lists, quotes, tables, bold, italic,
// links, footnotes and pictures (their files in the Images' Data, to be
// saved: SaveImages). Other objects are left out, a comment
// in their place.
func ReadODT(r io.ReaderAt, size int64) (*Doc, error) {
	z, err := zip.NewReader(r, size)
	if err != nil {
		return nil, errors.New("not an OpenDocument text (.odt): " + err.Error())
	}
	content, err := zipPart(z, "content.xml")
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, errors.New("not an OpenDocument text (.odt): it has no content.xml")
	}
	x := &odtReader{styles: map[string]*odtStyle{}, lists: map[string]map[int]bool{}, z: z}
	if s, err := zipPart(z, "styles.xml"); err == nil && s != nil {
		x.readStyles(s)
	}
	x.readStyles(content)
	d := &Doc{}
	if m, err := zipPart(z, "meta.xml"); err == nil && m != nil {
		d.Title = strings.TrimSpace(m.find("title").text())
	}
	x.doc = d
	d.Blocks = x.blocks(content.find("text"), "")
	return d, nil
}

type odtStyle struct {
	parent, display string
	bold, italic    *bool
	strike, mono    bool
	outline         int
}

type odtReader struct {
	doc    *Doc
	styles map[string]*odtStyle    // "paragraph:P1", "text:T1"
	lists  map[string]map[int]bool // a list style's levels that are numbered
	z      *zip.Reader
}

func (x *odtReader) readStyles(root *node) {
	for _, s := range root.all("style") {
		if s.Name.Space != "urn:oasis:names:tc:opendocument:xmlns:style:1.0" {
			continue
		}
		st := &odtStyle{parent: s.attr("parent-style-name"), display: cmpOr(s.attr("display-name"), s.attr("name"))}
		if tp := s.child("text-properties"); tp != nil {
			if w := tp.attr("font-weight"); w != "" {
				on := w == "bold" || w == "600" || w == "700" || w == "800" || w == "900"
				st.bold = &on
			}
			if f := tp.attr("font-style"); f != "" {
				on := f == "italic" || f == "oblique"
				st.italic = &on
			}
			if l := tp.attr("text-line-through-style"); l != "" && l != "none" {
				st.strike = true
			}
			st.mono = monospace(tp.attr("font-name")) || monospace(tp.attr("font-family")) || tp.attr("font-pitch") == "fixed"
		}
		if o, err := strconv.Atoi(s.attr("default-outline-level")); err == nil {
			st.outline = o
		}
		x.styles[s.attr("family")+":"+s.attr("name")] = st
	}
	for _, l := range root.all("list-style") {
		levels := map[int]bool{}
		for _, lv := range l.Children {
			n, err := strconv.Atoi(lv.attr("level"))
			if err != nil {
				continue
			}
			levels[n] = lv.Name.Local == "list-level-style-number" && lv.attr("num-format") != ""
		}
		x.lists[l.attr("name")] = levels
	}
}

// paraStyle is what a paragraph style says, with its parents: a heading's
// level, a quote, code.
func (x *odtReader) paraStyle(name string) (outline int, quote, code bool) {
	for i := 0; name != "" && i < 20; i++ {
		st := x.styles["paragraph:"+name]
		if st == nil {
			break
		}
		if outline == 0 {
			outline = st.outline
		}
		d := strings.ToLower(st.display + " " + name)
		if strings.Contains(d, "quot") || strings.Contains(d, "block text") {
			quote = true
		}
		if strings.Contains(d, "preformatted") || strings.Contains(d, "source") || strings.Contains(d, "code") || st.mono {
			code = true
		}
		name = st.parent
	}
	return
}

// textStyle puts a text style's format (with its parents') on f.
func (x *odtReader) textStyle(name string, f Span) Span {
	var bold, italic *bool
	for i := 0; name != "" && i < 20; i++ {
		st := x.styles["text:"+name]
		if st == nil {
			break
		}
		if bold == nil {
			bold = st.bold
		}
		if italic == nil {
			italic = st.italic
		}
		d := strings.ToLower(st.display)
		if bold == nil && (d == "strong emphasis" || d == "strong") {
			t := true
			bold = &t
		}
		if italic == nil && d == "emphasis" {
			t := true
			italic = &t
		}
		f.Strike = f.Strike || st.strike
		f.Code = f.Code || st.mono || d == "source text"
		name = st.parent
	}
	if bold != nil {
		f.Bold = *bold
	}
	if italic != nil {
		f.Italic = *italic
	}
	return f
}

// odtPara is a paragraph read, before quotes and code are put together.
type odtPara struct {
	block  *Block
	quote  bool
	code   string
	isCode bool
}

// blocks reads the blocks in n; list is the style of the list they are
// in.
func (x *odtReader) blocks(n *node, list string) []*Block {
	var ps []odtPara
	var walk func(n *node)
	walk = func(n *node) {
		pace.Yield()
		if n == nil {
			return
		}
		for _, c := range n.Children {
			switch c.Name.Local {
			case "h":
				lv, _ := strconv.Atoi(c.attr("outline-level"))
				ps = append(ps, odtPara{block: &Block{Kind: Heading, Level: min(max(lv, 1), 6), Spans: x.headingSpans(c)}})
			case "p":
				outline, quote, code := x.paraStyle(c.attr("style-name"))
				spans := x.spans(c, x.textStyle(c.attr("style-name"), Span{}))
				switch {
				case code:
					ps = append(ps, odtPara{isCode: true, code: spansText(spans)})
				case outline > 0:
					ps = append(ps, odtPara{block: &Block{Kind: Heading, Level: min(outline, 6), Spans: x.headingSpansOf(spans)}})
				default:
					ps = append(ps, odtPara{block: &Block{Kind: Para, Spans: trimSpans(spans)}, quote: quote})
				}
			case "list":
				ps = append(ps, odtPara{block: x.list(c, list, 1)})
			case "table":
				ps = append(ps, odtPara{block: x.table(c)})
			case "section", "deletion-wrapper", "index-body", "illustration-index", "text-content":
				walk(c)
			}
		}
	}
	walk(n)
	return x.assemble(ps)
}

// list reads a text:list at a depth (1 the outermost); style is the
// surrounding list's style, used if it has none of its own.
func (x *odtReader) list(l *node, style string, depth int) *Block {
	if s := l.attr("style-name"); s != "" {
		style = s
	}
	b := &Block{Kind: List, Ordered: x.lists[style][depth], Start: 1}
	for _, it := range l.Children {
		if it.Name.Local != "list-item" && it.Name.Local != "list-header" {
			continue
		}
		if v, err := strconv.Atoi(it.attr("start-value")); err == nil && len(b.Children) == 0 {
			b.Start = v
		}
		item := &Block{Kind: Item}
		var inner []*node
		flush := func() {
			if len(inner) > 0 {
				item.Children = append(item.Children, x.blocks(&node{Children: inner}, style)...)
				inner = nil
			}
		}
		for _, c := range it.Children {
			if c.Name.Local == "list" {
				flush()
				item.Children = append(item.Children, x.list(c, style, depth+1))
				continue
			}
			inner = append(inner, c)
		}
		flush()
		b.Children = append(b.Children, item)
	}
	return b
}

func (x *odtReader) headingSpans(h *node) []Span {
	return x.headingSpansOf(x.spans(h, x.textStyle(h.attr("style-name"), Span{})))
}

// headingSpansOf: a heading's bold is its style's, not emphasis.
func (x *odtReader) headingSpansOf(spans []Span) []Span {
	spans = trimSpans(spans)
	for i := range spans {
		spans[i].Bold = false
	}
	return spans
}

// spans reads the text in n in the format f.
func (x *odtReader) spans(n *node, f Span) []Span {
	var out []Span
	add := func(t string) {
		if t == "" {
			return
		}
		if k := len(out); k > 0 && !out[k-1].Break && out[k-1].Note == nil && out[k-1].Comment == "" && out[k-1].sameFormat(f) {
			out[k-1].Text += t
			return
		}
		s := f
		s.Text = t
		out = append(out, s)
	}
	for _, c := range n.Children {
		switch c.Name.Local {
		case "":
			add(collapse(c.Text))
		case "span":
			out = append(out, x.spans(c, x.textStyle(c.attr("style-name"), f))...)
		case "a":
			g := f
			if h := c.attr("href"); h != "" && !strings.HasPrefix(h, "#") {
				g.Link = h
			}
			out = append(out, x.spans(c, g)...)
		case "s":
			n, err := strconv.Atoi(c.attr("c"))
			if err != nil || n < 1 {
				n = 1
			}
			add(strings.Repeat(" ", n))
		case "tab":
			add("\t")
		case "line-break":
			out = append(out, Span{Break: true})
		case "note":
			blocks := x.blocks(c.child("note-body"), "")
			if len(blocks) == 0 {
				blocks = []*Block{{Kind: Para}}
			}
			out = append(out, Span{Note: blocks})
		case "frame":
			title, desc := strings.TrimSpace(c.child("title").text()), strings.TrimSpace(c.child("desc").text())
			alt := cmpOr(desc, title, c.attr("name"))
			if im := c.child("image"); im != nil {
				href := im.attr("href")
				if data, err := zipFile(x.z, strings.TrimPrefix(href, "./")); err == nil && href != "" {
					if desc == "" {
						title = ""
					}
					s := f
					s.Text, s.Image = alt, &Image{Src: path.Base(href), Alt: alt, Title: title, Data: data}
					out = append(out, s)
					continue
				}
			}
			x.doc.Dropped++
			out = append(out, Span{Comment: strings.TrimSpace("image left out: " + alt)})
		case "annotation", "annotation-end", "bookmark", "bookmark-start", "bookmark-end", "reference-mark",
			"reference-mark-start", "reference-mark-end", "change", "change-start", "change-end", "soft-page-break",
			"toc-mark", "alphabetical-index-mark", "alphabetical-index-mark-start", "alphabetical-index-mark-end", "ruby-text":
		default: // fields, ruby bases, meta: their text
			out = append(out, x.spans(c, f)...)
		}
	}
	return out
}

func (x *odtReader) table(t *node) *Block {
	b := &Block{Kind: Table}
	for _, row := range t.all("table-row") {
		var cells [][]Span
		for _, c := range row.Children {
			if c.Name.Local != "table-cell" {
				continue
			}
			var cell []Span
			for _, bl := range x.blocks(c, "") {
				if len(cell) > 0 {
					cell = append(cell, Span{Text: " "})
				}
				cell = append(cell, flatSpans(bl)...)
			}
			cells = append(cells, cell)
		}
		b.Rows = append(b.Rows, cells)
	}
	if len(b.Rows) > 0 {
		for i := range b.Rows[0] {
			for j := range b.Rows[0][i] {
				b.Rows[0][i][j].Bold = false
			}
		}
	}
	return b
}

// assemble puts quote paragraphs together into quotes and code lines into
// code, and leaves out empty paragraphs.
func (x *odtReader) assemble(ps []odtPara) []*Block {
	var out []*Block
	var quote, code *Block
	for _, p := range ps {
		if p.isCode {
			if code == nil {
				code = &Block{Kind: Code}
				out = append(out, code)
			}
			code.Text += p.code + "\n"
			quote = nil
			continue
		}
		code = nil
		b := p.block
		if b.Kind == Para && len(b.Spans) == 0 {
			continue
		}
		if p.quote {
			if quote == nil {
				quote = &Block{Kind: Quote}
				out = append(out, quote)
			}
			quote.Children = append(quote.Children, b)
			continue
		}
		quote = nil
		out = append(out, b)
	}
	return out
}

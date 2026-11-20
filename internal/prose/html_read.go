package prose

import (
	"io"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"

	"github.com/LaPingvino/accolade/internal/pace"
)

// ReadHTML reads a web page: its headings, paragraphs, lists, quotes,
// code, tables, emphasis and links. Scripts, styles, navigation and forms
// are left out; images too, a comment in their place.
//
// golang.org/x/net/html (the Go project's own HTML5 parser, already in
// Accolade's modules through pion's WebRTC) parses the page as browsers
// do; what is made of it is this small walk, as for DOCX and ODT, rather
// than a converter library with dependencies of its own.
func ReadHTML(r io.Reader) (*Doc, error) {
	// in the page's own encoding (a <meta charset>, a byte order mark), not
	// only UTF-8
	r, err := charset.NewReader(r, "")
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	x := &htmlReader{doc: &Doc{}}
	if t := findAtom(root, atom.Title); t != nil {
		x.doc.Title = strings.TrimSpace(textOf(t))
	}
	body := findAtom(root, atom.Body)
	if body == nil {
		body = root
	}
	x.doc.Blocks = x.blocks(body)
	return x.doc, nil
}

type htmlReader struct {
	doc *Doc
}

func findAtom(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if f := findAtom(c, a); f != nil {
			return f
		}
	}
	return nil
}

func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// skipped are elements whose content is not the text.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Head: true,
	atom.Nav: true, atom.Form: true, atom.Button: true, atom.Select: true, atom.Input: true,
	atom.Textarea: true, atom.Svg: true, atom.Math: true, atom.Iframe: true, atom.Object: true,
	atom.Embed: true, atom.Canvas: true, atom.Audio: true, atom.Video: true,
}

// blockAtoms are elements that make blocks of their own.
var blockAtoms = map[atom.Atom]bool{
	atom.P: true, atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Ul: true, atom.Ol: true, atom.Blockquote: true, atom.Pre: true, atom.Hr: true, atom.Table: true,
	atom.Div: true, atom.Section: true, atom.Article: true, atom.Main: true, atom.Header: true, atom.Footer: true,
	atom.Aside: true, atom.Figure: true, atom.Figcaption: true, atom.Dl: true, atom.Dt: true, atom.Dd: true,
	atom.Li: true, atom.Address: true, atom.Details: true, atom.Summary: true, atom.Center: true,
}

// blocks reads the blocks in n: its block elements, and the text between
// them as paragraphs.
func (x *htmlReader) blocks(n *html.Node) []*Block {
	pace.Yield()
	var out []*Block
	var loose []Span // inline content not in a paragraph
	flush := func() {
		if s := trimSpans(loose); len(s) > 0 {
			out = append(out, &Block{Kind: Para, Spans: s})
		}
		loose = nil
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && skipped[c.DataAtom] {
			continue
		}
		if c.Type != html.ElementNode || !blockAtoms[c.DataAtom] {
			if c.Type == html.ElementNode && c.DataAtom == atom.Br && len(loose) == 0 {
				continue
			}
			loose = append(loose, x.spans(c, Span{})...)
			continue
		}
		flush()
		out = append(out, x.block(c)...)
	}
	flush()
	return out
}

func (x *htmlReader) block(n *html.Node) []*Block {
	switch n.DataAtom {
	case atom.P, atom.Dt, atom.Figcaption, atom.Summary, atom.Address:
		f := Span{}
		if n.DataAtom == atom.Dt {
			f.Bold = true
		}
		// a paragraph with blocks in it (as some pages have): those too
		return x.blocksWith(n, f)
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		lv, _ := strconv.Atoi(n.Data[1:])
		spans := trimSpans(x.spans(n, Span{}))
		if len(spans) == 0 {
			return nil
		}
		return []*Block{{Kind: Heading, Level: lv, Spans: spans}}
	case atom.Ul, atom.Ol:
		l := &Block{Kind: List, Ordered: n.DataAtom == atom.Ol, Start: 1}
		if s, err := strconv.Atoi(attrOf(n, "start")); err == nil {
			l.Start = s
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.DataAtom == atom.Li {
				l.Children = append(l.Children, &Block{Kind: Item, Children: x.blocks(c)})
			}
		}
		if len(l.Children) == 0 {
			return nil
		}
		return []*Block{l}
	case atom.Blockquote:
		if bs := x.blocks(n); len(bs) > 0 {
			return []*Block{{Kind: Quote, Children: bs}}
		}
		return nil
	case atom.Pre:
		return []*Block{{Kind: Code, Text: strings.TrimPrefix(textOf(n), "\n")}}
	case atom.Hr:
		return []*Block{{Kind: Rule}}
	case atom.Table:
		return x.table(n)
	case atom.Dd:
		if bs := x.blocks(n); len(bs) > 0 {
			return []*Block{{Kind: Quote, Children: bs}}
		}
		return nil
	}
	return x.blocks(n) // div, section, article ...: what is in them
}

// blocksWith reads a paragraph's content in the format f.
func (x *htmlReader) blocksWith(n *html.Node, f Span) []*Block {
	bs := x.blocks(n)
	if f.Bold {
		for _, b := range bs {
			for i := range b.Spans {
				b.Spans[i].Bold = true
			}
		}
	}
	return bs
}

func (x *htmlReader) spans(n *html.Node, f Span) []Span {
	switch n.Type {
	case html.TextNode:
		s := f
		s.Text = n.Data
		if f.Code {
			s.Text = strings.ReplaceAll(n.Data, "\n", " ")
		}
		return []Span{s}
	case html.ElementNode:
	default:
		return nil
	}
	if skipped[n.DataAtom] {
		return nil
	}
	g := f
	switch n.DataAtom {
	case atom.Br:
		return []Span{{Break: true}}
	case atom.Img:
		x.doc.Dropped++
		alt := strings.TrimSpace(attrOf(n, "alt"))
		return []Span{{Comment: strings.TrimSpace("image left out: " + alt)}}
	case atom.Strong, atom.B:
		g.Bold = true
	case atom.Em, atom.I, atom.Cite, atom.Var, atom.Dfn:
		g.Italic = true
	case atom.Del, atom.S, atom.Strike:
		g.Strike = true
	case atom.Code, atom.Kbd, atom.Samp, atom.Tt:
		g.Code = true
	case atom.A:
		if h := attrOf(n, "href"); h != "" && !strings.HasPrefix(h, "#") && !strings.HasPrefix(strings.ToLower(h), "javascript:") {
			g.Link = h
		}
	case atom.Sup:
		// a footnote's mark, as pages have them: left as it is
	}
	if st := strings.ToLower(attrOf(n, "style")); st != "" {
		if strings.Contains(st, "font-weight:bold") || strings.Contains(st, "font-weight: bold") || strings.Contains(st, "font-weight:700") || strings.Contains(st, "font-weight: 700") {
			g.Bold = true
		}
		if strings.Contains(st, "font-style:italic") || strings.Contains(st, "font-style: italic") {
			g.Italic = true
		}
	}
	var out []Span
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && blockAtoms[c.DataAtom] {
			// a block inside inline content: its text, apart
			out = append(out, Span{Text: " "})
			for _, b := range x.block(c) {
				out = append(out, flatSpans(b)...)
				out = append(out, Span{Text: " "})
			}
			continue
		}
		out = append(out, x.spans(c, g)...)
	}
	return out
}

func (x *htmlReader) table(t *html.Node) []*Block {
	b := &Block{Kind: Table}
	var rows func(n *html.Node)
	rows = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			switch c.DataAtom {
			case atom.Thead, atom.Tbody, atom.Tfoot:
				rows(c)
			case atom.Tr:
				var row [][]Span
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if cell.Type == html.ElementNode && (cell.DataAtom == atom.Td || cell.DataAtom == atom.Th) {
						var spans []Span
						for _, bl := range x.blocks(cell) {
							if len(spans) > 0 {
								spans = append(spans, Span{Text: " "})
							}
							spans = append(spans, flatSpans(bl)...)
						}
						row = append(row, spans)
					}
				}
				b.Rows = append(b.Rows, row)
			}
		}
	}
	rows(t)
	if len(b.Rows) == 0 {
		return nil
	}
	return []*Block{b}
}

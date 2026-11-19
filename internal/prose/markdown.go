package prose

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/yuin/goldmark/extension"
)

// Markdown is the goldmark Accolade reads Markdown with: GitHub's
// (tables, strikethrough, autolinks) and footnotes.
var Markdown = goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))

// FromMarkdown reads Markdown into a document. Its first heading of
// level 1 is the title.
func FromMarkdown(src string) *Doc {
	source := []byte(src)
	root := Markdown.Parser().Parse(text.NewReader(source))
	r := &mdReader{src: source, notes: map[int][]*Block{}}
	// the footnotes first: their links come before them
	for n := root.FirstChild(); n != nil; n = n.NextSibling() {
		if list, ok := n.(*east.FootnoteList); ok {
			for f := list.FirstChild(); f != nil; f = f.NextSibling() {
				if fn, ok := f.(*east.Footnote); ok {
					r.notes[fn.Index] = r.blocks(fn)
				}
			}
		}
	}
	d := &Doc{Blocks: r.blocks(root)}
	for _, b := range d.Blocks {
		if b.Kind == Heading && b.Level == 1 {
			d.Title = spansText(b.Spans)
			break
		}
	}
	return d
}

type mdReader struct {
	src   []byte
	notes map[int][]*Block
}

func (r *mdReader) blocks(parent ast.Node) []*Block {
	var out []*Block
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if b := r.block(n); b != nil {
			out = append(out, b)
		}
	}
	return out
}

func (r *mdReader) block(n ast.Node) *Block {
	switch n := n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return &Block{Kind: Para, Spans: r.spans(n, Span{})}
	case *ast.Heading:
		return &Block{Kind: Heading, Level: n.Level, Spans: r.spans(n, Span{})}
	case *ast.Blockquote:
		return &Block{Kind: Quote, Children: r.blocks(n)}
	case *ast.List:
		l := &Block{Kind: List, Ordered: n.IsOrdered(), Start: n.Start}
		for it := n.FirstChild(); it != nil; it = it.NextSibling() {
			l.Children = append(l.Children, &Block{Kind: Item, Children: r.blocks(it)})
		}
		return l
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		var b strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			s := lines.At(i)
			b.Write(s.Value(r.src))
		}
		return &Block{Kind: Code, Text: b.String()}
	case *ast.ThematicBreak:
		return &Block{Kind: Rule}
	case *east.Table:
		t := &Block{Kind: Table}
		for row := n.FirstChild(); row != nil; row = row.NextSibling() {
			var cells [][]Span
			for c := row.FirstChild(); c != nil; c = c.NextSibling() {
				cells = append(cells, r.spans(c, Span{}))
			}
			t.Rows = append(t.Rows, cells)
		}
		return t
	}
	return nil // HTML blocks (comments, the notes), the footnotes' list
}

// spans are an inline node's children as spans in the format f, and what
// they add to it.
func (r *mdReader) spans(parent ast.Node, f Span) []Span {
	var out []Span
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		switch n := n.(type) {
		case *ast.Text:
			s := f
			s.Text = string(n.Value(r.src))
			out = append(out, s)
			if n.HardLineBreak() {
				out = append(out, Span{Break: true})
			} else if n.SoftLineBreak() {
				s.Text = " "
				out = append(out, s)
			}
		case *ast.String:
			s := f
			s.Text = string(n.Value)
			out = append(out, s)
		case *ast.CodeSpan:
			s := f
			s.Code = true
			var b strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					b.Write(t.Value(r.src))
				}
			}
			s.Text = b.String()
			out = append(out, s)
		case *ast.Emphasis:
			g := f
			if n.Level >= 2 {
				g.Bold = true
			} else {
				g.Italic = true
			}
			out = append(out, r.spans(n, g)...)
		case *east.Strikethrough:
			g := f
			g.Strike = true
			out = append(out, r.spans(n, g)...)
		case *ast.Link:
			g := f
			g.Link = string(n.Destination)
			out = append(out, r.spans(n, g)...)
		case *ast.AutoLink:
			s := f
			s.Link = string(n.URL(r.src))
			s.Text = string(n.Label(r.src))
			out = append(out, s)
		case *ast.Image:
			s := f
			s.Text = spansText(r.spans(n, Span{}))
			out = append(out, s)
		case *east.FootnoteLink:
			out = append(out, Span{Note: r.notes[n.Index]})
		case *ast.RawHTML:
			// left out: comments, tags
		default:
			out = append(out, r.spans(n, f)...)
		}
	}
	return out
}

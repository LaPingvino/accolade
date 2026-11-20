// Package prose converts prose documents to and from Markdown, in pure
// Go so the browser version has it too (docs/ADAPTATION.md, step 2): Word
// (DOCX) and OpenDocument (ODT) both ways, HTML in. Each goes through one
// small model of a document, a tree of blocks with formatted spans:
// Markdown is read into it with goldmark, a document read into it from
// its XML; written out, it is Markdown, a DOCX, an ODT or plain text.
//
// Why here and not in Lexington: Lexington writes screenplays, laid out
// by its element rules; prose has none of that, and its source here is
// Markdown, which Lexington has no reader for. Readers and writers share
// the model, so they stay together.
package prose

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/LaPingvino/accolade/internal/pace"
)

// Kind is a block's kind.
type Kind int

const (
	Para    Kind = iota
	Heading      // Level 1–6
	Quote        // Children: the quoted blocks
	List         // Children: Items; Ordered, Start
	Item         // Children: the item's blocks
	Code         // Text: the lines, as they are
	Rule         // a break: * * *
	Table        // Rows: cells of spans, the first the header
)

// Block is a paragraph, a heading, a quote, a list or one of its items, a
// piece of code, a break or a table.
type Block struct {
	Kind     Kind
	Level    int
	Spans    []Span
	Children []*Block
	Ordered  bool
	Start    int // an ordered list's first number
	Text     string
	Rows     [][][]Span
}

// Span is text in one format, a line break, a footnote or a comment.
type Span struct {
	Text                 string
	Bold, Italic, Strike bool
	Code                 bool   // in a monospaced font
	Link                 string // where the text links to
	Break                bool   // a line break, without text
	Note                 []*Block
	// Comment is kept in Markdown as an HTML comment (an image left out)
	// and nowhere else.
	Comment string
}

func (s Span) sameFormat(o Span) bool {
	return s.Bold == o.Bold && s.Italic == o.Italic && s.Strike == o.Strike && s.Code == o.Code && s.Link == o.Link
}

// Doc is a document.
type Doc struct {
	Title  string
	Blocks []*Block
	// Dropped counts what could not be carried over (images, objects).
	Dropped int
}

// Markdown is the document as Markdown: headings with #, emphasis with *,
// footnotes as [^1] with their text at the end.
func (d *Doc) Markdown() string {
	m := &mdWriter{}
	m.blocks(d.Blocks, "", "")
	out := strings.TrimRight(m.b.String(), "\n")
	for i, n := range m.notes {
		nm := &mdWriter{notes: m.notes}
		nm.blocks(n, "    ", "")
		body := strings.TrimRight(nm.b.String(), "\n")
		body = strings.TrimPrefix(body, "    ")
		out += fmt.Sprintf("\n\n[^%d]: %s", i+1, body)
		m.notes = nm.notes // notes in notes
	}
	if out == "" {
		return ""
	}
	return out + "\n"
}

type mdWriter struct {
	b     strings.Builder
	notes [][]*Block
	tight bool // the last thing written was a tight list's item
}

// blocks writes blocks, each line after the first started with indent,
// the first with first (a list item's mark) if it is not "".
func (m *mdWriter) blocks(bs []*Block, indent, first string) {
	m.items(bs, indent, first, false)
}

// items writes blocks as blocks does; in a list item (item) a list after
// a paragraph follows it without a blank line, so the list stays tight.
func (m *mdWriter) items(bs []*Block, indent, first string, item bool) {
	for i, b := range bs {
		pace.Yield()
		lead := indent
		if i == 0 && first != "" {
			lead = first
		} else if item && b.Kind == List && bs[i-1].Kind == Para {
			// no blank line
		} else if m.b.Len() > 0 {
			m.b.WriteString(strings.TrimRight(indent, " ") + "\n")
		}
		m.block(b, indent, lead)
	}
}

func (m *mdWriter) block(b *Block, indent, lead string) {
	switch b.Kind {
	case Para:
		text := m.inline(b.Spans, indent)
		if strings.TrimSpace(text) == "" {
			text = ""
		}
		m.b.WriteString(lead + escapeStart(text) + "\n")
	case Heading:
		text := strings.ReplaceAll(m.inline(b.Spans, indent), "\\\n"+indent, " ")
		m.b.WriteString(lead + strings.Repeat("#", min(max(b.Level, 1), 6)) + " " + text + "\n")
	case Quote:
		m.blocks(b.Children, indent+"> ", lead+"> ")
	case List:
		n := b.Start
		if n == 0 {
			n = 1
		}
		loose := false // an item of more than one paragraph: a blank line between items
		for _, it := range b.Children {
			paras := 0
			for _, c := range it.Children {
				if c.Kind != List {
					paras++
				}
			}
			loose = loose || paras > 1
		}
		for i, it := range b.Children {
			mark := "- "
			if b.Ordered {
				mark = strconv.Itoa(n+i) + ". "
			}
			l := indent
			if i == 0 {
				l = lead
			} else if loose {
				m.b.WriteString(strings.TrimRight(indent, " ") + "\n")
			}
			m.items(it.Children, indent+strings.Repeat(" ", len(mark)), l+mark, true)
			if len(it.Children) == 0 {
				m.b.WriteString(l + strings.TrimRight(mark, " ") + "\n")
			}
		}
	case Code:
		fence := "```"
		for strings.Contains(b.Text, fence) {
			fence += "`"
		}
		m.b.WriteString(lead + fence + "\n")
		for _, l := range strings.Split(strings.TrimRight(b.Text, "\n"), "\n") {
			m.b.WriteString(strings.TrimRight(indent+l, " ") + "\n")
		}
		m.b.WriteString(indent + fence + "\n")
	case Rule:
		m.b.WriteString(lead + "* * *\n")
	case Table:
		for r, row := range b.Rows {
			l := indent
			if r == 0 {
				l = lead
			}
			var cells []string
			for _, c := range row {
				t := strings.ReplaceAll(m.inline(c, ""), "\\\n", " ")
				cells = append(cells, strings.ReplaceAll(t, "|", `\|`))
			}
			m.b.WriteString(l + "| " + strings.Join(cells, " | ") + " |\n")
			if r == 0 {
				m.b.WriteString(indent + "|" + strings.Repeat(" --- |", len(row)) + "\n")
			}
		}
	}
}

// inline writes spans as Markdown: runs of one format merged, emphasis
// opened and closed where it changes, spaces kept outside the marks.
func (m *mdWriter) inline(spans []Span, indent string) string {
	// merged: neighbours in one format are one span
	var ss []Span
	for _, s := range spans {
		if n := len(ss); n > 0 && s.Note == nil && s.Comment == "" && !s.Break && ss[n-1].Note == nil &&
			ss[n-1].Comment == "" && !ss[n-1].Break && ss[n-1].sameFormat(s) {
			ss[n-1].Text += s.Text
			continue
		}
		ss = append(ss, s)
	}
	var b strings.Builder
	for i := 0; i < len(ss); i++ {
		s := ss[i]
		switch {
		case s.Break:
			if b.Len() > 0 && i < len(ss)-1 {
				b.WriteString("\\\n" + indent)
			}
			continue
		case s.Note != nil:
			m.notes = append(m.notes, s.Note)
			fmt.Fprintf(&b, "[^%d]", len(m.notes))
			continue
		case s.Comment != "":
			b.WriteString("<!-- " + strings.ReplaceAll(s.Comment, "--", "–") + " -->")
			continue
		}
		text := s.Text
		if s.Code {
			text = codeSpan(text)
		} else {
			text = escapeInline(text)
		}
		core := strings.TrimSpace(text)
		if core == "" {
			b.WriteString(collapse(text))
			continue
		}
		pre, post := text[:strings.Index(text, core)], text[strings.Index(text, core)+len(core):]
		open, close := "", ""
		if s.Strike {
			open, close = "~~", "~~"
		}
		switch {
		case s.Bold && s.Italic:
			open, close = open+"***", "***"+close
		case s.Bold:
			open, close = open+"**", "**"+close
		case s.Italic:
			open, close = open+"*", "*"+close
		}
		core = open + core + close
		if s.Link != "" {
			core = "[" + core + "](" + linkTarget(s.Link) + ")"
		}
		b.WriteString(collapse(pre) + core + collapse(post))
	}
	return strings.TrimSpace(b.String())
}

// collapse makes each run of spaces, tabs and line breaks one space (a
// loop: the regexp it was took half of reading a novel's passages).
func collapse(s string) string {
	if !strings.ContainsAny(s, "\t\r\n") && !strings.Contains(s, "  ") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case ' ', '\t', '\r', '\n':
			if !space {
				b.WriteByte(' ')
			}
			space = true
		default:
			b.WriteByte(c)
			space = false
		}
	}
	return b.String()
}

func linkTarget(url string) string {
	if strings.ContainsAny(url, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(url) + ">"
	}
	return url
}

func codeSpan(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return fence + " " + s + " " + fence
	}
	return fence + s + fence
}

// escapeInline escapes what would be markup in Markdown's running text.
func escapeInline(s string) string {
	s = collapse(s)
	return inlineEscaper.Replace(s)
}

var inlineEscaper = strings.NewReplacer(`\`, `\\`, `*`, `\*`, `_`, `\_`, "`", "\\`", `[`, `\[`, `]`, `\]`, `<`, `\<`, `~~`, `\~\~`)

// blockStart is what would make a paragraph something else at its start:
// a heading's #, a quote's >, a list's mark, a break, a table's |.
var blockStart = regexp.MustCompile(`^(#|>|[-+*] |[-+*]$|\d{1,9}[.)]( |$)|(\* *){3,}$|(- *){3,}$|=+$|\|)`)

func escapeStart(s string) string {
	m := blockStart.FindString(s)
	switch {
	case m == "":
		return s
	case m[0] >= '0' && m[0] <= '9':
		n := strings.IndexAny(m, ".)")
		return s[:n] + `\` + s[n:]
	}
	return `\` + s
}

// Plain is the document as plain text: headings, paragraphs and list
// items with a blank line between, quotes indented, footnotes at the end.
func (d *Doc) Plain() string {
	var notes [][]*Block
	text := func(spans []Span) string {
		var t strings.Builder
		for _, s := range spans {
			switch {
			case s.Break:
				t.WriteString("\n")
			case s.Note != nil:
				notes = append(notes, s.Note)
				fmt.Fprintf(&t, "[%d]", len(notes))
			default:
				t.WriteString(collapse(s.Text))
				if s.Link != "" && s.Link != s.Text {
					t.WriteString(" (" + s.Link + ")")
				}
			}
		}
		return strings.TrimSpace(t.String())
	}
	var walk func(b *strings.Builder, bs []*Block, indent string)
	walk = func(b *strings.Builder, bs []*Block, indent string) {
		line := func(s string) {
			for _, l := range strings.Split(s, "\n") {
				b.WriteString(strings.TrimRight(indent+l, " ") + "\n")
			}
			b.WriteString("\n")
		}
		for _, bl := range bs {
			pace.Yield()
			switch bl.Kind {
			case Para, Heading:
				line(text(bl.Spans))
			case Quote:
				walk(b, bl.Children, indent+"    ")
			case List:
				for i, it := range bl.Children {
					mark := "- "
					if bl.Ordered {
						mark = strconv.Itoa(max(bl.Start, 1)+i) + ". "
					}
					var sub strings.Builder
					walk(&sub, it.Children, "")
					item := strings.TrimSpace(sub.String())
					line(mark + strings.ReplaceAll(item, "\n", "\n"+strings.Repeat(" ", len(mark))))
				}
			case Code:
				line(strings.TrimRight(bl.Text, "\n"))
			case Rule:
				line("* * *")
			case Table:
				for _, row := range bl.Rows {
					var cells []string
					for _, c := range row {
						cells = append(cells, text(c))
					}
					b.WriteString(indent + strings.Join(cells, "\t") + "\n")
				}
				b.WriteString("\n")
			}
		}
	}
	var b strings.Builder
	walk(&b, d.Blocks, "")
	for i := 0; i < len(notes); i++ {
		var sub strings.Builder
		walk(&sub, notes[i], "")
		fmt.Fprintf(&b, "[%d] %s\n", i+1, strings.TrimSpace(sub.String()))
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// spansText is spans' text, unformatted.
func spansText(spans []Span) string {
	var b strings.Builder
	for _, s := range spans {
		if s.Break {
			b.WriteString(" ")
		}
		b.WriteString(s.Text)
	}
	return strings.TrimSpace(collapse(b.String()))
}

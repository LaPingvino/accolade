// Package opml reads and writes outlines: OPML, the file outliners such
// as Workflowy and OmniOutliner export and open, and the indented text
// they copy to the clipboard. An outline is a tree of items, each with a
// line of text and a note; Fountain turns one into a script's structure
// (sections, synopses, notes), the way Accolade's outline reads them, so
// Accolade's own export, opened again, gives the same text.
//
// Accolade's export marks each item with what it is in the script
// (_kind, an attribute outliners keep and do not show). An outline from
// elsewhere has none, and is read by its shape: an item holding others is
// a section (# at the top, ## inside one, …), its note its synopsis; a
// leaf is a synopsis standing on its own (a beat), unless its text is a
// scene heading or a note.
package opml

import (
	"bytes"
	"encoding/xml"
	"errors"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// What an item is in the script (_kind).
const (
	KindSection   = "section"   // a # heading, its note its own synopsis
	KindScene     = "scene"     // a scene heading, its note its own synopsis
	KindSynopsis  = "synopsis"  // a synopsis paragraph of its own (a beat), its note its further lines
	KindNote      = "note"      // a [[note]] paragraph, its note its further lines
	KindTitlePage = "titlepage" // the title page, its note the title page's lines
)

// Node is an outline's item.
type Node struct {
	Text string
	Note string // its note (_note): a synopsis, a note's further lines
	Kind string // what it is (_kind); "" from another outliner: read by its shape
	// Level is a section's number of #s when it is not its depth (a ##
	// with no # above it); 0: its depth
	Level int
	Kids  []*Node
}

// Doc is an outline: its title and its items.
type Doc struct {
	Title string
	Nodes []*Node
}

type xmlOutline struct {
	Text  string       `xml:"text,attr"`
	Title string       `xml:"title,attr,omitempty"`
	Note  string       `xml:"_note,attr,omitempty"`
	Kind  string       `xml:"_kind,attr,omitempty"`
	Level string       `xml:"_level,attr,omitempty"`
	Kids  []xmlOutline `xml:"outline"`
}

type xmlDoc struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    struct {
		Title string `xml:"title,omitempty"`
	} `xml:"head"`
	Body struct {
		Outlines []xmlOutline `xml:"outline"`
	} `xml:"body"`
}

// Sniff tells whether data is an OPML file, by its content (a name
// says nothing for a file saved as .xml, .txt or with none): XML whose
// first element is <opml>.
func Sniff(data []byte) bool {
	head := data[:min(len(data), 2048)]
	head = bytes.TrimSpace(bytes.TrimPrefix(head, []byte("\xef\xbb\xbf")))
	return bytes.HasPrefix(head, []byte("<")) && bytes.Contains(bytes.ToLower(head), []byte("<opml"))
}

// Read reads an OPML file.
func Read(r io.Reader) (Doc, error) {
	var x xmlDoc
	dec := xml.NewDecoder(r)
	dec.Strict = false // outliners' files are not all well-formed (&nbsp;)
	dec.Entity = xml.HTMLEntity
	dec.AutoClose = xml.HTMLAutoClose
	if err := dec.Decode(&x); err != nil {
		return Doc{}, errors.New("not an outline (OPML): " + err.Error())
	}
	var conv func(xs []xmlOutline) []*Node
	conv = func(xs []xmlOutline) []*Node {
		var out []*Node
		for _, o := range xs {
			n := &Node{Text: o.Text, Note: o.Note, Kind: o.Kind, Kids: conv(o.Kids)}
			if n.Text == "" {
				n.Text = o.Title
			}
			n.Level, _ = strconv.Atoi(o.Level)
			out = append(out, n)
		}
		return out
	}
	return Doc{Title: strings.TrimSpace(x.Head.Title), Nodes: conv(x.Body.Outlines)}, nil
}

// Write writes d as OPML 2.0.
func Write(w io.Writer, d Doc) error {
	var conv func(ns []*Node) []xmlOutline
	conv = func(ns []*Node) []xmlOutline {
		var out []xmlOutline
		for _, n := range ns {
			o := xmlOutline{Text: n.Text, Note: n.Note, Kind: n.Kind, Kids: conv(n.Kids)}
			if n.Level > 0 {
				o.Level = strconv.Itoa(n.Level)
			}
			out = append(out, o)
		}
		return out
	}
	x := xmlDoc{Version: "2.0"}
	x.Head.Title = d.Title
	x.Body.Outlines = conv(d.Nodes)
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(x); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// bullet is a list item's mark in copied text: "- ", "* ", "• ", "+ ",
// "◦ ", "▪ ", "‣ ", "– ", or a box: "[ ] ", "[x] ".
var bullet = regexp.MustCompile(`^(?:[-*•+◦▪‣–]\s+(?:\[[ xX]\]\s+)?|\[[ xX]\]\s+)`)

// ParseIndented reads indented text as an outliner copies it (Workflowy's
// "- item" with two spaces a level, OmniOutliner's tabs, a plain list):
// each line an item, nested by its indent. Where the lines have bullets,
// a line without one, indented under an item, is that item's note
// (Workflowy writes it in quotes).
func ParseIndented(text string) []*Node {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	bullets := false
	for _, l := range lines {
		if bullet.MatchString(strings.TrimSpace(l)) {
			bullets = true
			break
		}
	}
	type open struct {
		indent int
		n      *Node
	}
	var root []*Node
	var stack []open
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		indent := indentOf(l)
		marked := bullet.MatchString(t)
		t = strings.TrimSpace(bullet.ReplaceAllString(t, ""))
		if bullets && !marked && len(stack) > 0 && indent > stack[len(stack)-1].indent {
			p := stack[len(stack)-1].n // the item's note
			t = strings.TrimSuffix(strings.TrimPrefix(t, `"`), `"`)
			if p.Note != "" {
				p.Note += "\n"
			}
			p.Note += t
			continue
		}
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		n := &Node{Text: t}
		if len(stack) == 0 {
			root = append(root, n)
		} else {
			p := stack[len(stack)-1].n
			p.Kids = append(p.Kids, n)
		}
		stack = append(stack, open{indent, n})
	}
	return root
}

// indentOf is how far line l is indented, a tab four spaces.
func indentOf(l string) int {
	n := 0
	for _, r := range l {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// Looks reports whether text reads as an outline pasted from an
// outliner: OPML, or more than one line with some of them indented under
// others or bulleted.
func Looks(text string) bool {
	if Sniff([]byte(text)) {
		return true
	}
	lines := 0
	shaped := false
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		lines++
		if indentOf(l) > 0 || bullet.MatchString(t) {
			shaped = true
		}
	}
	return lines > 1 && shaped
}

// Fountain is outline d as a script's structure: sections, scene
// headings, synopses and notes, a blank line between paragraphs, a
// heading's own synopsis touching it. isScene tells whether a line is a
// scene heading in the script's language.
func Fountain(d Doc, isScene func(string) bool) string {
	var paras []string
	var walk func(ns []*Node, depth int, inScene bool)
	walk = func(ns []*Node, depth int, inScene bool) {
		for _, n := range ns {
			kind, text, level := n.Kind, n.Text, n.Level
			if kind == "" {
				kind, text, level = guess(n, inScene, isScene)
			}
			switch kind {
			case KindTitlePage:
				if s := strings.TrimSpace(n.Note); s != "" {
					paras = append(paras, s)
				}
			case KindSection:
				if inScene { // no section inside a scene: its lines stand on their own
					paras = append(paras, synopsis(oneLine(text), n.Note))
					walk(n.Kids, depth, true)
					continue
				}
				if level <= 0 {
					level = depth + 1
				}
				paras = append(paras, own(strings.Repeat("#", level)+" "+oneLine(text), n.Note))
				walk(n.Kids, level, false)
			case KindScene:
				h := oneLine(text)
				if n.Kind == "" && !isScene(h) {
					h = "." + h
				}
				paras = append(paras, own(h, n.Note))
				walk(n.Kids, depth, true)
			case KindNote:
				p := strings.TrimSpace(text)
				if s := strings.TrimSpace(n.Note); s != "" {
					p += "\n" + s
				}
				paras = append(paras, p)
				walk(n.Kids, depth, inScene)
			default: // a synopsis of its own; its items in it the same
				if s := synopsis(text, n.Note); s != "" {
					paras = append(paras, s)
				}
				walk(n.Kids, depth, inScene)
			}
		}
	}
	walk(d.Nodes, 0, false)
	if len(paras) == 0 {
		return ""
	}
	return strings.Join(paras, "\n\n") + "\n"
}

// guess is what an item from another outliner is, by its shape: a
// heading written as one (# …), a note ([[…]]), a scene heading, a
// section when it holds items (not inside a scene), else a synopsis. Its
// text is cleaned of the outliner's markup.
func guess(n *Node, inScene bool, isScene func(string) bool) (kind, text string, level int) {
	text = clean(n.Text)
	t := strings.TrimSpace(text)
	switch {
	case strings.HasPrefix(t, "#"):
		l := len(t) - len(strings.TrimLeft(t, "#"))
		return KindSection, strings.TrimSpace(t[l:]), l
	case strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]"):
		return KindNote, t, 0
	case strings.HasPrefix(t, "="):
		return KindSynopsis, strings.TrimSpace(strings.TrimLeft(t, "=")), 0
	case isScene(t):
		return KindScene, t, 0
	case len(n.Kids) > 0 && !inScene:
		return KindSection, t, 0
	}
	return KindSynopsis, t, 0
}

// markup is an outliner's inline HTML (Workflowy's <b>, <i>, <u>,
// <span class="colored">, links).
var markup = regexp.MustCompile(`(?i)</?(b|strong|i|em|u|span|a|font|mark|code|s|strike)\b[^>]*>`)

// clean is an item's text from another outliner as Fountain: bold,
// italics and underline as Fountain marks them, other markup left out,
// entities read, on one line.
func clean(s string) string {
	s = markup.ReplaceAllStringFunc(s, func(tag string) string {
		name := strings.ToLower(strings.Trim(strings.Fields(strings.Trim(tag, "<>/"))[0], "/"))
		switch name {
		case "b", "strong":
			return "**"
		case "i", "em":
			return "*"
		case "u":
			return "_"
		}
		return ""
	})
	return oneLine(html.UnescapeString(s))
}

// oneLine is s on one line: its lines joined by a space.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// own is a heading with its note as its own lines, touching it: each
// line a synopsis, a note ([[…]]) as it is.
func own(head, note string) string {
	lines := []string{head}
	for _, l := range strings.Split(note, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, synopsisLine(t))
		}
	}
	return strings.Join(lines, "\n")
}

// synopsis is a synopsis paragraph: text and its note's lines, each "= …"
// ("" if there is nothing in it).
func synopsis(text, note string) string {
	var lines []string
	for _, l := range strings.Split(text+"\n"+note, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			lines = append(lines, synopsisLine(t))
		}
	}
	return strings.Join(lines, "\n")
}

// synopsisLine is t as a synopsis line: "= t", a note ([[…]]) as it is.
func synopsisLine(t string) string {
	if strings.HasPrefix(t, "[[") && strings.HasSuffix(t, "]]") {
		return t
	}
	return "= " + strings.TrimSpace(strings.TrimLeft(t, "="))
}

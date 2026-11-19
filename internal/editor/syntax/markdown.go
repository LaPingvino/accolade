package syntax

import (
	"strings"
	"unicode"
)

// Markdown's kinds of line, after Fountain's. Plain text is Action and
// a blank line Empty, as in Fountain.
const (
	MdHeading     Kind = iota + Lyrics + 1 // # Heading, or a line underlined with === or ---
	MdQuote                                // > quoted
	MdListItem                             // - item, * item, 1. item
	MdCode                                 // in a fenced or indented code block
	MdCodeFence                            // ``` or ~~~
	MdRule                                 // ---, ***, ___
	MdFrontMatter                          // the --- block of fields a file can start with
	MdComment                              // <!-- ... -->
)

// ClassifyMarkdown returns the kind of each line of a Markdown text.
func ClassifyMarkdown(lines []string) []Kind {
	kinds := make([]Kind, len(lines))
	i := 0
	if len(lines) > 0 && strings.TrimRight(lines[0], " \t") == "---" { // front matter
		for j := 1; j < len(lines); j++ {
			if t := strings.TrimRight(lines[j], " \t"); t == "---" || t == "..." {
				for ; i <= j; i++ {
					kinds[i] = MdFrontMatter
				}
				break
			}
		}
	}
	fence := ""       // the open code fence's marker
	comment := false  // inside <!-- -->
	inList := false   // after a list item, until a paragraph that is not indented
	lastBlank := true // the line before was blank (or there was none)
	for ; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		indent := indentOf(l)
		switch {
		case fence != "":
			kinds[i] = MdCode
			if indent < 4 && strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				kinds[i], fence = MdCodeFence, ""
			}
		case comment:
			kinds[i] = MdComment
			comment = !strings.Contains(t, "-->")
		case t == "":
			kinds[i] = Empty
		case indent >= 4 && lastBlank && !inList:
			kinds[i] = MdCode
		case indent < 4 && openFence(t) != "":
			kinds[i], fence = MdCodeFence, openFence(t)
		case strings.HasPrefix(t, "<!--"):
			kinds[i] = MdComment
			comment = !strings.Contains(t[4:], "-->")
		case indent < 4 && atxHeading(t):
			kinds[i] = MdHeading
		case indent < 4 && !lastBlank && kinds[i-1] == Action && underline(t):
			kinds[i-1], kinds[i] = MdHeading, MdHeading
		case indent < 4 && rule(t):
			kinds[i] = MdRule
		case strings.HasPrefix(t, ">"):
			kinds[i] = MdQuote
		case listItem(t):
			kinds[i] = MdListItem
			inList = true
		default:
			kinds[i] = Action
			if indent == 0 && lastBlank {
				inList = false
			}
		}
		lastBlank = kinds[i] == Empty
	}
	return kinds
}

// indentOf counts a line's leading spaces, a tab as four.
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

// openFence is the marker of a line opening a code block (``` or ~~~,
// three or more), or "".
func openFence(t string) string {
	for _, c := range []string{"`", "~"} {
		n := len(t) - len(strings.TrimLeft(t, c))
		if n >= 3 && (c == "~" || !strings.Contains(t[n:], "`")) {
			return strings.Repeat(c, n)
		}
	}
	return ""
}

// atxHeading: one to six # and a space (or nothing after them).
func atxHeading(t string) bool {
	n := len(t) - len(strings.TrimLeft(t, "#"))
	return n >= 1 && n <= 6 && (len(t) == n || t[n] == ' ' || t[n] == '\t')
}

// underline: a line of = or - under a paragraph's line, which makes it a
// heading.
func underline(t string) bool {
	return t != "" && (strings.Trim(t, "=") == "" || strings.Trim(t, "-") == "")
}

// rule: three or more of the same *, - or _, spaces between allowed.
func rule(t string) bool {
	s := strings.ReplaceAll(strings.ReplaceAll(t, " ", ""), "\t", "")
	return len(s) >= 3 && strings.Trim(s, s[:1]) == "" && strings.Contains("*-_", s[:1])
}

// listItem: - + * and a space, or a number with . or ) and a space.
func listItem(t string) bool {
	if len(t) >= 2 && strings.Contains("-+*", t[:1]) && (t[1] == ' ' || t[1] == '\t') {
		return true
	}
	n := len(t) - len(strings.TrimLeft(t, "0123456789"))
	return n >= 1 && n <= 9 && len(t) > n+1 && (t[n] == '.' || t[n] == ')') && t[n+1] == ' '
}

// Heading is a heading of a Markdown text.
type Heading struct {
	Line  int // where it is (the text's line; a setext heading's first)
	Level int // 1 for #, 2 for ##, ...
	Text  string
}

// MarkdownHeadings are the headings of a Markdown text, in order.
func MarkdownHeadings(lines []string) []Heading {
	kinds := ClassifyMarkdown(lines)
	var out []Heading
	for i := 0; i < len(lines); i++ {
		if kinds[i] != MdHeading {
			continue
		}
		t := strings.TrimSpace(lines[i])
		if atxHeading(t) {
			level := len(t) - len(strings.TrimLeft(t, "#"))
			text := strings.TrimSpace(t[level:])
			// closing #s, when a space comes before them
			if s := strings.TrimRight(text, "#"); s == "" || strings.HasSuffix(s, " ") {
				text = strings.TrimSpace(s)
			}
			out = append(out, Heading{Line: i, Level: level, Text: text})
			continue
		}
		level := 1 // setext: the line, then its underline
		if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "-") {
			level = 2
		}
		out = append(out, Heading{Line: i, Level: level, Text: t})
		i++
	}
	return out
}

// InlineKind is how a stretch of a Markdown line is emphasised.
type InlineKind int

const (
	Strong   InlineKind = iota + 1 // **bold**
	Emphasis                       // *italic*
	CodeSpan                       // `code`
	Link                           // [text](url), <https://...>
)

// Span is an emphasised stretch of a line, in runes [Start, End), its
// markers included.
type Span struct {
	Start, End int
	Kind       InlineKind
}

// MarkdownInline finds the emphasis, code and links in a line of text.
// Emphasis is not looked for inside other emphasis.
func MarkdownInline(line string) []Span {
	r := []rune(line)
	var spans []Span
	// closing finds the marker m from i on; -1 if it is not there
	closing := func(i int, m string) int {
		mr := []rune(m)
		for j := i; j+len(mr) <= len(r); j++ {
			if r[j] == '\\' {
				j++
				continue
			}
			if string(r[j:j+len(mr)]) == m && (len(mr) > 1 || j+1 >= len(r) || r[j+1] != mr[0]) {
				return j
			}
		}
		return -1
	}
	wordAt := func(i int) bool { return i >= 0 && i < len(r) && (unicode.IsLetter(r[i]) || unicode.IsDigit(r[i])) }
	for i := 0; i < len(r); i++ {
		switch c := r[i]; {
		case c == '\\':
			i++
		case c == '`':
			n := 1
			for i+n < len(r) && r[i+n] == '`' {
				n++
			}
			if j := closing(i+n, strings.Repeat("`", n)); j >= 0 {
				spans = append(spans, Span{i, j + n, CodeSpan})
				i = j + n - 1
			} else {
				i += n - 1
			}
		case c == '[':
			end := closing(i+1, "]")
			if end >= 0 && end+1 < len(r) && r[end+1] == '(' {
				if j := closing(end+2, ")"); j >= 0 {
					spans = append(spans, Span{i, j + 1, Link})
					i = j
				}
			}
		case c == '<' && strings.HasPrefix(string(r[i+1:]), "http"):
			if j := closing(i+1, ">"); j >= 0 && !strings.ContainsRune(string(r[i:j]), ' ') {
				spans = append(spans, Span{i, j + 1, Link})
				i = j
			}
		case c == '*' || c == '_':
			m, kind := string(c), Emphasis
			if i+1 < len(r) && r[i+1] == c {
				m, kind = string(c)+string(c), Strong
			}
			after := i + len(m)
			if after >= len(r) || unicode.IsSpace(r[after]) || c == '_' && wordAt(i-1) {
				i = after - 1
				continue
			}
			j := closing(after, m)
			if j < 0 || j == after || unicode.IsSpace(r[j-1]) || c == '_' && wordAt(j+len(m)) {
				i = after - 1
				continue
			}
			spans = append(spans, Span{i, j + len(m), kind})
			i = j + len(m) - 1
		}
	}
	return spans
}

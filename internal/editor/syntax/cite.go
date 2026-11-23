package syntax

import (
	"strings"
	"unicode"
)

// Pandoc's citations (pandoc.org/MANUAL.html#citation-syntax), as
// Zettlr colours them: [@doe99], [@doe99, p. 12], [see @a; @b, ch. 2],
// [-@doe99] (the author left out), and @doe99 in the text (with an
// optional [p. 12] after it). An @ after a letter or digit (me@x.org)
// or after a backslash is no citation; a bracket followed by ( or [ is
// a link, not a citation.

// CiteKey is one key of a citation, in runes [Start, End) of the line:
// the @ included.
type CiteKey struct {
	Key        string
	Start, End int
	Suppress   bool // -@key: the author left out
}

// Citation is a citation in a line: in runes [Start, End), brackets
// included; InText for @key outside brackets.
type Citation struct {
	Start, End int
	Keys       []CiteKey
	InText     bool
}

// Citations are the citations in a line of Markdown, wherever they are
// (in emphasis too, which the editor's spans do not look into), not in
// code spans, links, autolinks or a note's mark.
func Citations(line string) []Citation {
	if !strings.ContainsRune(line, '@') {
		return nil
	}
	var out []Citation
	r := []rune(line)
	for i := 0; i < len(r); i++ {
		switch r[i] {
		case '\\':
			i++
		case '`':
			n := 1
			for i+n < len(r) && r[i+n] == '`' {
				n++
			}
			if j := codeSpanEnd(r, i+n, n); j >= 0 {
				i = j - 1
			} else {
				i += n - 1
			}
		case '<':
			if strings.HasPrefix(string(r[i+1:min(i+9, len(r))]), "http") {
				for j := i + 1; j < len(r) && r[j] != ' '; j++ {
					if r[j] == '>' {
						i = j
						break
					}
				}
			}
		case '[':
			k := closeBracket(r, i)
			if k > 0 && (i+1 < len(r) && r[i+1] == '^' || i > 0 && r[i-1] == ']') { // a note's mark, a reference link's label
				i = k
				continue
			}
			if k > 0 && k+1 < len(r) && r[k+1] == '(' { // a link: its text and address
				if e := closeParen(r, k+1); e > 0 {
					i = e
					continue
				}
			}
			if c, ok := citationAt(r, i, true); ok {
				out = append(out, c)
				i = c.End - 1
			}
		case '@':
			if c, ok := citationAt(r, i, true); ok {
				out = append(out, c)
				i = c.End - 1
			}
		}
	}
	return out
}

// codeSpanEnd is where a code span opened by n backticks ends (after its
// closing ones) from i on, -1 if it does not.
func codeSpanEnd(r []rune, i, n int) int {
	for j := i; j < len(r); j++ {
		if r[j] != '`' {
			continue
		}
		m := 1
		for j+m < len(r) && r[j+m] == '`' {
			m++
		}
		if m == n {
			return j + m
		}
		j += m - 1
	}
	return -1
}

// closeParen is where the parenthesis opened at r[i] closes, -1 if not.
func closeParen(r []rune, i int) int {
	depth := 0
	for j := i; j < len(r); j++ {
		switch r[j] {
		case '\\':
			j++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// keyRune is a rune a key may hold: letters, digits, _ and Pandoc's
// internal punctuation (which may not end it).
func keyRune(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || strings.ContainsRune(":.#$%&-+?<>~/", c)
}

func keyPunct(c rune) bool { return strings.ContainsRune(":.#$%&-+?<>~/", c) }

// keyEnd is where a key at r[i] (just after the @) ends: letters and
// the rest, punctuation only between them, or {anything} in braces; i if
// there is none.
func keyEnd(r []rune, i int) int {
	if i < len(r) && r[i] == '{' {
		for j := i + 1; j < len(r); j++ {
			if r[j] == '}' {
				if j == i+1 {
					return i
				}
				return j + 1
			}
		}
		return i
	}
	if i >= len(r) || !(unicode.IsLetter(r[i]) || unicode.IsDigit(r[i]) || r[i] == '_') {
		return i
	}
	j := i
	for j < len(r) && keyRune(r[j]) {
		j++
	}
	for j > i && keyPunct(r[j-1]) {
		j--
	}
	return j
}

// keyAt is the key at r[i] and where it ends ("" and i for none).
func keyAt(r []rune, i int) (string, int) {
	j := keyEnd(r, i)
	switch {
	case j == i:
		return "", i
	case r[i] == '{':
		return string(r[i+1 : j-1]), j
	}
	return string(r[i:j]), j
}

// citeBefore: an @ at r[i] may start a key (nothing word-like before it).
func citeBefore(r []rune, i int) bool {
	if i == 0 {
		return true
	}
	p := r[i-1]
	return !(unicode.IsLetter(p) || unicode.IsDigit(p) || p == '\\' || p == '@' || p == '_')
}

// citationAt reads the citation starting at r[i]: a bracket holding
// citations, or an @key (with its [locator]). Without keys, only where
// it ends is found (the editor's colouring, per key: nothing allocated).
func citationAt(r []rune, i int, keys bool) (Citation, bool) {
	if i >= len(r) {
		return Citation{}, false
	}
	if r[i] == '@' {
		if !citeBefore(r, i) {
			return Citation{}, false
		}
		end := keyEnd(r, i+1)
		if end == i+1 {
			return Citation{}, false
		}
		c := Citation{Start: i, End: end, InText: true}
		if keys {
			key, _ := keyAt(r, i+1)
			c.Keys = []CiteKey{{Key: key, Start: i, End: end}}
		}
		// @doe99 [p. 33]: the locator in brackets after it
		j := end
		if j < len(r) && r[j] == ' ' {
			j++
		}
		if j < len(r) && r[j] == '[' {
			if k := closeBracket(r, j); k > 0 && !hasAt(r[j:k]) && (k+1 >= len(r) || r[k+1] != '(' && r[k+1] != '[') && r[j+1] != '^' {
				c.End = k + 1
			}
		}
		return c, true
	}
	if r[i] != '[' || i+1 < len(r) && r[i+1] == '^' || i > 0 && r[i-1] == ']' { // a note's mark, a reference link's label
		return Citation{}, false
	}
	k := closeBracket(r, i)
	if k < 0 || k+1 < len(r) && (r[k+1] == '(' || r[k+1] == '[') {
		return Citation{}, false
	}
	c := Citation{Start: i, End: k + 1}
	// every part between ; holds a key
	start := i + 1
	for p := i + 1; p <= k; p++ {
		if p < k && r[p] != ';' {
			continue
		}
		found := false
		for q := start; q < p; q++ {
			if r[q] != '@' {
				continue
			}
			if !citeBefore(r, q) {
				continue
			}
			sup := q > start && r[q-1] == '-' && (q-1 == start || r[q-2] == ' ')
			end := keyEnd(r, q+1)
			if end == q+1 {
				continue
			}
			if keys {
				key, _ := keyAt(r, q+1)
				c.Keys = append(c.Keys, CiteKey{Key: key, Start: q, End: end, Suppress: sup})
			}
			found = true
			break
		}
		if !found {
			return Citation{}, false
		}
		start = p + 1
	}
	return c, true
}

func hasAt(r []rune) bool {
	for _, c := range r {
		if c == '@' {
			return true
		}
	}
	return false
}

// closeBracket is where the bracket opened at r[i] closes (nested ones
// counted, \] skipped), -1 if it does not on the line.
func closeBracket(r []rune, i int) int {
	depth := 0
	for j := i; j < len(r); j++ {
		switch r[j] {
		case '\\':
			j++
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// CiteTyped is the key being typed at the cursor (a rune offset in
// line): the @ starts it, the cursor is in it or at its end. start is the
// @'s offset; ok false when the cursor is not on a key's place.
func CiteTyped(line string, cursor int) (start int, typed string, ok bool) {
	r := []rune(line)
	if cursor > len(r) {
		return 0, "", false
	}
	i := cursor
	for i > 0 && keyRune(r[i-1]) {
		i--
	}
	if i == 0 || r[i-1] != '@' || !citeBefore(r, i-1) {
		return 0, "", false
	}
	if inCode(r, i-1) {
		return 0, "", false
	}
	return i - 1, string(r[i:cursor]), true
}

// inCode: offset i of r is in a code span.
func inCode(r []rune, i int) bool {
	for _, s := range MarkdownInline(string(r)) {
		if s.Kind == CodeSpan && i >= s.Start && i < s.End {
			return true
		}
	}
	return false
}

// CiteKeyEnd is where the key at the @ at start ends in line (runes):
// what completing it replaces.
func CiteKeyEnd(line string, start int) int {
	r := []rune(line)
	j := start + 1
	for j < len(r) && keyRune(r[j]) {
		j++
	}
	return j
}

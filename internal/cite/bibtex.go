package cite

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// ParseBibTeX reads a BibTeX or BibLaTeX library: @string macros and #
// joining, @comment and @preamble skipped, braced and quoted values,
// month macros, LaTeX accents and the common commands decoded, names
// split on "and", crossref's fields inherited. An entry it cannot read
// is skipped and named in the error, the others kept: err is non-nil
// only to say what was skipped (entries may still be returned).
func ParseBibTeX(data []byte) ([]*Entry, error) {
	p := &bibParser{s: []rune(strings.TrimPrefix(string(data), "\ufeff")), macros: map[string]string{}}
	for k, v := range months {
		p.macros[k] = v
	}
	var raws []*bibRaw
	var skipped []string
	for {
		at := p.next('@')
		if at < 0 {
			break
		}
		p.i = at + 1
		typ := strings.ToLower(p.ident())
		p.space()
		if p.i >= len(p.s) || (p.s[p.i] != '{' && p.s[p.i] != '(') {
			continue // an @ in a comment between entries
		}
		open := p.s[p.i]
		close := '}'
		if open == '(' {
			close = ')'
		}
		switch typ {
		case "comment":
			p.skipBalanced()
			continue
		case "preamble":
			p.skipBalanced()
			continue
		case "string":
			p.i++
			if err := p.fields(close, func(name, value string) { p.macros[strings.ToLower(name)] = value }); err != nil {
				skipped = append(skipped, "@string: "+err.Error())
			}
			continue
		}
		start := p.i
		p.i++
		p.space()
		key := p.key()
		r := &bibRaw{typ: typ, key: key, fields: map[string]string{}}
		p.space()
		if key == "" || p.i >= len(p.s) || (p.s[p.i] != ',' && p.s[p.i] != close) {
			skipped = append(skipped, fmt.Sprintf("an @%s at line %d (no key)", typ, p.line(start)))
			p.i = start + 1
			continue
		}
		if p.s[p.i] == ',' {
			p.i++
		}
		if err := p.fields(close, func(name, value string) {
			if _, dup := r.fields[name]; !dup {
				r.fields[strings.ToLower(name)] = value
			}
		}); err != nil {
			skipped = append(skipped, fmt.Sprintf("@%s{%s at line %d: %v", typ, key, p.line(start), err))
			p.i = start + 1
			continue
		}
		raws = append(raws, r)
	}
	byKey := map[string]*bibRaw{}
	for _, r := range raws {
		if _, ok := byKey[strings.ToLower(r.key)]; !ok {
			byKey[strings.ToLower(r.key)] = r
		}
	}
	out := make([]*Entry, 0, len(raws))
	for _, r := range raws {
		inherit(r, byKey)
		out = append(out, r.entry())
	}
	if len(skipped) > 0 {
		return out, fmt.Errorf("skipped %s", strings.Join(skipped, "; "))
	}
	return out, nil
}

// months are BibTeX's month macros.
var months = map[string]string{
	"jan": "January", "feb": "February", "mar": "March", "apr": "April", "may": "May", "jun": "June",
	"jul": "July", "aug": "August", "sep": "September", "oct": "October", "nov": "November", "dec": "December",
}

// bibRaw is an entry as written: its fields' values with macros put in,
// LaTeX not decoded yet.
type bibRaw struct {
	typ, key string
	fields   map[string]string
	done     bool // crossref resolved
}

type bibParser struct {
	s      []rune
	i      int
	macros map[string]string
}

func (p *bibParser) line(at int) int {
	n := 1
	for _, r := range p.s[:min(at, len(p.s))] {
		if r == '\n' {
			n++
		}
	}
	return n
}

// next is where the next c is from p.i on, -1 if there is none.
func (p *bibParser) next(c rune) int {
	for j := p.i; j < len(p.s); j++ {
		if p.s[j] == c {
			return j
		}
	}
	return -1
}

func (p *bibParser) space() {
	for p.i < len(p.s) && unicode.IsSpace(p.s[p.i]) {
		p.i++
	}
}

// ident reads a type, a field name or a macro.
func (p *bibParser) ident() string {
	start := p.i
	for p.i < len(p.s) {
		r := p.s[p.i]
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("_-:.+/", r) {
			p.i++
			continue
		}
		break
	}
	return string(p.s[start:p.i])
}

// key reads an entry's key: up to a comma, a space or the closing brace.
func (p *bibParser) key() string {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] != ',' && p.s[p.i] != '}' && p.s[p.i] != ')' && !unicode.IsSpace(p.s[p.i]) {
		p.i++
	}
	return string(p.s[start:p.i])
}

// skipBalanced goes past the braced (or parenthesised) part at p.i.
func (p *bibParser) skipBalanced() {
	open := p.s[p.i]
	close := '}'
	if open == '(' {
		close = ')'
	}
	depth := 0
	for ; p.i < len(p.s); p.i++ {
		switch p.s[p.i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				p.i++
				return
			}
		}
	}
}

// fields reads name = value pairs up to close, each given to set.
func (p *bibParser) fields(close rune, set func(name, value string)) error {
	for {
		p.space()
		if p.i >= len(p.s) {
			return fmt.Errorf("the entry does not end")
		}
		if p.s[p.i] == close {
			p.i++
			return nil
		}
		if p.s[p.i] == ',' {
			p.i++
			continue
		}
		name := p.ident()
		if name == "" {
			return fmt.Errorf("a field without a name at %q", string(p.s[p.i:min(p.i+12, len(p.s))]))
		}
		p.space()
		if p.i >= len(p.s) || p.s[p.i] != '=' {
			return fmt.Errorf("%s has no =", name)
		}
		p.i++
		v, err := p.value()
		if err != nil {
			return fmt.Errorf("%s: %v", name, err)
		}
		set(strings.ToLower(name), v)
	}
}

// value reads a value: parts joined with #, each braced, quoted, a
// number or a macro.
func (p *bibParser) value() (string, error) {
	var b strings.Builder
	for {
		p.space()
		if p.i >= len(p.s) {
			return "", fmt.Errorf("no value")
		}
		switch c := p.s[p.i]; {
		case c == '{':
			start := p.i + 1
			depth := 0
			for ; p.i < len(p.s); p.i++ {
				if p.s[p.i] == '\\' {
					p.i++
					continue
				}
				if p.s[p.i] == '{' {
					depth++
				} else if p.s[p.i] == '}' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			if p.i >= len(p.s) {
				return "", fmt.Errorf("a brace is not closed")
			}
			b.WriteString(string(p.s[start:p.i]))
			p.i++
		case c == '"':
			p.i++
			start := p.i
			depth := 0
			for ; p.i < len(p.s); p.i++ {
				r := p.s[p.i]
				if r == '\\' {
					p.i++
					continue
				}
				if r == '{' {
					depth++
				} else if r == '}' {
					depth--
				} else if r == '"' && depth == 0 {
					break
				}
			}
			if p.i >= len(p.s) {
				return "", fmt.Errorf("a quote is not closed")
			}
			b.WriteString(string(p.s[start:p.i]))
			p.i++
		case unicode.IsDigit(c):
			start := p.i
			for p.i < len(p.s) && unicode.IsDigit(p.s[p.i]) {
				p.i++
			}
			b.WriteString(string(p.s[start:p.i]))
		default:
			name := p.ident()
			if name == "" {
				return "", fmt.Errorf("unexpected %q", c)
			}
			if v, ok := p.macros[strings.ToLower(name)]; ok {
				b.WriteString(v)
			} else {
				b.WriteString(name) // an undefined macro: BibTeX warns and leaves it empty; its name says more
			}
		}
		p.space()
		if p.i < len(p.s) && p.s[p.i] == '#' {
			p.i++
			continue
		}
		return b.String(), nil
	}
}

// inherited are the fields a crossref's child takes from its parent when
// it has none of its own; the parent's title is the child's booktitle
// (BibLaTeX's inheritance for chapters and papers in a book or
// proceedings).
var inherited = []string{"author", "editor", "publisher", "address", "location", "year", "date", "month",
	"booktitle", "journal", "journaltitle", "volume", "series", "edition", "institution", "organization", "school", "isbn"}

func inherit(r *bibRaw, byKey map[string]*bibRaw) {
	if r.done {
		return
	}
	r.done = true
	ref := r.fields["crossref"]
	if ref == "" {
		return
	}
	parent := byKey[strings.ToLower(strings.TrimSpace(ref))]
	if parent == nil || parent == r {
		return
	}
	inherit(parent, byKey)
	for _, f := range inherited {
		if _, ok := r.fields[f]; !ok {
			if v, ok := parent.fields[f]; ok {
				r.fields[f] = v
			}
		}
	}
	if _, ok := r.fields["booktitle"]; !ok {
		if t, ok := parent.fields["title"]; ok {
			r.fields["booktitle"] = t
		}
	}
}

// bibTypes are BibTeX's and BibLaTeX's types as CSL's.
var bibTypes = map[string]string{
	"article": "article-journal", "book": "book", "mvbook": "book", "booklet": "pamphlet",
	"inbook": "chapter", "incollection": "chapter", "bookinbook": "chapter", "suppbook": "chapter",
	"collection": "book", "mvcollection": "book", "proceedings": "book", "mvproceedings": "book",
	"inproceedings": "paper-conference", "conference": "paper-conference",
	"manual": "report", "techreport": "report", "report": "report",
	"mastersthesis": "thesis", "phdthesis": "thesis", "thesis": "thesis",
	"misc": "document", "online": "webpage", "electronic": "webpage", "www": "webpage",
	"unpublished": "manuscript", "patent": "patent", "periodical": "periodical",
	"dataset": "dataset", "software": "software",
}

// entry decodes the fields.
func (r *bibRaw) entry() *Entry {
	f := func(names ...string) string {
		for _, n := range names {
			if v, ok := r.fields[n]; ok && strings.TrimSpace(v) != "" {
				return decodeLaTeX(v)
			}
		}
		return ""
	}
	e := &Entry{Key: r.key, Type: bibTypes[r.typ]}
	if e.Type == "" {
		e.Type = "document"
	}
	e.Title = f("title")
	if st := f("subtitle"); st != "" {
		e.Title += ": " + st
	}
	e.Authors = parseNames(r.fields["author"])
	e.Editors = parseNames(r.fields["editor"])
	e.Year = yearIn(f("year"))
	if e.Year == "" {
		e.Year = yearIn(f("date"))
	}
	e.Container = f("journaltitle", "journal", "booktitle")
	if e.Container == "" && (r.typ == "online" || r.typ == "misc") {
		e.Container = f("howpublished")
	}
	e.Publisher = f("publisher", "institution", "school", "organization")
	e.Place = f("location", "address")
	e.Volume = f("volume")
	e.Issue = f("number", "issue")
	e.Pages = f("pages")
	e.Edition = f("edition")
	e.Genre = f("type")
	if e.Genre == "" {
		switch r.typ {
		case "phdthesis":
			e.Genre = "PhD diss."
		case "mastersthesis":
			e.Genre = "Master's thesis"
		}
		if e.Genre != "" { // Pandoc's, for CSL
			e.setMore("genre#csl", strings.Replace(e.Genre, "diss.", "thesis", 1))
		}
	}
	e.DOI = f("doi")
	e.URL = f("url")
	e.Note = f("note", "addendum")
	// what the CSL styles read beyond those, by CSL's names
	d := isoDate(f("date"))
	if d[0] == 0 {
		d[0] = atoiOr(e.Year)
	}
	if m := bibMonth(f("month")); m > 0 && d[1] == 0 {
		d[1] = m
	}
	e.setDate("issued", d)
	if u := isoDate(f("urldate")); u[0] != 0 {
		e.setDate("accessed", u)
	}
	if r.typ != "article" {
		e.setMore("number", f("number"))
	}
	for bib, csl := range map[string]string{"series": "collection-title", "isbn": "ISBN", "issn": "ISSN",
		"chapter": "chapter-number", "shorttitle": "title-short", "pagetotal": "number-of-pages",
		"eventtitle": "event-title", "venue": "event-place", "version": "version", "howpublished": "medium"} {
		e.setMore(csl, f(bib))
	}
	// BibTeX's titles are in title case, CSL's in sentence case (its
	// styles make title case of them where they want it), as Pandoc
	// reads them: what is in braces, and words with capitals inside
	// (DNA, iPhone), as written; not for another language (langid)
	lang := strings.ToLower(f("langid", "language"))
	english := lang == "" || strings.HasPrefix(lang, "en") || strings.Contains(lang, "english") || strings.Contains(lang, "american") || strings.Contains(lang, "british")
	if e.Title != "" && english {
		t := sentenceCase(r.fields["title"])
		if st := r.fields["subtitle"]; st != "" {
			t += ": " + sentenceCase(st)
		}
		e.setMore("title#sentence", t)
	}
	if bt := r.fields["booktitle"]; bt != "" && english && e.Container == f("booktitle") {
		e.setMore("container-title#sentence", sentenceCase(bt))
	}
	for _, role := range []string{"translator"} {
		e.setNames(role, parseNames(r.fields[role]))
	}
	return e
}

// parseNames splits a name list on "and" (outside braces) and reads
// each as BibTeX does: "von Last, Jr, First", "von Last, First" or
// "First von Last"; a name wholly in braces is literal.
func parseNames(raw string) []Name {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []Name
	for _, part := range splitTop(raw, " and ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.EqualFold(part, "others") {
			out = append(out, Name{Literal: "others"})
			continue
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") && topLevelClose(part) == len(part)-1 {
			out = append(out, Name{Literal: decodeLaTeX(part)})
			continue
		}
		out = append(out, parseName(part))
	}
	return out
}

// topLevelClose is where the brace that opens s closes.
func topLevelClose(s string) int {
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTop splits s on sep (case-insensitively, whitespace around it
// counted as spaces) outside braces.
func splitTop(s, sep string) []string {
	s = strings.Join(strings.Fields(s), " ")
	var out []string
	depth, last := 0, 0
	low := strings.ToLower(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
		if depth == 0 && strings.HasPrefix(low[i:], sep) {
			out = append(out, s[last:i])
			i += len(sep) - 1
			last = i + 1
		}
	}
	return append(out, s[last:])
}

func parseName(s string) Name {
	parts := splitTop(s, ",")
	words := func(x string) []string { return splitTop(strings.TrimSpace(x), " ") }
	var n Name
	var first, von, last []string
	// lowers is where the first and last word in lower case are among
	// ws (-1, -1: none)
	lowers := func(ws []string) (int, int) {
		f, l := -1, -1
		for i, w := range ws {
			if lowerWord(w) {
				if f < 0 {
					f = i
				}
				l = i
			}
		}
		return f, l
	}
	if len(parts) >= 2 { // von Last, [Jr,] First
		vl := words(parts[0])
		if len(parts) >= 3 {
			n.Suffix = decodeLaTeX(strings.TrimSpace(parts[1]))
		}
		first = words(parts[len(parts)-1])
		last = vl
		if len(vl) > 1 && lowerWord(vl[0]) {
			_, l := lowers(vl[:len(vl)-1])
			von, last = vl[:l+1], vl[l+1:]
		}
	} else { // First von Last
		ws := words(s)
		if len(ws) > 0 {
			f, l := lowers(ws[:len(ws)-1])
			if f < 0 {
				first, last = ws[:len(ws)-1], ws[len(ws)-1:]
			} else {
				first, von, last = ws[:f], ws[f:l+1], ws[l+1:]
			}
		}
	}
	join := func(ws []string) string { return decodeLaTeX(strings.Join(ws, " ")) }
	n.Given, n.Particle, n.Family = join(first), join(von), join(last)
	n.Drops = n.Particle != ""
	return n
}

// lowerWord: a word starting in lower case, as BibTeX tells a von part
// (a word in braces, {Van}, is not; an accented letter, {\"o}, is its
// letter).
func lowerWord(w string) bool {
	if strings.HasPrefix(w, "{") && !strings.HasPrefix(w, "{\\") {
		return false
	}
	for _, r := range decodeLaTeX(w) {
		if unicode.IsLetter(r) {
			return unicode.IsLower(r)
		}
	}
	return false
}

// Keys are the library's keys, sorted (for tests and messages).
func Keys(es []*Entry) []string {
	var ks []string
	for _, e := range es {
		ks = append(ks, e.Key)
	}
	sort.Strings(ks)
	return ks
}

// bibMonth is a BibTeX month (jan, January, 5) as a number, 0 for none.
func bibMonth(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 12 {
		return n
	}
	for i, m := range []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"} {
		if strings.HasPrefix(s, m) {
			return i + 1
		}
	}
	return 0
}

// sentenceCase is a BibTeX title in sentence case: its first word and
// the first after a colon capitalized, the other words lowercase unless
// in braces or written with capitals after their first letter.
func sentenceCase(raw string) string {
	var out []string
	capNext := true
	for _, w := range splitTop(strings.Join(strings.Fields(raw), " "), " ") {
		d := decodeLaTeX(w)
		switch {
		case strings.HasPrefix(w, "{") || strings.HasPrefix(w, "\\"):
		case mixedCase(d):
		case capNext:
			d = capFirst(strings.ToLower(d))
		default:
			d = strings.ToLower(d)
		}
		capNext = strings.HasSuffix(d, ":") || strings.HasSuffix(d, "?") || strings.HasSuffix(d, "!")
		out = append(out, d)
	}
	return strings.Join(out, " ")
}

// mixedCase: a capital after the first letter.
func mixedCase(s string) bool {
	for i, r := range []rune(s) {
		if i > 0 && unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

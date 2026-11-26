package cite

import (
	"bytes"
	"embed"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// A CSL style (Citation Style Language 1.0), read from a .csl file: the
// common part of the language that APA, Chicago author-date, IEEE and
// Vancouver (NLM) use — macros, text, number, label, date, names (et
// al., initials, inverted names, substitutes), group, choose, sorting,
// citation numbers (collapsed to 1–3), years collapsed (Doe 1999,
// 2001) and told apart (1999a, 1999b), page ranges, the en-US terms.
// Left out: disambiguation by names and given names, note styles'
// footnotes (a note style cites in the text), other locales' terms,
// name-part formatting. What the subset does not know is skipped.

//go:embed csl
var cslFiles embed.FS

// builtinStyles are the styles Accolade has, by the names csl: takes.
var builtinStyles = map[string]string{
	"apa":                   "apa.csl",
	"chicago-author-date":   "chicago-author-date.csl",
	"ieee":                  "ieee.csl",
	"vancouver":             "nlm-citation-sequence.csl",
	"nlm-citation-sequence": "nlm-citation-sequence.csl",
}

// BuiltinStyles are the names of the styles Accolade has, for csl:.
func BuiltinStyles() []string {
	return []string{"apa", "chicago-author-date", "ieee", "vancouver"}
}

// BuiltinCSL is the .csl file of a style Accolade has: name is its name
// or a file name (apa, apa.csl, styles/ieee.csl); nil for none.
func BuiltinCSL(name string) []byte {
	b := strings.TrimSuffix(strings.ToLower(path.Base(strings.ReplaceAll(strings.TrimSpace(name), `\`, "/"))), ".csl")
	f, ok := builtinStyles[b]
	if !ok {
		return nil
	}
	data, _ := cslFiles.ReadFile("csl/" + f)
	return data
}

var styleCache sync.Map // string (the style's text) → *Style

// BuiltinStyle is a style Accolade has, read (nil for none).
func BuiltinStyle(name string) *Style {
	data := BuiltinCSL(name)
	if data == nil {
		return nil
	}
	st, _ := ParseStyle(data)
	return st
}

// node is an element of a style.
type node struct {
	name string
	attr map[string]string
	kids []*node
}

func (n *node) child(name string) *node {
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	return nil
}

func parseXML(data []byte) (*node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	var stack []*node
	var root *node
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := t.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local, attr: map[string]string{}}
			for _, a := range t.Attr {
				n.attr[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.kids = append(p.kids, n)
			} else {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 { // a term's text
				p := stack[len(stack)-1]
				p.attr["#text"] += string(t)
			}
		}
	}
	if root == nil || root.name != "style" {
		return nil, fmt.Errorf("not a CSL style")
	}
	return root, nil
}

// Style is a CSL style, read.
type Style struct {
	Title        string
	root         *node
	macros       map[string]*node
	cit, bib     *node
	terms        map[string][2]string // name/form → single, multiple
	dates        map[string]*node     // the locale's text and numeric dates
	punctInQuote bool
}

// ParseStyle reads a .csl file.
func ParseStyle(data []byte) (*Style, error) {
	if v, ok := styleCache.Load(string(data)); ok {
		return v.(*Style), nil
	}
	root, err := parseXML(data)
	if err != nil {
		return nil, err
	}
	st := &Style{root: root, macros: map[string]*node{}, terms: map[string][2]string{}, dates: map[string]*node{}}
	loc, _ := cslFiles.ReadFile("csl/locales-en-US.xml")
	if l, err := parseXMLAny(loc); err == nil {
		st.locale(l)
	}
	for _, k := range root.kids {
		switch k.name {
		case "info":
			if t := k.child("title"); t != nil {
				st.Title = strings.TrimSpace(t.attr["#text"])
			}
		case "macro":
			st.macros[k.attr["name"]] = k
		case "citation":
			st.cit = k
		case "bibliography":
			st.bib = k
		case "locale":
			if lang := k.attr["lang"]; lang == "" || strings.HasPrefix(lang, "en") {
				st.locale(k)
			}
		}
	}
	if st.cit == nil || st.cit.child("layout") == nil {
		return nil, fmt.Errorf("a CSL style without a citation")
	}
	styleCache.Store(string(data), st)
	return st, nil
}

// parseXMLAny is parseXML for a locale file.
func parseXMLAny(data []byte) (*node, error) {
	d := bytes.Replace(data, []byte("<locale"), []byte("<style"), 1)
	d = bytes.Replace(d, []byte("</locale>"), []byte("</style>"), 1)
	return parseXML(d)
}

// locale takes a locale's terms, dates and options.
func (st *Style) locale(l *node) {
	for _, k := range l.kids {
		switch k.name {
		case "style-options":
			if v, ok := k.attr["punctuation-in-quote"]; ok {
				st.punctInQuote = v == "true"
			}
		case "date":
			st.dates[k.attr["form"]] = k
		case "terms":
			for _, t := range k.kids {
				if t.name != "term" {
					continue
				}
				form := cmpOr(t.attr["form"], "long")
				var v [2]string
				if s, m := t.child("single"), t.child("multiple"); s != nil || m != nil {
					if s != nil {
						v[0] = s.attr["#text"]
					}
					if m != nil {
						v[1] = m.attr["#text"]
					}
				} else {
					v[0] = t.attr["#text"]
					v[1] = v[0]
				}
				st.terms[t.attr["name"]+"/"+form] = v
			}
		}
	}
}

// term is a term's text in a form (falling back as CSL does).
func (st *Style) term(name, form string, plural bool) string {
	forms := []string{form}
	switch form {
	case "verb-short":
		forms = append(forms, "verb", "long")
	case "symbol":
		forms = append(forms, "short", "long")
	case "short", "verb":
		forms = append(forms, "long")
	case "":
		forms = []string{"long"}
	}
	for _, f := range forms {
		if v, ok := st.terms[name+"/"+f]; ok {
			if plural {
				return v[1]
			}
			return v[0]
		}
	}
	return ""
}

// item is a cited work as a style shows it.
type item struct {
	e        *Entry
	num      int    // its citation number
	suffix   string // its year's letter
	locator  string
	label    string // the locator's: page, chapter …
	first    bool   // its first citation
	ibid     bool   // the same work as the citation before
	prefix   string // Markdown before it, as written
	after    string // Markdown after it, as written
	suppress bool   // its author left out
	key      string // as written, for one the library lacks
}

// rctx is one rendering of an item.
type rctx struct {
	st         *Style
	it         *item
	area       *node // the citation or the bibliography
	bib, sort  bool  // the bibliography's; a sort key's (plain text)
	suppressAu bool  // the first names are left out
	authorDone bool
	depth      int    // names within names (a substitute's)
	author     string // the first names, as rendered
	keyOpts    map[string]string
	used       map[string]bool // the variables a substitute used: not again
}

// opt is an inheritable option: the element's, the citation's or
// bibliography's, the style's.
func (c *rctx) opt(n *node, name string) string {
	if n != nil {
		if v, ok := n.attr[name]; ok {
			return v
		}
	}
	if v, ok := c.keyOpts[name]; ok {
		return v
	}
	if c.area != nil {
		if v, ok := c.area.attr[name]; ok {
			return v
		}
	}
	return c.st.root.attr[name]
}

// res is what an element rendered: its text, whether it asked for a
// variable and whether one had a value (a group of empty variables is
// left out).
type res struct {
	s              string
	called, filled bool
}

func (c *rctx) kids(n *node, delim string) res {
	var out res
	parts := c.collect(n.kids, &out)
	out.s = joinParts(parts, delim)
	return out
}

// collect renders elements, a choose's chosen branch in their place (a
// group's delimiter goes between the branch's elements too).
func (c *rctx) collect(ns []*node, out *res) []string {
	var parts []string
	for _, k := range ns {
		if k.name == "choose" {
			for _, br := range k.kids {
				if br.name == "else" || c.test(br) {
					parts = append(parts, c.collect(br.kids, out)...)
					break
				}
			}
			continue
		}
		r := c.el(k)
		out.called = out.called || r.called
		out.filled = out.filled || r.filled
		if r.s != "" {
			parts = append(parts, r.s)
		}
	}
	return parts
}

// el renders an element.
func (c *rctx) el(n *node) res {
	switch n.name {
	case "layout":
		return c.kids(n, n.attr["delimiter"])
	case "text":
		return c.text(n)
	case "number":
		v := c.variable(n.attr["variable"], "")
		r := res{called: true, filled: v != ""}
		if v != "" {
			r.s = c.format(n, c.number(n, v))
		}
		return r
	case "label":
		return res{s: c.format(n, c.label(n, n.attr["variable"], 0))}
	case "date":
		v := n.attr["variable"]
		r := res{called: true}
		if d := c.date(v); d[0] != 0 || c.dateLiteral(v) != "" {
			r.filled = true
			r.s = c.format(n, c.renderDate(n, d, c.dateLiteral(v)))
		}
		return r
	case "names":
		return c.names(n, nil)
	case "group":
		r := c.kids(n, n.attr["delimiter"])
		if r.called && !r.filled || r.s == "" {
			return res{called: r.called}
		}
		r.s = c.format(n, r.s)
		return r
	case "choose":
		return c.kids(&node{kids: []*node{n}}, "")
	}
	return res{}
}

func (c *rctx) text(n *node) res {
	var r res
	switch {
	case n.attr["macro"] != "":
		m := c.st.macros[n.attr["macro"]]
		if m == nil {
			return r
		}
		r = c.kids(m, "")
		if r.called && !r.filled {
			r.s = ""
		}
	case n.attr["variable"] != "":
		v := n.attr["variable"]
		r.called = true
		s := c.variable(v, n.attr["form"])
		if s == "" {
			return r
		}
		r.filled = true
		switch v {
		case "page", "locator":
			s = c.pageRange(s)
		}
		r.s = c.esc(s)
		if v == "citation-number" || v == "year-suffix" {
			r.s = s
		}
	case n.attr["term"] != "":
		r.s = c.esc(c.st.term(n.attr["term"], n.attr["form"], n.attr["plural"] == "true"))
	case n.attr["value"] != "":
		r.s = c.esc(n.attr["value"])
	}
	if r.s == "" {
		return r
	}
	r.s = c.format(n, r.s)
	return r
}

// esc is text as Markdown (as is in a sort key).
func (c *rctx) esc(s string) string {
	if c.sort {
		return s
	}
	return Escape(s)
}

// variable is an item's ordinary variable, "" when it has none (or a
// substitute used it).
func (c *rctx) variable(name, form string) string {
	if c.used[name] {
		return ""
	}
	e := c.it.e
	switch name {
	case "citation-number":
		if c.it.num == 0 {
			return ""
		}
		return strconv.Itoa(c.it.num)
	case "year-suffix":
		return c.it.suffix
	case "locator":
		return c.it.locator
	case "citation-label":
		return e.Key
	case "title":
		if form == "short" {
			if s := e.More["title-short"]; s != "" {
				return s
			}
			return shortTitle(cmpOr(e.More["title#sentence"], e.Title))
		}
		return cmpOr(e.More["title#sentence"], e.Title)
	case "container-title":
		if form == "short" {
			if s := e.More["container-title-short"]; s != "" {
				return s
			}
		}
		return cmpOr(e.More["container-title#sentence"], e.Container)
	case "publisher":
		return e.Publisher
	case "publisher-place":
		return e.Place
	case "volume":
		return e.Volume
	case "issue":
		return e.Issue
	case "page":
		return e.Pages
	case "page-first":
		if s := e.More["page-first"]; s != "" {
			return s
		}
		p := e.Pages
		if i := strings.IndexAny(p, "-–,"); i > 0 {
			p = p[:i]
		}
		return strings.TrimSpace(p)
	case "edition":
		return e.Edition
	case "genre":
		return e.Genre
	case "DOI":
		return e.DOI
	case "URL":
		return e.URL
	case "note":
		return e.Note
	}
	return e.More[name]
}

func (c *rctx) nameVar(name string) []Name {
	if c.used[name] {
		return nil
	}
	switch name {
	case "author":
		return c.it.e.Authors
	case "editor":
		return c.it.e.Editors
	}
	return c.it.e.MoreNames[name]
}

func (c *rctx) date(name string) [3]int {
	if c.used[name] {
		return [3]int{}
	}
	if d, ok := c.it.e.Dates[name]; ok {
		return d
	}
	if name == "issued" {
		return [3]int{atoiOr(c.it.e.Year)}
	}
	return [3]int{}
}

// dateLiteral is a date that is no number (forthcoming).
func (c *rctx) dateLiteral(name string) string {
	if name == "issued" && c.date(name)[0] == 0 && !c.used[name] {
		return c.it.e.Year
	}
	return ""
}

// test is a choose branch's condition.
func (c *rctx) test(n *node) bool {
	match := cmpOr(n.attr["match"], "all")
	var results []bool
	for cond, vals := range n.attr {
		for _, v := range strings.Fields(vals) {
			var t bool
			switch cond {
			case "type":
				t = c.it.e.Type == v
			case "variable":
				t = c.has(v)
			case "is-numeric":
				t = numeric(c.variable(v, ""))
			case "is-uncertain-date":
				t = false
			case "locator":
				t = c.it.locator != "" && c.it.label == v
			case "position":
				switch v {
				case "first":
					t = !c.bib && c.it.first
				case "subsequent":
					t = !c.bib && !c.it.first
				case "ibid", "ibid-with-locator":
					t = !c.bib && c.it.ibid
				}
			case "disambiguate":
				t = false
			default:
				continue
			}
			results = append(results, t)
		}
	}
	if len(results) == 0 {
		return false
	}
	switch match {
	case "any":
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	case "none":
		for _, r := range results {
			if r {
				return false
			}
		}
		return true
	}
	for _, r := range results {
		if !r {
			return false
		}
	}
	return true
}

// has: the item has the variable (of any kind).
func (c *rctx) has(v string) bool {
	return c.variable(v, "") != "" || len(c.nameVar(v)) > 0 || c.date(v)[0] != 0 || c.dateLiteral(v) != ""
}

// numeric: a number, numbers (12, 15–17 & 20) or one with a letter (2nd, 12a).
func numeric(s string) bool {
	if s == "" {
		return false
	}
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune("-–,&; ", r) }) {
		digits := 0
		for _, r := range p {
			if unicode.IsDigit(r) {
				digits++
			}
		}
		if digits == 0 || utf8.RuneCountInString(p)-digits > 2 || !unicode.IsDigit([]rune(p)[0]) {
			return false
		}
	}
	return true
}

func (c *rctx) number(n *node, v string) string {
	if !numeric(v) {
		return c.esc(v)
	}
	num, err := strconv.Atoi(v)
	if err != nil {
		return c.esc(dashes(v))
	}
	switch n.attr["form"] {
	case "ordinal", "long-ordinal":
		if n.attr["form"] == "long-ordinal" && num >= 1 && num <= 10 {
			if t := c.st.term(fmt.Sprintf("long-ordinal-%02d", num), "", false); t != "" {
				return t
			}
		}
		return strconv.Itoa(num) + c.ordinal(num)
	case "roman":
		return roman(num)
	}
	return v
}

// ordinal is a number's ordinal suffix (1st, 2nd, 11th).
func (c *rctx) ordinal(n int) string {
	if n%100 < 11 || n%100 > 13 {
		if t := c.st.term(fmt.Sprintf("ordinal-%02d", n%10), "", false); t != "" {
			return t
		}
	}
	return cmpOr(c.st.term("ordinal", "", false), "th")
}

func roman(n int) string {
	if n <= 0 || n >= 4000 {
		return strconv.Itoa(n)
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

// label is a variable's term (pp., ed., eds.); count is a name list's
// length (0: from the value).
func (c *rctx) label(n *node, v string, count int) string {
	term, val := v, ""
	switch v {
	case "locator":
		term, val = c.it.label, c.it.locator
	case "":
		return ""
	default:
		val = c.variable(v, "")
	}
	if count == 0 && val == "" {
		return ""
	}
	plural := false
	switch n.attr["plural"] {
	case "always":
		plural = true
	case "never":
	default:
		if count > 0 {
			plural = count > 1
		} else {
			plural = strings.ContainsAny(val, "-–,&") || strings.Contains(val, " and ")
		}
	}
	s := c.st.term(term, n.attr["form"], plural)
	if n.attr["strip-periods"] == "true" {
		s = strings.ReplaceAll(s, ".", "")
	}
	return c.esc(s)
}

// renderDate is a date by the element's parts (or the locale's form).
func (c *rctx) renderDate(n *node, d [3]int, literal string) string {
	if d[0] == 0 {
		return c.esc(literal)
	}
	parts := n.kids
	delim := n.attr["delimiter"]
	if form := n.attr["form"]; form != "" && c.st.dates[form] != nil {
		loc := c.st.dates[form]
		over := map[string]*node{}
		for _, k := range n.kids {
			over[k.attr["name"]] = k
		}
		want := cmpOr(n.attr["date-parts"], "year-month-day")
		parts = nil
		for _, k := range loc.kids {
			name := k.attr["name"]
			if !strings.Contains(want, name) {
				continue
			}
			m := &node{name: k.name, attr: map[string]string{}}
			for a, v := range k.attr {
				m.attr[a] = v
			}
			if o := over[name]; o != nil {
				for a, v := range o.attr {
					m.attr[a] = v
				}
			}
			parts = append(parts, m)
		}
		delim = loc.attr["delimiter"]
	}
	var out []string
	for _, p := range parts {
		if p.name != "date-part" {
			continue
		}
		var s string
		switch p.attr["name"] {
		case "year":
			s = strconv.Itoa(d[0])
			if p.attr["form"] == "short" {
				s = fmt.Sprintf("%02d", d[0]%100)
			}
		case "month":
			if d[1] < 1 || d[1] > 12 {
				continue
			}
			switch p.attr["form"] {
			case "numeric":
				s = strconv.Itoa(d[1])
			case "numeric-leading-zeros":
				s = fmt.Sprintf("%02d", d[1])
			default:
				s = c.st.term(fmt.Sprintf("month-%02d", d[1]), p.attr["form"], false)
				if p.attr["strip-periods"] == "true" {
					s = strings.ReplaceAll(s, ".", "")
				}
			}
		case "day":
			if d[2] == 0 {
				continue
			}
			s = strconv.Itoa(d[2])
			switch p.attr["form"] {
			case "numeric-leading-zeros":
				s = fmt.Sprintf("%02d", d[2])
			case "ordinal":
				s += c.ordinal(d[2])
			}
		}
		out = append(out, c.format(p, s))
	}
	return joinParts(out, delim)
}

// names renders a names element; inherit is the element whose name,
// et-al and label a substitute's bare names take.
func (c *rctx) names(n *node, inherit *node) res {
	c.depth++
	defer func() { c.depth-- }()
	r := res{called: true}
	src := n
	if inherit != nil && n.child("name") == nil && n.child("et-al") == nil && n.child("label") == nil {
		src = inherit
	}
	nameN, etAlN := src.child("name"), src.child("et-al")
	labelN := src.child("label")
	labelFirst := false // the label before the names (editor's Ed. Doe)
	if labelN != nil {
		for _, k := range src.kids {
			if k.name == "name" {
				break
			}
			if k == labelN {
				labelFirst = true
			}
		}
	}
	vars := strings.Fields(n.attr["variable"])
	var parts []string
	for _, v := range vars {
		list := c.nameVar(v)
		if len(list) == 0 {
			continue
		}
		s := c.nameList(nameN, etAlN, list)
		if s == "" {
			continue
		}
		if nameN != nil && nameN.attr["form"] == "count" {
			parts = append(parts, s)
			continue
		}
		if labelN != nil {
			if l := c.format(labelN, c.label(labelN, v, len(list))); l != "" {
				if labelFirst {
					s = l + s
				} else {
					s += l
				}
			}
		}
		parts = append(parts, s)
	}
	if len(parts) > 0 {
		r.filled = true
		r.s = c.format(n, joinParts(parts, cmpOr(n.attr["delimiter"], c.opt(nil, "names-delimiter"))))
	} else if sub := n.child("substitute"); sub != nil {
		for _, k := range sub.kids {
			before := c.usedSet()
			var kr res
			if k.name == "names" {
				kr = c.names(k, src)
			} else {
				kr = c.el(k)
			}
			if kr.s != "" {
				// the variables it used are not used again
				for _, v := range append(strings.Fields(k.attr["variable"]), substVars(c, k)...) {
					if !before[v] {
						c.used[v] = true
					}
				}
				r.filled, r.s = true, c.format(n, kr.s)
				break
			}
		}
	}
	if r.s != "" && !c.authorDone && !c.sort && c.depth == 1 {
		c.authorDone = true
		c.author = r.s
		if c.suppressAu {
			return res{called: true}
		}
	}
	return r
}

func (c *rctx) usedSet() map[string]bool {
	m := map[string]bool{}
	for k, v := range c.used {
		m[k] = v
	}
	return m
}

// substVars are the variables a substitute's text macro asks for
// directly (title, container-title), so they are not shown twice.
func substVars(c *rctx, k *node) []string {
	var out []string
	var walk func(n *node, depth int)
	walk = func(n *node, depth int) {
		if depth > 8 {
			return
		}
		if v := n.attr["variable"]; v != "" && (n.name == "text" || n.name == "names" || n.name == "date" || n.name == "number") {
			if c.has(v) {
				out = append(out, strings.Fields(v)...)
			}
		}
		if m := n.attr["macro"]; m != "" && c.st.macros[m] != nil {
			walk(c.st.macros[m], depth+1)
		}
		for _, kid := range n.kids {
			walk(kid, depth+1)
		}
	}
	walk(k, 0)
	return out
}

// nameList is a list of names as a name element sets it.
func (c *rctx) nameList(nn, etAlN *node, list []Name) string {
	if nn == nil {
		nn = &node{attr: map[string]string{}}
	}
	o := func(name string) string { return c.opt(nn, name) }
	form := cmpOr(o("form"), "long")
	// et al.
	names := list
	etAl := false
	if l := len(names); l > 0 && names[l-1].Literal == "others" {
		names, etAl = names[:l-1], true
	}
	min, first := atoiOr(o("et-al-min")), atoiOr(o("et-al-use-first"))
	if !c.bib && !c.it.first && !c.sort {
		if v := o("et-al-subsequent-min"); v != "" {
			min = atoiOr(v)
		}
		if v := o("et-al-subsequent-use-first"); v != "" {
			first = atoiOr(v)
		}
	}
	if c.sort {
		if v, ok := c.keyOpts["names-min"]; ok {
			min = atoiOr(v)
		}
		if v, ok := c.keyOpts["names-use-first"]; ok {
			first = atoiOr(v)
		}
	}
	var last *Name
	if min > 0 && first > 0 && len(names) >= min && first < len(names) {
		if o("et-al-use-last") == "true" && first+2 <= len(names) {
			l := names[len(names)-1]
			last = &l
		}
		names, etAl = names[:first], true
	}
	if form == "count" {
		return strconv.Itoa(len(names))
	}
	delim := cmpOr(o("delimiter"), ", ")
	sortOrder := o("name-as-sort-order")
	if c.sort {
		sortOrder = "all"
	}
	var out []string
	for i, n := range names {
		inv := sortOrder == "all" || sortOrder == "first" && i == 0
		out = append(out, c.oneName(nn, n, form, inv))
	}
	if len(out) == 0 {
		return ""
	}
	if c.sort {
		return strings.Join(out, " ")
	}
	and := ""
	switch o("and") {
	case "text":
		and = c.st.term("and", "", false)
	case "symbol":
		and = "&"
	}
	var b strings.Builder
	for i, s := range out {
		if i > 0 {
			if i == len(out)-1 && and != "" && !etAl {
				prev := sortOrder == "all" || sortOrder == "first" && i-1 == 0
				useDelim := false
				switch cmpOr(o("delimiter-precedes-last"), "contextual") {
				case "always":
					useDelim = true
				case "never":
				case "after-inverted-name":
					useDelim = prev
				default:
					useDelim = len(out) >= 3
				}
				if useDelim {
					b.WriteString(delim)
				} else {
					b.WriteString(" ")
				}
				b.WriteString(Escape(and) + " ")
			} else {
				b.WriteString(delim)
			}
		}
		b.WriteString(s)
	}
	if last != nil {
		b.WriteString(delim + "… " + c.oneName(nn, *last, form, sortOrder == "all"))
	} else if etAl {
		term := "et-al"
		if etAlN != nil && etAlN.attr["term"] != "" {
			term = etAlN.attr["term"]
		}
		t := Escape(c.st.term(term, "", false))
		if etAlN != nil {
			t = c.format(etAlN, t)
		}
		useDelim := false
		switch cmpOr(o("delimiter-precedes-et-al"), "contextual") {
		case "always":
			useDelim = true
		case "never":
		case "after-inverted-name":
			useDelim = sortOrder != ""
		default:
			useDelim = len(out) >= 2
		}
		if useDelim {
			b.WriteString(delim + t)
		} else {
			b.WriteString(" " + t)
		}
	}
	return c.format(nn, b.String())
}

// oneName is a name: short (the family name), long, or inverted.
func (c *rctx) oneName(nn *node, n Name, form string, inverted bool) string {
	if n.Literal != "" {
		return c.esc(n.Literal)
	}
	o := func(name string) string { return c.opt(nn, name) }
	family := n.Family
	particle := n.Particle
	if form == "short" {
		return c.esc(strings.TrimSpace(particle + " " + family))
	}
	given := n.Given
	if iw, ok := c.optOK(nn, "initialize-with"); ok && o("initialize") != "false" {
		given = initials(given, iw, o("initialize-with-hyphen") != "false")
	}
	if c.sort {
		return strings.TrimSpace(family + " " + given + " " + particle)
	}
	if !inverted {
		s := strings.TrimSpace(given + " " + strings.TrimSpace(particle+" "+family))
		if n.Suffix != "" {
			s += " " + n.Suffix
		}
		return c.esc(s)
	}
	sep := ", "
	if v, ok := c.optOK(nn, "sort-separator"); ok {
		sep = v
	}
	fam := strings.TrimSpace(particle + " " + family)
	if particle != "" && o("demote-non-dropping-particle") == "display-and-sort" {
		fam, given = family, strings.TrimSpace(given+" "+particle)
	}
	s := fam
	if given != "" {
		s += sep + given
	}
	if n.Suffix != "" {
		s += sep + n.Suffix
	}
	return c.esc(s)
}

func (c *rctx) optOK(n *node, name string) (string, bool) {
	if v, ok := n.attr[name]; ok {
		return v, true
	}
	if c.area != nil {
		if v, ok := c.area.attr[name]; ok {
			return v, true
		}
	}
	v, ok := c.st.root.attr[name]
	return v, ok
}

// initials are given names as initials: "John Ronald" with ". " is
// "J. R.", Jean-Paul "J.-P." (JP without the hyphen).
func initials(given, with string, hyphen bool) string {
	var out []string
	for _, w := range strings.Fields(given) {
		var parts []string
		for _, p := range strings.Split(w, "-") {
			r := []rune(strings.Trim(p, "."))
			if len(r) == 0 {
				continue
			}
			if len(r) > 1 && unicode.IsLower(r[0]) { // a particle: as is
				parts = append(parts, string(r))
				continue
			}
			parts = append(parts, string(unicode.ToUpper(r[0]))+with)
		}
		sep := ""
		if hyphen {
			sep = "-"
		}
		joined := strings.Join(parts, sep)
		if hyphen && len(parts) > 1 {
			joined = strings.Join(trimEach(parts), sep) + with
		}
		out = append(out, joined)
	}
	return strings.TrimSpace(strings.Join(out, ""))
}

func trimEach(ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = strings.TrimRight(p, " ")
	}
	return out
}

// pageRange is a range with an en dash, shortened as the style says.
func (c *rctx) pageRange(s string) string {
	s = dashes(s)
	format := c.opt(nil, "page-range-format")
	if format == "" || format == "expanded" {
		return s
	}
	return rangeDash2.ReplaceAllStringFunc(s, func(m string) string {
		p := strings.Split(m, "–")
		a, b := p[0], p[1]
		if len(b) < len(a) { // 321–28: written short
			b = a[:len(a)-len(b)] + b
		}
		if len(a) != len(b) {
			return a + "–" + b
		}
		keep := 0 // how many of b's last digits to show
		diff := 0
		for diff < len(a) && a[diff] == b[diff] {
			diff++
		}
		keep = len(b) - diff
		switch format {
		case "minimal":
		case "minimal-two":
			keep = max(keep, 2)
		case "chicago", "chicago-15", "chicago-16":
			n, _ := strconv.Atoi(a)
			switch {
			case n < 100 || n%100 == 0:
				keep = len(b)
			case n%100 < 10:
				// 101–8: the changed part
			default:
				keep = max(keep, 2)
			}
		}
		keep = min(max(keep, 1), len(b))
		return a + "–" + b[len(b)-keep:]
	})
}

var rangeDash2 = mustRegexp(`\d+–\d+`)

// format sets an element's text: its case, quotes, font, affixes.
func (c *rctx) format(n *node, s string) string {
	if s == "" {
		return ""
	}
	a := n.attr
	if a["strip-periods"] == "true" && n.name != "label" {
		s = strings.ReplaceAll(s, ".", "")
	}
	switch a["text-case"] {
	case "lowercase":
		s = strings.ToLower(s)
	case "uppercase":
		s = strings.ToUpper(s)
	case "capitalize-first", "sentence":
		s = capFirst(s)
	case "capitalize-all":
		s = capWords(s, false)
	case "title":
		s = capWords(s, true)
	}
	if !c.sort {
		if a["quotes"] == "true" {
			s = c.st.term("open-quote", "", false) + s + c.st.term("close-quote", "", false)
		}
		if a["font-style"] == "italic" || a["font-style"] == "oblique" {
			s = "*" + s + "*"
		}
		if a["font-weight"] == "bold" {
			s = "**" + s + "**"
		}
	}
	return c.affix(a["prefix"]) + s + c.affix(a["suffix"])
}

func (c *rctx) affix(s string) string {
	if s == "" || c.sort {
		return s
	}
	return Escape(s)
}

func capFirst(s string) string {
	for i, r := range s {
		if unicode.IsLetter(r) {
			return s[:i] + string(unicode.ToUpper(r)) + s[i+utf8.RuneLen(r):]
		}
	}
	return s
}

var smallWords = map[string]bool{"a": true, "an": true, "and": true, "as": true, "at": true, "but": true, "by": true,
	"down": true, "for": true, "from": true, "in": true, "into": true, "nor": true, "of": true, "on": true,
	"onto": true, "or": true, "over": true, "so": true, "the": true, "till": true, "to": true, "up": true,
	"via": true, "with": true, "yet": true}

// capWords capitalizes each word's first letter (title: not the small
// words inside, and only words written all lowercase).
func capWords(s string, title bool) string {
	words := strings.Split(s, " ")
	for i, w := range words {
		if title && i > 0 && i < len(words)-1 && smallWords[strings.ToLower(strings.Trim(w, "*,.:;“”\"'"))] {
			continue
		}
		if title && w != strings.ToLower(w) {
			continue
		}
		words[i] = capFirst(w)
	}
	return strings.Join(words, " ")
}

// joinParts joins non-empty parts with delim, a delimiter's punctuation
// not doubled.
func joinParts(parts []string, delim string) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteString(delim)
		}
		b.WriteString(p)
	}
	return b.String()
}

// tidy is the rendered text's punctuation: no doubled periods (Ed..,
// Title?.), a period or comma inside the closing quote (en-US).
func (st *Style) tidy(s string) string {
	s = doubled.ReplaceAllString(s, "$1$2")
	s = strings.ReplaceAll(s, ",,", ",")
	s = strings.ReplaceAll(s, " ,", ",")
	s = strings.ReplaceAll(s, "  ", " ")
	if st.punctInQuote {
		s = quotePunct.ReplaceAllString(s, "$2$1")
	}
	return s
}

var (
	doubled    = mustRegexp(`([.?!])((?:\*|”|’|\))*)\\?\.`)
	quotePunct = mustRegexp(`([”’]+)([.,])`)
)

// numberWorks gives each work its citation number and puts them in the
// bibliography's order: by the bibliography's sort (else as first
// cited).
func (st *Style) order(items []*item) {
	for i, it := range items {
		it.num = i + 1
	}
	if st.bib != nil && st.bib.child("sort") != nil {
		st.sortItems(items, st.bib, st.bib.child("sort"))
		for i, it := range items {
			it.num = i + 1
		}
	}
}

// sortItems sorts items by a sort element's keys.
func (st *Style) sortItems(items []*item, area, srt *node) {
	keys := make([][]string, len(items))
	for i, it := range items {
		for _, k := range srt.kids {
			if k.name != "key" {
				continue
			}
			keys[i] = append(keys[i], st.sortKey(it, area, k))
		}
	}
	idx := make([]int, len(items))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ka, kb := keys[idx[a]], keys[idx[b]]
		for j, k := range srt.kids {
			if j >= len(ka) {
				break
			}
			_ = k
			x, y := ka[j], kb[j]
			if x == y {
				continue
			}
			if x == "" || y == "" { // empty last
				return y == ""
			}
			desc := keyNode(srt, j).attr["sort"] == "descending"
			if desc {
				return x > y
			}
			return x < y
		}
		return false
	})
	sorted := make([]*item, len(items))
	for i, j := range idx {
		sorted[i] = items[j]
	}
	copy(items, sorted)
}

func keyNode(srt *node, j int) *node {
	n := 0
	for _, k := range srt.kids {
		if k.name == "key" {
			if n == j {
				return k
			}
			n++
		}
	}
	return &node{attr: map[string]string{}}
}

// sortKey is an item's value for a sort key, as text that sorts.
func (st *Style) sortKey(it *item, area, k *node) string {
	c := &rctx{st: st, it: it, area: area, bib: area == st.bib, sort: true, keyOpts: k.attr, used: map[string]bool{}}
	var s string
	if v := k.attr["variable"]; v != "" {
		switch {
		case v == "citation-number":
			return fmt.Sprintf("%08d", it.num)
		case len(c.nameVar(v)) > 0:
			s = c.nameList(nil, nil, c.nameVar(v))
		case c.date(v)[0] != 0:
			d := c.date(v)
			return fmt.Sprintf("%04d%02d%02d", d[0], d[1], d[2])
		default:
			s = c.variable(v, "")
			if numeric(s) {
				return fmt.Sprintf("%08d", atoiOr(strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsDigit(r) })[0]))
			}
		}
	} else if m := st.macros[k.attr["macro"]]; m != nil {
		s = c.kids(m, "").s
	}
	return fold(strings.TrimLeft(strings.TrimSpace(s), "“‘\"'([*"))
}

// render is an item in an area (the citation's or the bibliography's
// layout).
func (st *Style) render(it *item, area *node, suppress bool) (text, author string) {
	c := &rctx{st: st, it: it, area: area, bib: area == st.bib, suppressAu: suppress, used: map[string]bool{}}
	lay := area.child("layout")
	var r res
	if c.bib && area.attr["second-field-align"] != "" && len(lay.kids) > 1 {
		first := c.el(lay.kids[0]).s
		rest := c.kids(&node{kids: lay.kids[1:]}, lay.attr["delimiter"]).s
		r.s = joinParts(nonEmpty(first, rest), " ")
	} else {
		r = c.kids(lay, lay.attr["delimiter"])
	}
	if c.bib { // the citation's layout affixes are the whole citation's (cluster)
		r.s = c.format(lay, r.s)
	}
	return st.tidy(r.s), c.author
}

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// cluster is a citation of items (one in the text's when inText).
func (st *Style) cluster(items []*item, inText bool) string {
	lay := st.cit.child("layout")
	if srt := st.cit.child("sort"); srt != nil && !inText {
		var known []*item
		for _, it := range items {
			if it.e != nil {
				known = append(known, it)
			}
		}
		if len(known) == len(items) {
			st.sortItems(items, st.cit, srt)
		}
	}
	var author string
	if inText && len(items) == 1 && items[0].e != nil {
		it := items[0]
		// its author, then the citation without (a numeric style's
		// has none: the citation alone, as Pandoc's)
		_, author = st.render(it, st.cit, false)
		it.suppress = author != ""
	}
	collapse := st.cit.attr["collapse"]
	delim := lay.attr["delimiter"]
	var parts []string
	for i := 0; i < len(items); i++ {
		it := items[i]
		if it.e == nil {
			parts = append(parts, join(it.prefix, Escape(it.key)+"?")+it.after)
			continue
		}
		plain := func(x *item) bool { return x.e != nil && x.locator == "" && x.prefix == "" && x.after == "" }
		if collapse == "citation-number" && plain(it) {
			j := i
			for j+1 < len(items) && plain(items[j+1]) && items[j+1].num == items[j].num+1 {
				j++
			}
			if j-i >= 2 {
				a, _ := st.render(it, st.cit, false)
				b, _ := st.render(items[j], st.cit, false)
				parts = append(parts, a+"–"+b)
				i = j
				continue
			}
		}
		s, au := st.render(it, st.cit, it.suppress)
		if strings.HasPrefix(collapse, "year") && !it.suppress && au != "" {
			// the same authors' works after it: their years only
			group := []string{join(it.prefix, s)}
			for i+1 < len(items) && items[i+1].e != nil && items[i+1].prefix == "" && it.after == "" {
				n := items[i+1]
				_, nau := st.render(n, st.cit, false)
				if nau != au {
					break
				}
				ns, _ := st.render(n, st.cit, true)
				group = append(group, ns)
				it = n
				i++
			}
			parts = append(parts, strings.Join(group, cmpOr(st.cit.attr["cite-group-delimiter"], ", "))+it.after)
			continue
		}
		parts = append(parts, join(it.prefix, s)+it.after)
	}
	c := &rctx{st: st, area: st.cit, it: &item{e: &Entry{}}}
	out := strings.Join(parts, delim)
	if allMissing(items) && lay.attr["prefix"] == "" {
		out = "(" + out + ")"
	} else {
		out = c.format(&node{name: "layout", attr: map[string]string{"prefix": lay.attr["prefix"], "suffix": lay.attr["suffix"],
			"font-style": lay.attr["font-style"], "font-weight": lay.attr["font-weight"]}}, out)
	}
	if author != "" {
		out = author + " " + out
	}
	return st.tidy(out)
}

func allMissing(items []*item) bool {
	for _, it := range items {
		if it.e != nil {
			return false
		}
	}
	return true
}

// disambiguate gives works whose citations read the same their year's
// letters (1999a, 1999b), in the bibliography's order, when the style
// asks for it.
func (st *Style) disambiguate(items []*item) {
	if st.cit.attr["disambiguate-add-year-suffix"] != "true" {
		return
	}
	groups := map[string][]*item{}
	var order []string
	for _, it := range items {
		probe := *it
		probe.locator, probe.label, probe.first, probe.suppress = "", "", true, false
		s, _ := st.render(&probe, st.cit, false)
		if _, ok := groups[s]; !ok {
			order = append(order, s)
		}
		groups[s] = append(groups[s], it)
	}
	for _, s := range order {
		if g := groups[s]; len(g) > 1 {
			for i, it := range g {
				it.suffix = letters(i)
			}
		}
	}
}

// hanging: the bibliography's entries hang (or have their number apart).
func (st *Style) hanging() bool {
	return st.bib != nil && (st.bib.attr["hanging-indent"] == "true" || st.bib.attr["second-field-align"] != "")
}

func mustRegexp(s string) *regexp.Regexp { return regexp.MustCompile(s) }

package cite

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/prose"
)

// Accolade's own citation style, for what it renders itself (the Live
// Preview, the page view, its PDF, Word and ODT; Pandoc's exports cite
// with --citeproc): one author-date style after Chicago's (17th ed.),
// as Zettlr's preview has it, and no CSL.
//
// In the text: [@doe99] is (Doe 1999), [@doe99, p. 12] (Doe 1999, 12),
// [see @a; @b] (see Doe 1999; Roe 2001), [-@doe99] (1999), @doe99 Doe
// (1999) and @doe99 [p. 12] Doe (1999, 12); two works by the same
// authors one after the other (Doe 1999, 2001); four authors or more
// Doe et al.; the same authors and year twice 1999a and 1999b; a key the
// library lacks (key?). The bibliography holds what is cited and the
// front matter's nocite (@* for the whole library), sorted by author and
// year, each entry a paragraph set with a hanging indent.

// EntryMark starts each bibliography entry's paragraph in the Markdown
// Render makes: prose sets such a paragraph with a hanging indent.
const EntryMark = prose.HangingMark

// Skipped: citations in a line of this kind are none (code, comments,
// the front matter).
func Skipped(k syntax.Kind) bool {
	switch k {
	case syntax.MdCode, syntax.MdCodeFence, syntax.MdFrontMatter, syntax.MdComment:
		return true
	}
	return false
}

// EachCitation calls fn with each citation of a Markdown text, in
// order, with its line's number: none in code, comments or the front
// matter.
func EachCitation(text string, fn func(line int, c syntax.Citation)) {
	if !strings.ContainsRune(text, '@') {
		return
	}
	lines := strings.Split(text, "\n")
	kinds := syntax.ClassifyMarkdown(lines)
	for i, l := range lines {
		if Skipped(kinds[i]) {
			continue
		}
		for _, c := range syntax.Citations(l) {
			fn(i, c)
		}
	}
}

// Render is body (a Markdown document's text, without its front
// matter) with its citations in Accolade's style and its bibliography
// added: in a ::: {#refs} div if it has one, else at the end of its
// References (Bibliography, Works Cited) section, else at its end under
// a heading of its own. nocite are the keys listed without being cited.
// Without a library, body is as written.
func Render(body string, lib *Library, nocite []string) string {
	if lib == nil {
		return body
	}
	lines := strings.Split(body, "\n")
	byLine := map[int][]syntax.Citation{}
	var works []*Entry
	seen := map[*Entry]bool{}
	add := func(e *Entry) {
		if e != nil && !seen[e] {
			seen[e] = true
			works = append(works, e)
		}
	}
	EachCitation(body, func(i int, c syntax.Citation) {
		byLine[i] = append(byLine[i], c)
		for _, k := range c.Keys {
			add(lib.Get(k.Key))
		}
	})
	for _, k := range nocite {
		if k == "*" {
			for _, e := range lib.Entries {
				add(e)
			}
			continue
		}
		add(lib.Get(k))
	}
	if len(byLine) == 0 && len(works) == 0 {
		return body
	}
	s := newStyler(lib, works)
	for i, cs := range byLine {
		lines[i] = s.replace(lines[i], cs)
	}
	if len(works) == 0 { // only keys the library lacks
		return strings.Join(lines, "\n")
	}
	return s.place(lines)
}

// work is a cited entry as the style shows it.
type work struct {
	e     *Entry
	label string // who in the text (Markdown): "Doe", "Doe and Roe", "Doe et al.", a short title
	plain string // the label as text, for telling works apart
	year  string // the year with its letter: 1999a; n.d.
}

type styler struct {
	lib   *Library
	works []*work // in the bibliography's order
	of    map[*Entry]*work
}

func newStyler(lib *Library, es []*Entry) *styler {
	s := &styler{lib: lib, of: map[*Entry]*work{}}
	for _, e := range es {
		w := &work{e: e, year: cmpOr(e.Year, "n.d.")}
		w.label, w.plain = label(e)
		s.works = append(s.works, w)
		s.of[e] = w
	}
	sort.SliceStable(s.works, func(i, j int) bool { return sortKey(s.works[i].e) < sortKey(s.works[j].e) })
	// the same authors in the same year: 1999a, 1999b, in the
	// bibliography's order
	groups := map[string][]*work{}
	for _, w := range s.works {
		k := fold(w.plain) + "|" + w.year
		groups[k] = append(groups[k], w)
	}
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		for i, w := range g {
			letter := letters(i)
			if w.year == "n.d." {
				w.year += "-" + letter
			} else {
				w.year += letter
			}
		}
	}
	return s
}

// letters is the i'th disambiguating letter: a … z, aa, ab …
func letters(i int) string {
	if i < 26 {
		return string(rune('a' + i))
	}
	return letters(i/26-1) + string(rune('a'+i%26))
}

func cmpOr(s, or string) string {
	if s == "" {
		return or
	}
	return s
}

// creators are who the work is by, "others" (BibTeX's and others) left
// out; etAl says it was there.
func creators(e *Entry) (ns []Name, etAl bool) {
	for _, n := range e.Creators() {
		if n.Literal == "others" {
			etAl = true
			continue
		}
		ns = append(ns, n)
	}
	return ns, etAl
}

// label is who a work is by in the text (as Markdown and as text): the
// family names (et al. from four on), else its title, shortened.
func label(e *Entry) (md, plain string) {
	ns, etAl := creators(e)
	switch {
	case len(ns) == 0:
		t := shortTitle(e.Title)
		if t == "" {
			return Escape(e.Key), e.Key
		}
		if italicTitle(e) {
			return "*" + Escape(t) + "*", t
		}
		return "“" + Escape(t) + "”", t
	case len(ns) >= 4 || etAl:
		plain = ns[0].FamilyName() + " et al."
	case len(ns) == 1:
		plain = ns[0].FamilyName()
	case len(ns) == 2:
		plain = ns[0].FamilyName() + " and " + ns[1].FamilyName()
	default:
		plain = ns[0].FamilyName() + ", " + ns[1].FamilyName() + ", and " + ns[2].FamilyName()
	}
	return Escape(plain), plain
}

// shortTitle is a title without its subtitle.
func shortTitle(t string) string {
	if i := strings.IndexAny(t, ":?!"); i > 0 {
		if t[i] != ':' {
			i++
		}
		t = t[:i]
	}
	return strings.TrimSpace(t)
}

// italicTitle: the work's title is in italics (a book, a report, a whole
// work), not in quotes (an article, a chapter, a page, a thesis).
func italicTitle(e *Entry) bool {
	switch e.Type {
	case "article", "article-journal", "article-magazine", "article-newspaper", "chapter", "paper-conference",
		"entry", "entry-dictionary", "entry-encyclopedia", "thesis", "webpage", "post", "post-weblog", "manuscript", "speech":
		return false
	}
	return true
}

// sortKey orders the bibliography: by the authors (family name first),
// else the title without its article; then the year, then the title.
func sortKey(e *Entry) string {
	ns, _ := creators(e)
	var who []string
	for _, n := range ns {
		who = append(who, fold(cmpOr(n.Family, n.Literal)+" "+n.Given+" "+n.Particle))
	}
	first := strings.Join(who, "\x01")
	title := fold(e.Title)
	for _, a := range []string{"the ", "a ", "an "} {
		title = strings.TrimPrefix(title, a)
	}
	if first == "" {
		first = title
	}
	return first + "\x00" + cmpOr(e.Year, "9999") + "\x00" + title
}

// Escape is text as Markdown that shows it as written.
func Escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune("\\`*_{}[]<>()#+!|~@$^", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ref is one work of a citation as written: what comes before it and
// after it, its locator, whether its author is left out.
type ref struct {
	w        *work  // nil: the library lacks it
	key      string // as written
	prefix   string // Markdown, as written
	locator  string // p. 12 as 12, chap. 2 as is
	suffix   string // what follows that is no locator
	suppress bool
}

// replace puts the citations cs of line in the style.
func (s *styler) replace(line string, cs []syntax.Citation) string {
	r := []rune(line)
	for i := len(cs) - 1; i >= 0; i-- { // from the end: the offsets before stay
		c := cs[i]
		out := s.citation(r, c)
		r = append(append(append([]rune(nil), r[:c.Start]...), []rune(out)...), r[c.End:]...)
	}
	return string(r)
}

// citation is one citation in the style.
func (s *styler) citation(r []rune, c syntax.Citation) string {
	if c.InText {
		k := c.Keys[0]
		ct := s.ref(k.Key)
		if c.End > k.End { // @doe99 [p. 12]
			loc := strings.TrimSpace(string(r[k.End:c.End]))
			loc = strings.TrimSuffix(strings.TrimPrefix(loc, "["), "]")
			ct.locator, ct.suffix = locator(loc)
		}
		if ct.w == nil {
			return "(" + Escape(ct.key) + "?)"
		}
		in := ct.w.year
		if ct.locator != "" {
			in += ", " + ct.locator
		}
		return ct.w.label + " (" + in + ct.suffix + ")"
	}
	// the parts between ;, a key in each (syntax.Citations found them so)
	var parts []ref
	start := c.Start + 1
	ki := 0
	for p := c.Start + 1; p <= c.End-1 && ki < len(c.Keys); p++ {
		if p < c.End-1 && r[p] != ';' {
			continue
		}
		k := c.Keys[ki]
		ki++
		ct := s.ref(k.Key)
		ct.suppress = k.Suppress
		pre := strings.TrimSpace(string(r[start:k.Start]))
		if k.Suppress {
			pre = strings.TrimSpace(strings.TrimSuffix(pre, "-"))
		}
		ct.prefix = pre
		after := strings.TrimSpace(string(r[k.End:p]))
		if rest, ok := strings.CutPrefix(after, ","); ok {
			ct.locator, ct.suffix = locator(rest)
		} else if after != "" {
			ct.suffix = " " + after
		}
		parts = append(parts, ct)
		start = p + 1
	}
	var items []string
	for i := 0; i < len(parts); i++ {
		ct := parts[i]
		if ct.w == nil {
			items = append(items, join(ct.prefix, Escape(ct.key)+"?"))
			continue
		}
		// the same authors' works one after the other: their years together
		years := []string{yearLoc(ct)}
		hasLoc := ct.locator != ""
		last := ct
		for i+1 < len(parts) {
			n := parts[i+1]
			if n.w == nil || n.prefix != "" || last.suffix != "" || n.suppress != ct.suppress || n.w.plain != ct.w.plain {
				break
			}
			years = append(years, yearLoc(n))
			hasLoc = hasLoc || n.locator != ""
			last = n
			i++
		}
		sep := ", "
		if hasLoc {
			sep = "; "
		}
		it := strings.Join(years, sep)
		if !ct.suppress {
			it = ct.w.label + " " + it
		}
		items = append(items, join(ct.prefix, it)+last.suffix)
	}
	return "(" + strings.Join(items, "; ") + ")"
}

func yearLoc(c ref) string {
	if c.locator != "" {
		return c.w.year + ", " + c.locator
	}
	return c.w.year
}

func join(prefix, s string) string {
	if prefix == "" {
		return s
	}
	return prefix + " " + s
}

func (s *styler) ref(key string) ref {
	c := ref{key: key}
	if e := s.lib.Get(key); e != nil {
		c.w = s.of[e]
	}
	return c
}

// pageLabel is a locator's page label, which the style leaves out.
var pageLabel = regexp.MustCompile(`^(?i)(pp?\.|pages?|pg\.)\s*`)

// locator reads what follows a key's comma: the locator ("p. 12" is 12,
// "chap. 2" as written, page ranges with a dash) and what follows it
// (", emphasis added"), both as written otherwise.
func locator(s string) (loc, suffix string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if i := strings.Index(s, ", "); i > 0 && !unicode.IsDigit(rune(s[i+2])) {
		s, suffix = s[:i], s[i:]
	}
	s = pageLabel.ReplaceAllString(s, "")
	return dashes(s), suffix
}

// dashes are a range's hyphens as an en dash: 12-15, 12--15 as 12–15.
func dashes(s string) string {
	return rangeDash.ReplaceAllString(s, "$1–$2")
}

var rangeDash = regexp.MustCompile(`(\d)\s*-{1,2}\s*(\d)`)

// place puts the bibliography in lines: in the {#refs} div, else at the
// end of the References section, else after the text.
func (s *styler) place(lines []string) string {
	kinds := syntax.ClassifyMarkdown(lines)
	refs := s.bibliography()
	// ::: {#refs} … :::
	for i, l := range lines {
		if kinds[i] == syntax.Action && refsDiv.MatchString(strings.TrimSpace(l)) {
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if t := strings.TrimSpace(lines[j]); strings.HasPrefix(t, ":::") && strings.Trim(t, ":") == "" {
					end = j
					break
				}
			}
			out := append(append([]string(nil), lines[:i]...), "")
			out = append(out, lines[i+1:end]...)
			out = append(out, "", refs, "")
			if end < len(lines) {
				out = append(out, lines[end+1:]...)
			}
			return strings.Join(out, "\n")
		}
	}
	// the References heading's section
	h1s, minLevel := 0, 0
	for i := range lines {
		lv, text := heading(lines, kinds, i)
		if lv == 0 {
			continue
		}
		if lv == 1 {
			h1s++
		}
		if minLevel == 0 || lv < minLevel {
			minLevel = lv
		}
		if !refsHeading(text) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l2, _ := heading(lines, kinds, j); l2 > 0 && l2 <= lv {
				end = j
				break
			}
		}
		out := append([]string(nil), lines[:end]...)
		out = append(out, "", refs, "")
		out = append(out, lines[end:]...)
		return strings.Join(out, "\n")
	}
	// after the text, under a heading at the level of its chapters (one
	// of level 1 alone is the title: under it, then)
	level := max(minLevel, 2)
	if h1s >= 2 {
		level = 1
	}
	text := strings.TrimRight(strings.Join(lines, "\n"), "\n ")
	return text + "\n\n" + strings.Repeat("#", level) + " References\n\n" + refs + "\n"
}

var refsDiv = regexp.MustCompile(`^:::+\s*\{[^}]*#refs\b[^}]*\}\s*:*$`)

// refsHeading: a heading that names the bibliography.
func refsHeading(t string) bool {
	switch fold(strings.TrimSpace(strings.TrimRight(t, ":"))) {
	case "references", "bibliography", "works cited", "literature", "literature cited", "sources", "reference list":
		return true
	}
	return false
}

// heading is the level and text of line i if it is a heading (ATX, or
// a setext one's text line), else 0.
func heading(lines []string, kinds []syntax.Kind, i int) (int, string) {
	if kinds[i] != syntax.MdHeading {
		return 0, ""
	}
	t := strings.TrimSpace(lines[i])
	if strings.HasPrefix(t, "#") {
		n := len(t) - len(strings.TrimLeft(t, "#"))
		return n, strings.TrimSpace(strings.TrimRight(t[n:], "# "))
	}
	if i+1 < len(lines) && kinds[i+1] == syntax.MdHeading {
		u := strings.TrimSpace(lines[i+1])
		if u != "" && strings.Trim(u, "=") == "" {
			return 1, t
		}
		if u != "" && strings.Trim(u, "-") == "" {
			return 2, t
		}
	}
	return 0, ""
}

// bibliography is the entries, as Markdown paragraphs each starting with
// EntryMark.
func (s *styler) bibliography() string {
	var out []string
	for _, w := range s.works {
		out = append(out, EntryMark+s.entry(w))
	}
	return strings.Join(out, "\n\n")
}

// entry is a work in the bibliography, as Markdown.
func (s *styler) entry(w *work) string {
	e := w.e
	var b strings.Builder
	part := func(md string) { // a part of the entry, ending in a period
		md = strings.TrimSpace(md)
		if md == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(md)
		if !endsSentence(md) {
			b.WriteByte('.')
		}
	}
	quoted := func(t string) string { // “A Title.”: the period inside
		t = strings.TrimSpace(t)
		if t == "" {
			return ""
		}
		if endsSentence(t) {
			return "“" + Escape(t) + "”"
		}
		return "“" + Escape(t) + ".”"
	}
	italic := func(t string) string {
		if t = strings.TrimSpace(t); t == "" {
			return ""
		}
		return "*" + Escape(t) + "*"
	}
	ns, etAl := creators(e)
	byEditors := len(e.Authors) == 0 && len(ns) > 0
	title := quoted(e.Title)
	if italicTitle(e) {
		title = italic(e.Title)
	}
	if len(ns) == 0 { // the title in the author's place
		part(title)
		part(w.year)
		title = ""
	} else {
		who := Escape(nameList(ns, etAl, true))
		if byEditors {
			if len(ns) > 1 {
				who += ", eds"
			} else {
				who += ", ed"
			}
		}
		part(who)
		part(w.year)
	}
	if title != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(title)
		if !strings.HasSuffix(title, "”") && !endsSentence(e.Title) {
			b.WriteByte('.')
		}
	}
	pubPlace := func() string {
		pub := Escape(e.Publisher)
		if e.Place != "" && pub != "" {
			return Escape(e.Place) + ": " + pub
		}
		return cmpOr(pub, Escape(e.Place))
	}
	editedBy := ""
	if len(e.Editors) > 0 && !byEditors {
		editedBy = "edited by " + Escape(nameList(e.Editors, false, false))
	}
	pages := dashes(e.Pages)
	switch e.Type {
	case "article", "article-journal", "article-magazine", "article-newspaper":
		j := italic(e.Container)
		if e.Volume != "" {
			j += " " + Escape(e.Volume)
		}
		if e.Issue != "" {
			j += " (" + Escape(e.Issue) + ")"
		}
		if pages != "" {
			if e.Volume != "" || e.Issue != "" {
				j += ": " + Escape(pages)
			} else {
				j += ", " + Escape(pages)
			}
		}
		part(j)
	case "chapter", "paper-conference", "entry", "entry-dictionary", "entry-encyclopedia":
		in := ""
		if e.Container != "" {
			in = "In " + italic(e.Container)
			if editedBy != "" {
				in += ", " + editedBy
			}
			if pages != "" {
				in += ", " + Escape(pages)
			}
		}
		part(in)
		part(edition(e.Edition))
		part(pubPlace())
	case "thesis":
		genre := cmpOr(e.Genre, "PhD diss.")
		if e.Publisher != "" {
			genre += ", " + e.Publisher
		}
		part(Escape(genre))
	case "webpage", "post", "post-weblog":
		part(Escape(e.Container)) // a site's name: not in italics
		part(Escape(e.Publisher))
	case "report":
		if e.Genre != "" || e.Issue != "" {
			part(Escape(strings.TrimSpace(cmpOr(e.Genre, "Report") + " " + e.Issue)))
		}
		part(pubPlace())
	default: // book and the rest
		if e.Container != "" {
			part(italic(e.Container))
		}
		part(edition(e.Edition))
		if editedBy != "" {
			part(strings.ToUpper(editedBy[:1]) + editedBy[1:])
		}
		if e.Type == "manuscript" && e.Genre != "" {
			part(Escape(e.Genre))
		}
		part(pubPlace())
	}
	switch {
	case e.DOI != "":
		part("<https://doi.org/" + strings.TrimPrefix(strings.TrimPrefix(e.DOI, "https://doi.org/"), "doi:") + ">")
	case e.URL != "":
		part("<" + e.URL + ">")
	}
	return b.String()
}

// edition is "2" as "2nd ed.", "Revised" as "Revised ed.".
func edition(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "1" {
		return ""
	}
	if n, err := strconv.Atoi(s); err == nil {
		suffix := "th"
		switch {
		case n%100 >= 11 && n%100 <= 13:
		case n%10 == 1:
			suffix = "st"
		case n%10 == 2:
			suffix = "nd"
		case n%10 == 3:
			suffix = "rd"
		}
		return strconv.Itoa(n) + suffix + " ed."
	}
	if strings.Contains(strings.ToLower(s), "ed") {
		return Escape(s)
	}
	return Escape(s) + " ed."
}

// endsSentence: s ends in a period, ? or ! (quotes and brackets after
// them looked through).
func endsSentence(s string) bool {
	s = strings.TrimRight(s, "”’\"')*>")
	return strings.HasSuffix(s, ".") || strings.HasSuffix(s, "?") || strings.HasSuffix(s, "!")
}

// invertedName is a name family first, its particle after the given
// names, as Chicago's lists sort and show it: "Goethe, Johann Wolfgang
// von" (the text keeps "von Goethe").
func invertedName(n Name) string {
	if n.Literal != "" || n.Particle == "" {
		return n.Inverted()
	}
	s := n.Family
	if g := strings.TrimSpace(n.Given + " " + n.Particle); g != "" {
		s += ", " + g
	}
	if n.Suffix != "" {
		s += ", " + n.Suffix
	}
	return s
}

// nameList is the names as the bibliography lists them: the first
// family name first when inverted ("Doe, Jane, John Roe, and Ann
// Poe"); seven and et al. for more than ten.
func nameList(ns []Name, etAl, inverted bool) string {
	if len(ns) > 10 {
		ns, etAl = ns[:7], true
	}
	var parts []string
	for i, n := range ns {
		if i == 0 && inverted {
			parts = append(parts, invertedName(n))
		} else {
			parts = append(parts, n.String())
		}
	}
	switch {
	case etAl:
		return strings.Join(parts, ", ") + ", et al."
	case len(parts) == 1:
		return parts[0]
	case len(parts) == 2 && !inverted:
		return parts[0] + " and " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}

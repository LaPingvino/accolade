// Package help is Accolade's help: pages written once as Markdown in this
// folder, embedded in the app (the Help tab beside Notes, Source and
// Production) and published as web pages with the browser version
// (cmd/helpsite, /help/<page>.html). It is pure Go, without Fyne, so the
// web pages are made where no graphics libraries are installed.
//
// Some pages are made from the code, so they never drift from it: the
// keyboard shortcuts from the command registry (written into
// shortcuts.md and cheat-shortcuts.md by the app's tests, made live in
// the app for the document's mode), Lexington's flags (lexington.md and
// cheat-lexington.md, by this package's tests) and the revision colours,
// breakdown categories and Day Out of Days codes (generated.go, from
// Lexington's own tables).
package help

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
)

//go:embed *.md
var files embed.FS

// Site is where the help is online: the browser version's /help/.
const Site = "https://lapingvino.github.io/accolade/help/"

// Issues is where ideas and bugs go (Help > Send Feedback…).
const Issues = "https://github.com/LaPingvino/accolade/issues/new"

// Page is a help page or a cheat sheet.
type Page struct {
	Slug     string // its file's name without .md, and its web page's
	Title    string
	Cheat    bool // a cheat sheet: one screen, printable as a card
	Markdown string
}

// The pages in the order the Help menu and the contents list them.
var pageSlugs = []string{"index", "fountain", "final-draft", "adapting", "production", "shortcuts", "formats", "settings", "lexington", "vim"}

// The cheat sheets, in a screenplay's order (CheatsFor has a Markdown
// document's).
var cheatSlugs = []string{"cheat-shortcuts", "cheat-fountain", "cheat-blocks", "cheat-revisions", "cheat-breakdown",
	"cheat-dood", "cheat-markdown", "cheat-lexington"}

// generated are the sheets made here from Lexington's tables.
var generated = map[string]func() string{
	"cheat-revisions": revisionsSheet,
	"cheat-breakdown": breakdownSheet,
	"cheat-dood":      doodSheet,
	"settings":        settingsPage,
}

// Get is the page with slug.
func Get(slug string) (Page, bool) {
	md := ""
	if g, ok := generated[slug]; ok {
		md = g()
	} else {
		b, err := files.ReadFile(slug + ".md")
		if err != nil {
			return Page{}, false
		}
		md = string(b)
	}
	return Page{Slug: slug, Title: titleOf(md, slug), Cheat: strings.HasPrefix(slug, "cheat-"), Markdown: md}, true
}

// must is the page with slug; the catalogue is checked by the tests.
func must(slug string) Page {
	p, ok := Get(slug)
	if !ok {
		panic("help: no page " + slug)
	}
	return p
}

// Pages are the full pages, in order.
func Pages() []Page {
	out := make([]Page, len(pageSlugs))
	for i, s := range pageSlugs {
		out[i] = must(s)
	}
	return out
}

// Cheats are the cheat sheets, a screenplay's order.
func Cheats() []Page { return CheatsFor(false) }

// CheatsFor are the cheat sheets in the order for the document's mode:
// a Markdown document's Markdown sheet first.
func CheatsFor(markdown bool) []Page {
	order := cheatSlugs
	if markdown {
		order = []string{"cheat-markdown", "cheat-shortcuts", "cheat-blocks", "cheat-fountain", "cheat-revisions",
			"cheat-breakdown", "cheat-dood", "cheat-lexington"}
	}
	out := make([]Page, len(order))
	for i, s := range order {
		out[i] = must(s)
	}
	return out
}

// All are the pages and then the cheat sheets.
func All() []Page { return append(Pages(), Cheats()...) }

// titleOf is a page's first heading, else its slug.
func titleOf(md, slug string) string {
	for _, l := range strings.Split(md, "\n") {
		if strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(l[2:])
		}
	}
	return slug
}

// Anchor is a heading's anchor, the same in the app and online: lower
// case, spaces as "-", only letters, digits and "-" kept.
func Anchor(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Section is a page's part under one heading (the part before the first
// heading has none).
type Section struct {
	Heading  string // as written, without the #s
	Level    int    // 0 before the first heading
	Anchor   string
	Markdown string // the heading line and what follows it
}

// Sections splits md at its headings (not inside fenced code).
func Sections(md string) []Section {
	var out []Section
	cur := Section{}
	var body []string
	fenced := false
	flush := func() {
		cur.Markdown = strings.Trim(strings.Join(body, "\n"), "\n")
		if cur.Markdown != "" || cur.Level > 0 {
			out = append(out, cur)
		}
	}
	for _, l := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
		}
		if level := headingLevel(l); level > 0 && !fenced {
			flush()
			h := plainText(strings.TrimSpace(l[level:]))
			cur, body = Section{Heading: h, Level: level, Anchor: Anchor(h)}, nil
		}
		body = append(body, l)
	}
	flush()
	return out
}

// headingLevel is the level of an ATX heading line, 0 for another line.
func headingLevel(l string) int {
	n := 0
	for n < len(l) && l[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n >= len(l) || l[n] != ' ' {
		return 0
	}
	return n
}

var emphasisMarks = strings.NewReplacer("**", "", "`", "", "__", "")

// plainText is a heading's text without its emphasis marks.
func plainText(s string) string { return emphasisMarks.Replace(s) }

var linkRE = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// codeSpanRE is inline code, whose brackets are no links.
var codeSpanRE = regexp.MustCompile("`[^`]*`")

// Links are the targets of md's links, in order.
func Links(md string) []string {
	var out []string
	inFence := false
	for _, l := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range linkRE.FindAllStringSubmatch(codeSpanRE.ReplaceAllString(l, ""), -1) {
			out = append(out, m[1])
		}
	}
	return out
}

// Target is where a link goes: a page (and a heading on it) here, or a
// web address.
type Target struct {
	Slug, Anchor string
	URL          string // a web address (Slug "")
}

// Resolve reads a link's target written on page from: "page.md#anchor",
// "#anchor" (on from), or a web address.
func Resolve(from, link string) Target {
	if u, err := url.Parse(link); err == nil && u.Scheme != "" {
		return Target{URL: link}
	}
	path, anchor, _ := strings.Cut(link, "#")
	if path == "" {
		return Target{Slug: from, Anchor: anchor}
	}
	return Target{Slug: strings.TrimSuffix(strings.TrimSuffix(path, ".md"), ".html"), Anchor: anchor}
}

// Hit is a search result: a section that has every word asked for.
type Hit struct {
	Page    Page
	Section Section
	Snippet string
}

// Search finds words in pages (case aside): the sections having all of
// them, a page's own sections first in the pages' order.
func Search(pages []Page, query string) []Hit {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}
	var out []Hit
	for _, p := range pages {
		for _, s := range Sections(p.Markdown) {
			low := strings.ToLower(s.Markdown)
			all := true
			for _, w := range words {
				if !strings.Contains(low, w) {
					all = false
					break
				}
			}
			if all {
				out = append(out, Hit{Page: p, Section: s, Snippet: snippet(s.Markdown, words[0])})
			}
		}
	}
	return out
}

// snippet is the line of md with word, shortened around it.
func snippet(md, word string) string {
	for _, l := range strings.Split(md, "\n") {
		if headingLevel(l) > 0 {
			continue
		}
		low := strings.ToLower(l)
		i := strings.Index(low, word)
		if i < 0 {
			continue
		}
		l = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "-*>"))
		r := []rune(l)
		at := len([]rune(low[:i]))
		from := max(0, at-30)
		to := min(len(r), at+60)
		s := string(r[from:to])
		if from > 0 {
			s = "…" + s
		}
		if to < len(r) {
			s += "…"
		}
		return plainText(s)
	}
	return ""
}

// URL is page slug's web address, with the version of the app asking (so
// the page can say when it is for another one), and a heading's anchor.
func URL(slug, version, anchor string) string {
	u := Site + slug + ".html"
	if version != "" {
		u += "?v=" + url.QueryEscape(version)
	}
	if anchor != "" {
		u += "#" + anchor
	}
	return u
}

// FeedbackURL is a new GitHub issue with what helps to answer it filled
// in (the version, where Accolade runs, the document's mode); nothing of
// the script goes in, and nothing is sent until the writer sends it.
func FeedbackURL(version, platform, mode string) string {
	body := fmt.Sprintf("**What happened, or what would you like?**\n\n\n\n"+
		"**What did you expect?**\n\n\n\n---\nAccolade %s · %s · %s document\n", version, platform, mode)
	return Issues + "?" + url.Values{"title": {""}, "body": {body}}.Encode()
}

// ids gives goldmark's headings the anchors Anchor gives.
type ids struct{ seen map[string]int }

func (i *ids) Generate(value []byte, _ ast.NodeKind) []byte {
	a := Anchor(plainText(string(value)))
	if n := i.seen[a]; n > 0 {
		i.seen[a] = n + 1
		return []byte(fmt.Sprintf("%s-%d", a, n))
	}
	i.seen[a] = 1
	return []byte(a)
}

func (i *ids) Put(value []byte) { i.seen[string(value)]++ }

var md = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(parser.WithAutoHeadingID()))

var mdLinkRE = regexp.MustCompile(`\]\(([a-z0-9-]+)\.md(#[^)]*)?\)`)

// Body is a page's Markdown as HTML, its links to pages here going to
// their web pages.
func Body(p Page) (string, error) {
	src := mdLinkRE.ReplaceAllString(p.Markdown, "]($1.html$2)")
	var b bytes.Buffer
	ctx := parser.NewContext(parser.WithIDs(&ids{seen: map[string]int{}}))
	doc := md.Parser().Parse(gmtext.NewReader([]byte(src)), parser.WithContext(ctx))
	if err := md.Renderer().Render(&b, []byte(src), doc); err != nil {
		return "", err
	}
	return swatches(b.String()), nil
}

var swatchRE = regexp.MustCompile(`<code>(#[0-9A-Fa-f]{6})</code>`)

// swatches puts a square of a colour before its code (#0000FF): the
// breakdown categories' colours.
func swatches(s string) string {
	return swatchRE.ReplaceAllString(s, `<span class="swatch" style="background:$1"></span><code>$1</code>`)
}

// HTML is a page as a web page: online with the contents beside it and
// the version it is for; as a card (card), a cheat sheet to print on one
// page.
func HTML(p Page, version string, card bool) ([]byte, error) {
	body, err := Body(p)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	esc := html.EscapeString
	fmt.Fprintf(&b, "<!doctype html>\n<html lang=\"en\"><head><meta charset=\"utf-8\">\n"+
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n"+
		"<title>%s · Accolade Help</title>\n<style>%s</style>\n</head>\n", esc(p.Title), style)
	if card {
		fmt.Fprintf(&b, "<body class=\"card\">\n<main>\n%s</main>\n<footer>Accolade %s · %s</footer>\n</body></html>\n",
			body, esc(version), esc(URL(p.Slug, "", "")))
		return b.Bytes(), nil
	}
	b.WriteString("<body>\n<nav><a class=\"home\" href=\"index.html\">Accolade Help</a>\n<ul>")
	for _, q := range Pages()[1:] {
		fmt.Fprintf(&b, "<li><a href=\"%s.html\"%s>%s</a></li>", q.Slug, current(p, q), esc(q.Title))
	}
	b.WriteString("</ul>\n<p class=\"group\">Cheat sheets</p><ul>")
	for _, q := range Cheats() {
		fmt.Fprintf(&b, "<li><a href=\"%s.html\"%s>%s</a></li>", q.Slug, current(p, q), esc(q.Title))
	}
	fmt.Fprintf(&b, "</ul>\n<p><a href=\"../\">Open Accolade in the browser</a></p></nav>\n<main>\n"+
		"<p class=\"version\" id=\"version\" data-version=\"%s\">This is the help for Accolade %s.</p>\n%s</main>\n"+
		"<script>%s</script>\n</body></html>\n", esc(version), esc(version), body, versionScript)
	return b.Bytes(), nil
}

func current(p, q Page) string {
	if p.Slug == q.Slug {
		return ` aria-current="page"`
	}
	return ""
}

// versionScript says so when the app that opened the page (?v=) is
// another version than the page's.
const versionScript = `(function(){var v=new URLSearchParams(location.search).get("v");` +
	`var e=document.getElementById("version");if(!v||!e)return;var p=e.getAttribute("data-version");` +
	`if(v!==p){e.className+=" other";e.textContent="This is the help for Accolade "+p+"; yours is "+v+": some things may differ.";}})();`

const style = `:root{color-scheme:light dark;--fg:#1d1d1f;--bg:#fff;--muted:#666;--line:#ddd;--accent:#2a5db0}
@media (prefers-color-scheme:dark){:root{--fg:#e8e8e8;--bg:#1b1b1d;--muted:#aaa;--line:#444;--accent:#8ab4f8}}
body{margin:0;background:var(--bg);color:var(--fg);font:16px/1.55 system-ui,sans-serif;display:flex;flex-wrap:wrap}
nav{flex:0 0 15em;padding:1em;border-right:1px solid var(--line);font-size:.95em}
nav ul{list-style:none;padding:0;margin:.3em 0 1em}nav li{margin:.2em 0}
nav a[aria-current]{font-weight:bold}.home{font-weight:bold;font-size:1.1em}.group{color:var(--muted);margin:0}
main{flex:1 1 30em;max-width:46em;padding:1em 1.5em 3em;min-width:0}
a{color:var(--accent)}code,pre{font-family:ui-monospace,monospace;font-size:.92em}
pre{background:color-mix(in srgb,var(--fg) 6%,transparent);padding:.6em .8em;overflow-x:auto;border-radius:4px}
table{border-collapse:collapse}td,th{border:1px solid var(--line);padding:.2em .5em;text-align:left}
.version{color:var(--muted);font-size:.9em}.version.other{color:#b5651d}
.swatch{display:inline-block;width:.9em;height:.9em;border:1px solid var(--line);vertical-align:-.1em;margin-right:.3em}
@media (max-width:40em){nav{flex-basis:100%;border-right:0;border-bottom:1px solid var(--line)}nav ul{display:flex;flex-wrap:wrap;gap:.2em 1em}}
body.card{display:block;font-size:10pt;line-height:1.35}
.card main{max-width:none;padding:0;columns:2;column-gap:2em}
.card h1{column-span:all;font-size:15pt;margin:0 0 .4em}.card h2{font-size:11pt;margin:.6em 0 .2em}
.card ul{padding-left:1.1em;margin:.2em 0}.card p{margin:.3em 0}.card li,.card pre{break-inside:avoid}
.card footer{color:var(--muted);font-size:8pt;margin-top:1em}
@page{size:auto;margin:1.2cm}
@media print{:root{--fg:#000;--bg:#fff}nav,.version{display:none}main{max-width:none}}`

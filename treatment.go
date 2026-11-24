package main

import (
	"regexp"
	"strings"

	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/opml"
)

// A treatment (Export › Treatment, PDF or Word): the script's structure
// as a document to read and pass round, made from what the outline
// reads (outlineDoc): the sections as headings, the scenes' headings
// under them, each one's synopsis, the beats as paragraphs and the notes
// in italics; the scenes' own text left out. It is written as Markdown
// and printed by Accolade's own document export (internal/prose: the
// PDF through Lexington's drawing, Word), as a Markdown document is.

// treatmentMarkdown is the script's treatment as Markdown, its title and
// its title page's lines.
func treatmentMarkdown(text string, starts []string) (md, title string, titlePage []string) {
	d := outlineDoc(text, starts)
	title = d.Title
	var paras []string
	para := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			paras = append(paras, s)
		}
	}
	// own is a heading's note: its synopsis as a paragraph, its notes in
	// italics (a colour's note is the heading's colour: left out)
	own := func(note string) {
		var syn []string
		for _, l := range strings.Split(note, "\n") {
			t := strings.TrimSpace(l)
			switch {
			case t == "":
			case isNoteLine(t):
				para(strings.Join(syn, " "))
				syn = nil
				if n := treatmentNote(t); n != "" {
					para("*" + treatmentEscape(n) + "*")
				}
			default:
				syn = append(syn, treatmentEscape(lex.StripNotes(t)))
			}
		}
		para(strings.Join(syn, " "))
	}
	var walk func(ns []*opml.Node, level int)
	walk = func(ns []*opml.Node, level int) {
		for _, n := range ns {
			switch n.Kind {
			case opml.KindTitlePage:
				titlePage = treatmentTitlePage(strings.Split(n.Note, "\n"))
			case opml.KindSection:
				l := level + 1
				if n.Level > 0 {
					l = n.Level
				}
				para(strings.Repeat("#", min(l, 6)) + " " + treatmentEscape(lex.SectionText(n.Text)))
				own(n.Note)
				walk(n.Kids, l)
			case opml.KindScene:
				h, _ := lex.SceneNumber(strings.TrimPrefix(lex.StripNotes(n.Text), "."))
				para(strings.Repeat("#", min(level+1, 6)) + " " + treatmentEscape(strings.ToUpper(strings.TrimSpace(h))))
				own(n.Note)
				walk(n.Kids, level)
			case opml.KindNote:
				if t := treatmentNote(n.Text + " " + n.Note); t != "" {
					para("*" + treatmentEscape(t) + "*")
				}
			default: // a beat: its lines one paragraph, a note touching it in italics
				own(n.Text + "\n" + n.Note)
			}
		}
	}
	walk(d.Nodes, 0)
	if title == "" {
		title = "Treatment"
	}
	if len(titlePage) == 0 {
		titlePage = []string{title, "", "A treatment"}
	}
	return strings.Join(paras, "\n\n") + "\n", title, titlePage
}

// treatmentNote is a note's words, [[ ]] taken off ("" for a colour's).
func treatmentNote(s string) string {
	t := strings.TrimSpace(s)
	t = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "[["), "]]"))
	t = strings.Join(strings.Fields(strings.ReplaceAll(t, "]] [[", " ")), " ")
	if _, colour := noteRGB(t); colour {
		return ""
	}
	return t
}

// treatmentTitlePage is a treatment's title page from the script's: its
// title, "A treatment", and who wrote it.
func treatmentTitlePage(lines []string) []string {
	fields := map[string]string{}
	key := ""
	for _, l := range lines {
		k, v, ok := strings.Cut(l, ":")
		if ok && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") {
			key = strings.ToLower(strings.TrimSpace(k))
			fields[key] = strings.TrimSpace(v)
			continue
		}
		if key != "" && strings.TrimSpace(l) != "" {
			fields[key] = strings.TrimSpace(fields[key] + " " + strings.TrimSpace(l))
		}
	}
	out := []string{lex.Unstyled(fields["title"]), "", "A treatment"}
	by := strings.TrimSpace(fields["credit"] + " " + fields["author"])
	if fields["author"] == "" {
		by = fields["authors"]
		if by != "" {
			by = strings.TrimSpace(fields["credit"] + " " + by)
		}
	}
	if by != "" {
		out = append(out, "", lex.Unstyled(by))
	}
	return out
}

// mdSpecial are the characters Markdown reads in a line that a
// treatment's text has as they are (Fountain's * and _ for emphasis are
// Markdown's too, and kept).
var mdSpecial = strings.NewReplacer(`\`, `\\`, "`", "\\`", "[", `\[`, "]", `\]`, "<", `\<`, ">", `\>`, "#", `\#`, "|", `\|`, "~", `\~`)

// mdStart is what makes a line a list item or a quote at its start.
var mdStart = regexp.MustCompile(`^([-+]|\d+[.)])(\s)`)

// treatmentEscape is s as Markdown text.
func treatmentEscape(s string) string {
	s = mdSpecial.Replace(s)
	return mdStart.ReplaceAllStringFunc(s, func(m string) string { return `\` + m })
}

// exportTreatment writes the script's treatment to out as format id
// ("pdf", "docx", "odt") on paper ("" the default).
func exportTreatment(text string, starts []string, out, id, paper string) error {
	md, title, tp := treatmentMarkdown(text, starts)
	return exportMarkdownWith(md, title, out, id, mdPageSetup{paper: paper, titlePage: true}, tp)
}

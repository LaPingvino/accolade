package main

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/layout"
	"github.com/LaPingvino/lexington/lex"
	"github.com/LaPingvino/lexington/rules"
)

// View > Statistics: the script as Lexington prints it (in the chosen
// script format), counted: its length, its scenes, how long it plays
// (a page a minute, the screenwriters' rule), and who speaks how much.

// role is a character's part in the script.
type role struct {
	Name     string
	Speeches int
	Words    int
}

// details are the script's statistics.
type details struct {
	Words, Characters int
	Pages             int
	Lines             int // printed lines, without the title page
	ActionLines       int
	DialogueLines     int // names, parentheticals and dialogue
	Scenes            int
	Interior          int
	Exterior          int
	Tags              []tag  // scenes by what their headings start with, most first
	Roles             []role // most speeches first
}

// tag is a scene heading start (INT, EXT, BIN, ИНТ, ...) and how many
// scenes have it.
type tag struct {
	Tag    string
	Scenes int
}

// linesAPage is a screenplay page in Courier 12: 55 lines.
const linesAPage = 55

// insideOutside reports whether a scene heading is inside, outside or
// both, in the language of scenes (its heading starts): the first one
// with a slash pairs them (INT./EXT, BIN./BUI, EN/EKST, ИНТ/НАТ), and a
// heading starting with a slash is both.
func insideOutside(heading string, scenes []string) (in, out bool) {
	word, _, _ := strings.Cut(strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(heading)), "."), " ")
	if strings.Contains(word, "/") {
		return true, true
	}
	clean := func(s string) string { return strings.Trim(s, ". ") }
	word = clean(word)
	for _, s := range scenes {
		if i, o, ok := strings.Cut(strings.ToUpper(s), "/"); ok {
			like := func(half string) bool {
				half = clean(half)
				return half != "" && word != "" && (strings.HasPrefix(word, half) || strings.HasPrefix(half, word))
			}
			return like(i), like(o)
		}
	}
	return false, false
}

// headingTag is the start of a scene heading as written in the script's
// language (scenes: its starts), without its dot; "" for a heading
// without one (forced with a dot).
func headingTag(heading string, scenes []string) string {
	h := strings.ToUpper(strings.TrimSpace(heading))
	best := ""
	for _, s := range scenes {
		t := strings.TrimRight(strings.ToUpper(s), ".")
		if len(t) > len(best) && strings.HasPrefix(h, t) && (len(h) == len(t) || strings.ContainsRune(". -", rune(h[len(t)]))) {
			best = t
		}
	}
	return best
}

func detailsOf(text string, scenes []string, elements rules.Set) details {
	var d details
	tags := map[string]int{}
	s := fountain.Parse(scenes, strings.NewReader(text))
	roles := map[string]*role{}
	var speaker *role
	inTitle := false
	for _, l := range s {
		switch l.Type {
		case lex.TypeTitlePage:
			inTitle = true
		case lex.TypeNewPage:
			inTitle = false
		case lex.TypeScene:
			d.Scenes++
			in, out := insideOutside(l.Contents, scenes)
			if in {
				d.Interior++
			}
			if out {
				d.Exterior++
			}
			if t := headingTag(l.Contents, scenes); t != "" {
				tags[t]++
			}
		case lex.TypeSpeaker:
			name := strings.TrimSpace(l.Contents)
			if i := strings.Index(name, "("); i > 0 {
				name = strings.TrimSpace(name[:i])
			}
			if roles[name] == nil {
				roles[name] = &role{Name: name}
			}
			speaker = roles[name]
			speaker.Speeches++
		case lex.TypeDialog:
			if speaker != nil {
				speaker.Words += len(strings.Fields(l.Contents))
			}
		case lex.TypeEmpty, lex.TypeParen:
		default:
			speaker = nil
		}
		if !inTitle && l.Type != fountain.TypeBoneyard {
			d.Words += len(strings.Fields(l.Contents))
			d.Characters += len([]rune(strings.TrimSpace(l.Contents)))
		}
	}
	// the printed lines, after the title page
	lines, page := layout.Lay(s, elements), 0
	afterTitle := false
	for _, l := range lines {
		if l.PageBreak {
			if afterTitle || !hasTitlePage(s) {
				d.Pages++
				page = 0
			}
			afterTitle = true
			continue
		}
		if l.Block != "" { // the title page
			continue
		}
		page++
		d.Lines++
		switch l.Type {
		case lex.TypeSpeaker, lex.TypeDialog, lex.TypeParen:
			d.DialogueLines++
		case lex.TypeAction:
			d.ActionLines++
		}
		if page >= linesAPage {
			d.Pages++
			page = 0
		}
	}
	if page > 0 {
		d.Pages++
	}
	for _, r := range roles {
		d.Roles = append(d.Roles, *r)
	}
	sort.Slice(d.Roles, func(i, j int) bool {
		if d.Roles[i].Speeches != d.Roles[j].Speeches {
			return d.Roles[i].Speeches > d.Roles[j].Speeches
		}
		return d.Roles[i].Name < d.Roles[j].Name
	})
	for t, n := range tags {
		d.Tags = append(d.Tags, tag{t, n})
	}
	sort.Slice(d.Tags, func(i, j int) bool {
		if d.Tags[i].Scenes != d.Tags[j].Scenes {
			return d.Tags[i].Scenes > d.Tags[j].Scenes
		}
		return d.Tags[i].Tag < d.Tags[j].Tag
	})
	return d
}

func hasTitlePage(s lex.Screenplay) bool {
	return len(s) > 0 && s[0].Type == lex.TypeTitlePage
}

// minutes is how long lines play: a page (55 lines) a minute.
func minutes(lines int) string {
	secs := lines * 60 / linesAPage
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// showStatistics shows the script's statistics.
func (w *MainWindow) showStatistics() {
	d := detailsOf(w.textEditor.Text(), sceneHeaders(w.language), currentScriptFormat())
	row := func(label, value string) fyne.CanvasObject {
		v := widget.NewLabel(value)
		v.Alignment = fyne.TextAlignTrailing
		return container.NewBorder(nil, nil, widget.NewLabel(label), v)
	}
	length := widget.NewCard("Length", "", container.NewVBox(
		row("Pages", fmt.Sprint(d.Pages)),
		row("Words", fmt.Sprint(d.Words)),
		row("Characters", fmt.Sprint(d.Characters)),
	))
	duration := widget.NewCard("Duration", "A page a minute", container.NewVBox(
		row("In all (with headings)", minutes(d.Lines)),
		row("Action", minutes(d.ActionLines)),
		row("Dialogue", minutes(d.DialogueLines)),
	))
	scenes := widget.NewCard("Scenes", "", container.NewVBox(
		row("Scenes", fmt.Sprint(d.Scenes)),
		row("Interior", fmt.Sprint(d.Interior)),
		row("Exterior", fmt.Sprint(d.Exterior)),
	))
	if len(d.Tags) > 0 { // each heading start as written, folded away
		byTag := container.NewVBox()
		for _, t := range d.Tags {
			byTag.Add(row(t.Tag, fmt.Sprint(t.Scenes)))
		}
		scenes.Content.(*fyne.Container).Add(widget.NewAccordion(widget.NewAccordionItem("By heading", byTag)))
	}
	cast := container.NewVBox()
	for _, r := range d.Roles {
		speeches := fmt.Sprintf("%d speeches", r.Speeches)
		if r.Speeches == 1 {
			speeches = "1 speech"
		}
		cast.Add(row(r.Name, speeches+", "+plural(r.Words, "word")))
	}
	if len(d.Roles) == 0 {
		cast.Add(widget.NewLabel("No one speaks yet."))
	}
	roles := widget.NewCard("Characters", "", cast)
	content := container.NewVScroll(container.NewVBox(
		container.NewGridWithColumns(3, length, duration, scenes), roles))
	content.SetMinSize(fyne.NewSize(620, 460))
	dialog.ShowCustom("Statistics", "Close", content, w.fyneWindow)
}

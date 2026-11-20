package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/prose"
	"github.com/LaPingvino/accolade/internal/storyread"
)

// What the source shows (docs/ADAPTATION.md, step 5): the picked
// passage as the story reader (internal/storyread) reads it, read-only
// at the bottom of the source pane: who says what, the scene it is in
// and the heading that suggests, what a camera cannot see, the names in
// it. Suggestions in the editor build on this later (step 6).

// readingHeight is how tall the view is at most, before it scrolls.
const readingHeight = 190

// readingView is the view at the bottom of the source pane.
type readingView struct {
	title  *widget.Label
	close  *widget.Button
	text   *widget.RichText
	box    *fyne.Container
	closed bool // closed until another passage is picked
}

func newReadingView(s *sourcePane) *readingView {
	v := &readingView{}
	v.title = widget.NewLabelWithStyle("What the source shows", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.title.SizeName = theme.SizeNameCaptionText
	v.close = widget.NewButtonWithIcon("", theme.WindowCloseIcon(), func() {
		v.closed = true
		v.box.Hide()
	})
	v.close.Importance = widget.LowImportance
	fyne.SetAccessibleLabel(v.close, "Close what the source shows")
	v.text = widget.NewRichText()
	v.text.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(v.text)
	scroll.SetMinSize(fyne.NewSize(0, readingHeight))
	v.box = container.NewBorder(container.NewBorder(widget.NewSeparator(), nil, nil, v.close, v.title), nil, nil, nil, scroll)
	v.box.Hide()
	return v
}

// storyPassages are the passages as the reader takes them: a list item
// written with "-" may be a line of dialogue (— Bonjour in Markdown).
func storyPassages(ps []passage) []storyread.Passage {
	out := make([]storyread.Passage, len(ps))
	for i, p := range ps {
		sp := storyread.Passage{Text: p.Plain}
		switch p.Kind {
		case prose.Heading:
			sp.Kind, sp.Level = storyread.Heading, p.Level
		case prose.Rule:
			sp.Kind = storyread.Break
		case prose.Item:
			if strings.HasPrefix(strings.TrimSpace(p.Markdown), "-") {
				sp.Kind = storyread.DashItem
			}
		}
		out[i] = sp
	}
	return out
}

// readingLanguage is the language the source is read in: its own if its
// words say so clearly, else the script's if it has word lists, else
// English (the punctuation rules hold for every language).
func readingLanguage(src, script string) (lang string, detected bool) {
	if l, sure := storyread.Detect(src); sure {
		return l, true
	}
	if storyread.Has(script) {
		return script, false
	}
	return "en", false
}

// currentReading is the source read, made again when it or the script's
// language changed.
func (s *sourcePane) currentReading() *storyread.Reading {
	key := s.w.language + "\x00" + s.src
	if s.reading == nil || s.readingFor != key {
		lang, _ := readingLanguage(s.src, s.w.language)
		s.reading, s.readingFor = storyread.Read(storyPassages(s.passages), lang, storyread.Hints{}), key
	}
	return s.reading
}

// showReading shows what the picked passage shows, or hides the view
// when none is picked.
func (s *sourcePane) showReading(picked bool) {
	v := s.view
	if picked {
		v.closed = false
	}
	if s.selected < 0 || s.selected >= len(s.passages) || s.ref.Origin == sourceNone || s.editing || v.closed {
		v.box.Hide()
		return
	}
	in, out, either, ok := placeWords(sceneHeaders(s.w.language))
	if !ok {
		in, out, either = "INT.", "EXT.", "INT./EXT."
	}
	v.text.ParseMarkdown(readingMarkdown(s.currentReading(), s.selected, in, out, either))
	v.box.Show()
	v.box.Refresh()
}

// mdEscape keeps text from reading as Markdown.
var mdEscape = strings.NewReplacer(`\`, `\\`, "*", `\*`, "_", `\_`, "`", "\\`", "[", `\[`, "]", `\]`, "#", `\#`, "<", `\<`)

// readingMarkdown is what passage p shows, as the view's Markdown.
func readingMarkdown(r *storyread.Reading, p int, in, out, either string) string {
	pr := r.Passages[p]
	sc := r.Scenes[pr.Scene]
	var b strings.Builder
	where := "in the scene from passage " + fmt.Sprint(sc.First+1)
	if pr.SceneStart != "" {
		where = "a scene starts here: " + pr.SceneStart
	}
	fmt.Fprintf(&b, "**Scene** %s\n\n*%s*\n\n", mdEscape.Replace(sc.Heading(in, out, either)), mdEscape.Replace(where))
	if len(pr.Lines) > 0 {
		b.WriteString("**Who says what**\n\n")
		for _, l := range pr.Lines {
			who := "?"
			if l.Speaker != "" {
				who = strings.ToUpper(l.Speaker)
			}
			fmt.Fprintf(&b, "- **%s**", mdEscape.Replace(who))
			if l.Manner != "" {
				fmt.Fprintf(&b, " (%s)", mdEscape.Replace(l.Manner))
			}
			fmt.Fprintf(&b, ": %s *(%s)*\n", mdEscape.Replace(shortLine(l.Text)), l.By)
		}
		b.WriteString("\n")
	}
	if len(pr.Flags) > 0 {
		b.WriteString("**Not seen on screen**: show it, cut it, or make it voice-over (V.O.)\n\n")
		for _, f := range pr.Flags {
			fmt.Fprintf(&b, "- *%s*: %s\n", mdEscape.Replace(f.Word), mdEscape.Replace(shortLine(f.Sentence)))
		}
		b.WriteString("\n")
	}
	var names []string
	if len(pr.Names) > 0 {
		names = append(names, "**Named** "+mdEscape.Replace(strings.Join(pr.Names, ", ")))
	}
	if len(pr.Places) > 0 {
		names = append(names, "**Places** "+mdEscape.Replace(strings.Join(pr.Places, ", ")))
	}
	if len(names) > 0 {
		b.WriteString(strings.Join(names, " · ") + "\n\n")
	}
	if len(pr.Lines) == 0 && len(pr.Flags) == 0 && len(names) == 0 {
		b.WriteString("*Nothing said, no names, nothing to flag.*\n\n")
	}
	fmt.Fprintf(&b, "*Read as %s, by word lists: guesses, not facts.*", mdEscape.Replace(storyread.Language(r.Lang).Name))
	return b.String()
}

// shortLine is a line or sentence cut to a readable length.
func shortLine(s string) string {
	words := strings.Fields(s)
	if len(words) > 24 {
		return strings.Join(words[:24], " ") + "…"
	}
	return strings.Join(words, " ")
}

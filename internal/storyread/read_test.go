package storyread

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/LaPingvino/accolade/internal/prose"
)

// passagesFrom reads Markdown as the source pane gives it (source.go's
// passagesOf): paragraphs, headings, breaks, each list item; an item
// written with "-" may be dialogue.
func passagesFrom(md string) []Passage {
	if strings.HasPrefix(md, "---\n") { // front matter
		if i := strings.Index(md[4:], "\n---\n"); i >= 0 {
			md = md[4+i+5:]
		}
	}
	var out []Passage
	for _, b := range prose.FromMarkdown(md).Blocks {
		text := func(bs []*prose.Block) string {
			return strings.TrimSpace((&prose.Doc{Blocks: bs}).Plain())
		}
		switch b.Kind {
		case prose.List:
			for _, it := range b.Children {
				out = append(out, Passage{Kind: DashItem, Text: text(it.Children)})
			}
		case prose.Heading:
			out = append(out, Passage{Kind: Heading, Level: b.Level, Text: text([]*prose.Block{b})})
		case prose.Rule:
			out = append(out, Passage{Kind: Break})
		case prose.Quote:
			out = append(out, Passage{Text: text(b.Children)})
		default:
			out = append(out, Passage{Text: text([]*prose.Block{b})})
		}
	}
	return out
}

// norm is text as compared: lower case, letters and digits, one space.
func norm(s string) string {
	var words []string
	for _, t := range tokenize(s) {
		words = append(words, t.low)
	}
	return strings.Join(words, " ")
}

type wantLine struct{ speaker, opening string }

type want struct {
	heldOut    bool
	floor      float64
	aliases    map[string][]string // a canonical name: its other names
	lines      []wantLine
	characters []string
	places     []string
}

func readWant(t *testing.T, path string) want {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w := want{aliases: map[string][]string{}}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "held-out:"):
			w.heldOut = strings.TrimSpace(strings.TrimPrefix(line, "held-out:")) == "yes"
		case strings.HasPrefix(line, "floor:"):
			w.floor, _ = strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, "floor:")), 64)
		case strings.HasPrefix(line, "="):
			names := strings.Split(strings.TrimPrefix(line, "="), "|")
			for i := range names {
				names[i] = strings.TrimSpace(names[i])
			}
			w.aliases[names[0]] = names[1:]
		case strings.HasPrefix(line, "characters:"):
			for _, n := range strings.Split(strings.TrimPrefix(line, "characters:"), ",") {
				w.characters = append(w.characters, strings.TrimSpace(n))
			}
		case strings.HasPrefix(line, "places:"):
			for _, n := range strings.Split(strings.TrimPrefix(line, "places:"), ",") {
				w.places = append(w.places, strings.TrimSpace(n))
			}
		default:
			who, opening, ok := strings.Cut(line, "|")
			if !ok {
				t.Fatalf("%s: %q", path, line)
			}
			w.lines = append(w.lines, wantLine{strings.TrimSpace(who), strings.TrimSpace(opening)})
		}
	}
	return w
}

// score is how a reading did on an excerpt.
type score struct {
	expected, found, attributed, exact, alias, extra int
	misses                                           []string
}

func (s score) add(o score) score {
	return score{s.expected + o.expected, s.found + o.found, s.attributed + o.attributed, s.exact + o.exact,
		s.alias + o.alias, s.extra + o.extra, nil}
}

func pct(a, b int) string {
	if b == 0 {
		return "  -  "
	}
	return fmt.Sprintf("%3.0f%%", 100*float64(a)/float64(b))
}

func (s score) String() string {
	return fmt.Sprintf("found %2d/%2d (%s)  attributed %2d/%2d (%s)  right %2d/%2d (%s; exact name %s)  extra lines %d",
		s.found, s.expected, pct(s.found, s.expected), s.attributed, s.found, pct(s.attributed, s.found),
		s.exact+s.alias, s.attributed, pct(s.exact+s.alias, s.attributed), pct(s.exact, s.attributed), s.extra)
}

func scoreReading(r *Reading, w want) score {
	type found struct {
		Line
		used bool
	}
	var lines []*found
	for _, p := range r.Passages {
		for _, l := range p.Lines {
			lines = append(lines, &found{Line: l})
		}
	}
	s := score{expected: len(w.lines)}
	at := 0
	for _, wl := range w.lines {
		open := norm(wl.opening)
		var hit *found
		for i := at; i < len(lines); i++ {
			if !lines[i].used && strings.HasPrefix(norm(lines[i].Text), open) {
				hit, at = lines[i], i+1
				break
			}
		}
		if hit == nil {
			s.misses = append(s.misses, fmt.Sprintf("not found: %q (%s)", wl.opening, wl.speaker))
			continue
		}
		hit.used = true
		s.found++
		if hit.Speaker == "" {
			s.misses = append(s.misses, fmt.Sprintf("no speaker: %q (%s)", wl.opening, wl.speaker))
			continue
		}
		s.attributed++
		switch got := norm(hit.Speaker); {
		case got == norm(wl.speaker):
			s.exact++
		case slices.ContainsFunc(w.aliases[wl.speaker], func(a string) bool { return norm(a) == got }):
			s.alias++
		default:
			s.misses = append(s.misses, fmt.Sprintf("wrong: %q is %s, read as %s (%s)", wl.opening, wl.speaker, hit.Speaker, hit.By))
		}
	}
	for _, l := range lines {
		if !l.used {
			s.extra++
		}
	}
	return s
}

// TestExcerpts reads the public-domain excerpts in testdata and scores
// who it says speaks each line against what was read by hand (the .want
// files). Run with -v for the table; it fails when an excerpt falls
// below its floor.
func TestExcerpts(t *testing.T) {
	files, _ := filepath.Glob("testdata/*.md")
	if len(files) == 0 {
		t.Fatal("no excerpts")
	}
	var tuned, held score
	perLang := map[string]score{}
	var report []string
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		lang := name[:2]
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		w := readWant(t, strings.TrimSuffix(f, ".md")+".want")
		ps := passagesFrom(string(b))
		if got, sure := Detect(string(b)); got != lang || !sure {
			t.Errorf("%s: language read as %s (sure %v), not %s", name, got, sure, lang)
		}
		r := Read(ps, lang, Hints{})
		s := scoreReading(r, w)
		tag := "tuned on"
		if w.heldOut {
			tag = "held out"
			held = held.add(s)
		} else {
			tuned = tuned.add(s)
		}
		perLang[lang] = perLang[lang].add(s)
		report = append(report, fmt.Sprintf("%-18s %s  %s", name, tag, s))
		for _, m := range s.misses {
			report = append(report, "    "+m)
		}
		var chars, places []string
		for _, c := range r.Characters {
			chars = append(chars, c.Name)
		}
		for _, p := range r.Places {
			places = append(places, p.Name)
		}
		for _, c := range w.characters {
			if !slices.ContainsFunc(chars, func(n string) bool { return norm(n) == norm(c) }) {
				report = append(report, "    character not found: "+c)
			}
		}
		for _, p := range w.places {
			if !slices.ContainsFunc(places, func(n string) bool { return norm(n) == norm(p) }) {
				report = append(report, "    place not found: "+p)
			}
		}
		report = append(report, "    characters: "+strings.Join(chars, ", "), "    places: "+strings.Join(places, ", "))
		if right := float64(s.exact+s.alias) / float64(max(s.expected, 1)); right < w.floor {
			t.Errorf("%s: %.0f%% of the lines have the right speaker, below the floor of %.0f%%", name, 100*right, 100*w.floor)
		}
	}
	t.Log("\n" + strings.Join(report, "\n"))
	var langLines []string
	for _, l := range Languages() {
		langLines = append(langLines, fmt.Sprintf("%s  %s", l, perLang[l]))
	}
	t.Logf("\nper language:\n%s\ntuned on: %s\nheld out: %s", strings.Join(langLines, "\n"), tuned, held)
}

// A language without word lists is not taken for one with them, and reads
// with English's: the quotes still find its lines.
func TestUnlistedLanguage(t *testing.T) {
	it := "«Dove vai?» domandò Lucia, sulla porta della casa. «Al paese» rispose il ragazzo, e se ne andò per la strada " +
		"senza voltarsi. La donna rimase a guardarlo finché non fu sparito dietro la chiesa, poi rientrò in cucina."
	if l, sure := Detect(it); sure {
		t.Errorf("Italian read as %s, sure", l)
	}
	if Has("it") || Language("it").Code != "en" {
		t.Error("no Italian lists, so English's")
	}
	r := Read([]Passage{{Text: it}}, "it", Hints{})
	if got := r.Passages[0].Lines; len(got) != 2 || got[0].Text != "Dove vai?" || got[1].Text != "Al paese" {
		t.Errorf("lines: %+v", got)
	}
}

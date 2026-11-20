package help

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LaPingvino/lexington/cli"
)

// The Lexington page and cheat sheet are made from Lexington's own flags
// (its -help) and formats (cli.Formats), between their marks
// (<!-- flags --> … <!-- /flags -->). The test fails when they are not
// what Lexington has; UPDATE_HELP=1 writes them again:
//
//	UPDATE_HELP=1 go test -run TestLexingtonPages ./docs/help/
//
// Lexington has no exported flag set, so its -help is read from stderr.

// lexFlag is one of Lexington's flags.
type lexFlag struct {
	name, kind, usage string
}

// lexingtonUsage is what lexington -help prints.
func lexingtonUsage(t *testing.T) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	code := cli.Main(context.Background(), []string{"-help"})
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	if code != 0 {
		t.Fatalf("lexington -help: exit %d", code)
	}
	return string(out)
}

var flagLineRE = regexp.MustCompile(`^  -(\S+)(?: (\S+))?$`)

// parseUsage reads flag.PrintDefaults' text.
func parseUsage(s string) []lexFlag {
	var out []lexFlag
	for _, l := range strings.Split(s, "\n") {
		if m := flagLineRE.FindStringSubmatch(l); m != nil {
			out = append(out, lexFlag{name: m[1], kind: m[2]})
			continue
		}
		if len(out) > 0 && strings.HasPrefix(l, "    \t") {
			f := &out[len(out)-1]
			f.usage = strings.TrimSpace(f.usage + " " + strings.TrimSpace(l))
		}
	}
	return out
}

// flagGroups put the flags in groups, in this order; a flag not here
// goes under Other.
var flagGroups = []struct {
	title string
	flags []string
}{
	{"Input and output", []string{"i", "o", "from", "to", "template"}},
	{"Format, paper and pagination", []string{"e", "presets", "formats", "page", "pagination", "config", "dumpconfig", "scenein", "sceneout"}},
	{"Scene numbers, pages and reports", []string{"number", "mergeomitted", "continueds", "lockpages", "lock", "lengths", "report"}},
	{"Colours", []string{"colors", "colorstyle", "colorscope", "colortint", "colorpatterns"}},
	{"Parts of the script and sides", []string{"tag", "character", "sides", "cuewords"}},
	{"Other", []string{"lint", "help", "version"}},
}

// grouped are the flags by group, in flagGroups' order.
func grouped(flags []lexFlag) (titles []string, groups [][]lexFlag) {
	byName := map[string]lexFlag{}
	for _, f := range flags {
		byName[f.name] = f
	}
	placed := map[string]bool{}
	for _, g := range flagGroups {
		var list []lexFlag
		for _, n := range g.flags {
			if f, ok := byName[n]; ok {
				list = append(list, f)
				placed[n] = true
			}
		}
		if g.title == "Other" {
			for _, f := range flags {
				if !placed[f.name] {
					list = append(list, f)
				}
			}
		}
		if len(list) > 0 {
			titles = append(titles, g.title)
			groups = append(groups, list)
		}
	}
	return titles, groups
}

// flagsMarkdown is the flags as the page (or, short, the cheat sheet)
// has them.
func flagsMarkdown(flags []lexFlag, short bool) string {
	titles, groups := grouped(flags)
	var b strings.Builder
	for i, g := range groups {
		fmt.Fprintf(&b, "\n### %s\n\n", titles[i])
		for _, f := range g {
			name := "-" + f.name
			if f.kind != "" && !short {
				name += " " + f.kind
			}
			usage := f.usage
			if short {
				usage = firstSentence(usage)
			}
			fmt.Fprintf(&b, "- `%s` — %s\n", name, usage)
		}
	}
	return b.String()
}

// firstSentence is s up to its first full stop outside parentheses;
// a long one ends at its first colon or semicolon, with "…".
func firstSentence(s string) string {
	cut := func(stops string) string {
		depth := 0
		for i, r := range s {
			switch {
			case r == '(':
				depth++
			case r == ')':
				depth--
			case depth == 0 && strings.ContainsRune(stops, r) && (i+1 == len(s) || s[i+1] == ' '):
				return strings.TrimSpace(s[:i+1])
			}
		}
		return s
	}
	first := cut(".")
	if len(first) > 110 {
		if short := cut(":;"); len(short) < len(first) {
			return strings.TrimRight(short, ":;") + " …"
		}
	}
	return first
}

// formatsMarkdown is Lexington's output formats.
func formatsMarkdown() string {
	var own, other []string
	for _, f := range cli.Formats {
		item := fmt.Sprintf("%s (`%s`)", f.Name, f.Key)
		if f.Needs == "" {
			own = append(own, item)
		} else {
			other = append(other, fmt.Sprintf("%s, with %s", item, f.Needs))
		}
	}
	return "\n- **Its own:** " + strings.Join(own, ", ") + ".\n" +
		"- **With other programs installed:** " + strings.Join(other, "; ") + ".\n"
}

var regionRE = func(mark string) *regexp.Regexp {
	return regexp.MustCompile(`(?s)<!-- ` + mark + ` -->.*?<!-- /` + mark + ` -->`)
}

// fill puts body between mark's marks in text.
func fill(text, mark, body string) string {
	return regionRE(mark).ReplaceAllLiteralString(text, "<!-- "+mark+" -->\n"+body+"\n<!-- /"+mark+" -->")
}

func TestLexingtonPages(t *testing.T) {
	flags := parseUsage(lexingtonUsage(t))
	if len(flags) < 30 {
		t.Fatalf("read only %d flags from lexington -help", len(flags))
	}
	for _, p := range []struct {
		file  string
		fills map[string]string
	}{
		{"lexington.md", map[string]string{"flags": flagsMarkdown(flags, false), "formats": formatsMarkdown()}},
		{"cheat-lexington.md", map[string]string{"flags": flagsMarkdown(flags, true)}},
	} {
		b, err := os.ReadFile(p.file)
		if err != nil {
			t.Fatal(err)
		}
		want := string(b)
		for mark, body := range p.fills {
			if !regionRE(mark).MatchString(want) {
				t.Fatalf("%s has no <!-- %s --> marks", p.file, mark)
			}
			want = fill(want, mark, body)
		}
		if os.Getenv("UPDATE_HELP") != "" {
			if err := os.WriteFile(filepath.Join(".", p.file), []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if string(b) != want {
			t.Errorf("%s is not what Lexington's flags make: UPDATE_HELP=1 go test -run TestLexingtonPages ./docs/help/", p.file)
		}
		for _, f := range flags {
			if !strings.Contains(string(b), "`-"+f.name) {
				t.Errorf("%s does not list -%s", p.file, f.name)
			}
		}
	}
}

// The examples on the page are Lexington's flags and values: each one's
// flags parse, and the ones that need no file of their own run.
func TestLexingtonExamples(t *testing.T) {
	page := must("lexington").Markdown
	var cmds []string
	for _, l := range strings.Split(page, "\n") {
		if strings.HasPrefix(l, "lexington ") {
			cmds = append(cmds, l)
		}
	}
	if len(cmds) < 7 {
		t.Fatalf("only %d examples", len(cmds))
	}
	dir := t.TempDir()
	script := "Title: Test\n\nINT. HOUSE - DAY\n\nRain.\n\nANNA\nHello.\n\nEXT. GARDEN - NIGHT\n\nWind.\n\nANNA\nBye.\n"
	if err := os.WriteFile(filepath.Join(dir, "script.fountain"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	ran := 0
	for _, c := range cmds {
		args := strings.Fields(strings.TrimPrefix(c, "lexington "))
		needs := ""
		for i := range args {
			if i > 0 && args[i-1] == "-i" && args[i] != "script.fountain" {
				needs = args[i]
			}
			if i > 0 && (args[i-1] == "-i" || args[i-1] == "-o" || args[i-1] == "-lock") {
				args[i] = filepath.Join(dir, args[i])
			}
		}
		if needs != "" {
			continue // an FDX or a PDF of its own: the flags are the same as above
		}
		if code := cli.Main(context.Background(), args); code != 0 {
			t.Errorf("%q: exit %d", c, code)
		}
		ran++
	}
	if ran < 6 {
		t.Errorf("ran %d examples", ran)
	}
}

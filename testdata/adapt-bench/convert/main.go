// Command convert turns the Project Gutenberg plain-text files of the
// adaptation benchmark's pairs into what the benchmark reads: the story
// as Markdown (source.md: its chapters as ## headings, its paragraphs as
// passages) and the play as Fountain (script.fountain: acts as # sections,
// scenes as forced headings, cues, dialogue, stage directions as action).
//
//	go run ./testdata/adapt-bench/convert <pair> <story.txt> <play.txt>
//
// writes testdata/adapt-bench/<pair>/source.md and script.fountain (run
// from the repository's root). The texts are Gutenberg's, fetched by hand
// (see each pair's README.md); only the pairs below are known.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// pair is how one pair's two files are read.
type pair struct {
	title, author, adapter, credit string
	storyEtext, playEtext          int
	chapter                        *regexp.Regexp // a chapter's heading line in the story
	storyEnd                       string         // the story's last line before Gutenberg's end (left out), or ""
	playStart                      string         // the play's first line (its first act)
	playEnd                        string         // the play's last line (kept), its last occurrence
	cue                            *regexp.Regexp // a speech's start: the cue as written, then the speech
	cues                           map[string]string
}

var pairs = map[string]pair{
	"christmas-carol": {
		title: "A Christmas Carol; or, The Miser's Warning!", author: "C. Z. Barnett",
		adapter: "Adapted from Charles Dickens' A Christmas Carol", credit: "First produced at the Royal Surrey Theatre, 5 February 1844",
		storyEtext: 46, playEtext: 41739,
		chapter:   regexp.MustCompile(`^STAVE [IVX]+:`),
		storyEnd:  "",
		playStart: "ACT I.", playEnd: "THE CURTAIN FALLS.",
		cue: regexp.MustCompile(`^([0-9A-Z][0-9A-Z' ]*[A-Z](?:\. [A-Z])?)\. (.*)$`),
		cues: map[string]string{
			"SCR": "SCROOGE", "SCROOGE": "SCROOGE", "BOB": "BOB", "BOY": "BOY", "CHEER": "MR CHEERLY",
			"CHILD": "CHILD", "CHILDREN": "CHILDREN", "ELLEN": "ELLEN", "EUS": "EUSTON", "FRANK": "FRANK",
			"GHOST": "MARLEY'S GHOST", "GIRL": "GIRL", "HEART": "MR HEARTLY", "JOE": "OLD JOE", "LADIES": "LADIES",
			"MART": "MARTHA", "MARTHA": "MARTHA", "MRS. C": "MRS CRATCHIT", "MRS. M": "MRS MILDEW", "OMNES": "ALL",
			"SAM": "DARK SAM", "1ST SPIRIT": "GHOST OF CHRISTMAS PAST", "2ND SPIRIT": "GHOST OF CHRISTMAS PRESENT",
		},
	},
	"jane-eyre": {
		title: "Jane Eyre: A Drama in Five Acts", author: "John Brougham",
		adapter: "Adapted from Charlotte Brontë's novel", credit: "First performed at Laura Keene's Varieties, New York, 1856",
		storyEtext: 1260, playEtext: 79159,
		chapter:   regexp.MustCompile(`^CHAPTER [IVXL]+\b`),
		playStart: "ACT I.", playEnd: "_Curtain._",
		cue: regexp.MustCompile(`^_([^_]+)\._ (.*)$`),
		cues: map[string]string{
			"All": "ALL", "Blan": "BLANCHE", "Lady Blan": "BLANCHE", "Brock": "BROCKLEHURST",
			"Col. Dent": "COLONEL DENT", "Dent": "COLONEL DENT", "Dow": "DOWAGER", "Grace": "GRACE POOL",
			"Jane": "JANE", "Jame": "JANE", "John": "JOHN", "Lady Mary": "MARY INGRAM", "L. Mary": "MARY INGRAM",
			"Mary": "MARY INGRAM", "Lord Ing": "LORD INGRAM", "Miss G": "MISS GRYCE", "Miss S": "MISS SCATCHERD",
			"Miss T": "MISS TEMPLE", "Mrs. D": "MRS DOWNEY", "Rob": "ROBERT", "Roch": "ROCHESTER",
		},
	},
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: convert <pair> <story.txt> <play.txt>")
		os.Exit(2)
	}
	p, ok := pairs[os.Args[1]]
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown pair", os.Args[1])
		os.Exit(2)
	}
	story, err := gutenbergBody(os.Args[2])
	check(err)
	play, err := gutenbergBody(os.Args[3])
	check(err)
	dir := filepath.Join("testdata", "adapt-bench", os.Args[1])
	check(os.MkdirAll(dir, 0o755))
	check(os.WriteFile(filepath.Join(dir, "source.md"), []byte(storyMarkdown(p, story)), 0o644))
	check(os.WriteFile(filepath.Join(dir, "script.fountain"), []byte(playFountain(p, play)), 0o644))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// gutenbergBody is the file's lines between Gutenberg's START and END
// lines, without carriage returns.
func gutenbergBody(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n")
	start, end := -1, len(lines)
	for i, l := range lines {
		if strings.HasPrefix(l, "*** START OF") && start < 0 {
			start = i + 1
		}
		if strings.HasPrefix(l, "*** END OF") {
			end = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("%s: no Gutenberg START line", path)
	}
	return lines[start:end], nil
}

// blocks are the lines' paragraphs (split at blank lines), each with its
// lines kept, and whether its first line was indented.
type block struct {
	lines    []string
	indented bool
}

func blocksOf(lines []string) []block {
	var out []block
	var cur []string
	ind := false
	flush := func() {
		if len(cur) > 0 {
			out = append(out, block{cur, ind})
			cur = nil
		}
	}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			flush()
			continue
		}
		if len(cur) == 0 {
			ind = strings.HasPrefix(l, " ")
		}
		cur = append(cur, strings.TrimSpace(l))
	}
	flush()
	return out
}

func (b block) text() string { return strings.Join(b.lines, " ") }

var illustration = regexp.MustCompile(`^\[Illustration`)

// storyMarkdown: from the first chapter on, each chapter heading a ##
// heading (its title in title case left as Gutenberg has it), each other
// paragraph one line; illustrations left out.
func storyMarkdown(p pair, lines []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", storyTitle(p))
	started := false
	bs := blocksOf(lines)
	for i := 0; i < len(bs); i++ {
		b := bs[i]
		first := b.lines[0]
		if p.chapter.MatchString(first) {
			started = true
			h := strings.Join(strings.Fields(b.text()), " ")
			// Jane Eyre: CHAPTER I alone on its line
			fmt.Fprintf(&sb, "## %s\n\n", h)
			continue
		}
		if !started || illustration.MatchString(first) {
			continue
		}
		if p.storyEnd != "" && first == p.storyEnd {
			break
		}
		t := strings.Join(strings.Fields(b.text()), " ")
		if t == "THE END" || t == "THE END." {
			continue
		}
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") || strings.HasPrefix(t, "*") || startsNumbered(t) {
			t = `\` + t // not a heading, list or rule
		}
		sb.WriteString(t + "\n\n")
	}
	return sb.String()
}

var numbered = regexp.MustCompile(`^\d+[.)] `)

func startsNumbered(t string) bool { return numbered.MatchString(t) }

func storyTitle(p pair) string {
	switch p.storyEtext {
	case 46:
		return "A Christmas Carol"
	case 1260:
		return "Jane Eyre"
	}
	return p.title
}

var (
	sceneLine = regexp.MustCompile(`^SCENE ([IVX]+)\.?\s*(?:--|—)?\s*(.*)$`)
	actLine   = regexp.MustCompile(`^ACT ([IVX]+)\.$`)
	italics   = regexp.MustCompile(`_([^_]+)_`)
	jointCue  = regexp.MustCompile(`^_([^_]+)\._ ?\} ?_([^_]+)\._ ?\} (.*)$`)
	inlineDir = regexp.MustCompile(`\(\s*_[^)]*\)|\[_[^\]]*(?:\]|$)`)
)

// playFountain: the play from its first act to its last line.
func playFountain(p pair, lines []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Title: %s\nCredit: by\nAuthor: %s\nSource: %s\nNotes: %s. Project Gutenberg eBook #%d, converted for Accolade's adaptation benchmark.\n\n",
		p.title, p.author, p.adapter, p.credit, p.playEtext)
	bs := blocksOf(lines)
	start, end := -1, -1
	for i, b := range bs {
		if start < 0 && b.text() == p.playStart {
			start = i
		}
		if b.text() == p.playEnd {
			end = i
		}
	}
	if start < 0 || end < start {
		fmt.Fprintln(os.Stderr, "play start or end not found")
		os.Exit(1)
	}
	act, last := "", "" // the act, and the last one to speak
	for _, b := range bs[start : end+1] {
		t := strings.Join(strings.Fields(b.text()), " ")
		if m := actLine.FindStringSubmatch(t); m != nil {
			act, last = m[1], ""
			fmt.Fprintf(&sb, "# ACT %s\n\n", act)
			continue
		}
		if m := sceneLine.FindStringSubmatch(t); m != nil {
			desc := plain(m[2])
			fmt.Fprintf(&sb, ".ACT %s SCENE %s - %s\n\n", act, m[1], place(desc))
			last = ""
			if desc != "" {
				sb.WriteString(action(desc) + "\n\n")
			}
			continue
		}
		if m := jointCue.FindStringSubmatch(t); m != nil { // two speak at once: _Dow._ } _Blan._} The governess!
			a, okA := p.cues[m[1]]
			c, okC := p.cues[m[2]]
			if okA && okC {
				sb.WriteString(speech(a+" AND "+c, m[3]))
				last = ""
				continue
			}
		}
		if m := p.cue.FindStringSubmatch(t); m != nil {
			if name, ok := p.cues[m[1]]; ok && !b.indented {
				sb.WriteString(speech(name, m[2]))
				last = name
				continue
			}
		}
		if !b.indented && last != "" && !strings.ContainsAny(t[:1], "(_[") && t != strings.ToUpper(t) {
			// a speech going on after a stage direction (an entrance), its
			// speaker not named again
			sb.WriteString(speech(last+" (CONT'D)", t))
			continue
		}
		sb.WriteString(action(plain(t)) + "\n\n")
	}
	return sb.String()
}

// place is a scene's short name for its heading: the description up to
// its first full stop or dash, at most a few words.
func place(desc string) string {
	if strings.HasPrefix(desc, "Enter") { // no place of its own: where the last one was
		return "CONTINUED"
	}
	desc = strings.NewReplacer("Mr. ", "Mr ", "Mrs. ", "Mrs ").Replace(desc)
	if i := strings.IndexAny(desc, ".—"); i > 0 {
		desc = desc[:i]
	}
	if i := strings.Index(desc, "--"); i > 0 {
		desc = desc[:i]
	}
	ws := strings.Fields(strings.ToUpper(strings.Trim(desc, " ,;:")))
	if len(ws) > 8 {
		ws = ws[:8]
	}
	s := strings.Join(ws, " ")
	if s == "" {
		s = "CONTINUED"
	}
	return s
}

// plain is text without Gutenberg's _italics_ marks.
func plain(s string) string { return strings.TrimSpace(italics.ReplaceAllString(s, "$1")) }

// action is a stage direction as Fountain action: its brackets gone, and
// forced (!) when it would read as a cue or a heading.
func action(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") {
		s = strings.TrimPrefix(s, "(")
		s = strings.TrimSuffix(s, ")")
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "["))
	if s == strings.ToUpper(s) || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "=") || strings.HasPrefix(s, ">") {
		s = "!" + s
	}
	return s
}

// speech is a cue and its dialogue, the stage directions in it as
// parentheticals on their own lines.
func speech(name, text string) string {
	var sb strings.Builder
	sb.WriteString(name + "\n")
	rest := text
	for rest != "" {
		loc := inlineDir.FindStringIndex(rest)
		if loc == nil {
			writeLine(&sb, rest)
			break
		}
		writeLine(&sb, rest[:loc[0]])
		d := plain(rest[loc[0]:loc[1]])
		d = strings.Trim(d, "()[] ")
		if d != "" {
			sb.WriteString("(" + d + ")\n")
		}
		rest = rest[loc[1]:]
	}
	sb.WriteString("\n")
	return sb.String()
}

func writeLine(sb *strings.Builder, s string) {
	s = strings.TrimSpace(italics.ReplaceAllString(s, "*$1*"))
	if s != "" {
		sb.WriteString(s + "\n")
	}
}

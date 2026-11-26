package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The Table Read (LOOP.md, TABLE READ / CAST, levels 1 and 2): the script
// read aloud around a table, or shown on a TV or projector, one scene at
// a time in large type on dark. A panel picks what is read (a range of
// scenes, a character's scenes, a breakdown element's, a shoot day's, a
// colour's); each actor's lines have a tint of their own, a narrator
// reads the headings and the action in narrator mode, the storyboard's
// frames can show between the lines, a cast photo by each name. It reads
// the script and changes nothing: what it keeps is the pick, in memory.
// This file is what it reads; tableread_view.go shows it,
// tableread_follow.go lets it follow another device.

// trKind is what a line of the table read is.
type trKind int

const (
	trHeading trKind = iota
	trAction
	trCue
	trParen
	trSpeech
	trTransition
	trLyric
	trFrame // a storyboard frame
)

// trLine is a line as the table read shows it.
type trLine struct {
	Line    int // the script's line it is (the heading's for a Storyboarder frame)
	Kind    trKind
	Text    string // as read: without notes, markers, emphasis
	Speaker string // a cue's, parenthetical's or speech's character (charKey)
	// Narrator: read by the narrator (a heading, action, a transition);
	// NarratorStart: the first of a run of them (the mark shows there)
	Narrator, NarratorStart bool
	// First: the speaker's first cue in the script (the photo shows there)
	First bool
	Frame string // a frame's picture, as the loader names it
	// Secs: a frame's time on screen, in seconds (0: not set)
	Secs float64
}

// trScene is a scene of the table read (the lines before the first
// heading, if any, are a scene of their own with no heading).
type trScene struct {
	Line     int    // its heading's line (or its first line)
	Heading  string // as printed ("" for the opening)
	Number   string // its scene number as written, else counted
	Color    string // its heading's colour note
	Lines    []trLine
	Speakers []string // who speaks in it, in order (charKey)
	Rows     int      // its printed rows, for the estimate
	// Timed: its frames' time on screen, in seconds (a [[board:]]'s
	// duration, a Storyboarder board's, else its scene's default);
	// Untimed: how many of its frames have none
	Timed   float64
	Untimed int
}

// secs is how long the scene takes, in seconds: its frames' time when
// every frame has one (the boards are the cut), else a page a minute,
// but never shorter than the frames timed; byBoards: the frames' time.
func (sc trScene) secs() (secs float64, byBoards bool) {
	read := float64(sc.Rows*60) / linesPerPage
	if sc.Timed > 0 && (sc.Untimed == 0 || sc.Timed > read) {
		return sc.Timed, true
	}
	return read, false
}

// trScript is a script read for the table read.
type trScript struct {
	Scenes []trScene
	// Cast: everyone who speaks, in the order they first do (their tint
	// goes by it); Names: each one's name as shown
	Cast  []string
	Names map[string]string
}

// readTable reads a script's body for the table read; sb, if not nil, is
// the boards Storyboarder keeps for it.
func readTable(text string, starts []string, d *fountain.Dialect, sb *sbProject) *trScript {
	body := text[:bodyEndByte(text)]
	lines := strings.Split(body, "\n")
	kinds := syntax.ClassifyIn(lines, starts, d)
	dataPath := boardDataPath(text)
	list := charListOf(text)
	s := &trScript{Names: map[string]string{}}
	var sc *trScene
	counted := 0
	speaker, narrating, inBoneyard := "", false, false
	newScene := func(line int) {
		s.Scenes = append(s.Scenes, trScene{Line: line})
		sc = &s.Scenes[len(s.Scenes)-1]
		narrating = false
	}
	add := func(l trLine) {
		if sc == nil {
			newScene(l.Line)
		}
		if l.Narrator {
			l.NarratorStart = !narrating
		}
		narrating = l.Narrator || l.Kind == trFrame && narrating
		if l.Kind == trFrame {
			sc.Timed += l.Secs
			if l.Secs == 0 {
				sc.Untimed++
			}
		}
		sc.Lines = append(sc.Lines, l)
	}
	for i, raw := range lines {
		t := strings.TrimSpace(raw)
		if inBoneyard || strings.HasPrefix(t, "/*") {
			inBoneyard = !strings.Contains(strings.TrimPrefix(t, "/*"), "*/")
			continue
		}
		k := kinds[i]
		switch k {
		case syntax.TitlePage, syntax.Empty, syntax.Section, syntax.Synopsis, syntax.PageBreak:
			if k == syntax.Empty {
				speaker = ""
			}
			continue
		case syntax.Note:
			if t := strings.TrimSpace(t); strings.HasPrefix(t, "[[") && strings.Count(t, "[[") == 1 {
				if b, ok := lex.ParseBoard(t[2 : len(t)-2]); ok {
					add(trLine{Line: i, Kind: trFrame, Frame: boardSrc(b, dataPath), Speaker: speaker, Secs: b.Duration})
				}
			}
			continue
		}
		shown := readText(t)
		if shown == "" && k != syntax.Dialogue {
			continue
		}
		width := actionColumns
		switch k {
		case syntax.SceneHeading:
			counted++
			newScene(i)
			sc.Heading = printedHeading(t)
			_, num := lex.SceneNumber(lex.StripNotes(t))
			sc.Number = num
			if sc.Number == "" {
				sc.Number = fmt.Sprint(counted)
			}
			sc.Color = lex.NoteColor(t)
			add(trLine{Line: i, Kind: trHeading, Text: sc.Heading, Narrator: true})
			speaker = ""
			if sb != nil {
				if m := sbSceneID.FindStringSubmatch(t); m != nil {
					if p := sb.scenes[m[2]]; p != nil {
						for j, pic := range p.picture {
							add(trLine{Line: i, Kind: trFrame, Frame: pic, Secs: p.boardSecs(j)})
						}
					}
				}
			}
		case syntax.Character:
			speaker = charKey(t)
			name := cueShown(t)
			if _, ok := s.Names[speaker]; !ok {
				s.Cast = append(s.Cast, speaker)
				if e := list.find(speaker); e >= 0 {
					s.Names[speaker] = strings.TrimSpace(list.entries[e].Name)
				} else {
					s.Names[speaker] = cueName(name)
				}
				add(trLine{Line: i, Kind: trCue, Text: name, Speaker: speaker, First: true})
			} else {
				add(trLine{Line: i, Kind: trCue, Text: name, Speaker: speaker})
			}
			if !slices.Contains(sc.Speakers, speaker) {
				sc.Speakers = append(sc.Speakers, speaker)
			}
		case syntax.Parenthetical:
			width = parenColumns
			add(trLine{Line: i, Kind: trParen, Text: shown, Speaker: speaker})
		case syntax.Dialogue:
			width = dialogColumns
			if shown == "" {
				continue
			}
			add(trLine{Line: i, Kind: trSpeech, Text: shown, Speaker: speaker})
		case syntax.Lyrics:
			width = dialogColumns
			add(trLine{Line: i, Kind: trLyric, Text: shown, Speaker: speaker, Narrator: speaker == ""})
		case syntax.Transition:
			add(trLine{Line: i, Kind: trTransition, Text: shown, Narrator: true})
		default: // action, centred
			add(trLine{Line: i, Kind: trAction, Text: shown, Narrator: true})
		}
		sc.Rows += max(1, (utf8.RuneCountInString(shown)+width-1)/width)
	}
	return s
}

// readText is a line as read aloud: its notes, its forcing marks and its
// emphasis markers left out.
func readText(t string) string {
	t = strings.TrimSpace(stripNotes(t))
	switch {
	case strings.HasPrefix(t, ">") && strings.HasSuffix(t, "<"):
		t = strings.TrimSuffix(t[1:], "<")
	case strings.HasPrefix(t, "!"), strings.HasPrefix(t, "@"), strings.HasPrefix(t, ">"), strings.HasPrefix(t, "~"):
		t = t[1:]
	case strings.HasPrefix(t, ".") && !strings.HasPrefix(t, ".."):
		t = t[1:]
	}
	t = strings.NewReplacer("**", "", "*", "", "_", "").Replace(t)
	return strings.Join(strings.Fields(t), " ")
}

// cueShown is a cue as the table read shows it: its name and extension,
// without @ and the dual dialogue's ^.
func cueShown(t string) string {
	t = strings.TrimSpace(stripNotes(t))
	t = strings.TrimSpace(strings.TrimSuffix(t, "^"))
	return strings.TrimPrefix(t, "@")
}

// trPick is what the table read shows: a range of scenes (by index,
// inclusive; To < 0: to the end), only a character's (charKey; "": all),
// only a tag's (trTags' key; "": all).
type trPick struct {
	From, To  int
	Character string
	Tag       string
}

// trTags are the tags the panel can pick by: each one's label and its
// scenes (indexes into trScript.Scenes), in the panel's order.
type trTags struct {
	keys   []string
	labels map[string]string
	scenes map[string][]int
}

func (t *trTags) add(key, label string, scene int) {
	if t.labels == nil {
		t.labels, t.scenes = map[string]string{}, map[string][]int{}
	}
	if _, ok := t.labels[key]; !ok {
		t.keys = append(t.keys, key)
		t.labels[key] = label
	}
	if !slices.Contains(t.scenes[key], scene) {
		t.scenes[key] = append(t.scenes[key], scene)
	}
}

// picked are the scenes p shows, in order.
func (s *trScript) picked(p trPick, tags *trTags) []int {
	var out []int
	to := p.To
	if to < 0 || to >= len(s.Scenes) {
		to = len(s.Scenes) - 1
	}
	for i := max(0, p.From); i <= to; i++ {
		if p.Character != "" && !slices.Contains(s.Scenes[i].Speakers, p.Character) {
			continue
		}
		if p.Tag != "" && (tags == nil || !slices.Contains(tags.scenes[p.Tag], i)) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// sceneAtLine is the scene line is in (-1: none).
func (s *trScript) sceneAtLine(line int) int {
	at := -1
	for i, sc := range s.Scenes {
		if sc.Line <= line {
			at = i
		}
	}
	return at
}

// tintOf is speaker's tint: its place in the cast, round the palette.
func (s *trScript) tintOf(speaker string) int {
	i := slices.Index(s.Cast, speaker)
	if i < 0 {
		return -1
	}
	return i % len(trTints)
}

// rowsOf are the printed rows of the scenes, and their time (secs).
func (s *trScript) rowsOf(scenes []int) (rows int, secs float64) {
	for _, i := range scenes {
		t, _ := s.Scenes[i].secs()
		rows, secs = rows+s.Scenes[i].Rows, secs+t
	}
	return rows, secs
}

// trEstimate is rows' length as pages (in eighths, as a schedule counts
// them) and secs as a time: "1 3/8 pg · ~1 min".
func trEstimate(rows int, secs float64) string {
	if rows == 0 {
		return "0 pg"
	}
	eighths := max(1, (rows*8+linesPerPage/2)/linesPerPage)
	pages := ""
	switch whole, part := eighths/8, eighths%8; {
	case whole == 0:
		pages = fmt.Sprintf("%d/8", part)
	case part == 0:
		pages = fmt.Sprint(whole)
	default:
		pages = fmt.Sprintf("%d %d/8", whole, part)
	}
	n := int(secs + 0.5)
	t := "~" + plural(max(1, (n+30)/60), "min")
	if n < 45 {
		t = fmt.Sprintf("~%d s", max(5, (n+4)/5*5))
	}
	return pages + " pg · " + t
}

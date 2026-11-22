package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"

	"github.com/LaPingvino/accolade/internal/storyread"
)

// What the Characters pane lists (characters_pane.go): in a screenplay,
// everyone who speaks (their cues as Lexington reads them, the learned
// notes' other names merged in as the reports merge them) and everyone in
// the script's character list (characters_block.go), the most present
// first; in a Markdown document, the names its words read as someone's
// (storyread, as the source pane reads a story).

// cueSpan is a cue's name in the script: its line and the runes of the
// name on it (without @, extension, ^), and the name as written there.
type cueSpan struct {
	line, start, end int // start, end: runes in the text
	name             string
}

// charScene is a scene of the script as the pane names it.
type charScene struct {
	line    int
	heading string
	label   string // its number as the script shows it: locked (2, 1A), else counted
}

// charInfo is a character as the pane lists it.
type charInfo struct {
	Name     string // shown: the list's spelling, else the cue's in capitals
	Key      string // charKey
	Cues     []cueSpan
	Scenes   []int // indexes into charModel.scenes
	Speeches int
	Words    int
	Aliases  []string // the source's other names (the learned notes)
	Entry    int      // its entry in the list (-1: none)
	Not      bool     // on the learned notes' not characters list
	Mentions []int    // Markdown: the runes where the name stands
	Cast     int      // its cast number (report.Cast), 0 until known
}

// charModel is the pane's list, as read from a text.
type charModel struct {
	markdown bool
	chars    []charInfo
	scenes   []charScene
	list     charList
}

// find is the index of the character with key (-1: none).
func (m *charModel) find(key string) int {
	return slices.IndexFunc(m.chars, func(c charInfo) bool { return c.Key == key })
}

// screenplayChars reads the characters of a screenplay.
func screenplayChars(text string, starts []string) *charModel {
	m := &charModel{list: charListOf(text)}
	body := text[:bodyEndByte(text)]
	if starts == nil {
		starts = englishScenes
	}
	lines := strings.Split(body, "\n")
	lineStart := make([]int, len(lines)) // in runes
	at := 0
	for i, l := range lines {
		lineStart[i] = at
		at += utf8.RuneCountInString(l) + 1
	}
	// the learned notes' names merged into their cues, as the reports do
	learnedNotes := learnedOf(text)
	aliasOf := map[string]string{}
	aliasesOf := map[string][]string{}
	for _, a := range learnedNotes.Aliases {
		ck := charKey(a.Cue)
		for _, n := range a.Names {
			aliasOf[charKey(n)] = ck
			aliasesOf[ck] = append(aliasesOf[ck], n)
		}
	}
	byKey := map[string]*charInfo{}
	var order []string
	get := func(key, name string) *charInfo {
		c := byKey[key]
		if c == nil {
			c = &charInfo{Name: name, Key: key, Entry: -1}
			byKey[key] = c
			order = append(order, key)
		}
		return c
	}
	labels := spanLabels(text, starts)
	scene := -1
	var speaker *charInfo
	inTitle := false
	for _, row := range fountain.ParseLines(starts, strings.NewReader(body)) {
		switch row.Type {
		case lex.TypeTitlePage:
			inTitle = true
		case lex.TypeNewPage:
			inTitle = false
		}
		if inTitle || row.Source <= 0 || row.Source > len(lines) {
			continue
		}
		line := row.Source - 1
		switch row.Type {
		case lex.TypeScene:
			heading, _ := lex.SceneNumber(strings.TrimPrefix(strings.TrimSpace(lines[line]), "."))
			label, ok := strings.CutPrefix(labels[line], "Scene ")
			if !ok {
				label = fmt.Sprint(len(m.scenes) + 1)
			}
			m.scenes = append(m.scenes, charScene{line: line, heading: strings.TrimSpace(lex.StripNotes(heading)), label: label})
			scene = len(m.scenes) - 1
			speaker = nil
		case lex.TypeSpeaker:
			span, ok := cueSpanOf(lines[line])
			key := charKey(string(row.Contents))
			if !ok || key == "" {
				speaker = nil
				continue
			}
			own := true // the cue is the character's own name, not another of theirs
			if a, ok := aliasOf[key]; ok {
				key, own = a, false
			}
			c := get(key, key)
			if own && c.Name == c.Key && c.Entry < 0 {
				c.Name = span.name // as written (McCLANE), not in capitals
			}
			span.line, span.start, span.end = line, lineStart[line]+span.start, lineStart[line]+span.end
			c.Cues = append(c.Cues, span)
			c.Speeches++
			if scene >= 0 && !slices.Contains(c.Scenes, scene) {
				c.Scenes = append(c.Scenes, scene)
			}
			speaker = c
		case lex.TypeDialog:
			if speaker != nil {
				speaker.Words += len(strings.Fields(lex.Unstyled(lex.StripNotes(string(row.Contents)))))
			}
		case lex.TypeParen, lex.TypeEmpty, lex.TypeDualOpen, lex.TypeDualNext, lex.TypeDualClose, lex.TypeNote:
		default:
			speaker = nil
		}
	}
	first := map[string]int{}
	for i, k := range order {
		first[k] = i
	}
	// the list's characters, those who do not speak yet too, by its spelling
	speakers := len(order)
	for i, e := range m.list.entries {
		k := e.key()
		if k == "" {
			continue
		}
		if a, ok := aliasOf[k]; ok && byKey[k] == nil {
			k = a
		}
		c := get(k, e.Name)
		if c.Entry < 0 {
			c.Entry, c.Name = i, e.Name
		}
		if _, ok := first[k]; !ok {
			first[k] = speakers + i
		}
	}
	for k, names := range aliasesOf {
		if c := byKey[k]; c != nil {
			c.Aliases = names
		}
	}
	for _, c := range byKey {
		c.Not = learnedNotes.isNot(c.Name) && c.Entry < 0
	}
	for _, k := range order {
		m.chars = append(m.chars, *byKey[k])
	}
	// the most present first, as Lexington's cast list orders them (its
	// eighths aside: the pages are not laid out here)
	slices.SortStableFunc(m.chars, func(a, b charInfo) int {
		return cmp.Or(
			cmp.Compare(len(b.Scenes), len(a.Scenes)),
			cmp.Compare(b.Speeches, a.Speeches),
			cmp.Compare(b.Words, a.Words),
			cmp.Compare(first[a.Key], first[b.Key]),
			strings.Compare(a.Key, b.Key),
		)
	})
	return m
}

// cueSpanOf is the name on a cue's line, in runes from the line's start:
// after the indent and @, before an extension, ^ or a note.
func cueSpanOf(line string) (cueSpan, bool) {
	rs := []rune(line)
	i := 0
	for i < len(rs) && unicode.IsSpace(rs[i]) {
		i++
	}
	if i < len(rs) && rs[i] == '@' {
		i++
	}
	j := i
	for j < len(rs) && rs[j] != '(' && rs[j] != '^' && !(rs[j] == '[' && j+1 < len(rs) && rs[j+1] == '[') {
		j++
	}
	for j > i && unicode.IsSpace(rs[j-1]) {
		j--
	}
	if j <= i {
		return cueSpan{}, false
	}
	return cueSpan{start: i, end: j, name: string(rs[i:j])}, true
}

// markdownChars reads the names of a Markdown document, as a story is
// read: who is named, the most weighty first, and where each stands.
func markdownChars(text, lang string) *charModel {
	m := &charModel{markdown: true}
	r := storyread.Read(storyPassages(passagesOf(text)), lang, storyread.Hints{})
	rs := []rune(text)
	for _, c := range r.Characters {
		if c.Name == storyread.Narrator {
			continue
		}
		info := charInfo{Name: c.Name, Key: charKey(c.Name), Entry: -1, Speeches: c.Lines,
			Mentions: wordPlaces(rs, c.Name)}
		m.chars = append(m.chars, info)
	}
	return m
}

// wordPlaces are the runes where name stands in text as a word (or words).
func wordPlaces(rs []rune, name string) []int {
	var out []int
	if name == "" {
		return nil
	}
	want := []rune(name)
	for i := 0; i+len(want) <= len(rs); i++ {
		if rs[i] != want[0] || (i > 0 && isWordRune(rs[i-1])) {
			continue
		}
		if string(rs[i:i+len(want)]) == name && (i+len(want) == len(rs) || !isWordRune(rs[i+len(want)])) {
			out = append(out, i)
			i += len(want) - 1
		}
	}
	return out
}

// places are the runes the pane steps through for the character: its
// cues, or in Markdown the places it is named.
func (c charInfo) places() []int {
	if c.Mentions != nil {
		return c.Mentions
	}
	out := make([]int, len(c.Cues))
	for i, s := range c.Cues {
		out[i] = s.start
	}
	return out
}

// nextPlace is the first of places after cursor, from the start again
// after the last (-1: none).
func nextPlace(places []int, cursor int) int {
	if len(places) == 0 {
		return -1
	}
	for _, p := range places {
		if p > cursor {
			return p
		}
	}
	return places[0]
}

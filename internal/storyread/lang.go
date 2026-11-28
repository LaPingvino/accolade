// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package storyread

import (
	"embed"
	"fmt"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// A language's word lists are a text file each, lang/xx.txt, read when
// the package loads; the format is described at the top of lang/en.txt.

//go:embed lang/*.txt
var langFiles embed.FS

// set is a list of words, for asking whether one is in it.
type set map[string]bool

// phrase is a list entry of more than one word: a scene's shift ("the
// next morning") with its value.
type phrase struct {
	words []string
	value string
}

// Lang is a language's word lists.
type Lang struct {
	Code, Name string
	capNouns   bool // German: every noun is capitalised

	speech       map[string]string // verb: the manner it gives ("" none)
	time         map[string]string // word: the heading's time
	placeBefore  map[string]float64
	shift        []phrase // longest first
	mannerSuffix []string

	thought, titles, prepositions, interior, exterior, chapter, notNames,
	determiners, possessives, articles, pronouns, narrator, particles,
	common, notManner set
}

var langs = mustLoad()

func mustLoad() map[string]*Lang {
	out := map[string]*Lang{}
	files, _ := langFiles.ReadDir("lang")
	for _, f := range files {
		code := strings.TrimSuffix(f.Name(), ".txt")
		b, err := langFiles.ReadFile(path.Join("lang", f.Name()))
		if err != nil {
			panic(err)
		}
		l, err := parseLang(code, string(b))
		if err != nil {
			panic(fmt.Sprintf("storyread: lang/%s: %v", f.Name(), err))
		}
		out[code] = l
	}
	return out
}

// Languages are the codes of the languages with word lists, in order.
func Languages() []string {
	var out []string
	for c := range langs {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Language is the word lists for a language code ("pt"), or English's
// for one without lists of its own.
func Language(code string) *Lang {
	if l := langs[strings.ToLower(code)]; l != nil {
		return l
	}
	return langs["en"]
}

// Has reports whether a language has word lists of its own.
func Has(code string) bool { return langs[strings.ToLower(code)] != nil }

// key is an entry as the text's words are compared: its words lowered,
// apostrophes made straight, one space between them.
func key(entry string) string {
	var words []string
	for _, t := range tokenize(entry) {
		words = append(words, t.low)
	}
	return strings.Join(words, " ")
}

func parseLang(code, text string) (*Lang, error) {
	l := &Lang{Code: code, speech: map[string]string{}, time: map[string]string{}, placeBefore: map[string]float64{}}
	sets := map[string]*set{
		"thought": &l.thought, "titles": &l.titles, "prepositions": &l.prepositions,
		"interior": &l.interior, "exterior": &l.exterior, "chapter": &l.chapter,
		"not-names": &l.notNames, "determiners": &l.determiners, "possessives": &l.possessives,
		"articles": &l.articles, "pronouns": &l.pronouns, "narrator": &l.narrator,
		"particles": &l.particles, "common": &l.common, "not-manner": &l.notManner,
	}
	for _, s := range sets {
		*s = set{}
	}
	section := ""
	for n, line := range strings.Split(text, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}
		entry, value, _ := strings.Cut(line, "=")
		entry, value = strings.TrimSpace(entry), strings.TrimSpace(value)
		k := key(entry)
		switch section {
		case "name":
			l.Name = line
		case "options":
			if k == "capitalised-nouns" {
				l.capNouns = value == "yes"
			}
		case "speech":
			l.speech[k] = value
		case "time":
			l.time[k] = value
		case "place-before":
			w := 1.0
			if value != "" {
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return nil, fmt.Errorf("line %d: weight %q", n+1, value)
				}
				w = f
			}
			l.placeBefore[k] = w
		case "shift":
			l.shift = append(l.shift, phrase{words: strings.Fields(k), value: value})
		case "manner-suffix":
			l.mannerSuffix = append(l.mannerSuffix, strings.ToLower(entry))
		default:
			s, ok := sets[section]
			if !ok {
				return nil, fmt.Errorf("line %d: no section [%s]", n+1, section)
			}
			(*s)[k] = true
		}
	}
	slices.SortStableFunc(l.shift, func(a, b phrase) int { return len(b.words) - len(a.words) })
	if l.Name == "" {
		l.Name = code
	}
	return l, nil
}

// isSpeech reports whether a word is a speech verb, and the manner it
// gives: the word itself, or its first part ("dit" of dit-il), or its last
// ("écria" of s'écria).
func (l *Lang) isSpeech(t *token) (manner string, ok bool) {
	if m, ok := l.speech[t.low]; ok {
		return m, true
	}
	for _, part := range t.parts[min(1, len(t.parts)):] {
		if m, ok := l.speech[part]; ok {
			return m, true
		}
	}
	if len(t.parts) > 1 {
		if m, ok := l.speech[t.parts[0]]; ok {
			return m, true
		}
	}
	return "", false
}

// in reports whether a word, or its first part, is in the list.
func (s set) in(t *token) bool {
	return s[t.low] || len(t.parts) > 1 && (s[t.parts[0]] || s[t.parts[len(t.parts)-1]])
}

// Detect guesses the language of a text from its frequent words: the
// code, and whether it is sure (enough of them, well ahead of the next).
func Detect(text string) (code string, sure bool) {
	best, rate, second := detectScore(text)
	return best, rate*6000 >= 8 && rate >= detectRate && rate >= 1.3*second
}

// detectRate is how many of a text's words, at least, are its language's
// frequent ones for Detect to be sure: prose in a listed language has a
// quarter or more of them; another language shares a few by chance.
const detectRate = 0.15

// detectScore is Detect's evidence, for calibrating: the best language's
// share of the words, and the runner-up's.
func detectScore(text string) (best string, rate, second float64) {
	toks := tokenize(text)
	if len(toks) > 6000 {
		toks = toks[:6000]
	}
	score := map[string]int{}
	for _, t := range toks {
		for c, l := range langs {
			if l.common[t.low] {
				score[c]++
			}
		}
	}
	for _, c := range Languages() {
		switch {
		case best == "" || score[c] > score[best]:
			if best != "" {
				second = float64(score[best])
			}
			best = c
		case float64(score[c]) > second:
			second = float64(score[c])
		}
	}
	n := float64(max(len(toks), 1))
	return best, float64(score[best]) / n, second / n
}

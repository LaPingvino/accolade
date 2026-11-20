package storyread

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// A passage's text is read as words (tokens) and pieces (segments):
// what is said, between quotes or after a dash, and the narration
// around it.

// token is a word of a passage.
type token struct {
	text       string   // as written: Mr, dit-il, l’hôte
	low        string   // lowered, apostrophes straight
	parts      []string // low's parts at hyphens and apostrophes
	start, end int      // bytes, in the passage's text
	before     string   // the text between it and the word before
	after      string   // the text between it and the next word
	cap        bool     // starts with a capital
	sent       bool     // starts a sentence
	seg        int      // the segment it is in
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r)
}

// straight makes apostrophes and hyphens plain.
var straight = strings.NewReplacer("’", "'", "‐", "-")

// tokenize parts a text into its words: letters and digits, with a hyphen
// or apostrophe inside a word kept in it (dit-il, m'nheer, l’hôte).
func tokenize(s string) []token {
	var toks []token
	i := 0
	for i < len(s) {
		r, w := utf8.DecodeRuneInString(s[i:])
		if !isWordRune(r) {
			i += w
			continue
		}
		start := i
		i += w
		for i < len(s) {
			r, w := utf8.DecodeRuneInString(s[i:])
			if isWordRune(r) {
				i += w
				continue
			}
			if r == '-' || r == '\'' || r == '’' || r == '‐' {
				if n, _ := utf8.DecodeRuneInString(s[i+w:]); i+w < len(s) && unicode.IsLetter(n) {
					i += w
					continue
				}
			}
			break
		}
		text := s[start:i]
		low := strings.ToLower(straight.Replace(text))
		first, _ := utf8.DecodeRuneInString(text)
		toks = append(toks, token{text: text, low: low, start: start, end: i, cap: unicode.IsUpper(first),
			parts: strings.FieldsFunc(low, func(r rune) bool { return r == '-' || r == '\'' })})
	}
	for k := range toks {
		if k > 0 {
			toks[k].before = s[toks[k-1].end:toks[k].start]
		} else {
			toks[k].before = s[:toks[k].start]
		}
		if k+1 < len(toks) {
			toks[k].after = s[toks[k].end:toks[k+1].start]
		} else {
			toks[k].after = s[toks[k].end:]
		}
	}
	return toks
}

// segment is a stretch of a passage: said (speech) or narration.
type segment struct {
	start, end int // bytes
	speech     bool
	open       bool // a quote not closed by the passage's end
}

// dashMarks are what a line of dialogue starts with in the dash style.
var dashMarks = []string{"—", "―", "–", "--", "-"}

// dashStart is where a dash-style line of dialogue's words start, if the
// text is one: "— Bonjour", "—Hola", "--Continue", "-Para mí" (a plain
// hyphen only before a capital, ¿ or ¡).
func dashStart(s string) (int, bool) {
	t := strings.TrimLeft(s, " \t")
	lead := len(s) - len(t)
	for _, m := range dashMarks {
		if !strings.HasPrefix(t, m) {
			continue
		}
		rest := strings.TrimLeft(t[len(m):], "-")
		if m == "-" {
			r, _ := utf8.DecodeRuneInString(rest)
			if !unicode.IsUpper(r) && r != '¿' && r != '¡' {
				return 0, false
			}
		}
		after := strings.TrimLeft(rest, " \t")
		return lead + len(t) - len(after), after != ""
	}
	return 0, false
}

// openers are the quotes that open what is said, with those that close
// it: “English” "straight" «French» „German“ »German« ‘single’.
var openers = map[rune]string{'“': "”", '"': `"`, '«': "»", '„': "“”", '»': "«", '‚': "‘’", '‘': "’"}

// quoteSegments parts a passage at its quotes. Single quotes count only
// in a passage without double ones (else they are quotes inside quotes,
// or apostrophes).
func quoteSegments(s string) []segment {
	var out []segment
	hasDouble := strings.ContainsAny(s, "“”\"«»„")
	in, closers, from := false, "", 0
	add := func(end int, speech, open bool) {
		if strings.TrimSpace(s[from:end]) != "" {
			out = append(out, segment{start: from, end: end, speech: speech, open: open})
		}
	}
	prev := ' '
	for i, r := range s {
		next, _ := utf8.DecodeRuneInString(s[i+utf8.RuneLen(r):])
		switch {
		case !in:
			cl, ok := openers[r]
			if !ok || (r == '‘' || r == '‚') && hasDouble || r == '‘' && unicode.IsLetter(prev) {
				break
			}
			add(i, false, false)
			in, closers, from = true, cl, i+utf8.RuneLen(r)
		case strings.ContainsRune(closers, r):
			if r == '’' && unicode.IsLetter(next) {
				break // an apostrophe
			}
			add(i, true, false)
			in, from = false, i+utf8.RuneLen(r)
		}
		prev = r
	}
	add(len(s), in, in)
	return out
}

// dashSegments parts a dash-style passage from where its words start: a
// dash inside it changes from what is said to narration and back
// ("—Hola —dijo Juan—. ¿Cómo estás?").
func dashSegments(s string, from int) []segment {
	var out []segment
	speech := true
	add := func(end int) {
		if strings.TrimSpace(s[from:end]) != "" {
			out = append(out, segment{start: from, end: end, speech: speech})
		}
	}
	for i := from; i < len(s); {
		n := 0
		switch {
		case strings.HasPrefix(s[i:], "--"):
			n = len(s[i:]) - len(strings.TrimLeft(s[i:], "-"))
		case strings.HasPrefix(s[i:], "—"), strings.HasPrefix(s[i:], "―"):
			n = len("—")
		case strings.HasPrefix(s[i:], "–") && (i == 0 || s[i-1] == ' ' || i+3 < len(s) && s[i+3] == ' '):
			n = len("–")
		}
		if n == 0 {
			_, w := utf8.DecodeRuneInString(s[i:])
			i += w
			continue
		}
		add(i)
		speech = !speech
		i += n
		from = i
	}
	add(len(s))
	return out
}

// sentenceEnd reports whether text between words ends a sentence.
func sentenceEnd(between string) bool { return strings.ContainsAny(between, ".!?…:;") }

// fullStop reports whether text between words ends a sentence for
// capitals: not a semicolon.
func fullStop(between string) bool { return strings.ContainsAny(between, ".!?…:") }

// trimLine is what is said as a line: without its quotes, dashes and
// the spaces around.
func trimLine(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "“”\"«»„‚‘’—―–- \t"))
}

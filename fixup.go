// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/fountain"
)

// The pattern fixer (LOOP.md, "SCREENPLAY SOURCES", Joop 2026-11-23: "easy
// tools available to fix this up … with a pattern or format matcher that
// should be easy to fix at once"). An imported script comes out of a PDF
// or a scan with its own conventions ("Scene 6 - int. day -" headings,
// names glued to their parentheticals, blank lines inside paragraphs): the
// writer picks one such line, Accolade proposes the pattern it fits in
// plain words ("Scene <number> - <text>", a regular expression under it),
// shows every line it matches, and makes them all scene headings, cues,
// transitions or narration, or drops them as page debris — previewed
// first, applied in one undo step, the rule kept in the script's
// /* Fix-Up */ block to apply again after a re-import. Nothing changes
// without the writer's choice.
//
// This file is the part without Fyne: patterns, matching, what each
// action makes of the lines, and the rules as the block keeps them.
// fixup_dialog.go is the dialog.

// fixAction is what a rule makes of the lines it matches.
type fixAction string

const (
	fixHeading    fixAction = "heading"
	fixCue        fixAction = "cue"
	fixSplitCue   fixAction = "split cue"
	fixTransition fixAction = "transition"
	fixNarration  fixAction = "narration"
	fixDrop       fixAction = "drop"
	fixJoin       fixAction = "join"
	fixHyphens    fixAction = "hyphens"
	fixSpaces     fixAction = "spaces"
)

// fixActions are the actions in the dialog's order, with their labels.
var fixActions = []struct {
	action fixAction
	label  string
}{
	{fixHeading, "Scene headings"},
	{fixCue, "Character cues"},
	{fixSplitCue, "Split glued cues NAME(paren)"},
	{fixTransition, "Transitions"},
	{fixNarration, "Narration (V.O.)"},
	{fixDrop, "Drop them (page debris)"},
	{fixJoin, "Join across blank lines"},
	{fixHyphens, "Join words broken by hyphens"},
	{fixSpaces, "Collapse runs of spaces"},
}

// label is the action's label in the dialog.
func (a fixAction) label() string {
	for _, f := range fixActions {
		if f.action == a {
			return f.label
		}
	}
	return string(a)
}

// templated: the action writes its result from a template.
func (a fixAction) templated() bool {
	return a == fixHeading || a == fixCue || a == fixTransition || a == fixNarration
}

// fixRule is one rule: what to make of the lines its pattern matches.
// Pattern is in plain words; Regex, if set, was written by hand and is
// used instead.
type fixRule struct {
	Action   fixAction
	Pattern  string
	Regex    string
	Template string
}

// The placeholders of a plain-words pattern, and what each matches.
var fixTokens = []struct {
	name, re, about string
}{
	{"<number>", `(\d+[A-Za-z]?)`, "a number (12, 12A)"},
	{"<int/ext>", `((?i:int\.?\s*/\s*ext|ext\.?\s*/\s*int|i/e|int|ext|est))\b\.?`, "INT, EXT, INT./EXT., I/E (with or without the point)"},
	{"<CAPITALS>", `([^\p{Ll}]*?\p{Lu}[^\p{Ll}]*?)`, "words in capitals"},
	{"<NAME>", `(\p{Lu}[\p{Lu}\p{N}'’.&-]*(?:\s+\p{Lu}[\p{Lu}\p{N}'’.&-]*){0,3})`, "a name in capitals, one to four words (DR. BECKER)"},
	{"<Name>", `(\p{Lu}[\p{L}'’.-]*(?:\s+\p{Lu}[\p{L}'’.-]*){0,3})`, "one to four capitalised words (Mrs. Schultz)"},
	{"<unfinished>", `(.*?[^.!?:;"”’)\]…\s])`, "text that does not end a sentence"},
	{"<spaces>", `([ \t]{2,})`, "two spaces or more"},
	{"<text>", `(.+?)`, "any text"},
}

// fixPlaceholders are the placeholders' names, for the dialog's hint.
func fixPlaceholders() string {
	var p []string
	for _, t := range fixTokens {
		p = append(p, t.name+" "+t.about)
	}
	return strings.Join(p, "; ")
}

// patternRegex is the regular expression of a plain-words pattern, matched
// against a whole line (its force marks taken off, fixBody), and the names
// of its groups in order ("text", "text 2", …).
func patternRegex(pattern string) (string, []string) {
	var b strings.Builder
	var names []string
	seen := map[string]int{}
	b.WriteString(`^\s*`)
	rest := strings.TrimSpace(pattern)
	lit := ""
	flush := func() {
		b.WriteString(literalRegex(lit))
		lit = ""
	}
	for rest != "" {
		tok := -1
		for i, t := range fixTokens {
			if strings.HasPrefix(rest, t.name) {
				tok = i
				break
			}
		}
		if tok < 0 {
			r, n := utf8.DecodeRuneInString(rest)
			lit += string(r)
			rest = rest[n:]
			continue
		}
		flush()
		t := fixTokens[tok]
		rest = rest[len(t.name):]
		re := t.re
		if strings.TrimSpace(rest) == "" { // the last: as much as there is
			re = strings.Replace(re, "*?", "*", -1)
			re = strings.Replace(re, "+?", "+", -1)
		}
		b.WriteString(re)
		name := strings.Trim(t.name, "<>")
		seen[name]++
		if seen[name] > 1 {
			name = fmt.Sprintf("%s %d", name, seen[name])
		}
		names = append(names, name)
	}
	flush()
	b.WriteString(`\s*$`)
	return b.String(), names
}

// literalRegex is a pattern's literal text as a regular expression: words
// in any case, any dash for a dash, spaces as any spaces (none needed next
// to punctuation).
func literalRegex(lit string) string {
	var b strings.Builder
	rs := []rune(lit)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case unicode.IsSpace(r):
			j := i
			for j < len(rs) && unicode.IsSpace(rs[j]) {
				j++
			}
			before, after := i > 0 && wordRune(rs[i-1]), j < len(rs) && wordRune(rs[j])
			if before && after {
				b.WriteString(`\s+`)
			} else {
				b.WriteString(`\s*`)
			}
			i = j - 1
		case r == '-' || r == '–' || r == '—':
			b.WriteString(`[-–—]`)
		case unicode.IsLetter(r) && unicode.ToLower(r) != unicode.ToUpper(r):
			b.WriteString("[" + regexp.QuoteMeta(string(unicode.ToLower(r))) + regexp.QuoteMeta(string(unicode.ToUpper(r))) + "]")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}

func wordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// compiled is the rule's regular expression and its groups' names.
func (r fixRule) compiled() (*regexp.Regexp, []string, error) {
	if r.Regex != "" {
		re, err := regexp.Compile(r.Regex)
		if err != nil {
			return nil, nil, err
		}
		var names []string
		for i, n := range re.SubexpNames()[1:] {
			if n == "" {
				n = fmt.Sprintf("%d", i+1)
			}
			names = append(names, n)
		}
		return re, names, nil
	}
	src, names := patternRegex(r.Pattern)
	re, err := regexp.Compile(src)
	return re, names, err
}

// fixBody is a line without its force marks (!action, .heading, @cue,
// >transition, ~lyric) and the space around it; a centred line keeps its
// > <, which say how it was set.
func fixBody(line string) string {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, ">") && strings.HasSuffix(t, "<") {
		return t
	}
	switch {
	case strings.HasPrefix(t, "!"), strings.HasPrefix(t, "@"), strings.HasPrefix(t, ">"), strings.HasPrefix(t, "~"):
		t = t[1:]
	case strings.HasPrefix(t, ".") && !strings.HasPrefix(t, ".."):
		t = t[1:]
	}
	return strings.TrimSpace(t)
}

// plainBody is fixBody without a centred line's > < too: the text itself.
func plainBody(line string) string {
	t := fixBody(line)
	if strings.HasPrefix(t, ">") && strings.HasSuffix(t, "<") && len(t) >= 2 {
		t = strings.TrimSpace(t[1 : len(t)-1])
	}
	return t
}

// fixLines is a script split for the fixer: its body's lines, which of
// them the rules may touch (not the title page, not a boneyard), and the
// text after the body (its end blocks), kept as it is.
type fixLines struct {
	lines []string
	open  []bool
	tail  string
}

// splitForFix splits a script for the fixer.
func splitForFix(text string) fixLines {
	_, body := fountain.EndBlocks(text)
	f := fixLines{lines: strings.Split(text[:body], "\n"), tail: text[body:]}
	f.open = make([]bool, len(f.lines))
	inBone, title := false, titlePageStart(f.lines)
	for i, l := range f.lines {
		if title {
			if strings.TrimSpace(l) == "" {
				title = false
			}
			continue
		}
		t := strings.TrimSpace(l)
		if inBone {
			if strings.Contains(t, "*/") {
				inBone = false
			}
			continue
		}
		if strings.HasPrefix(t, "/*") {
			inBone = !strings.Contains(t[2:], "*/")
			continue
		}
		f.open[i] = true
	}
	return f
}

// titleKey is a title page's first line: "Title: …", "Author: …", in
// any language's letters ("Titolo: …", "片名: …").
var titleKey = regexp.MustCompile(`^\p{L}[\p{L}\p{M}' ]{0,20}:\s*\S`)

// titlePageStart: the script starts with a title page.
func titlePageStart(lines []string) bool {
	return len(lines) > 0 && titleKey.MatchString(lines[0]) && !strings.HasSuffix(strings.TrimSpace(lines[0]), "TO:")
}

// text is the script again.
func (f fixLines) text() string { return strings.Join(f.lines, "\n") + f.tail }

// fixMatch is a line a rule matches: its index and what its groups hold,
// by name, "line" the whole line.
type fixMatch struct {
	line   int
	groups map[string]string
}

// matches are the lines the rule matches.
func (f fixLines) matches(r fixRule) ([]fixMatch, error) {
	re, names, err := r.compiled()
	if err != nil {
		return nil, err
	}
	var out []fixMatch
	for i, l := range f.lines {
		if !f.open[i] || strings.TrimSpace(l) == "" {
			continue
		}
		// a centred line keeps its > <: a pattern for it says so
		m := re.FindStringSubmatch(fixBody(l))
		if m == nil {
			continue
		}
		g := map[string]string{"line": plainBody(l)}
		for k, n := range names {
			if k+1 < len(m) {
				g[n] = strings.TrimSpace(m[k+1])
			}
		}
		out = append(out, fixMatch{i, g})
	}
	return out, nil
}

// fixChange is one place a rule changes: lines [from, to) become after.
type fixChange struct {
	from, to int
	after    []string
}

// before is what the change replaces.
func (c fixChange) before(f fixLines) []string { return f.lines[c.from:c.to] }

// fixEnv is what the actions need to know of the script: the words its
// scene headings start with (INT, EXT …) and the cue for narration.
type fixEnv struct {
	scenes []string
}

// changes are what the rule changes, in order, none overlapping.
func (f fixLines) changes(r fixRule, env fixEnv) ([]fixChange, error) {
	ms, err := f.matches(r)
	if err != nil {
		return nil, err
	}
	words := map[string]int{}
	if r.Action == fixHyphens {
		words = wordCounts(f)
	}
	var out []fixChange
	next := 0 // the first line not taken by a change yet
	for _, m := range ms {
		if m.line < next {
			continue
		}
		c, ok := f.change(r, m, env, words)
		if !ok || c.from < next {
			continue
		}
		if slices.Equal(f.lines[c.from:c.to], c.after) {
			continue // nothing to do
		}
		out = append(out, c)
		next = c.to
	}
	return out, nil
}

// blank: the line is empty, or only spaces.
func blank(s string) bool { return strings.TrimSpace(s) == "" }

// fillTemplate is the template with the match's groups put in.
func fillTemplate(template string, m fixMatch) string {
	if template == "" {
		template = "<line>"
	}
	out := template
	// the longer names first: <text 2> before <text>
	names := make([]string, 0, len(m.groups))
	for n := range m.groups {
		names = append(names, n)
	}
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			if len(names[j]) > len(names[i]) {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, n := range names {
		v := m.groups[n]
		if n != "line" {
			v = strings.Trim(v, " -–—:,;")
		}
		out = strings.ReplaceAll(out, "<"+n+">", v)
	}
	return strings.Join(strings.Fields(out), " ")
}

// sceneStart is a heading's scene word at its start, written as Fountain
// reads it: "EXT – STREET" is "EXT. STREET".
var sceneStart = regexp.MustCompile(`^(?i)(INT\.?\s*/\s*EXT|EXT\.?\s*/\s*INT|I/E|INT|EXT|EST)(?:\.|\b)\s*[-–—:]?\s*`)

// headingText is a heading as Fountain reads it: in capitals, its scene
// word as "INT. ", forced with a point if it starts with none of the
// script's scene words.
func headingText(s string, env fixEnv) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if m := sceneStart.FindStringSubmatch(s); m != nil {
		if rest := strings.TrimSpace(s[len(m[0]):]); rest != "" && !strings.HasPrefix(rest, "#") {
			w := strings.ReplaceAll(strings.ReplaceAll(m[1], " ", ""), ".", "")
			switch a, b, two := strings.Cut(w, "/"); {
			case w == "I/E":
				w = "I/E."
			case two:
				w = a + "./" + b + "."
			default:
				w += "."
			}
			s = w + " " + rest
		}
	}
	for _, w := range append(env.scenes, "INT", "EXT", "EST", "I/E") {
		w = strings.ToUpper(strings.TrimSuffix(strings.TrimSpace(w), "."))
		if w != "" && strings.HasPrefix(s, w) && len(s) > len(w) && strings.ContainsRune(". /", rune(s[len(w)])) {
			return s
		}
	}
	return "." + s
}

// cueExtension: a parenthetical that belongs with the name (V.O.).
var cueExtension = regexp.MustCompile(`^(?i)(v\.?\s?o\.?|o\.?\s?s\.?|o\.?\s?c\.?|cont['’]?d\.?|cont\.?|off|on phone|filtered|pre-?lap)$`)

// gluedParen is a name with a parenthetical after it: "THE WOMAN(very calmly)".
var gluedParen = regexp.MustCompile(`^(.*?\S)\s*\(([^()]*)\)\s*$`)

// cueLines is a cue as Fountain reads it: the name in capitals (forced
// with @ if it has none), a parenthetical after it on its own line unless
// it is an extension (V.O., CONT'D).
func cueLines(s string) []string {
	s = strings.TrimSpace(s)
	var paren string
	if m := gluedParen.FindStringSubmatch(s); m != nil && !cueExtension.MatchString(strings.TrimSpace(m[2])) {
		s, paren = m[1], "("+strings.TrimSpace(m[2])+")"
	}
	name := strings.ToUpper(s)
	if !hasUpper(name) {
		name = "@" + name
	}
	if paren != "" {
		return []string{name, paren}
	}
	return []string{name}
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// speechLine is a line of what follows a cue, its marks taken off: a
// centred line's > <, a forced action's !.
func speechLine(s string) string { return plainBody(s) }

// change is what the rule makes of one match, and whether it does.
//
//nolint:gocyclo // one case an action
func (f fixLines) change(r fixRule, m fixMatch, env fixEnv, words map[string]int) (fixChange, bool) {
	i := m.line
	lines := f.lines
	c := fixChange{from: i, to: i + 1}
	// blankBefore: a blank line before the result, if the line before is not
	blankBefore := func() []string {
		if i > 0 && !blank(lines[i-1]) {
			return []string{""}
		}
		return nil
	}
	blankAfter := func(at int) []string {
		if at < len(lines) && !blank(lines[at]) {
			return []string{""}
		}
		return nil
	}
	switch r.Action {
	case fixHeading:
		c.after = append(append(blankBefore(), headingText(fillTemplate(r.Template, m), env)), blankAfter(i+1)...)
	case fixTransition:
		t := strings.ToUpper(fillTemplate(r.Template, m))
		if !strings.HasSuffix(t, "TO:") {
			t = "> " + t
		}
		c.after = append(append(blankBefore(), t), blankAfter(i+1)...)
	case fixCue, fixSplitCue, fixNarration:
		var cue []string
		at := i // the speech's first line
		if r.Action == fixNarration {
			t := r.Template
			if t == "" {
				t = "NARRATOR (V.O.)"
			}
			cue = cueLines(fillTemplate(t, m))
		} else {
			cue = cueLines(fillTemplate(r.Template, m))
			at = i + 1
			// the blank lines between a cue and its speech go
			k := at
			for k < len(lines) && blank(lines[k]) && f.open[k] {
				k++
			}
			if k > at && k < len(lines) && f.open[k] && speechCan(lines[k]) {
				at = k
			}
		}
		j := at
		var speech []string
		for j < len(lines) && f.open[j] && !blank(lines[j]) {
			speech = append(speech, speechLine(lines[j]))
			j++
			// a parenthetical alone, then a blank line: the speech is after it
			if r.Action != fixNarration && j < len(lines) && blank(lines[j]) && allParens(speech) {
				k := j
				for k < len(lines) && blank(lines[k]) && f.open[k] {
					k++
				}
				if k < len(lines) && f.open[k] && speechCan(lines[k]) && !allParens([]string{plainBody(lines[k])}) {
					j = k
				}
			}
		}
		if len(speech) == 0 && r.Action != fixNarration {
			// a cue with nothing after it: the name alone
			c.after = append(blankBefore(), cue...)
			return c, true
		}
		c.to = j
		c.after = append(append(blankBefore(), cue...), speech...)
	case fixDrop:
		c.after = nil
		// one blank line left where the line was between two
		if (i == 0 || blank(lines[i-1])) && i+1 < len(lines) && blank(lines[i+1]) {
			c.to = i + 2
		}
	case fixJoin:
		if !continues(lines[i]) {
			return c, false // a heading, a cue: not a paragraph's
		}
		joined := strings.TrimRight(lines[i], " \t")
		j := i
		for j+2 < len(lines) && blank(lines[j+1]) && !blank(lines[j+2]) && f.open[j+2] && continues(lines[j+2]) &&
			(j == i || unfinished(joined)) {
			joined += " " + strings.TrimSpace(speechLine(lines[j+2]))
			j += 2
		}
		if j == i {
			return c, false
		}
		c.to, c.after = j+1, []string{joined}
	case fixHyphens:
		t := lines[i]
		j := i
		// across lines: a word ending "-" at a line's end, the next going on in lower case
		for j+1 < len(lines) && f.open[j+1] && hyphenEnd.MatchString(t) && startsLower(lines[j+1]) {
			t = strings.TrimRight(t, " \t") + "\x00" + strings.TrimSpace(speechLine(lines[j+1]))
			j++
		}
		t = strings.ReplaceAll(t, "\x00", " ")
		t = dehyphenate(t, words)
		c.to, c.after = j+1, []string{t}
	case fixSpaces:
		c.after = []string{collapseSpaces(lines[i])}
	default:
		return c, false
	}
	return c, true
}

// speechCan: a line that can be a speech's, after a cue and a blank
// line: not a heading, a transition, a section or a synopsis.
func speechCan(s string) bool {
	t := strings.TrimSpace(s)
	b := plainBody(s)
	switch {
	case t == "", strings.HasPrefix(t, ".") && !strings.HasPrefix(t, ".."), strings.HasPrefix(t, "#"), strings.HasPrefix(t, "="):
		return false
	case b == strings.ToUpper(b) && (sceneStart.MatchString(b) || transitionTo.MatchString(b)):
		return false
	}
	return true
}

// allParens: the lines are all parentheticals.
func allParens(ls []string) bool {
	for _, l := range ls {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "(") || !strings.HasSuffix(t, ")") {
			return false
		}
	}
	return len(ls) > 0
}

// unfinished: a line that does not end a sentence.
func unfinished(s string) bool {
	t := strings.TrimRight(s, " \t")
	if t == "" {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(t)
	return !strings.ContainsRune(`.!?:;"”’)]…`, r)
}

// continues: a line that can go on a paragraph, not one that starts an
// element of its own (a heading, a cue, a transition, a centred line).
func continues(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" || strings.HasPrefix(t, ".") && !strings.HasPrefix(t, "..") || strings.HasPrefix(t, "@") ||
		strings.HasPrefix(t, ">") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "=") || strings.HasPrefix(t, "[[") {
		return false
	}
	if sceneStart.MatchString(t) && t == strings.ToUpper(t) {
		return false
	}
	// a name alone in capitals: a cue
	if b := plainBody(s); b == strings.ToUpper(b) && hasUpper(b) && len([]rune(b)) <= 30 && !strings.ContainsAny(b, ".!?,") {
		return false
	}
	return true
}

// hyphenEnd is a line ending in a word broken with a hyphen.
var hyphenEnd = regexp.MustCompile(`\p{L}-\s*$`)

// startsLower: the line's text starts with a lower-case letter.
func startsLower(s string) bool {
	r, _ := utf8.DecodeRuneInString(plainBody(s))
	return unicode.IsLower(r)
}

// brokenWord is a word broken at a line's end and joined with a space:
// "sol- diers" (an import's lines put together).
var brokenWord = regexp.MustCompile(`(\p{L}+)- (\p{L}+)`)

// dehyphenate joins the words broken at line ends in s, the script's own
// words saying how: one it has whole elsewhere is joined ("soldiers"),
// one whose second part it has as a word of its own keeps its hyphen
// ("self-control"), the rest are joined. Only where the second part goes
// on in the same case (a lower-case one after letters, or both in
// capitals).
func dehyphenate(s string, words map[string]int) string {
	return brokenWord.ReplaceAllStringFunc(s, func(w string) string {
		m := brokenWord.FindStringSubmatch(w)
		a, b := m[1], m[2]
		br, _ := utf8.DecodeRuneInString(b)
		upper := a == strings.ToUpper(a) && b == strings.ToUpper(b)
		if !unicode.IsLower(br) && !upper {
			return w // "Anna- Maria", a dash before a name
		}
		whole := strings.ToLower(a + b)
		switch {
		case words[whole] > 0:
			return a + b
		case words[strings.ToLower(a)+"-"+strings.ToLower(b)] > 0, words[strings.ToLower(b)] > 0 && len([]rune(b)) >= 3:
			return a + "-" + b
		}
		return a + b
	})
}

// wordRe is a word, with the hyphens inside it.
var wordRe = regexp.MustCompile(`\p{L}+(?:-\p{L}+)*`)

// wordCounts are the script's words (lower case), and its hyphenated
// ones too, as dehyphenate asks.
func wordCounts(f fixLines) map[string]int {
	out := map[string]int{}
	for i, l := range f.lines {
		if !f.open[i] {
			continue
		}
		// the broken words are not words: "sol- diers" counts neither
		l = brokenWord.ReplaceAllString(l, " ")
		for _, w := range wordRe.FindAllString(l, -1) {
			w = strings.ToLower(w)
			out[w]++
			if strings.Contains(w, "-") {
				for _, p := range strings.Split(w, "-") {
					out[p]++
				}
			}
		}
	}
	return out
}

// spaceRun is two spaces or more between words.
var spaceRun = regexp.MustCompile(`(\S)[ \t]{2,}`)

// collapseSpaces makes the runs of spaces inside a line one space, its
// indent kept.
func collapseSpaces(s string) string {
	indent := len(s) - len(strings.TrimLeft(s, " \t"))
	return s[:indent] + strings.TrimRight(spaceRun.ReplaceAllString(s[indent:], "$1 "), " \t")
}

// apply is the script with the changes made.
func (f fixLines) apply(cs []fixChange) fixLines {
	out := fixLines{tail: f.tail}
	at := 0
	for _, c := range cs {
		for ; at < c.from; at++ {
			out.lines = append(out.lines, f.lines[at])
			out.open = append(out.open, f.open[at])
		}
		for _, a := range c.after {
			out.lines = append(out.lines, a)
			out.open = append(out.open, true)
		}
		at = c.to
	}
	for ; at < len(f.lines); at++ {
		out.lines = append(out.lines, f.lines[at])
		out.open = append(out.open, f.open[at])
	}
	return out
}

// applyRules is text with the rules applied one after the other, and how
// many places each changed.
func applyRules(text string, rules []fixRule, env fixEnv) (string, []int) {
	f := splitForFix(text)
	counts := make([]int, len(rules))
	for k, r := range rules {
		cs, err := f.changes(r, env)
		if err != nil {
			continue
		}
		counts[k] = len(cs)
		f = f.apply(cs)
	}
	return f.text(), counts
}

// --- proposing a pattern from a line ---

var (
	onlyNumber   = regexp.MustCompile(`^\(?\d+[A-Za-z]?\.?\)?$`)
	keywordNum   = regexp.MustCompile(`^(\p{L}+\.?)\s*(\d+[A-Za-z]?)\s*([-–—:.,]*)\s*(.*)$`)
	numberFirst  = regexp.MustCompile(`^(\d+[A-Za-z]?)(\s*[.):-]*\s*)(.*)$`)
	intExtFirst  = regexp.MustCompile(`^(?i)(int\.?\s*/\s*ext|ext\.?\s*/\s*int|i/e|int|ext|est)\b\.?(\s*[-–—:]?\s*)(.*)$`)
	trailingNum  = regexp.MustCompile(`\s*#[^#\s]+#\s*$`)
	parenAtEnd   = regexp.MustCompile(`^(.*?)(\s*)\(([^()]*)\)$`)
	nameLike     = regexp.MustCompile(`^\p{Lu}[\p{L}'’.-]*(?:\s+\p{Lu}[\p{L}'’.-]*){0,3}$`)
	quotedStart  = regexp.MustCompile(`^["“„«]`)
	endPunct     = regexp.MustCompile(`([-–—:.,;]+)$`)
	transitionTo = regexp.MustCompile(`(?i)\bTO:$`)
	pageWord     = regexp.MustCompile(`^(?i)(page|p\.|pg\.?|seite|página|pagina|paĝo)\s*\d+\.?$`)
)

// proposeRule is the rule Accolade proposes for an example line: the
// pattern it fits, in plain words, and what to make of the lines like it.
// lines are the script's, to see how often the line itself comes back.
//
//nolint:gocyclo // one case a kind of line
func proposeRule(example string, f fixLines) fixRule {
	body := fixBody(example)
	text := plainBody(example)
	centred := body != text
	r := fixRule{Action: fixHeading}
	wrap := func(p string) string {
		if centred {
			return ">" + p + "<"
		}
		return p
	}
	upper := text == strings.ToUpper(text) && hasUpper(text)
	repeats := 0
	for k, l := range f.lines {
		if f.open[k] && plainBody(l) == text {
			repeats++
		}
	}
	tail := ""
	if m := trailingNum.FindString(text); m != "" && !onlyNumber.MatchString(text) {
		tail = " #<number>#"
		text = strings.TrimSpace(strings.TrimSuffix(text, m))
	}
	switch {
	case text == "":
		r.Action, r.Pattern = fixJoin, "<unfinished>"
	case onlyNumber.MatchString(text):
		// a page number
		p := "<number>"
		if strings.HasSuffix(text, ".") {
			p += "."
		}
		if strings.HasPrefix(text, "(") {
			p = "(" + p + ")"
		}
		r.Action, r.Pattern = fixDrop, wrap(p)
	case spaceRun.MatchString(strings.TrimSpace(example)):
		r.Action, r.Pattern = fixSpaces, "<text><spaces><text>"
	case upper && transitionTo.MatchString(text):
		r.Action, r.Pattern = fixTransition, wrap("<CAPITALS> TO:")
	case pageWord.MatchString(text):
		r.Action, r.Pattern = fixDrop, wrap(strings.Fields(text)[0]+" <number>")
	case keywordNum.MatchString(text) && !intExtFirst.MatchString(text):
		// "Scene 6 - int. day -": the word and the number, then anything
		m := keywordNum.FindStringSubmatch(text)
		p := m[1] + " <number>"
		if m[3] != "" {
			p += " " + m[3]
		}
		if m[4] != "" {
			p += " <text>"
		}
		r.Pattern = wrap(p + tail)
		r.Template = "<text> #<number>#"
		if m[4] == "" {
			r.Template = "<line>"
		}
	case numberFirst.MatchString(text) && hasLetterRune(text):
		m := numberFirst.FindStringSubmatch(text)
		rest := m[3]
		sep := strings.TrimSpace(m[2])
		p := "<number>" + sep + " "
		if im := intExtFirst.FindStringSubmatch(rest); im != nil {
			// "3. EXT – STREET – DAY": the number, the scene word, the rest
			p += "<int/ext>"
			if d := strings.TrimSpace(im[2]); d != "" {
				p += " " + d
			}
			p += " <text>"
			r.Template = "<int/ext> <text> #<number>#"
		} else if rest == strings.ToUpper(rest) && hasUpper(rest) {
			// "39 A GLASS OF BEER—": a numbered shot
			p += "<CAPITALS>"
			if e := endPunct.FindString(rest); e != "" {
				p += e
			}
			r.Template = "<CAPITALS> #<number>#"
		} else {
			p += "<text>"
			r.Template = "<text> #<number>#"
		}
		r.Pattern = wrap(strings.TrimSpace(p) + tail)
	case intExtFirst.MatchString(text):
		im := intExtFirst.FindStringSubmatch(text)
		p := "<int/ext>"
		if d := strings.TrimSpace(im[2]); d != "" {
			p += " " + d
		}
		r.Pattern, r.Template = wrap(p+" <text>"+tail), "<line>"
	case quotedStart.MatchString(text) && !upper:
		// a quoted speech without a cue: narration
		r.Action, r.Pattern, r.Template = fixNarration, wrap(string([]rune(text)[:1])+"<text>"), "NARRATOR (V.O.)"
	case parenAtEnd.MatchString(text) && len([]rune(text)) <= 60:
		m := parenAtEnd.FindStringSubmatch(text)
		name, gap, inside := strings.TrimSpace(m[1]), m[2], m[3]
		switch {
		case name == "":
			// "(MORE)", "(CONT'D)": page furniture; else a parenthetical
			if inside == strings.ToUpper(inside) && hasUpper(inside) {
				r.Action, r.Pattern = fixDrop, wrap(text)
			} else {
				r.Action, r.Pattern = fixJoin, "<unfinished>"
			}
		case name == strings.ToUpper(name) && hasUpper(name):
			r.Action, r.Pattern = fixCue, wrap("<NAME>"+gap+"(<text>)")
			if gap == "" {
				r.Action = fixSplitCue
			}
		case nameLike.MatchString(name):
			r.Action, r.Pattern = fixCue, wrap("<Name>"+gap+"(<text>)")
		default:
			r.Action, r.Pattern = fixJoin, "<unfinished>"
		}
	case upper && repeats >= 3 && !centred:
		// the same line over and over: "VOICE OVER", "CONTINUED"
		r.Pattern = wrap(text)
		r.Action = fixDrop
		if len([]rune(text)) <= 30 && !strings.ContainsAny(text, ".:") {
			r.Action, r.Template = fixCue, "<line>"
			if voiceOver.MatchString(text) {
				r.Template = "NARRATOR (V.O.)"
			}
		}
	case upper && len([]rune(text)) <= 30 && !strings.ContainsAny(text, ".!?,:") && endPunct.FindString(text) == "":
		// a name in capitals, alone (centred, or not)
		r.Action, r.Pattern = fixCue, wrap("<NAME>")
	case upper:
		// a line in capitals: a shot ("A FRENCH GUN—"), or a heading of another kind
		p := "<CAPITALS>"
		if e := endPunct.FindString(text); e != "" {
			p += e
		}
		r.Pattern, r.Template = wrap(p+tail), "<line>"
	case nameLike.MatchString(text) && len(strings.Fields(text)) <= 4:
		r.Action, r.Pattern = fixCue, wrap("<Name>")
	default:
		r.Action, r.Pattern = fixJoin, "<unfinished>"
		if !unfinished(text) {
			r.Action, r.Pattern = fixHyphens, "<text>"
		}
	}
	if r.Template == "" && r.Action.templated() {
		r.Template = "<line>"
	}
	return r
}

// voiceOver is a narrator's line: VOICE OVER, V.O., NARRATOR.
var voiceOver = regexp.MustCompile(`^(?i)(voice[ -]?over|v\.?o\.?|narrat(or|ion)|narrador|rakontanto)$`)

func hasLetterRune(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// defaultPattern is the pattern an action starts from when the writer
// chooses it for a line whose own pattern does not suit it.
func defaultPattern(a fixAction) string {
	switch a {
	case fixJoin:
		return "<unfinished>"
	case fixHyphens:
		return "<text>"
	case fixSpaces:
		return "<text><spaces><text>"
	}
	return ""
}

// --- the rules as the script keeps them ---

// fixUpMarks are the /* Fix-Up */ block's.
var fixUpMarks = endMarks(fountain.EndFixUp)

// String is the rule as the block keeps it, one line: "heading: Scene
// <number> - <text> → <text> #<number>#"; a regular expression written by
// hand between slashes.
func (r fixRule) String() string {
	p := r.Pattern
	if r.Regex != "" {
		p = "/" + r.Regex + "/"
	}
	s := string(r.Action) + ": " + p
	if r.Action.templated() && r.Template != "" {
		s += " → " + r.Template
	}
	return s
}

// parseFixRule reads a rule's line; ok false if it is not one.
func parseFixRule(line string) (fixRule, bool) {
	a, rest, ok := strings.Cut(strings.TrimSpace(line), ": ")
	if !ok {
		return fixRule{}, false
	}
	r := fixRule{Action: fixAction(a)}
	if r.Action.label() == a {
		return fixRule{}, false // not an action
	}
	if r.Action.templated() {
		if p, t, ok := strings.Cut(rest, " → "); ok {
			rest, r.Template = p, strings.TrimSpace(t)
		}
	}
	rest = strings.TrimSpace(rest)
	if len(rest) >= 2 && strings.HasPrefix(rest, "/") && strings.HasSuffix(rest, "/") {
		r.Regex = rest[1 : len(rest)-1]
	} else {
		r.Pattern = rest
	}
	return r, true
}

// fixRulesOf are the rules the script keeps.
func fixRulesOf(text string) []fixRule {
	inside, ok := fixUpInside(text)
	if !ok {
		return nil
	}
	var out []fixRule
	for _, l := range strings.Split(inside, "\n") {
		if r, ok := parseFixRule(l); ok {
			out = append(out, r)
		}
	}
	return out
}

// fixUpInside is what the script's Fix-Up block holds.
func fixUpInside(text string) (string, bool) {
	b := tailSetOf(text).fix
	return b.inside, b.ok
}

// withFixRule is where the Fix-Up block goes in text (runes) and the block
// with the rule added (one the same already there is not added again).
func withFixRule(text string, r fixRule) (start, end int, block string) {
	b := tailSetOf(text).fix
	var lines []string
	if b.ok && b.inside != "" {
		lines = strings.Split(b.inside, "\n")
	}
	line := r.String()
	for _, l := range lines {
		if strings.TrimSpace(l) == line {
			return withTailBlock(text, b, fixUpMarks, b.inside)
		}
	}
	return withTailBlock(text, b, fixUpMarks, strings.Join(append(lines, line), "\n"))
}

// changedSpan is where two texts differ, in runes: [start, end) of old
// becomes mid.
func changedSpan(old, nw string) (start, end int, mid string) {
	o, n := []rune(old), []rune(nw)
	p := 0
	for p < len(o) && p < len(n) && o[p] == n[p] {
		p++
	}
	s := 0
	for s < len(o)-p && s < len(n)-p && o[len(o)-1-s] == n[len(n)-1-s] {
		s++
	}
	return p, len(o) - s, string(n[p : len(n)-s])
}

// lineRanges are the lines' places in text, in runes, [start, end).
func lineRanges(lines []string, want []int) [][2]int {
	starts := make([]int, len(lines)+1)
	for i, l := range lines {
		starts[i+1] = starts[i] + utf8.RuneCountInString(l) + 1
	}
	out := make([][2]int, 0, len(want))
	for _, i := range want {
		out = append(out, [2]int{starts[i], starts[i] + utf8.RuneCountInString(lines[i])})
	}
	return out
}

package main

import (
	"strings"

	"github.com/LaPingvino/lexington/fountain"
	"github.com/LaPingvino/lexington/lex"
)

// A script in another language than English is a dialect of Fountain:
// INT. and EXT. are part of Fountain, so "EN. DOMO - TAGO" is a scene
// heading only to Accolade and Lexington, which read the script with
// Esperanto's words (Joop, 2026-11-20). Where another Fountain app reads
// the text — a plain .fountain, a bundle's text.fountain — it is written
// as standard Fountain: every element only the language's rules see gets
// Fountain's force mark, and the marks are taken back when Accolade reads
// it again.
//
// The language changes only scene headings (Lexington's Scenes; " TO:"
// transitions and capitals for a character are the same in every
// language), both ways:
//   - a heading only the language knows gets a dot: .EN. DOMO - TAGO;
//   - a line English would take for a heading but the language does not
//     ("INT. DOMO" in Esperanto) keeps what it is with ! (action) or @
//     (a character). Dialogue has no force mark: such a line in a speech
//     stays as it is, and English reads it as a heading.
//
// Taking them back undoes exactly those marks, and only where the line
// is then what the mark said: a dot or ! the writer put where the
// language needs none (".EN. DOMO" in Esperanto) is the same text either
// way, so it goes too; any other mark stays.

// englishFountain is lang's text as standard Fountain.
func englishFountain(text, lang string) string {
	if !dialect(lang) {
		return text
	}
	types := lineTypes(text, sceneHeaders(lang))
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		t := types[i+1]
		if forceMarked(line) {
			continue
		}
		mine, english := sceneIn(line, lang), sceneIn(line, "en")
		switch {
		case t == lex.TypeScene && mine && !english:
			lines[i] = "." + line
		case t == lex.TypeAction && english:
			lines[i] = "!" + line
		case t == lex.TypeSpeaker && english:
			lines[i] = "@" + line
		}
	}
	return strings.Join(lines, "\n")
}

// dialectFountain is standard Fountain read as lang's text: the marks
// englishFountain gives taken back.
func dialectFountain(text, lang string) string {
	if !dialect(lang) {
		return text
	}
	english := lineTypes(text, sceneHeaders("en"))
	lines := strings.Split(text, "\n")
	// the ! and @ that may go: they go if the line is then what they said
	want := map[int]string{}
	for i, line := range lines {
		t := english[i+1]
		if len(line) < 2 {
			continue
		}
		rest := line[1:]
		switch {
		case line[0] == '.' && rest[0] != '.' && t == lex.TypeScene && !forceMarked(rest) &&
			sceneIn(rest, lang) && !sceneIn(rest, "en"):
			lines[i] = rest // a heading by itself in lang
		case line[0] == '!' && t == lex.TypeAction && !forceMarked(rest) && sceneIn(rest, "en") && !sceneIn(rest, lang):
			want[i] = lex.TypeAction
		case line[0] == '@' && t == lex.TypeSpeaker && !forceMarked(rest) && sceneIn(rest, "en") && !sceneIn(rest, lang):
			want[i] = lex.TypeSpeaker
		}
	}
	for len(want) > 0 { // one taken back can change the next (a speaker's speech): until none changes
		try := append([]string(nil), lines...)
		for i := range want {
			try[i] = lines[i][1:]
		}
		types := lineTypes(strings.Join(try, "\n"), sceneHeaders(lang))
		kept := false
		for i, t := range want {
			if types[i+1] != t {
				delete(want, i) // it is needed: it stays
				kept = true
			}
		}
		if !kept {
			lines = try
			break
		}
	}
	return strings.Join(lines, "\n")
}

// dialect tells whether lang is a language whose scripts are a dialect
// of Fountain: a known one other than English.
func dialect(lang string) bool {
	return lang != "" && lang != "en" && sceneConf[lang] != nil
}

// forceMarked tells whether a line starts with one of Fountain's force
// marks for an element (Lexington's CheckForce, and the forced heading).
func forceMarked(line string) bool {
	return line != "" && strings.ContainsRune("@~!.", rune(line[0]))
}

// sceneIn tells whether Lexington reads line as a scene heading in lang
// by its words (CheckScene; notes before it set apart as the parser
// does).
func sceneIn(line, lang string) bool {
	row := strings.TrimRight(line, " \t\r")
	if strings.Contains(row, "[[") {
		for {
			r := strings.TrimLeft(row, " \t")
			if !strings.HasPrefix(r, "[[") {
				break
			}
			end := strings.Index(r, "]]")
			if end < 0 {
				break
			}
			row = r[end+2:]
		}
		row = strings.TrimLeft(row, " \t")
	}
	upper := strings.ToUpper(row)
	for _, w := range sceneHeaders(lang) {
		if fountain.StartsScene(upper, w) {
			return true
		}
	}
	return false
}

// lineTypes are what Lexington reads each line of text as, read with
// scenes' heading words, by line (counted from 1): the element of the row
// it starts; title page fields and the boneyard as their own.
func lineTypes(text string, scenes []string) map[int]string {
	out := map[int]string{}
	for _, row := range fountain.ParseLines(scenes, strings.NewReader(text)) {
		switch row.Type {
		case lex.TypeDualOpen, lex.TypeDualNext, lex.TypeDualClose, lex.TypeTitlePage, lex.TypeNewPage, "metasection":
			continue
		}
		if _, ok := out[row.Source]; !ok && row.Source > 0 {
			out[row.Source] = row.Type
		}
	}
	return out
}

package main

import (
	"sync"

	"slices"

	"github.com/LaPingvino/lexington/fountain"

	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LaPingvino/lexington/rules"
)

// A script's language decides what its scene headings start with: INT.
// and EXT. in English, EN. and EKST. in Esperanto, BIN. and BUI. in
// Dutch. Lexington's configuration has them; Accolade takes the language
// from the file's name, as Lexington does (scene.eo.fountain is
// Esperanto), or for a new script from the preferences.

var sceneConf = lexSettings.Scenes

// A lexConf is Lexington's configuration as one script reads it: the
// machine's (lexSettings), and for a script that brought its own
// (an .accolade, lexington_carried.go) that laid over it. The functions
// below without one read the machine's; a window's script reads its own
// (MainWindow.lex).
type lexConf struct {
	rules.TOMLConf
	// carried is the script's own part, nil for none
	carried *carriedConf
}

// machineLex is the machine's configuration.
func machineLex() lexConf { return lexConf{TOMLConf: lexSettings} }

// knows tells whether lang is a language with scene headings.
func (c lexConf) knows(lang string) bool { return c.Knows(lang) }

// scriptLanguages are the languages with scene headings, in order.
func scriptLanguages() []string { return machineLex().languages() }

func (c lexConf) languages() []string {
	return c.LanguageNames()
}

// languageOf a file's name ("scene.eo.fountain": "eo"), or "" if it does
// not say.
func languageOf(path string) string {
	if strings.EqualFold(filepath.Ext(path), rules.HekayaExt) {
		return rules.Hekaya // Hekaya's file: Arabic Fountain, its own rules
	}
	parts := strings.Split(filepath.Base(path), ".")
	if len(parts) < 3 {
		return ""
	}
	if l := strings.ToLower(parts[len(parts)-2]); sceneConf(l) != nil {
		return l
	}
	return ""
}

// sceneHeaders are a language's scene heading words, as Lexington's
// parser takes them ("INT", "EXT"); English for an unknown language.
func sceneHeaders(lang string) []string { return machineLex().headers(lang) }

func (c lexConf) headers(lang string) []string {
	if h := c.Scenes(lang); h != nil {
		return h
	}
	if h := c.Scenes("en"); h != nil {
		return h
	}
	return rules.DefaultConf().Scenes("en")
}

// sceneStarts are what a language's scene headings start with, for
// telling them apart while typing: each word with a dot or a space.
func sceneStarts(lang string) []string { return machineLex().starts(lang) }

func (c lexConf) starts(lang string) []string {
	var out []string
	for _, w := range c.headers(lang) {
		h := strings.TrimSuffix(strings.ToUpper(w), ".")
		if last, _ := utf8.DecodeLastRuneInString(h); h != "" && !unicode.IsLetter(last) && !unicode.IsDigit(last) {
			out = append(out, h) // a sign needs nothing after it: ○喫茶店
			continue
		}
		if strings.HasSuffix(w, ".") {
			out = append(out, h+".") // EN. only with its dot: "En la domo" is action
			continue
		}
		out = append(out, h+".", h+" ")
	}
	return out
}

// isSceneHeadingIn reports whether a line starts like a scene heading of
// the language whose starts are given (nil: English).
func isSceneHeadingIn(line string, starts []string) bool {
	if starts == nil {
		starts = sceneStarts("en")
	}
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, p := range starts {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	// a heading by one of the language's patterns ("场景1 小区楼下 日 外")
	if d := dialectForStarts(starts); d != nil && len(d.Lang.Headings) > 0 {
		return d.Heading(strings.TrimSpace(line))
	}
	return false
}

// startsDialects are the dialects of the languages by their starts
// (dialectForStarts), forgotten when the settings change.
var startsDialects sync.Map

// dialectForStarts is the dialect of the language whose scene starts
// these are (the script's), nil for none the machine knows.
func dialectForStarts(starts []string) *fountain.Dialect {
	key := strings.Join(starts, "\x00")
	if d, ok := startsDialects.Load(key); ok {
		return d.(*fountain.Dialect)
	}
	var d *fountain.Dialect
	for _, l := range append(scriptLanguages(), rules.Hekaya) {
		if slices.Equal(sceneStarts(l), starts) {
			d = fountain.NewDialect(lexSettings.Language(l))
			break
		}
	}
	startsDialects.Store(key, d)
	return d
}

// allSceneHeaders are every language's scene heading starts.
func allSceneHeaders() []string {
	var all []string
	for _, l := range scriptLanguages() {
		all = append(all, sceneConf(l)...)
	}
	return all
}

// headingsLanguage is the language most of a script's scene headings are
// in: of those that tie (INT. is English, German and French), the
// preferred one, else English.
func headingsLanguage(text, preferred string) string {
	count := map[string]int{}
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimPrefix(strings.TrimSpace(line), ".")
		for _, l := range scriptLanguages() {
			for _, s := range sceneStarts(l) {
				if strings.HasPrefix(strings.ToUpper(t), s) {
					count[l]++
					break
				}
			}
		}
	}
	most := 0
	for _, n := range count {
		most = max(most, n)
	}
	for _, l := range append([]string{preferred, "en"}, scriptLanguages()...) {
		if count[l] == most {
			return l
		}
	}
	return preferred
}

// langFor is the language whose scene words scenes are, with all its
// rules (cue, transition, heading patterns: Lexington's rules.Language),
// for the code that is handed a language's scene words: the window's
// language first, then any the machine knows; English's rules with
// those words for none.
func langFor(scenes []string) rules.Language {
	if scenes == nil {
		return lexSettings.Language("en")
	}
	for _, l := range append(scriptLanguages(), rules.Hekaya) {
		if slices.Equal(lexSettings.Scenes(l), scenes) {
			return lexSettings.Language(l)
		}
	}
	return rules.Words(scenes)
}

// dialect is the script's language's rules, ready to read lines with.
func (w *MainWindow) dialect() *fountain.Dialect {
	lang := "en"
	if w != nil && w.language != "" {
		lang = w.language
	}
	return fountain.NewDialect(w.lex().Language(lang))
}

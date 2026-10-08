package main

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/LaPingvino/lexington/rules"
)

// A script's language decides what its scene headings start with: INT.
// and EXT. in English, EN. and EKST. in Esperanto, BIN. and BUI. in
// Dutch. Lexington's configuration has them; Accolade takes the language
// from the file's name, as Lexington does (scene.eo.fountain is
// Esperanto), or for a new script from the preferences.

var sceneConf = rules.DefaultConf().Scenes

// scriptLanguages are the languages with scene headings, in order.
func scriptLanguages() []string {
	var ls []string
	for l := range sceneConf {
		ls = append(ls, l)
	}
	sort.Strings(ls)
	return ls
}

// languageOf a file's name ("scene.eo.fountain": "eo"), or "" if it does
// not say.
func languageOf(path string) string {
	parts := strings.Split(filepath.Base(path), ".")
	if len(parts) < 3 {
		return ""
	}
	if l := strings.ToLower(parts[len(parts)-2]); sceneConf[l] != nil {
		return l
	}
	return ""
}

// sceneHeaders are a language's scene heading words, as Lexington's
// parser takes them ("INT", "EXT"); English for an unknown language.
func sceneHeaders(lang string) []string {
	if h := sceneConf[lang]; h != nil {
		return h
	}
	return sceneConf["en"]
}

// sceneStarts are what a language's scene headings start with, for
// telling them apart while typing: each word with a dot or a space.
func sceneStarts(lang string) []string {
	var out []string
	for _, h := range sceneHeaders(lang) {
		h = strings.TrimSuffix(strings.ToUpper(h), ".")
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
	return false
}

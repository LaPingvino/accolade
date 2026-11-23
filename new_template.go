package main

import (
	"slices"
	"strings"
	"time"

	"github.com/LaPingvino/lexington/rules"
)

// New's text by language (LOOP.md, NEW BY LANGUAGE AND FORMAT, part 2):
// the language's own sample if it has one (sample.go, examples/),
// else the English template with every word bound to the language taken
// from Lexington's rules for it: the title page keys, the scene word and
// time of the heading, the cue's extension, the closing transition. The
// prose stays English, a placeholder; the direction and the cue rules
// are the language's own, as the editor reads it.

// newScriptText is what New starts a script in lang with.
func (w *MainWindow) newScriptText(lang string) string {
	if s := languageSample(lang); s != "" {
		return s
	}
	if base, _, ok := strings.Cut(lang, "-"); ok { // zh-Hant: Chinese's
		if s := languageSample(base); s != "" {
			return s
		}
	}
	if lang == "" || lang == "en" {
		return newScriptTemplate()
	}
	return templateIn(w.lex().Language(lang))
}

// templateIn is the English template in language l's words.
func templateIn(l rules.Language) string {
	key := func(std string) string { // the language's own title key, the shortest
		if l.TitleKeys[std] == std { // written as it is (hi: English first)
			return std
		}
		var own []string
		for k, v := range l.TitleKeys {
			if v == std {
				own = append(own, k)
			}
		}
		if len(own) == 0 {
			return std
		}
		slices.SortFunc(own, func(a, b string) int {
			if d := len([]rune(a)) - len([]rune(b)); d != 0 {
				return d
			}
			return strings.Compare(a, b)
		})
		return own[0]
	}
	first := func(words []string, or string) string {
		for _, w := range words {
			if w != "" {
				return w
			}
		}
		return or
	}
	scene := first(l.Matchup, first(l.Scenes, "INT."))
	heading := scene + " LIVING ROOM - " + first(l.Times, "DAY")
	if !strings.HasSuffix(scene, ".") && strings.ToUpper(scene) != strings.ToLower(scene) && slices.Contains(l.Scenes, scene) {
		heading = scene + ". LIVING ROOM - " + first(l.Times, "DAY") // INT.: its full stop
	}
	end := "> FADE OUT."
	if n := len(l.Transitions); n > 0 && l.Transitions[n-1] != "" {
		end = l.Transitions[n-1] // the lists end on a fade out (淡出, F・O) or a cut
	}
	lines := []string{
		key("Title") + ": UNTITLED SCREENPLAY",
		key("Credit") + ": Written by",
		key("Author") + ": Your Name",
		key("Draft date") + ": " + time.Now().Format("January 2, 2006"),
		key("Contact") + ":",
		"    Your Name",
		"    Your Address",
		"    Your Phone",
		"    Your Email",
		"",
		heading,
		"",
		"A simple room with basic furniture. Light streams through the windows.",
		"",
		"JOHN sits at a desk, typing on a laptop. He looks frustrated.",
		"",
		"JOHN",
		"(sighing)",
		"This screenplay isn't writing itself.",
		"",
	}
	if ext := first(l.Extensions, ""); ext != "" {
		lines = append(lines, "JOHN ("+ext+")", "Neither is the next one.", "")
	}
	return strings.Join(append(lines,
		"He takes a sip of coffee and continues typing.",
		"",
		end,
		"",
	), "\n")
}

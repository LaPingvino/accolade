package main

import (
	"math"
	"strings"

	"github.com/LaPingvino/lexington/layout"
	"github.com/LaPingvino/lexington/rules"

	"github.com/LaPingvino/accolade/internal/editor/syntax"
	"github.com/LaPingvino/accolade/internal/editor/wrap"
)

// The editor lays the script out as Lexington prints it: each element in
// its column, width and alignment from the script format's rules. The
// text itself has no layout spaces (Fountain does not need them).

// elementRules are the Lexington rules of the editor's elements; the
// others are laid out as action.
var elementRules = map[syntax.Kind]string{
	syntax.SceneHeading:  "scene",
	syntax.Character:     "speaker",
	syntax.Parenthetical: "paren",
	syntax.Dialogue:      "dialog",
	syntax.Transition:    "trans",
	syntax.Centered:      "center",
}

// editorFormat is how the editor lays out each element with a set of
// rules, and how many columns wide its page is.
func editorFormat(set rules.Set) (func(syntax.Kind) wrap.Format, int) {
	if set == nil {
		set = rules.Default
	}
	margin := layout.Margin(set)
	columns := func(inches float64) int { return int(math.Round(inches * layout.CharsPerInch)) }
	format := func(key string) wrap.Format {
		r := set.Get(key)
		f := wrap.Format{Indent: columns(r.Left - margin), Width: columns(layout.PageWidth - r.Left - r.Right)}
		if r.Align != "" {
			f.Align = strings.ToUpper(r.Align)[0]
		}
		return f
	}
	action := format("action")
	formats := map[syntax.Kind]wrap.Format{}
	page := action.Indent + action.Width
	for k, key := range elementRules {
		formats[k] = format(key)
		page = max(page, formats[k].Indent+formats[k].Width)
	}
	return func(k syntax.Kind) wrap.Format {
		if f, ok := formats[k]; ok {
			return f
		}
		return action
	}, page
}

// withoutIndents takes the layout spaces off the lines of elements the
// editor lays out itself (Accolade put them in the text before).
func withoutIndents(text string, starts []string) string {
	lines := strings.Split(text, "\n")
	for i, k := range syntax.ClassifyWith(lines, starts) {
		if _, ok := elementRules[k]; ok {
			lines[i] = strings.TrimLeft(lines[i], " \t")
		}
	}
	return strings.Join(lines, "\n")
}

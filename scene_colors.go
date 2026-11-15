package main

import (
	"image/color"
	"strconv"
	"strings"
	"unicode"

	"github.com/LaPingvino/lexington/lex"
)

// Scene and section colors, as Beat has them: a note on the heading line
// ("INT. BARN - DAY [[red]]", "# ACT ONE [[color blue]]", "[[#ff8800]]")
// gives the scene or section its color; lex.NoteColor reads it the way
// Beat does. The outline and the cards show it, and the color picker
// writes it as Beat writes it: "[[red]]" at the end of the line.

// beatColorNames are the colors the picker offers, in Beat's order.
var beatColorNames = []string{"red", "blue", "green", "pink", "magenta", "gray", "purple", "cyan", "teal", "yellow", "orange", "brown"}

// beatPalette is Beat's own RGB for its color names: BeatColors.m in
// lmparppei/Beat (Frameworks/BeatCore/BeatCore/BeatColors, commit
// 521fb62), with its "grey" for gray.
var beatPalette = map[string]color.NRGBA{
	"red":     {239, 0, 73, 255},
	"blue":    {0, 129, 239, 255},
	"green":   {0, 223, 121, 255},
	"pink":    {254, 131, 223, 255},
	"magenta": {236, 0, 140, 255},
	"gray":    {128, 128, 128, 255},
	"grey":    {128, 128, 128, 255},
	"purple":  {181, 32, 218, 255},
	"cyan":    {7, 189, 235, 255},
	"teal":    {12, 224, 227, 255},
	"yellow":  {251, 200, 45, 255},
	"orange":  {255, 161, 13, 255},
	"brown":   {169, 106, 7, 255},
}

// noteRGB is the color a lex.NoteColor color stands for: one of Beat's
// names or "#rrggbb". Anything else (an unknown name, "#zzzzzz") is no
// color.
func noteRGB(name string) (color.NRGBA, bool) {
	if c, ok := beatPalette[name]; ok {
		return c, true
	}
	if len(name) == 7 && name[0] == '#' {
		if v, err := strconv.ParseUint(name[1:], 16, 32); err == nil {
			return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, true
		}
	}
	return color.NRGBA{}, false
}

// withColor is a heading or section line with the color c, written as
// Beat writes it: "[[c]]" at the end of the line, in place of the color
// notes it had (wherever they were, each with the white space in front of
// it); "" or "none" takes them out. Other notes stay where they are. On
// a line with a "[[" that is never closed, the color goes before that.
func withColor(line, c string) string {
	isBlank := func(r rune) bool { return r != '\n' && unicode.IsSpace(r) }
	var b strings.Builder
	rest, unclosed := line, ""
	for {
		open := strings.Index(rest, "[[")
		if open < 0 {
			break
		}
		end := strings.Index(rest[open+2:], "]]")
		if end < 0 { // never closed: no note, and the color goes before it
			rest, unclosed = rest[:open], rest[open:]
			break
		}
		end += open + 4
		if lex.NoteColor(rest[open:end]) == "" { // another kind of note
			b.WriteString(rest[:end])
		} else {
			b.WriteString(strings.TrimRightFunc(rest[:open], isBlank))
			if b.Len() == 0 { // at the start of the line: the space after it goes
				rest = strings.TrimLeftFunc(rest[end:], isBlank)
				continue
			}
		}
		rest = rest[end:]
	}
	b.WriteString(rest)
	out := b.String()
	if c = strings.ToLower(strings.TrimSpace(c)); c == "" || c == "none" {
		return out + unclosed
	}
	if unclosed != "" { // "[[c]]" after it would be read as part of it
		return strings.TrimRightFunc(out, isBlank) + " [[" + c + "]] " + unclosed
	}
	return strings.TrimRightFunc(out, isBlank) + " [[" + c + "]]"
}

// setItemColor gives item i of the script's outline the color c ("" or
// "none": none).
func setItemColor(text string, starts []string, i int, c string) string {
	items := outlineOf(text, starts)
	if i < 0 || i >= len(items) {
		return text
	}
	lines := strings.Split(text, "\n")
	lines[items[i].Line] = withColor(lines[items[i].Line], c)
	return strings.Join(lines, "\n")
}

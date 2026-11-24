package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/LaPingvino/accolade/internal/editor"
)

// The contrast checker: every text colour Accolade draws against every
// surface it is drawn on, and every mark that must be seen against its
// background, with WCAG 2's AA thresholds (4.5:1 for text, 3:1 for large
// text and for the parts of the interface: marks, focus, card edges).
// The tests run it over every theme; the theme editor shows its warnings
// on the colours being set.

const (
	contrastText = 4.5 // normal text (WCAG 1.4.3 AA)
	contrastUI   = 3.0 // large text, and marks, focus and edges (1.4.11)
)

// contrastPair is one colour drawn on another.
type contrastPair struct {
	What   string      // what is drawn, on what
	Keys   string      // the custom theme's keys the pair depends on ("text,background")
	FG, BG color.NRGBA // as seen: the foreground composited over the background
	Ratio  float64
	Min    float64
}

func (p contrastPair) ok() bool { return p.Ratio+1e-9 >= p.Min }

func (p contrastPair) String() string {
	mark := "ok"
	if !p.ok() {
		mark = "LOW"
	}
	return fmt.Sprintf("%-52s %s on %s  %5.2f:1 (min %.1f) %s", p.What, hexOf(p.FG), hexOf(p.BG), p.Ratio, p.Min, mark)
}

// contrastPairs are a theme's pairs in variant v.
func contrastPairs(th fyne.Theme, v fyne.ThemeVariant) []contrastPair {
	c := func(n fyne.ThemeColorName) color.Color { return th.Color(n, v) }
	opaque := func(x color.Color) color.NRGBA { return over(x, hexColor("#ffffff")) }
	window := opaque(c(theme.ColorNameBackground))
	// the surfaces text sits on, each over the window
	type surface struct {
		name string
		keys string
		c    color.NRGBA
	}
	on := func(n fyne.ThemeColorName) color.NRGBA { return over(c(n), window) }
	surfaces := []surface{
		{"the window", "background", window},
		{"an entry", "surface", on(theme.ColorNameInputBackground)},
		{"a menu", "surface", on(theme.ColorNameMenuBackground)},
		{"a dialog", "surface", on(theme.ColorNameOverlayBackground)},
		{"a button", "raised", on(theme.ColorNameButton)},
		{"a header", "surface", on(theme.ColorNameHeaderBackground)},
	}
	if r := editor.Themed(th, colorNameRaised, v, nil); r != nil {
		surfaces = append(surfaces, surface{"a card", "raised", on(colorNameRaised)})
	}
	dialog := on(theme.ColorNameOverlayBackground)
	tinted := func(n fyne.ThemeColorName, a uint8) color.NRGBA {
		x := color.NRGBAModel.Convert(c(n)).(color.NRGBA)
		x.A = a
		return over(x, window)
	}

	var out []contrastPair
	add := func(what, keys string, fg color.Color, bg color.NRGBA, min float64) {
		f := over(fg, bg)
		out = append(out, contrastPair{What: what, Keys: keys, FG: f, BG: bg, Ratio: contrastRatio(f, bg), Min: min})
	}
	fg := c(theme.ColorNameForeground)
	// body text, everywhere it is
	for _, s := range surfaces {
		add("body text on "+s.name, "text,"+s.keys, fg, s.c, contrastText)
	}
	add("body text, selected", "text,accent", fg, over(c(theme.ColorNameSelection), window), contrastText)
	add("body text, hovered", "text,background", fg, over(c(theme.ColorNameHover), window), contrastText)
	add("body text, pressed", "text,background", fg, over(c(theme.ColorNamePressed), window), contrastText)
	add("a tab's label, focused", "text,accent", fg, over(c(theme.ColorNameFocus), window), contrastText)
	add("the status bar's hint", "text,accent", fg, tinted(theme.ColorNamePrimary, 0x40), contrastText)
	add("search highlight", "text,warning", fg, tinted(theme.ColorNameWarning, 0x48), contrastText)
	// secondary text: low-importance labels (hints, the notes pane's kinds), placeholders
	muted := c(theme.ColorNameDisabled)
	for _, s := range surfaces {
		add("secondary text on "+s.name, "muted,"+s.keys, muted, s.c, contrastText)
	}
	add("a note's kind label (Notes pane)", "muted,background", muted, window, contrastText)
	add("placeholder in an entry", "muted,surface", c(theme.ColorNamePlaceHolder), on(theme.ColorNameInputBackground), contrastText)
	add("line numbers", "muted,background", c(theme.ColorNamePlaceHolder), window, contrastText)
	// the ribbon and the tabs
	add("a ribbon key tip", "text,background", window, opaque(fg), contrastText)
	if a := editor.Themed(th, colorNameActive, v, nil); a != nil {
		add("the active tab's label", "active", c(colorNameOnActive), over(a, window), contrastText)
	}
	add("an important button's label", "accent", c(theme.ColorNameForegroundOnPrimary), on(theme.ColorNamePrimary), contrastText)
	add("an important button's label, focused", "accent", c(theme.ColorNameForegroundOnPrimary), over(c(theme.ColorNameFocus), on(theme.ColorNamePrimary)), contrastText)
	add("an important button, focused, against unfocused", "accent", over(c(theme.ColorNameFocus), on(theme.ColorNamePrimary)), on(theme.ColorNamePrimary), 1.1)
	add("an error button's label", "error", c(theme.ColorNameForegroundOnError), on(theme.ColorNameError), contrastText)
	add("a success button's label", "success", c(theme.ColorNameForegroundOnSuccess), on(theme.ColorNameSuccess), contrastText)
	add("a warning button's label", "warning", c(theme.ColorNameForegroundOnWarning), on(theme.ColorNameWarning), contrastText)
	add("accent text (important labels)", "accent,background", c(theme.ColorNamePrimary), window, contrastText)
	add("accent text in a dialog", "accent,surface", c(theme.ColorNamePrimary), dialog, contrastText)
	add("a link", "link,background", c(theme.ColorNameHyperlink), window, contrastText)
	add("a link in a dialog", "link,surface", c(theme.ColorNameHyperlink), dialog, contrastText)
	// the editor
	heading := editor.Themed(th, editor.ColorNameSceneHeading, v, c(theme.ColorNamePrimary))
	note := editor.Themed(th, editor.ColorNameNote, v, blend(opaque(fg), window, 0.45))
	add("scene headings", "scene,background", heading, window, contrastText)
	add("scene headings, selected", "scene,accent", heading, over(c(theme.ColorNameSelection), window), contrastText)
	add("notes in the script", "note,background", note, window, contrastText)
	add("a folded block's summary", "muted,background", editor.Themed(th, editor.ColorNameFold, v, c(theme.ColorNamePlaceHolder)), window, contrastText)
	add("a chip's label", "accent,background", c(theme.ColorNamePrimary), tinted(theme.ColorNamePrimary, 0x28), contrastText)
	add("a warning chip's label", "warning,background", c(theme.ColorNameWarning), tinted(theme.ColorNameWarning, 0x28), contrastText)
	add("cut text (drafts, merge)", "error,background", c(theme.ColorNameError), window, contrastText)
	add("added text (drafts, merge)", "success,background", c(theme.ColorNameSuccess), window, contrastText)
	add("a citation", "success,text", blend(opaque(c(theme.ColorNameSuccess)), opaque(fg), 0.35), window, contrastText)
	add("a source warning", "warning,background", c(theme.ColorNameWarning), window, contrastText)
	// marks and edges
	add("focus ring (accent)", "accent,background", c(theme.ColorNamePrimary), window, contrastUI)
	add("focus ring in an entry", "accent,surface", c(theme.ColorNamePrimary), on(theme.ColorNameInputBackground), contrastUI)
	for _, k := range []openKind{openTodo, openCut, openBeat, openNote, openRevision, openCheck} {
		add(k.String()+"'s card edge", kindKeys(k), kindColorOf(th, v, k), window, contrastUI)
		add(k.String()+"'s margin mark", kindKeys(k), marginColorOf(th, v, k), window, contrastUI)
	}
	add("fold marks", "muted,background", editor.Themed(th, editor.ColorNameFold, v, c(theme.ColorNamePlaceHolder)), window, contrastUI)
	for _, set := range []string{"White", "Blue", "Pink", "Yellow", "Green", "Goldenrod"} {
		add(set+" revision mark", "background", fitInk(revisionInk(set), window), window, contrastUI)
	}
	return out
}

// kindKeys are the custom keys a kind's colour comes from.
func kindKeys(k openKind) string {
	switch k {
	case openTodo:
		return "warning,background"
	case openCut:
		return "accent,background"
	case openBeat:
		return "success,background"
	case openRevision:
		return "background"
	case openCheck:
		return "error,background"
	}
	return "violet,background"
}

// lowPairs are the pairs below their threshold.
func lowPairs(ps []contrastPair) []contrastPair {
	var out []contrastPair
	for _, p := range ps {
		if !p.ok() {
			out = append(out, p)
		}
	}
	return out
}

// lowKeys are the custom keys with a pair below its threshold, and what.
func lowKeys(ps []contrastPair) map[string][]string {
	out := map[string][]string{}
	for _, p := range lowPairs(ps) {
		for _, k := range strings.Split(p.Keys, ",") {
			out[k] = append(out[k], fmt.Sprintf("%s %.1f:1 (needs %.1f)", p.What, p.Ratio, p.Min))
		}
	}
	return out
}

// themeVariants are a theme's variants, as the checker runs them.
func themeVariants(d *themeDef) []fyne.ThemeVariant {
	switch {
	case d.light == nil && d.dark == nil: // Fyne's own: both
		return []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark}
	case d.light == nil:
		return []fyne.ThemeVariant{theme.VariantDark}
	case d.dark == nil:
		return []fyne.ThemeVariant{theme.VariantLight}
	}
	return []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark}
}

func variantName(v fyne.ThemeVariant) string {
	if v == theme.VariantDark {
		return "dark"
	}
	return "light"
}

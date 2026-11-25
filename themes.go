package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor"
)

// Accolade's themes. Each is a set of key colours (background, panels,
// cards, text, accent, the semantic colours, the editor's own) in a light
// and/or a dark variant; every colour Fyne asks for is derived from them,
// so a theme is whole in either variant. A theme with both follows the
// system's light or dark preference (or the "theme-variant" setting);
// one with only one declares itself light-only or dark-only. The custom
// theme is a base theme with some key colours changed (theme_custom.go).

// Accolade's own colour names, beside Fyne's and the editor's.
const (
	colorNameActive   fyne.ThemeColorName = "accolade-active"    // the active tab's fill (none: an underline)
	colorNameOnActive fyne.ThemeColorName = "accolade-on-active" // its text
	colorNameRaised   fyne.ThemeColorName = "accolade-raised"    // cards over the panels
	colorNamePanel    fyne.ThemeColorName = "accolade-panel"     // the side panes over the window
	colorNameNoteMark fyne.ThemeColorName = "accolade-note-mark" // a note's edge in the Notes pane
)

// themePalette is one variant of a theme: its key colours. A zero colour is
// derived from the others (see color).
type themePalette struct {
	dark bool

	Background color.NRGBA // the window
	Surface    color.NRGBA // panels, menus, dialogs
	Raised     color.NRGBA // cards and buttons over the panels
	Text       color.NRGBA // body text and headings
	Muted      color.NRGBA // secondary text
	Accent     color.NRGBA // Fyne's primary: focus, checks, important buttons
	Active     color.NRGBA // the active tab's fill (zero: an accent underline instead)
	Link       color.NRGBA
	Success    color.NRGBA // green: done, beats
	Warning    color.NRGBA // amber: to-dos
	Error      color.NRGBA // red
	Violet     color.NRGBA // notes' edges
	Line       color.NRGBA // separators, borders

	Scene color.NRGBA // the editor's scene headings (zero: the accent)
	Note  color.NRGBA // the editor's notes (zero: the text, dimmed)

	// exact are colours kept as an older theme had them, over the derived.
	exact map[fyne.ThemeColorName]color.Color
}

// themeDef is one of Accolade's themes.
type themeDef struct {
	id, label   string
	light, dark *themePalette // nil: no such variant
	// the chrome's traits
	caps  bool    // small-caps-like labels: short labels in capitals
	mono  bool    // all text in the monospace font
	sizes float32 // padding scale (0: Fyne's)
	round float32 // corner radius (0: Fyne's; <0: square)
}

// both is "light and dark", or which one the theme declares.
func (d *themeDef) modes() string {
	switch {
	case d.light != nil && d.dark != nil:
		return "light and dark"
	case d.dark != nil:
		return "dark only"
	}
	return "light only"
}

func hexColor(s string) color.NRGBA {
	var r, g, b uint8
	fmt.Sscanf(strings.TrimPrefix(s, "#"), "%02x%02x%02x", &r, &g, &b)
	return color.NRGBA{r, g, b, 0xff}
}

// hexOf is c as #rrggbb.
func hexOf(c color.Color) string {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%02x%02x%02x", n.R, n.G, n.B)
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{r, g, b, 0xff} }

// The themes, in the pickers' order. "system" is Fyne's own theme.
var themeDefs = []*themeDef{
	{id: "system", label: "System", light: systemLight(), dark: systemDark()},
	{id: "light", label: "Light", light: oldLight()},
	{id: "dark", label: "Dark", dark: oldDark()},
	{id: "sepia", label: "Sepia", light: oldSepia()},
	{id: "slate", label: "Slate", caps: true, sizes: 0.8, round: 2,
		// the board game's colours (sidenote-wargame's setup.gml)
		dark: &themePalette{dark: true,
			Background: rgb(27, 34, 48), Surface: rgb(37, 46, 64), Raised: rgb(52, 64, 88),
			Text: rgb(238, 234, 224), Muted: rgb(164, 172, 186), Accent: rgb(184, 150, 228),
			Active: rgb(238, 234, 224), Link: rgb(150, 200, 255),
			Success: rgb(92, 190, 132), Warning: rgb(226, 176, 80), Error: rgb(226, 112, 76),
			Violet: rgb(124, 162, 226), Line: rgb(78, 92, 118)},
		light: &themePalette{
			Background: hexColor("#dfe3ea"), Surface: hexColor("#eceff4"), Raised: hexColor("#f8f9fb"),
			Text: hexColor("#1b2230"), Muted: hexColor("#3f4a5e"), Accent: hexColor("#5f429a"),
			Active: hexColor("#252e40"), Link: hexColor("#2c5ea8"), Note: hexColor("#525d72"),
			Success: hexColor("#1f6c44"), Warning: hexColor("#734f00"), Error: hexColor("#a33c20"),
			Violet: hexColor("#4a6fb0"), Line: hexColor("#b4bccb")},
	},
	{id: "contrast", label: "High Contrast", round: -1,
		light: &themePalette{
			Background: hexColor("#ffffff"), Surface: hexColor("#ffffff"), Raised: hexColor("#f0f0f0"),
			Text: hexColor("#000000"), Muted: hexColor("#262626"), Accent: hexColor("#0033b3"),
			Active: hexColor("#000000"), Link: hexColor("#0033b3"),
			Success: hexColor("#006b2d"), Warning: hexColor("#8a5300"), Error: hexColor("#b00020"),
			Violet: hexColor("#5b1fa6"), Line: hexColor("#000000")},
		dark: &themePalette{dark: true,
			Background: hexColor("#000000"), Surface: hexColor("#000000"), Raised: hexColor("#1c1c1c"),
			Text: hexColor("#ffffff"), Muted: hexColor("#e6e6e6"), Accent: hexColor("#ffd400"),
			Active: hexColor("#ffd400"), Link: hexColor("#7fd4ff"),
			Success: hexColor("#3dff7f"), Warning: hexColor("#ffb000"), Error: hexColor("#ff6b6b"),
			Violet: hexColor("#d1a6ff"), Line: hexColor("#ffffff")},
	},
	{id: "pastel", label: "Pastel", round: 10,
		light: &themePalette{
			Background: hexColor("#f8f4fc"), Surface: hexColor("#efe8f8"), Raised: hexColor("#ffffff"),
			Text: hexColor("#3a3350"), Muted: hexColor("#5f5778"), Accent: hexColor("#6c4aab"),
			Active: hexColor("#cfeadd"), Link: hexColor("#33609e"), Note: hexColor("#6a6380"),
			Success: hexColor("#2f7a58"), Warning: hexColor("#7d5208"), Error: hexColor("#ae3f57"),
			Violet: hexColor("#8f70d0"), Line: hexColor("#dcd2ea"),
			Scene: hexColor("#8e3a6c")},
	},
	{id: "terminal", label: "Terminal", mono: true, sizes: 0.85, round: -1,
		dark: &themePalette{dark: true,
			Background: hexColor("#0b0f0a"), Surface: hexColor("#10160e"), Raised: hexColor("#172015"),
			Text: hexColor("#b6f2a4"), Muted: hexColor("#6fa663"), Accent: hexColor("#f2c14e"),
			Active: hexColor("#b6f2a4"), Link: hexColor("#7fd3e8"),
			Success: hexColor("#6ee07a"), Warning: hexColor("#f2c14e"), Error: hexColor("#ff6e5a"),
			Violet: hexColor("#c792ea"), Line: hexColor("#2c3b28")},
	},
}

// System is Fyne's own theme, light or dark as the system is, with what
// read too faintly in it made readable: Fyne draws a low-importance
// label (a hint) in its disabled colour, #e3e3e3 on white (1.3:1).
func systemLight() *themePalette {
	return &themePalette{
		Background: hexColor("#ffffff"), Surface: hexColor("#f5f5f5"), Raised: hexColor("#f5f5f5"),
		Text: hexColor("#565656"), Muted: hexColor("#5c5c5c"), Accent: hexColor("#1a56c4"),
		Success: hexColor("#1e7e34"), Warning: hexColor("#8a5900"), Error: hexColor("#c62828"), Line: hexColor("#e3e3e3"),
		exact: map[fyne.ThemeColorName]color.Color{
			theme.ColorNameInputBackground:     hexColor("#f3f3f3"),
			theme.ColorNameOverlayBackground:   hexColor("#ffffff"),
			theme.ColorNameHeaderBackground:    hexColor("#f9f9f9"),
			theme.ColorNameHover:               color.NRGBA{0, 0, 0, 0x0f},
			theme.ColorNamePressed:             color.NRGBA{0, 0, 0, 0x19},
			theme.ColorNameScrollBar:           color.NRGBA{0, 0, 0, 0x99},
			theme.ColorNameScrollBarBackground: hexColor("#dbdbdb"),
			theme.ColorNameShadow:              color.NRGBA{0, 0, 0, 0x44},
		},
	}
}

func systemDark() *themePalette {
	return &themePalette{dark: true,
		Background: hexColor("#171718"), Surface: hexColor("#28292e"), Raised: hexColor("#28292e"),
		Text: hexColor("#f3f3f3"), Muted: hexColor("#b2b2b2"), Accent: hexColor("#76a6f8"),
		Success: hexColor("#43f436"), Warning: hexColor("#ff9800"), Error: hexColor("#f65a4e"), Line: hexColor("#39393a"),
		exact: map[fyne.ThemeColorName]color.Color{
			theme.ColorNameInputBackground:     hexColor("#202023"),
			theme.ColorNameOverlayBackground:   hexColor("#181d25"),
			theme.ColorNameHeaderBackground:    hexColor("#1b1b1b"),
			theme.ColorNameHover:               color.NRGBA{0xff, 0xff, 0xff, 0x0f},
			theme.ColorNameScrollBar:           color.NRGBA{0xff, 0xff, 0xff, 0x99},
			theme.ColorNameScrollBarBackground: hexColor("#202023"),
			theme.ColorNameShadow:              color.NRGBA{0, 0, 0, 0x77},
			theme.ColorNameSeparator:           hexColor("#000000"),
		},
	}
}

// The themes Accolade had before the set, kept as they were.

func oldLight() *themePalette {
	return &themePalette{
		Background: rgb(255, 255, 255), Surface: rgb(255, 255, 255), Raised: rgb(248, 249, 250),
		Text: rgb(0, 0, 0), Muted: rgb(90, 90, 90), Accent: rgb(52, 96, 140),
		Success: rgb(30, 126, 52), Warning: rgb(138, 89, 0), Error: rgb(220, 53, 69), Line: rgb(200, 200, 200),
		exact: map[fyne.ThemeColorName]color.Color{
			theme.ColorNameHover:           rgb(240, 240, 240),
			theme.ColorNamePressed:         rgb(220, 220, 220),
			theme.ColorNameSelection:       color.NRGBA{52, 96, 140, 60},
			theme.ColorNameScrollBar:       rgb(200, 200, 200),
			theme.ColorNameShadow:          color.NRGBA{0, 0, 0, 50},
			theme.ColorNameInputBorder:     rgb(200, 200, 200),
			theme.ColorNameInputBackground: rgb(255, 255, 255),
		},
	}
}

func oldDark() *themePalette {
	return &themePalette{dark: true,
		Background: rgb(35, 35, 35), Surface: rgb(40, 40, 40), Raised: rgb(50, 50, 50),
		Text: rgb(240, 240, 240), Muted: rgb(170, 170, 170), Accent: rgb(140, 176, 214),
		Success: rgb(100, 220, 130), Warning: rgb(255, 220, 100), Error: rgb(255, 100, 120), Line: rgb(80, 80, 80),
		exact: map[fyne.ThemeColorName]color.Color{
			theme.ColorNameHover:           rgb(60, 60, 60),
			theme.ColorNamePressed:         rgb(80, 80, 80),
			theme.ColorNameSelection:       color.NRGBA{140, 176, 214, 44},
			theme.ColorNameScrollBar:       rgb(80, 80, 80),
			theme.ColorNameShadow:          color.NRGBA{0, 0, 0, 80},
			theme.ColorNameInputBackground: rgb(45, 45, 45),
			theme.ColorNameInputBorder:     rgb(80, 80, 80),
		},
	}
}

func oldSepia() *themePalette {
	return &themePalette{
		Background: rgb(252, 248, 227), Surface: rgb(250, 245, 225), Raised: rgb(248, 242, 220),
		Text: rgb(92, 80, 65), Muted: rgb(100, 87, 68), Accent: rgb(132, 83, 34),
		Success: rgb(82, 106, 26), Warning: rgb(122, 86, 0), Violet: rgb(150, 120, 160), Note: rgb(118, 104, 84), Error: rgb(180, 65, 47), Line: rgb(220, 200, 160),
		exact: map[fyne.ThemeColorName]color.Color{
			theme.ColorNameHover:           rgb(245, 240, 215),
			theme.ColorNamePressed:         rgb(235, 225, 195),
			theme.ColorNameSelection:       color.NRGBA{184, 134, 11, 60},
			theme.ColorNameScrollBar:       rgb(220, 200, 160),
			theme.ColorNameShadow:          color.NRGBA{92, 80, 65, 50},
			theme.ColorNameInputBackground: rgb(255, 252, 235),
			theme.ColorNameInputBorder:     rgb(220, 200, 160),
		},
	}
}

// themeDefByID is the theme of an id (nil: none, as "system").
func themeDefByID(id string) *themeDef {
	for _, d := range themeDefs {
		if d.id == id {
			return d
		}
	}
	return nil
}

// themeChoices are the themes, as the settings name them, and their
// labels (the ribbon's, the themePalette's, :set theme's).
var themeChoices = func() [][2]string {
	var out [][2]string
	for _, d := range themeDefs {
		out = append(out, [2]string{d.id, d.label})
	}
	return append(out, [2]string{customThemeID, "Custom"})
}()

// The settings: "theme" is the id; "theme-variant" is "system", "light"
// or "dark"; "theme-custom" the custom theme (customTheme, JSON).
const (
	themeVariantPref = "theme-variant"
	themeCustomPref  = "theme-custom"
	customThemeID    = "custom"
)

// themeVariantChoices are the light-or-dark choices and their labels.
var themeVariantChoices = [][2]string{{"system", "Follow the System"}, {"light", "Light"}, {"dark", "Dark"}}

// accoladeTheme is a theme as shown: its palettes (a custom theme's
// changed) and the variant chosen (forced false: the system's).
type accoladeTheme struct {
	def         *themeDef
	id          string // the setting's: the def's, or "custom"
	light, dark *themePalette
	forced      bool
	variant     fyne.ThemeVariant
}

// newAccoladeTheme is the theme of an id with the variant setting ("",
// "system", "light", "dark") and the custom theme (for "custom").
func newAccoladeTheme(id, variant string, custom *customTheme) fyne.Theme {
	if id == customThemeID {
		if custom == nil {
			custom = &customTheme{Base: "slate"}
		}
		t := newAccoladeTheme(custom.Base, variant, nil)
		at, ok := t.(*accoladeTheme)
		if !ok { // based on Fyne's own: Light's keys to change
			at = newAccoladeTheme("light", variant, nil).(*accoladeTheme)
			at.light, at.dark = oldLight(), oldDark()
		}
		at.id = customThemeID
		at.light, at.dark = custom.apply(at.light), custom.applyDark(at.dark)
		return at
	}
	d := themeDefByID(id)
	if d == nil || d.light == nil && d.dark == nil {
		switch variant { // Fyne's own, as the system has it or as chosen
		case "light":
			return &variantTheme{theme.DefaultTheme(), theme.VariantLight}
		case "dark":
			return &variantTheme{theme.DefaultTheme(), theme.VariantDark}
		}
		return theme.DefaultTheme()
	}
	t := &accoladeTheme{def: d, id: d.id, light: d.light, dark: d.dark}
	switch variant {
	case "light":
		t.forced, t.variant = true, theme.VariantLight
	case "dark":
		t.forced, t.variant = true, theme.VariantDark
	}
	return t
}

// variantTheme is a theme in one variant whatever the system's.
type variantTheme struct {
	fyne.Theme
	v fyne.ThemeVariant
}

func (t *variantTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(n, t.v)
}

// pal is the themePalette shown for the system's variant v.
func (t *accoladeTheme) pal(v fyne.ThemeVariant) *themePalette {
	if t.forced {
		v = t.variant
	}
	if t.dark != nil && (v == theme.VariantDark || t.light == nil) {
		return t.dark
	}
	if t.light != nil {
		return t.light
	}
	return t.dark
}

func (t *accoladeTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	p := t.pal(v)
	if c := p.color(n); c != nil {
		return c
	}
	own := theme.VariantLight
	if p.dark {
		own = theme.VariantDark
	}
	return theme.DefaultTheme().Color(n, own)
}

func (t *accoladeTheme) Font(s fyne.TextStyle) fyne.Resource {
	if t.def != nil && t.def.mono && !s.Symbol {
		s.Monospace = true
	}
	return theme.DefaultTheme().Font(s)
}

func (t *accoladeTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (t *accoladeTheme) Size(n fyne.ThemeSizeName) float32 {
	s := theme.DefaultTheme().Size(n)
	if t.def == nil {
		return s
	}
	switch n {
	case theme.SizeNamePadding, theme.SizeNameInnerPadding, theme.SizeNameLineSpacing:
		if t.def.sizes > 0 {
			return float32(math.Round(float64(s * t.def.sizes)))
		}
	case theme.SizeNameInputRadius, theme.SizeNameSelectionRadius:
		switch {
		case t.def.round > 0:
			return t.def.round
		case t.def.round < 0:
			return 0 // square
		}
	}
	return s
}

// themeTraits is the shown theme's chrome traits (none for Fyne's).
func themeTraits() *themeDef {
	if a := fyne.CurrentApp(); a != nil {
		if t, ok := a.Settings().Theme().(*accoladeTheme); ok {
			return t.def
		}
	}
	return &themeDef{}
}

// capsLabel is a short label as the theme shows it: in capitals where
// the theme has small-caps-like labels (Slate).
func capsLabel(s string) string {
	if themeTraits().caps {
		return strings.ToUpper(s)
	}
	return s
}

// shownThemeID is the id of the theme shown ("system" for Fyne's own).
func shownThemeID() string {
	if a := fyne.CurrentApp(); a != nil {
		if t, ok := a.Settings().Theme().(*accoladeTheme); ok {
			return t.id
		}
	}
	return "system"
}

// muted is the secondary text's colour (derived when not set).
func (p *themePalette) muted() color.NRGBA { return or(p.Muted, blend(p.Text, p.Background, 0.4)) }

// alpha is c with alpha a.
func alpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

func zero(c color.NRGBA) bool { return c == color.NRGBA{} }

// or is c, or d when c is zero.
func or(c, d color.NRGBA) color.NRGBA {
	if zero(c) {
		return d
	}
	return c
}

// blend mixes a towards b by t (both opaque).
func blend(a, b color.NRGBA, t float64) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-t) + float64(y)*t)) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 0xff}
}

// onColor is black or white, whichever reads better on c, or the
// themePalette's own text or background where that reads as well.
func (p *themePalette) onColor(c color.NRGBA) color.NRGBA {
	best, bestR := hexColor("#000000"), contrastRatio(hexColor("#000000"), c)
	for _, o := range []color.NRGBA{p.Text, p.Background, hexColor("#ffffff")} {
		if r := contrastRatio(o, c); r > bestR+0.01 && (r >= 4.5 || bestR < 4.5) {
			best, bestR = o, r
		}
	}
	// the themePalette's own colours win when they read well enough
	for _, o := range []color.NRGBA{p.Background, p.Text} {
		if contrastRatio(o, c) >= 7 {
			return o
		}
	}
	return best
}

// color is the themePalette's colour of a name (nil: one it does not know).
func (p *themePalette) color(n fyne.ThemeColorName) color.Color {
	if c, ok := p.exact[n]; ok {
		return c
	}
	line := or(p.Line, blend(p.Background, p.Text, 0.22))
	ink := hexColor("#000000")
	if p.dark {
		ink = hexColor("#ffffff")
	}
	switch n {
	case theme.ColorNameBackground:
		return p.Background
	case theme.ColorNameForeground:
		return p.Text
	case theme.ColorNamePrimary:
		return p.Accent
	case theme.ColorNameForegroundOnPrimary:
		return p.onColor(p.Accent)
	case theme.ColorNameHyperlink:
		return or(p.Link, p.Accent)
	case theme.ColorNameFocus: // the accent, shifted: seen on an accent button too
		if p.dark {
			return alpha(blend(p.Accent, p.Background, 0.4), 0x90)
		}
		return alpha(blend(p.Accent, p.Background, 0.3), 0x60)
	case theme.ColorNameHover:
		return alpha(ink, 0x18)
	case theme.ColorNamePressed:
		return alpha(ink, 0x30)
	case theme.ColorNameSelection:
		return alpha(p.Accent, 0x30)
	case theme.ColorNameSeparator:
		return line
	case theme.ColorNameError:
		return p.Error
	case theme.ColorNameForegroundOnError:
		return p.onColor(p.Error)
	case theme.ColorNameWarning:
		return p.Warning
	case theme.ColorNameForegroundOnWarning:
		return p.onColor(p.Warning)
	case theme.ColorNameSuccess:
		return p.Success
	case theme.ColorNameForegroundOnSuccess:
		return p.onColor(p.Success)
	case theme.ColorNameButton:
		return p.Raised
	case theme.ColorNameDisabledButton:
		return blend(p.Surface, p.Background, 0.5)
	case theme.ColorNameDisabled: // a low-importance label's: secondary text, read as such
		return or(p.Muted, blend(p.Text, p.Background, 0.4))
	case theme.ColorNamePlaceHolder:
		return or(p.Muted, blend(p.Text, p.Background, 0.4))
	case theme.ColorNameScrollBar:
		return line
	case theme.ColorNameScrollBarBackground:
		return alpha(ink, 0x0c)
	case theme.ColorNameShadow:
		if p.dark {
			return color.NRGBA{0, 0, 0, 0x60}
		}
		return color.NRGBA{0, 0, 0, 0x30}
	case theme.ColorNameInputBackground:
		if p.dark {
			return p.Background // a well in the panel
		}
		return p.Raised
	case theme.ColorNameInputBorder:
		return line
	case theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground, theme.ColorNameHeaderBackground:
		return p.Surface
	case theme.ColorNameInnerWindowBorder:
		return p.Accent
	case theme.ColorNameInnerWindowBorderInactive:
		return line
	case colorNameActive:
		if zero(p.Active) {
			return color.Transparent
		}
		return p.Active
	case colorNameOnActive:
		if zero(p.Active) {
			return color.Transparent
		}
		return p.onColor(p.Active)
	case colorNameRaised:
		return p.Raised
	case colorNamePanel:
		return p.Surface
	case colorNameNoteMark:
		return or(p.Violet, p.muted())
	case editor.ColorNameSceneHeading:
		return or(p.Scene, p.Accent)
	case editor.ColorNameNote:
		return or(p.Note, p.muted())
	case editor.ColorNameDim:
		return p.muted()
	case editor.ColorNameFold:
		return or(p.Muted, blend(p.Text, p.Background, 0.4))
	}
	return nil
}

// relativeLuminance is WCAG 2's of an opaque colour.
func relativeLuminance(c color.Color) float64 {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	ch := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(n.R) + 0.7152*ch(n.G) + 0.0722*ch(n.B)
}

// over is c composited over the opaque bg.
func over(c, bg color.Color) color.NRGBA {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	b := color.NRGBAModel.Convert(bg).(color.NRGBA)
	return blend(b, color.NRGBA{n.R, n.G, n.B, 0xff}, float64(n.A)/255)
}

// contrastRatio is WCAG 2's contrast of fg on bg (fg's alpha over bg).
func contrastRatio(fg, bg color.Color) float64 {
	a, b := relativeLuminance(over(fg, bg)), relativeLuminance(over(bg, hexColor("#ffffff")))
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}

// fitInk is an ink (a revision's colour) as readable on bg: its hue,
// moved towards white on a dark background or black on a light one until
// it stands out (3:1, as a mark must).
func fitInk(ink, bg color.Color) color.Color {
	c := color.NRGBAModel.Convert(ink).(color.NRGBA)
	b := over(bg, hexColor("#ffffff"))
	to := hexColor("#000000")
	if relativeLuminance(b) < 0.2 {
		to = hexColor("#ffffff")
	}
	for i := 0; i < 8 && contrastRatio(c, b) < 3; i++ {
		c = blend(c, to, 0.15)
	}
	return c
}

// customTheme is the custom theme: a base and the key colours changed,
// for each variant, by key (customKeys) as #rrggbb.
type customTheme struct {
	Base  string            `json:"base"`
	Light map[string]string `json:"light,omitempty"`
	Dark  map[string]string `json:"dark,omitempty"`
}

// customKey is a key colour the custom theme can change.
type customKey struct {
	id, label string
	field     func(p *themePalette) *color.NRGBA
	exact     []fyne.ThemeColorName // an older theme's exact colours it replaces
}

var customKeys = []customKey{
	{"background", "Background", func(p *themePalette) *color.NRGBA { return &p.Background }, []fyne.ThemeColorName{theme.ColorNameHover, theme.ColorNamePressed}},
	{"surface", "Panels and dialogs", func(p *themePalette) *color.NRGBA { return &p.Surface }, []fyne.ThemeColorName{theme.ColorNameInputBackground}},
	{"raised", "Cards and buttons", func(p *themePalette) *color.NRGBA { return &p.Raised }, nil},
	{"text", "Text", func(p *themePalette) *color.NRGBA { return &p.Text }, nil},
	{"muted", "Secondary text", func(p *themePalette) *color.NRGBA { return &p.Muted }, []fyne.ThemeColorName{theme.ColorNameDisabled}},
	{"accent", "Accent", func(p *themePalette) *color.NRGBA { return &p.Accent }, []fyne.ThemeColorName{theme.ColorNameFocus, theme.ColorNameSelection}},
	{"active", "Active tab", func(p *themePalette) *color.NRGBA { return &p.Active }, nil},
	{"link", "Links", func(p *themePalette) *color.NRGBA { return &p.Link }, nil},
	{"line", "Lines and borders", func(p *themePalette) *color.NRGBA { return &p.Line }, []fyne.ThemeColorName{theme.ColorNameScrollBar, theme.ColorNameInputBorder}},
	{"success", "Green (beats, done)", func(p *themePalette) *color.NRGBA { return &p.Success }, nil},
	{"warning", "Amber (to-dos)", func(p *themePalette) *color.NRGBA { return &p.Warning }, nil},
	{"error", "Red (errors)", func(p *themePalette) *color.NRGBA { return &p.Error }, nil},
	{"violet", "Notes' edge", func(p *themePalette) *color.NRGBA { return &p.Violet }, nil},
	{"scene", "Scene headings", func(p *themePalette) *color.NRGBA { return &p.Scene }, nil},
	{"note", "Notes in the script", func(p *themePalette) *color.NRGBA { return &p.Note }, nil},
}

// keyColor is a themePalette's key colour as shown (a zero one derived).
func keyColor(p *themePalette, id string) color.NRGBA {
	switch id {
	case "active":
		return or(p.Active, p.Accent)
	case "link":
		return or(p.Link, p.Accent)
	case "line":
		return or(p.Line, blend(p.Background, p.Text, 0.22))
	case "muted":
		return or(p.Muted, blend(p.Text, p.Background, 0.4))
	case "violet":
		return or(p.Violet, p.muted())
	case "scene":
		return or(p.Scene, p.Accent)
	case "note":
		return or(p.Note, p.muted())
	}
	for _, k := range customKeys {
		if k.id == id {
			return *k.field(p)
		}
	}
	return color.NRGBA{}
}

// changed is a copy of p with the colours of set (key: #rrggbb).
func changed(p *themePalette, set map[string]string) *themePalette {
	if p == nil {
		return nil
	}
	q := *p
	q.exact = map[fyne.ThemeColorName]color.Color{}
	for n, c := range p.exact {
		q.exact[n] = c
	}
	for _, k := range customKeys {
		v, ok := set[k.id]
		if !ok {
			continue
		}
		*k.field(&q) = hexColor(v)
		for _, n := range k.exact {
			delete(q.exact, n)
		}
	}
	return &q
}

func (c *customTheme) apply(p *themePalette) *themePalette     { return changed(p, c.Light) }
func (c *customTheme) applyDark(p *themePalette) *themePalette { return changed(p, c.Dark) }

// loadCustomTheme is the custom theme in the settings (nil: none).
func loadCustomTheme(s *Settings) *customTheme {
	raw := s.GetString(themeCustomPref)
	if raw == "" {
		return nil
	}
	var c customTheme
	if json.Unmarshal([]byte(raw), &c) != nil || c.Base == "" {
		return nil
	}
	return &c
}

// saveCustomTheme keeps the custom theme in the settings.
func saveCustomTheme(s *Settings, c *customTheme) {
	b, _ := json.Marshal(c)
	s.SetString(themeCustomPref, string(b))
}

// themeFromSettings is the theme the settings choose.
func themeFromSettings(s *Settings) fyne.Theme {
	return newAccoladeTheme(s.GetString("theme"), s.GetString(themeVariantPref), loadCustomTheme(s))
}

// paintTab colours a tab (the ribbon's, the dock's): the active one
// filled where the theme has an active colour (Slate's cream), else
// underlined in the accent; a focused or hovered one shaded.
func paintTab(bg, line *canvas.Rectangle, text *canvas.Text, selected, focused, hovered bool) {
	text.TextStyle.Bold = selected
	text.Color = theme.Color(theme.ColorNameForeground)
	line.FillColor = color.Transparent
	active := editor.Themed(fyne.CurrentApp().Settings().Theme(), colorNameActive, fyne.CurrentApp().Settings().ThemeVariant(), nil)
	switch {
	case focused:
		bg.FillColor = theme.Color(theme.ColorNameFocus)
	case selected && active != nil:
		bg.FillColor = active
		text.Color = theme.Color(colorNameOnActive)
	case hovered:
		bg.FillColor = theme.Color(theme.ColorNameHover)
	default:
		bg.FillColor = color.Transparent
	}
	if selected && (active == nil || focused) {
		line.FillColor = theme.Color(theme.ColorNamePrimary)
	}
}

// choiceLabels are the labels of choices (id, label).
func choiceLabels(cs [][2]string) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c[1])
	}
	return out
}

// choiceLabel is the label of an id among choices (the first's for none).
func choiceLabel(cs [][2]string, id string) string {
	for _, c := range cs {
		if strings.EqualFold(c[0], id) {
			return c[1]
		}
	}
	return cs[0][1]
}

// choiceID is the id of a label among choices.
func choiceID(cs [][2]string, label string) string {
	for _, c := range cs {
		if c[1] == label {
			return c[0]
		}
	}
	return cs[0][0]
}

func themeLabels() []string         { return choiceLabels(themeChoices) }
func themeLabel(id string) string   { return choiceLabel(themeChoices, id) }
func themeIDOf(label string) string { return choiceID(themeChoices, label) }

// inVariant is a theme as seen in variant v, whatever is chosen: a view
// that is always paper (the stripboard) or always dark (the table read).
// A theme without that variant gives System's.
func inVariant(th fyne.Theme, v fyne.ThemeVariant) fyne.Theme {
	at, ok := th.(*accoladeTheme)
	if !ok {
		return th
	}
	has := at.light != nil
	if v == theme.VariantDark {
		has = at.dark != nil
	}
	if !has {
		at = newAccoladeTheme("system", "", nil).(*accoladeTheme)
	}
	out := *at
	out.forced, out.variant = true, v
	return &out
}

// themedFill is a rectangle in a colour of the theme, following a change
// of theme (a plain rectangle keeps the colour it was made with). A
// theme without the colour (Fyne's) leaves it clear.
type themedFill struct {
	widget.BaseWidget
	name fyne.ThemeColorName
	min  fyne.Size
}

func newThemedFill(n fyne.ThemeColorName) *themedFill {
	f := &themedFill{name: n}
	f.ExtendBaseWidget(f)
	return f
}

func (f *themedFill) MinSize() fyne.Size { return f.min }

func (f *themedFill) CreateRenderer() fyne.WidgetRenderer {
	r := canvas.NewRectangle(color.Transparent)
	rr := &ribbonRenderer{objects: []fyne.CanvasObject{r}}
	rr.layout = func(s fyne.Size) { r.Resize(s) }
	rr.min = func() fyne.Size { return f.min }
	rr.refresh = func() {
		r.FillColor = color.Transparent
		if c := editor.Themed(fyne.CurrentApp().Settings().Theme(), f.name, fyne.CurrentApp().Settings().ThemeVariant(), nil); c != nil {
			r.FillColor = c
		}
		r.Refresh()
	}
	rr.refresh()
	return rr
}

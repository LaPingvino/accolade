package main

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// TestThemeContrast runs the contrast checker over every theme in every
// variant it has: each text colour on each surface it is drawn on at
// 4.5:1, each mark, focus ring and card edge at 3:1. With
// CONTRAST_REPORT set, the table goes to that file, with the themes as
// they were before (Fyne's own for System) in a column beside.
func TestThemeContrast(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	var report strings.Builder
	for _, d := range themeDefs {
		th := newAccoladeTheme(d.id, "", nil)
		for _, v := range themeVariants(d) {
			pairs := contrastPairs(th, v)
			var before map[string]contrastPair
			if b := beforeTheme(d.id); b != nil {
				before = map[string]contrastPair{}
				for _, p := range contrastPairs(b, v) {
					before[p.What] = p
				}
			}
			fmt.Fprintf(&report, "\n## %s (%s)\n\n| pair | colours | ratio | min |", d.label, variantName(v))
			if before != nil {
				report.WriteString(" before |")
			}
			report.WriteString("\n|---|---|---|---|")
			if before != nil {
				report.WriteString("---|")
			}
			report.WriteString("\n")
			for _, p := range pairs {
				mark := ""
				if !p.ok() {
					mark = " **LOW**"
				}
				fmt.Fprintf(&report, "| %s | %s on %s | %.2f%s | %.1f |", p.What, hexOf(p.FG), hexOf(p.BG), p.Ratio, mark, p.Min)
				if before != nil {
					if b, ok := before[p.What]; ok {
						bm := ""
						if !b.ok() {
							bm = " **LOW**"
						}
						fmt.Fprintf(&report, " %.2f%s |", b.Ratio, bm)
					} else {
						report.WriteString(" – |")
					}
				}
				report.WriteString("\n")
				if !p.ok() {
					t.Errorf("%s (%s): %s", d.label, variantName(v), p)
				}
			}
		}
	}
	if f := os.Getenv("CONTRAST_REPORT"); f != "" {
		if err := os.WriteFile(f, []byte("# Accolade's themes: contrast\n"+report.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// beforeTheme is a theme as it was before the set (nil: a new one).
func beforeTheme(id string) fyne.Theme {
	switch id {
	case "system":
		return theme.DefaultTheme()
	case "light":
		return &beforeLight{}
	case "dark":
		return &beforeDark{}
	case "sepia":
		return &beforeSepia{}
	}
	return nil
}

// Every colour Fyne asks for is the theme's own, in its own variant: a
// dark-only theme on a light system gets no light colour from Fyne's.
func TestThemesDefineEveryColor(t *testing.T) {
	names := []fyne.ThemeColorName{theme.ColorNameBackground, theme.ColorNameButton, theme.ColorNameDisabled,
		theme.ColorNameDisabledButton, theme.ColorNameError, theme.ColorNameFocus, theme.ColorNameForeground,
		theme.ColorNameForegroundOnError, theme.ColorNameForegroundOnPrimary, theme.ColorNameForegroundOnSuccess,
		theme.ColorNameForegroundOnWarning, theme.ColorNameHeaderBackground, theme.ColorNameHover,
		theme.ColorNameHyperlink, theme.ColorNameInnerWindowBorder, theme.ColorNameInnerWindowBorderInactive,
		theme.ColorNameInputBackground, theme.ColorNameInputBorder, theme.ColorNameMenuBackground,
		theme.ColorNameOverlayBackground, theme.ColorNamePlaceHolder, theme.ColorNamePressed, theme.ColorNamePrimary,
		theme.ColorNameScrollBar, theme.ColorNameScrollBarBackground, theme.ColorNameSelection,
		theme.ColorNameSeparator, theme.ColorNameShadow, theme.ColorNameSuccess, theme.ColorNameWarning}
	for _, d := range themeDefs {
		for _, p := range []*themePalette{d.light, d.dark} {
			if p == nil {
				continue
			}
			for _, n := range names {
				if p.color(n) == nil {
					t.Errorf("%s: no %s", d.id, n)
				}
			}
		}
	}
	// a dark-only theme asked for the light variant answers dark
	th := newAccoladeTheme("slate", "dark", nil)
	if c := th.Color(theme.ColorNameBackground, theme.VariantLight); c != hexColor("#1b2230") {
		t.Errorf("Slate forced dark, asked light: %v", c)
	}
	if c := newAccoladeTheme("terminal", "", nil).Color(theme.ColorNameHeaderBackground, theme.VariantLight); relativeLuminance(c) > 0.2 {
		t.Errorf("Terminal's header asked light: %v", c)
	}
}

// The custom theme is its base with the keys changed, and nothing else.
func TestCustomTheme(t *testing.T) {
	c := &customTheme{Base: "dark", Dark: map[string]string{"accent": "#ff8800", "background": "#101010"}}
	th := newAccoladeTheme(customThemeID, "", c)
	if got := th.Color(theme.ColorNamePrimary, theme.VariantDark); got != hexColor("#ff8800") {
		t.Errorf("accent: %v", got)
	}
	if got := th.Color(theme.ColorNameBackground, theme.VariantDark); got != hexColor("#101010") {
		t.Errorf("background: %v", got)
	}
	base := newAccoladeTheme("dark", "", nil)
	if got, want := th.Color(theme.ColorNameForeground, theme.VariantDark), base.Color(theme.ColorNameForeground, theme.VariantDark); got != want {
		t.Errorf("text changed: %v, %v", got, want)
	}
	if base.Color(theme.ColorNamePrimary, theme.VariantDark) == color.Color(hexColor("#ff8800")) {
		t.Error("the base theme changed with the custom one")
	}
}

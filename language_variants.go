package main

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// A language's variants (Joop, 2026-11-23: "hi is best English first,
// with Latn and Deva options as supported but not shown by default"): a
// language whose tag is another's with a script or region (hi-Latn,
// hi-Deva beside hi) is a variant of it. The pickers (New in:, Script
// Language, Rename, the preferences) show the languages without their
// variants, and the variants behind More variants…; everything else
// (a file's name, a bundle's language, the settings editor) knows them all.

// moreVariants is the pickers' entry for the variants.
const moreVariants = "More variants…"

// isVariant tells whether lang is a variant of a language among all.
func isVariant(lang string, all []string) bool {
	base, _, ok := strings.Cut(lang, "-")
	return ok && base != "" && slices.Contains(all, base)
}

// shownLanguages are the languages of all a picker shows at first: those
// that are no variant, and keep (the one picked now) though it is one;
// hidden are the rest.
func shownLanguages(all []string, keep string) (shown, hidden []string) {
	for _, l := range all {
		if isVariant(l, all) && l != keep {
			hidden = append(hidden, l)
		} else {
			shown = append(shown, l)
		}
	}
	return shown, hidden
}

// offerVariants gives a language Select shown's options and More
// variants… (when there are more), which shows all's in their place. Set
// after its OnChanged, which it keeps for every other option.
func offerVariants(sel *widget.Select, shown, all []string) {
	if len(shown) == len(all) {
		sel.SetOptions(all)
		return
	}
	sel.SetOptions(append(slices.Clone(shown), moreVariants))
	changed := sel.OnChanged
	sel.OnChanged = func(v string) {
		if v != moreVariants {
			if changed != nil {
				changed(v)
			}
			return
		}
		sel.OnChanged = nil
		sel.ClearSelected()
		sel.SetOptions(all) // a tap again opens them all
		sel.OnChanged = changed
	}
}

// flatItems are items with a child menu's items in its place (More
// variants…'s, for the command palette and the ex line).
func flatItems(items []*fyne.MenuItem) []*fyne.MenuItem {
	var out []*fyne.MenuItem
	for _, it := range items {
		if it.ChildMenu != nil && it.Action == nil {
			out = append(out, it.ChildMenu.Items...)
			continue
		}
		out = append(out, it)
	}
	return out
}

// selectOption selects v in a language Select, its options all's first
// when v is among the variants not shown.
func selectOption(sel *widget.Select, v string, all []string) {
	if !slices.Contains(sel.Options, v) && slices.Contains(all, v) {
		changed := sel.OnChanged
		sel.OnChanged = nil
		sel.SetOptions(all)
		sel.OnChanged = changed
	}
	sel.SetSelected(v)
}

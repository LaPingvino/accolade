// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"cmp"
	"strings"

	"github.com/LaPingvino/lexington/rules"
)

// The title page template is the script's own (Joop, 2026-11-23: "A title
// page is part of a script's look, so each script should carry its own
// choice"). Page Setup's "title-page-template" setting is the default: a
// script that has no choice of its own prints with it, and keeps the one
// it printed with when it is saved (on this machine). Choosing a template
// (the Title Page tab, Page Setup) makes it the script's, and the default
// for new scripts too, unless only the script has that template (one it
// brought), as the script format does (script_format.go).
//
// Where the choice is kept, as the script's other print choices are:
//   - an .accolade: Accolade's key in info.json, "titlePage" (bundle.go),
//     names the template it prints with (the standard one only when chosen
//     for it), and one of one's own goes with it in its lexington.toml's
//     [TitlePages] (carryFor), so it prints the same anywhere, Lexington's
//     command line too (bundleTitlePage);
//   - every script with a file, bundles too: the settings'
//     "title-page-templates", "key\t/path" as the cards' looks are kept
//     (cards_tab.go), which also works in the browser.
// The key is rules.StandardTitlePage for Fountain's, never "" (that is
// "no choice yet").

const titleChoicesPref = "title-page-templates"

// titleChoiceOf is the template key kept for the script at path, "" for
// none.
func (w *MainWindow) titleChoiceOf(path string) string {
	if path == "" || w.settings == nil {
		return ""
	}
	path = absPath(path)
	for _, e := range w.settings.GetStringSlice(titleChoicesPref) {
		if key, p, ok := strings.Cut(e, "\t"); ok && p == path {
			return key
		}
	}
	return ""
}

// keepTitleChoice keeps key as the template of the script at path.
func (w *MainWindow) keepTitleChoice(path, key string) {
	if path == "" || key == "" || w.settings == nil || w.titleChoiceOf(path) == key {
		return
	}
	path = absPath(path)
	list := []string{key + "\t" + path}
	for _, e := range w.settings.GetStringSlice(titleChoicesPref) {
		if _, p, _ := strings.Cut(e, "\t"); p != path && len(list) < maxDocumentTypes {
			list = append(list, e)
		}
	}
	w.settings.SetStringSlice(titleChoicesPref, list)
	_ = w.settings.Save()
}

// titleChoiceKey is a template's key as a choice is kept: the standard
// one by its name.
func titleChoiceKey(key string) string {
	if key == "" {
		return rules.StandardTitlePage
	}
	return key
}

// defaultTitleTemplate is the template new scripts start with ("" the
// standard one).
func (w *MainWindow) defaultTitleTemplate() string {
	if w.settings == nil {
		return ""
	}
	return pageSetupOf(w.settings).titleTemplate
}

// scriptTitleTemplate is the key of the script's template as chosen, ""
// for the standard one: its own, else the one it keeps on this machine,
// else the default (not checked against the configuration:
// titleTemplate).
func (w *MainWindow) scriptTitleTemplate() string {
	key := cmp.Or(w.titleChoice, w.titleKept)
	if key == "" {
		key = w.defaultTitleTemplate()
	}
	if key == rules.StandardTitlePage {
		return ""
	}
	return key
}

// bundleTitlePage is the template an .accolade names in info.json, as the
// format is named (bundle.go): the one chosen for it, the standard one
// too; else the one it prints with, unless that is the standard one ("":
// nothing to name, so a bundle saved with only its text changed keeps
// its info.json). One of one's own goes with it (carryFor, given
// titleTemplate).
func (w *MainWindow) bundleTitlePage() string {
	if w.titleChoice != "" {
		return w.titleChoice
	}
	return w.titleTemplate()
}

// openTitleChoice takes the opened script's choice: its .accolade's, and
// the one kept for its file on this machine; none, and it prints with the
// default.
func (w *MainWindow) openTitleChoice(path string, b *textBundle) {
	w.titleChoice, w.titleKept = "", w.titleChoiceOf(path)
	if b != nil {
		w.titleChoice = b.ours.TitlePage
	}
}

// pinTitleChoice keeps, as the script is saved to path, the template it
// prints with: its own, else the default (from now on its own on this
// machine).
func (w *MainWindow) pinTitleChoice(path string) {
	if w.titleChoice == "" && w.titleKept == "" {
		w.titleKept = titleChoiceKey(w.defaultTitleTemplate())
	}
	w.keepTitleChoice(path, cmp.Or(w.titleChoice, w.titleKept))
}

// chooseTitleTemplate makes key ("" the standard one) the script's
// template, kept for its file at once (an .accolade's info.json on its
// next save, which this asks for).
func (w *MainWindow) chooseTitleTemplate(key string) {
	key = titleChoiceKey(key)
	if key == w.titleChoice {
		return
	}
	w.titleChoice, w.titleKept = key, key
	w.keepTitleChoice(w.currentFile, key)
	if w.bundle != nil && carriesConf(w.currentFile) && w.bundle.ours.TitlePage != key {
		w.hasChanges = true
		w.updateTitle()
	}
}

package main

import (
	"github.com/LaPingvino/lexington/rules"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The character picker (the toolbar's Character button), the quick one
// beside the Characters pane (characters_pane.go): the script's
// characters, who speaks most first, then the character list's who do
// not speak yet; one picked goes in as a character cue, ready for the
// dialogue.

// scriptCharacters are the names that speak in text, most speeches first.
func scriptCharacters(text string, starts []string) []string {
	lines := strings.Split(text, "\n")
	count := map[string]int{}
	for i, k := range syntax.ClassifyIn(lines, starts, dialectForStarts(starts)) {
		if k == syntax.Character {
			if name := cueName(lines[i]); name != "" {
				count[name]++
			}
		}
	}
	names := make([]string, 0, len(count))
	for n := range count {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if count[names[i]] != count[names[j]] {
			return count[names[i]] > count[names[j]]
		}
		return names[i] < names[j]
	})
	return names
}

// cueName is a cue's name: without @, ^, and extensions such as (V.O.).
func cueName(cue string) string {
	n := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(cue), "^"))
	n = strings.TrimPrefix(n, "@")
	if i := strings.Index(n, "("); i >= 0 {
		n = n[:i]
	}
	return strings.TrimSpace(n)
}

// pickCharacter shows the characters under the button; one picked is
// put in as a cue.
func (w *MainWindow) pickCharacter(at fyne.CanvasObject) {
	if w.scriptAway("Character Name") { // another middle tab in front (middle.go)
		return
	}
	text := w.textEditor.Text()
	names := scriptCharacters(text, w.sceneStarts())
	// and the character list's, who speak not yet (characters_block.go)
	have := map[string]bool{}
	for _, n := range names {
		have[charKey(n)] = true
	}
	for _, n := range listedNames(text) {
		if k := charKey(n); k != "" && !have[k] {
			have[k] = true
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		w.setStatus("No characters speak yet")
		return
	}
	var items []*fyne.MenuItem
	for _, n := range names {
		items = append(items, fyne.NewMenuItem(n, func() { w.insertCue(n) }))
	}
	c := w.fyneWindow.Canvas()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(at).Add(fyne.NewPos(0, at.Size().Height))
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), c, pos)
}

// insertCue puts a character cue in as a paragraph of its own, the cursor
// on the line after it for the dialogue.
func (w *MainWindow) insertCue(name string) {
	if name != strings.ToUpper(name) && !strings.HasPrefix(name, "@") &&
		!(w.dialect().Lang.Has(rules.CueList) && isListed(w.textEditor.Text(), name)) {
		name = "@" + name // a listed name in small letters: Fountain's forced cue (until the language's list rule reads it)
	}
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() { insertButton(b, formatButton{Insert: name + "\n"}) })
	})
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

package main

import (
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/LaPingvino/accolade/internal/editor/buffer"
	"github.com/LaPingvino/accolade/internal/editor/syntax"
)

// The character picker (the toolbar's Character button): the script's
// characters, who speaks most first; one picked goes in as a character
// cue, ready for the dialogue.

// scriptCharacters are the names that speak in text, most speeches first.
func scriptCharacters(text string, starts []string) []string {
	lines := strings.Split(text, "\n")
	count := map[string]int{}
	for i, k := range syntax.ClassifyWith(lines, starts) {
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
	names := scriptCharacters(w.textEditor.Text(), w.sceneStarts())
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
	w.textEditor.Edit(func(b *buffer.Buffer) {
		b.Group(func() { insertButton(b, formatButton{Insert: name + "\n"}) })
	})
	w.fyneWindow.Canvas().Focus(w.textEditor)
}

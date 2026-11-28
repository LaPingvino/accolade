// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// The Characters tab (LOOP.md, CONTEXTUAL RIBBON TABS (5)): the ribbon's
// contextual tab while the Characters pane is in sight (in the dock or in
// the middle), brought forward as it comes. It acts on the current
// character: the one opened in the pane, else the one tapped last —
// Rename Everywhere…, Merge Into…, Add Alias…, Not a Character, their
// Note, the next and previous cue, Their Scenes (a scene to go to), and
// Close. In a Markdown document the rename, the note and the places
// apply (the learned notes' are a screenplay's). (The cast photo, later.)

// The rules of the Characters tab's commands.
func charactersInSight(e cmdEnv) bool  { return e.characters }
func charactersOfScript(e cmdEnv) bool { return e.characters && !e.markdown }
func characterChosen(e cmdEnv) bool    { return e.character }
func characterHasScenes(e cmdEnv) bool { return e.character && !e.markdown }
func (w *MainWindow) charactersEnv(e *cmdEnv) {
	e.characters = w.dock != nil && w.paneInSight("characters")
	if e.characters && w.charsPane != nil {
		_, e.character = w.charsPane.current()
	}
}

// current is the character the tab acts on: the one opened, else the one
// tapped last (false: none, or gone from the script).
func (p *charactersPane) current() (charInfo, bool) {
	key := p.open
	if key == "" {
		key = p.lit
	}
	if key == "" {
		return charInfo{}, false
	}
	return p.charOf(key)
}

// onCharacter runs f on the current character, or says how to choose one.
func (w *MainWindow) onCharacter(f func(p *charactersPane, c charInfo)) {
	if w.charsPane == nil {
		w.setStatus("The Characters pane is not open: View > Characters")
		return
	}
	p := w.charsPane
	c, ok := p.current()
	if !ok {
		w.setStatus("Tap a character in the Characters pane first")
		return
	}
	f(p, c)
}

// characterNamed is a command's tooltip with the current character's name.
func characterNamed(label string) func(w *MainWindow) string {
	return func(w *MainWindow) string {
		if w.charsPane != nil {
			if c, ok := w.charsPane.current(); ok {
				return label + ": " + c.Name
			}
		}
		return label + " (tap a character first)"
	}
}

// theirScenes is Their Scenes' menu: the current character's scenes, each
// going there.
func (w *MainWindow) theirScenes() *fyne.Menu {
	var items []*fyne.MenuItem
	if w.charsPane != nil {
		if c, ok := w.charsPane.current(); ok && w.charsPane.model != nil {
			m := w.charsPane.model
			for _, sc := range c.Scenes {
				if sc < 0 || sc >= len(m.scenes) {
					continue
				}
				s := m.scenes[sc]
				items = append(items, fyne.NewMenuItem(s.label+". "+s.heading, func() { w.goToScript(s.line) }))
			}
		}
	}
	if len(items) == 0 {
		none := fyne.NewMenuItem("No scenes yet", nil)
		none.Disabled = true
		items = append(items, none)
	}
	return fyne.NewMenu("Their Scenes", items...)
}

// charactersCommands are the Characters tab's.
func charactersCommands() []*command {
	return []*command{
		{id: "chars.rename", label: "Rename Everywhere…", ribbonLabel: "Rename", ex: "rename-character", icon: theme.DocumentCreateIcon,
			tip: "The character renamed wherever the name stands, each in its own case, one undo step", nameFor: characterNamed("Rename Everywhere…"),
			when: charactersInSight, enabled: characterChosen,
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.askRename(c.Key) }) }},
		{id: "chars.merge", label: "Merge Into…", ribbonLabel: "Merge", ex: "merge-character", icon: theme.ContentPasteIcon,
			tip: "The character's cues take another's name, their notes together", nameFor: characterNamed("Merge Into…"),
			when: charactersOfScript, enabled: characterChosen,
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.askMerge(c.Key) }) }},
		{id: "chars.alias", label: "Add Alias…", ribbonLabel: "Alias", ex: "add-alias", icon: theme.AccountIcon,
			tip: "Another name the story or a cue has for the character (the learned notes)", nameFor: characterNamed("Add Alias…"),
			when: charactersOfScript, enabled: characterChosen,
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.askAlias(c.Key) }) }},
		{id: "chars.not", label: "Not a Character", ex: "not-a-character", icon: theme.CancelIcon,
			tip: "Left out of the characters and the source's reading (the learned notes); again: a character after all", nameFor: characterNamed("Not a Character"),
			when: charactersOfScript, enabled: characterChosen,
			checked: func(w *MainWindow) bool {
				if w.charsPane == nil {
					return false
				}
				c, ok := w.charsPane.current()
				return ok && c.Not
			},
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.setNot(c.Key, !c.Not) }) }},
		{id: "chars.note", label: "Their Note", ribbonLabel: "Note", ex: "character-note", icon: theme.FileTextIcon,
			tip: "The character opened in the pane, their note to type in (kept in the character list at the document's end)", nameFor: characterNamed("Their Note"),
			when: charactersInSight, enabled: characterChosen,
			run: func(w *MainWindow) {
				w.onCharacter(func(p *charactersPane, c charInfo) {
					p.openChar(c.Key)
					w.fyneWindow.Canvas().Focus(p.detail.note)
				})
			}},
		{id: "chars.nextCue", label: "Next Cue", ribbonLabel: "Next", ex: "next-cue", icon: theme.NavigateNextIcon,
			tip: "The script's cursor on the character's next cue", nameFor: characterNamed("Next Cue"),
			when: charactersInSight, enabled: characterChosen,
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.next(c.Key, 1) }) }},
		{id: "chars.previousCue", label: "Previous Cue", ribbonLabel: "Previous", ex: "previous-cue", icon: theme.NavigateBackIcon,
			tip: "The script's cursor on the character's cue before", nameFor: characterNamed("Previous Cue"),
			when: charactersInSight, enabled: characterChosen,
			run: func(w *MainWindow) { w.onCharacter(func(p *charactersPane, c charInfo) { p.next(c.Key, -1) }) }},
		{id: "chars.scenes", label: "Their Scenes", ribbonLabel: "Scenes", ex: "their-scenes", icon: theme.ListIcon,
			tip: "The scenes the character speaks in: one to go to", nameFor: characterNamed("Their Scenes"),
			when: charactersInSight, enabled: characterHasScenes,
			submenu: (*MainWindow).theirScenes},
		{id: "chars.close", label: "Close Characters", ribbonLabel: "Close", icon: theme.CancelIcon,
			tip: "The Characters pane put away", when: charactersInSight,
			run: func(w *MainWindow) {
				if w.mid.has("characters") {
					w.closeMiddle("characters")
				} else {
					w.hidePane("characters")
				}
			}},
	}
}

// charactersTab is the Characters tab's layout.
func charactersTab() ribbonTab {
	return ribbonTab{title: "Characters", when: charactersInSight, contextual: true, forward: true, groups: []ribbonGroup{
		{title: "Character", homes: []home{big("chars.rename"), at("chars.merge"), at("chars.alias"), at("chars.not")}},
		{title: "Notes", homes: []home{big("chars.note")}},
		{title: "Places", homes: []home{at("chars.previousCue"), at("chars.nextCue"), at("chars.scenes")}},
		{title: "Close", homes: []home{big("chars.close")}},
	}}
}

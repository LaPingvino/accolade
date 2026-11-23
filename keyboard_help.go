package main

import (
	"fmt"
	"strings"
)

// The keyboard's help made from the tables (the keyboard pass): the
// vim and helix keys on the Keyboard Shortcuts page and its cheat sheet,
// per mode, and the "Keyboard Only" page, whose keys are looked up in
// the registry as it is written, so it never names a key a command does
// not have. docs/help/keyboard.md is the page for the web, written by
// TestKeyboardOnlyPage with UPDATE_HELP=1.

// modesSection is the shortcuts page's part on vim and helix: both
// modes for the web page (both), the window's own in the app.
func modesSection(env cmdEnv, both bool) string {
	var b strings.Builder
	b.WriteString("\n## Vim and helix\n\n")
	b.WriteString("Preferences > Editor > Keys gives the editor vim's or helix's keys; the menus' shortcuts above stay, and come first. All of them, with what differs from vim and helix: [Vim and Helix Modes](vim.md); the `:` line's commands, which it completes as you type: [Ex Commands](ex.md).\n")
	modes := []string{"vim", "helix"}
	if !both {
		switch env.editorKeys {
		case "vim", "helix":
			modes = []string{env.editorKeys}
		default:
			b.WriteString("\nThe editor has the standard keys now.\n")
			return b.String()
		}
	}
	for _, m := range modes {
		fmt.Fprintf(&b, "\n### %s\n", modeTitle(m))
		b.WriteString(strings.ReplaceAll(modeKeysMarkdown(m, "####"), "](ex.md)", "](ex.md)"))
	}
	b.WriteString("\n### The `:` line for production and structure\n\n")
	for _, c := range exCommands().cmds {
		if c.group == exGroupProd {
			fmt.Fprintf(&b, "- `:%s %s` — %s\n", c.name, c.usage, c.desc)
		}
	}
	return b.String()
}

// modesSheet is the cheat sheet's part on vim and helix: the keys short.
func modesSheet(env cmdEnv, both bool) string {
	modes := []string{"vim", "helix"}
	if !both {
		if env.editorKeys != "vim" && env.editorKeys != "helix" {
			return ""
		}
		modes = []string{env.editorKeys}
	}
	var b strings.Builder
	for _, m := range modes {
		fmt.Fprintf(&b, "\n## %s mode\n\n", modeTitle(m))
		var line []string
		for _, g := range modeKeyGroups {
			for _, k := range keysOfMode(m) {
				if k.group == g {
					line = append(line, "**"+k.keys+"**")
				}
			}
			if len(line) > 0 {
				fmt.Fprintf(&b, "- %s: %s\n", g, strings.Join(line, " · "))
				line = nil
			}
		}
	}
	b.WriteString("\n`:renumber` `:lock` `:unlock` `:omit` `:restore` `:move 3..5 after 9` `:swap 3 with #Act` `:issue` `:tag props` `:draft name` — [Ex Commands](ex.md)\n")
	return b.String()
}

// keyOf is a command's first key as the page writes it ("" none).
func keyOf(id string) string {
	ks := commandKeys(cmd(id))
	if len(ks) == 0 {
		return ""
	}
	return "**" + ks[0] + "**"
}

// keyOrName is a command's key, else where it lives.
func keyOrName(id string) string {
	if k := keyOf(id); k != "" {
		return k
	}
	if hs := homesOf(id); len(hs) > 0 {
		return strings.TrimSuffix(hs[0], "…")
	}
	return strings.TrimSuffix(cmd(id).label, "…")
}

// keyboardOnlyPage is the Keyboard Only page: writing, adapting and
// production without a mouse.
func keyboardOnlyPage() string {
	k := keyOrName
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Keyboard Only\n\n")
	p("Everything Accolade does can be done without a mouse. This page walks through a day's work by the keys alone; its keys are looked up in Accolade's own list of commands as the page is made, so they are the ones the app has. Ctrl is Cmd on a Mac; a few keys are others there and in the browser ([Keyboard Shortcuts](shortcuts.md) says which).\n\n")

	p("## Finding anything\n\n")
	p("- %s or **F1** — the command palette: type a few letters of any command (the letters in order: “nsp” is Next Source Passage), **Enter** runs it. Every command of the menus and the ribbon is there.\n", k("app.palette"))
	p("- %s, or **Alt** pressed and let go (not on a Mac) — the ribbon's key tips: a letter on each tab, then on each button.\n", k("view.keyTips"))
	p("- %s and %s — from the script to each pane in turn and back; **Esc** in a pane goes back to the script.\n", k("view.nextPane"), k("view.previousPane"))
	p("- %s and %s — the middle tabs (the script, the cards, the stripboard, the Table Read).\n", k("view.midNext"), k("view.midPrev"))
	p("- **Tab** and **Shift+Tab** go through every dialog's controls (a dialog's list scrolls to the one with the focus); **Enter** presses the button in blue, **Esc** closes. A row of tabs is one stop: the arrow keys go along it.\n")
	p("- **Shift+F10** on a pane's tab or a middle tab opens its menu (Move to Top, Open in the Middle, Close).\n\n")

	p("## Writing\n\n")
	p("- **Tab** on an empty line starts a character cue; **Enter** after it is the dialogue; **Tab** in dialogue a parenthetical; **Enter** on an empty dialogue line is action. [Coming from Final Draft](final-draft.md) has the whole flow.\n")
	p("- %s … %s — the line's element (Scene Heading to Lyrics); %s — a new scene, field by field.\n", k(elementCommandID("Scene Heading")), k(elementCommandID("Lyrics")), k("write.newScene"))
	p("- While “Tab: …” shows on the status line, **Tab** takes the name or heading offered, again for the next; **Esc** puts it away.\n")
	p("- %s, %s, %s — find, find next, replace.\n", k("edit.find"), k("edit.findNext"), k("edit.replace"))
	p("- %s — a marker to come back to (not printed); %s and %s — the next and the one before.\n", k("write.marker"), k("write.nextMarker"), k("write.previousMarker"))
	p("- %s — the pages beside the script; %s — the outline; %s — the cards.\n\n", k("view.preview"), k("view.outline"), k("view.cards"))

	p("## Moving scenes and sections\n\n")
	p("- %s and %s move the scene, section, speech or paragraph at the cursor past the next one; %s and %s promote and demote it (View > Structure has the rest).\n", k("structure.up"), k("structure.down"), k("structure.promote"), k("structure.demote"))
	p("- %s and %s — a run of scenes or a section moved or swapped by a dialog: pick the first, the last, where.\n", k("structure.move"), k("structure.swap"))
	p("- With vim's or helix's keys: `]]` and `[[` from scene to scene, `das` deletes a scene and `p` puts it after the next, `mas` then `after 12` moves it; `:move 3..5 after 9` and `:swap 3 with #Act Two` do the same by the scenes' labels ([Vim and Helix Modes](vim.md)).\n\n")

	p("## Adapting\n\n")
	p("- %s and %s — the next and previous passage still to adapt; %s — Adapt…: how and where it goes in, **Enter** takes the likely choice.\n", k("source.next"), k("source.previous"), k("source.adapt"))
	p("- %s — the scene's guessed link to the source; %s — link the selection to the source passage.\n", k("source.linkGuess"), k("source.link"))
	p("- %s reaches the Source pane beside the script, Tab its passages' buttons.\n\n", k("view.nextPane"))

	p("## Production\n\n")
	p("- %s, then %s or `:renumber keep 5,12A` — the numbers locked, then renumbered by a strategy; %s and %s (`:omit 4..6`, `:restore 5`).\n", k("production.lockNumbers"), k("production.renumber"), k("production.omit"), k("production.restore"))
	p("- %s — tag the selection for the breakdown (`:tag props`); %s — the breakdown; %s — reports.\n", k("production.tag"), k("production.breakdown"), k("production.reports"))
	p("- %s — the stripboard: **Up**/**Down** choose a strip, **Alt+Up**/**Alt+Down** move it, %s moves it to a day.\n", k("production.stripboard"), k("strips.moveTo"))
	p("- %s (`:issue`) — issue the draft; %s (`:draft name`) — keep a draft; %s — the drafts, to compare.\n", k("production.issue"), k("file.saveDraft"), k("file.drafts"))
	p("- %s — the Table Read: **Right**, **Space** or **PgDn** the next scene, **Left** or **PgUp** the one before, **F** full screen, **Esc** back.\n", k("view.tableRead"))
	return b.String()
}

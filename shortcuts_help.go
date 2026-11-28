// SPDX-License-Identifier: MIT OR GPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

// The Keyboard Shortcuts help page and cheat sheet, made from the command
// registry (commands.go), so they list every key a command has. The app
// makes them for the document's mode (helpPage); docs/help/shortcuts.md
// and cheat-shortcuts.md are the same made for both modes, for the web
// pages, written by TestShortcutsHelp with UPDATE_HELP=1.

// keyName is a shortcut as the help writes it: "Ctrl+Shift+P", "Alt+Up",
// "F6" (Ctrl is Cmd on a Mac, the page says).
func keyName(s fyne.Shortcut) string {
	ks, ok := s.(fyne.KeyboardShortcut)
	if !ok {
		return s.ShortcutName()
	}
	var b strings.Builder
	m := ks.Mod()
	if m&(fyne.KeyModifierShortcutDefault|fyne.KeyModifierControl|fyne.KeyModifierSuper) != 0 {
		b.WriteString("Ctrl+")
	}
	if m&fyne.KeyModifierShift != 0 {
		b.WriteString("Shift+")
	}
	if m&fyne.KeyModifierAlt != 0 {
		b.WriteString("Alt+")
	}
	b.WriteString(keyLabel(ks.Key()))
	return b.String()
}

// keyLabel is a key's name as written on keyboards.
func keyLabel(k fyne.KeyName) string {
	switch k {
	case fyne.KeyReturn, fyne.KeyEnter:
		return "Enter"
	case fyne.KeyComma:
		return ","
	case fyne.KeyPeriod:
		return "."
	case fyne.KeyPageDown:
		return "PgDn"
	case fyne.KeyPageUp:
		return "PgUp"
	case fyne.KeyEscape:
		return "Esc"
	case fyne.KeyBackspace:
		return "Backspace"
	}
	return string(k)
}

// commandKeys are c's keys as the help writes them, once each (Enter and
// the keypad's Enter are one).
func commandKeys(c *command) []string {
	var out []string
	seen := map[string]bool{}
	add := func(k string) {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	ks := c.keys
	if c.pcKeys != nil { // the page is one for all: a Mac's are a note (macNote)
		ks = c.pcKeys
	}
	for _, k := range ks {
		add(keyName(k))
	}
	if c.shownKeys != "" {
		add("Ctrl+" + c.shownKeys)
	}
	return out
}

// shortcutLine is one command's line: its label and keys, and where it
// applies when the page is for both modes (both) or on the web (env).
func shortcutLine(c *command, env cmdEnv, both bool) string {
	label := c.labelIn(env)
	line := fmt.Sprintf("- **%s** — %s", strings.TrimSuffix(label, "…"), strings.Join(commandKeys(c), ", "))
	var notes []string
	if both {
		sp, md := c.applies(cmdEnv{browser: env.browser}), c.applies(cmdEnv{markdown: true, browser: env.browser})
		switch {
		case sp && !md:
			notes = append(notes, "screenplay")
		case md && !sp:
			notes = append(notes, "Markdown")
		}
		if c.mdLabel != "" {
			notes = append(notes, "“"+c.mdLabel+"” in Markdown")
		}
	}
	if c.role == productionRole { // put away for a writer (role.go)
		notes = append(notes, "working as Production")
	}
	inBrowser := c.applies(cmdEnv{markdown: env.markdown, browser: true})
	if both { // in either mode's browser
		inBrowser = c.applies(cmdEnv{browser: true}) || c.applies(cmdEnv{markdown: true, browser: true})
	}
	if !env.browser && !inBrowser {
		notes = append(notes, "not in the browser")
	}
	if n := macNote(c); n != "" {
		notes = append(notes, n)
	}
	if n := webNote(c); n != "" && (env.browser || both) {
		notes = append(notes, n)
	}
	if len(notes) > 0 {
		line += " (" + strings.Join(notes, "; ") + ")"
	}
	return line + "\n"
}

// menuShortcuts are the menu bar's commands with keys, menu by menu, as
// in env (both: either mode's).
func menuShortcuts(env cmdEnv, both bool) (titles []string, lists [][]*command) {
	seen := map[string]bool{}
	for _, m := range commands().menuBar {
		var list []*command
		for _, h := range flatHomes(m.homes) {
			if h.id == "" || seen[h.id] {
				continue
			}
			c := cmd(h.id)
			in := c.applies(env)
			if both {
				in = c.applies(cmdEnv{browser: env.browser}) || c.applies(cmdEnv{markdown: true, browser: env.browser})
			}
			if !in || len(commandKeys(c)) == 0 {
				continue
			}
			seen[h.id] = true
			list = append(list, c)
		}
		if len(list) > 0 {
			titles = append(titles, m.title+" menu")
			lists = append(lists, list)
		}
	}
	// keys of commands no menu has (F6 for the panes)
	homed := map[string]bool{}
	for _, m := range commands().menuBar {
		for _, h := range flatHomes(m.homes) {
			homed[h.id] = true
		}
	}
	var others []*command
	for _, c := range commands().commands {
		in := c.applies(env)
		if both {
			in = c.applies(cmdEnv{browser: env.browser}) || c.applies(cmdEnv{markdown: true, browser: env.browser})
		}
		if !homed[c.id] && in && len(commandKeys(c)) > 0 {
			others = append(others, c)
		}
	}
	if len(others) > 0 {
		titles = append(titles, "Not in a menu")
		lists = append(lists, others)
	}
	return titles, lists
}

// writingKeys are the keys the editor has while writing, not a menu's.
func writingKeys(env cmdEnv, both bool) []string {
	var out []string
	if !env.markdown || both {
		mark := ""
		if both {
			mark = " (screenplay)"
		}
		out = append(out,
			"- **Tab** on an empty line — start a character cue"+mark,
			"- **Tab** or **Enter** after a cue — its dialogue",
			"- **Tab** in dialogue — a parenthetical, (|)",
			"- **Tab** or **Enter** in a parenthetical — back to the dialogue",
			"- **Enter** on an empty dialogue line — action",
			"- **Enter** after a scene heading — action, the blank line written",
			"- **Tab** while the status bar shows “Tab: …” — take the name, heading or suggestion; again for the next",
			"- **Esc** — put a suggestion away",
		)
	}
	if env.markdown || both {
		mark := ""
		if both {
			mark = " (Markdown)"
		}
		out = append(out, "- **Enter** in a list or quote — the next item"+mark)
	}
	out = append(out,
		"- **Tab** at a heading's start — fold or unfold it; **Shift+Tab** — all",
		"- **Ctrl+Z** — undo; **Ctrl+Y** — redo",
		"- **Ctrl+A** — select all; **Ctrl+C**, **Ctrl+X**, **Ctrl+V** — copy, cut, paste",
		"- **Ctrl+Tab**, **Ctrl+Shift+Tab** — out of the editor to the next or previous field (the browser keeps these for its tabs: there **F6** goes to the panes)",
	)
	if !env.browser {
		out = append(out, "- **Ctrl+Q** — quit (not in the browser)")
	}
	return out
}

// shortcutsPage is the Keyboard Shortcuts page for env (both: for either
// mode, the web page's).
func shortcutsPage(env cmdEnv, both bool) string {
	var b strings.Builder
	b.WriteString("# Keyboard Shortcuts\n\n")
	b.WriteString("Ctrl is Cmd on a Mac. This page is made from Accolade's own list of commands, so no shortcut is missing")
	switch {
	case both:
		b.WriteString("; the app shows the ones for the document open")
	case env.markdown:
		b.WriteString("; these are a Markdown document's")
	default:
		b.WriteString("; these are a screenplay's")
	}
	b.WriteString(". See also the [cheat sheet](cheat-shortcuts.md), [Keyboard Only](keyboard.md) for working without a mouse, and for a Final Draft writer's habits [Coming from Final Draft](final-draft.md).\n\n")
	b.WriteString("## While writing\n\n")
	b.WriteString(strings.Join(writingKeys(env, both), "\n") + "\n")
	titles, lists := menuShortcuts(env, both)
	for i, list := range lists {
		fmt.Fprintf(&b, "\n## %s\n\n", titles[i])
		for _, c := range list {
			b.WriteString(shortcutLine(c, env, both))
		}
	}
	b.WriteString("\n## The command palette and the ribbon\n\n" + strings.Join(paletteRibbonKeys(), "\n") + "\n")
	b.WriteString(modesSection(env, both))
	return b.String()
}

// paletteRibbonKeys are the palette's keys and the ribbon's key tips.
func paletteRibbonKeys() []string {
	palette := strings.Join(commandKeys(cmd("app.palette")), "** or **")
	return []string{
		"- **" + palette + "** — the command palette: type part of a command's name (the letters in order: “nsp” finds Next Source Passage), **Up**/**Down** choose, **Enter** runs it, **Esc** closes",
		"- **F10**, or **Alt** pressed and let go alone (not on a Mac) — the ribbon's key tips (Preferences > General > Look: Ribbon): a letter on each tab, the palette, ☰ (the menus), the lock (the ribbon's order: Smart or Fixed) and the Quick row's buttons; type one to open that tab, then the letter on a button to press it",
		"- **Esc** in the key tips — back to the tabs' letters, again to leave them",
		"- **Tab**, **Shift+Tab** — through the ribbon's tabs and buttons; **Left**/**Right** along a row, **Down** from a tab to its buttons, **Up** back; **Enter** or **Space** presses",
	}
}

// shortcutsSheet is the cheat sheet: the same keys, short.
func shortcutsSheet(env cmdEnv, both bool) string {
	var b strings.Builder
	b.WriteString("# Keyboard Shortcuts\n\nCtrl is Cmd on a Mac. All of them: [Keyboard Shortcuts](shortcuts.md).\n\n## Writing\n\n")
	if !env.markdown || both {
		b.WriteString("- **Tab** on an empty line: a cue; after the cue: its dialogue; in dialogue: (a parenthetical)\n" +
			"- **Enter** after a cue or parenthetical: dialogue; on an empty dialogue line: action; after a heading: action\n")
	}
	if env.markdown || both {
		b.WriteString("- **Enter** in a list → the next item\n")
	}
	b.WriteString("- **Tab** on “Tab: …” — take it; **Esc** — put it away\n")
	b.WriteString("- **" + strings.Join(commandKeys(cmd("app.palette")), "**, **") + "** — the command palette; **F10** or **Alt** — the ribbon's key tips (the lock's letter switches Smart and Fixed)\n")
	titles, lists := menuShortcuts(env, both)
	for i, list := range lists {
		fmt.Fprintf(&b, "\n## %s\n\n", strings.TrimSuffix(titles[i], " menu"))
		for _, c := range list {
			label := strings.TrimSuffix(c.labelIn(env), "…")
			fmt.Fprintf(&b, "- **%s** %s\n", strings.Join(commandKeys(c), ", "), label)
		}
	}
	var mac []string // the keys a Mac has instead
	for _, c := range commands().commands {
		if n := macNote(c); n != "" && (c.applies(env) || both) {
			mac = append(mac, "- **"+strings.TrimPrefix(n, "on a Mac ")+"** "+strings.TrimSuffix(c.labelIn(env), "…"))
		}
	}
	b.WriteString(modesSheet(env, both))
	if len(mac) > 0 {
		b.WriteString("\n## On a Mac\n\nOption+Left and Option+Right move by word there, so these take Control+Option, and Cmd+H hides the app:\n\n" + strings.Join(mac, "\n") + "\n")
	}
	var web []string // the keys the browser leaves the page instead
	for _, c := range commands().commands {
		if n := webNote(c); n != "" && (c.applies(withBrowser(env)) || both) {
			web = append(web, "- **"+strings.TrimPrefix(n, "in the browser ")+"** "+strings.TrimSuffix(c.labelIn(env), "…"))
		}
	}
	if len(web) > 0 && (env.browser || both) {
		b.WriteString("\n## In the browser\n\nThe browser keeps Ctrl+N, Ctrl+T, Ctrl+W, Ctrl+Tab and Ctrl+PgUp/PgDn for its own windows and tabs, so these are others there:\n\n" + strings.Join(web, "\n") + "\n")
	}
	return b.String()
}

// withBrowser is e in the browser.
func withBrowser(e cmdEnv) cmdEnv {
	e.browser = true
	return e
}

// webNote says a command's keys in the browser where they are others (""
// none): the browser keeps some for itself (browserKeptKeys).
func webNote(c *command) string {
	if c.webKeys == nil {
		return ""
	}
	if len(c.webKeys) == 0 {
		return "no key in the browser"
	}
	var ks []string
	for _, k := range c.webKeys {
		ks = append(ks, keyName(k))
	}
	return "in the browser " + strings.Join(ks, ", ")
}

// macNote says a command's keys on a Mac where they are others ("" none).
func macNote(c *command) string {
	if c.macKeys == nil {
		return ""
	}
	var ks []string
	for _, k := range c.macKeys {
		ks = append(ks, macKeyName(k))
	}
	return "on a Mac " + strings.Join(ks, ", ")
}

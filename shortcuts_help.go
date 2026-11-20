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
	for _, k := range c.keys {
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
	if !env.browser && !c.applies(cmdEnv{markdown: env.markdown, browser: true}) {
		notes = append(notes, "not in the browser")
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
		for _, h := range m.homes {
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
		for _, h := range m.homes {
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
		"- **Ctrl+Tab**, **Ctrl+Shift+Tab** — out of the editor to the next or previous field",
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
	b.WriteString(". See also the [cheat sheet](cheat-shortcuts.md), and for a Final Draft writer's habits [Coming from Final Draft](final-draft.md).\n\n")
	b.WriteString("## While writing\n\n")
	b.WriteString(strings.Join(writingKeys(env, both), "\n") + "\n")
	titles, lists := menuShortcuts(env, both)
	for i, list := range lists {
		fmt.Fprintf(&b, "\n## %s\n\n", titles[i])
		for _, c := range list {
			b.WriteString(shortcutLine(c, env, both))
		}
	}
	b.WriteString("\n## Vim and helix\n\nPreferences > Editor gives the editor vim's or helix's keys; the menus' shortcuts stay. " +
		"Their own keys are not documented yet: [Vim and Helix Modes](vim.md).\n")
	return b.String()
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
	titles, lists := menuShortcuts(env, both)
	for i, list := range lists {
		fmt.Fprintf(&b, "\n## %s\n\n", strings.TrimSuffix(titles[i], " menu"))
		for _, c := range list {
			label := strings.TrimSuffix(c.labelIn(env), "…")
			fmt.Fprintf(&b, "- **%s** %s\n", strings.Join(commandKeys(c), ", "), label)
		}
	}
	return b.String()
}

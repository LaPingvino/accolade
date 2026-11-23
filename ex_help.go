package main

import (
	"fmt"
	"strings"

	"github.com/LaPingvino/accolade/docs/help"
)

// The Ex Commands help page, made from the ex table (ex_table.go), so it
// lists every command the ':' line runs. The app shows it live
// (helpPage); docs/help/ex.md is the same, for the web pages, written by
// TestExHelp with UPDATE_HELP=1.

// exHelpPage is the page.
func exHelpPage() string {
	t := exCommands()
	var b strings.Builder
	b.WriteString("# Ex Commands\n\n")
	b.WriteString("With vim's or helix's keys (Preferences > Editor), **:** opens the command line at the window's foot. " +
		"This page is made from Accolade's own table of its commands, so none is missing; `:help` opens it, `:h` and a command's name at that command's part.\n\n")
	b.WriteString("## Finding a command\n\n")
	b.WriteString("- As you type, a list above the line shows the commands it may be, each with what it does: those it begins first, then those with its letters in order (`:nsp` finds `:next-source-passage`). An empty line lists the most useful ones.\n" +
		"- **Tab** writes the chosen one in and goes on to the next; **Shift+Tab** back. A command alone in the list is taken, and what it takes listed: scene headings for `:scene`, elements for `:element`, formats for `:export`, files and folders for `:w` and `:e`, a menu's items for a command with a submenu (`:theme`, `:recent`).\n" +
		"- **Up** and **Down** choose without writing it in; **Enter** runs the line as typed, or the one chosen. On an empty line **Up** goes back through the commands run before.\n" +
		"- **Esc** closes the line. On a phone, tap a row: it is completed, and run if it wants nothing more.\n" +
		"- Every name can be shortened to a beginning no other name has, and vim's abbreviations work as in vim (`:s`, `:d`, `:co`); the list shows them as `s[ubstitute]`.\n\n")
	b.WriteString("## Ranges\n\n")
	b.WriteString("Line commands take a range of lines before their name; without one they act on the cursor's line (`:g`, `:v` and `:w` on all).\n\n" +
		"- `12` a line, `.` the cursor's, `$` the last, `%` all of them, `'<,'>` the lines last selected in visual mode\n" +
		"- `/pattern/` the next line that matches, `?pattern?` the one before; each with `+n` or `-n` after it\n" +
		"- `from,to` the lines between; `:12` alone goes to line 12\n" +
		"- Patterns are vim's: `\\(` `\\)` groups, `\\|` or, `\\<` `\\>` a word's ends, `\\c` any case; `\\v` takes Go's syntax as it is\n")
	groups := []string{exGroupLines, exGroupMoves, exGroupFiles, exGroupWriting, exGroupHelp}
	for _, g := range groups {
		fmt.Fprintf(&b, "\n## %s\n\n", g)
		for _, c := range t.cmds {
			if c.group == g {
				b.WriteString(exHelpLine(c))
			}
		}
		if g == exGroupMoves {
			b.WriteString("\nMoves are named `move-…-up` and `move-…-down`, so `:mo` lists them all; jumps `go-…`, `next-…` and `previous-…`.\n")
		}
		if g == exGroupWriting {
			var names []string
			for _, el := range toolbarElements {
				names = append(names, "`"+exElementName(el)+"`")
			}
			b.WriteString("\nThe elements: " + strings.Join(names, ", ") + ".\n")
		}
		if g == exGroupHelp {
			var names []string
			for _, p := range append(help.Pages(), help.Cheats()...) {
				names = append(names, "`"+p.Slug+"`")
			}
			b.WriteString("\nThe pages: " + strings.Join(names, ", ") + ".\n")
		}
	}
	b.WriteString("\n## " + exGroupMenus + "\n\nEvery command of the menus, the ribbon and the palette, by a name made from its label (Next Source Passage is `:next-source-passage`), shorter for the common ones. Where it lives is said after it.\n")
	var menus []string
	byMenu := map[string][]*exCmd{}
	for _, c := range t.cmds {
		if c.group != exGroupMenus {
			continue
		}
		m := exHelpSection(c)
		if _, ok := byMenu[m]; !ok {
			menus = append(menus, m)
		}
		byMenu[m] = append(byMenu[m], c)
	}
	for _, m := range menus {
		fmt.Fprintf(&b, "\n### %s\n\n", m)
		for _, c := range byMenu[m] {
			b.WriteString(exHelpLine(c))
		}
	}
	return b.String()
}

// exHelpSection is the heading c is under: its group, or for a registry
// command its menu.
func exHelpSection(c *exCmd) string {
	if c.group != exGroupMenus {
		return c.group
	}
	if hs := homesOf(c.id); len(hs) > 0 {
		m, _, _ := strings.Cut(hs[0], " > ")
		return m
	}
	return "Elsewhere"
}

// exHelpLine is a command's line: its name (with its abbreviation), what
// follows it, what it does, and where it applies.
func exHelpLine(c *exCmd) string {
	name := c.name
	if c.short != "" && c.short != c.name {
		name = c.short + "[" + strings.TrimPrefix(c.name, c.short) + "]"
	}
	line := "- **:" + name + "**"
	if c.usage != "" {
		line += " `" + c.usage + "`"
	}
	line += " — " + c.desc
	if rc := commands().byID[c.id]; rc != nil && !c.line() {
		var notes []string
		sp, md := rc.appliesForRole(cmdEnv{}), rc.appliesForRole(cmdEnv{markdown: true})
		switch {
		case sp && !md:
			notes = append(notes, "screenplay")
		case md && !sp:
			notes = append(notes, "Markdown")
		}
		if rc.role == productionRole {
			notes = append(notes, "working as Production")
		}
		if !rc.appliesForRole(cmdEnv{browser: true}) && !rc.appliesForRole(cmdEnv{markdown: true, browser: true}) {
			notes = append(notes, "not in the browser")
		}
		if len(notes) > 0 {
			line += " (" + strings.Join(notes, "; ") + ")"
		}
	}
	return line + "\n"
}

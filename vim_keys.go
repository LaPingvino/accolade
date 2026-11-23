package main

import (
	"fmt"
	"strings"
)

// The keys of vim's and helix's modes as a table: the Vim and Helix
// Modes help page and the keyboard cheat sheet are made from it, and a
// test (vim_keys_test.go) types every key in each mode and fails on one
// that does something without a line here.

// modeKey is one line of the table.
type modeKey struct {
	mode  string // "vim", "helix" or "both"
	group string
	keys  string // as the page writes them
	desc  string
	// seq are the keys as typed, for the test: each a run of runes typed
	// in normal mode (none: a key with a modifier, or one the test does
	// not type)
	seq []string
}

// The groups, in the page's order.
var modeKeyGroups = []string{"Modes", "Moving", "Changing", "Text objects and the structure", "Selecting", "Several selections", "Search and commands", "Folds and the outline"}

// modeKeys is the table.
var modeKeys = []modeKey{
	// Modes
	{"both", "Modes", "Esc", "back to normal mode (in vim onto the character typed last)", nil},
	{"both", "Modes", "i a", "insert before / after the cursor (helix: the selection's start / end)", []string{"i", "a"}},
	{"both", "Modes", "I A", "insert at the line's first character / its end", []string{"I", "A"}},
	{"both", "Modes", "o O", "a new line below / above, inserting", []string{"o", "O"}},
	{"vim", "Modes", "v V", "visual mode by characters / by lines; again: back to normal", []string{"v", "V"}},
	{"helix", "Modes", "v", "select mode: motions extend the selection; again: back", []string{"v"}},

	// Moving
	{"both", "Moving", "h j k l", "left, down, up, right (helix: the selection dropped)", []string{"h", "j", "k", "l"}},
	{"both", "Moving", "w b e", "the next word's start, the word's start before, the word's end (helix: selecting the word)", []string{"w", "b", "e"}},
	{"both", "Moving", "0 ^ $", "the line's start, its first character, its end", []string{"0", "^", "$"}},
	{"both", "Moving", "gg G", "the first line / the last", []string{"gg", "G"}},
	{"both", "Moving", "Enter, Backspace", "down a line to its start; left", nil},
	{"both", "Moving", "1 … 9", "a count before a motion or command: 3w, 2dd, 2]]", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},

	// Changing
	{"vim", "Changing", "d c y + a motion", "delete, change (then insert), copy what the motion passes: dw, c$, y3j", []string{"d", "c", "y"}},
	{"vim", "Changing", "dd cc yy", "the line (a count: lines)", []string{"dd", "cc", "yy"}},
	{"vim", "Changing", "D C Y", "to the line's end; Y the line", []string{"D", "C", "Y"}},
	{"vim", "Changing", "x X", "delete the character under / before the cursor", []string{"x", "X"}},
	{"helix", "Changing", "d c y", "delete, change, copy the selection (none: the character under the cursor)", []string{"d", "c", "y"}},
	{"both", "Changing", "p P", "put after / before (lines after or before the line; a scene or section after or before the cursor's)", []string{"p", "P"}},
	{"vim", "Changing", "J", "join the line below onto this one", []string{"J"}},
	{"both", "Changing", "r + a character", "replace the character under the cursor", []string{"r"}},
	{"vim", "Changing", "u, Ctrl+R", "undo, redo (each command is one step)", []string{"u"}},
	{"helix", "Changing", "u U, Ctrl+R", "undo, redo", []string{"u", "U"}},

	// Text objects and the structure
	{"both", "Text objects and the structure", "]] [[", "the next / previous scene heading (helix: select that scene)", []string{"]]", "[["}},
	{"both", "Text objects and the structure", "]s [s", "the next / previous section heading (helix: select that section)", []string{"]s", "[s"}},
	{"vim", "Text objects and the structure", "as is", "a text object: a scene with the blank lines after it / its inside: das, cis, yas, vis, 2das", []string{"das", "dis", "vas", "vis"}},
	{"vim", "Text objects and the structure", "aS iS", "a section with all it holds / its inside: daS, yiS, vaS", []string{"daS", "diS", "vaS", "viS"}},
	{"vim", "Text objects and the structure", "m + an object", "move it through the structure engine: mas opens :'<,'>move , then after 12 or before #Act Two", []string{"m", "mas", "maS", "mis", "miS"}},
	{"helix", "Text objects and the structure", "mas mis maS miS", "select a scene, its inside, a section, its inside; then d c y p, or : to move it", []string{"m", "mas", "mis", "maS", "miS"}},
	{"both", "Text objects and the structure", "Alt+j Alt+k", "move the unit at the cursor (a scene or section with all it holds, a speech, a paragraph) down / up past the next", nil},
	{"both", "Text objects and the structure", "Alt+h Alt+l", "promote / demote it (a scene out of its section, into the one before)", nil},
	{"both", "Text objects and the structure", "Alt+J Alt+K", "into the next / previous scene", nil},
	{"both", "Text objects and the structure", "Alt+H Alt+L", "pass the last speech on / take in the next (barf, slurp)", nil},

	// Selecting
	{"vim", "Selecting", "a motion in visual mode", "extends the selection; then d x y c act on it", []string{"vj"}},
	{"vim", "Selecting", "vas viS …", "the scene or section as the selection, line by line", nil},
	{"helix", "Selecting", "x", "the line; again: the next one too", []string{"x"}},
	{"helix", "Selecting", "%", "everything", []string{"%"}},
	{"helix", "Selecting", ";", "the selection down to the cursor", []string{";"}},

	// Several selections
	{"helix", "Several selections", "s", "select what a pattern matches inside the selections (asked on the command line)", []string{"s"}},
	{"helix", "Several selections", "S", "split the selections where a pattern matches", []string{"S"}},
	{"helix", "Several selections", "Alt+s", "split the selections into lines", nil},
	{"helix", "Several selections", "C", "a selection like the last one on the next line", []string{"C"}},
	{"helix", "Several selections", ",", "keep the primary selection only", []string{","}},
	{"helix", "Several selections", "Alt+,", "drop the primary selection", nil},
	{"helix", "Several selections", "d c i a with several", "act at every selection; typing goes to all of them", nil},

	// Search and commands
	{"both", "Search and commands", "/", "the search bar (Find)", []string{"/"}},
	{"both", "Search and commands", ":", "the command line, completing as you type (in visual or with a selection: '<,'> the lines): [Ex Commands](ex.md)", []string{":"}},

	// Folds and the outline
	{"both", "Folds and the outline", "za zc zo", "fold or unfold the heading at the cursor, fold it, unfold it", []string{"za", "zc", "zo"}},
	{"both", "Folds and the outline", "zR zM", "unfold all, fold all", []string{"zR", "zM"}},
	{"both", "Folds and the outline", "Tab, Shift+Tab", "on a heading: cycle its fold; all of them", nil},
	{"both", "Folds and the outline", "gj gk", "the next scene or section heading / this one's or the one before", []string{"gj", "gk"}},
	{"both", "Folds and the outline", "gh gl", "the section the heading is in / the first heading inside it", []string{"gh", "gl"}},
	{"both", "Folds and the outline", "gs", "the scene's guessed link to the source (Link Guess for This Scene…)", []string{"gs"}},
}

// modeUnsupported is what Accolade's modes do not have, said on the page.
var modeUnsupported = map[string]string{
	"vim": "marks (m and ' with a letter: m is the structure's move), named registers (\" and a letter: one register), macros (q, @), repeat (.), " +
		"f t F T, %, * #, n N (the search bar's Find Next and Find Previous, Ctrl+G, do that), the word and paragraph objects (iw aw ip ap), " +
		"block visual (Ctrl+V), > < ~, and :set",
	"helix": "goto mode beyond gg (g and a letter: the outline's, as vim's), space mode, match mode beyond mi and ma with s and S (mm, ms), " +
		"registers, macros, repeat (.), f t, n N (the search bar does that), the tree-sitter objects (a function, a class)",
}

// keysOfMode are the table's lines for mode ("vim" or "helix").
func keysOfMode(mode string) []modeKey {
	var out []modeKey
	for _, k := range modeKeys {
		if k.mode == mode || k.mode == "both" {
			out = append(out, k)
		}
	}
	return out
}

// modeTitle is a mode's name on the page.
func modeTitle(mode string) string {
	if mode == "helix" {
		return "Helix"
	}
	return "Vim"
}

// modeKeysMarkdown is a mode's table as the page has it.
func modeKeysMarkdown(mode string, level string) string {
	var b strings.Builder
	ks := keysOfMode(mode)
	for _, g := range modeKeyGroups {
		var lines []string
		for _, k := range ks {
			if k.group == g {
				lines = append(lines, fmt.Sprintf("- **%s** — %s", k.keys, k.desc))
			}
		}
		if len(lines) > 0 {
			fmt.Fprintf(&b, "\n%s %s\n\n%s\n", level, g, strings.Join(lines, "\n"))
		}
	}
	return b.String()
}

// vimHelpPage is the Vim and Helix Modes page (docs/help/vim.md, written
// by TestVimHelpPage with UPDATE_HELP=1).
func vimHelpPage() string {
	var b strings.Builder
	b.WriteString("# Vim and Helix Modes\n\n")
	b.WriteString("Preferences > Editor > Keys makes the editor's keys vim's or helix's. This page is made from Accolade's own table of those keys, and a test types every key in each mode: none does something without a line here.\n\n")
	b.WriteString("Both start in normal mode, where letters move and edit; i, a and the rest go to insert mode, where you write as ever — Tab and Enter there are the screenplay's ([Coming from Final Draft](final-draft.md)) — and Esc comes back. " +
		"The menus' shortcuts work in every mode ([Keyboard Shortcuts](shortcuts.md)), and so do the command palette and the ribbon's key tips. " +
		"**:** opens the command line, which completes every command as you type, Accolade's own (`:renumber`, `:move 3..5 after 9`, `:scene`) among vim's: [Ex Commands](ex.md).\n\n")
	b.WriteString("The screenplay's own: a **scene** is a text object (`as`, `is`), so is a **section** with all it holds (`aS`, `iS`); `]]` and `[[` go from scene to scene, `]s` and `[s` from section to section. " +
		"A scene deleted or yanked as one (`das`, `yas`) is put back after or before the cursor's scene, never inside it. Moving goes through the structure engine, as Alt+Up and View > Structure do: the links, tags, colours and locked numbers go with the scenes.\n")
	for _, mode := range []string{"vim", "helix"} {
		fmt.Fprintf(&b, "\n## %s\n", modeTitle(mode))
		if mode == "helix" {
			b.WriteString("\nSelection first: a motion selects what it passes over, then a command acts on the selection.\n")
		}
		b.WriteString(modeKeysMarkdown(mode, "###"))
		fmt.Fprintf(&b, "\n### Not there\n\n%s is not a full %s: %s.\n", modeTitle(mode)+" mode", modeTitle(mode), modeUnsupported[mode])
	}
	b.WriteString("\n## How they differ from vim and helix\n\n" +
		"- One register, shared by the modes and the `:` line's :d and :y; the system clipboard is Ctrl+C and Ctrl+V as ever.\n" +
		"- Each command is one undo step, and Undo asks before taking back a step an issue tagged, as the menu's does.\n" +
		"- Ctrl and Alt keys of the menus come first: a menu's shortcut is never a mode's (the keyboard audit checks that, docs/KEYBOARD.md).\n" +
		"- The search is the search bar's, with its own options; `:s` and `:g` take vim's patterns.\n")
	return b.String()
}

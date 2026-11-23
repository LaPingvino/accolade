package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// keyEnvs are the states a window can be in that change which commands
// apply: the plain screenplay and Markdown document, and each view or
// moment with commands of its own (the contextual tabs').
var keyEnvs = []struct {
	name string
	env  cmdEnv
}{
	{"screenplay", cmdEnv{}},
	{"Markdown", cmdEnv{markdown: true}},
	{"screenplay, adapting and in production", cmdEnv{source: true, locked: true, pages: true, issued: true, sourceLocked: true, renamed: true, brought: true}},
	{"the source in the middle", cmdEnv{source: true, away: true, midDoc: true, midTabs: true, midSide: true}},
	{"the stripboard", cmdEnv{away: true, midTabs: true, strips: true}},
	{"the cards", cmdEnv{away: true, midTabs: true, cards: true}},
	{"the cards' timeline", cmdEnv{away: true, midTabs: true, cards: true, timeline: true}},
	{"the pages", cmdEnv{preview: true, pagesShown: true, pdfFile: true, refreshable: true, setup: true}},
	{"Markdown's Live Preview", cmdEnv{markdown: true, preview: true, livePreview: true}},
	{"the ribbon, Smart", cmdEnv{ribbon: true, smart: true, topEdited: true}},
	{"the ribbon, Fixed", cmdEnv{ribbon: true}},
	{"with pandoc", cmdEnv{pandoc: true}},
	{"an import to fix", cmdEnv{fixup: true, fixupPDF: true, fixupRules: true}},
	{"the title page", cmdEnv{titlePage: true}},
	{"a character", cmdEnv{characters: true, character: true}},
	{"a shared script", cmdEnv{shared: true, others: true, following: true}},
	{"Compare", cmdEnv{away: true, midTabs: true, compare: true, compareChanged: true}},
	{"merging", cmdEnv{away: true, midTabs: true, compare: true, compareChanged: true, merging: true, mergeTaken: true}},
	{"the Table Read", cmdEnv{away: true, midTabs: true, tableRead: true, tableReadOpen: true}},
	{"a phone", cmdEnv{phone: true}},
}

// keyClashes are the clashes of commands' keys in env on p: a key two
// commands have, a key p keeps for itself; and in vim's and helix's
// modes a key the mode takes that a command (seen first) would take from
// it.
func keyClashes(env cmdEnv, p keyPlatform) []string {
	if p == platBrowser {
		env.browser = true
	}
	var out []string
	seen := map[string]string{}
	kept := map[string]string{}
	for _, k := range keptKeysOn(p) {
		kept[k.key] = k.by
	}
	modeKeys := map[string][]string{}
	for mode, ks := range modeShortcuts {
		for _, k := range ks {
			modeKeys[k] = append(modeKeys[k], mode)
		}
	}
	for _, c := range commands().commands {
		if !c.applies(env) {
			continue
		}
		names := keyNamesOn(c, p)
		for i, s := range keysOn(c, p) {
			k := names[i]
			if prev, dup := seen[k]; dup && prev != c.id {
				out = append(out, fmt.Sprintf("%s: %s and %s", k, prev, c.id))
			}
			seen[k] = c.id
			if by, ok := kept[k]; ok {
				out = append(out, fmt.Sprintf("%s: %s, but %s keeps it (%s)", k, c.id, p, by))
			}
			if p != platMac { // a Mac's are Cmd: never the modes' Control or Option
				for _, mode := range modeKeys[keyName(s)] {
					out = append(out, fmt.Sprintf("%s: %s takes it from %s's normal mode", k, c.id, mode))
				}
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// No two commands share a key, in any state of the window, on any
// platform; none has a key the platform keeps (the browser's tabs, a
// Mac's Hide, a desktop's lock), and none takes a key from vim's or
// helix's normal mode. A new clash fails here: give the command a
// macKeys or webKeys (commands.go), or another key.
func TestNoKeyClashes(t *testing.T) {
	for _, e := range keyEnvs {
		for _, p := range keyPlatforms {
			for _, c := range keyClashes(e.env, p) {
				t.Errorf("%s, %s: %s", e.name, p, c)
			}
		}
	}
}

// The clash check finds clashes: two keys alike, a kept key, a mode's.
func TestKeyClashesFound(t *testing.T) {
	c := cmd("edit.replace")
	was := c.keys
	defer func() { c.keys = was }()
	c.keys = cmd("edit.find").keys
	if got := strings.Join(keyClashes(cmdEnv{}, platDesktop), "\n"); !strings.Contains(got, "edit.find and edit.replace") {
		t.Errorf("a shared key not found: %q", got)
	}
	c.keys = []fyne.Shortcut{shortcut(fyne.KeyW, 0)}
	if got := strings.Join(keyClashes(cmdEnv{}, platBrowser), "\n"); !strings.Contains(got, "browser keeps it") {
		t.Errorf("a browser's key not found: %q", got)
	}
	c.keys = []fyne.Shortcut{altKey(fyne.KeyJ)}
	if got := strings.Join(keyClashes(cmdEnv{}, platDesktop), "\n"); !strings.Contains(got, "vim's normal mode") {
		t.Errorf("a mode's key not found: %q", got)
	}
}

// Every command can be given from the keyboard: the palette finds it in
// some state of the window (and the ':' line by name).
func TestEveryCommandByKeyboard(t *testing.T) {
	for _, c := range commands().commands {
		found := false
		for _, e := range keyEnvs {
			for _, b := range []bool{false, true} {
				env := e.env
				env.browser = b
				found = found || c.appliesForRole(env)
			}
		}
		if !found && c.id != "app.palette" {
			t.Errorf("%s (%s): no state of the window has it in the palette", c.id, c.label)
		}
	}
}

// docs/help/keyboard.md is what the registry makes, and every key it
// names is a command's:
//
//	UPDATE_HELP=1 go test -run TestKeyboardOnlyPage .
func TestKeyboardOnlyPage(t *testing.T) {
	want := keyboardOnlyPage()
	path := filepath.Join("docs", "help", "keyboard.md")
	if os.Getenv("UPDATE_HELP") != "" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != want {
		t.Errorf("%s is not what the registry makes: UPDATE_HELP=1 go test -run TestKeyboardOnlyPage .", path)
	}
	if strings.Contains(want, "****") {
		t.Error("a key looked up is empty")
	}
}

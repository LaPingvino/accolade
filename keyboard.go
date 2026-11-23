package main

import (
	"strings"

	"fyne.io/fyne/v2"
)

// The keyboard pass (LOOP.md 2026-11-20, docs/KEYBOARD.md): every
// command's keys per platform, the keys a platform keeps for itself, and
// the keys vim's and helix's modes take before the menus would. The
// clash test (keyboard_test.go) and docs/KEYBOARD.md read these.

// keyPlatform is where Accolade runs, as far as keys go.
type keyPlatform int

const (
	platDesktop keyPlatform = iota // Linux and Windows
	platMac
	platBrowser // the WebAssembly build, any system (Ctrl)
)

var keyPlatforms = []keyPlatform{platDesktop, platMac, platBrowser}

func (p keyPlatform) String() string {
	return [...]string{"Linux/Windows", "macOS", "browser"}[p]
}

// keysOn are c's keys on p: its macKeys on a Mac, its webKeys in the
// browser, else its keys (as the registry was made, whatever this build
// swapped in: pcKeys keep the others').
func keysOn(c *command, p keyPlatform) []fyne.Shortcut {
	switch {
	case p == platMac && c.macKeys != nil:
		return c.macKeys
	case p == platBrowser && c.webKeys != nil:
		return c.webKeys
	}
	if c.pcKeys != nil {
		if (onMac && c.macKeys != nil) || (inBrowser && c.webKeys != nil) {
			return c.pcKeys
		}
	}
	return c.keys
}

// macKeyName is a key of a command's macKeys as a Mac names it: they
// are written in a Mac's own modifiers (Super is Cmd, Control is
// Control), whatever this build is.
func macKeyName(s fyne.Shortcut) string { return macName(s, false) }

// macName is s on a Mac; primary: s is a key for every system, its
// Primary (Ctrl here, on Linux) Cmd there.
func macName(s fyne.Shortcut, primary bool) string {
	ks, ok := s.(fyne.KeyboardShortcut)
	if !ok {
		return s.ShortcutName()
	}
	m := ks.Mod()
	cmd := m&fyne.KeyModifierSuper != 0
	ctrl := m&fyne.KeyModifierControl != 0
	if primary && m&fyne.KeyModifierShortcutDefault != 0 {
		cmd, ctrl = true, ctrl && fyne.KeyModifierShortcutDefault != fyne.KeyModifierControl
	}
	var b strings.Builder
	if cmd {
		b.WriteString("Cmd+")
	}
	if ctrl {
		b.WriteString("Control+")
	}
	if m&fyne.KeyModifierShift != 0 {
		b.WriteString("Shift+")
	}
	if m&fyne.KeyModifierAlt != 0 {
		b.WriteString("Option+")
	}
	b.WriteString(keyLabel(ks.Key()))
	return b.String()
}

// keyNamesOn are c's keys on p as p's keyboards name them.
func keyNamesOn(c *command, p keyPlatform) []string {
	var out []string
	for _, s := range keysOn(c, p) {
		switch {
		case p == platMac && c.macKeys != nil:
			out = append(out, macKeyName(s))
		case p == platMac:
			out = append(out, macName(s, true))
		default:
			out = append(out, keyName(s))
		}
	}
	return out
}

// keptKey is a key a platform keeps for itself: a command given it never
// gets it.
type keptKey struct{ key, by string }

// browserKeptKeys are the keys a browser never gives the page: Chromium's
// reserved commands (chrome/browser/ui/accelerator_table.cc,
// kReservedCommandIds: close tab and window, new window, incognito, new
// tab, restore tab, next and previous tab, exit) as its accelerator table
// maps them, read 2026-11-23; Firefox keeps the same ones.
var browserKeptKeys = []keptKey{
	{"Ctrl+N", "a new browser window"},
	{"Ctrl+Shift+N", "a new private window"},
	{"Ctrl+T", "a new tab"},
	{"Ctrl+Shift+T", "the tab closed last, again"},
	{"Ctrl+W", "closes the tab"},
	{"Ctrl+F4", "closes the tab"},
	{"Ctrl+Shift+W", "closes the browser window"},
	{"Alt+F4", "closes the browser window"},
	{"Ctrl+Tab", "the next tab"},
	{"Ctrl+Shift+Tab", "the tab before"},
	{"Ctrl+PgDn", "the next tab"},
	{"Ctrl+PgUp", "the tab before"},
}

// macKeptKeys are the keys macOS keeps for itself (or the app menu has,
// that every Mac app has): Hide, Minimize, the window and app switchers,
// Spotlight, the screenshots, the Dock, Mission Control and the Spaces.
var macKeptKeys = []keptKey{
	{"Cmd+H", "hides the app"},
	{"Cmd+Option+H", "hides the other apps"},
	{"Cmd+M", "minimizes the window"},
	{"Cmd+Tab", "the app switcher"},
	{"Cmd+`", "the app's next window"},
	{"Cmd+Space", "Spotlight"},
	{"Cmd+Shift+3", "a screenshot"},
	{"Cmd+Shift+4", "a screenshot of a part"},
	{"Cmd+Shift+5", "the screenshot tools"},
	{"Cmd+Option+D", "shows or hides the Dock"},
	{"Control+Up", "Mission Control"},
	{"Control+Down", "the app's windows"},
	{"Control+Left", "the Space before"},
	{"Control+Right", "the next Space"},
}

// desktopKeptKeys are the keys Linux desktops (GNOME, KDE, Ubuntu's) and
// Windows keep for themselves, by default.
var desktopKeptKeys = []keptKey{
	{"Ctrl+Alt+L", "locks the screen (GNOME, KDE)"},
	{"Ctrl+Alt+T", "a terminal (Ubuntu)"},
	{"Ctrl+Alt+D", "shows the desktop (Ubuntu)"},
	{"Ctrl+Alt+Delete", "the log-out or security screen"},
	{"Ctrl+Alt+Up", "the workspace above (GNOME)"},
	{"Ctrl+Alt+Down", "the workspace below (GNOME)"},
	{"Ctrl+Alt+Left", "the workspace before (GNOME, KDE)"},
	{"Ctrl+Alt+Right", "the next workspace (GNOME, KDE)"},
	{"Alt+Tab", "the window switcher"},
	{"Alt+F4", "closes the window"},
	{"Alt+F2", "the run dialog (GNOME, KDE)"},
	{"Alt+F1", "the applications menu"},
	{"Alt+Space", "the window menu"},
}

// keptKeysOn are p's kept keys.
func keptKeysOn(p keyPlatform) []keptKey {
	switch p {
	case platMac:
		return macKeptKeys
	case platBrowser:
		return browserKeptKeys
	}
	return desktopKeptKeys
}

// modeShortcuts are the keys with a modifier that vim's and helix's
// normal modes take (vim.go shortcut, helix_multi.go multiShortcut): the
// menu bar's shortcuts are looked at first (commands.go), so a command
// with one of these keys would take it from the mode.
var modeShortcuts = map[string][]string{
	"vim":   {"Ctrl+R", "Alt+J", "Alt+K", "Alt+H", "Alt+L", "Shift+Alt+J", "Shift+Alt+K", "Shift+Alt+H", "Shift+Alt+L"},
	"helix": {"Ctrl+R", "Alt+J", "Alt+K", "Alt+H", "Alt+L", "Shift+Alt+J", "Shift+Alt+K", "Shift+Alt+H", "Shift+Alt+L", "Alt+S", "Alt+,"},
}

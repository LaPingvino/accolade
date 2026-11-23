package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
)

// docs/KEYBOARD.md, the keyboard audit, is made from the registry, the
// ribbon's key tips on the sample, the ex table and the audits' tables:
//
//	UPDATE_KEYBOARD=1 go test -run TestKeyboardAudit .
func TestKeyboardAudit(t *testing.T) {
	w := sampleWindow(t, sampleAccolade)
	keepAllSettings(t, w)
	w.fyneWindow.Resize(fyne.NewSize(1400, 900))
	r := ribbonOnFor(t, w)
	r.refresh()
	tips := map[string]string{} // a command's key tips: the tab's letter, the button's
	r.tipsOn = true
	r.assignTips(1)
	tabTip := map[string]string{}
	for _, tb := range r.shownTabs() {
		tabTip[tb.title] = tb.tip
	}
	for _, tb := range r.shownTabs() {
		r.selectTab(tb.title)
		r.tipsOn = true
		r.assignTips(2)
		for _, b := range r.buttons {
			if b.c != nil && b.tip != "" && tips[b.c.id] == "" {
				tips[b.c.id] = tabTip[tb.title] + " " + b.tip
			}
		}
	}
	r.leaveKeyTips(false)
	exOf := map[string]string{}
	for _, c := range exCommands().cmds {
		if c.id != "" && exOf[c.id] == "" {
			exOf[c.id] = ":" + c.name
		}
	}
	for _, c := range commands().commands {
		if by, ok := exByCommand(c); ok {
			exOf[c.id] = ":" + by
		}
	}
	got := keyboardAudit(tips, exOf)
	path := filepath.Join("docs", "KEYBOARD.md")
	if os.Getenv("UPDATE_KEYBOARD") != "" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if b, err := os.ReadFile(path); err != nil || strings.ReplaceAll(string(b), "\r\n", "\n") != got { // a Windows checkout may give CRLF
		t.Errorf("%s is not what the registry makes: UPDATE_KEYBOARD=1 go test -run TestKeyboardAudit .", path)
	}
}

// keyboardAudit is docs/KEYBOARD.md.
func keyboardAudit(tips, exOf map[string]string) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# The keyboard audit\n\n")
	p("Made by `UPDATE_KEYBOARD=1 go test -run TestKeyboardAudit .` from the command registry (commands.go), the ribbon's key tips on the sample, the ex table (ex_table.go) and the tables of keyboard.go; the test fails when this file is stale. LOOP.md's keyboard pass (2026-11-20) asked for it.\n\n")
	p("## How every command is reached\n\n")
	p("Every command of the registry is in the command palette (Ctrl+K, F1) in some state of the window (TestEveryCommandByKeyboard), and on the `:` line by its ex name (vim's and helix's keys). ")
	p("Fyne's menus have no mnemonics (no Alt+letter into a menu, no underlined letters; the word is nowhere in the Fyne fork's source): the keyboard's way into the ribbon is its **key tips** instead — F10, or Alt pressed and let go (not on a Mac), then the tab's letter, then the button's (the column below; a disabled button has none at that moment, and the letters are given again when the buttons change). In the Menus look, the ribbon's ☰ and the palette are the way. The key tips below are the sample's (a script in production at the Schedule stage): the Production tab shows the stage the script is at first, so another stage's buttons have their tips when the script is at it.\n\n")
	p("Keys per platform: Linux and Windows have the registry's keys, a Mac its macKeys where set (Ctrl is Cmd there otherwise), the browser its webKeys where set (the browser keeps some for itself). Contextual commands (the Stripboard's, the Cards', the Table Read's …) are there while their view is.\n\n")
	p("| Command | id | Linux/Windows | macOS | browser | key tips | ex |\n|---|---|---|---|---|---|---|\n")
	for _, c := range commands().commands {
		keys := func(pl keyPlatform) string {
			ks := keyNamesOn(c, pl)
			if pl == platDesktop && c.shownKeys != "" {
				ks = append(ks, "Ctrl+"+c.shownKeys)
			}
			return strings.Join(ks, ", ")
		}
		desk, mac, web := keys(platDesktop), keys(platMac), keys(platBrowser)
		if !c.applies(cmdEnv{browser: true}) && !c.applies(cmdEnv{markdown: true, browser: true}) && !slices.ContainsFunc(keyEnvs, func(e struct {
			name string
			env  cmdEnv
		}) bool {
			x := e.env
			x.browser = true
			return c.applies(x)
		}) {
			web = "—"
		}
		p("| %s | `%s` | %s | %s | %s | %s | %s |\n", strings.ReplaceAll(c.label, "|", "/"), c.id, desk, mac, web, tips[c.id], exOf[c.id])
	}
	p("\n## The keys each platform keeps\n\nA command never has one of these (TestNoKeyClashes).\n\n")
	for _, pl := range keyPlatforms {
		var ks []string
		for _, k := range keptKeysOn(pl) {
			ks = append(ks, k.key+" ("+k.by+")")
		}
		p("- **%s:** %s\n", pl, strings.Join(ks, "; "))
	}
	p("\nThe browser's are Chromium's reserved commands (chrome/browser/ui/accelerator_table.cc, kReservedCommandIds, as its accelerator table maps them; read 2026-11-23), which a page can never take. Firefox's list was not read: it is believed to be like it, unchecked.\n")
	p("\n## Clashes, per mode and platform\n\nNone: TestNoKeyClashes looks at every state of the window with commands of its own (")
	var names []string
	for _, e := range keyEnvs {
		names = append(names, e.name)
	}
	p("%s), on each platform, and fails on two commands with one key, a command with a key its platform keeps, and a command with a key vim's or helix's normal mode takes (", strings.Join(names, ", "))
	for _, m := range []string{"vim", "helix"} {
		p("%s: %s; ", m, strings.Join(modeShortcuts[m], ", "))
	}
	p("the menus' shortcuts are seen first, so such a command would take the key from the mode). TestModeKeysDocumented types every key in each mode and fails on one that does something without a line in the modes' table (vim_keys.go), which the help pages are made from.\n\n")
	p("Found and fixed by this audit:\n\n")
	p("- The browser: New (Ctrl+N), Title Page (Ctrl+Shift+T) and the middle tabs (Ctrl+PgDn/PgUp) had keys the browser keeps; there they are Ctrl+Alt+N (New Empty: Ctrl+Shift+Alt+N), Ctrl+Shift+Alt+T and Alt+PgDn/PgUp. Ctrl+Tab out of the editor is the browser's too: F6 goes to the panes.\n")
	p("- A Mac: Replace's Cmd+H hid the app; it is Cmd+Option+F there (the Mac's own Replace).\n")
	p("- Inside or Outside… and Character Name… panicked when given from a menu, the palette or the `:` line (no button to open under): their menu opens under the cursor now.\n")
	p("- The markers (Highland's directives) had no keys: they have Highland's, Control+M (Marker…) and Control+Option+M (Next Marker) on a Mac, and Control+Shift+Option+M for Previous Marker beside them; elsewhere the same with Ctrl (Ctrl+M, Ctrl+Alt+M, Ctrl+Shift+Alt+M), none of which clashed.\n")
	p("- The help wrote Ctrl+Next and Ctrl+Prior for Ctrl+PgDn and Ctrl+PgUp.\n")
	p("- A dialog's scroll did not follow the focus (Tab onto a control below the fold left it out of sight): Tab now scrolls every scroll holding the control to it (scrollToFocus, keyboard.go; not when the focused widget keeps Tab, as the editor does).\n")
	p("- The Drafts list hid its focus under the selected draft: a line round it now.\n")
	p("- vim's Ctrl+Shift+R was redo too; it is Ctrl+R alone. The modes' keys with a modifier (modeShortcuts) are checked against what the modes take (TestModeShortcutsListed), on a Mac as Control and Option.\n")
	p("- The Table Read's reading area took the focus and did not show it: a line round it now.\n")
	p("- helix's `/` did nothing (vim's opened the search bar): it opens the search bar.\n")
	p("\nLeft as they are: Ctrl+Alt+J/K/A/G (Adaptation) are AltGr+J/K/A/G on Windows with a keyboard layout that has AltGr (Polish, German …): a letter typed there may be taken by them; there is no Ctrl+Alt key free of that, and Joop chose these. Fyne's file dialogs (Open, Import, Save PDF) are Fyne's own and not audited.\n")
	p("\n## Tab order and focus in the dialogs, views and panes\n\n")
	p("TestDialogTabOrder opens on the sample every command whose label ends in … (Accolade's own dialogs), the views (stripboard, cards, Table Read, Compare, the merge view), the side panes and the settings editor, and fails when Tab does not reach one of its shown, enabled controls (a row of tabs is one stop: its tab in front, the arrow keys along it). These must open, or it fails: %s and the panes (%s).\n\n", strings.Join(mustOpen, ", "), strings.Join(slices.Sorted(maps.Values(auditPanes)), ", "))
	p("These open nothing over the window on the sample as it opens, so their Tab order is not audited there (the test fails when this list is not what happened):\n\n")
	for _, id := range openedNothing {
		p("- %s (`%s`): it needs %s\n", strings.TrimSuffix(cmd(id).label, "…"), id, openedNothingWhy[id])
	}
	p("\nTestDialogsFocusVisible checks pixel by pixel that the focus shows on every control Tab reaches in %s on a desktop, and in %s on a phone too (Tab scrolls as the window does: TestTabScrollsToFocus goes through the window's key preview). ", strings.Join(focusDialogs, ", "), strings.Join(focusShots, ", "))
	p("The views and panes are TestFocusIsVisible's, TestDockFocusVisible's, TestStripboardFocusVisible's, TestRibbonFocusVisible's, TestPageViewFocusVisible's and the Characters', Adapt and Save As tests'.\n\n")
	p("Not opened by the audit:\n\n")
	var skips []string
	for id, why := range notOwnDialogs {
		skips = append(skips, fmt.Sprintf("- %s (`%s`): %s", strings.TrimSuffix(cmd(id).label, "…"), id, why))
	}
	slices.Sort(skips)
	p("%s\n", strings.Join(skips, "\n"))
	return b.String()
}

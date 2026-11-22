package main

import (
	"github.com/LaPingvino/accolade/docs/help"

	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// menuTitles are the menu bar's menus.
func menuTitles(m *fyne.MainMenu) []string {
	var out []string
	for _, menu := range m.Items {
		out = append(out, menu.Label)
	}
	return out
}

// labelsOf are a menu's labels ("---" for a separator).
func labelsOf(m *fyne.MainMenu, title string) []string {
	var out []string
	for _, menu := range m.Items {
		if menu.Label != title {
			continue
		}
		for _, it := range menu.Items {
			if it.IsSeparator {
				out = append(out, "---")
			} else {
				out = append(out, it.Label)
			}
		}
	}
	return out
}

// Every layout names commands that are defined, every command has a
// place (a menu, a row, the context menu or a key of its own), and the
// shortcuts are each one command's, in a screenplay and in Markdown.
func TestRegistryIsWhole(t *testing.T) {
	r := commands()
	placed := map[string]bool{}
	note := func(homes []home) {
		for _, h := range homes {
			if h.id != "" && h.id != "elements" && h.id != "formatButtons" && h.id != "scriptFormat" && h.id != "pageCount" {
				cmd(h.id) // panics for one not defined
				placed[h.id] = true
			}
		}
	}
	for _, m := range r.menuBar {
		note(m.homes)
	}
	note(r.topRow)
	note(r.screenplayRow)
	note(r.markdownRow)
	note(r.context)
	note(r.ribbonEnd)
	for _, tab := range r.ribbon {
		for _, g := range tab.groups {
			note(g.homes)
		}
	}
	for _, el := range toolbarElements {
		placed[elementCommandID(el)] = true
	}
	for _, pk := range r.previewKeys {
		placed[pk.c.id] = true
	}
	for _, p := range help.Cheats() { // Help > Cheat Sheets, made with the menu
		placed["help."+p.Slug] = true
	}
	for _, c := range r.commands {
		if !placed[c.id] {
			t.Errorf("%s is nowhere", c.id)
		}
		if c.label == "" {
			t.Errorf("%s has no label", c.id)
		}
		if c.run == nil && c.pick == nil && c.submenu == nil {
			t.Errorf("%s does nothing", c.id)
		}
	}
	for _, env := range []cmdEnv{{}, {markdown: true}, {browser: true}} {
		seen := map[string]string{}
		for _, c := range r.commands {
			if !c.applies(env) {
				continue
			}
			for _, k := range c.keys {
				if prev, dup := seen[shortcutText(k)]; dup {
					t.Errorf("%+v: %s and %s share %s", env, prev, c.id, shortcutText(k))
				}
				seen[shortcutText(k)] = c.id
			}
		}
	}
	if got := homesOf("view.source"); !slices.Equal(got, []string{"Adaptation > Source Pane", "View > Source"}) {
		t.Errorf("the source pane's homes: %q", got)
	}
}

// The rules for when a command applies and when it can be given: a
// screenplay's in a screenplay, Markdown's in Markdown, the desktop's on
// the desktop; the source's with a source, unlocking with a lock.
func TestCommandRulesPerMode(t *testing.T) {
	screenplay, markdown, browser := cmdEnv{}, cmdEnv{markdown: true}, cmdEnv{browser: true}
	for _, tc := range []struct {
		id                   string
		screenplay, md, wasm bool
	}{
		{"file.new", true, true, true},
		{"file.newWindow", true, true, false},
		{"file.importPandoc", false, false, false}, // no pandoc in these
		{"file.titlePage", true, false, true},
		{"file.startScreenplay", false, true, false},
		{"format.element.Scene Heading", true, false, true},
		{"format.bold", true, true, true},
		{"format.underline", true, false, true},
		{"source.choose", true, false, true},
		{"source.learned", true, false, true},
		{"production.lockNumbers", true, false, true},
		{"view.cards", true, false, true},
		{"view.changes", true, true, true},
		{"view.fullscreen", true, true, true},
	} {
		for _, x := range []struct {
			env  cmdEnv
			want bool
		}{{screenplay, tc.screenplay}, {markdown, tc.md}, {browser, tc.wasm}} {
			if got := cmd(tc.id).applies(x.env); got != x.want {
				t.Errorf("%s in %+v: applies %v", tc.id, x.env, got)
			}
		}
	}
	if !cmd("file.importPandoc").applies(cmdEnv{pandoc: true}) || cmd("file.importPandoc").applies(cmdEnv{pandoc: true, browser: true}) {
		t.Error("Import with pandoc: on the desktop with pandoc only")
	}
	// can be given now: the source's with a source, unlocking with a lock
	for _, tc := range []struct {
		id   string
		env  cmdEnv
		want bool
	}{
		{"source.choose", cmdEnv{}, true},
		{"source.link", cmdEnv{}, false},
		{"source.link", cmdEnv{source: true}, true},
		{"source.saveAs", cmdEnv{source: true, markdown: true}, false},
		{"production.unlockNumbers", cmdEnv{}, false},
		{"production.unlockNumbers", cmdEnv{locked: true}, true},
		{"production.unlockPages", cmdEnv{locked: true}, false},
		{"production.unlockPages", cmdEnv{pages: true}, true},
		{"production.lockNumbers", cmdEnv{}, true},
	} {
		if got := cmd(tc.id).canRun(tc.env); got != tc.want {
			t.Errorf("%s in %+v: can run %v", tc.id, tc.env, got)
		}
	}
	// the window's own state
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	if e := w.cmdEnv(); e.markdown || e.source || e.locked || e.pages {
		t.Errorf("a new script: %+v", e)
	}
	w.lockSceneNumbers()
	if !w.cmdEnv().locked {
		t.Error("locked numbers not seen")
	}
	w.setDocumentType(true)
	if e := w.cmdEnv(); !e.markdown || e.locked {
		t.Errorf("Markdown: %+v", e)
	}
}

// The browser build's menus: one window (no New Window), no pandoc; the
// rest as the desktop's.
func TestBrowserMenus(t *testing.T) {
	w := newTestWindow(t, "")
	desktopBar := w.menuBar(cmdEnv{pandoc: true})
	browserBar := w.menuBar(cmdEnv{browser: true, pandoc: true})
	if !slices.Equal(menuTitles(browserBar), []string{"File", "Edit", "Format", "Adaptation", "Production", "Collaboration", "View", "Help"}) {
		t.Errorf("browser menus %q", menuTitles(browserBar))
	}
	gone := []string{"New Window", "New Empty Window", "Import with pandoc…"}
	file := labelsOf(desktopBar, "File")
	for _, l := range gone {
		if !slices.Contains(file, l) {
			t.Errorf("desktop File has no %s", l)
		}
	}
	var want []string
	for _, l := range file {
		if !slices.Contains(gone, l) {
			want = append(want, l)
		}
	}
	if got := labelsOf(browserBar, "File"); !slices.Equal(got, want) {
		t.Errorf("browser File %q, want %q", got, want)
	}
	for _, title := range []string{"Edit", "Format", "Adaptation", "Production", "Collaboration", "View", "Help"} {
		if !slices.Equal(labelsOf(browserBar, title), labelsOf(desktopBar, title)) {
			t.Errorf("browser %s differs: %q", title, labelsOf(browserBar, title))
		}
	}
	md := w.menuBar(cmdEnv{browser: true, markdown: true})
	if !slices.Equal(menuTitles(md), []string{"File", "Edit", "Format", "Collaboration", "View", "Help"}) {
		t.Errorf("browser Markdown menus %q", menuTitles(md))
	}
}

// The Adaptation menu: beside Production, a screenplay's; the source's
// items out of File and Format, Start a Screenplay from This Source
// still in a Markdown document's File; the source pane in View too.
func TestAdaptationMenu(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	m := w.fyneWindow.MainMenu()
	if !slices.Equal(menuTitles(m), []string{"File", "Edit", "Format", "Adaptation", "Production", "Collaboration", "View", "Help"}) {
		t.Fatalf("menus %q", menuTitles(m))
	}
	want := []string{"Choose Source…", "Save Source As…", "Keep Source in the Script", "Source Pane", "Swap Source and Script", "---",
		"Link to Source Passage", "Next Source Passage", "Mark Source Passage as Cut", "Suggest Links to the Source",
		"Update Source Links", "Source Headings as Sections", "---", "What Accolade Learned…"}
	if got := labelsOf(m, "Adaptation"); !slices.Equal(got, want) {
		t.Errorf("Adaptation %q", got)
	}
	for _, title := range []string{"File", "Format"} {
		for _, l := range labelsOf(m, title) {
			if strings.Contains(l, "Source") {
				t.Errorf("%s still has %s", title, l)
			}
		}
	}
	if !slices.Contains(labelsOf(m, "View"), "Source") {
		t.Error("View lost Source")
	}
	if got := labelsOf(m, "Format"); got[len(got)-1] == "---" {
		t.Errorf("Format ends with a separator: %q", got)
	}
	w.setDocumentType(true)
	m = w.fyneWindow.MainMenu()
	if slices.Contains(menuTitles(m), "Adaptation") {
		t.Error("Markdown has an Adaptation menu")
	}
	if !slices.Contains(labelsOf(m, "File"), startScreenplayLabel) {
		t.Error("Markdown's File lost " + startScreenplayLabel)
	}
}

// What Accolade Learned… opens the source pane with its list open; with
// no source it says so.
func TestWhatAccoladeLearned(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	menuItemByLabel(t, w, "What Accolade Learned…").Action()
	if !strings.Contains(status(w), "No source") {
		t.Errorf("no source: status %q", status(w))
	}

	script := suggestScript + "/* Adaptation\ncharacters: ANNA = Anna, Annie\n*/\n"
	w = suggestWindow(t, script)
	s := w.sourcePane()
	if s.shown() {
		w.togglePane("source")
	}
	menuItemByLabel(t, w, "What Accolade Learned…").Action()
	if !s.shown() || !s.learned.open || !s.learned.box.Visible() || !s.learned.scroll.Visible() {
		t.Errorf("pane %v, list open %v, shown %v", s.shown(), s.learned.open, s.learned.box.Visible())
	}
	menuItemByLabel(t, w, "What Accolade Learned…").Action() // again: stays open
	if !s.shown() || !s.learned.open {
		t.Error("a second time closed it")
	}
}

// The Adaptation menu's shortcut still works: Ctrl+Alt+J, the next
// source passage, as the Format menu had it.
func TestAdaptationShortcutKept(t *testing.T) {
	w := newTestWindow(t, "INT. BARN - DAY\n\nRain.\n")
	it := menuItemByLabel(t, w, "Next Source Passage")
	if it.Shortcut == nil || shortcutText(it.Shortcut) != "Primary+Alt+J" {
		t.Errorf("Next Source Passage's shortcut %v", it.Shortcut)
	}
}

// Screenshots of the Adaptation and Format menus open, at a desktop's
// size (SHOT_DIR=… go test -run TestCommandMenuScreenshots).
func TestCommandMenuScreenshots(t *testing.T) {
	dir := os.Getenv("SHOT_DIR")
	if dir == "" {
		t.Skip("set SHOT_DIR to render screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Adaptation", "Format"} {
		w := suggestWindow(t, suggestScript)
		w.fyneWindow.Resize(fyne.NewSize(1000, 650))
		for _, m := range w.fyneWindow.MainMenu().Items {
			if m.Label == title {
				widget.NewPopUpMenu(m, w.fyneWindow.Canvas()).ShowAtPosition(fyne.NewPos(160, 4))
			}
		}
		f, err := os.Create(filepath.Join(dir, "menu-"+strings.ToLower(title)+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, w.fyneWindow.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
}
